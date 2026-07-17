## Context

当前 `cmd/collector/main.go` 同时通过带业务默认值的 `-objective`、`-queries` 构造 `collector.Request`，并通过 `agents/collector/prompts/prompt.go` 的 `go:embed` 把查询规划 prompt 编译进二进制。业务采集意图因此散落在 CLI 默认值、prompt 示例和运行时请求中；修改 prompt 文件不会改变已编译二进制，下次启动仍使用旧内容。

现有数据流是：CLI 业务 flags → `Request{Objective, SearchQueries}` → DeepSeek planner → 原始查询优先合并模型查询 → connector fan-out → materializer。DeepSeek 已通过 Eino `model.BaseChatModel.Generate` 接收 system/user messages，并配置 `ResponseFormatTypeJSONObject`；planner 的严格 JSON decoder 只接受 `{"queries":[...]}`。workflow 仅把模型输出用于查询，connector response 才能形成 `Candidate`。

本 change 跨越进程启动、prompt 资产、planner 合约和正式规格，且删除现有 CLI 参数，因此需要在实现前固定所有权、路径、失败和迁移决策。使用者包括本地开发、定时任务和部署脚本维护者；开发 Leader负责两道人工门禁，执行 Agent负责获批后的实现和交付。

### Eino reference audit

- `eino-ext` `components/model/deepseek`：`ChatModelConfig.ResponseFormatType` 支持 `ResponseFormatTypeJSONObject`，`Generate` 原样接收 Eino system/user messages；继续复用，不新增 provider adapter。
- `eino-examples` `compose/chain/main.go`、`components/prompt/chat_prompt/chat_prompt.go`：展示 prompt 与运行时 variables 分层，并把 chat template 放在 model 前；采用“稳定技术上下文 + 每次运行变量”的分层思想，不复制示例基础设施。
- `eino` `components/prompt` 与 `schema/message.go`：`FString`、`GoTemplate`、`Jinja2` 在 `Format` 时替换变量，缺失变量会在运行时失败。本 change 的业务 prompt 不需要模板语法，因此按不透明 UTF-8 文本加载，避免把可频繁编辑的采集意图变成隐含 DSL。
- gap：上游不负责项目的运行时 prompt 文件路径、大小/空值校验、业务意图唯一来源、查询域校验或 connector-only 事实边界；这些保留在项目层薄适配中。
- commits：`eino-ext` `9137edd89e72b72735ede69db1c5ae29178a6e41`；`eino-examples` `171220631fb7068ead50b7cd964b8c471647117d`；`eino` `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`。

## Goals / Non-Goals

**Goals:**

- 让一个运行时 prompt 文件成为“采集什么内容”的唯一自然语言来源，编辑后在下一次 collector 进程启动生效，无需修改 Go 或重新编译。
- 从 `cmd/collector` 删除业务主题、`-objective` 和 `-queries`，只保留技术参数。
- 以最小且严格的合约从 prompt 得到不可变 `Objective` 和模型生成的 `SearchQueries`。
- 在任何模型或 connector 调用前验证 prompt 文件，并保证错误不泄露 prompt 全文、模型 raw response 或 API key。
- 保持 DeepSeek 只做 connector 前查询规划，所有 `Candidate` 事实仍来自 connector response。
- 通过 fake/临时文件测试覆盖启动加载、重启生效、规划输出和失败边界，不触网、不使用真实 key。

**Non-Goals:**

- 不实现进程内热重载、文件监听、远程 prompt 服务、版本数据库或管理 UI。
- 不为 prompt 引入 frontmatter、YAML/JSON 配置、Eino template variables 或自定义 DSL。
- 不让 DeepSeek 返回 `Objective`、事实、证据或 `Candidate` 字段。
- 不改变 connector API、materialization 合约、模型 provider 或其他 Agent。

## Decisions

### 1. 选择 A：prompt 全文作为 `Request.Objective`，模型只输出 queries

启动时读取的 prompt 字节在通过 UTF-8/大小/非空校验后原样转为字符串，作为本次 `Request.Objective`。planner 把该 `Objective` 作为 system message，把 `CollectedAt` 和 `TimeWindowHours` 等纯技术运行上下文编码为 user message；初始 `SearchQueries` 为空。模型仍只允许返回唯一顶层结构 `{"queries":[...]}`，planner 只替换请求副本的 `SearchQueries`，`Objective` 和其他字段保持不变。

prompt 文件同时包含业务采集意图和模型必须遵守的查询输出约束；其中所有“采集什么内容”的表述只能存在于该文件。Go 代码可以保留不含具体行业/主题的结构校验、字段名、上限和安全错误。

备选方案比较：

- A（选定）：prompt 全文作为 `Request.Objective`，模型只输出 queries。优点是一个自然语言来源、沿用现有严格输出 schema、无需模型复述意图；缺点是 connector 看到的 `Objective` 也包含查询规划约束，但该字段本就作为采集目标上下文，且不会物化为事实。
- B：模型输出 `objective + queries`。这会增加 schema、校验和失败面，并让模型有机会改写用户意图，破坏 prompt 的唯一权威性；不采用。
- C：prompt 使用机器可解析 frontmatter。它能显式区分 objective 和 instructions，但会引入格式版本、解析器、转义和编辑门槛，当前没有多个结构化字段需要独立治理；不采用。

### 2. 启动时加载一次运行时文件，不再 `go:embed`

新增 `-prompt-file` 技术 flag，默认 `agents/collector/prompts/query_planner_v1.md`。绝对路径直接使用；相对路径由标准文件系统语义相对进程启动时的 current working directory 解析。不会向上搜索仓库根目录，也不会隐式回退到嵌入内容，以避免同一命令在不同目录静默读取不同文件。部署工作目录不含仓库资产时必须传绝对路径或显式设置工作目录。

loader 在每个进程启动时读取一次，运行期间持有该不可变字符串。文件在进程运行中变更不会影响当前 run；下一次启动重新读取并立即使用新内容。这正好满足“下一次启动生效”，同时避免热重载造成同一进程内语义漂移。

默认 prompt 保持版本控制，但删除 Go embed 包装器。回滚 prompt 可以恢复文件上一版本并重启；回滚程序可以部署 change 前二进制并恢复旧 flags 调用方式。

### 3. prompt loader 采用最小严格校验与安全错误

loader 只接受可读的普通文件（符号链接解析后的目标可以是普通文件）、最大 64 KiB、合法 UTF-8，且 `TrimSpace` 后非空；返回的 `Objective` 保留原始有效文本，不对内容做业务解析或改写。64 KiB 上限限制意外大文件的内存和模型输入成本，同时远高于当前 prompt。

缺失、路径指向目录/非普通文件、权限错误、读取错误、超限、非法 UTF-8 或空文件都在 DeepSeek provider、planner workflow 和 connector 初始化/调用前 fail-fast。对外错误使用稳定类别（例如“load collector prompt failed”或“collector prompt is invalid”）和必要的 prompt path 上下文；不得拼接文件内容、底层模型 raw response、环境变量值或 API key。测试断言敏感哨兵不出现在错误中。

### 4. 删除原始查询优先语义，模型查询确定性规范化

`-queries` 被移除后不再存在“用户原始查询”。planner 对模型 `queries` 执行现有的严格 JSON 验证、`TrimSpace`、精确字符串稳定去重和 256 rune 单条上限，并按模型顺序保留最多 12 条；空数组、空项、超长项、未知字段或无可用项继续 fail-closed。不会把 prompt 内文本直接当搜索词，也不会保留第二个静态查询来源。

备选的“保留 queries flag 但默认空”仍会允许采集内容散落到启动命令，违背唯一来源目标；“从 prompt 文件逐行解析查询”会把自然语言文件变成非显式配置协议，等同于方案 C，均不采用。

### 5. 保持 connector-only 事实边界

数据流固定为：

`-prompt-file` → 启动期安全 loader → `Request.Objective`（`SearchQueries=nil`）→ DeepSeek system message + 技术 user context → 严格 `queries` decoder/规范化 → 请求副本的 `SearchQueries` → 所有 connector → connector response `Candidate` → materializer。

planner 返回的唯一领域数据是查询字符串；`RunID`、`Objective`、`CandidateLimit`、`TimeWindowHours`、`CollectedAt` 不变。workflow 仍在 planner 成功后才 fan-out，仍只在 connector 节点把 connector response 标记为 `ContentOrigin=connector_response`。模型响应不进入 `Candidate`、日志或采集产物。

### 6. TDD 与验证

Apply 阶段先用临时目录/文件、fake chat model、fake connector 写 RED 测试，覆盖默认/绝对/相对路径、缺失/空/超限/非法 UTF-8/非普通文件、安全错误、prompt 修改后新启动读取新值、无业务 flags、A 方案消息与字段保持、模型查询去重/上限以及事实边界。GREEN 只实现通过测试的最小 loader、装配和 planner 变化；REFACTOR 后运行聚焦测试、`go test ./...`、strict OpenSpec validation、diff/凭据/禁止路径检查。

## Risks / Trade-offs

- [相对路径依赖 current working directory] → 文档明确规则，默认服务配置固定工作目录；其他部署传绝对 `-prompt-file`，不做隐式搜索。
- [prompt 编辑错误导致启动失败] → fail-fast、稳定错误类别、版本控制默认文件和临时文件测试；恢复上一版 prompt 并重启即可回滚。
- [prompt 语义质量下降但格式仍合法] → 这是可编辑业务内容的固有取舍；保持代码侧查询数量/长度/JSON 边界，领域内容通过 prompt review 管理。
- [完整 prompt 作为 Objective 增加 token] → 每次 run 仍只有一次规划调用，并以 64 KiB 硬上限控制；不增加模型或调用次数。
- [删除 flags 破坏旧脚本] → 在 README/示例中给出 `-prompt-file` 迁移，breaking change 在 proposal 和规格中明确。
- [底层文件或 provider 错误包含敏感内容] → 边界层返回新建的安全错误，不透传底层错误文本；保留 `context.Canceled`/deadline 的可判定因果但不暴露 provider payload。

## Migration Plan

1. 在部署前把当前期望采集意图和查询输出约束整理到默认 prompt 文件，更新启动脚本以删除 `-objective`/`-queries`；非仓库工作目录显式传绝对 `-prompt-file`。
2. 先运行 fake 聚焦测试与全量测试，再部署新二进制和 prompt 文件；启动失败时不会调用模型或 connector，也不会产生新产物。
3. 修改采集意图时只编辑 prompt 文件并重启 collector；通过 fake/日志中的安全阶段状态确认新进程已成功加载，但不输出 prompt 全文。
4. 回滚时恢复上一版本 prompt 并重启；若需回滚二进制，则同时恢复旧启动 flags。无数据迁移。
5. Leader Acceptance 后执行 Sync、Archive、提交、推送和 PR，交付 cleanup handoff；Merge/Cleanup 是归档后的主任务 operational state，不回写已归档 tasks。

## Open Questions

无。默认路径、相对路径基准、文件上限、重启生效语义、模型输出 schema 和 breaking CLI 迁移均在本设计中固定，等待 Leader Review。
