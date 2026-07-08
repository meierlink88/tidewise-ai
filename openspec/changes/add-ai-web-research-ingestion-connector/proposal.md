## Why

当前采集层已经能够通过 RSS、Eastmoney HTTP、本地回灌等固定来源写入 `raw_documents`，但缺少一种面向开放互联网的主动发现能力。为了捕捉过去 1 天内可能影响股票市场的全球政治经济热点，需要把 Web Search API provider 接入采集流程，获取 URL、标题、摘要、来源、发布时间和搜索元数据，并用大模型对搜索结果做结构化整理。

前期验证表明，单独依赖大模型原生联网能力不稳定；使用 Tavily Search API 作为工具调用、DeepSeek 作为结构化整理模型，可以返回可审计的候选财经材料。同时，中国财经信息检索不能只依赖一个全球搜索 provider，需要补充中文和百度生态搜索 provider，并通过来源白名单、provider profile 和程序侧校验控制来源质量。

本 change 只解决“Web Search 工具召回材料、LLM 结构化整理、采集层校验并入库候选原始文档”的问题，不做事件抽取、实体关联、图谱构建或投资判断。

## What Changes

- 新增 AI Web Research 采集 connector：由 `source_catalogs` 中的 active source 驱动，读取 `source_config` 中的 Web Search provider 参数、LLM 结构化参数、提示词、来源偏好和输出 schema 配置。
- 首期把 Web Search provider 抽象为可替换 adapter，默认纳入 `tavily`、`bocha_web_search` 和 `searchapi_baidu` 三类 provider profile：Tavily 用于全球通用搜索和已验证链路，博查用于中文 AI 搜索，SearchAPI Baidu 用于百度搜索结果补充。
- 由 Go 后端确定性调用搜索 API，获取 URL、标题、摘要、发布时间、来源名称、搜索排名、provider 原始元数据和可选网页正文，再交给 LLM 做结构化整理。
- 将 Web Search provider 与 LLM 的非敏感运行参数放入 `source_config`，包括 `search_provider`、`provider_profile`、`search_options`、`source_preferences`、`trusted_domains`、`llm_provider`、`api_base_url`、`api_protocol`、`model`、`prompt`、`prompt_version`、`freshness_window`、`max_results` 和输出 schema。
- 将真实 API key 继续放在环境变量或部署 secret 中，`source_catalogs.credential_ref` 或 `source_config.credential_refs` 只保存凭证引用名，不保存真实密钥。
- 定义 AI Web Research 返回结构化数组的契约，要求结果包含标题、正文或摘要、来源归因、发布时间、语言、地域、主题标签、证据摘录和相关性说明；来源归因优先使用真实 URL，也允许使用 provider 返回的来源名称、引用文本或检索来源说明。
- 将 Web Search provider 搜索结果和模型结构化输出标准化为原始文档候选对象，并通过现有 repository 幂等写入 `raw_documents`。
- 为内容来源质量增加明确标记：区分来自网页原文的 `fetched_source_text`、来自搜索摘要的 `search_snippet` 和来自模型总结的 `llm_generated_summary`。
- 保持采集层职责边界：不得在本 change 中生成事件、标签、实体关联、实体关系、利好利空判断、涨跌预测、传导强度或投资建议。
- 不在本 change 中引入独立 Agent 平台、独立图数据库、独立向量数据库或前端展示能力。

## Capabilities

### New Capabilities

- `ai-web-research-ingestion-connector`: 定义通过 Tavily Web Search、可选网页抓取、LLM 结构化整理和统一采集 connector 将开放互联网政经材料写入 `raw_documents` 的能力。

### Modified Capabilities

- `data-ingestion-layer`: 扩展采集通道，允许 `source_catalogs` 驱动 AI Web Research connector，并将非敏感搜索、模型与提示词参数保存在 `source_config`。
- `persistence-and-contracts`: 明确 `source_config` 可保存 Tavily 和 LLM connector 的非敏感运行参数，真实 API key 只能通过凭证引用指向环境变量或部署 secret。

## Impact

- 影响 `backend/internal/apps/ingestion/connectors`：新增或注册 `llm_web_research` connector，用于调用多个 Web Search provider、可选网页抓取和 LLM 结构化接口。
- 影响 `backend/internal/apps/ingestion/parsers`：新增或扩展解析器，将 AI 返回的结构化 items 转换为统一原始文档候选对象。
- 影响 `backend/internal/apps/ingestion/core`：可能需要补充 AI connector 所需的原始响应、来源质量、内容来源和元数据表达。
- 影响 `backend/internal/apps/ingestion/runtime` 与后续 scheduler：AI source 必须复用统一采集触发、并发、限流、失败隔离和 report 机制。
- 影响 `backend/data/source_catalogs`：需要新增 AI Web Research source seed 示例，配置 Tavily、博查、SearchAPI Baidu 等 provider profile、模型接口、提示词、来源偏好、输出 schema、凭证引用和启用状态。
- 影响 `backend/internal/repositories`：如现有 raw document 元数据不足以记录 AI 检索参数、prompt 版本和内容来源质量，需通过非破坏性方式补齐。
- 不影响 `frontend/miniapp/`、`prototype` 和 `doc`；不实现事件图谱抽取，该能力应留给后续独立 change。
