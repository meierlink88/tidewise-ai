## Context

当前系统已经建立采集源目录、connector、parser、runtime 和 `raw_documents` 写入边界。`event-knowledge-schema` 主规格也已经定义原始文档和事件事实分离：原始文档是证据层，事件事实必须通过后续抽取流程生成，并通过事件来源证据表关联原文。

本 change 设计的是采集后的 AI 语义处理流水线。它不是采集器的一部分，也不是前端展示能力。它要解决的是：原始材料入库后，如何可靠、可审计、可重试、可回放地提取事件、标签、实体关联和证据链。

## Goals / Non-Goals

**Goals:**

- 建立从 `raw_documents` 到结构化 `events`、事件标签、事件实体关联和事件证据的异步 AI 提取流程。
- 使用 job 表或等价持久化任务边界触发提取，采集流程只投递任务，不等待 AI 提取完成。
- 通过独立 worker 消费待处理任务，并支持重试、失败记录、跳过、幂等和人工或 CLI 重跑。
- 使用 repo 内版本化 prompt 文件和结构化输出 schema 管理 LLM 提取规则。
- 严格校验 LLM 输出，拒绝无证据、无来源、越界投资建议或无法映射的结构化结果。
- 把事件与实体基础库关联，记录关联类型、证据摘录、置信度和模型输出元数据。
- 使用 Go 单元测试、fake LLM、fixture 和 repository 边界验证提取流程，不在普通测试中调用真实模型或真实外部网络。

**Non-Goals:**

- 不在本 change 中实现 AI Web Research、RSS、网页或行情数据采集。
- 不在采集 connector 中同步执行事件提取。
- 不生成买入卖出、涨跌预测、利好利空结论、传导强度或投资建议。
- 不引入独立图数据库、向量数据库、LangChain、LangGraph 或外部 Agent 平台。
- 不实现小程序页面、管理后台、报告生成或订阅通知。

## Decisions

### Decision: 事件提取是独立子系统，不属于 ingestion connector

事件提取应放在 `backend/internal/apps/eventextraction` 或等价子系统内，和 `internal/apps/ingestion` 通过 `raw_documents` 与 job 表连接。采集器只负责写入原始材料和创建待提取任务，事件提取 worker 负责读取任务、调用 AI、校验输出和写入事件事实。

这样可以避免采集失败和模型失败互相污染，也允许 prompt 升级后对历史 raw document 重新提取。

### Decision: 通过持久化 job 表触发，而不是进程内回调

主触发方式是采集写入新 raw document 后，在同一事务或可靠补偿流程中创建 `event_extraction_jobs`。worker 轮询或领取 pending job 后执行提取。

补偿方式是 scanner 周期性扫描未处理或需要重跑的 `raw_documents`，补建 job。人工或 CLI 可以指定 raw document、source、时间窗口或 prompt 版本重跑。

进程内回调虽然简单，但无法可靠处理 worker 崩溃、部署重启、历史重跑和 prompt 版本升级。

### Decision: LLM 提取输出必须先进入候选模型，再写入事实表

LLM 输出先解析为 `ExtractedEventCandidate`、`ExtractedTagCandidate`、`ExtractedEntityLinkCandidate` 和 `EvidenceCandidate`。这些候选对象必须通过结构化校验、来源证据校验、实体匹配校验和安全边界校验后，才能写入事件事实表。

模型输出不得直接当作系统事实。无法校验的候选结果应记录为失败或跳过，并保留错误原因。

### Decision: 事件去重和证据追加是流水线核心能力

同一个事件可能来自多篇 raw document。系统应根据事件标题、发生时间、地点、参与方、事件类型、来源证据和内容哈希生成候选去重 key，并支持匹配已有事件后追加 evidence、tag 或 entity link。

首期可以使用确定性规则和可测试的相似度策略，不要求复杂语义聚类。后续如需更强语义合并，再通过独立 change 引入向量召回或外部 Agent 推理。

### Decision: 实体关联基于实体基础库，不让原文直连实体

事件提取 worker 可以读取实体基础库，用于名称匹配、别名匹配、证券代码匹配、市场代码匹配和实体类型约束。最终只允许事件实体关联表表达 event 到 entity 的关系，`raw_documents` 仍然不直接关联实体。

每条事件实体关联必须保存关联类型、证据摘录、置信度、匹配方式和模型或规则来源。

### Decision: prompt 和 schema 版本化

事件提取、标签标注、实体关联和事件合并的长 prompt 使用 repo 文件管理，例如：

```text
backend/data/prompts/event_extraction/
├── extract-events.v1.md
├── classify-tags.v1.md
├── link-entities.v1.md
└── merge-events.v1.md
```

运行记录必须保存 prompt 版本、模型 provider、模型名、schema 版本、输入 raw document ID、输出摘要和错误。真实 API key 仍通过环境变量或部署 secret 注入。

## Risks / Trade-offs

- [Risk] LLM 抽取结果存在幻觉或无来源事实。→ Mitigation：要求 evidence excerpt 和 raw document 引用，缺少证据不得写入事件事实。
- [Risk] 事件去重过弱导致重复事件。→ Mitigation：首期保留可解释去重 key 和 evidence 追加机制，后续再引入更强语义合并。
- [Risk] 实体匹配误关联。→ Mitigation：保存匹配方式、置信度和证据摘录；低置信度结果可标记为 pending_review。
- [Risk] 提取成本被采集量放大。→ Mitigation：通过 job 优先级、批次大小、模型配置、重试次数和 source 等级控制处理范围。
- [Risk] 提取失败阻塞采集。→ Mitigation：采集和提取异步解耦，采集成功不等待提取完成。
- [Risk] 未来需要图数据库。→ Mitigation：MVP 先在 PostgreSQL 保存事件、实体和关系，保留图谱投影边界。

## Migration Plan

1. 在后续实现阶段补充必要 migration：事件提取 job、extraction run、事件标签关联、事件实体关联或现有表缺失字段。
2. 增加 repository 测试，验证 job 领取、状态流转、幂等写入、事件 upsert、evidence 追加和实体关联写入。
3. 增加 fake LLM extractor 和 fixture，先用测试驱动候选解析、输出校验和安全边界。
4. 实现 `eventextraction` 子系统和 worker 入口，复用统一 config、database、repository、prompt loader 和 credential resolver。
5. 在采集写库路径中只增加 job 投递或待提取状态，不把 AI 提取逻辑放入 connector。
6. 运行 `go test ./...` 和 `openspec validate add-ai-event-extraction-pipeline`。

回滚策略：如果事件提取不稳定，可以暂停 worker 或将 job 创建开关关闭；已采集 raw document 保持不变，已写入事件事实需通过状态字段或审核流程降级，不得删除原始证据。

## Open Questions

- 首期事件类型、标签 taxonomy 和实体关联类型是否完全使用固定枚举，还是允许少量模型建议标签进入 pending_review。
- 低置信度实体关联是否在 MVP 直接写入 pending 状态，还是只记录在 extraction run metadata 中等待后续审核 change。
- 首期事件去重是否只做确定性 key，还是加入轻量文本相似度。
