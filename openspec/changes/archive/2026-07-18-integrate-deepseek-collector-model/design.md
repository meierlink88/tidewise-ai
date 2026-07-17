## Context

当前 `cmd/collector` 构造一个 `collector.Request`，`internal/collector.NewWorkflow` 使用 Eino `compose.Workflow` 将同一请求直接 fan-out 到 Parallel Search、Tavily 和博查节点，再汇总 `ConnectorRun` 交给 materializer。系统没有 ChatModel、prompt 或 LLM 配置；`Candidate` 的事实字段和最终 Markdown 内容均来自 connector response，并以 `content_origin: "connector_response"` 标记。

本 change 引入一个新的外部模型服务。它只负责 connector 前的查询规划，不进入候选结果、去重或物化路径。主要参与者是 AI 采集器运行者、维护 collector workflow 的开发者及负责密钥注入的部署环境。执行仍遵循 Leader Review 和 Leader Acceptance 门禁，PR merge 由用户控制；Merge/Cleanup 是归档后由主任务跟踪的 operational state，不回写已归档 tasks。

### Eino reference audit

- `eino-ext`（commit `9137edd89e72b72735ede69db1c5ae29178a6e41`）：完整检查 `components/model/deepseek/` 的 README、实现、配置、options、测试、examples 和模块依赖。`components/model/deepseek/v0.1.7` 已实现 `model.ToolCallingChatModel`，`NewChatModel` 支持 `APIKey`、必需的 `Model`、`BaseURL`、`Timeout`、`HTTPClient` 和 `ResponseFormatTypeJSONObject`，`Generate` 对空 choices 返回错误。结论是直接复用 provider，不自建 DeepSeek HTTP client。
- `eino-examples`（commit `171220631fb7068ead50b7cd964b8c471647117d`）：完整检查 `quickstart/chat/` 与 `compose/workflow/` 六个示例。采用“入口初始化 provider、prompt 生成 messages、通过稳定模型接口调用”和“用 workflow field mapping 在 DAG 中传递显式数据依赖”的分层方式；不复制示例中的 `log.Fatal` 错误处理。
- `eino`（commit `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`）：检查 `components/model/interface.go`、`compose/workflow.go`、`compose/graph.go`、trigger/compile options 及 DAG/workflow 聚焦测试。`BaseChatModel` 提供 `Generate/Stream`，`ToolCallingChatModel.WithTools` 返回不可变派生实例；`Workflow.AddChatModelNode` 接受 `BaseChatModel`，而 `Workflow` 固定使用 `AllPredecessor` DAG，节点在全部执行前驱完成后触发。本 change 不绑定工具，模型实例保持只读共享。
- gap：上游不提供本项目的查询规划 schema、prompt、原始查询保留与上限规则、失败策略和 connector-only 证据边界；这些作为项目侧领域适配实现。

## Goals / Non-Goals

**Goals:**

- 每次采集在搜索 connector 前使用 DeepSeek 进行一次结构化查询规划。
- 规划输入显式包含 `Objective`、`SearchQueries`、`TimeWindowHours` 和 `CollectedAt`，输出经过确定性校验、规范化、去重和上限控制。
- 保留用户原始查询的优先级，并让所有 connector 接收同一份规划后请求。
- 复用官方 DeepSeek Eino provider，通过窄接口与 fake 隔离业务测试和真实 API。
- 对配置、prompt 版本、模型输出、错误与事实保真边界形成可测试契约。

**Non-Goals:**

- 多轮对话、会话记忆、`AgenticModel`、tool calling 或模型直接驱动 connector。
- 用模型替代搜索 connector，或让模型生成、总结、补写、改写 `Candidate` 事实字段和证据内容。
- 在首版提供可选 provider、多模型路由、自动重试、模型输出修复循环或真实 DeepSeek API 单元测试。
- 改变 connector API 合约、materialized document schema、三个任务 Agent 的职责边界或其他 Agent 的运行流程。

## Decisions

### 1. 在 connector 前增加单次查询规划节点

新数据路径为：

```text
CLI Request
  -> DeepSeek query planner
  -> validated planned Request
  -> Parallel Search / Tavily / Bocha (fan-out)
  -> ConnectorRun aggregation
  -> materialize
```

planner 返回 `Request` 的副本，只替换 `SearchQueries`；`RunID`、`Objective`、`CandidateLimit`、`TimeWindowHours` 和 `CollectedAt` 保持不变。所有 connector 节点从 planner 节点读取完整输出，不再直接从 `START` 读取，因此 Eino `AllPredecessor` 语义保证模型成功并完成校验后才 fan-out。materializer 继续处理 connector runs，模型响应不会进入 `Candidate` 或产物。

替代方案：

- 搜索后由模型筛选、摘要或重写结果：可能污染事实原文和证据链，且成本随候选数量增长，拒绝。
- 让模型直接调用搜索工具或使用 `AgenticModel`：引入多轮状态、工具权限与不可预测调用，超出首版最小范围，拒绝。
- 在每个 connector 内分别调用模型：重复成本、输出不一致并把 provider 逻辑散落到 connector，拒绝。
- 不接入 workflow、仅在 CLI 预处理查询：难以对编排顺序和所有调用方形成统一契约，拒绝。

### 2. 项目侧 `QueryPlanner` 薄适配器调用 `model.BaseChatModel.Generate`

`internal/collector` 定义 workflow 所需的窄 `QueryPlanner` 能力；DeepSeek planner 适配器依赖 `model.BaseChatModel` 的非流式 `Generate`，负责构造 messages、解析 JSON 和应用领域规则。`cmd/collector` 读取配置、调用 `deepseek.NewChatModel` 并将 planner 注入 workflow。业务测试注入 fake planner 或 fake chat model，不访问网络，也不需要 DeepSeek 密钥。

选择 lambda planner 节点而非直接串联 `AddChatTemplateNode -> AddChatModelNode -> parser`，因为领域输入/输出不是 `[]*schema.Message`，还需要严格的 JSON schema 校验、原始查询优先合并和 `Request` 复制。薄适配器让这些规则可独立单测，同时仍复用 Eino 的 `BaseChatModel` 契约。若未来多个 workflow 复用相同模型节点，再评估拆成原生多节点组合。

### 3. 使用版本化 prompt 和最小 JSON 合约

prompt 放在 `agents/collector/prompts/`，使用带版本号的文件名和 Go 嵌入加载器，避免依赖运行时工作目录并保持 prompt 可评审。system prompt 明确模型只规划搜索词，不输出事实；user message 传入目标、UTC 采集时间、时间窗口和原始查询。provider 配置 `ResponseFormatTypeJSONObject`，期望唯一顶层结构：

```json
{"queries":["query 1","query 2"]}
```

解析器拒绝空 content、非 JSON、未知顶层字段、缺少或非数组 `queries`、非字符串元素、空查询以及没有任何可用查询的响应。查询逐项 `TrimSpace`，按精确字符串稳定去重；不做语义改写，以免改变用户意图。

替代方案是 tool calling/JSON Schema function 参数。首版只需固定 JSON 对象，DeepSeek provider 原生支持 `json_object`，无需绑定工具或引入工具节点，因此选择更小的接口。

### 4. 原始查询优先、统一硬上限和确定性超限处理

规划结果的合并顺序为：先按输入顺序加入规范化后的用户原始查询，再按模型返回顺序加入新的唯一查询。总查询数硬上限为 12，单条 UTF-8 查询按 rune 计数不得超过 256；重复项不计入上限。超过总数上限的尾部模型查询被确定性丢弃；如果用户原始查询本身超过 12 条，则只保留去重后的前 12 条，且不再加入模型扩展。任何单条超长查询视为非法模型/输入结果并返回错误，不静默截断文本。

这一选择满足“保留、去重并限制原始查询”，同时避免模型输出造成 connector 请求膨胀。替代方案是超限即让整次运行失败；它会让可安全裁剪的额外查询降低可用性，因此只对结构或单条长度非法 fail closed，对数量超限采用稳定裁剪。

### 5. 配置集中校验，DeepSeek 故障 fail closed

`internal/config` 提供 DeepSeek 配置加载与校验，`cmd/collector` 负责装配：

- `DEEPSEEK_API_KEY`：必需，trim 后为空则启动失败；值不出现在错误、日志或结果中。
- `DEEPSEEK_MODEL`：可选，默认 `deepseek-chat`。
- `DEEPSEEK_BASE_URL`：可选，空值使用 provider 默认地址。
- `DEEPSEEK_TIMEOUT`：可选 Go duration，默认 `30s`，必须大于 0。

缺失或非法配置在 connector 启动前失败。模型 API 错误、context deadline、空响应、非法 JSON、非法字段、无可用查询或单条超长查询均使本次 workflow 失败，connector 不执行，且不会产生新的采集产物。错误包含阶段和类别但不包含 API key、完整 prompt 或模型原始响应。首版不自动 fallback 到原始查询：静默绕过模型会让“已启用查询规划”的运行语义不可信；运维可通过回滚版本恢复旧路径。

替代方案是 fail open 到用户原始查询。它能提高短期可用性，但无法从结果判断 planner 是否生效，也可能隐藏持续的配置或输出合约故障，因此首版拒绝。未来如需降级，必须新增显式配置、可观测状态和规格。

### 6. 依赖固定为 DeepSeek provider `v0.1.7`

引入 `github.com/cloudwego/eino-ext/components/model/deepseek@v0.1.7`。该 module 的 `go.mod` 使用 Go 1.24 并要求 `github.com/cloudwego/eino v0.7.13`；本项目使用 Go 1.24.7 和 Eino `v0.9.12`。Go minimal version selection 将继续选择项目较高的 `v0.9.12`，审计到的 `BaseChatModel`/`ToolCallingChatModel` 方法集与 provider 实现兼容。Apply 阶段以 `go mod tidy` 和 `go test ./...` 验证真实依赖图，不添加 `replace`，也不降级 Eino。

成本为每次 collector run 一次 chat completion 的网络延迟和 token 消耗，以及 `deepseek-go` 等传递依赖。prompt 只发送目标、时间信息和查询，不发送 connector 结果或密钥，从输入规模上限制成本与数据暴露。

### 7. 测试与可观测边界

RED 测试先覆盖配置默认/拒绝、planner messages 和 JSON 校验、原始查询优先合并/去重/上限、planner failure 阻止 connector、所有 connector 接收同一规划请求，以及 `Candidate.ContentOrigin` 和事实字段不受模型响应影响。GREEN 只实现使这些测试通过的最小代码；REFACTOR 后运行相关包和全量测试。所有聚焦测试使用 fake，不访问真实 DeepSeek。

错误只向 CLI 返回安全摘要。首版不记录 prompt/response body，不新增持久化模型响应，也不改变 summary schema；可通过错误类别和 connector 未启动这一可观察行为判断失败阶段。

## Risks / Trade-offs

- [DeepSeek 不可用会阻断整次采集] → 使用 30 秒可配置超时、启动前配置校验、清晰错误类别和可回滚版本；首版接受 fail closed 以保证运行语义。
- [JSON mode 仍可能返回不符合领域 schema 的内容] → 使用严格 decoder、未知字段拒绝、字段与长度校验，并以 fake 覆盖空响应、非法 JSON 和边界值。
- [模型扩展查询偏离用户意图] → 原始查询优先、模型仅追加、稳定去重和 12 条硬上限；prompt 禁止输出事实或改变原始查询。
- [新增依赖与 Eino 版本产生兼容问题] → 固定 `v0.1.7`，保持项目 Eino `v0.9.12`，通过模块整理、编译、聚焦测试和 `go test ./...` 验证。
- [每次运行增加延迟与 token 成本] → 仅一次非流式调用、短输入、结构化短输出和超时；不把候选正文发送给模型。
- [prompt 变更改变查询行为] → prompt 文件版本化并由测试验证关键约束；后续行为变化需更新规格和测试。
- [数量裁剪可能丢失低优先级查询] → 保持确定性顺序并优先保留用户输入；上限和裁剪规则写入规格。

## Migration Plan

1. 在获 Leader Review 后按 RED -> GREEN -> REFACTOR 实施配置、prompt、planner、workflow 装配和文档。
2. 固定新增 provider 版本并运行 `go mod tidy`，确认 Eino 仍解析为 `v0.9.12`。
3. 部署前通过 Secret Manager 或本地 `.env` 注入 `DEEPSEEK_API_KEY`，按需设置 model、base URL 和 timeout。
4. 运行 fake 覆盖的聚焦测试、`go test ./...`、strict validation、diff/密钥/禁止路径检查；不以真实 API 作为自动化门禁。
5. 获 Leader Acceptance 后 Sync、Archive、Deliver；用户决定是否 merge。

回滚时回退本 change 的代码与依赖提交并恢复旧的 `START -> connectors` 路径；新增环境变量可留存或由部署环境删除，不涉及数据迁移。由于模型响应从不写入采集产物，回滚无需转换已有数据。

## Open Questions

无。首版固定 12 条总上限、256 rune 单条上限、`30s` 默认超时和 fail-closed 策略；后续若需可配置上限、重试或显式 fallback，另行提出 change。
