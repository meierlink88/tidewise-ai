# Theme Analysis 系统侧需求与设计确认

## 状态

设计已冻结。用户于 2026-07-29 明确确认全部架构级需求决策已关闭并允许进入
`to-spec`；后续实现必须以本文及对应 GitHub Issue 为边界。设计冻结前未修改代码、
数据库或 API。

最近核对：2026-07-29。

实施 Issue：[Build Theme MVP: real-time Analysis Context and atomic strong-lineage publication
#141](https://github.com/meierlink88/tidewise-ai/issues/141)。

## 1. 目标与非目标

目标闭环：

```text
Data Service 提供可信且可复现的 Analysis Context
→ Codex theme-analysis 执行四层推理并生成 Candidate
→ Data Service 确定性校验、保存和发布
→ Miniapp 读取已发布 Theme / Reason Tree
```

本轮不实施，不修改 Miniapp，不让 Codex 直连 PostgreSQL 或 Neo4j，不把四层推理策略
写入 Data API，也不新增 Connector。Neo4j Graph Context API 可后续建设，不阻断首份
MVP；第一版允许 Data Service 基于 PostgreSQL 中已有产业链、Chain Node 和关系事实组装
上下文。

Theme 不是 Event 的后续状态，也不存在 Event “升级”或“晋升”为 Theme。Codex 分析师
Agent 自主决定是否形成 Theme、形成几个 Theme、使用一个还是多个 Event；系统侧不建设
对应状态机、阈值或任务机制。

## 2. 权威来源

- `docs/architecture/event-variable-semantics-phase-one.md`
- `docs/architecture/research-theme-reasoning-tree-spec.md`
- `docs/architecture/agentrun/first-real-research-report-mvp-v1.md`
- `docs/contexts/data/CONTEXT.md`
- `docs/contexts/agentrun/CONTEXT.md`
- `analyse-data-service/backend/api/data/v1/openapi.yaml`
- `agent-run/backend/api/agentrun/v1/openapi.yaml`
- `../research/methodology/investment-reasoning-methodology-v2.md`
- `../research/methodology/investment-research-report-template-v2.md`
- `../theme-analysis/docs/architecture/theme-generation-orchestration-v1.md`

方法论决定研究语义；Data OpenAPI、Context 和实现决定系统事实。两者冲突时不得让
Codex 自行猜测。

## 3. Owner Map

| 能力或事实 | Owner | 边界 |
|---|---|---|
| Event、Evidence、Entity、关系、TBox、accepted Event Semantics | Data Service | PostgreSQL 正式事实与版本化 API |
| Event Semantic Work Item、Agent Execution、模型调用与自动审核执行 | AgentRun | 不拥有 Data 事实；不运行或监控 Theme 分析师 |
| 四层推理、反方复核、Investment Analysis、Theme/Tree Candidate | 工程外部的 Codex `theme-analysis` | 不属于本工程部署单元；不直写数据库，不编造正式 ID |
| Theme、Theme Impact、Reason Tree、发布回执 | Data Service | 确定性校验、持久化、发布可见性 |
| Neo4j | Data Service 管理的可重建投影 | 本阶段不是首份 MVP 的硬依赖 |
| Miniapp | Miniapp Backend / Frontend | 只读取已发布结果，本轮 UI 不改 |

## 4. Phase One 收尾门禁

### 4.1 已核实的前瞻 Signal 契约差异

Data DTO、AgentRun 领域类型和 PostgreSQL 已能承载：

- `statement_at`
- `valid_from` / `valid_until`
- `forecast_period_start` / `forecast_period_end`
- `extraction_confidence`
- 结构化 `MeasurementValue[]`

但当前 Event Semantic Generator 的提示输出 Schema 只声明 `measurements: []`，没有
声明上述时间、置信度和 Measurement 元素结构。模型即使理论上能输出，当前合同也没有
稳定引导其完整生成。

这属于首份真实 Theme 的强制门禁：政策 `stated_intent`、企业
`source_forecast` 和财报区间值未通过真实链路前，不得用合成供给场景替代生产验收。

### 4.2 已核实的 Event Context / Evidence 差异

当前 Event Semantic Context 向 Agent 提供 Event `occurred_at`，Evidence 提供 excerpt、
source level、relation、supports fields 和 `raw_document_id`，但没有提供 Raw Document
`published_at` 或结构化声明主体。

`published_at` 不能替代 Event `occurred_at`；两者都不能自动替代 `statement_at`。
对于政策意图和来源预测，缺少声明时间或声明主体时应形成明确缺口、拒绝或隔离，不能
由模型用采集时间补齐。

### 4.3 Golden Scenario 最小范围

真实生产验收仍需使用既有 Collector → Event Fact Extractor → Data Event →
Event Semantic 链路覆盖：

1. 供给冲击：实际供给或产量变化，并至少产生一条 `rule_inferred` Direct Impact；
2. 政策：明确声明主体、声明时间和 `stated_intent`，允许因缺少 approved Rule 而不产生
   Direct Impact；
3. 企业订单或财报：实际值或 `source_forecast`，覆盖结构化单值或区间 Measurement；
4. 行业需求：实际或预测需求变化，并至少覆盖一条 `event_explicit` Direct Impact。

至少包含 `event_explicit`、`rule_inferred` 正向案例，以及确定性拒绝、Reviewer 拦截、
重试成功和重试耗尽隔离案例。现有 Connector 能覆盖时不得把新增 Connector 混入本阶段。

Theme Golden Scenario 只用于验证 Codex 分析师能否正确消费系统输入并生成符合发布合同
的 Theme Aggregate，不形成 Data Service 的 Theme readiness 或 Event 数量门禁。Agent
能力验收可分别覆盖：

- 单个高权威 Event 形成 Theme；
- 多个独立 Event 或 Signal 汇聚形成 Theme；
- 信号冲突、证据不足或无投资意义时产生零 Theme；
- 没有 Direct Impact 时，仍能基于正式实体关系和产业链关系完成合理推理；
- 不同 prediction horizon 下对同一批 Event 形成不同结论。

这些场景的期望结果由 Codex 分析师方法论和 Golden fixture 定义，不由 Data Service
Acceptance Policy 决定。

#### 当前本地数据就绪度（2026-07-29 只读复核）

Data PostgreSQL：

- `events = 8`，且 8 条均为 `confirmed + verified`、各有 Evidence；
- `accepted VariableSignal = 0`；
- `accepted DirectImpactAssertion = 0`；
- `research_themes = 3`、`research_reasoning_trees = 2`，属于既有 Theme/Tree 合同数据，
  不能证明 Phase One → Theme 新链路；
- 现有 8 条 Event 与 Phase One 12.1 记录一致，仍不足以覆盖四类真实正向 Golden。

AgentRun PostgreSQL：

- `event_fact_canonical_events = 8`；
- `event_publication_journal = 8`，全部 `acknowledged`；
- Event extraction units：`published = 8`、`rejected = 16`、`no_events = 2`；
- `event_semantic_work_items = 0`。

因此当前真实数据链路停在“Event 已发布”，尚未进入 Event Semantic Worker。实时批量
Analysis Context 即使完成，也会得到零个 accepted Signal/Impact，不能用于首份真实
Theme/Reason Tree 验收。合成 E2E 使用独立临时数据库，只证明工程通路，不替代该门禁。

### 4.4 `entity_role`

当前 Data 契约只要求非空字符串，尚未发现工程侧权威词汇表。经“事件推理模型”核对，
MVP 最小受控词汇冻结为：

- `event_subject`：Event 直接描述其状态或行为的实体；
- `actor`：实施行为的实体；
- `affected_entity`：被 Event 直接影响但不是陈述主语的实体；
- `statement_source`：政策、指引或预测的原始声明主体；
- `event_object`：行为直接指向的对象；
- `context`：只用于消歧、范围或背景的实体。

约束：

- 使用 `event_subject`，不使用语义含混的 `subject`；
- 同一实体可具有多个 role；
- `event_subject` 只是 VariableSignal subject 的默认候选，仍需独立语义断言；
- `affected_entity` 不能自动转化为 Direct Impact；
- 媒体转载者不是 `statement_source`，应记录原始声明主体；
- buyer、seller、producer、regulator 等细粒度角色暂不扩充枚举，优先由 EntityRelation
  与 mechanism 表达。

### 4.5 工程治理分类

| 问题 | 分类 | 处理要求 |
|---|---|---|
| Event Fact 与 Event Semantic 顺序共享一个 Shutdown deadline | 首份真实生产运行前门禁；不阻断本轮需求设计 | 两个 Worker 必须各自获得有界取消/等待机会，不能因前一 Worker 耗尽 deadline 而跳过后一 Worker |
| Data PostgreSQL Adapter 内存在审核决策和状态传播 Biz 逻辑 | 并行技术债；不阻断 Analysis Context 合同设计 | 不在新 Analysis Context Adapter 中继续扩散该模式；领域选择留在 Biz |
| Data CONTEXT 仍声明旧目录结构，与 Kratos 标准冲突 | 下一阶段实施前文档门禁；不阻断本轮 Grill | 实施前统一权威文档，避免 Agent 收到冲突指令 |

### 4.6 Phase One 收尾门禁

需求设计冻结与真实数据就绪分开。Theme MVP 可以在架构决策关闭后进入实现，但以下项目
完成前，不得宣布首份真实 Theme/Reason Tree 生产验收通过：

1. Event Semantic Generator Schema 完整声明 `statement_at`、valid/forecast period、
   `extraction_confidence` 和 Measurement 元素结构；
2. Event Semantic Context 从 Data 正式数据提供 statement source、Evidence 逐字 excerpt
   与必要来源/知识可得时间元数据；
3. `entity_role` 六项受控词汇在 Data DTO/Biz 确定性校验与 Agent 输出合同中一致；
4. AgentRun 为选定真实 Event 建立并闭环 Event Semantic work item，Data 产生 accepted
   Signal；供给正例同时产生 accepted rule-inferred DirectImpact；
5. 四类真实 Golden positive 与关键 negative/quarantine fixture 通过既有
   Collector/Event Extractor/Event Semantic 链路；
6. Event Fact 与 Event Semantic Worker 的 Shutdown 等待均获得独立有界机会。

实施顺序建议：

```text
Phase One 合同补齐
→ Golden Data Preparation（使用既有链路）
→ 实时批量 Analysis Context
→ Theme Aggregate 原子发布与强血缘
→ 外部 Codex Golden 验收
```

Data PostgreSQL Adapter 的既有 Biz 逻辑和旧 CONTEXT 目录描述按 4.5 分类并行治理，不扩大
到新能力，但不因其存在阻断需求冻结。

## 5. Theme Analysis Context

### 5.1 当前事实

现有 `GET /api/data/v1/events/{event_id}/semantics` 是单 Event 完整语义和审核状态读取，
适合审计与故障排查。它不提供时间窗口发现、跨 Event 批量 accepted 选择、产业链上下文、
稳定分页或分析快照，因此不足以作为 Codex 的正式批量生产输入。

### 5.2 最小逻辑内容

Analysis Context 至少需要：

- 明确的 `analysis_as_of`、Event discovery window 和时间边界；
- 仅 `confirmed + verified` 的 Event；
- 每个 Event 最新、`accepted`、未 `superseded` 的 EventEntityLink、
  VariableSignal、Measurement 和 DirectImpactAssertion；
- 相关 Entity、Industry Chain、Chain Node、membership、正式关系和必要 Graph Edge；
- Evidence 与 RawDocument 血缘；
- VariableDefinition、适用 EntityType、DirectTransmissionRule、AcceptancePolicy；
- 每类 TBox、规则和策略的版本；
- 稳定排序、游标分页、完整性标记和查询契约版本。

Data Service 负责事实选择、状态门禁、引用完整性和同次查询一致性；不负责判断哪些 Event
应聚合为 Theme、推导多跳投资方向或选择最佳产业链路径。

Data Service 也不负责判断 Evidence 是否足以形成 Theme、Signal 冲突应如何处理、是否
具有投资价值，或最少需要几个 Event / Direct Impact。这些均属于 Codex 分析师 Agent。

时间与未来信息隔离语义：

- Event discovery window 使用“信息可被系统获知的时间”，不得只用 `occurred_at`；
- Evidence 只有在 `published_at <= analysis_as_of` 且首次发现或采集时间
  `<= analysis_as_of` 时才能进入本次查询；
- 对历史数据，可用 `max(published_at, first_seen_at)` 形成可审计的
  `knowledge_available_at`，避免回测读取后来补录的信息；
- “最新版”是 `analysis_as_of` 当时已知的最新 accepted、未 superseded 版本，不是今天
  回看时的最新版；
- Theme MVP 依赖分析期间正式语义数据不发生并发更新；若该前提不成立，Codex 丢弃本次
  分页结果并重新查询；
- 已过 `valid_until` 的 Signal 可保留为历史证据，但不能作为当前有效驱动。

Codex 只接收 Data Service 正式 Evidence 与来源血缘，不读取 RawDocument 完整正文，也
不允许绕过 Data Service 访问 AgentRun Artifact、Collector 原始响应或 AgentRun 数据库。

最小 Evidence 合同必须提供：

- `evidence_id`、`evidence_hash`、relation 和被支撑对象的类型/ID；
- 未改变原意的逐字 `excerpt`，保留主体、否定/条件词、数值、单位和时间期间；
- `raw_document_id` 作为 Data 血缘引用；
- title、source name/type、source URL 或稳定 locator；
- `published_at`、`first_seen_at/ingested_at`、`knowledge_available_at`、accepted time；
- primary/supporting 等 Evidence 角色；
- 对 `stated_intent` / `source_forecast`，额外返回 statement source、`statement_at` 和
  有效/预测期间。

RawDocument ID 不能替代 Evidence，也不赋予 Codex 读取 AgentRun 原始材料的权限。对于
`rule_inferred` Impact，Data 还应返回 Signal Evidence、DirectTransmissionRule ID/版本
和 EntityRelation ID。

明确不进入 Analysis Context：

- RawDocument 完整正文、HTML、PDF、图片或 OCR 全文；
- AgentRun Artifact 内容/SHA、Collector 原始响应或中间文件；
- AgentRun 提取器输入输出、Prompt 或完整模型回复；
- rejected/pending 候选正文、向量或未进入 Data 正式 Evidence 的片段；
- AgentRun 数据库中的任何记录。

正式 Evidence 不足时，Codex 只能继续查询 Data Service 正式证据、缩小结论、降低置信、
删除无依据路径、记录 Evidence Gap 或产生零 Theme；不得从标题猜测正文或自行补全事实。

### 5.3 已确认：D-TA-001 实时查询模式

MVP 采用实时查询，不建设或保存不可变 Analysis Context Snapshot。Codex
`theme-analysis` 在分析开始时通过 Data Service 查询正式数据；Data Service 只负责同步
查询和一致性选择，不承担分析任务、Run 或调度。

实时查询规则：

- Codex 固定 `analysis_as_of`，表达本次投研站在哪个时点判断；
- 后续页面绑定相同的 discovery window、`analysis_as_of`、过滤条件、查询契约和
  TBox/规则版本；
- 使用包含唯一终结字段的稳定总排序和不透明游标，不使用 offset 分页；
- 每页只返回查询时为 latest、accepted、non-superseded 的对象；
- 返回对象必须具有稳定 ID，以及在领域中存在的显式版本；
- 游标失效、参数不一致或检测到数据变化时，整次分析重新开始，禁止拼接新旧页面。

Theme 生成批次保存最小输入血缘：

- `analysis_as_of`、discovery window；
- 查询条件和查询契约版本；
- 实际使用的 Event、Signal、Impact、Relation、Evidence ID，以及领域存在时的显式版本
  或内容 hash；
- 实际使用的 VariableDefinition 和 DirectTransmissionRule 版本。

这是一份输入身份清单，不是 Analysis Context Snapshot，不保存完整查询 Payload。
整体 fingerprint 可作为后续增强，不是 MVP 强制项。

#### 5.3.1 已确认：D-TA-013 不建设状态水位

用户确认 Theme 分析期间中间语义数据不会更新，MVP 不关心跨页状态水位。因此不为
Analysis Context 新增 `accepted_at`、`superseded_at`、状态历史、短时 Token 或选择清单，
也不提供 point-in-time 分页保证。

现有 Event Semantic reanalysis 仍可能原地把旧对象状态改为 superseded，但并发 reanalysis
不属于本次 Theme MVP 保证范围。如果运行前提被打破或 Codex 观察到游标/对象状态不一致，
必须从第一页重新查询；未来需要并发重分析时，再独立引入状态有效期或 Snapshot 机制。

明确延后：

- Analysis Context Snapshot 表或领域对象；
- snapshot prepare/read/expire/cleanup API；
- 完整输入 Payload 的不可变存档、差异比较与历史恢复；
- Data Service 内的分析 Run/Task 调度；
- 长时间数据库事务或长期读锁；
- PostgreSQL 与 Neo4j 的跨存储一致性快照；
- 任意历史时点完整重放和永久 fingerprint 签名。

### 5.4 已确认：D-TA-002 Event discovery window 所有权

Theme MVP 的 Event discovery window 由 Codex 分析师根据研究目标显式决定。Data Service
不设置 7 天、30 天等固定投研默认窗口，也不判断什么时间范围更适合生成 Theme。

正式 Theme 查询必须提交：

- `discovery_window_start`，含边界；
- `discovery_window_end`，不含边界；
- `analysis_as_of`；
- 当输出包含未来趋势、投资方向或利好利空判断时，同时提交
  `prediction_horizon_start` / `prediction_horizon_end`。

Data Service 只执行确定性校验：

- 时间格式、时区、精度和先后顺序合法；
- `discovery_window_end <= analysis_as_of`；
- 预测周期存在时，起点不早于 `analysis_as_of` 且终点晚于起点；
- Evidence 的 `knowledge_available_at <= analysis_as_of`；
- 后续游标继续绑定同一窗口、`analysis_as_of` 和过滤条件；
- 分页、授权、查询成本和资源保护等技术边界。

缺少显式窗口时，正式查询应拒绝，不能补默认值。窗口超过技术预算时，Data Service
应明确报错，不能静默截断或自行缩短；由 Codex 决定缩小窗口或按同一
`analysis_as_of` 分段读取并按对象稳定身份去重。

discovery window 按 `knowledge_available_at` 选择信息；`occurred_at` 只表达事件实际
发生时序和 Signal 时间资格，两者不得互相替代。允许历史查询，但当系统缺少当时的
accepted/superseded 和 TBox 版本历史时，只能标记为“回顾性重建”，不能宣称为严格
历史回测。MVP 响应固定显式返回
`temporal_semantics=retrospective_reconstruction` 与非空 `temporal_limitation`；
Event ABox 按 `analysis_as_of` 过滤，TBox、Entity 与关系字典同时排除该时点之后创建或
更新的事实，但由于本阶段不建设状态历史，仍不宣称任意历史时点的严格重放。已知
accepted/superseded 状态无法恢复时直接返回 `422`。

### 5.5 已确认：D-TA-009 实时批量 Analysis Context

Data Service 提供一个实时批量 Analysis Context 查询能力。Codex 显式提交 discovery
window、`analysis_as_of` 和分页参数；Data Service 返回查询时最新的正式 Event
semantics、Evidence、Entity、产业链关系和必要 TBox。

分页基本单位是完整 `EventSemanticBundle`。同一个 Bundle 必须包含：

- Event 核心字段、时间和正式状态；
- accepted/latest/non-superseded EventEntityLinks；
- VariableSignals 及其 Measurements；
- 每个 Signal 的完整 `direct_impacts` 集合；
- Event、Signal 和 DirectImpact 使用的逐字 Evidence excerpt 与来源元数据；
- Bundle 内引用 Entity 的 ID、类型和规范名称。

Bundle 不得跨页拆分；page size 按 Event Bundle 数量计算。Evidence excerpt 随 Bundle
返回，不延迟到其他页面。

Canonical Entity 完整投影、IndustryChain、ChainNode、关系、VariableDefinition、
EntityTypeDefinition、DirectTransmissionRule、RelationDefinition 和 AcceptancePolicy
可以通过响应字典去重或跨页复用，但必须携带稳定 ID、领域存在时的显式版本、关系方向，
并在相同 TBox 版本下可确定性解析；不允许出现无法解析的裸 ID。

DirectImpact 可为空，必须表达为：

```json
{
  "signal_id": "...",
  "direct_impacts": []
}
```

空数组表示该页查询时没有 accepted/latest/non-superseded DirectImpact，不表示
负向影响、未加载或错误，也不得生成“暂无影响”的伪对象。

接口只允许将同一 Event 的正式对象技术组装为 Bundle，不得：

- 聚合多个 Event 或净额计算 Signal；
- 判断 Theme readiness、Evidence 是否充分或最少 Event 数量；
- 解决信号冲突、选择主要研究对象、主产业链或最佳路径；
- 计算投资方向、评分或执行多跳推理；
- 生成 Theme/Reason Tree，调用 LLM，或保存/调度分析任务。

零个合格 Event 是合法空结果，不返回错误，也不附带 Theme readiness 判断。

#### 5.5.1 HTTP 与分页合同

MVP 冻结为同步无状态查询：

```text
GET /api/data/v1/research-analysis-context
```

第一页显式传入：

- `discovery_window_start`
- `discovery_window_end`
- `analysis_as_of`
- 可选 prediction horizon
- `page_size`

后续页使用 Data 返回的不透明 `cursor`。Cursor 绑定查询参数、排序和 TBox 版本；调用方
不能修改或拼接。响应返回有效查询参数、contract version、TBox version、
EventSemanticBundles、解析所需字典、`next_cursor` 和 `has_more`。

- 合法空结果返回 `200` 和空 bundles；
- 参数、时间顺序、page size 或 cursor 错误返回 `400`；
- 统一 Token 认证失败返回 `401/403`；
- 请求的严格历史语义无法满足时返回明确 `422`，不得伪装成严格回测；
- 资源保护、暂时不可用和内部错误使用标准 `429/503/500`；
- Data 在 PostgreSQL 聚合嵌套 JSON 前先执行窗口、Bundle 行数/原始字节和字典
  行数/原始字节 preflight；通过 preflight 后仍执行最终编码字节预算；
- 不静默截断窗口或 Bundle；超过技术预算时明确失败，由 Codex 缩小或分段查询；
- 同一 Bundle 过大时返回稳定错误，不把其子对象拆页；
- 所有错误使用标准 ErrorEnvelope 和 Request ID。

读取接口没有写入幂等身份或 point-in-time 水位。不透明 cursor 只保证查询参数和稳定
排序连续性。服务端与客户端超时使用现有可配置基础设施边界，不在领域合同中写死秒数。

### 5.6 已确认：D-TA-010 Codex 认证

工程外部 Codex 复用 Data Service 现有统一 API Token，不新增 Codex 专用 Service
Identity、Credential 表、Token 签发或轮换机制。

- Analysis Context 与 Theme Aggregate 发布均复用现有认证中间件；
- Token 只通过既有认证 Header 传递，不进入请求 Body；
- Token 或其派生值不得写入日志、数据库、发布回执或错误响应；
- 复用统一 Token 不扩大系统边界：Codex 仍只能调用 Data Service API，不能直连
  PostgreSQL、Neo4j 或访问 AgentRun 数据。

## 6. Theme / Reason Tree Candidate 与发布

### 6.1 已可复用能力

Data Service 已有两个正式发布合同：

- `POST /api/data/v1/research-theme-imports`
- `POST /api/data/v1/research-reasoning-tree-imports`

它们已经具备严格 DTO、正式引用校验、确定性顺序、canonical hash、幂等回放、冲突、
事务原子性和不可变回执，这些能力可复用。

用户确认现有 Theme/Reason Tree API 尚未形成需要向后兼容的正式外部合同，因此其路径、
DTO、聚合边界和发布语义都可直接调整。现有“Theme 先可见、Reason Tree 随后独立发布”
行为不是新设计的兼容约束。

### 6.2 尚缺能力

- Theme 与 Reason Tree Candidate 的组合式确定性校验与原子发布；
- 保存 Theme 生成批次实际使用的输入对象稳定身份清单和查询时间边界；
- 将正式 VariableSignal/DirectImpactAssertion UUID、Submission、Evidence 血缘和
  Reason Tree 显示快照建立可审计关系。

现有 Data 发布校验明确不判断研究语义是否正确。研究语义门禁属于 Codex
`theme-analysis`；Data 仍需对 Candidate 的 DTO、引用、状态、所有权和血缘做确定性
校验。

### 6.3 已确认：D-TA-003 发布 Service 内整体校验

MVP 不新增独立 prepare-only API、Publication Plan、预校验 Token 或临时状态资源。
Codex 一次性提交完整 Theme Aggregate Candidate；Data Service 的一个发布 Service 方法
负责“整体确定性校验 + 原子保存”。

校验范围仅包括：

- DTO 和枚举合法；
- Theme 与 Reason Tree 内部引用完整；
- 正式 Entity、IndustryChain、ChainNode、Event、Signal、Impact、Evidence 等 ID 存在，
  领域存在显式版本时版本匹配，且状态可用；
- Analysis Context 输入身份清单和 Evidence 血缘完整；
- canonical identity、幂等身份和现有正式发布合同能够接受该 Candidate。

Data Service 不判断研究结论、投资方向、传导逻辑或证据解释是否正确；这些研究语义门禁
仍由 Codex `theme-analysis` 负责。

执行与事务边界：

1. 可脱离数据库完成的 DTO、枚举和纯结构检查可在开启事务前快速失败；
2. 正式 ID/显式版本、accepted/latest/non-superseded 状态、幂等和冲突等依赖数据库当前状态
   的检查，与 Theme/Reason Tree 写入处于同一事务边界；
3. 任一检查失败即返回结构化错误，事务不产生任何 Theme/Reason Tree 写入；
4. 全部检查通过后，在同一事务内原子保存一条 Theme 及其全部 Reason Trees；
5. canonical candidate hash 仍可作为幂等与冲突检测的内部实现，不需要成为两次 API 调用
   之间的握手协议。

### 6.4 已确认：D-TA-004 Theme Publication Aggregate

经事件推理模型确认，正式 Theme 必须至少包含一棵完整 Reason Tree。Theme 与其全部
Reason Trees 构成同一个 Publication Aggregate；Reason Tree 不是可脱离 Theme 独立发布
的内容。

```text
Theme Publication Aggregate
├── 1 Theme
└── 1..N Reason Trees
```

发布边界：

- Theme Candidate 及其全部 Reason Tree Candidates 整体提交和校验；
- 只有结构、传导路径、时间资格、正式语义引用和 Evidence 血缘全部通过后才能发布；
- 以“一条 Theme + 其全部 Reason Trees”为事务原子单位，要么全部发布，要么全部不发布；
- 一次分析可生成零个、一个或多个 Theme Aggregate；
- 不同 Theme Aggregate 彼此独立，一个失败不阻断其他已通过的 Aggregate；
- “零 Theme”是 Codex 侧合法分析结果，不向 Data 发布空 Theme 或占位 Reason Tree。

Candidate、失败原因和零 Theme 状态属于工程外部 Codex 运行态，不进入 Data Service。
Data 只接收待发布 Aggregate；正式发布失败后由 Codex 修正或重新生成 Candidate。已发布
Reason Tree 不独立修订。

### 6.5 已确认：D-TA-005 Theme 不更新

分析师重新分析时创建新的 Theme Publication Aggregate，不更新、覆盖、替换或为既有
Theme 建立版本链。已发布 Theme 及其全部 Reason Trees 保持不可变。

需要区分：

- 同一次发布请求因超时或网络失败而重试：使用相同幂等身份，返回同一个发布结果，不重复
  创建 Theme；
- 新的一次分析产生新结论：使用新的发布身份和 Theme ID，即使主题相近，也创建独立的
  Theme Aggregate。

MVP 不建设 `current_version`、supersede/replaced_by、Theme 更新 API 或 Reason Tree 独立
修订机制。现有读取合同按 `published_at` 的查询窗口展示最新内容，因此旧 Theme 可保留
审计并自然离开最新列表。本轮不新增 Theme 失效、过期或撤回状态；未来如出现合规撤回等
独立需求，再单独设计。

正式 Signal/Impact 后来被 supersede，不回写、动态改文案或自动失效既有 Theme。历史
Theme 继续引用发布时使用的正式对象版本；新的分析如形成结论，则创建新的 Theme
Aggregate。

### 6.6 已确认：D-TA-011 正式推理输入强血缘

现有 Theme/Reason Tree 合同中的 `variable_signal_key + direction + display_summary` 仅为
分析师显示快照，且明确不校验正式 VariableSignal/Evidence。该设计不能满足 Phase One
后的可追溯目标，实施前必须由新合同覆盖。

Reason Tree 必须区分三种语义：

1. **正式 VariableSignal**：节点展示 Data Service 正式 Signal 时，保存
   `variable_signal_id` 及其 `semantic_submission_id`；
2. **正式 DirectImpactAssertion**：一跳影响属于传导边语义，保存
   `direct_impact_assertion_id` 及其 `semantic_submission_id`；若不新增独立 Edge 表，
   可存于下游节点 incoming transmission；
3. **Codex analyst inference**：没有正式 Signal/Impact 的多跳分析师推导，必须明确标记
   为 inference，引用上游正式 Signal/Impact 和实际使用的 Entity/IndustryChain
   Relation，不得伪造正式 ID。

强引用与显示快照同时保留：

- 强引用回答“依据哪个正式事实版本”；
- direction、Measurement 显示值、display summary、transmission summary 等不可变显示
  snapshot 回答“发布时用户看到什么”；
- 读取时不得根据当前 Signal、Entity 名称或规则动态重写已发布显示内容。

Theme/Tree Event association 只用于来源索引、展示和统计，不能替代 Signal/Impact/Evidence
血缘。最小正式路径是：

```text
Tree Node
→ VariableSignal ID + semantic_submission_id
→ Evidence ID + evidence_hash
→ Event ID
```

或：

```text
Incoming Transmission
→ DirectImpactAssertion ID + semantic_submission_id
→ VariableSignal + Evidence
→ DirectTransmissionRule / EntityRelation
```

Phase One 工程事实是：Signal/Impact 重分析时创建新 UUID，旧 Submission 及其对象被
supersede；对象没有独立 version 列。因此 UUID 本身就是具体事实版本身份，MVP 不新增
冗余 `variable_signal_version` 或 `direct_impact_assertion_version`。该决定要求：

- Signal/Impact 语义内容不得在同一 UUID 上原地覆盖；
- 新分析生成新 UUID；
- superseded 旧对象长期保留可读；
- 已发布 Theme 不自动跟随新 UUID；
- 有显式版本的 VariableDefinition、DirectTransmissionRule 和 Relation 继续固定版本；
- Evidence 使用不可变 ID 与 `evidence_hash` 固定逐字内容。

Data 发布 Service 必须确定性校验并在发布回执/强关联中证明：

- Signal/Impact ID 存在，且在发布时为 accepted/latest/non-superseded；
- Signal 的 Event、subject、VariableDefinition 和 Evidence 关系存在；
- 正式 Signal 节点的 Entity 与 Signal subject 一致；
- DirectImpact 的 source Signal subject 必须等于前一个 Reason Tree Node，target 必须等于
  当前下游 Node，affected variable、direction 与传导结构一致；
- `event_explicit` Evidence 或 `rule_inferred` Signal/Rule/Relation 血缘完整；
- Theme Event association 覆盖 Aggregate 使用的所有正式来源 Event；每棵 Tree 自己的
  Event association 还必须覆盖该 Tree 引用的所有正式 Signal/Impact 及上游正式事实的
  来源 Event，仅出现在 Theme Event 集合中不构成该 Tree 的有效血缘；
- 正式引用不晚于 `analysis_as_of`；
- analyst inference 没有冒充正式 Signal 或 DirectImpact。
- 校验与 Aggregate 写入处于同一事务边界；发布记录保留引用 UUID、
  `semantic_submission_id` 以及发布时通过 accepted/latest/non-superseded 门禁的事实。

Data 不判断 Codex 为什么选择这些 Signal、证据是否充分、多跳推理是否专业或结论是否
正确。

### 6.7 已确认：D-TA-012 单 Theme Aggregate 发布合同

每次发布请求只包含一个 Theme 及其全部 Reason Trees。一次 Codex 分析产生多个 Theme
时分别调用；每个 Aggregate 使用独立事务和幂等身份，一个失败不影响其他 Theme。

最小顶层请求：

```text
analysis_batch_id
analysis_as_of
discovery_window_start
discovery_window_end
theme
reasoning_trees[1..N]
```

`analysis_batch_id` 在本合同中只是外部调用方提供的稳定发布/幂等身份，不代表 Data
Service 拥有 Codex Analysis Run，也不保存模型执行状态。

请求规则：

- 顶层只能有一个 `theme`；
- `reasoning_trees` 至少一棵，全部属于该 Theme；
- Theme、Tree、Node、Event、Signal/Impact lineage 使用严格 DTO，拒绝未知字段；
- Theme 和现有 Reason Tree 展示字段继续复用已冻结的字段语义；
- Node Signal 增加正式 Signal 引用或 analyst-inference 来源类型；
- incoming transmission 可增加正式 DirectImpact 引用；
- Strong lineage 字段只进入发布/审计合同；Miniapp 显示 DTO 无需暴露不使用的内部引用，
  本轮不改 UI。

幂等语义：

- 首次成功返回 `201` 和不可变发布回执；
- 相同 publisher、相同 `analysis_batch_id`、相同 canonical payload 重试返回原回执，
  `200 + replayed=true`；
- 相同身份提交不同 canonical payload 返回 `409`；
- 新的分析结论必须使用新的 `analysis_batch_id`，从而创建新的 Theme Aggregate。

错误语义：

- `400`：JSON/DTO/枚举/顺序/时间格式或未知字段错误；
- `401/403`：现有统一 API Token 缺失或无权访问；
- `409`：幂等身份冲突或不可变唯一性冲突；
- `413`：超过明确的 Body 大小上限；
- `422`：正式 ID 不存在、状态/版本/范围不合法、血缘或 Tree 结构引用不一致；
- `500/503`：内部或依赖故障，不暴露 SQL、Token 或内部实现。

所有错误继续使用 Data 标准 ErrorEnvelope 和 Request ID。发布是同步、有界、单事务操作，
不新增异步 Job、轮询接口、队列或失败占位记录。未知 POST 结果只能以完全相同的幂等请求
重试。

#### 6.7.1 HTTP 形态

直接调整现有：

```text
POST /api/data/v1/research-theme-imports
```

使其接收一个 Theme Publication Aggregate，而不是 Theme 数组。独立
`POST /api/data/v1/research-reasoning-tree-imports` 写入口从新合同移除；Reason Tree 只
能随所属 Theme Aggregate 原子发布。现有 Theme/Reason Tree 读取资源继续保留，Miniapp
显示 DTO 不因内部强血缘字段而增加 UI 依赖。

## 7. 已冻结的跨系统流程

```text
1. Codex 按 `analysis_as_of` 请求 Data 查询 Analysis Context
2. Data 返回实时、稳定排序、可分页的正式 Event Semantic Bundles
3. Codex 生成 Investment Analysis 和 Theme/Reason Tree Candidate
4. Codex 本地执行研究语义门禁
5. Codex 使用稳定发布身份提交单个 Theme Aggregate Candidate
6. Data 的发布 Service 在一个事务边界内完成确定性校验与原子保存
7. Data 保存领域输入血缘、幂等身份和发布回执
8. Miniapp 只读取已发布结果
```

实时查询 DTO、发布合同和 provenance 边界均已冻结。

## 8. 决策日志

| ID | 决策 | 状态 | 来源 |
|---|---|---|---|
| C-TA-001 | Codex 不直连 PostgreSQL 或 Neo4j，只使用 Data Service 版本化能力 | 已确认 | 既有系统边界与用户确认 |
| C-TA-002 | Neo4j Graph Context API 不阻断首份 MVP，第一版可由 PG 事实组装 | 已确认 | 事件推理模型任务与用户确认 |
| C-TA-003 | Miniapp 本轮不改，只读取已发布 Theme/Reason Tree | 已确认 | 本轮范围 |
| C-TA-004 | 当前单 Event semantics GET 不作为 Codex 批量生产输入 | 工程事实 | OpenAPI 与实现核对 |
| C-TA-005 | `entity_role` 采用六项 MVP 受控词汇，并与 Signal subject、Direct Impact 分离 | 语义已确认 | 事件推理模型 |
| C-TA-006 | Analysis Context 按知识可用时间隔离未来信息；分页使用稳定排序和游标 | 语义已确认 | 事件推理模型 |
| C-TA-007 | Golden Scenario 验收分析师能力，不形成 Event 数量或 Theme readiness 系统门禁 | 已确认（覆盖原“两条 Event”基线） | 用户决议，事件推理模型复核 |
| D-TA-001 | MVP 实时查询，不建设 Snapshot 或查询任务 | 已确认 | 用户决议，事件推理模型复核 |
| D-TA-002 | Event discovery window 由 Codex 显式决定；Data 不设投研默认值 | 已确认 | 用户决议，事件推理模型复核 |
| D-TA-003 | 不设独立 prepare；发布 Service 内整体确定性校验，通过后原子保存 | 已确认（覆盖此前 prepare-only 表述） | 用户决议 |
| D-TA-004 | Theme + 1..N Reason Trees 是单个 Publication Aggregate，按 Aggregate 原子发布 | 语义已确认 | 用户确认 API 无兼容约束，事件推理模型复核 |
| D-TA-005 | 新分析创建新 Theme Aggregate；已发布 Theme/Reason Trees 不更新或版本化 | 已确认 | 用户决议 |
| D-TA-006 | Codex 只消费 Data 正式数据和逐字 Evidence；不返回正文或访问 AgentRun 原始数据 | 已确认 | 用户决议，事件推理模型复核 |
| D-TA-007 | Theme 分析师运行在工程外部 Codex；AgentRun 不新增、调度或监控 Theme Agent | 已确认 | 用户决议 |
| D-TA-008 | Data 不保存 Codex 执行、模型、Skill、Prompt、方法论或输出 Schema 版本 | 已确认 | 用户决议 |
| D-TA-009 | Data 提供实时批量查询；以完整 EventSemanticBundle 分页，不做投研聚合 | 已确认 | 用户决议，事件推理模型复核 |
| D-TA-010 | 工程外部 Codex 复用 Data Service 现有统一 API Token，不新增专用身份 | 已确认 | 用户决议 |
| D-TA-011 | Reason Tree 强引用正式 Signal/Impact UUID+Submission；UUID 承担事实版本身份，并保留显示快照 | 语义与工程已确认 | 事件推理模型；覆盖现有弱 Signal snapshot 合同 |
| D-TA-012 | 每次请求只发布一个 Theme + 其全部 Trees；多个 Theme 分别独立原子发布 | 已确认 | 用户决议 |
| D-TA-013 | 不建设状态水位/历史/Token；依赖分析期间中间语义数据不更新 | 已确认 | 用户决议 |

## 9. 实施与验收准备

架构级产品决策已关闭。其余为实施/验收准备：

1. 通过既有链路准备四类真实 Golden Event，记录最终 Event/Evidence ID；
2. 完成 4.6 的 Phase One 合同与 Worker 收尾；
3. 已将本需求中的新决策同步回
   `research-theme-reasoning-tree-spec.md`、Data Context 和正式 OpenAPI；
4. 完成后由外部 Codex 分析师执行单 Event、多 Event 汇聚、冲突/零 Theme 和强血缘
   Aggregate 验收。

## 10. 设计冻结门禁

以下条件已于 2026-07-29 满足：

- 所有架构级 Open 决策已关闭；
- `entity_role` 和前瞻 Signal/Evidence 语义已冻结；
- Analysis Context provider/consumer DTO、认证、超时、分页、幂等、错误和实时一致性语义已冻结；
- Aggregate publish 与 provenance 已冻结；
- Data、AgentRun、Codex、Neo4j、Miniapp Owner 无冲突；
- 用户明确确认“设计冻结”。

设计冻结不等于真实生产验收。4.6 与 Golden Data Preparation 未完成时，可以进入按阶段
实现，但不得生成或宣称首份真实 Theme/Reason Tree 已验收。
