## Why

当前 AI 采集器只把用户给定的 `SearchQueries` 原样发送到搜索 connector，无法结合采集目标和时间窗口补足检索覆盖面。接入 DeepSeek 作为 connector 前的结构化查询规划阶段，可以在不改变事实来源边界的前提下提升多通道检索质量，并为后续模型能力建立可测试、可替换的最小集成边界。

## What Changes

- 在 collector workflow 的 connector fan-out 前新增 DeepSeek 查询规划阶段，根据 `Objective`、`TimeWindowHours`、`CollectedAt` 和用户原始 `SearchQueries` 生成结构化查询。
- 保留、规范化、去重并限制用户原始查询；严格校验模型 JSON 输出，再将规划后的 `Request` 同时发送给 Parallel Search、Tavily 和博查。
- 使用官方 `github.com/cloudwego/eino-ext/components/model/deepseek` provider，并通过项目侧薄适配器和可注入模型接口隔离领域规划逻辑与 provider 初始化。
- 新增 `DEEPSEEK_API_KEY`、默认 `deepseek-chat` 的 `DEEPSEEK_MODEL`、可选 `DEEPSEEK_BASE_URL` 和超时配置；密钥只从环境或本地 `.env` 读取，不进入日志、产物或版本库。
- 将查询规划 prompt 版本化放入 `agents/collector/prompts/`，并为缺失配置、模型调用错误、空响应、非法 JSON 和超限查询定义明确的失败语义。
- 保持 `Candidate` 的标题、URL、发布时间提示、正文和证据仅来自 `connector_response`；模型不得生成、补写或改写事实资料。
- 非目标：多轮对话、`AgenticModel`、模型直接调用工具、模型替代搜索 connector、模型生成事实内容，以及依赖真实 DeepSeek API 的单元测试。
- 本 change 不改变三个任务 Agent 的职责边界；仅影响 AI 采集器。执行 Agent 在 Leader Review 前不进入 Apply，在 Leader Acceptance 前不执行 Sync/Archive/Deliver；用户仍控制 PR merge，执行 Agent在 PR 交付时提供 cleanup handoff。

## Capabilities

### New Capabilities

- `collector-query-planning`: 定义 AI 采集器使用 DeepSeek 生成、校验和应用结构化搜索查询的行为、配置、失败语义及事实保真边界。

### Modified Capabilities

无。

## Impact

- 受影响代码：`cmd/collector/`、`internal/collector/`、`internal/config/`、`agents/collector/prompts/`、`.env.example`、`README.md` 及相关测试。
- 新增依赖：`github.com/cloudwego/eino-ext/components/model/deepseek` 及其传递依赖；项目继续使用 `github.com/cloudwego/eino v0.9.12`。
- 外部系统：运行时新增一次 DeepSeek chat completion 调用，增加延迟、模型 token 成本和一个外部故障点；既有搜索 connector 与采集产物 schema 保持兼容。
- 主要风险：模型输出不稳定、查询膨胀、配置或网络故障、provider 版本兼容；通过 JSON 模式、领域校验、硬上限、fake 测试、超时和可回滚装配控制。
