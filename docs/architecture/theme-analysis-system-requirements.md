# Theme Analysis 系统侧需求与设计确认

## 状态

设计已冻结。用户于 2026-07-29 明确确认全部架构级需求决策已关闭并允许进入
`to-spec`；后续实现必须以本文及对应 GitHub Issue 为边界。设计冻结前未修改代码、
数据库或 API。

最近核对：2026-07-30。

2026-07-30 补充冻结：主库已证明全库背景字典无法继续作为每一页 Analysis Context
Payload。本文新增 D-TA-014—D-TA-017，冻结“Event 原始输入不裁剪、页级最小引用闭包、
Codex 主导的受控图谱检索、cursor/fingerprint 与结构化资源错误”边界。该补充不恢复
Snapshot，不改变 Theme 分析师位于工程外部 Codex 的既有决议。

2026-07-30 实施裁决：Analysis Context 尚未正式发布，因此不创建
`research-analysis-context.v2`。页级闭包、组件 fingerprint、cursor 和资源错误直接
修正现有 `research-analysis-context.v1`；provider 与工程外 `theme-analysis`
consumer 必须联动适配后才能关闭实施 Issue。

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

| 能力或事实                                                        | Owner                             | 边界                                              |
| ----------------------------------------------------------------- | --------------------------------- | ------------------------------------------------- |
| Event、Evidence、Entity、关系、TBox、accepted Event Semantics     | Data Service                      | PostgreSQL 正式事实与版本化 API                   |
| Event Semantic Work Item、Agent Execution、模型调用与自动审核执行 | AgentRun                          | 不拥有 Data 事实；不运行或监控 Theme 分析师       |
| 四层推理、反方复核、Investment Analysis、Theme/Tree Candidate     | 工程外部的 Codex `theme-analysis` | 不属于本工程部署单元；不直写数据库，不编造正式 ID |
| Theme、Theme Impact、Reason Tree、发布回执                        | Data Service                      | 确定性校验、持久化、发布可见性                    |
| Neo4j                                                             | Data Service 管理的可重建投影     | 本阶段不是首份 MVP 的硬依赖                       |
| Miniapp                                                           | Miniapp Backend / Frontend        | 只读取已发布结果，本轮 UI 不改                    |

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

| 问题                                                        | 分类                                         | 处理要求                                                                                     |
| ----------------------------------------------------------- | -------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Event Fact 与 Event Semantic 顺序共享一个 Shutdown deadline | 首份真实生产运行前门禁；不阻断本轮需求设计   | 两个 Worker 必须各自获得有界取消/等待机会，不能因前一 Worker 耗尽 deadline 而跳过后一 Worker |
| Data PostgreSQL Adapter 内存在审核决策和状态传播 Biz 逻辑   | 并行技术债；不阻断 Analysis Context 合同设计 | 不在新 Analysis Context Adapter 中继续扩散该模式；领域选择留在 Biz                           |
| Data CONTEXT 仍声明旧目录结构，与 Kratos 标准冲突           | 下一阶段实施前文档门禁；不阻断本轮 Grill     | 实施前统一权威文档，避免 Agent 收到冲突指令                                                  |

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

响应字典只返回当前页 EventSemanticBundles 的**最小引用闭包**，不返回全库背景字典。
闭包至少覆盖：

- EntityLink 的 Entity、Signal subject 和 DirectImpact target；
- 正式 Signal/Impact 引用的 VariableDefinition；
- 正式 DirectImpact 引用的 DirectTransmissionRule、EntityRelation 和对应
  RelationDefinition；
- 上述 EntityRelation 的全部端点 Entity；
- 闭包内 Entity 使用的 EntityTypeDefinition；
- 被页内正式对象显式引用的其他 TBox、产业链定义、Membership 或 Graph Edge。

闭包对象必须携带稳定 ID、领域存在时的显式版本和关系方向，并在本页内可确定性解析；
不允许出现无法解析的裸 ID，也不允许以预算为由静默删除页内正式引用。未被本页 Event
正式引用的全库 Entity、EntityRelation、IndustryChain、Membership、Graph Edge、TBox
或 AcceptancePolicy 不进入该页字典。Codex 如需从 seed Entity 扩展产业链背景，必须
使用 5.7 的受控图谱检索能力，不得把 Analysis Context 字典重新扩大为全库下载。

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

后续页使用 Data 返回的不透明 `cursor`。Cursor 绑定查询参数、稳定排序版本、查询合同
版本和 TBox contract version，不绑定全库或页级字典 Payload fingerprint；调用方不能
修改或拼接。响应返回有效查询参数、contract version、TBox contract version、
EventSemanticBundles、页级解析字典、页级/组件级 fingerprint、`next_cursor` 和
`has_more`。

- 合法空结果返回 `200` 和空 bundles；
- 参数、时间顺序、page size 或 cursor 错误返回 `400`；
- 统一 Token 认证失败返回 `401/403`；
- 请求的严格历史语义无法满足时返回明确 `422`，不得伪装成严格回测；
- 资源保护、暂时不可用和内部错误使用标准 `429/503/500`；
- Data 先选择本页 Event，再在 PostgreSQL 聚合嵌套 JSON 前对 Bundle 和本页最小引用
  闭包执行行数/原始字节 preflight；通过 preflight 后仍执行最终编码字节预算；
- 不静默截断窗口或 Bundle；超过技术预算时明确失败，由 Codex 缩小或分段查询；
- 同一 Bundle 过大时返回稳定错误，不把其子对象拆页；
- 所有错误使用标准 ErrorEnvelope 和 Request ID。资源错误的安全 details 至少包含
  超限组件、允许 rows/bytes 上限和可行的查询调整方向；只有完成计数或测量时才返回
  `actual_rows`/`actual_bytes`，bounded traversal 仅探测到 `budget+1` 时必须省略
  `actual_rows`，不得把下界伪装为实际总数。错误不返回 SQL、表名、查询计划或内部
  敏感信息。

读取接口没有写入幂等身份或 point-in-time 水位。不透明 cursor 只保证查询参数和稳定
排序连续性。服务端与客户端超时使用现有可配置基础设施边界，不在领域合同中写死秒数。

### 5.6 已确认：D-TA-014 页级最小引用闭包

#### 5.6.1 主库故障事实

当前实现并不符合 5.5 的可扩展目标：

1. PostgreSQL Adapter 在选择本页 Event 前执行 `preflightDictionaryBudget`；
2. preflight 与字典聚合读取所有满足 `analysis_as_of` 的 active Entity、EntityRelation、
   IndustryChain、Membership、Industry Chain Graph Edge 和必要 TBox；
3. 每一页重复返回该全库 Payload；
4. Biz cursor 保存并校验这个全库 `dictionary_fingerprint`；
5. 当前预算为字典 4 MiB/50,000 行、单页 8 MiB。

主库已经触发 `429 RESEARCH_ANALYSIS_CONTEXT_RESOURCE_LIMIT`，错误原因为 dictionaries
exceed preflight budget。该失败发生在 Event 选择之前，因此缩短 discovery window、
减少合格 Event 或改变 Event 页数都不能解决；它不是 migration 问题。

现有权威与实现冲突：

| 位置               | 当前事实                                                                                         | 与补充冻结要求的冲突                                         | 后续处理                                                               |
| ------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------ | ---------------------------------------------------------------------- |
| 本文 5.2/5.5       | 要求完整 Event Bundle、可解析字典且 Data 不做投研选择                                            | 原文未明确“相关”字典必须是页级闭包，容许被实现为全库 Payload | 由 D-TA-014 明确覆盖                                                   |
| Data Context       | 声明 cursor 绑定查询与字典 fingerprint，并对字典做全局预算                                       | 把全库 Payload fingerprint 变成分页连续性的前提              | 工作包 A 同步为 query/TBox contract 绑定和页级组件 fingerprint         |
| Data OpenAPI       | `dictionary_fingerprint` 是必填单值，旧 v1 未表达页级闭包和资源 details                         | consumer 无法区分全库字典与页级闭包，也无法获得安全超限信息  | 工作包 A 在未发布的 v1 上直接修正 DTO/Error details，不创建 v2         |
| Data Biz           | cursor 保存 `DictionaryFingerprint`，后续页要求相等                                              | 无关全库事实变化会使 Event cursor 失效                       | 移除 Payload fingerprint 的 cursor 门禁，保留 query/TBox contract 门禁 |
| PostgreSQL Adapter | Event 选择前扫描全库字典；每页再次聚合全库                                                       | 主库永久 429，窗口和 Event 分页无法解阻                      | Event-first 后查询本页引用闭包                                         |
| 既有测试           | 明确断言字典 fingerprint 变化使 cursor 失效                                                      | 固化了已证明不可扩展的旧语义                                 | 改为 query/contract mismatch、页级闭包完整性和主库规模测试             |

#### 5.6.2 短期解阻合同

`GET /api/data/v1/research-analysis-context` 保持同步、无状态、实时和完整 Event Bundle
分页语义，但内部读取顺序必须变为：

```text
校验实时查询参数与 cursor
→ 按 knowledge_available_at ASC, event_id ASC 选择本页完整 Event
→ 读取每个完整 EventSemanticBundle
→ 从本页正式对象收集稳定引用
→ 查询最小、闭合、可解析的页级 dictionaries
→ 执行 Bundle/闭包/整页预算与引用完整性校验
→ 返回本页及下一页 cursor
```

任何时间窗口内的合格 Event 都必须能够通过连续分页被读取；不能用减少 Event、
Evidence、EntityLink、Signal、Measurement 或 DirectImpact 作为容量修复。每个 Bundle
继续完整返回 latest/accepted/non-superseded 正式语义、逐字 Evidence excerpt、来源
元数据和正式血缘。

空 Event 页返回空 Bundle 和空引用闭包，不下载全库 TBox。页内闭包的确定性顺序按对象
类型的稳定业务键/显式版本/UUID 排序；同一正式引用在一页只出现一次。若单个完整
Bundle 或其必需闭包仍超限，Data 返回结构化 `429`，不得拆 Bundle、返回裸 ID 或悄悄
删除引用。

短期解阻只恢复直接一跳 Theme 流程所需的 Event 原始输入与正式引用解析；在 5.7 的图谱
检索能力交付前，不得宣称完整 Theme/Reason Tree 分析能力已经恢复。

#### 5.6.3 Cursor 与 fingerprint

MVP 继续遵守“实时查询、无 Snapshot、无 selection cutoff、发现不一致后从第一页重查”：

- `query_fingerprint` 只覆盖 normalized query、page size、contract version、稳定排序
  version 和 TBox contract version；
- cursor 只携带版本、`query_fingerprint` 和末项
  `(knowledge_available_at, event_id)`，不携带或比较全库/页级字典 Payload fingerprint；
- `event_page_fingerprint` 对本页完整 EventSemanticBundles 的 canonical Payload
  计算；
- `reference_closure_fingerprint` 对本页最小引用闭包的 canonical Payload 计算；
- 两个组件 fingerprint 用于输入血缘、诊断和 Codex 自检，不提供 Snapshot 或跨页状态
  水位保证；
- 显式对象版本仍是正式 TBox/规则事实身份，fingerprint 不替代版本；
- cursor 参数不匹配返回 `400`；查询期间发现引用漂移或无法闭合时返回结构化冲突并要求
  Codex 从第一页重查；已知历史状态不可恢复仍使用既有 `422`。

### 5.7 已确认：D-TA-015 Codex 主导的受控图谱检索

页级引用闭包只负责解释 Event 原始事实，不负责给 Codex 选择研究空间。完整 Theme /
Reason Tree 能力需要 Data Service 提供一个独立的、无状态、受预算约束的研究图谱检索
合同。推荐使用结构化 POST 形式的幂等只读搜索，例如：

```text
POST /api/data/v1/research-graph:search
```

最终路径命名属于 OpenAPI 实施细节，但合同边界冻结为：

**Codex 显式决定**

- 一个或多个 `seed_entity_ids`；
- 允许的 `relation_types`；
- 每类关系的 `direction=outgoing|incoming|both`；
- `max_depth`；
- 可选 `industry_chain_entity_id` scope；
- `node_budget` 与 `edge_budget`；
- 与 Analysis Context 一致的 `analysis_as_of`。

**Data Service 只负责**

- 校验 Entity、RelationDefinition、可选 IndustryChain 和查询参数；
- 仅检索当时可用的 approved/active 正式 Entity、EntityRelation、Membership 和 Graph
  Edge；
- 按 depth、seed、relation type、端点和 Edge ID 的稳定规则执行 bounded traversal；
- 返回完整端点 Entity、Relation/Graph Edge、Membership、IndustryChain 和必要显式版本，
  保证没有裸 ID；
- 执行权限、深度、node/edge/响应字节预算及引用完整性校验；
- 返回 query/component fingerprint、实际深度和结构化资源错误。

Data Service 不选择 seed、不补默认主产业链、不选择“最佳”路径、不计算 Theme
readiness、不判断投资方向或关系重要性，也不调用 LLM。第一版可由 PostgreSQL bounded
recursive CTE 实现；未来 Neo4j 投影可以替换 Data Adapter，但不得改变 API 语义，Codex
始终不直连 PostgreSQL 或 Neo4j。

预算命中不能返回未声明的部分子图。第一版可在完整结果超过请求或服务硬预算时返回安全
`429`，由 Codex 缩小 relation type、direction、depth 或 chain scope 后重查；未来如增加
子图分页，必须提供显式 `has_more`、不透明 cursor 和完整 frontier 语义，不能把截断结果
伪装成完整图。

### 5.8 已确认：D-TA-016 资源错误与分析召回验收

资源限制继续保留，不通过简单提高 4 MiB/50,000 行上限掩盖全库查询问题。
`RESEARCH_ANALYSIS_CONTEXT_RESOURCE_LIMIT` 与未来 Graph Search 资源错误至少提供：

```json
{
  "component": "reference_closure",
  "actual_rows": 1234,
  "max_rows": 1000,
  "actual_bytes": 1048576,
  "max_bytes": 524288,
  "retry_guidance": "reduce_page_size"
}
```

字段允许按标准 ErrorEnvelope 的 details 表达；未知值可省略，但服务端已完成计数或测量的
行数、字节数不得只写入日志而不返回。为保护查询而在 `budget+1` 停止的 bounded
traversal 只证明超限，不知道实际可达总数，因此必须省略 `actual_rows`，保留
`max_rows`。`component` 使用稳定枚举，如
`event_semantic_bundle`、`reference_closure`、`analysis_context_page`、
`research_graph_nodes`、`research_graph_edges`、`research_graph_result`。节点和边
预算分别报告各自实际值与上限，不能把两个维度相加后形成不可解释的限制。
`retry_guidance` 只描述技术查询调整，不给出投研选择。

不降低 Theme 原始输入与分析召回率的验收标准：

1. 对固定窗口的控制查询与连续分页结果比较，Event ID 集合、数量和稳定顺序完全一致；
2. 每个 Event 的 accepted/latest/non-superseded EntityLink、Signal、Measurement、
   DirectImpact 和 Evidence 集合与修复前无容量限制的语义基线完全一致；
3. Evidence excerpt 逐字一致，来源、时间和对象血缘不丢失；
4. 页内每个 Entity、Variable、Rule、Relation、Graph Edge 引用均能在闭包中解析，且闭包
   不含与本页无引用关系的全库对象；
5. 主库即使全库字典超过旧预算，只要单个完整 Bundle/页级闭包在预算内，page_size=1
   必须能返回 `200` 并最终遍历全部合格 Event；
6. 缩短窗口只改变 Event 选择，不再决定全库字典 preflight 是否成功；
7. 单 Bundle、页级闭包或整页真实超限时返回包含组件和上限的安全 `429`；完成测量时
   同时返回实际值，提前停止的有界遍历不得伪造实际总数；响应不得截断或泄露 SQL；
8. cursor 不再因无关的全库 Entity/Relationship 变化失效；参数或合同不匹配仍被拒绝，
   检测到页内引用不一致时 Codex 从第一页重查；
9. 对固定 seed/filter/direction/depth/scope，分段图谱查询结果并集与 PostgreSQL 控制
   traversal 的可达节点/边集合一致；改变预算不得静默改变“完整结果”含义；
10. Codex consumer 必须读取全部 Event 页，并把页级闭包与按需 Graph Search 结果按稳定
    ID/显式版本合并；不得把第一页或预算内子集当成全部输入。

### 5.9 已确认：D-TA-017 工作包、风险与迁移边界

推荐拆为两个可独立验收、按顺序交付的系统工作包：

**工作包 A：主库短期解阻**

- 同步本文、Data Context、正式 OpenAPI 和 Codex consumer contract version；
- 调整 Data Biz Store Port、cursor、page/component fingerprint 和结构化资源错误；
- PostgreSQL 改为 Event-first 选择和页级引用闭包查询；
- 更新 API DTO、OpenAPI、HTTP/Biz/Data/合同测试；
- Codex consumer 适配页级闭包、全页遍历和从第一页重查；
- 用会触发旧全库 50,000 行门禁的数据规模执行 PostgreSQL 集成验收。

**工作包 B：完整按需图谱穿透**

- 冻结 Graph Search OpenAPI、DTO、认证、预算和错误；
- Data Biz 建立存储无关 traversal Port；
- PostgreSQL bounded recursive CTE Adapter 与确定性完整性校验；
- Codex consumer 显式编排 seed/relation/direction/depth/scope 查询；
- 覆盖正常、反向、双向、chain scope、环、预算超限、无结果和未来 Neo4j 语义一致性
  fixture。

主要风险及门禁：

- 页内 ID 集合较大导致 SQL 参数/查询计划退化：使用 typed array/CTE 和实际主库 explain
  验证，不逐 ID 循环查询；
- 闭包规则遗漏新引用类型：以通用引用完整性测试和 OpenAPI required fields 为门禁；
- 实时跨页变化：维持 D-TA-013，检测到不一致即整次重查，不伪造 Snapshot；
- Graph explosion 或环：bounded traversal、去重、硬深度与 node/edge/byte 预算；
- PostgreSQL/Neo4j 结果漂移：未来 Adapter 必须通过同一 provider/consumer fixture；
- 页级字典重复增加带宽：允许 Codex 按 ID+版本缓存，但不得改成全库预加载。

当前表已经拥有 Event、Entity、EntityRelation、IndustryChain、Membership、Graph Edge
与版本化 TBox，第一阶段未发现 PostgreSQL schema migration 的必要性。若实现发现必须
新增持久化结构，必须单独说明无法由现有事实/查询表达的责任，并重新请求架构裁决；不得
把 Snapshot、查询任务或图遍历缓存表夹带进本修复。

### 5.10 已确认：D-TA-010 Codex 认证

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

| ID       | 决策                                                                                                     | 状态                                 | 来源                                          |
| -------- | -------------------------------------------------------------------------------------------------------- | ------------------------------------ | --------------------------------------------- |
| C-TA-001 | Codex 不直连 PostgreSQL 或 Neo4j，只使用 Data Service 版本化能力                                         | 已确认                               | 既有系统边界与用户确认                        |
| C-TA-002 | Neo4j Graph Context API 不阻断首份 MVP，第一版可由 PG 事实组装                                           | 已确认                               | 事件推理模型任务与用户确认                    |
| C-TA-003 | Miniapp 本轮不改，只读取已发布 Theme/Reason Tree                                                         | 已确认                               | 本轮范围                                      |
| C-TA-004 | 当前单 Event semantics GET 不作为 Codex 批量生产输入                                                     | 工程事实                             | OpenAPI 与实现核对                            |
| C-TA-005 | `entity_role` 采用六项 MVP 受控词汇，并与 Signal subject、Direct Impact 分离                             | 语义已确认                           | 事件推理模型                                  |
| C-TA-006 | Analysis Context 按知识可用时间隔离未来信息；分页使用稳定排序和游标                                      | 语义已确认                           | 事件推理模型                                  |
| C-TA-007 | Golden Scenario 验收分析师能力，不形成 Event 数量或 Theme readiness 系统门禁                             | 已确认（覆盖原“两条 Event”基线）     | 用户决议，事件推理模型复核                    |
| D-TA-001 | MVP 实时查询，不建设 Snapshot 或查询任务                                                                 | 已确认                               | 用户决议，事件推理模型复核                    |
| D-TA-002 | Event discovery window 由 Codex 显式决定；Data 不设投研默认值                                            | 已确认                               | 用户决议，事件推理模型复核                    |
| D-TA-003 | 不设独立 prepare；发布 Service 内整体确定性校验，通过后原子保存                                          | 已确认（覆盖此前 prepare-only 表述） | 用户决议                                      |
| D-TA-004 | Theme + 1..N Reason Trees 是单个 Publication Aggregate，按 Aggregate 原子发布                            | 语义已确认                           | 用户确认 API 无兼容约束，事件推理模型复核     |
| D-TA-005 | 新分析创建新 Theme Aggregate；已发布 Theme/Reason Trees 不更新或版本化                                   | 已确认                               | 用户决议                                      |
| D-TA-006 | Codex 只消费 Data 正式数据和逐字 Evidence；不返回正文或访问 AgentRun 原始数据                            | 已确认                               | 用户决议，事件推理模型复核                    |
| D-TA-007 | Theme 分析师运行在工程外部 Codex；AgentRun 不新增、调度或监控 Theme Agent                                | 已确认                               | 用户决议                                      |
| D-TA-008 | Data 不保存 Codex 执行、模型、Skill、Prompt、方法论或输出 Schema 版本                                    | 已确认                               | 用户决议                                      |
| D-TA-009 | Data 提供实时批量查询；以完整 EventSemanticBundle 分页，不做投研聚合                                     | 已确认                               | 用户决议，事件推理模型复核                    |
| D-TA-010 | 工程外部 Codex 复用 Data Service 现有统一 API Token，不新增专用身份                                      | 已确认                               | 用户决议                                      |
| D-TA-011 | Reason Tree 强引用正式 Signal/Impact UUID+Submission；UUID 承担事实版本身份，并保留显示快照              | 语义与工程已确认                     | 事件推理模型；覆盖现有弱 Signal snapshot 合同 |
| D-TA-012 | 每次请求只发布一个 Theme + 其全部 Trees；多个 Theme 分别独立原子发布                                     | 已确认                               | 用户决议                                      |
| D-TA-013 | 不建设状态水位/历史/Token；依赖分析期间中间语义数据不更新                                                | 已确认                               | 用户决议                                      |
| D-TA-014 | Analysis Context 先选择本页完整 Event，再返回页级最小引用闭包；不减少 Event 原始输入                     | 已确认                               | 2026-07-30 主库容量故障与冻结修复边界         |
| D-TA-015 | Codex 通过 Data 受控 Graph Search 显式选择 seed/relation/direction/depth/scope/budget；Data 不做投研选择 | 已确认                               | 2026-07-30 冻结修复边界                       |
| D-TA-016 | Cursor 不绑定全库字典 Payload；返回页级组件 fingerprint；资源错误仅在完成测量时返回实际值                  | 已确认                               | 2026-07-30 冻结修复边界及实施裁决             |
| D-TA-017 | 分“主库短期解阻”和“完整图谱穿透”两个工作包；第一阶段默认无 PostgreSQL migration                          | 已确认                               | 2026-07-30 冻结修复边界                       |

## 9. 实施与验收准备

架构级产品决策已关闭。其余为实施/验收准备：

1. 通过既有链路准备四类真实 Golden Event，记录最终 Event/Evidence ID；
2. 完成 4.6 的 Phase One 合同与 Worker 收尾；
3. 已将本需求中的新决策同步回
   `research-theme-reasoning-tree-spec.md`、Data Context 和正式 OpenAPI；
4. 完成后由外部 Codex 分析师执行单 Event、多 Event 汇聚、冲突/零 Theme 和强血缘
   Aggregate 验收。
5. 先交付 D-TA-014/016 的主库短期解阻，再交付 D-TA-015 的受控图谱检索；前者完成后
   只能宣称直接一跳输入恢复，两个工作包均完成后才可宣称完整 Theme/Reason Tree
   Analysis Context 能力恢复。

第 3 项描述的是 2026-07-29 原冻结设计的同步状态。D-TA-014—D-TA-017 尚未同步到 Data
Context、OpenAPI 或实现；这些属于新工作包的明确交付项，不能因本文已记录而视为已经上线。

## 10. 设计冻结门禁

以下条件已于 2026-07-29 满足：

- 所有架构级 Open 决策已关闭；
- `entity_role` 和前瞻 Signal/Evidence 语义已冻结；
- Analysis Context provider/consumer DTO、认证、超时、分页、幂等、错误和实时一致性语义已冻结；
- Aggregate publish 与 provenance 已冻结；
- Data、AgentRun、Codex、Neo4j、Miniapp Owner 无冲突；
- 用户明确确认“设计冻结”。
- D-TA-014—D-TA-017 不引入新的产品语义选择；接口最终路径命名、硬预算数值和 SQL
  组织属于实现期工程决策，遵守本文边界即可，无需逐项产品确认。

设计冻结不等于真实生产验收。4.6 与 Golden Data Preparation 未完成时，可以进入按阶段
实现，但不得生成或宣称首份真实 Theme/Reason Tree 已验收。D-TA-015 未完成时，也不得
把仅完成页级字典裁剪表述为完整 Theme/Reason Tree 分析能力已恢复。
