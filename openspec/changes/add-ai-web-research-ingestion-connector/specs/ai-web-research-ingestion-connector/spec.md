## ADDED Requirements

### Requirement: AI Web Research 采集连接器
系统 SHALL 将多个 Web Search API tool 与 LLM 结构化整理接入为 `source_catalogs` 驱动的单一 AI Web Research 采集连接器，并将检索结果标准化为原始文档候选对象。

#### Scenario: 执行 AI Web Research source
- **WHEN** 采集任务读取到 active 状态且 `connector_key=llm_web_research` 的采集源
- **THEN** 系统必须根据该 source 的 `provider_key`、`credential_ref`、`source_config.web_search_plan` 和限流策略调用一个或多个 Web Search adapter、LLM normalizer 或 fake provider

#### Scenario: Tavily 搜索工具召回
- **WHEN** AI Web Research source 的 `web_search_plan` 包含 `provider=tavily`
- **THEN** 系统必须通过 Tavily 搜索获取标题、URL、摘要、来源、发布时间、搜索排名和原始搜索元数据，并把搜索失败、空结果和参数错误写入采集 report

#### Scenario: 中文搜索工具召回
- **WHEN** AI Web Research source 的 `web_search_plan` 包含 `provider=bocha_web_search` 或 `provider=searchapi_baidu`
- **THEN** 系统必须通过对应 adapter 获取中文搜索结果的标题、URL、摘要、来源站点、发布时间、搜索排名和原始 provider 元数据，并把搜索失败、空结果和参数错误写入采集 report

#### Scenario: 多搜索工具合并
- **WHEN** AI Web Research source 的 `web_search_plan` 配置多个搜索工具
- **THEN** 系统必须按配置的 parallel、fallback 或 sequential 模式执行搜索，并在调用 LLM 前完成结果归一化、URL 规范化、标题或内容哈希去重、可信域名标记、排序和总量截断

#### Scenario: 中国财经来源优先
- **WHEN** AI Web Research source 配置中国财经类来源偏好或可信域名
- **THEN** 系统必须优先保留、排序或标记中国官方机构、交易所、主流财经媒体和可信中文财经站点结果，并在 raw metadata 中记录命中的来源偏好

#### Scenario: LLM 结构化整理
- **WHEN** Web Search provider 返回搜索结果
- **THEN** 系统可以调用 LLM normalizer 将搜索结果整理为 `items` JSON，但不得把 LLM 生成的 model、query_time 或未经校验的来源字段当作系统事实

#### Scenario: 缺少凭证引用
- **WHEN** AI Web Research source 需要 Web Search API key 或 LLM API key 但凭证引用为空或无法解析到真实 secret
- **THEN** 系统必须跳过该 source 并记录明确错误，不得调用外部 API 或写入伪造原始文档

### Requirement: AI connector source_config
系统 SHALL 使用 `source_config` 保存 AI Web Research connector 的非敏感运行参数，并拒绝包含真实密钥的配置。

#### Scenario: 读取模型运行参数
- **WHEN** connector 准备执行 AI Web Research source
- **THEN** 系统必须能够从 `source_config` 读取 `web_search_plan`、来源偏好、可信域名、LLM provider、API base URL、API 协议、模型名、`prompt_ref`、`prompt_version`、`prompt_variables`、时间窗口、最大结果数、语言和输出 schema

#### Scenario: 读取 repo prompt 文件
- **WHEN** connector 准备调用 LLM normalizer
- **THEN** 系统必须根据 `prompt_ref` 和 `prompt_version` 加载 repo 内版本化 prompt 文件，并用 `prompt_variables` 渲染运行时变量，不得要求 `source_config` 保存完整提示词正文

#### Scenario: 拒绝敏感配置
- **WHEN** `source_config` 包含 API key、bearer token、cookie 或其他真实凭证字段
- **THEN** 系统必须拒绝加载该 source 或返回配置错误，真实凭证只能通过 `credential_ref` 注入

### Requirement: 结构化搜索结果契约
系统 SHALL 要求 AI Web Research provider 返回可校验的结构化 JSON 对象，并以 `items` 数组表达待写入的原始材料。

#### Scenario: 解析有效结果
- **WHEN** provider 返回包含 `items` 数组的结构化 JSON
- **THEN** parser 必须逐条校验标题、来源归因、正文或摘要、内容来源类型、发布时间、语言、地域、主题标签、证据摘录和相关性说明

#### Scenario: 接受非 URL 来源归因
- **WHEN** AI Web Research item 没有来源 URL，但包含来源名称、来源说明、引用文本或 provider 返回的来源描述
- **THEN** parser 可以将该 item 转换为原始文档候选对象，并必须在 raw metadata 中记录 `source_attribution_type` 和原始来源说明

#### Scenario: 拒绝无来源归因结果
- **WHEN** AI Web Research item 同时缺少来源 URL、来源名称、来源说明、引用文本和 provider 来源描述
- **THEN** 系统必须拒绝该条目进入 raw document 写入边界，并在采集 report 中记录跳过原因

#### Scenario: 拒绝无效结果
- **WHEN** provider 返回非 JSON、缺少 `items`、条目字段缺失、条数超过配置上限或内容来源类型不受支持
- **THEN** 系统必须返回解析错误并阻止无效条目进入 raw document 写入边界

### Requirement: 原始文档内容来源标记
系统 SHALL 为 AI Web Research 生成的原始文档记录内容来源和检索方法，使后续事件抽取能够判断证据质量。

#### Scenario: 保存网页原文
- **WHEN** AI Web Research item 的正文来自真实网页抓取或 provider 返回的网页正文
- **THEN** 原始文档候选对象必须标记 `content_origin=fetched_source_text`

#### Scenario: 保存模型总结
- **WHEN** AI Web Research item 的正文来自模型根据搜索结果生成的总结
- **THEN** 原始文档候选对象必须标记 `content_origin=llm_generated_summary`，不得把该内容伪装成原始新闻全文

#### Scenario: 保存搜索摘录
- **WHEN** AI Web Research item 只有搜索结果摘要或片段
- **THEN** 原始文档候选对象必须标记 `content_origin=search_snippet`，并保留来源归因、搜索元数据和证据摘录

### Requirement: 采集职责安全边界
系统 SHALL 保证 AI Web Research connector 只采集和保存原始材料，不生成投资建议或事件图谱事实。

#### Scenario: 处理模型输出
- **WHEN** provider 返回利好利空、买入卖出、涨跌预测、传导强度、事件评分、实体关系或图谱结论字段
- **THEN** 系统不得把这些字段写成系统事实，必须丢弃、记录为越界字段或将该条目标记为校验失败

#### Scenario: 写入 raw_documents
- **WHEN** AI Web Research item 通过结构化校验
- **THEN** 系统只能将其作为原始文档或采集元数据写入，不得在本 change 中创建事件、标签、实体关联或实体关系
