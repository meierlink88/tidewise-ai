## Context

当前采集层已经通过 `source_catalogs`、connector、parser、runtime 和 repository 写入 `raw_documents`。`source_config` 已作为 `source_catalogs` 的 JSONB 扩展字段存在，可保存 connector 专属的非敏感运行参数；真实凭证通过 `credential_ref` 指向环境变量或部署 secret。

本 change 的设计目标是把具备联网搜索能力的大模型 API 接入为一种采集 connector。它不是事件图谱抽取能力，也不是外部 Agent 平台编排能力。它只负责按采集系统统一触发方式执行提示词搜索，返回结构化原始信息数组，并将每个条目按现有 ingestion 语义写入 `raw_documents`。

## Goals / Non-Goals

**Goals:**

- 新增 `llm_web_research` connector 和 `llm_research_items` parser，使 AI Web Research 成为 `source_catalogs` 驱动的采集通道。
- 通过 `source_config` 保存非敏感运行参数：API base URL、协议类型、模型名、提示词、提示词版本、搜索选项、结果条数、时间窗口、语言和输出 schema。
- 通过 `credential_ref` 引用真实 API key，禁止把 API key 写入 `source_config`、seed 文件、配置文件或源码。
- 约束模型返回结构化 JSON 对象，核心字段为 `items` 数组，每个 item 可转为一条原始文档候选对象。
- 记录内容来源质量，区分网页原文、搜索摘录和模型总结，供后续事件抽取判断证据质量。
- 单元测试使用 fake 或 `httptest` 验证模型 API 请求、响应解析、凭证缺失、非法 JSON、结构化校验和 raw document 转换。

**Non-Goals:**

- 不在本 change 中抽取事件、标签、实体关联或实体关系。
- 不在本 change 中生成投资建议、涨跌预测、利好利空、传导强度或事件评分。
- 不要求所有大模型都天然具备联网搜索；实际能力由 provider、API 协议和 source 配置共同决定。
- 不单独创建 prompt 文件目录或 prompt 模板表；MVP 阶段提示词作为 `source_config` 的运行参数。
- 不引入独立图数据库、向量数据库、外部 Agent 工作流平台或前端展示。

## Decisions

### Decision: AI Web Research 归属 ingestion connector

AI Web Research 的产物是原始外部材料，应进入 `internal/apps/ingestion` 子系统，并复用现有 connector、parser、runtime、source catalog 和 repository 边界。后续事件理解、实体关联和图谱投影应由独立 change 从 `raw_documents` 继续处理。

备选方案是继续使用 `eventgraph` change 承载该能力。该方案会把“找材料”和“理解材料”混在一起，导致 change 范围过大，不利于验证和审计。

### Decision: `source_config` 保存非敏感 connector 参数

AI connector 的 API base URL、协议类型、模型名、提示词、搜索选项、时间窗口、结果条数和输出 schema 与具体采集源强相关，应该随 `source_catalogs` 一起治理、seed、审计和查询。因此这些非敏感参数放入 `source_config`。

真实 API key 仍通过 `credential_ref` 指向环境变量或部署 secret。代码不得从 `source_config` 读取真实密钥。

### Decision: 首期使用 provider-neutral 请求模型

connector 内部使用统一请求模型表达 provider、protocol、model、prompt、search options、max results 和 timeout。首期可以先实现 OpenAI-compatible Chat/Responses 或 Qwen 兼容接口中的一种，但接口设计必须允许后续接入其他 provider。

备选方案是把 Qwen 请求格式直接散落在 connector 中。该方案实现最快，但会让后续 OpenAI、Perplexity、Tavily 或 Dify 类接口接入时重复改核心流程。

### Decision: 返回结构采用对象包裹 `items` 数组

模型响应必须解析为 JSON 对象，并包含 `items` 数组。每个 item 至少包含标题、来源 URL 或来源说明、正文或摘要、内容来源类型和相关性说明。对象外层可以记录批次、查询时间、模型、提示词版本和统计信息。

备选方案是要求模型直接返回裸数组。裸数组难以扩展批次元数据，也不便于记录模型执行状态和错误。

### Decision: raw document 保留证据质量标记

如果 item 的正文来自真实网页抓取，应标记为 `fetched_source_text`；如果只来自搜索摘录，应标记为 `search_snippet`；如果是模型根据搜索结果总结，应标记为 `llm_generated_summary`。后续事件抽取不得把模型总结误认为原始新闻全文。

## Risks / Trade-offs

- [Risk] 不同模型 provider 的联网搜索能力差异大。→ Mitigation：通过 `api_protocol`、`search_enabled`、`search_options` 和 provider-neutral 请求模型表达差异，首期用 fake/fixture 验证边界。
- [Risk] 模型返回非 JSON、字段缺失或编造来源。→ Mitigation：parser 必须严格校验 JSON schema、URL、标题、内容、内容来源类型和条数上限，失败时不写入伪造文档。
- [Risk] `source_config` 变成任意配置垃圾桶。→ Mitigation：为 `llm_web_research` 定义必填字段、禁止字段和类型校验，测试覆盖无效配置。
- [Risk] 提示词过长或频繁修改影响审计。→ Mitigation：保存 `prompt_version`、`prompt_purpose` 和执行元数据；后续如提示词治理复杂，再独立 change 引入 prompt 模板表或 repo prompt 文件。
- [Risk] AI 搜索成本和限流不可控。→ Mitigation：复用 provider 级限流，要求 `max_results`、timeout 和批次大小可配置，并在 report 中记录成功、失败和跳过数量。
- [Risk] 模型总结被误用为事实原文。→ Mitigation：强制记录 `content_origin` 和 `retrieval_method`，后续事件抽取可按证据质量降权或跳过。

## Migration Plan

1. 定义 AI Web Research source seed 示例和 `source_config` 字段约束。
2. 编写 connector/parser 的配置校验、fake provider、结构化响应解析和 raw document candidate 转换测试。
3. 实现 `llm_web_research` connector、`llm_research_items` parser 和 registry 注册。
4. 将 AI source 纳入现有 runtime/scheduler 可触发路径，复用 source catalog、credential resolver、rate limit 和 report。
5. 使用 fake provider 端到端验证 100 条结构化 item 中的少量 fixture 可以幂等写入 raw document 候选对象。
6. 待用户提供 Qwen API 调用方式后，通过独立验证或 gated smoke 判断 Qwen 接口是否支持稳定联网搜索，再决定是否启用真实 Qwen source。

回滚策略：如真实 provider 不稳定，可将 AI Web Research source 的 `status` 改为 `inactive` 或 `disabled`，保留 connector 代码和历史 raw document，不删除已有采集数据。

## Open Questions

- Qwen 目标 API 是 OpenAI-compatible Chat、Responses、DashScope 原生接口，还是百炼应用 API？
- Qwen API 的联网搜索返回中是否包含可解析的 URL、引用、网页正文或仅有模型总结？
- 首期是否需要二次 `web_fetch` 抓取模型返回的 URL，还是先只保存模型返回的结构化结果并标记内容来源？
