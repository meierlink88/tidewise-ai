# Collector Query Planning Specification

## Purpose

定义 AI 采集器如何在搜索 connector 前使用 DeepSeek 规划结构化查询，同时保证配置安全、失败可观察、查询数量受控，并维持 connector-only 的事实与证据边界。

## Requirements

### Requirement: DeepSeek 查询规划位于 connector 前
AI 采集器 SHALL 在任何搜索 connector 执行前，使用 DeepSeek 查询规划器根据 `Objective`、`SearchQueries`、`TimeWindowHours` 和 `CollectedAt` 生成规划后的搜索查询，并 SHALL 将同一份规划后 `Request` 发送给所有已配置 connector。

#### Scenario: 成功规划后执行多 connector 采集
- **WHEN** DeepSeek 返回通过校验的结构化查询，并且 collector 配置了多个 connector
- **THEN** 所有 connector 在 planner 完成后各执行一次，并接收相同的规划后 `SearchQueries`

#### Scenario: planner 尚未完成
- **WHEN** 查询规划节点仍在执行
- **THEN** 任何搜索 connector MUST NOT 开始执行

### Requirement: 规划输入和 Request 字段保持
查询规划器 SHALL 将采集目标、UTC 采集时间、时间窗口和用户原始查询提供给模型。规划器 SHALL 只替换 `Request.SearchQueries`，并 MUST 保持 `RunID`、`Objective`、`CandidateLimit`、`TimeWindowHours` 和 `CollectedAt` 不变。

#### Scenario: 生成规划请求
- **WHEN** collector 收到包含完整字段的 `Request`
- **THEN** 模型输入包含目标、UTC 时间、时间窗口和原始查询，且 planner 输出除 `SearchQueries` 外的字段与输入相同

### Requirement: 结构化模型输出严格校验
查询规划器 SHALL 要求模型返回唯一顶层结构 `{"queries":[...]}`，并 SHALL 拒绝空响应、非 JSON、未知顶层字段、缺失或非数组 `queries`、非字符串元素、空查询、无任何可用查询以及超过 256 rune 的单条查询。错误 MUST NOT 包含 API key、完整 prompt 或完整模型原始响应。

#### Scenario: 合法 JSON 输出
- **WHEN** 模型返回仅含非空字符串数组 `queries` 的 JSON object，且每条查询不超过 256 rune
- **THEN** planner 接受输出并进入规范化、去重和合并

#### Scenario: 非法模型输出
- **WHEN** 模型返回空响应、非法 JSON、未知字段、错误字段类型、空查询或超长查询
- **THEN** workflow 返回安全的查询规划错误，任何 connector 不执行，且不产生新的采集产物

### Requirement: 原始查询优先合并、去重和限制
查询规划器 SHALL 对查询执行 `TrimSpace`，按精确字符串稳定去重，并 SHALL 先按输入顺序保留用户原始查询，再按模型返回顺序追加新的唯一查询。最终查询总数 MUST 不超过 12；超过上限的尾部查询 SHALL 被确定性丢弃，且用户原始查询 SHALL 优先于模型扩展查询。

#### Scenario: 原始查询与模型查询重叠
- **WHEN** 模型返回的查询包含用户原始查询的重复项和新的查询
- **THEN** 最终列表保留一个原始查询，并在其后按模型顺序追加新的唯一查询

#### Scenario: 模型扩展导致总数超限
- **WHEN** 合并后的唯一查询超过 12 条且用户原始查询不超过 12 条
- **THEN** planner 保留全部用户原始查询，并仅保留可填满 12 条上限的最前模型扩展查询

#### Scenario: 用户原始查询自身超限
- **WHEN** 规范化和去重后的用户原始查询超过 12 条
- **THEN** planner 仅保留前 12 条用户原始查询且不追加模型扩展查询

### Requirement: DeepSeek 配置和安全默认值
collector SHALL 从环境或本地 dotenv 读取 DeepSeek 配置。`DEEPSEEK_API_KEY` SHALL 为必需且非空；`DEEPSEEK_MODEL` 为空时 SHALL 默认为 `deepseek-chat`；`DEEPSEEK_BASE_URL` SHALL 可选；`DEEPSEEK_TIMEOUT` 为空时 SHALL 默认为 `30s`，非空时 MUST 为大于 0 的 Go duration。任何密钥 MUST NOT 写入日志、错误、prompt、模型响应存储或采集产物。

#### Scenario: 使用最小有效配置启动
- **WHEN** 部署环境仅提供非空 `DEEPSEEK_API_KEY`
- **THEN** collector 使用 `deepseek-chat`、provider 默认 base URL 和 `30s` timeout 初始化 DeepSeek provider

#### Scenario: 缺失密钥或非法超时
- **WHEN** `DEEPSEEK_API_KEY` 为空，或 `DEEPSEEK_TIMEOUT` 无法解析或不大于 0
- **THEN** collector 在模型和 connector 调用前启动失败，并返回不包含配置值的安全错误

### Requirement: 模型故障采用 fail-closed 语义
模型 API 错误、context deadline、空响应、结构校验失败或无可用查询 SHALL 使本次采集失败。collector MUST NOT 静默回退到未规划的原始查询，MUST NOT 执行 connector，并 MUST NOT 为失败运行创建新的采集产物。

#### Scenario: DeepSeek API 调用失败
- **WHEN** DeepSeek provider 返回网络、鉴权、限流、服务端或 deadline 错误
- **THEN** collector 返回带查询规划阶段类别的安全错误，且 connector 调用次数为零

### Requirement: connector-only 事实与证据边界
DeepSeek 输出 SHALL 仅用于 `Request.SearchQueries`。模型 MUST NOT 创建或修改 `Candidate.Title`、`Candidate.URL`、`Candidate.PublishedAtHint`、`Candidate.Content`、来源字段、证据内容或 `ContentOrigin`；所有物化候选内容 SHALL 继续来自 connector response 并标记为 `connector_response`。

#### Scenario: 成功采集并物化结果
- **WHEN** planner 成功且 connector 返回候选结果
- **THEN** 物化结果的标题、URL、发布时间提示、正文和来源字段等于 connector response 对应字段，且 `ContentOrigin` 为 `connector_response`

#### Scenario: 模型响应包含事实性文本
- **WHEN** 模型在结构化查询之外尝试提供事实资料或证据文本
- **THEN** planner 将该响应判定为不符合输出合约，任何事实文本 MUST NOT 进入 `Candidate` 或采集产物

#### Scenario: 查询字符串包含事实性措辞
- **WHEN** 合法 `queries` 数组中的字符串包含事实性措辞
- **THEN** collector 仅将该字符串作为搜索查询发送给 connector，MUST NOT 将其直接写入 `Candidate` 或采集产物

### Requirement: 自动化测试不得访问真实 DeepSeek
查询规划、workflow 和配置的自动化测试 SHALL 使用 fake planner 或 fake chat model，且 MUST NOT 要求真实 `DEEPSEEK_API_KEY`、消耗模型额度或依赖公网可用性。

#### Scenario: 运行聚焦测试和全量测试
- **WHEN** 开发者运行查询规划相关测试或 `go test ./...`
- **THEN** 测试通过可注入 fake 覆盖成功和失败场景，不向真实 DeepSeek endpoint 发出请求
