## Context

当前采集层已经通过 `source_catalogs`、connector、parser、runtime 和 repository 写入 `raw_documents`。`source_config` 已作为 `source_catalogs` 的 JSONB 扩展字段存在，可保存 connector 专属的非敏感运行参数；真实凭证通过 `credential_ref` 指向环境变量或部署 secret。

本 change 的设计目标是把 Tavily Web Search 接入为开放互联网材料召回工具，并用大模型对召回结果做结构化整理。它不是事件图谱抽取能力，也不是外部 Agent 平台编排能力。它只负责按采集系统统一触发方式执行搜索、可选网页抓取、结构化整理和 raw document 候选对象写入。

## Goals / Non-Goals

**Goals:**

- 新增 `llm_web_research` connector 和 `llm_research_items` parser，使 Tavily + LLM Web Research 成为 `source_catalogs` 驱动的采集通道。
- 通过 `source_config` 保存非敏感运行参数：搜索 provider、搜索选项、LLM provider、API base URL、协议类型、模型名、提示词、提示词版本、结果条数、时间窗口、语言和输出 schema。
- 通过 `credential_ref` 或 `source_config.credential_refs` 引用真实 API key，禁止把 Tavily key、LLM key 写入 `source_config`、seed 文件、配置文件或源码。
- 约束模型返回结构化 JSON 对象，核心字段为 `items` 数组，每个 item 可转为一条原始文档候选对象。
- 记录内容来源质量，区分网页原文、搜索摘录和模型总结，供后续事件抽取判断证据质量。
- 单元测试使用 fake 或 `httptest` 验证 Tavily 搜索请求、LLM 结构化请求、响应解析、凭证缺失、非法 JSON、结构化校验和 raw document 转换。

**Non-Goals:**

- 不在本 change 中抽取事件、标签、实体关联或实体关系。
- 不在本 change 中生成投资建议、涨跌预测、利好利空、传导强度或事件评分。
- 不要求大模型天然具备联网搜索；首期搜索能力由 Tavily 提供，LLM 只负责结构化整理。DeepSeek tool-call loop 可作为后续增强，不作为首期必要路径。
- 不单独创建 prompt 文件目录或 prompt 模板表；MVP 阶段提示词作为 `source_config` 的运行参数。
- 不引入独立图数据库、向量数据库、外部 Agent 工作流平台或前端展示。

## Decisions

### Decision: AI Web Research 归属 ingestion connector

AI Web Research 的产物是原始外部材料，应进入 `internal/apps/ingestion` 子系统，并复用现有 connector、parser、runtime、source catalog 和 repository 边界。后续事件理解、实体关联和图谱投影应由独立 change 从 `raw_documents` 继续处理。

备选方案是继续使用 `eventgraph` change 承载该能力。该方案会把“找材料”和“理解材料”混在一起，导致 change 范围过大，不利于验证和审计。

### Decision: `source_config` 保存非敏感 connector 参数

AI connector 的搜索 provider、搜索选项、LLM API base URL、协议类型、模型名、提示词、时间窗口、结果条数和输出 schema 与具体采集源强相关，应该随 `source_catalogs` 一起治理、seed、审计和查询。因此这些非敏感参数放入 `source_config`。

真实 API key 仍通过 `credential_ref` 或 `source_config.credential_refs` 指向环境变量或部署 secret。代码不得从 `source_config` 读取真实密钥。

### Decision: 首期使用 Tavily 作为 Web Search 工具

首期不依赖 Qwen、DeepSeek 或其他大模型自带联网能力。connector 先由 Go 后端确定性调用 Tavily Search API，获取标题、URL、摘要、来源、发布时间、原始搜索元数据和可选 raw content，再把搜索结果交给 LLM 做结构化整理。

备选方案是直接让模型通过 Chat Completions 自己搜索。前期验证显示 Qwen 和 DeepSeek Chat 直接调用容易出现旧新闻、错误时间、伪造 model、缺失 URL 或空数组，不能作为稳定采集源。Tavily 作为明确搜索工具更利于审计、重试、限流和测试。

### Decision: LLM 结构化整理与搜索召回分离

connector 内部使用统一请求模型表达 search provider、LLM provider、protocol、model、prompt、search options、max results 和 timeout。Tavily 负责搜索召回；LLM 负责把搜索结果整理为 `items` JSON；parser 负责硬校验和 raw document candidate 转换。

DeepSeek Tool Calls 可以在后续版本中作为增强模式：模型先返回 `web_search` 或 `web_fetch` tool call，Go 后端执行 Tavily 或网页抓取，再回传工具结果。但首期优先实现确定性搜索/fetch + LLM normalizer，降低 agent 多轮不稳定性。

### Decision: 返回结构采用对象包裹 `items` 数组

模型响应必须解析为 JSON 对象，并包含 `items` 数组。每个 item 至少包含标题、来源归因、正文或摘要、内容来源类型和相关性说明。来源归因优先使用真实 URL；如果 provider 只返回来源名称、引用文本、搜索结果来源说明或模型可解释的来源描述，也可以入库，但必须标记为非 URL 来源归因。对象外层可以记录批次、查询时间、模型、提示词版本和统计信息。

备选方案是要求模型直接返回裸数组。裸数组难以扩展批次元数据，也不便于记录模型执行状态和错误。

### Decision: raw document 保留证据质量标记

如果 item 的正文来自真实网页抓取，应标记为 `fetched_source_text`；如果只来自搜索摘录，应标记为 `search_snippet`；如果是模型根据搜索结果总结，应标记为 `llm_generated_summary`。后续事件抽取不得把模型总结误认为原始新闻全文。

### Decision: 来源归因不强制要求 URL

AI Web Research 的入库门槛不是“必须有可点击链接”，而是“必须有可审计来源归因”。`source_url` 是最高优先级来源字段；当 provider 无法返回 URL 时，parser 可以接受 `source_name`、`source_reference`、`citation_text` 或 `provider_source_note` 等来源说明，并在 raw metadata 中记录 `source_attribution_type=url|named_source|citation_text|provider_note`。

如果 item 同时缺少 URL、来源名称、引用文本和 provider 来源说明，系统必须拒绝该条目。后续事件抽取应根据 `source_attribution_type`、`content_origin` 和 `retrieval_method` 判断证据强弱。

## Risks / Trade-offs

- [Risk] Tavily 参数或插件版本差异导致搜索失败。→ Mitigation：将 Tavily 请求参数类型、默认值、错误码和返回字段纳入 adapter 测试；Dify 插件参数差异不作为后端 Tavily adapter 的真实协议来源。
- [Risk] 不同模型 provider 的结构化能力差异大。→ Mitigation：通过 `api_protocol`、`llm_provider`、`output_schema` 和 provider-neutral 请求模型表达差异，首期用 fake/fixture 验证边界。
- [Risk] 模型返回非 JSON、字段缺失或编造来源。→ Mitigation：parser 必须严格校验 JSON schema、来源归因、标题、内容、内容来源类型和条数上限，失败时不写入伪造文档；没有 URL 但有来源说明的条目必须降级标记来源归因类型。
- [Risk] `source_config` 变成任意配置垃圾桶。→ Mitigation：为 `llm_web_research` 定义必填字段、禁止字段和类型校验，测试覆盖无效配置。
- [Risk] 提示词过长或频繁修改影响审计。→ Mitigation：保存 `prompt_version`、`prompt_purpose` 和执行元数据；后续如提示词治理复杂，再独立 change 引入 prompt 模板表或 repo prompt 文件。
- [Risk] Tavily 搜索和 LLM 结构化成本、限流不可控。→ Mitigation：复用 provider 级限流，要求 `max_results`、timeout、搜索深度和批次大小可配置，并在 report 中记录搜索成功、搜索失败、LLM 成功、LLM 失败和跳过数量。
- [Risk] 模型总结被误用为事实原文。→ Mitigation：强制记录 `content_origin` 和 `retrieval_method`，后续事件抽取可按证据质量降权或跳过。

## Migration Plan

1. 定义 Tavily + LLM AI Web Research source seed 示例和 `source_config` 字段约束。
2. 编写 Tavily adapter、LLM normalizer、parser 的配置校验、fake provider、结构化响应解析和 raw document candidate 转换测试。
3. 实现 `llm_web_research` connector、Tavily search adapter、LLM normalizer、`llm_research_items` parser 和 registry 注册。
4. 将 AI source 纳入现有 runtime/scheduler 可触发路径，复用 source catalog、credential resolver、rate limit 和 report。
5. 使用 fake Tavily 和 fake LLM 端到端验证搜索结果可以转为结构化 items，并幂等写入 raw document 候选对象。
6. 通过 gated smoke 验证真实 Tavily source 的搜索返回字段、错误处理和成本统计；LLM 真实调用同样必须显式启用，不得进入普通单元测试。

回滚策略：如真实 provider 不稳定，可将 AI Web Research source 的 `status` 改为 `inactive` 或 `disabled`，保留 connector 代码和历史 raw document，不删除已有采集数据。

## Open Questions

- Tavily 作为首期 Web Search 工具已确认；后续需要确定使用 Tavily 官方 API 还是 Dify Tavily workflow 作为生产调用入口。
- 首期 LLM normalizer 可以先支持 OpenAI-compatible provider，并通过 `source_config.llm_provider` 配置 Qwen、DeepSeek 或其他模型。
- DeepSeek Tool Calls 可作为后续增强模式，但不阻塞 Tavily deterministic search/fetch + LLM normalizer 的首期实现。
