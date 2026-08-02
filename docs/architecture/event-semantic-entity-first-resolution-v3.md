# Event Semantic Entity-first Resolution V3

状态：Accepted，已由用户冻结并进入 Issue #164 实施

日期：2026-08-02

Owners：Data Service（正式 Entity/TBox、Context 与 Submission 合同、PG→Qdrant 投影）、
AgentRun（Event Semantic Execution、Eino Workflow、Qdrant 查询、LLM 提取/选择/审核）

本 Spec 在接受后取代 `event-semantic-qdrant-retrieval-v2.md` 中以下内容：

- Generator 同时生成 Mention、预测 Entity Type、VariableSignal 与 Measurement；
- 按 LLM `predicted_entity_type` 对 Qdrant exact/vector retrieval 做硬过滤；
- 在 Entity Resolution 前校验预测类型与类型相关 Role；
- 任一候选第二次不合法即终止整个 Event 且不提交其他合法候选；
- EventEntityLink Mention 必须逐字出现在每一条所引 Evidence 中。

V2 的其他边界继续有效：PostgreSQL 是正式事实源；Qdrant 是可重建召回投影；Data 拥有
PG→Qdrant 写入，AgentRun 直接查询 Qdrant；Event Semantic 只产生 EventEntityLink、
VariableSignal 与 Measurement，不产生 DirectImpact、Theme、Reason Tree 或投资结论。

## 1. Problem Statement

当前 V2 在首次模型调用中要求 LLM 同时完成四项工作：提取原文 Mention、预测正式
Entity Type、分配 Event Role、选择 Variable Definition 并生成 Signal。随后 exact 与 vector
retrieval 都使用预测类型作为 Qdrant 硬过滤条件。只要类型预测错误，正式 Entity 即使存在于
Qdrant，也不会进入候选集，后续 selector 无法纠正。

当前 TBox 只提供 Entity Type key、版本、Signal Subject 权限和 Allowed Event Roles，不包含
中英文名称、业务定义、纳入/排除边界。模型能够看到 `company`、`policy_body`、`economy` 等
枚举，却没有正式定义可用于区分公司、品牌、证券、政策机构、国家、经济体和市场。

固定约 100 Event 验收还暴露出批次级误杀：类型、Role、Evidence literal、Selector coverage 或
Signal key 中任一候选第二次不合法，整个 Event 即终止。EventEntityLink 本身允许独立于
VariableSignal 存在，但现有前置校验没有把这种独立性贯彻到 Workflow 失败语义。

## 2. Outcome

把 Event Semantic 改为 entity-first 的分阶段客观语义流程：

```text
Event + Evidence
      ↓
提取原文 Mention
      ↓
Qdrant 跨类型 exact / vector recall
      ↓
LLM 从正式候选选择 Entity ID 或 no_match
      ↓
正式 Entity 决定 Entity Type；LLM 按该类型分配 Event Role
      ↓
按正式 Entity Type 筛选 Variable Definition
      ↓
LLM 生成 Event-native VariableSignal 与 Measurement
      ↓
候选级确定性校验、独立 AI Review、Data 正式发布
```

目标不是提高错误绑定容忍度，而是把事实门禁放在正确阶段：LLM 不再创造或预测正式类型；
被选择的正式 Entity 自带类型，Data 最终按 PostgreSQL 复核；错误隔离在候选级，不再让一个
非法候选删除同一 Event 中其他合法事实。

## 3. Domain Language

### Raw Mention

Event title/summary 或 Event Evidence title/excerpt 中实际出现的连续原文片段。Raw Mention
只表达“原文写了什么”，不拥有正式 Entity ID 或 Entity Type。

### Entity Candidate

Qdrant 从正式 Entity 投影返回的候选对象，携带 Entity ID、Entity Type、名称、规范名、别名、
简短描述和相似度分数。它是召回结果，不是 accepted 事实。

### Resolved Entity

LLM 从某一 Raw Mention 紧邻的候选集中选择的正式 Entity。Workflow 使用候选携带的
Entity Type；Data 在 Submission 时以 PostgreSQL 中同一 Entity ID 的类型和状态作最终复核。

### Entity Type Definition

Data 拥有并版本化的 TBox 对象，定义一种正式 Entity 类型的业务含义、纳入/排除边界、是否
允许建立 EventEntityLink、是否允许作为 Signal Subject 以及可承担的 Event Roles。它不是
Entity 实例、Prompt 自由词汇或模型预测结果。

### EventEntityLink

一个正式 Event 与一个 Resolved Entity 之间的客观关联，保留 Raw Mention、Event Role、
Evidence lineage 和 resolution method。它可以在没有 VariableSignal 时独立成立。

### VariableSignal

Event 对一个已成立 EventEntityLink 所指 Entity 作出的 Event-native 受控变量变化陈述。
Variable Definition 的选择必须发生在 Entity Resolution 之后，并受该正式 Entity Type 的
适用范围约束。

## 4. Ownership And Authority

| 事实或行为 | Owner | 权威规则 |
| --- | --- | --- |
| Entity、Entity Type、Variable Definition | Data / PostgreSQL | PostgreSQL 是最终事实源 |
| PG→Qdrant projection | Data | 只投影 active/current 正式数据 |
| Qdrant query 与候选集 | AgentRun | Qdrant 只召回，不拥有 accepted 状态 |
| Mention 提取、候选选择、Signal 提取 | AgentRun | 只能使用当前 Event/Evidence、候选集和 pinned TBox |
| EventEntityLink/VariableSignal 发布 | Data | Submission precheck + AI Review 后持久化 |
| Work Item、Execution、重试 | AgentRun | Data Context Lease 不是任务队列 |

AgentRun 不连接 Data PostgreSQL。Data Service 不代理 Qdrant 查询，也不依赖 Eino。

## 5. Entity Type TBox Contract

### 5.1 Required fields

`entity_type_definitions` 在现有字段基础上增加：

| 字段 | 语义 |
| --- | --- |
| `name_zh` | Entity Type 中文正式名称 |
| `name_en` | Entity Type 英文正式名称 |
| `business_definition` | 一到两句稳定业务定义 |
| `inclusion_criteria` | 属于该类型的受控边界条目 |
| `exclusion_criteria` | 容易混淆但不属于该类型的边界条目 |
| `event_link_allowed` | 该类型是否允许成为 EventEntityLink target |

以下现有字段继续保留：`type_key`、`version`、`signal_subject_allowed`、
`allowed_event_roles`、`status`。`direct_target_mode` 仅作为历史 DirectImpact 合同保留，不参与
V3 Event Semantic。

### 5.2 Definition requirements

- active Entity Type 必须具备非空中英文名、业务定义和明确的纳入/排除边界；
- `event_link_allowed` 与 `signal_subject_allowed` 分开：允许与 Event 建链不代表允许产生 Signal；
- Allowed Event Role 必须来自正式 Role 词汇；
- Context Lease 固定具体 Type key/version；AgentRun 不允许自行补充、改写或创造类型；
- “国家是否建立独立 `country` 类型，还是使用正式目录中现有 `economy` 类型”由 TBox catalog
  决定；V3 Workflow 只使用被选正式候选携带的类型，不自行推断或改写。

### 5.3 Example boundaries

以下只规定定义形式，不在本 Spec 直接发布新的 catalog row：

```text
company / 公司
定义：具有独立经营身份的企业、集团或法人经营主体。
包含：上市公司、非上市公司、集团公司、可识别经营主体。
排除：公司发行的证券、品牌、产品、产业链节点、政府机构。

economy / 经济体
定义：用于宏观经济分析和统计的国家、地区或经济区域。
包含：国家经济体、欧元区等跨国家经济区域。
排除：政府部门、中央银行、证券市场、企业。

country / 国家（若后续正式创建）
定义：具有主权和政治法律身份的国家。
包含：正式主权国家。
排除：经济区域、政府部门、中央银行、市场和企业。
```

### 5.4 Existing definition content backfill

这不是一个长期系统功能。本次实施 Agent 在执行 V3 任务时，直接读取数据库和现有 migration
中的全部 Entity Type Definition，为每个现有 active `(type_key, version)` 编写并随本次变更
交付以下完整内容：

- `name_zh`；
- `name_en`；
- `business_definition`；
- `inclusion_criteria`；
- `exclusion_criteria`；
- `event_link_allowed`。

这些内容通过本次 forward migration/seed 一次性写入现有 Entity Type Definition row。实施完成
时不允许存在未补充的 active 类型、空字符串、通用占位文本或只把 type key 机械翻译成定义。
本次不建设 TBox 生成 Agent、Curator 服务、运行时审核、后台管理页面或持续同步机制。

新增字段是正式、长期的 Entity Type Definition 领域合同，因此本次必须同步扩展 Data 的领域
模型、Biz 校验、PostgreSQL Data Layer 查询/扫描/写入、Context 水合与 fingerprint、OpenAPI、
provider-consumer fixture、PG→Qdrant 投影范围和 AgentRun consumer。只有“为当前 local PG
中既有 active `(type_key, version)` 编写字段内容”属于实施时的一次性人工数据回填，不得把
这项内容生产扩展成 LLM 生成、Curator、CRUD、后台页面、运行时补全或持续同步功能。

## 6. Target Workflow

### 6.1 Discover and claim Event

AgentRun 继续通过 Data eligible scan 发现待处理 Event，并在 AgentRun 创建/领取 Event Semantic
Work Item。generation、disposition、keyset pagination、retry 与 terminal isolation 继续使用现有
所有权边界，本 Spec 不把任务队列迁移到 Data。

### 6.2 Create Context Lease and hydrate Context

Context Lease 继续固定：

- Event 与 Evidence identity/fingerprint；
- Entity Type Definition key/version；
- Variable Definition key/version；
- assertion modalities、Measurement contract、Ontology/Policy version；
- Agent Execution、Worker 和 lease expiration。

Context API 仍可返回完整的小型 active/current Entity Type 与 Variable Definition 目录给
AgentRun，但 AgentRun 必须按 Workflow 阶段选择 Prompt 输入；“Context 中存在”不等于“全部
进入第一次模型调用”。Manifest 不保存完整 TBox、ABox 或 EntityRelation。

### 6.3 Stage A — Extract Raw Mentions

模型输入只包含：

- Event；
- Event Evidence；
- Raw Mention 与 Evidence 引用规则；
- 固定 JSON Schema。

不得输入完整 Entity Type 或 Variable Definition 目录。输出：

```json
{
  "mentions": [
    {
      "candidate_key": "mention_1",
      "mention": "安靠科技",
      "evidence_ids": ["evidence-uuid"]
    }
  ]
}
```

Stage A 不输出 `predicted_entity_type`、`entity_role`、Entity ID、VariableSignal 或 Measurement。
明确实体专名即使位于将来时、公告、报告或状态陈述中仍应提取，但必须收窄为原文中的实体 span：
“日本央行将公布决议”提取“日本央行”，“多数美联储委员”提取“美联储”，“央行报告”提取
“央行”；不得把整段陈述、报告名或状态当成 Mention。

### 6.4 Stage B — Cross-type Qdrant retrieval

AgentRun 对一个 Event 的所有合法 Mention 执行：

1. Event-batched normalized name/alias/UUID exact lookup；
2. 对未唯一命中的 Mention 调用一次 `EmbedStrings`；
3. 在同一次 Qdrant query batch 中执行跨类型 vector recall；
4. 每个 Mention 返回有界候选集。

Exact 与 vector retrieval 都不得以模型预测类型作硬过滤。未来可以增加不影响召回完整性的
`type_hint` 作为排序提示，但它不能排除其他正式类型。Vector Top-K 是可配置 retrieval policy，
初始建议 10、上限 20，最终值用固定 100 Event 标注样本校准，不作为领域常量。

如果 exact lookup 只返回一个 active/current、normalized name 唯一的正式候选，可以直接进入
Resolved Entity；同名跨类型或多个 exact 候选必须进入 Stage C。

### 6.5 Stage C — Select Entity and Event Role

模型输入：

- Event 与 Evidence；
- Raw Mention；
- 该 Mention 的紧邻 Entity Candidates；
- 候选集中实际出现的少量 Entity Type Definitions；
- 这些类型允许的 Event Roles；
- 固定 selector JSON Schema。

模型输出：

```json
{
  "selections": [
    {
      "candidate_key": "mention_1",
      "entity_id": "entity-uuid",
      "entity_role": "actor",
      "no_match": false
    }
  ]
}
```

模型不输出 Entity Type。Workflow 从被选候选读取 projected Entity Type，并在 Submission 时由
Data 以 PostgreSQL 复核。Selector 采用“召回优先、Review 兜底”：只要 vector 候选中存在与
Mention 合理表示同一业务对象的候选，就选择最合理候选进入独立 Review；只有全部候选明显不是
同一对象时才 `no_match`。省略“系统、服务、设备、产品”等通用后缀属于允许审核的规范化差异，
例如“非侵入式脑机接口”可以选择“非侵入式脑机接口系统”。相似名称、同类型、同行业、同产业链
或背景知识仍不能替代对象同一性。

唯一 canonical name/alias exact 候选继续作为确定性 identity 解析结果，不允许 Selector 以
`no_match` 静默删除；模型只需为其分配允许 Role，最终仍进入独立 Review。Vector Top-1 不能绕过
Review 直接成为正式事实。对于 exact identity 被模型拒绝，或明显规范化等价候选仍被 primary
Selector 判为 `no_match` 的情况，Workflow 必须使用独立 Reviewer 执行一次有界二次选择复核并
记录初次与复核结论，避免单点误杀。

`mention_not_entity` 仅允许日期、数值、状态、行为、报告、会议等真正非实体 Mention。真实公司、
产品、技术、指数或其他实体指称即使 ABox 缺失或当前 TBox out of scope，也必须使用对应的
ABox/TBox/identity gap 分类，不得伪装成 Stage A 非实体错误。Role 必须区分 `statement_source`、
`actor` 与 `context`；“某人称……”中的发言主体不能与被谈论对象机械地都标为 `actor`。

### 6.6 Stage D — Filter Variable Definitions

AgentRun 以 Resolved Entity 的正式 Entity Type，在 pinned complete Variable Definition directory
中确定性筛选 applicable definitions。每个 Resolved Entity 的 Signal prompt 只获得适用于其类型
的 definitions；Qdrant variable collection 如保留，只能排序，不能删除目录中的合法定义。

### 6.7 Stage E — Generate VariableSignal and Measurement

模型输入：

- Event 与 Evidence；
- 已解析 EventEntityLinks；
- 对各 EntityLink 适用的 Variable Definitions；
- assertion modalities；
- Measurement narrative contract；
- 固定 Signal JSON Schema。

模型只生成 Event 对已解析 Entity 的 Event-native 客观陈述。允许一个 EventEntityLink 没有
VariableSignal，允许一个 Signal 没有 Measurement，也允许一个 Signal 携带多个 Measurement。
不得生成 DirectImpact、跨实体传导、analyst inference、Theme 或投资判断。

### 6.8 Stage F — Submit, review and finalize

AgentRun 提交合法 EventEntityLink、VariableSignal 和 Measurement。Data 按 pinned Context、
PostgreSQL Entity/TBox/Evidence 做确定性 precheck；AI Reviewer 按候选独立审核对象同一性、
Event-native Signal 和 Measurement Evidence fidelity。accepted/latest/non-superseded 结果才供下游
Theme Analyst 使用。

对象同一性不在 AgentRun 代码中维护手写简称、职衔、国别前缀或证券后缀规则。唯一正式
canonical/alias exact identity 由投影确定性解析；其他候选由 Selector 提议并由独立 AI Reviewer
仅审核“是否为同一个业务对象”。相似、上下级、隶属或业务相关不能替代 identity，例如
“巴西总统卢拉→冯德莱恩”“欧元区→欧盟”“中国国际进口博览局→商务部”以及
“Robotaxi→自动驾驶系统”必须 fail。合法简称应进入正式 Entity alias 数据治理；不得为无法穷举
的语言现象在 Workflow 中建设第二套 TBox 或字符串启发式门禁。

## 7. Validation Policy

### 7.1 Mention grounding

一个 Mention 合法，当且仅当：

- `candidate_key` 非空且在 Stage A 输出中唯一；
- `mention` 是 Event title/summary 或至少一条所引 Evidence title/excerpt 中出现的连续原文片段；
- `evidence_ids` 非空、唯一且都属于当前 Event；
- 如果 Mention 只出现在 Event title/summary，至少引用一条该 Event 的 primary supporting Evidence；
- 不要求每一条所引 Evidence 都逐字包含 Mention。

Entity 别名、简称、跨语言名称和类别规范化由 Stage C 的对象同一性判断负责；它们不能被用来
伪造一个原文中不存在的 Raw Mention。

### 7.2 Entity selection

- 被选 `entity_id` 必须属于该 Mention 紧邻的 exact/vector candidate set；
- 不得使用其他 Mention、之前 retry 或其他 Event 的候选；
- candidate 必须具有合法 Entity ID、Entity Type、active status 和 projection identity；
- AgentRun 必须逐 point 校验 Qdrant 外层 `point id == payload.entity_id`，并校验审计元数据
  `source_identity`、`projection_version`、`embedding_model` 与非空 SHA-256
  `content_fingerprint`；旧版本、错误模型、异源、不完整或身份不一致 payload 一律以 retrieval
  contract failure fail closed，不得进入 Selector。payload 不重复保存 `point_id`；
- candidate Entity Type 必须在 pinned TBox 中 active 且 `event_link_allowed=true`；
- `entity_role` 必须被被选 Entity Type 的 Allowed Event Roles 接受；
- Data 必须验证 Qdrant projected Entity ID/type 与 PostgreSQL 正式 Entity ID/type 一致；
- Similarity score 只用于排序，不构成自动绑定阈值。

### 7.3 Signal and Measurement

- Signal Subject 必须引用本次已成立 EventEntityLink；
- Variable key/version 必须来自 pinned directory，并适用于 PG 正式 Entity Type；
- direction 与 assertion modality 必须属于正式受控值；
- Signal 必须是 Event-native 陈述，不能把关系推导或投资判断包装成 Signal；
- Measurement 继续只做非空/长度/Evidence referential integrity 的确定性校验；
- AI Review 校验 Measurement 整句含义及限定条件；失败拒绝父 Signal，不撤销 EntityLink。

### 7.4 Candidate-level isolation

- 单个非法 Mention 只隔离该 Mention 及其下游候选；
- exact/vector 无候选或 selector 无法判断时，该 Mention 变为 unresolved/no_match；
- selector 少返回一个 candidate_key 时，缺失项确定性视为 unresolved，不终止其他完整选择；
- 单个非法 Entity selection 只拒绝该 EventEntityLink；
- 单个非法 Signal 只拒绝该 Signal；
- Measurement 失败只拒绝父 Signal；
- EventEntityLink 不依赖 VariableSignal 存在；一个只有合法 EntityLink 的 Submission 是有效结果。

每个模型阶段的顶层 JSON envelope 是 Execution 级合同。`mention_extraction` 必须显式包含
`mentions` 数组、`entity_selection` 必须显式包含 `selections` 数组、`signal_extraction` 必须显式
包含 `variable_signals` 数组、Review 必须显式包含 `items` 数组。数组可以合法为空，但顶层
`null`、`{}`、缺字段、字段为 `null` 或错误类型均不合法；不得因 Go 零值把这些输出静默解释为
零 Mention、no_match、空 Signal 或全拒绝。首次非法只允许一次 bounded repair，repair 后仍非法
才以 terminal `model-contract failure` 结束。

只有以下情形可以终止整个 Execution：

- Stage 输出在一次 bounded repair 后仍不是可解析的固定 JSON envelope；
- Context Lease/Context contract 无效或发生不可恢复 drift；
- Qdrant、Embedding 或 Data 调用发生不可恢复的 transport/contract failure；
- Submission 未知结果无法通过既有 reconciliation 恢复。

## 8. Contract And Persistence Changes

### Data Service

- 使用 forward migration 扩展 `entity_type_definitions`；不修改已进入共享历史的 migration；
- 实施 Agent 在本次任务中直接为全部现有 active Entity Type 编写新增字段内容，并通过本次
  forward migration/seed 一次性回填；
- Backfill 完成后收紧新增必填约束，不增加长期生成或管理机制；
- Context contract 增加 Entity Type 语义定义与 `event_link_allowed`；
- Context manifest 继续只保存 key/version/fingerprint，不复制完整定义；
- Submission API 不增加 predicted type；既有 EntityLink、Signal、Measurement 物理模型足以承接 V3；
- Data precheck 从 PostgreSQL Entity 取得最终类型，并按该类型校验 Role 与 Variable applicability；
- Mention grounding 与 candidate-level decision 对齐本 Spec。

### AgentRun

- Agent Version 升级为 V3；拆分 Mention、Entity Selection、Signal Generation 三个模型阶段；
- Mention DTO 删除 `predicted_entity_type` 与前置 `entity_role`；
- EntityLookup/Qdrant Port 删除强制 predicted type filter；
- selector 输出增加选中 Entity 的 `entity_role`，但不输出 Entity Type；
- Signal Stage 在 Entity Resolution 后运行，只接收 applicable Variable Definitions；
- 原生候选与 selector 校验改为 candidate-level isolation；
- 保留 typed、acyclic Eino Workflow，不引入开放式 Tool loop 或 Data PG 访问。
- 进程启动配置必须在 worker 领取 Work Item、创建 Lease/Context 前验证非空
  `EMBEDDING_API_KEY`；运行到 Semantic Runtime 构造时才暴露缺密钥不符合启动合同。

### Qdrant projection

- `entity_semantic_v1` point identity、vector size、embedding model、正式 Entity payload 与 ownership 不变；
- retrieval query 取消 Entity Type hard filter；
- Entity Type Definition 不需要重复嵌入每个 Entity point；selector 从 pinned Context 按候选类型取得定义；
- 如果 projector 当前只投影旧 V2 allowlist，需要在同一变更中按 `event_link_allowed` 调整投影范围并重建 collection；
- 增量同步、CDC 和生产自动重建仍不在本次范围。

### Versioning

- Context wire/manifest contract 升级为 `event-semantic-context-manifest.v3`；
- Agent prompt/workflow/schema hashes 全部升级；
- 更新 provider-consumer fixture、OpenAPI、Data/AgentRun Context 与相关 ADR；
- V2 Work Item/Submission 历史只读保留，不原地解释成 V3。

## 9. Observability And Audit

AgentRun 必须为每个 Event 保存有界、脱敏、可审计的阶段摘要：

- Raw Mention、Evidence IDs；
- exact/vector candidate 的 Entity ID、Entity Type、名称与 score；
- selector 的 selected ID/no_match/Role；
- 每个 Resolved Entity 实际可用的 Variable key/version；
- Stage 初次与 repair 后的 violation codes；
- candidate isolation/rejection reason 与 owner classification。

不得保存 Secret、Authorization、完整 Prompt、完整模型原始响应或 RawDocument 正文。审计摘要必须
足以区分 ABox/TBox gap、retrieval miss、model selection、validation isolation 和 transport failure。

Owner/reason 采用可审计的运行时分类：无可用 Qdrant 候选为 `entity_no_candidates /
qdrant_projection`，候选因正式类型目录被全部排除为
`entity_candidates_not_event_link_allowed / tbox`；selector 在存在 exact identity 候选时仍选择
`no_match` 为 `selector_rejected_exact_candidates / model_selection`。Selector 的受控
`no_match_reason` 进一步区分 Stage A 抽取了非实体（`stage_a_non_entity_mention /
model_extraction`）、候选中没有同一对象（`identity_projection_gap / abox_or_retrieval`）和上下文
不足（`selector_insufficient_context / model_selection`）。`identity_projection_gap` 只表示当前
canonical/alias identity projection 与 Top-K 候选无覆盖，验收时再用 PG 人工核验区分“正式
Entity 缺失”和“正式 Entity 存在但 alias/retrieval 未覆盖”；AgentRun 不得为完成该分类而直连
Data PostgreSQL。非法字段、Evidence 或 candidate selection 继续归 validation/model isolation，
transport failure 记录为 Execution failure。

固定样本验收报告不得继续输出合并后的 `abox_or_retrieval` 或把真实实体归为
`stage_a_non_entity`。报告层必须以 PostgreSQL、pinned TBox、Qdrant 候选与 Stage audit 交叉核验，
最终使用 `correct_reject`、`abox_missing`、`tbox_out_of_scope`、`mention_extraction_miss`、
`retrieval_miss`、`selector_false_reject`、`review_reject`、`model_contract_failure` 八类。AgentRun
运行时不能访问 PostgreSQL，因此运行时保留安全的原始原因，验收工具/报告在 Service 边界外完成
最终 owner 拆分。

## 10. Acceptance

### Deterministic contract fixtures

- 全部现有 active `(type_key, version)` 都必须具有本次直接生成的完整扩展字段，不得存在空定义
  或通用占位文本；
- Mention extraction schema 不含 predicted Entity Type、Role、Signal 或 Measurement；
- exact 与 vector request 不包含 Entity Type hard filter；
- 同一 Mention 的跨类型候选可以进入 selector；
- selector 只能选择当前候选 ID 或 no_match；最终类型等于候选类型并与 PG 一致；
- selector Role 按被选 Entity 的正式类型校验；
- Signal prompt 只包含 resolved Entity Type applicable definitions；
- 没有 Signal 的合法 EventEntityLink 可以提交、审核并 accepted；
- 一个非法 Mention/Signal 不阻止同一 Event 的其他合法候选发布；
- exact identity 只来自正式 canonical/alias；其他候选必须经过独立 AI Review，AgentRun 不维护
  无法穷举的简称/全称字符串规则；
- `null`、`{}`、缺少必填顶层数组和数组错误类型经过一次 repair 后仍会 terminal；显式合法空数组
  保持成功；
- 外层 point ID 与 payload Entity ID 不一致、错 projection version、embedding model、source
  identity 或缺少
  content fingerprint 的 Qdrant point 均不得进入 Selector；
- AgentRun 缺少 `EMBEDDING_API_KEY` 时在启动配置阶段失败；
- Event title/summary 中的 Mention 可以通过 primary Evidence lineage 建链；不再要求所有 Evidence
  都逐字包含 Mention；
- Event Semantic V3 仍产生零 DirectImpact、零 Theme、零投资结论。

### Fixed sample acceptance

复用同一批约 100 Event 和已固定 NVIDIA/Amkor Event，报告：

- Mention 完整率、unresolved/no_match、exact/vector recall、跨类型 candidate distribution；
- correct Entity ID、wrong Entity binding、type-from-candidate/PG mismatch；
- EventEntityLink-only 成功数、Signal/Measurement 成功数；
- candidate isolation 数与 whole-Execution terminal 数；
- ABox/TBox gap、retrieval miss、model selection、validation 与 transport 分类；
- Prompt/context bytes、模型调用次数/延迟、Qdrant batch 数与 p50/p95；
- 与 V2 的 16 accepted / 57 rejected / 27 model-contract failed 基线逐项对比。
- 对每个 rejected Event 给出八类最终归因，并单独给出 accepted EventEntityLink 的对象精度与
  Role 问题；显式核验 DirectImpact=0、Direct Target/Transmission Rule 调用=0，并以 V3 DTO、
  Prompt、Workflow/调用审计中不存在 Theme/Reason Tree 路径作为确定性越界审计证据。

验收必须证明：正式候选不会再因错误 predicted type 被搜索前排除；一个候选错误不会删除同一
Event 中可接受的其他 EventEntityLink；错误绑定率不能因跨类型 recall 而上升。

## 11. Testing Decisions

- Data migration 测试覆盖 forward chain、现有 active 类型 backfill、约束与历史兼容；
- Data backfill 测试覆盖全部 active type 均已补齐、字段非空和约束收紧；
- Data API/fixture 测试覆盖 V3 Entity Type Definition wire contract；
- Data Biz 测试覆盖 PG type authority、event_link_allowed、Role、Mention grounding、link-only Submission；
- AgentRun compiled Workflow 测试覆盖三阶段 Prompt、跨类型 exact/vector recall、selector、按类型变量过滤；
- candidate isolation 测试覆盖非法 Mention、缺失 selection、非法 Signal 和 Measurement fail；
- Qdrant adapter 使用真实 Qdrant 验证无类型过滤的一次 Event-batched exact + vector query；
- Stage envelope 合同测试覆盖 `null`、`{}`、缺字段、错误类型、合法空数组与 repair exhaustion；
- Qdrant adapter 合同测试覆盖 projection identity/version/model/fingerprint fail closed；
- AgentRun configuration 测试覆盖缺少 Embedding 密钥时的启动拒绝；
- Data confidence 校验与 OpenAPI decimal string pattern 保持同一合同，拒绝 `.5`、`1e-3` 等格式；
- provider-consumer、OpenAPI、configuration、architecture、cross-service E2E 与固定样本验收全部通过；
- 最终运行 gofmt、go vet、受影响 Go suites、binary build、migration chain、Standards/Spec code review。

## 12. Rollout And Rollback

Rollout 顺序：

1. Data forward migration 与 Entity Type 定义 backfill；
2. Data V3 Context/Submission provider 与 fixture；
3. 如投影范围变化，暂停 Event Semantic 并全量重建 Qdrant collection；
4. AgentRun V3 consumer、Retriever 与 Workflow；
5. 运行固定样本验收；
6. 启用 Event Semantic V3。

Mixed V2/V3 期间保持 Event Semantic 暂停，不允许 V2 Agent 消费 V3 Context。回滚时停用 Event
Semantic，回退 AgentRun/Data binaries；forward Schema 与历史数据保留，不执行破坏性 down migration。

## 13. Out Of Scope

- 自动创建缺失 Entity、Entity Type、Alias 或 Variable Definition；
- 在本 Spec 内决定 `country`、`region` 等新类型的完整 catalog；
- EntityRelation、Neo4j 或产业链图谱参与 Event Semantic；
- DirectImpact、跨实体因果传导、analyst inference；
- Theme、Reason Tree、机会/风险/不确定投资结论；
- Qdrant 增量同步、CDC、生产 HA、备份和监控；
- 用 similarity threshold 自动接受 EntityLink；
- 结构化 Observation 或 Measurement 数值计算。

## 14. Review Checklist

用户评审本 Draft 时需要确认：

1. 是否接受“LLM 不再预测正式 Entity Type；类型来自 selected Entity，PG 最终复核”；
2. 是否接受 Entity Type TBox 增加语义定义与 `event_link_allowed`；
3. 是否接受由本次实施 Agent 直接为全部现有 active Entity Type 编写扩展字段内容，并随
   forward migration/seed 一次性回填，不建设长期生成机制；
4. 是否接受第一阶段只提取 Mention，Variable Definition 延后到 Entity Resolution 之后；
5. 是否接受跨类型 Qdrant recall，Top-K 由固定样本校准；
6. 是否接受 candidate-level isolation 与 EventEntityLink 可独立于 VariableSignal 发布；
7. 是否接受 Mention 出现在 Event 或至少一条 Evidence，并保留 primary Evidence lineage；
8. `country` 是否在后续独立 TBox catalog 任务中决定，而不是本次顺带创建。

## 15. Corrective Eino reference-first audit

本轮在以下只读固定参考上重新完成 reference-first gate：

| Reference | Commit | Inspected | Decision |
| --- | --- | --- | --- |
| `cloudwego/eino` | `922b6a8a233b5233fe47eecee6cd2c005e8c39cd` | `compose/workflow.go`、`components/embedding/interface.go` | 继续使用 typed、acyclic、显式 Compile 的 Workflow 和批量 `EmbedStrings`；Stage envelope 的业务必填数组校验仍由 AgentRun Lambda 边界负责。 |
| `cloudwego/eino-ext` | `9137edd89e72b72735ede69db1c5ae29178a6e41` | `components/embedding/openai/embedding.go`、`components/retriever/qdrant/retriever.go` | 继续使用官方 OpenAI-compatible Embedder；标准 Retriever 仅支持单 query/单次 embedding 与单次 Qdrant query，无法表达 Event 多 Mention 批量和候选白名单，因此不采用。 |
| `cloudwego/eino-examples` | `171220631fb7068ead50b7cd964b8c471647117d` | `compose/workflow/1_simple/main.go` | 采用 composition root 显式构造/Compile 的结构；拒绝示例中的全局 callback 与调用期 `context.Background()`。 |

本轮没有发现可替代严格 JSON envelope、projection provenance 门禁、candidate whitelist 或
Data PostgreSQL authority 的 Eino/Eino Ext 组件；这些保持为 Tidewise AgentRun/Data 合同。

用户最新决议允许本次验收使用现有 active/current TBox 中的 `economy`、`index`、`market`、
`instrument` 等新增定义。Workflow 不在代码中维护类型 denylist，仍完全服从 pinned TBox 的
`event_link_allowed`；只有正式定义未启用的类型才归为 `tbox_out_of_scope`。本次整改不额外生成、
删除或改写这些 catalog row，也不以扩大目录掩盖 Workflow 质量问题。
