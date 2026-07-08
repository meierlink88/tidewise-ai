## ADDED Requirements

### Requirement: AI Web Research 采集通道
系统 SHALL 将 AI Web Research 纳入第一批可扩展采集通道，使采集系统可以通过统一 source catalog、connector、parser、runtime 和 scheduler 边界触发大模型联网搜索采集。

#### Scenario: source catalog 驱动 AI 采集
- **WHEN** 采集源使用 `ingest_channel=llm_web_research`、`connector_key=llm_web_research` 和 `parser_key=llm_research_items`
- **THEN** 系统必须通过采集子系统注册表找到对应 connector/parser，并按该 source 的 `source_config` 和 `credential_ref` 执行采集

#### Scenario: 统一调度触发
- **WHEN** 后续 scheduler 读取多个 active source 并包含 AI Web Research source
- **THEN** AI Web Research source 必须与其他 connector 一样进入统一调度、限流、失败隔离、汇总 report 和幂等写入流程

### Requirement: AI 采集源配置校验
系统 SHALL 对 AI Web Research source 的 `source_config` 执行 connector 专属校验，确保采集运行参数完整、非敏感且可审计。

#### Scenario: 校验必填参数
- **WHEN** seed 或运行时加载 AI Web Research source
- **THEN** 系统必须校验 `api_base_url`、`api_protocol`、`model`、`prompt`、`prompt_version`、`max_results`、`output_schema` 和 `search_options` 的类型与取值

#### Scenario: 保护提示词和模型参数
- **WHEN** 开发者查看 repo 内 AI Web Research source seed
- **THEN** seed 可以包含提示词、模型名、base URL 和搜索参数，但不得包含真实 API key、token、cookie 或私有凭证值

### Requirement: AI 搜索结果原始文档标准化
系统 SHALL 将 AI Web Research 返回的结构化 items 标准化为与其他 connector 一致的原始文档候选对象。

#### Scenario: 标准化结构化 item
- **WHEN** AI Web Research parser 接收到已校验的 item
- **THEN** 系统必须生成包含标题、正文或摘要、来源 URL、来源名称、发布时间、采集时间、内容哈希、来源等级、内容来源类型和 raw metadata 的原始文档候选对象

#### Scenario: 幂等处理重复结果
- **WHEN** AI Web Research 多次返回相同来源 URL、外部 ID 或内容哈希的 item
- **THEN** 系统必须复用现有 raw document 幂等策略，不得创建重复事实基础
