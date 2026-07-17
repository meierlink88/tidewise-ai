## ADDED Requirements

### Requirement: Tavily 执行可配置 topic 的确定性搜索
AI 采集器 SHALL 提供默认值为 `general` 的 Tavily 专用 topic 配置，且仅接受 `general`、`news`、`finance`。Tavily connector SHALL 在单次搜索请求中显式发送该配置值、`search_depth: "advanced"`、`chunks_per_source: 3`、`include_answer: false` 以及由采集时间窗口计算的 `start_date` 和 `end_date`，并 MUST NOT 启用或发送 `auto_parameters` 与 `include_raw_content`。默认值由 design 所记录的受控真实对比确定：`general` 对固定中文 A 股 query 的严格相关性最佳且明显无关结果最少。

#### Scenario: 请求包含确定参数且排除自动与原文参数
- **WHEN** AI 采集器以已知 `CollectedAt` 和 `TimeWindowHours` 调用 Tavily
- **THEN** fake HTTP 捕获到的请求包含配置的 topic、advanced 和日期窗口参数，且不存在 `auto_parameters` 与 `include_raw_content`

#### Scenario: 未覆盖时使用证据所选默认 topic
- **WHEN** 操作者未显式配置 Tavily topic
- **THEN** AI 采集器使用 `general`

#### Scenario: 拒绝不支持的 topic
- **WHEN** Tavily topic 不是 `general`、`news` 或 `finance`
- **THEN** AI 采集器在发送 Tavily HTTP 请求前返回配置错误

### Requirement: Tavily 保持单 query 单调用
Tavily connector SHALL 按既有顺序把 `Request.SearchQueries` 以空格拼接为一个 query，并 SHALL 每次 `Collect` 最多向 Tavily Search API 发起一次请求。

#### Scenario: 多个规划查询合并为一次请求
- **WHEN** `SearchQueries` 包含两个或以上查询
- **THEN** fake HTTP 仅收到一次请求，且请求 query 等于按原顺序以空格拼接的查询

### Requirement: Tavily 使用独立的质量优先结果预算
AI 采集器 SHALL 提供默认值为 5、可在 1–20 范围配置的 Tavily 专用结果上限。Tavily 请求的 `max_results` SHALL 等于该专用上限与通用 `CandidateLimit` 的较小值，且该配置 MUST NOT 改变其他 connector 的结果预算。

#### Scenario: 默认专用上限低于通用上限
- **WHEN** 使用默认 Tavily 上限 5 且通用 `CandidateLimit` 为 10
- **THEN** Tavily 请求的 `max_results` 等于 5

#### Scenario: 通用上限仍是硬上界
- **WHEN** Tavily 专用上限为 8 且通用 `CandidateLimit` 为 6
- **THEN** Tavily 请求的 `max_results` 等于 6

#### Scenario: 拒绝非法专用上限
- **WHEN** Tavily 专用上限小于 1 或大于 20
- **THEN** AI 采集器在发送 Tavily HTTP 请求前返回配置错误

### Requirement: Tavily 候选仅采用 query-related content
Tavily connector SHALL 仅使用每条 result 的 `content` 生成候选内容，MUST NOT 使用 `raw_content` 或 answer。非空 `content` SHALL 标记为 `snippet`；空白 `content` SHALL 回退为标题并标记为 `title_only`。

#### Scenario: content 与 raw_content 同时出现在 fake 响应
- **WHEN** fake HTTP 响应同时包含 query-related `content` 和不同的 `raw_content`
- **THEN** 候选内容等于 `content`、内容等级为 `snippet`，且候选不包含 `raw_content`

#### Scenario: content 为空时回退标题
- **WHEN** Tavily result 的 `content` 为空白且标题与 URL 有效
- **THEN** 候选内容等于标题并标记为 `title_only`

### Requirement: Tavily 契约测试隔离真实服务与密钥
Tavily 请求和响应行为的自动化测试 SHALL 使用 fake HTTP transport 或 fake server，并 MUST NOT 访问真实 Tavily API、读取真实 API key 或持久化密钥。

#### Scenario: 自动化测试验证 provider 契约
- **WHEN** 执行 Tavily connector 的聚焦测试或全量测试
- **THEN** 请求被本地 fake HTTP 捕获，响应由测试构造，且无需任何真实 provider 凭据或网络访问
