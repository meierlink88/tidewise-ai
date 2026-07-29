# 阶段一：变量本体与 Event 语义落地

## 状态

- 文档状态：Frozen
- 设计状态：已冻结并授权实施
- 当前阶段：Issue #138 实施
- 最后更新：2026-07-29
- 所属 Context：Data、AgentRun、Admin Portal
- 不涉及 Context：Miniapp
- 前置事实：Research Theme 与 Reason Tree 的结构性调整已经完成

用户已通过 `$to-spec` 明确将已确认设计转为实施规格，并授权交接实施。GitHub Issue
[#138](https://github.com/meierlink88/tidewise-ai/issues/138) 是实现、验证和 PR
追踪锚点。最终生产验收仍受 12.1.3 的 Golden Data Preparation 门禁约束。

## 1. 阶段结果

本阶段打通：

```text
真实 Event
→ 规范 Entity
→ Variable Signal
→ 直接影响 Entity / Chain Node
→ Data Service 确定性校验与保存
→ Codex 分析师通过 Data Service 重新查询完整语义结果
```

本阶段为第一份真实 Research Theme / Reason Tree 提供合格输入，但不生成、更新或发布
Research Theme / Reason Tree。

每条通过验收的真实 Event 至少形成：

1. 一个已解析的规范 Entity；
2. 一个引用受控 Variable Definition 的合法 Variable Signal；
3. 一个合法 Direct Target 及其直接影响断言；
4. 可回溯到正式 Event Evidence 的证据；
5. 可追踪到 AgentRun 执行与 Data 发布结果的运行血缘；
6. 可由消费者通过 Data Service 重新读取的完整结果。

## 2. 明确不做

- Event Episode 及 Episode 状态机；
- 时间衰减；
- 产业链多跳传导；
- 完整 Transmission Rule 引擎；阶段一只保留 3–5 条 Golden Scenario 所需的人工
  `DirectTransmissionRule`；
- Inference Step；
- Investment Conclusion；
- Research Theme / Reason Tree 自动生成；
- Research Theme 版本跟踪；
- Miniapp 改动；
- Event、Variable Definition、Variable Signal 或 Event Episode 的全量 Neo4j 投影；
- Agent 直连 PostgreSQL 或 Neo4j；
- 模型创建 Entity ID、Chain Node ID 或未定义 Variable Definition；
- 重建现有 Entity 实例。

## 3. 权威边界与 Owner Map

| 责任 | Owner | 本阶段边界 |
| --- | --- | --- |
| Event、Evidence、Entity、产业链主数据 | Data Service | 正式领域事实与 PostgreSQL 唯一 Owner |
| Entity Type / Variable Definition | Data Service | TBox、版本、状态和适用范围 |
| Variable Signal | Data Service | Event-native ABox、确定性校验、审核结果和持久化 |
| Direct Transmission Rule | Data Service | 版本化 TBox 因果映射、审核状态和 PostgreSQL 持久化 |
| Direct Impact Assertion | Data Service | 一跳分析候选、确定性校验、审核结果和持久化 |
| Agent Definition / Version / Execution | AgentRun | Agent 平台身份与运行态 |
| Event Semantic Enricher 候选生成 | AgentRun | 生成严格结构化候选，不拥有 Data 事实 |
| Provider API / OpenAPI | Data Service | 版本化 HTTP 合同 |
| Consumer Port / typed client | AgentRun | 不 import Data Service 的 Go DTO 或实现 |
| Agent Status 监控事实 | AgentRun | Agent Registry、是否工作中、当前 Execution 状态与更新时间 |
| Agent Status Monitor | Admin Portal | 只读展示 AgentRun 当前 Agent 状态，不拥有运行状态或审核裁决 |
| PostgreSQL | Data Service 与 AgentRun 各自拥有 | 不共享表、凭据、repository 或事务 |
| Neo4j | Data Service | 本阶段只读复用已有 Entity / 产业链关系能力 |
| Miniapp | Miniapp | 本阶段无改动 |

跨边界只允许版本化 HTTP API 与冻结 JSON fixture。AgentRun 不读取 Data PostgreSQL，
Data Service 不读取 AgentRun 数据库或 Artifact 文件系统。

## 4. 当前事实与已发现的合同差异

### 4.1 现有 Entity 类型不是候选清单的同义集合

Data 代码当前支持：

```text
alliance_org, economy, policy_body, market, index, benchmark, sector,
industry, concept, industry_chain, chain_node, theme, company, security,
instrument, metric, commodity, person
```

阶段候选清单中的 `Product`、`Technology`、`Policy`、`Region` 当前不是正式
`entity_nodes.entity_type`；`Policy` 也不能未经决定就等同于现有 `policy_body`，
后者表示政策机构而不是政策文件或政策措施。`Region` 也不能未经决定就等同于
`economy`。

因此，“第一批 Entity Type”必须从当前正式类型中选择，或者另行明确批准主数据模型
扩展；不得只为适配 Prompt 而发明类型映射。

### 4.2 `event_entity_links` 已存在

当前表和 Biz 模型已经包含：

- `event_id`；
- `entity_id`；
- `entity_role`；
- `assign_source`；
- `review_status`；
- `evidence_note`；
- 唯一性：`event_id + entity_id + entity_role`。

当前不包含：

- `confidence`；
- `resolution_method`；
- `semantic_submission_id`；
- 对正式 Event Evidence 的结构化引用。

因此本阶段应先决定兼容演进、替换或建立独立语义发布聚合，不能按“新建
EventEntityLink”处理。

D-009 只读发现：

- 当前本地表为 0 行；
- 除初始化 migration、Entity Seed preflight/orphan 检查和未被调用的 Biz struct 外，
  当前没有 Repository、HTTP API 或 AgentRun consumer；
- 现有 `UNIQUE(event_id, entity_id, entity_role)` 无法保留同一 Event 的多次分析、
  rejected/pending/superseded 候选；
- `evidence_note` 是自由文本，不能替代正式 Event Evidence 引用；
- 现有 `pending | approved | rejected` 与 D-006 的自动审核状态词不一致。

D-009 已确认采用以下兼容演进：

- 原表原位演进，不建立第二套 canonical `event_entity_links`，也不做双写或旧 API；
- 增加 EventSemanticSubmission、候选稳定 key、resolved mention、confidence、
  resolution_method 和结构化 Evidence 关联；
- 状态迁移为 `pending_review | needs_reanalysis | quarantined | accepted | rejected |
  superseded`；
- 移除全局 `(event_id, entity_id, entity_role)` 唯一约束，改为 Submission 内候选唯一；
- 同一 Event/Entity/Role 同时最多一个 active accepted Link；新 accepted Link 必须在同一
  事务 supersede 旧 accepted Link；
- 新语义行必须引用 Submission 和 Evidence；迁移前已有 legacy 行若其他环境非空，不删除，
  允许保留可空 Submission/Evidence 并标记 legacy provenance，后续不冒充新流水线产物；
- `evidence_note` 仅保留为可选说明，不作为审核依据。

### 4.3 当前 `VariableSignalKey` 不是 Variable Signal 事实

Reason Tree 当前保存的是节点内的 `variable_signal_key`、方向、角色和显示摘要快照。
它没有独立 Variable Definition、适用实体类型、数值、有效期、审核状态或 Event
血缘。阶段一的新对象不能把该展示快照误当成已经存在的 Variable Signal ABox。

### 4.4 当前 Provider API 不具备完整阶段一能力

现有 Data V1 OpenAPI 已包含 Event 发布、Event Tag Catalog、Theme/Reason Tree
读写等合同，但尚未提供本阶段候选的 Entity Resolution、Ontology Context、
Variable Definition、Event Semantics 发布与语义结果读取合同。

早期首报 Spec 只冻结了 “Event → Direct Node Mapping V1”，而本阶段候选 Direct
Target 可能扩展到 Company、Industry、Concept、Sector、Security 等 Entity。若确认
扩展，这将是对早期最窄范围的显式替代，必须记录理由和兼容边界。

## 5. 候选领域模型

本节只记录待 Grill 的候选形状，不代表表名、字段名或持久化方式已批准。

### 5.1 Entity Type Definition

用途：定义哪些现有 Entity Type 可参与阶段一推理，不重建 Entity 实例。

事件推理模型建议采用“最小推理实体集 + 上下文实体集”，不把当前全部 Entity Type
开放给 Variable Signal。

语义上建议的最小推理实体集：

| Entity Type | Signal Subject | Direct Target | 约束 |
| --- | --- | --- | --- |
| `commodity` | 是 | 是 | 商品供需、价格、库存 |
| `product` | 是 | 是 | 阶段一新增类型；产品需求、价格、订单、产量 |
| `chain_node` | 是 | 是 | 产业链变量的核心承载对象 |
| `industry_chain` | 有条件 | 有条件 | 只在 Event 明确作用于整条链时使用；能定位节点时优先节点 |
| `industry` | 是 | 是 | 行业需求、景气与政策影响 |
| `company` | 是 | 是 | 收入、订单、产能、利润等经营变量 |
| `security` | 是 | 是 | 只限证券本身的 Event 或市场观测；公司经营事实不能直接写成证券事实 |
| `sector` | 是 | 是 | 板块资金、估值和景气预期 |
| `concept` | 是 | 是 | 概念热度、政策预期和资金关注 |

只作为 Event actor、陈述主体、政策发布者或 Evidence 归因上下文，不作为阶段一 Signal
Subject 或 Direct Target：

```text
policy_body, person, alliance_org
```

延后开放：

| Entity Type | 阶段一边界 |
| --- | --- |
| `economy` | 暂作 Event 上下文；宏观变量模型进入后再开放 |
| `market`、`index` | 延后到 Observation / 市场数据模型 |
| `benchmark` | 只作比较口径 |
| `instrument` | 股票 MVP 先使用 `security` |
| `metric` | 不作 Signal Subject；变量语义由 Variable Definition / Observation 承担 |
| `theme` | 不进入底层推理，避免与产品 Research Theme 循环混淆 |

初始候选新类型的语义结论：

- `product`：建议阶段一新增一等类型。Product 是企业生产、销售或被需求的对象，
  不等于生产环节 `chain_node`，也不等于标准化可交易的 `commodity`；
- `technology`：延后且不永久映射为 Concept；阶段一保留为 Mention/context，并解析
  到直接相关 Product 或 Chain Node；
- `policy`：延后且禁止映射为 `policy_body`；Event 表达政策发布事实，政策机构是陈述
  主体，受影响 Industry/Chain Node/Company 承载 Signal；
- `region`：延后且禁止映射为 `economy`；阶段一作为 Event location 或 Signal 地域
  适用范围。

用户已确认 `product` 进入阶段一 Data 主数据模型。冻结边界是不自动重分类或批量迁移
现有 Chain Node / Commodity，只为 Golden Scenario 通过受控主数据流程建立必要
Product 及其正式关系。Product Profile、正式关系类型、解析与 API 字段仍需在后续
工程问题中逐项确认。

其余待确认：

- 它是否需要成为版本化持久化对象，还是由现有 Entity Type 枚举加 Ontology
  Context 投影表达；
- active / deprecated 等状态是否独立于现有 Entity Type。

边界样例：

- “企业获得 800G 光模块新增订单”：Product `800G 光模块` 是 Signal Subject，
  Direct Target 可以是 Company 与“光模块制造”Chain Node；
- “企业毛利率同比上升”：Company 是 Signal Subject/Direct Target，不能直接写成
  Security 利好；
- “政策机构发布新能源汽车补贴措施”：`policy_body` 是陈述主体，Industry 或
  Chain Node 承载政策支持强度 Signal；
- “智利某铜矿停产”：Company 是 Actor，智利是地域范围，Commodity 铜或铜矿开采
  Chain Node 承载供给下降 Signal。

### 5.2 Variable Definition

Variable Definition 是变量 TBox 定义。候选字段：

- `key`；
- `name_zh`、`name_en`；
- `domain`；
- `business_definition`；
- `value_type`；
- `canonical_unit` 或允许单位集合；
- `allowed_directions`；
- `status`；
- `version`。

`VariableDefinitionApplicableEntityType` 候选表达一个 Variable Definition 可用于哪些
Entity Type。它是否独立成表、是否需要生效时间，以及版本升级时旧 Signal 如何解释，
仍未确认。

### 5.3 Variable Signal

Variable Signal 是 Event 支撑的变量语义 ABox 候选。初始字段：

- 来源：`source_type`、`source_event_id`；
- 主体：`subject_entity_id`；
- 变量：`variable_definition_id`；
- 方向：`direction`；
- 断言模态：`assertion_modality`；
- 量化：零个或多个受控 `MeasurementValue`；
- 时间：`valid_from`、`valid_until`；
- 证据：Evidence 引用与必要的证据说明；
- 质量：`confidence`、`review_status`；
- 血缘：Data-owned publication/run reference 与 AgentRun execution reference。

阶段一允许 Variable Signal 保留 Event 原有的三类断言模态：

| `assertion_modality` | 语义 |
| --- | --- |
| `actual` | 已经发生、已经公布的实际结果或当前可观测状态 |
| `stated_intent` | 明确宣布但尚未兑现的计划、目标、政策意图或行动承诺 |
| `source_forecast` | 企业、政府、研究机构等明确发布的预测、指引或预期 |

使用 `source_forecast` 而不是 `forecast`，用于明确预测来自 Event 引用的外部陈述主体，
不是 Agent 自己的预测。三类 Signal 都必须回溯到正式 Event 中的明确陈述主体、Evidence
和适用时间；模型只提取原有模态，不能把自己的未来推测、下游影响推导、媒体评论或匿名
市场观点升级为 Event 原生 Variable Signal。

阶段一不增加前瞻 Signal 的 `realized/cancelled` 生命周期；兑现与失效跟踪属于后续
阶段。哪些 `source_forecast` 来源可自动接受，留待 D-006 的审核 policy 一并确认。

边界样例：

- 海关公布 6 月铜精矿进口量同比下降 12%：
  `import_volume + decrease + actual`；
- 企业公告计划 2027 年前新增 30 万吨产能：
  `capacity + increase + stated_intent`，不能写成当前产能已经增加；
- 公司指引下一季度收入同比增长 15%–20%：
  `revenue + increase + source_forecast`，不能写成实际收入增长；
- 新闻评论“市场可能担忧供应，原油价格将上涨”：
  若没有 Event 中可识别主体的明确预测，不得形成原油价格 Event 原生 Signal。

### 5.4 Event Entity Link

复用现有结构的候选方向：

- 保留现有 `entity_role`、`assign_source`、`review_status` 和唯一键；
- 评估增加 `confidence`、`resolution_method`、Evidence 引用和发布血缘；
- 禁止未知 Mention 自动创建 Entity；
- 同一 Mention 多候选、同一 Entity 多角色、merged/inactive Entity、解析歧义的处理待确认。

### 5.5 Direct Impact Assertion

规范术语暂定 **Direct Impact Assertion**。Agent 生成阶段的对象称
`DirectImpactCandidate`；Data 接受后的对象称 `Accepted DirectImpactAssertion`。
它们都不是 Event 原生事实：

- Variable Signal 是 Event Evidence 直接支持的 Event-native ABox；
- Direct Impact Assertion 是一个 Signal 向不同 Entity 传播的一跳分析断言；
- Accepted 表示该断言已通过引用、关系、变量、时间、规则和审核校验，不把它升级为
  客观事实。

最小字段候选：

- `source_variable_signal_id`；
- `target_entity_id`；
- `affected_variable_definition_id`；
- `affected_direction`；
- `derivation_type = event_explicit | rule_inferred`；
- `mechanism_summary`；
- Event Evidence 引用；
- Entity Relation 引用；
- 可空的一跳 Transmission Rule 引用；
- `confidence`、`assertion_status`；
- `effective_from`、`effective_to`；
- Data publication / analysis 与 Agent Execution 血缘；
- 审核时间和原因。

必须区分：

- `signal_direction`：Subject 上该变量如何变化；
- `affected_direction`：该 Signal 直接影响 Target 上的哪个受控变量、该变量如何变化。

例如，“铜价上涨”中的 Signal 可以是 Commodity 铜的 `price increase`，但对下游电缆
制造 Chain Node 的直接影响必须表达为 `input_cost increase`，不能只写含义不明的
`negative`。业务有利/不利、投资利好/利空属于后续 Investment Conclusion。

`affected_variable_definition_id` 必填；若目录中没有目标变量，就不能用
`mechanism_summary` 或 `positive/negative` 绕过 TBox。`impact_strength` 不进入最小
V1；只有 Event 或规则有量化依据时才可在后续增加。

推导依据：

- `event_explicit`：Event Evidence 必须直接包含 A 对 B 的因果陈述，并能定位 A、B、
  受影响变量和方向；
- `rule_inferred`：必须同时引用 Source Signal 的 Event Evidence、A/B 的直接
  Entity Relation 和已批准的一跳 Transmission Rule；Agent 常识和机制文字不能替代
  结构化依据。

Data 保存已提交的 Candidate、校验和审核结果、拒绝原因及接受后的断言版本。
Candidate/Pending/Rejected 只对审核、审计和对应运行可见；只有 Accepted 可以进入
后续推理。AgentRun 在提交前拥有运行产物，提交后 Data 拥有领域候选及其审核状态；
修正产生新候选或版本，不覆盖原记录。

D-004 已确认以下目标与一跳语义；此处只确定结构合法性，不提前决定 D-005 的事实和
审核状态：

| 分类 | Entity Type | 规则 |
| --- | --- | --- |
| Allow | `commodity`, `product`, `chain_node`, `company` | 只要存在明确的跨实体一跳关系和单一机制即可成为 Target |
| Conditional | `industry` | 只有明确的一跳行业级投入、产出或需求关系时允许；不能作为找不到 Product/ChainNode 的兜底 |
| Deny / Deferred | `industry_chain`, `security`, `sector`, `concept` | IndustryChain 应作为 Signal Subject 或图谱组织结构；Security 当前无合法证券专属跨实体变量入口；Sector/Concept 属于后续市场或主题聚合 |
| Context-only | `policy_body`, `person`, `alliance_org` | 只能作为 Event actor、声明主体或上下文 |

`Direct` 必须同时满足：

1. 已存在合法 Variable Signal，主体为 A；
2. A 与 Target B 存在明确的直接业务关系，或 Event 动作明确指向 B；
3. A 到 B 只需要一个传导机制，不依赖第三个中间实体；
4. 不依赖投资者情绪、估值或证券价格反应；
5. 时间窗口匹配；
6. 机制可由 Event Evidence、现有 Entity Relation 或已批准的一跳规则支持。

结构规则：

- `target_entity_id == variable_signal.subject_entity_id` 一律禁止；
- Signal 本身已经表达变量落点，没有合法跨实体 Target 时 `DirectImpact[]` 合法为空；
- 同一 Signal 可以有多个 Target，但每个 Target 必须有独立一跳关系和机制；
- 同一 Target 可以有多个不同机制或不同受影响变量；仅 Evidence 不同的同一机制应合并
  Evidence，不重复创建 Impact。

政策支持对象直接作为 `policy_support_intensity` Signal Subject，而不是该 Signal 的
Target。例如政策支持 800G 光模块时，Product 800G 光模块是 Signal Subject，
`DirectImpact[]` 可以为空；PolicyBody 只是 actor，不能为了构造路径成为 Signal
Subject。普通经营、财务或政策 Signal 不得直接跳到 Security；证券投资表现属于后续
盈利预期、估值和市场定价推理。

#### 5.5.1 最小 Direct Transmission Rule 注册表

阶段一包含一个只服务 Golden Scenario 的最小一跳 `DirectTransmissionRule` 注册表。
它是可版本化、可审核的 TBox 因果映射表，不是完整规则引擎：

```text
source_entity_type
+ source_variable_definition
+ source_direction
+ relation_type
+ target_entity_type
= affected_variable_definition
+ affected_direction
```

规则字段至少包含：

- 稳定 `rule_key` 和正整数 `version`；
- `source_entity_type`；
- `source_variable_definition_id` 或等价的受控 key/version 引用；
- `source_direction`；
- `relation_type`；
- `target_entity_type`；
- `affected_variable_definition_id` 或等价的受控 key/version 引用；
- `affected_direction`；
- `condition_summary`；
- `mechanism_template`；
- `status = draft | approved | deprecated`；
- 创建、审核和版本血缘。

规则不得包含具体 Event ID、Company ID、Product ID 或其他实例 ID。“只服务 Golden
Scenario”表示首批只建立样例实际需要的少量通用规则，不把样例写死。

Owner 与运行边界：

- Data Service 是 Rule 唯一 Owner，规则版本和审核状态持久化在 Data PostgreSQL；
- AgentRun 只通过版本化 Data API 读取 approved Rule 并提交候选，不拥有规则事实；
- Neo4j 只提供 Entity Relation 投影，本阶段不投影 Rule；
- accepted `rule_inferred` DirectImpactAssertion 必须引用具体 Rule key/version 和
  Entity Relation ID；
- Source 类型、Variable、Direction、Relation、Target 类型任一不匹配，或 Rule 不是
  approved，都不得形成 Accepted Assertion；
- Rule 输出的 affected Variable Definition 必须已经 active、适用于 Target 类型且
  允许目标方向，否则 Rule 本身不能 approved。

Golden 示例：

```text
Event:
  某晶圆厂 8 英寸晶圆产量下降 10%

VariableSignal:
  subject = 某晶圆厂 Company
  variable = production_volume
  direction = decrease
  modality = actual

EntityRelation:
  某晶圆厂 produces 8 英寸晶圆 Product

DirectTransmissionRule:
  rule_key = production_decrease_reduces_product_supply
  version = 1
  company + production_volume + decrease
  + produces + product
  = market_supply + decrease

Accepted DirectImpactAssertion:
  target = 8 英寸晶圆 Product
  affected_variable = market_supply
  affected_direction = decrease
  rule = production_decrease_reduces_product_supply@1
  relation = 具体 produces Relation
```

四类对象严格区分：

- VariableSignal：具体 Event 产生的 Event-native ABox；
- EntityRelation：具体实体之间的关系事实；
- DirectTransmissionRule：可重复使用、无实例 ID 的 TBox 一跳因果规则；
- DirectImpactAssertion：本次 Signal + Relation + Rule 得出的具体一跳分析断言。

本阶段只建设约 3–5 条人工定义、人工批准、单跳、无递归、无自动学习、无主观强弱判断
的 Rule。不做多跳、通用 DSL、复杂条件计算或完整 Transmission Rule 引擎。

### 5.6 Event Semantic Submission / Execution Provenance

AgentRun 已拥有 Agent Definition、Agent Version 和 Agent Execution。Data 不应复制
AgentRun 的完整执行状态机。候选设计需要区分：

- AgentRun-owned `Agent Execution`：实际运行、模型/Prompt/工作流版本、执行错误；
- Data-owned 发布或分析血缘记录：收到哪个 Event、来自哪个 Agent Execution、使用
  哪个 TBox/规则版本、提交内容哈希、校验与审核结果。

只读工程发现：

- AgentRun 通用 `agent_executions` 已拥有 `execution_id`、Agent key/version、
  idempotency key、trigger、运行状态、错误、开始/完成时间和产物摘要；
- 现有 Event Extractor 采用专用 `event_extractor_executions`，以同一个 execution UUID
  作为主键和外键，一对一补充 Prompt/Schema hash、Provider/Model、Tag Catalog 版本及
  Generator/Reviewer 调用计数；
- Data 的 Event Publication Receipt 只保存外部 `extractor_execution_id` 和 Agent
  version 等不可变发布血缘，不对 AgentRun 建外键，也不复制运行状态；
- Data 已有 Research Theme `analysis_batch_id`，其语义是 Theme 发布批次和幂等身份，
  不应与本阶段 Event 语义运行混用。

D-008 已确认：

- AgentRun 为 Event Semantic Enricher 建立专用 execution profile，继续复用通用
  Agent Execution 的运行状态与错误；
- Data 侧对象明确命名为 `EventSemanticSubmission`，只拥有 Event 语义提交、确定性
  校验、AI Review Result、Acceptance Policy 决策和产物血缘；
- Data 只保存外部 Agent Execution ID、Agent key/version、Generator/Reviewer/
  Adjudicator Prompt hash 与 Model 标识快照、Ontology/Variable/Rule/Acceptance Policy
  版本、canonical payload hash、候选计数和最终决策摘要；
- AgentRun runtime error、重试调度和模型调用明细不复制到 Data；Data 只保存
  submission/validation/review 的领域失败；
- 一个 Agent Execution 只分析一个 Data Event，并一对一映射一个
  EventSemanticSubmission；重新分析使用新的 Work Item、Execution 和 Submission，通过
  `supersedes_submission_id` 关联旧 Submission，不原地覆盖。

### 5.7 AgentRun Agent 清单与可观察性补充

只读发现必须区分“代码和 Registry 支持的 Agent”与“此刻正在执行的 Agent”。

当前正式注册：

| Agent Definition / Version | 职责 | 触发方式 |
| --- | --- | --- |
| `collector / collector.v1` | 规划采集查询，调用固定 Connector，经确定性门禁生成 Raw Document Artifact | API 或 Schedule |
| `event-fact-extractor / event-fact-extractor.v1` | 消费 Artifact Ready Signal，提取和独立复核原子 Event，并通过 Data Event Publication 发布 | Collector Artifact 发布后的 dependent worker |
| `event-semantic-enricher / event-semantic-enricher.v1` | 消费 AgentRun-owned Event Semantic Work Item，生成候选、独立 AI 复核并发布 Data Submission | Eligible Event 发现或显式重分析请求 |

`event-semantic-enricher.v1` 已注册。D-006 的 Generator、Reviewer 和可选
Adjudicator 是同一 Event Semantic Enricher Execution 内的独立模型调用/阶段，不注册成
三个独立 Agent Definition。

2026-07-29 本地运行快照：

- AgentRun 服务容器当前未运行，只有 PostgreSQL 与 Neo4j 容器运行；
- 两类 Agent 的 active execution 均为 0；
- Collector Schedule 为 `*/5 * * * *`，但 `enabled=false`；
- Collector 历史 20 次：12 succeeded、4 partially_succeeded、3 failed、1 skipped；
- Event Fact Extractor 历史 69 次：13 succeeded、56 failed；56 次均为
  `event_fact_rejected`，属于确定性候选拒绝而非基础设施故障；
- 当前 Artifact Ready Signal 已 dispatched；Extraction Unit 为 8 published、
  16 rejected、2 no_events；没有正在等待的 active work。

现有可观察能力：

- `/healthz` 与 `/readyz` 提供进程和依赖就绪检查；
- AgentRun Admin API `GET /api/admin/v1/agent-executions` 支持按 `agent_key` 查询分页
  Execution 审计元数据，包括状态、触发、错误、阻塞关系和时间；
- Schedule Admin API 可查看 Agent 计划；目前只有 Collector 可调度；
- Collector 专用 `GET /api/v1/collector/runs/{execution_id}` 可看单次 Collector
  详情；
- Admin Portal 有 Collector 配置与执行记录页面，但 BFF 将列表固定过滤为
  `agent_key=collector`，看不到 Event Fact Extractor；
- AgentRun 输出 JSON access/lifecycle log 到 stdout；
- PostgreSQL 保存 Agent Execution、Event Extraction Work Item/Unit、Publication
  Journal 等审计事实。

实现前缺口：

- 没有通用当前运行总览；
- 没有 Event Fact Extractor 或未来 Event Semantic Enricher 的安全执行详情 API；
- Admin Portal 看不到 Generator/Reviewer/Adjudicator 阶段、Data Submission/Receipt、
  accepted/pending/quarantined/rejected 计数、自动重试或队列滞留；
- `event_fact_rejected` 记为 Execution failed，会把预期领域拒绝与基础设施失败混在一起；
- 没有 `/metrics`、OpenTelemetry trace、Dashboard 或告警规则。

Phase One 已增加最小只读 **Agent Status Monitor**，由 AgentRun 提供安全数据接口、
Admin Portal 展示。每个已注册 Agent 只返回：`agent_key`、显示名称、当前 Version、
`is_working`、当前 Execution Status（无在途 Execution 时为 `idle`）及 `updated_at`。它不
展示单次执行详情、执行阶段、耗时、重试、候选计数、quarantine、错误详情、Data Submission 关联、
Prompt、模型自由推理、Evidence 正文、Connector 响应或凭据。Prometheus / OpenTelemetry、
长期 Dashboard 和告警系统继续延后。

## 6. 第一批 Variable Definition 候选

事件推理模型基于四类目标场景把初始全集压缩为以下 12 个语义候选。只读发现已经取得
现有 Event、Artifact、Evidence 与发布血缘，但当前已发布样例不足以覆盖四类场景；
若干语义较好的 Artifact 又尚未成为 Data Event。因此这些定义仍未通过完整 Golden
Scenario 验证，不能视为已冻结首批目录。

统一候选规则：

- 所有定义允许 `increase / decrease / unchanged / mixed / uncertain`；
- 数值可以为空，但只有 Event 明确给出方向性陈述时才允许为空；
- Event 给出数值时必须提取，并保留单位、口径和比较周期；
- `actual / stated_intent / source_forecast` 由 Signal `assertion_modality` 表达，不为
  预测另建 Variable Definition；
- Event 没有明确陈述的下游变量变化只能进入 Direct Impact 或后续推理，不能伪装成
  Event 原生 Variable Signal。

### 6.1 供需与产业变量

| Key | 名称 | Domain | Value type | 数值/单位候选 | 适用 Entity Type |
| --- | --- | --- | --- | --- | --- |
| `market_supply` | 市场供给 / Market Supply | `supply_demand` | `quantity_or_index` | 可空；物理量、指数或变化百分比 | commodity, product, chain_node, industry, industry_chain（有条件） |
| `market_demand` | 市场需求 / Market Demand | `supply_demand` | `quantity_or_index` | 可空；物理量、指数或变化百分比 | commodity, product, chain_node, industry, industry_chain（有条件） |
| `market_price` | 市场价格 / Market Price | `pricing` | `monetary_per_unit` | 可空；货币/计量单位或变化百分比 | commodity, product |
| `production_volume` | 产量 / Production Volume | `operations` | `quantity` | 可空；物理单位或变化百分比 | commodity, product, chain_node, company, industry |
| `sales_volume` | 销量 / Sales Volume | `operations` | `quantity` | 可空；物理单位或变化百分比 | product, company, industry |

业务边界：

- `market_supply` 表示指定市场和时间窗口中的可供给数量或充足程度；
- `market_demand` 表示指定市场和时间窗口中的需求数量或强弱；
- `market_price` 只接受 Event 明确提供的成交、现货、合同或公开报价事实；
- `production_volume` 与 `sales_volume` 分别表达生产和销售/交付，不互相替代。

### 6.2 企业订单与财务变量

| Key | 名称 | Domain | Value type | 数值/单位候选 | 适用 Entity Type |
| --- | --- | --- | --- | --- | --- |
| `order_quantity` | 订单数量 / Order Quantity | `company_operations` | `quantity` | 可空；有值时使用产品数量单位 | company, product |
| `order_value` | 订单金额 / Order Value | `company_operations` | `monetary` | 可空；有值时包含币种与金额尺度 | company, product |
| `revenue` | 营业收入 / Revenue | `company_financials` | `monetary` | 可空；包含币种、期间和同比/环比口径 | company |
| `net_profit` | 净利润 / Net Profit | `company_financials` | `monetary` | 可空；包含利润口径、币种和期间 | company |
| `gross_margin` | 毛利率 / Gross Margin | `company_financials` | `ratio` | 可空；区分百分比与百分点变化 | company |

订单数量与订单金额不得混成同一 Variable Definition。实际财报与企业未来指引使用同一
Variable Definition，通过 `assertion_modality` 区分。

### 6.3 政策变量

| Key | 名称 | Domain | Value type | 数值/单位候选 | 适用 Entity Type |
| --- | --- | --- | --- | --- | --- |
| `policy_support_intensity` | 政策支持强度 / Policy Support Intensity | `policy` | `ordinal_directional` | 阶段一只保存方向、措施和 Evidence | commodity, product, industry_chain, chain_node, industry, company, sector, concept |
| `regulatory_restriction_intensity` | 监管限制强度 / Regulatory Restriction Intensity | `policy` | `ordinal_directional` | 阶段一只保存方向、措施和 Evidence | commodity, product, industry_chain, chain_node, industry, company, security, sector, concept |

阶段一不为了 Security、Sector、Concept 强行增加股价、资金或热度变量；它们可作为
Direct Target。只有真实 Event 明确描述这些对象自身变量时，才复审目录。

### 6.4 四类场景覆盖

候选真实 Event 类别：

1. 供给冲击：停产、事故、制裁、出口限制、扩产或复产；
2. 政策：补贴、许可、监管收紧/放松或产业目标；
3. 企业订单 / 财报：订单、收入、利润、毛利率、资本开支或指引；
4. 行业需求：终端需求、云厂商资本开支、库存周期或交付周期变化。

约束样例：

- 矿山停产可直接支持 `production_volume decrease`；若 Event 未声明整体供给或价格
  变化，不得同时生成 `market_supply decrease` 或 `market_price increase`；
- 正式公布但尚未生效的补贴政策可形成
  `policy_support_intensity increase + stated_intent`；
- “获得 10 万台订单”使用 `order_quantity`，“签署 20 亿元合同”使用
  `order_value`；
- 行业协会公布销量同比增长可形成 `sales_volume increase + actual`；明确机构预测
  需求增长可形成 `market_demand increase + source_forecast`。

### 6.5 明确延后或排除

| 初始候选 | 处理 | 理由 |
| --- | --- | --- |
| 成本、原材料成本 | 延后 | 在定义 `input_cost` 前，成本机制只能保留为不可接受的候选说明，不能用 `negative` 绕过 affected Variable |
| 库存 | 延后 | 依赖 Observation 和时间序列口径 |
| 产能 | 延后候选 | A/C 证明其有价值，但不为适配混合文章扩大首批目录；应继续寻找可由 `production_volume` 或 `market_supply` 表达的纯供给冲击 |
| 产能利用率 | 延后 Observation 候选 | 通常是时点水平值，依赖一致的产能、产量和 Observation 口径 |
| 交付周期 | 延后 | 起止环节、单位和样本口径复杂 |
| 进出口量 | 条件加入 | 依赖 Region、贸易方向、口岸和统计口径 |
| 市场份额 | 延后 | 必须冻结市场边界和分母 |
| 资本开支 | 延后 | 尚未被四类首批样例证明为必要 |
| 行业景气度 | 不作一级原生变量 | 通常是多个变量综合形成的推理指标 |
| 盈利预期 | 不单独建定义 | 使用 `revenue` / `net_profit + source_forecast` |
| 资金流入、市场关注度、估值水平 | 延后 | 依赖市场 Observation 与明确聚合口径 |
| 模糊“利润” | 不发布 | 首批使用口径明确的 `net_profit` |

正式冻结第一批 Variable Definition 前，至少需要四条带 Event ID、Artifact/Raw
Document ID、`occurred_at`、Evidence ID/原文片段、来源和审核状态的真实候选。未被
真实样例使用且不是后续 Reason Tree 必需的定义，不进入首批发布。

## 7. AgentRun 候选工作流

```text
AgentRun 查询 Data eligible Event
→ AgentRun 幂等建立本地 Event Semantic Work Item
→ AgentRun 租约领取 Work Item 并创建 Agent Execution
→ Data 以 Agent Execution ID 为幂等身份，为该 Event / superseded Submission 创建短时 Context Lease
→ Data 在 Lease 创建事务中持久化固定 Event / Evidence / Ontology / EntityRelation Context
→ 识别 Entity Mention 与角色
→ 调 Data Service 解析规范 Entity
→ 提取 Variable Signal 候选
→ 查找 Direct Target 候选
→ 生成严格结构化候选
→ 提交 Data Service
→ Data 确定性校验、独立 AI Review 与原子保存
→ AgentRun 关闭 Execution，并按本地 Work Item 策略重试或终结
→ Data Read API 读回验证
```

模型只能输出 Data Service 已提供的 ID 和 Variable Definition；不能生成新 ID 或自由
变量名。确定性 application code，而不是模型，拥有提交、重试和最终状态选择。

## 8. Data Service 最小能力候选

下列是能力名，不是已批准的 HTTP route 或 operation ID：

| 候选能力 | 当前评估 | 待确认事项 |
| --- | --- | --- |
| `list_eligible_events` | 新建只读查询 | confirmed + verified、有时间和 Evidence，排除已有活动 Submission / Context Lease |
| `get_event_with_evidence` | 可能扩展现有 Event read | Evidence 完整度、正文边界、批量读取 |
| `get_ontology_context` | 新能力候选 | Lease 创建时持久化的 TBox、Entity、EntityRelation 完整一致快照 |
| `search_entities` | 新能力候选 | 搜索策略、候选上限、类型过滤 |
| `resolve_entities` | 新能力候选 | 确定性输入、歧义、批量和拒绝语义 |
| `get_variable_definitions` | 新能力候选 | 版本快照、active filter、缓存 |
| `get_direct_transmission_rules` | 新能力候选；也可并入有界 Ontology Context | approved 版本快照、关系类型和条件表达 |
| `search_direct_targets` | 新能力候选 | 允许类型、关系范围、排序与截断 |
| `create_context_lease` | 新能力 | 为 AgentRun 已领取的 Work Item 创建 Event/Ontology 快照租约；不是任务队列 |
| `create_event_semantic_submission` | 新写入能力 | 持久化候选、执行 Data 确定性预检并返回 Reviewer 工作包 |
| `submit_event_semantic_review` | 新写入能力 | 接收独立 Reviewer / Adjudicator 结构化结果，由 Data 最终裁决 |
| `get_event_semantics` | 验收所需读取能力 | 完整结果、状态、血缘和稳定排序 |

每个最终 API 都必须冻结：

- provider / consumer；
- route、method、operation ID、DTO 和冻结 fixture；
- service-token scope；
- 总 timeout；
- GET retry 与 mutation 重放规则；
- 请求/响应 body 上限；
- null、time、order、pagination 语义；
- `400/401/403/404/409/422/500/503` 的稳定错误分类；
- 批次原子性与部分失败语义；
- rollout 顺序和旧/新版本共存窗口。

### 8.1 D-010 已确认的两阶段语义提交

为保证 Data 的确定性门禁先于独立 AI Reviewer，阶段一不使用一次性
`submit_event_semantics`。正式写入合同分为两步：

```text
AgentRun Candidate Generator
→ POST create_event_semantic_submission
→ Data 持久化 Candidate + 确定性预检
→ 返回可复核 Candidate 的 canonical Reviewer 工作包
→ AgentRun 独立 Reviewer / 可选 Adjudicator
→ POST submit_event_semantic_review
→ Data 结合 Review Result + Acceptance Policy 最终裁决
```

第一步的 Data 响应逐项给出预检状态；结构合法但领域不成立的 Candidate 已持久化为
`rejected`，不进入 Reviewer。第二步只接受第一步产生的 Candidate identity 和结构化
Review Result；Data 重新校验引用、依赖、版本和状态，再决定 accepted / pending / needs
reanalysis / quarantined / rejected。AgentRun 不拥有或直写 Data 最终状态。

每个 API 的确切 route、DTO、认证、timeout、重放和部分失败语义仍在 D-010 后续 Grill 中
冻结。

### 8.2 D-010B 已确认的单 Event 原子快照与逐项裁决

一个 `EventSemanticSubmission` 固定只处理一个 Data Event。一次候选提交或一次 Review
提交以该 Submission 为数据库事务边界：该次请求中的全部结构化 Candidate / Review 及其 Data
裁决要么完整持久化，要么全部不持久化，不留下半份快照。

这不意味着候选必须全成全败。进入 Data 的结构与引用校验后，各 Candidate 独立裁决：同一
Submission 可同时保存 `accepted`、`pending_review`、`needs_reanalysis`、`rejected` 或
`quarantined` 的结果。一个 Candidate 被拒绝或待重分析，不能回滚同一次提交中其他合法
Candidate 的保存与裁决。无法解析或不符合 DTO 合同的整个请求仍以 4xx 拒绝，且不创建
快照。

### 8.3 D-010C 已确认的 Submission 幂等与重新分析

`agent_execution_id` 是一次 AgentRun 执行创建 Data 语义 Submission 的唯一幂等身份。同一个
`agent_execution_id` 只能创建一个 `EventSemanticSubmission`：携带完全相同 canonical
payload 的重试返回既有 Submission 与既有结果；同一身份携带不同 payload 必须返回
`409 Conflict`，不得静默覆盖或产生第二个 Submission。

未知 POST 结果时，AgentRun 只允许重放完全相同的请求。真正的重新分析（包括新模型、
新 Prompt、补充 Evidence、重新实体解析或不同规则 / Policy 快照）必须创建新的 Agent
Work Item、Execution 和 Data Submission，并以 `supersedes_submission_id` 显式关联旧
Submission；旧 Submission 继续保留完整审计记录。显式重新分析请求首先进入 AgentRun
内部 API 和本地 Work Item 队列，Data Service 不建立重分析任务。

### 8.4 D-010D AgentRun Work Item 与 Data Context Lease

用户授权阶段一的常规执行机制按项目开发规范、可恢复性与不过度设计原则自行冻结，只有
架构边界或阶段目标变化才再次逐项确认。任务闭环完全属于 AgentRun：

- AgentRun 持久化 `event_semantic_work_items`，拥有 pending/running/succeeded/failed、
  执行租约、尝试次数、最多两次重试、幂等键和当前 Agent Execution；
- Eligible Event 以 `event-semantic-initial:{event_id}` 幂等建初始 Work Item；
- 显式重分析通过 AgentRun
  `POST /api/agentrun/v1/event-semantic-reanalysis` 创建 Work Item，必须携带目标 Event、
  被替代 Submission 和 Idempotency-Key；
- Worker 只从 AgentRun PostgreSQL 领取 Work Item。失联或执行租约到期时由 AgentRun
  重排队；预算耗尽后 Work Item 失败，不再自动运行；
- 同一 Work Item 的短暂重试保持同一个 Agent Execution ID，先按该身份从 Data Read API
  对账终态 Submission，再以该身份续租原 Context Lease 并重放完全相同的
  Submission/Review identity；续租复用同一 Context snapshot，不重新读取实时事实；
  只有显式重新分析才创建新的 Work Item 和 Agent Execution；
- Data 的 `ContextLease` 仅授权一次已领取 Work Item 读取固定 Event/Evidence/Ontology/
  Entity/EntityRelation 快照并提交对应 Submission。Lease 与 `agent_execution_id` 一一绑定；
  可恢复重试只延长有效期，不重建或刷新 snapshot。它不调度 Agent、不保存重试预算，
  也不是任务队列；
- Data 校验 initial/reanalysis 边界和 `supersedes_submission_id`，Context Lease 消费后
  不可复用；
- 不建设分布式协调器、长事务、全局锁或第二套 Data 任务机制。

### 8.5 D-010E 最小版本化 API 合同（授权按既有模式冻结）

阶段一沿用 Data Service 的 `/api/data/v1`、OpenAPI operation ID、Bearer service token、
scope 和请求大小限制模式；不经由 AgentRun 数据库代理 Data 事实，也不新增 API gateway。
正式路由按单一 `event-semantics` 资源域组织：

| Operation | Route | 主责与边界 |
| --- | --- | --- |
| `listEligibleEventSemanticEvents` | `GET /event-semantics/eligible-events` | 只读返回可创建初始 Work Item 的 Event；任务去重和领取由 AgentRun 完成。 |
| `createEventSemanticContextLease` | `POST /event-semantics/context-leases` | 以 `agent_execution_id` 为幂等身份，为已领取的 Event / superseded Submission 创建或精确续期短时数据快照租约；不创建任务。 |
| `getEventSemanticContext` | `GET /event-semantics/context-leases/{context_lease_id}/context` | 返回该 Lease 对应 Event、完整可用 Evidence、受限 Ontology / Variable / approved Rule 快照。 |
| `resolveEventSemanticEntities` | `POST /event-semantics/entity-resolutions` | 用名称、类型约束和 Event 上下文解析既有规范 Entity；不创建 Entity。 |
| `searchEventSemanticDirectTargets` | `POST /event-semantics/direct-targets:search` | 在已批准的 EntityRelation 范围内查询一跳 Direct Target；不做图遍历。 |
| `createEventSemanticSubmission` | `POST /event-semantics/submissions` | 持久化一个 Context Lease/Event 的 Candidate 快照并执行确定性预检，返回 canonical Reviewer 工作包。 |
| `submitEventSemanticReview` | `POST /event-semantics/submissions/{submission_id}/reviews` | 接收独立 Reviewer / Adjudicator 的结构化结果；由 Data 最终裁决。 |
| `getEventSemantics` | `GET /events/{event_id}/semantics` | 重新读取完整语义结果、状态、血缘和版本快照；供验收和未来下游使用。 |

所有 `event-semantics` 读操作需要 `data.event-semantics.read`，Context Lease、解析、搜索和写入需要
`data.event-semantics.write`；它们只授予 Event Semantic Enricher 的服务身份。语义 Submission 的
写接口继续使用 D-010C 的 `agent_execution_id + canonical payload` 幂等规则，而非增加
通用 Idempotency-Key header。

沿用当前 Data API 默认 1 MiB 请求上限；Context / Read 采用短时读取预算，写操作采用
与既有 import 相同的有界 15 秒总预算。稳定错误语义为：认证/授权 `401/403`、不存在或
过期 Context Lease `404`、幂等或 Lease/Submission 冲突 `409`、不可解析 DTO `400`、有效 DTO 的领域校验
失败 `422`、过大 `413`，暂时不可用 `503`。这些参数不是业务 Policy，不需要新增配置
平台；如 Golden Fixture 证明不足，再以兼容版本调整。

## 9. 确定性校验候选

至少覆盖：

- Event 和 Evidence 存在且可用于语义分析；
- Entity / Chain Node ID 存在、active 且类型正确；
- Variable Definition active 且版本明确；
- Variable Definition 适用于 Subject Entity Type；
- `signal_direction` 属于 Variable Definition 允许集合；
- 数值、幅度、单位和 `valid_from/valid_until` 合法；
- Direct Target 类型和关系范围合法；
- Direct Impact 的 affected Variable active、适用于 Target 类型且方向合法；
- `affected_direction` 与 `signal_direction` 分字段；
- `rule_inferred` 的 Source 类型、Variable、Direction、Relation、Target 类型与
  approved DirectTransmissionRule 完整匹配；
- Accepted `rule_inferred` Assertion 引用具体 Rule key/version 与 Entity Relation；
- Evidence 引用属于该 Event，禁止只有自由文本证据；
- 批次内稳定去重；
- 自动接受、待复核、拒绝由确定性 policy 选择；
- 未定义变量、模型编造 ID、越界 Entity Type 直接拒绝；
- Data 不信任 AgentRun 已做过的校验，Provider 重新验证全部领域不变量。

## 10. 状态、幂等和失败语义待确认

### 10.1 D-006 审核状态与依赖传播

EventEntityLink、VariableSignal 和 DirectImpactAssertion 独立审核；AnalysisRun /
提交批次只汇总结果，不以一个共享状态覆盖所有对象。

```text
EventEntityLink accepted
→ VariableSignal 才可能 accepted

VariableSignal accepted
→ DirectImpactAssertion 才可能 accepted
```

| 上游状态 | 下游处理 |
| --- | --- |
| `accepted` | 继续独立校验 |
| `pending_review` | 下游只能 `pending_review`，原因 `upstream_pending` |
| `rejected` | 下游为 `rejected`，原因 `upstream_rejected` |

Direct Impact Target 不强制在 EventEntityLink 中出现，但 Target Entity、Entity
Relation、affected Variable 和 Rule 必须有效。

请求级 4xx 不创建候选，只用于整个请求无法安全处理：认证/授权失败、非法 JSON、请求级
必填字段缺失、不支持的契约版本、请求过大、请求级幂等身份与载荷冲突、越权引用，或
ID/枚举连 DTO 都无法解析。

结构合法但领域上确定不成立的项保存为 `rejected` Candidate，并记录稳定原因，例如：

- Entity/Event/Evidence/Rule 引用不存在或已失效；
- Entity 类型、Variable applicability、方向或单位维度非法；
- Evidence 不属于 Event 或不支持变量、方向、模态；
- Agent 预测冒充 Event-native Signal；
- Target 等于 Subject 或 Target 类型被禁止；
- affected Variable 未定义；
- `event_explicit` 没有直接因果 Evidence；
- `rule_inferred` 缺少 Relation，或 Rule 未批准/版本失效/不匹配；
- 上游已 rejected。

信息可能成立但不足以确定时进入 `pending_review`，例如实体解析歧义、Evidence 语义
支持不确定、时间或预测期可修复但缺失、数值区间/币种/单位口径可修复、独立语义审核
无法确定、上游 pending，或版本化 Acceptance Policy 将其路由到待复核。

自动接受必须同时满足各对象的确定性门禁：

| 对象 | Accepted 必要条件 |
| --- | --- |
| EventEntityLink | Event `confirmed + verified`；有有效 Evidence；Entity active、类型和角色合法；Evidence 能定位规范名称/别名/明确指代；唯一无冲突匹配；确定性解析或独立语义审核通过 |
| VariableSignal | Subject Link accepted；Variable approved 且适用；Evidence 逐字支持变量、方向和模态；时间、数值、区间、比较口径、原始单位合法；有值时完整提取；独立语义审核通过 |
| `event_explicit` Impact | Source Signal accepted；Target 合法且不同于 Subject；affected Variable 适用；Evidence 直接表达因果、目标变量和方向；不含估值、股价或投资推测；独立语义审核通过 |
| `rule_inferred` Impact | Source Signal accepted；唯一有效 EntityRelation；唯一 approved Rule 版本；Source/Relation/Target/affected Variable/Direction 和条件全部确定性匹配 |

Confidence 分别保留为 `entity_resolution_confidence`、
`signal_extraction_confidence` 和仅适用于 `event_explicit` 的
`impact_assertion_confidence`。它们是模型或解析路由特征，不是领域真值，不能单独让
候选 accepted 或 rejected。当前没有标注集与校准基线，禁止把 0.95/0.92/0.70 等
伪精确数字写成领域常量。数值阈值只能在 Golden positive/negative fixture 校准后，
进入版本化 `AcceptancePolicy`，并按对象和 Signal modality 分别管理。

现有 Evidence Grade 保持原义：

- A：full text 且 Source Type 为 official/government/filing；
- B：其他 full text；
- C：summary/snippet 等非全文覆盖。

Evidence Grade 和 `source_level = secondary` 只能用于审核优先级、补取全文、置信分析
和质量统计，不能单独决定状态。`stated_intent` / `source_forecast` 的严格门禁通过
原始声明主体、statement time、effective/forecast period、逐字 Evidence 和归因血缘
独立表达。

区间值必须保留上下界；保存原始值与原始单位，只有确定性转换表才能产生规范值。
`published_at` 不能自动替代 Event `occurred_at`。时间确认无法恢复时 rejected，可修复
时 pending。

Phase One 没有人工审核人员，也不建设人工审核 UI。所有审核由 AI 与 Data Service
确定性规则自动完成：

```text
Candidate Generator AI
→ Data Service 确定性门禁
→ 独立 AI Semantic Reviewer
→ Data Service + Versioned Acceptance Policy 最终裁决
```

流程约束：

1. Generator 生成 EventEntityLink、VariableSignal 和 DirectImpactCandidate；
2. Data Service 先校验 ID、Entity Type、Variable Definition、Evidence 血缘、时间、
   单位、Entity Relation、Rule 和依赖状态；
3. 通过门禁后，独立 Reviewer 输出结构化 `pass | fail | indeterminate`，逐项引用
   Evidence 并检查实体、变量、方向、modality 和因果支持；
4. 只有 `pass + 全部确定性门禁通过` 才能由 Data Service 按 Acceptance Policy
   标记 accepted；
5. `fail` 或确定性硬冲突进入 rejected，并保存原因；
6. `indeterminate` 或可修复缺口进入 pending_review / needs_reanalysis，允许补取全文、
   重新解析实体或用独立 Reviewer 再审；
7. 自动重试必须有上限；预算耗尽仍不确定时进入 quarantined，原因
   `unresolved_after_retry_budget`，长期保存但永不进入下游。

Generator 与 Reviewer 必须是独立模型调用、独立 Prompt/版本。MVP 可以使用同一基础
LLM，但 Reviewer 不接收 Generator 的自由推理过程，只接收候选、Event Evidence、
Ontology Context 与校验清单。两者分歧时，可以在重试预算内调用最多一次独立
Adjudicator；不建设开放式多 Agent 讨论。

Data Service 是最终状态裁决者。AI Reviewer 只能提交结构化审核结果，不能直接写
accepted。AgentRun 拥有 Generator、Reviewer、可选 Adjudicator 的执行与模型调用记录；
Data 以确定性门禁、AI Review Result 和 Acceptance Policy 共同裁决并保存领域状态。

逻辑状态语义：

| 状态 | 含义 |
| --- | --- |
| `pending_review` | 正在等待自动 AI 复核或可恢复处理，不表示等待人工 |
| `needs_reanalysis` | 需要重新提取、补充 Evidence 或重新 Entity Resolution |
| `quarantined` | 自动流程和重试预算已经耗尽，长期隔离且禁止进入下游 |
| `accepted` | 确定性门禁通过、独立 Reviewer pass，且 Acceptance Policy 允许 |
| `rejected` | 确定性失败或独立 Reviewer 明确 fail |
| `superseded` | 已由重新分析产生的新候选替代；旧记录继续保留审计 |

这些逻辑语义最终使用单一状态字段还是“review status + processing disposition”两个字段，
留待数据模型与 API Grill；不得改变各状态的下游可见性。没有人工审核时允许 Pending /
Quarantined 长期存在，绝不能因此降低门槛或自动 accepted。

Golden 验收至少覆盖：

- `submitted → accepted | pending_review | rejected`；
- `pending_review → accepted | rejected | superseded`；
- 上游 pending/rejected 的下游传播；
- 精确 Entity 匹配、三种 Signal modality、event-explicit 和 rule-inferred Impact；
- 批次内 accepted/pending/rejected 并存且互不污染；
- 幂等重提不产生重复候选；
- Generator 正确但 Reviewer fail；
- Generator 错误被 Reviewer 拦截；
- Reviewer indeterminate 后在预算内重试成功；
- 重试耗尽进入 quarantine；
- 只有 accepted 候选能进入下游；
- 稳定拒绝原因，包括 entity/variable/evidence/time/unit/target/relation/rule/upstream
  各类失败。

### 10.2 仍待确认

以下状态与失败语义按已确认的自动审核范围执行；如其改变 Data / AgentRun 所有权或阶段
目标，再单独 Grill：

- `pending_review` 的自动复核调度、重试预算与隔离触发由版本化 Acceptance Policy 参数
  决定，不引入人工审核 UI；
- Submission 采用新不可变版本并 `supersede` 旧结果，详见 D-008 与 D-010C。

## 11. Measurement Value 与 Observation

D-007 确认 Phase One 不建立独立 Observation。当前所有数值均来自某条 Event
Evidence，是 Event-native VariableSignal 的组成部分：

```text
Event Evidence
→ VariableSignal
→ MeasurementValue[]
```

`MeasurementValue` 是未来 Observation 可以直接复用的领域值对象，不是松散的
`value/unit` 字段。一个 Signal 允许零个或多个 Measurement；每个 Measurement 必须有
受控角色：

| `measurement_role` | 含义 |
| --- | --- |
| `absolute_level` | 收入、利润、价格、产量等绝对水平 |
| `absolute_change` | 相对基准期增加或减少的绝对数量/金额 |
| `relative_change` | 同比、环比等百分比变化 |
| `percentage_point_change` | 毛利率等比率变化的百分点 |

`value_shape` 只允许 `exact | range | lower_bound | upper_bound`。概念字段至少能够表达：

- 原始值、上下界、单位、原文和来源精度；
- 规范值、上下界和规范单位；
- `currency`、`scale`；
- `comparison_basis`、`comparison_period`；
- `is_approximate`。

同一 Measurement Role + comparison basis 原则上只能出现一次。预计净利润 17 亿至
18.3 亿元、同比增加 41.5% 至 52.32% 是一个 `net_profit + source_forecast`
Signal，携带 `absolute_level range` 和 `relative_change range` 两个 Measurement，
不能拆成两个 Signal。

方向与量化规则：

- Evidence 明确陈述方向时允许 `measurements=[]`；
- 只有绝对值、没有比较或变化陈述时，`direction=uncertain`；
- 变化区间全为正/负/零分别对应 increase/decrease/unchanged；
- 变化区间跨越零为 uncertain；mixed 只用于 Evidence 明确包含不同子项或时期的相反
  方向；
- “最高上涨 15%”使用 `relative_change + upper_bound`，不得改为 exact，也不得假设
  下界为零；
- 数值与文字方向存在可修复歧义时 needs_reanalysis，明确冲突时 rejected。

时间必须区分：

- Event `occurred_at`：现实事件或声明行为发生时间；
- `statement_at`：公司、政府或机构作出意图/预测的时间；
- `reference_period` / `measured_at`：actual 数值描述的报告期或测量期；
- `effective_from/effective_to` 或 target period：stated_intent 指向的期间；
- `forecast_period_start/end`：source_forecast 指向的期间。

`published_at` 只表示文档发布时间，不能替代业务时间。

数值与单位规则：

- 必须保留原始数字文本、单位、边界、Evidence span 和小数精度；
- 金额、比率和数量使用十进制语义，不使用会引入误差的浮点语义；
- 金额拆分数值、尺度和币种；Phase One 只做尺度规范化，不做汇率转换；
- 百分比与百分点严格区分；
- `lower <= upper`；不取中位数、不把单边界扩为区间、不丢失“约/超过/最高”；
- 规范值不得制造高于来源的精度；
- 只有批准的确定性单位注册表可以产生规范值；维度不匹配 rejected，含义可修复但不清楚
  时 needs_reanalysis。

出现连续多期采集、Event 外指标更新、同比/环比自动计算、数据 vintage、多来源融合、
时间序列查询、预测兑现验证，或 capacity utilization / 库存 / 资金流 / 估值等持续
指标需求时，Phase Two 再正式引入 Observation。届时复用同一 MeasurementValue，
将内嵌 Measurement 无损迁移为 Observation，并保留 Event、Evidence、业务时间和
Signal 派生关系。

Measurement 的 PostgreSQL 物理形态、Decimal/API 序列化、单位编码、数组上限和时间
DTO 留待工程 Grill。

## 12. 验收候选

### 12.1 真实样例

Golden Scenario 是用现有系统真实 Event 验证 Variable Definition 与 Event 语义合同的
标准验收样例，不是新数据源、采集需求或新领域对象。阶段一默认输入是 Data Service
中已经存在、Evidence 完整且具备分析资格的 Event。

选择时不限制最近 48 小时，优先使用事实明确、证据完整、血缘可追溯的历史 Event。
先通过只读工程发现，从现有 Data 和 AgentRun 各选择一条：

- 供给冲击；
- 政策；
- 企业订单或财报；
- 行业需求。

每条候选必须列出：

- Data Event ID；
- Artifact / Raw Document ID；
- Evidence 摘录与引用；
- 来源；
- `occurred_at`；
- Event / Fact / Review 状态；
- AgentRun 提取执行和发布记录；
- 预期 Entity Resolution、Variable Definition、Signal、Direct Target 和审核结果。

选择顺序：

1. 优先使用 Data Service 中已经发布、证据完整、符合分析资格的 Event；
2. 若现有数据库没有合格样例，运行现有 Collector 与 Event Extractor 生成少量真实
   Event；
3. 或选取少量真实公开材料，通过现有发布链路进入 Data Service，形成冻结 fixture；
4. 只有证明现有采集能力无法覆盖必要来源时，才提出独立 Connector 前置任务，且不得
   混入阶段一变量模型与 Event 语义实现。

互联网到 Artifact/Event 的链路属于现有 Collector / Event Extractor。阶段一只负责
Event → Entity → Variable Signal → Direct Target。

只有数据库/环境不可访问、必要凭据缺失，或现有工程事实无法只读发现时，才向用户确认
具体环境或权限。

#### 12.1.1 只读工程发现（2026-07-29）

本地 PostgreSQL 可访问；Data 和 AgentRun HTTP 服务当前未运行，但不影响本轮数据库、
Artifact Volume 和发布血缘的只读检查。

当前 Data 基线：

- `events = 8`、`raw_documents = 8`、`event_sources = 8`、
  `event_publication_receipts = 8`；
- 8 条 Event 均为 `event_status = confirmed`、`fact_status = verified`；
- 每条均有一个 `primary + supports` Evidence、Raw Document 和 publication receipt；
- 其中 6 条没有 `event_time`；现有 8 条不能组成供给冲击、政策、企业订单/财报、
  行业需求四类合格样例。

当前唯一直接命中候选变量目录的已发布 Event 是：

| Event | Raw Document / Artifact | 来源与时间 | Evidence | 只读评估 |
| --- | --- | --- | --- | --- |
| `ee8c891b-b048-55d9-b24c-0ddbae6de329` | `c4b74bc6-188a-515a-9dc5-1b7949a40b64` / `sha256:5a4b7682bf33b98ee698a9ea93c9180da1f52dae397580ddb56165db7bcf1245` | 招商证券；`published_at = 2026-07-26T12:00:03Z`；`event_time = null` | “存储器价格上涨” | 可映射 `market_price increase + actual`，但 Evidence 过薄且发生时间缺失，待事件推理模型判断是否只能作为负例 |

Collector execution `9519b19c-25c9-477a-af46-fe80e66f621b` 的 Artifact 与 AgentRun
extraction unit 可追溯，但以下候选均为 `unit_status = rejected`，尚无 Data Event ID、
Event Evidence 或发布回执：

| 类别 | Artifact | 来源 | 可核验原文事实 | 当前缺口 |
| --- | --- | --- | --- | --- |
| 供需 / 供给冲击 | `sha256:abece75c9d0eabfe9994295c270dadfb120aa01589b4cdab501b6d43180761cf` | 第一财经《供需缺口持续扩大，功率半导体迎涨价潮》 | STMicro 部分产品涨价最高 15%；TSMC / Samsung 缩减 8 英寸成熟制程产能；AI 服务器需求增强；部分晶圆厂产能利用率接近 90% | 需裁剪为原子 Event，并经现有 Extractor / publication 链路形成 Data Event |
| 企业财报 | `sha256:def69f26cb0964b45cc87179c4ed72d1be43d52ad68d4cc6b9bd0ffab1518c6d` | 腾讯网 / 科创板日报《科创板半导体公司业绩高增态势凸显》 | 海光信息预计净利润 17 亿至 18.3 亿元、同比增加 41.5% 至 52.32%；另含摩尔线程收入预告 | 需选择单一公司、单一报告期的原子 Event，并经现有链路发布 |
| 行业供给 / 产能候选 | `sha256:21b21c165ad36374803e21d657550ba2b1035a64aa711c4be719340631de7fc4` | 中新网 / 人民日报《截至 6 月底我国智能算力规模同比增长 177%》 | 智能算力规模 2185 EFLOPS（FP16）、同比增加 177%；设施平均利用率 71.4% | 语义更接近 capacity / utilization，不当然证明 `market_demand`；待判断是否扩充首批目录 |

另有两个原文语义候选，但它们所在 Collector run 在当前 AgentRun 数据库中没有可核验
execution / ready signal，不能直接视为现有合格输入：

| 类别 | Artifact | 来源与内容 | 当前处理 |
| --- | --- | --- | --- |
| 政策 | `sha256:82bbdef573ce20f91e568e39f2541e9fd3cc55e81f2bee5f5c362a832e54e78f` | 深圳高新投转载工信部《人工智能+信息通信创新发展实施意见（2026—2028 年）》完整文本，含 17 项任务及 400/800G、光芯片、CPO 等措施 | 仅作为语义候选；若采用，必须通过现有采集、提取和发布链路重新进入 Data |
| 行业需求 | `sha256:b75ab1fd47dcf311640df9973abf43373cde32c09a7a1554914fa4564e70d50f` | 界面新闻《价格涨超 300%，订单排到 2028 年，AI 算力“硬通货”PCB 持续爆发》，含产能利用率、排期、订单及需求预测 | 仅作为替补语义候选；需先判断混合事实、预测口径与来源是否适合 Golden fixture |

因此当前结论不是“现有系统没有真实数据”，而是“现有已发布 Event 不足以构成四类
Golden positive fixture”。下一步先由事件推理模型裁决语义适配；本轮不运行 Collector、
Extractor 或 publication，不新增 Connector。

#### 12.1.2 D-003 语义裁决

事件推理模型裁决：不能为了凑齐四类样例，降低 Evidence、时间或血缘标准。

| 场景 | 裁决 | 最小合法 Signal | Fixture 处理 |
| --- | --- | --- | --- |
| 供给冲击 | 当前缺口 | 继续寻找能被原文直接支持的 `production_volume decrease` 或 `market_supply decrease`；产能缩减不能伪装成二者 | 第一财经 A 只作拆分和边界材料 |
| 政策 | D 语义接受、血缘缺口 | 明确 Product / ChainNode / Industry / IndustryChain 上的 `policy_support_intensity increase + stated_intent` | 经现有链路重新进入后可作 positive fixture |
| 企业财报 | B 接受，必须按公司和报告期拆分 | 海光信息 `net_profit increase + source_forecast`；摩尔线程 `revenue uncertain + source_forecast`（若 Evidence 无比较方向） | 经现有链路重新进入；聚合文章不能作为一个多公司 Event |
| 行业需求 | 当前缺口 | 明确对象、发布主体、期间和数值的 `market_demand increase + source_forecast/actual` | E 仅为条件替补；C 拒绝作为需求样例 |

额外边界：

- 已发布“存储器价格上涨”保留为 Golden negative fixture：即使表面可映射
  `market_price increase + actual`，`event_time = null` 且 Evidence 缺少产品、市场、
  幅度和时间时，不得进入正向语义结果；
- STMicro 涨价只能在 Product 明确时形成 `market_price increase`；“最高 15%”是
  上界，不能写成统一涨幅；
- TSMC 和 Samsung 的产能缩减必须按公司拆分，但当前目录无合法原生 Signal；
- “短缺”是供需缺口状态，不自动等于 `market_supply decrease` 或
  `market_demand increase`；
- 智能算力 2185 EFLOPS、同比增加 177% 和利用率 71.4% 是 capacity /
  capacity utilization 观测，不是 production 或 demand；
- 业绩预告即使来自公司正式公告也属于 `source_forecast`；正式财报才产生新的
  `actual` Signal；
- 政策发布是实际 Event，但其支持措施尚待执行，故 Signal 为 `stated_intent`；
- `capacity` 标记为 `DEFERRED_CANDIDATE`，`capacity_utilization` 标记为
  `DEFERRED_OBSERVATION_CANDIDATE`，不加入首批 12 项。

Golden positive fixture 采用“第一手来源优先；可信媒体明确引用可追溯的正式公告可作为
待复核替补、不可作为自动接受基准”的标准。Phase One 不生成 Theme/Reason Tree，因此
第一份 Theme 是否覆盖 Capacity 不属于本阶段验收或冻结条件。

#### 12.1.3 D-011 Golden 验收集合裁决

生产验收的最小正向集合固定为四条真实 Event，四类均不可用较弱样例替代；它们分别验证
不同的语义路径。若四条中没有任何一条 Evidence 明确陈述一跳因果，则另补第 5 条
`event_explicit` DirectImpact 正向 fixture。

| Golden 场景 | 必须验证 | 最小正向约束 | 当前状态 |
| --- | --- | --- | --- |
| 供给冲击 | `actual`、跨实体、approved Rule、`rule_inferred` Impact | 明确 Company/ChainNode 对具体 Product/Commodity 的停产、减产或恢复生产；`production_volume decrease` 经 `produces` 映射为 Target `market_supply decrease` | 正向缺口 |
| 政策 | `stated_intent`、政策主体与受影响主体分离 | 政府/监管机构正式发布，机构、措施、对象和生效/目标时间明确；Signal 为 `policy_support_intensity` 或 `regulatory_restriction_intensity`；DirectImpact 为空 | D 语义就绪，需经既有链路补齐血缘 |
| 企业财报/订单 | `source_forecast`、公司、区间 Measurement | 单公司、单报告期、声明时间有效；Evidence 逐字给出变量、区间和比较口径；例如海光 `net_profit increase` 同时保存绝对值与同比区间；DirectImpact 为空 | B 语义就绪，需拆分并重新进入既有链路 |
| 行业需求 | Product/Industry 需求语义，且不混同 capacity | 明确需求/销量/订单/出货的事实或具名主体预测、对象与期间；`market_demand` 或 `sales_volume` 的 actual/source_forecast | 条件缺口；E 仅在预测主体、期间和需求/出货口径完整时可用 |

供给冲击正向 fixture 必须覆盖已批准的
`production_volume decrease + produces -> market_supply decrease` Rule；Target 非空，关系、变量、
方向、实体类型和 Rule version 必须精确匹配。其他三类的最小正向 fixture 可以没有
DirectImpact，且公司利润 Signal 不能直接 Target Security。

负向 fixture 必须至少覆盖：已发布“存储器价格上涨”因时间缺失、Evidence 过薄和 Subject
不具体而不能 accepted；不可追溯原始主体的二手政策；将多公司、多报告期财报文章当作一个
Event；将智能算力规模/利用率伪装为 `market_demand increase`；以及自影响、利润直接影响
Security、Rule 未匹配或非 approved、affected variable 未定义、Evidence 不属于 Event、
预测缺主体/期间、区间被折成单值、`published_at` 被当作 `occurred_at`。

生产验收必须同时有至少一条 `rule_inferred` 和一条 `event_explicit` Accepted DirectImpact，
且每类至少一个关键负向/错误 Candidate 经自动审核正确阻断。不得为保持“四条”而伪造因果。

设计冻结与数据就绪分开：Entity、Variable、Signal、Measurement、Target、Impact、Rule、
自动审核和各类 Golden 条件可做**语义设计冻结**；但在四类真实正向 Golden、
`event_explicit` 覆盖及其反例实际运行前，不能最终冻结 Confidence 路由阈值、自动接受
准确率、Impact 误报率或宣布阶段一生产验收完成。

缺口记录使用 `ready | reingest_required | lineage_gap | missing_positive_fixture |
negative_only`。补齐路径仅限现有 Data/AgentRun 重新筛选、既有 Collector/Event Extractor/
Publication 链路重新进入，或用既有 Publication Contract 导入少量真实官方材料；这属于
Golden Data Preparation，不是新 Connector。只有证明现有能力无法取得必要材料时，才登记
独立的 Source Coverage Gap。

### 12.2 最高可观察 seam

```text
真实形态 Data Event + Evidence
→ 真实 Data Ontology / Entity API
→ 真实编译的 AgentRun typed workflow + Fake ChatModel
→ 严格结构化候选
→ 真实 Data publication API + 隔离 PostgreSQL
→ Data read API 读回
→ AgentRun consumer 解码与血缘断言
```

至少验证：

- 正常形成 Entity Link、Variable Signal 和 Direct Impact；
- 相同输入重复运行幂等；
- 非法 Variable Definition、Entity ID、Chain Node ID 和单位被拒绝；
- 低置信结果进入已确认的待复核边界；
- Event / Evidence / Agent Execution / TBox / Prompt / Model / rule version 可追踪；
- Provider OpenAPI、Consumer typed client 与冻结 fixture 一致；
- 一个 Data provider failure 的降级路径；
- 未知 publication 结果不重新调用模型，只重放相同候选；
- Data Service 可重新读取完整语义结果；
- 不生成 Theme / Reason Tree，不修改 Miniapp，不新增 Neo4j 投影。

### 12.3 隔离模拟链路

`agent-run/backend/cmd/server/event_semantics_synthetic_e2e_test.go` 提供显式开启的跨服务
验收，不进入普通单元测试默认路径。测试装置调用 Data-owned migration command 创建
临时 Data PostgreSQL，启动真实 Data HTTP 服务，并创建独立 AgentRun PostgreSQL；
AgentRun 仍只通过 Data v1 API 读取和发布，测试结束强制删除两个临时数据库。

运行条件：

```text
EVENT_SEMANTICS_SYNTHETIC_E2E=1
TIDEWISE_TEST_DATABASE_URL=<loopback Data PostgreSQL URL>
AGENTRUN_TEST_DATABASE_URL=<loopback AgentRun PostgreSQL URL>
go test ./agent-run/backend/cmd/server \
  -run '^TestSyntheticEventSemanticsEndToEnd$' -count=1 -v
```

固定覆盖：

- `20000000-0000-4000-8000-000000000002`：Generator 产生 Company Entity Link、
  `production_volume decrease + actual` Signal；Data 解析规范实体并以 approved
  `production_decrease_reduces_product_supply@1` 规则形成 Product
  `market_supply decrease` Direct Impact；独立 Reviewer pass 后 Data accepted；
- `21000000-0000-4000-8000-000000000002`：相同合法候选由 Reviewer 与一次
  Adjudicator 连续返回 indeterminate；Data 在 retry budget 耗尽后 quarantined，
  不允许进入下游；
- Data read API 读回 Evidence / Raw Document、Prompt / Model / Ontology / Policy、
  Review Snapshot 和 accepted record ID；
- Agent Status 为非工作态 `idle`，两条 Event Semantic Execution 历史状态均为
  `succeeded`；
- `research_themes` 与 `research_reasoning_trees` 在执行前后均为 0。

该模拟链路证明工程通路和分析师读取合同可用，但不替代 12.1 的真实 Event Golden
生产门禁，也不降低真实 Evidence、时间、血缘或 `event_explicit` 覆盖要求。

## 13. Eino Reference-First 记录

实现采用以下最窄参考审计：

| Repository | Commit | 检查文件 | 采用 / 拒绝 / 项目补足 |
| --- | --- | --- | --- |
| `cloudwego/eino` | `922b6a8a233b5233fe47eecee6cd2c005e8c39cd` | `compose/workflow.go`、`compose/checkpoint.go` | 采用 typed、all-predecessor、无循环 `compose.Workflow` 和显式 Compile；拒绝 checkpoint 作为业务恢复，因为 durable execution、幂等和审核状态属于 AgentRun/Data Service |
| `cloudwego/eino-ext` | `9137edd89e72b72735ede69db1c5ae29178a6e41` | `components/model/openai/go.mod`、`components/model/openai/chatmodel.go`、`components/model/openai/examples/structured/structured.go` | 复用仓库已固定的 OpenAI-compatible `v0.1.13` adapter 连接 DeepSeek，业务只依赖 `model.BaseChatModel`；该 Ext module 声明 Eino `v0.7.13`，Go MVS 由根 `go.mod` 统一选择 Eino `v0.9.12`，已通过 `go mod`、全仓编译和 fake-provider tests 验证兼容；不引入 DeepSeek 具体类型、Ext checkpoint 或 provider 配置到业务包 |
| `cloudwego/eino-examples` | `171220631fb7068ead50b7cd964b8c471647117d` | `compose/workflow/1_simple/main.go`、`compose/workflow/2_field_mapping/main.go`、`compose/batch/main.go` | 采用 typed workflow、字段映射和测试思路；拒绝 demo 的 `context.Background()`、全局 callback、batch/human interrupt、随机审核和进程内状态 |

最终实现使用 capability-local typed Workflow，Generator 与 Reviewer 为两次独立模型
调用；确定性实体解析、目标搜索、Data publication 与状态裁决仍在显式 application /
Data client 边界。Eino 没有提供本阶段所需的跨服务事实所有权、publication 幂等和
独立审核裁决，这些缺口分别由 AgentRun durable execution 与 Data Service versioned
API/transaction 补足。未采用 Graph、ADK、Batch、checkpoint 或模型可自由选择的 Data
mutation Tool。

## 14. 已确认约束

| ID | 决定 | 来源 |
| --- | --- | --- |
| C-001 | 阶段名为“阶段一——变量本体与 Event 语义落地” | 用户初始要求 |
| C-002 | Data Service 是 TBox、ABox、领域事实和 PostgreSQL 的唯一 Owner | 用户初始要求 |
| C-003 | AgentRun 只生成候选并拥有运行态，通过版本化 Data API 读写 | 用户初始要求 |
| C-004 | Agent 不直接访问 PostgreSQL 或 Neo4j | 用户初始要求 |
| C-005 | Neo4j 本阶段不新增 Event / Variable 相关投影 | 用户初始要求 |
| C-006 | Miniapp 本阶段不改动 | 用户初始要求 |
| C-007 | 本阶段不生成 Theme / Reason Tree | 用户初始要求 |
| C-008 | 未明确确认设计冻结前禁止实施 | 用户初始要求 |
| D-001 | Event 原生 Variable Signal 允许 `actual`、`stated_intent`、`source_forecast`，但禁止 Agent 自行预测 | 事件推理模型确认 |
| C-009 | 研究语义、变量范围和真实 Event 样例优先由事件推理模型回答；只向用户确认工程问题及模型无法回答的问题 | 用户确认 |
| D-002 | 采用最小推理实体集；Technology/Policy/Region 延后且不错误映射；阶段一新增一等 Product，但不自动重分类或批量迁移现有 Entity | 事件推理模型建议，用户确认 Product 工程范围 |
| D-003 | 第一批目录维持 12 项；不加入 `capacity` / `capacity_utilization`；已明确四类 Golden 的首选、替补、拒绝与缺口 | 事件推理模型基于只读真实数据裁决 |
| D-004 | Direct Target 只允许跨实体一跳：默认 Commodity/Product/ChainNode/Company，Industry 有条件允许；禁止自影响，IndustryChain/Security/Sector/Concept 延后 | 事件推理模型回答并经 subject/target 消歧 |
| D-005 | Direct Impact 是一跳分析断言而非 Event 原生事实；必须声明 Target 上的受控变量和方向；Data 保存候选与审核结果，仅 Accepted 可供下游使用；阶段一包含 3–5 条人工批准的版本化 DirectTransmissionRule，不建设完整规则引擎 | 事件推理模型回答，用户确认 Rule 范围 |
| D-006 | Entity Link、Signal、Impact 独立审核并传播依赖状态；Data 确定性门禁、独立 AI Reviewer 与版本化 Acceptance Policy 共同裁决；无人工 UI，Pending 自动复核，预算耗尽后 quarantine；Confidence 与 Evidence Grade 仅作路由特征 | 事件推理模型回答并依据现有 Evidence Grade 语义修订；用户确认无人工审核范围 |
| D-007 | Phase One 不建独立 Observation；VariableSignal 携带可复用的受控 MeasurementValue 数组，支持绝对值/变化值、区间、原始与规范单位及业务时间；Phase Two 再无损迁移 | 事件推理模型回答 |
| D-008 | 一个 AgentRun Agent Execution 只分析一个 Data Event，并一对一映射 Data-owned EventSemanticSubmission；Data 保存提交/审核/版本/产物血缘，不复制 AgentRun runtime 状态与错误；重新分析创建新 Work Item、Execution 和 Submission 并 supersede 旧 Submission | 用户确认 |
| D-009 | 原位演进现有 event_entity_links，不建立第二套 canonical Link 或双写；新语义行绑定 Submission/Evidence 并支持自动审核与 supersede；其他环境的旧行保留为 legacy provenance | 用户确认 |
| C-012 | Phase One 实现面向全部 Agent 的只读 Agent Status Monitor；AgentRun 提供当前是否工作与当前执行状态的数据接口，Admin Portal 展示；不返回执行详情，不建设人工审核 UI | 用户确认并收紧范围 |
| D-010A | Data 语义提交采用两阶段合同：Data 先持久化 Candidate 并确定性预检，AgentRun 再独立复核，Data 按 Review Result 与 Acceptance Policy 最终裁决 | 用户确认 |
| D-010B | 一个 EventSemanticSubmission 只处理一个 Event；每次 Candidate / Review 提交原子保存完整快照，但 Candidate 按项独立裁决，单项失败不回滚其他项 | 用户确认 |
| D-010C | `agent_execution_id` 是创建 Data Submission 的唯一幂等身份；相同 canonical payload 可安全重放，不同 payload 返回 409；重新分析必须创建新 Work Item、Execution 和 Submission 并 supersede 旧 Submission | 用户确认 |
| D-010D | AgentRun 持久化并领取 Event Semantic Work Item、管理执行租约和有限重试；Data Context Lease 只固定 Event/Ontology 快照和提交边界，不承担任务机制 | 用户再次确认架构边界 |
| D-010E | 最小 API 沿用 Data v1、OpenAPI、Bearer scope、1 MiB 请求上限和有界 timeout；以 Eligible Event、Context Lease、Context、Entity Resolution、Target Search、Submission、Review、Read 八项能力闭环；显式重分析进入 AgentRun 内部 API | 用户授权，按既有工程事实冻结 |
| D-011 | Golden 最小生产验收为供给冲击、政策、企业财报/订单、行业需求四条真实正向 fixture；供给冲击覆盖 rule_inferred，另需至少一条 event_explicit；每类含关键负向。当前缺口按 Golden Data Preparation 处理，不混入 Connector 开发 | 事件推理模型裁决 |
| C-010 | Golden Scenario 优先从现有 Data/AgentRun 真实 Event 中只读发现；不限制 48 小时，不把新 Connector 混入阶段一 | 事件推理模型补充边界 |
| C-011 | 只读发现确认现有 8 条 Data Event 血缘完整但不足以覆盖四类 Golden Scenario；语义较强但未发布的 Artifact 只能作为候选，不能冒充合格 Event | 本地 Data、AgentRun 与 Artifact 只读检查 |

## 15. Grill 决策队列

每次只确认一个会实质改变模型或边界的问题；确认后立即更新本文，必要时同步 Context
术语或提出 ADR。

| 顺序 | ID | 问题 | 状态 |
| ---: | --- | --- | --- |
| 1 | D-001 | Variable Signal 是仅观测，还是也允许 Event 支撑的预测/指引/计划？ | 已确认 |
| 2 | D-002 | 第一批正式 Entity Type，以及 Product/Technology/Policy/Region 的处理 | 已确认 |
| 3 | D-003 | 用真实 Event 裁剪第一批 Variable Definition | 已确认；12 项维持，四类 fixture 尚有缺口 |
| 4 | D-004 | Direct Target 允许哪些 Entity Type | 已确认 |
| 5 | D-005 | Direct Impact 的对象性质，以及是否包含最小一跳 Transmission Rule 注册表 | 已确认 |
| 6 | D-006 | 自动接受阈值、自动 AI 复核和低置信结果边界 | 已确认 |
| 7 | D-007 | Observation 本阶段建表还是延后 | 已确认；Observation 延后，保留 MeasurementValue |
| 8 | D-008 | Data 血缘对象与 Agent Execution 的映射及命名 | 已确认 |
| 9 | D-009 | `event_entity_links` 的兼容迁移方式 | 已确认 |
| 9a | C-012 | AgentRun 执行监控机制与 Admin Portal 可见性 | 已确认 |
| 10 | D-010 | API 复用/新增、DTO、认证、timeout、幂等和部分失败 | 已确认；字段级合同由 Issue #138 的 provider-consumer fixture 固化 |
| 11 | D-011 | 第一批真实 Event 与最终验收 fixture | 已确认；生产验收保留数据就绪门禁 |
| 12 | D-012 | 设计冻结与实施授权 | 已确认；用户通过 `$to-spec` 要求交接实施 |

## 16. 设计冻结门禁

以下冻结门禁已经由本文、所属 Context 与 Issue #138 的实施规格共同满足：

- MVP Entity Type 与 Variable Definition 清单明确；
- Signal、Impact、Observation 和 Submission 的术语与事实/候选边界明确；
- 对象身份、版本、时间、状态、审核和替换语义明确；
- Data / AgentRun Owner 与 API 调用方向明确；
- Provider OpenAPI 与 Consumer fixture 已冻结到字段级；
- 认证、timeout、retry、idempotency、partial failure 和 safe error 明确；
- migration、兼容窗口、rollout 与 rollback 明确；
- 真实 Event Golden Scenario 和最高可观察验收 seam 明确；
- 用户明确确认“设计冻结”。

设计已冻结并获得实施授权。实现不得通过降低 Evidence、时间、血缘或自动审核门槛来
绕过尚未完成的 Golden Data Preparation。
