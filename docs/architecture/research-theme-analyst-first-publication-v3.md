# Research Theme Analyst-first Snapshot Publication V3

状态：事件推理模型第三轮复审 PASS，用户已于 2026-08-03 批准实施；实现已完成并通过
Standards/Spec 双轴复核，待 PR review；
实施票据：GitHub Issue #181

## 1. Outcome

让现有 Theme + Reason Tree publication seam 接受 Theme Analyst 自己推断的分析对象名称、
变量名称和变量状态，即使 Data 当前没有相应的正式 Entity、VariableDefinition、
VariableSignal、IndustryChain、Relation 或 GraphEdge，也能原子发布、不可变保存、完整
回读并供 Miniapp 按现有 UI 展示。

Analyst-first publication 是一份不可变报告快照，不是本体事实发布：

- Theme Analyst 可以在分析阶段参考 Data 的正式实体、变量、Signal 或图谱；
- “分析时使用过正式数据”不等于发布时必须绑定这些正式 ID；
- Data 是否已经存在对应 Entity、IndustryChain、VariableDefinition、VariableSignal、
  DirectImpact、Relation 或 GraphEdge，绝不得决定 V3 snapshot 能否发布；
- 没有正式 Entity/Variable/Signal 只表示本体覆盖不足，不表示 Theme 不可发布；
- 报告中的对象、变量状态、传导和投资结论由分析师负责，不因发布而成为 Data 本体事实；
- Data 负责请求合同、正式 Event/Evidence 来源引用、结构、事务、幂等、不可变和读取。

最高可观察验收 seam 是 Data Service PostgreSQL integration test：以一个完全不含
Ontology/Signal/Graph formal ID、但含正式 Event（及 optional Evidence）的真实 Analyst-first Theme +
多棵 Tree 执行 publish、replay、conflict、atomic rollback 和 detail readback，并通过
Miniapp consumer contract test 证明现有页面可展示。

## 2. Non-goals

- 不新增 Theme/Reason Tree 发布接口；继续使用现有
  `POST /api/data/v1/research-theme-imports`。
- 不建设 provisional -> formal、本体晋升、绑定审核、绑定状态或 data nature 派生体系。
- 不要求 Data 判断分析对象是否应成为 Entity、变量是否标准、Signal 是否正确或传导是否
  符合图谱。
- 不引入新的事实注册表、推理关系注册表或多级 grounding 模型。
- 不修改 Theme/Reason Tree 推理方法、Theme 拆分方法或 Miniapp UI。
- 不修改 Event Semantic、AgentRun、Neo4j、Qdrant 或其他投影。
- 不让 Data 执行 LLM 推理、Theme readiness、置信阈值或投资质量判断。
- 不删除现有 V1/V2 formal publication/read 能力。
- 实施不得超出本 Spec 的 snapshot publication 边界。

## 3. Authority 与事实基线

本 Spec 遵循：

- `AGENTS.md`；
- `docs/agents/engineering-standard.md`、`coding-standard.md`、`workflow.md`、
  `testing.md` 与 `domain.md`；
- `docs/contexts/data/CONTEXT.md`、`docs/contexts/miniapp/CONTEXT.md`；
- `docs/adr/0002-backend-service-architecture.md`、
  `docs/adr/0003-research-theme-batch-snapshots.md`；
- `docs/architecture/kratos-backend-development-standard-v1.md`；
- `docs/architecture/research-theme-reasoning-tree-spec.md`；
- `analyse-data-service/backend/api/data/v1/openapi.yaml`。

V3 只取代现有 Theme publication 中“必须绑定 formal Ontology/Signal/Graph 才能发布”的
规则。未被本文件改变的 Theme/Tree narrative、枚举、排序、分页、鉴权、错误、事务、
幂等和 V1/V2 读取合同继续有效。

真实验收基线为：

- `theme-analysis/runs/uat-analyst-first-cached-20260730t0300-0700-20260803t110116z/`
  下的 `acceptance-report.md`、`analysis.json`、
  `research-thread-candidates-frozen.json`；
- Theme Analyst 的 `AGENTS.md`、`generate-research-themes/SKILL.md`、
  `orchestration-contract-v1.md` 与 `quality-gates-v1.md`；
- 现有 Theme Analyst publication adapter 与 V2 request/readback artifacts。

该 UAT 在正式 IndustryChain membership、EntityRelation 和 GraphEdge 缺失时仍产生可用
Theme 与 Reason Tree，证明 formal Ontology 覆盖不能继续作为 Analyst-first publication
门禁。

### Taro reference-first brief

参考案例：Taro 4.x React 官方合同与项目现有 typed Port/Adapter/strict parser。

适用部分：保持 React 18 + Taro 4 的 typed data adapter，更新 DTO、parser 和 mapping。

不适用部分：不引入新组件、路由、平台 API、状态库或 UI library。

版本/平台限制：当前 `@tarojs/taro ^4.0.0`、React 18.2，微信优先且保留抖音兼容。

本项目落地方式：Miniapp 只把 formal UUID identity 改为 local key + display snapshot；页面
结构、交互和平台行为不变。参考：`https://docs.taro.zone/docs/react-overall`。

## 4. Grill with Docs 结论

最新领域决议已冻结本次最小边界，无阻塞产品问题。上一轮评审提出的扩展项按最新决议
明确降级：

| 上一轮扩展 | V3 最小决议 |
| --- | --- |
| formal/mixed/provisional component 分类 | 不建模、不派生、不在 Miniapp 暴露 |
| 通用 Path Scope 与 path type | 不新增领域分类；Tree 只保存 `tree_key`、`display_name` 和既有 narrative |
| FactSignalSnapshot registry | 不建 registry；Node 下保存 Miniapp 需要的 Signal display snapshot |
| InferenceRelation registry / grounding DAG | 不建 registry；非根 Node 保存既有 incoming transmission display snapshot |
| Research Thread provenance / candidate fingerprint | 不进入本次 Data publication 合同 |
| optional formal ID binding matrix | Analyst snapshot payload 不提交 formal Ontology/Signal/Graph IDs |

这不会放松旧 V1/V2 formal branch；它是在同一 route 增加一个明确隔离的
`analyst_snapshot` request variant。

## 5. Owner map

| 责任 | Owner | V3 边界 |
| --- | --- | --- |
| Event/Evidence 正式事实 | Data | 提供并验证正式 Event/Evidence 引用 |
| Theme、Reason Tree、对象/变量/传导/投资判断 | Theme Analyst | 生成不可变 display snapshot，并决定是否发布 |
| publication 事务、幂等、持久化、读取 | Data | 保存报告快照，不把内容升级为本体事实 |
| Miniapp public projection | Miniapp Backend | 机械映射 Data snapshot，不复制校验或数据库 |
| 页面展示 | Miniapp Frontend | 使用 local key/display snapshot，UI 不改版 |
| formal master data | Data 对应 master-data workflow | V3 snapshot publication 不写入、不更新 |

依赖方向保持 Theme Analyst -> Data REST -> Miniapp Backend REST -> Miniapp Frontend。禁止共享
数据库、导入对方 domain model 或 Frontend 直连 Data。

## 6. Current gap

### 6.1 Publish

当前 V2 request、Biz validator 和数据库约束要求：

- Theme impact `chain_node_entity_id` 必填且必须是 formal ChainNode；
- Tree `industry_chain_entity_id` 必填且必须是 active/approved IndustryChain；
- Node `chain_node_entity_id` 必填且必须属于该 IndustryChain；
- `formal_signal` 必须通过 Signal -> Submission -> Evidence -> Event -> Entity 血缘；
- `analyst_inference` 仍必须有 accepted formal Signal/Impact grounding，并由 formal
  Relation/GraphEdge 连接相邻节点。

因此，最新 UAT 中仅有 display object/variable/state/transmission，但数据库没有对应 formal
master/semantic ID 的 Tree 会被 `422` 拒绝。

### 6.2 Persistence/read

- Theme impact、Tree、Node 主要保存 formal ID，display name 读取时 join 当前 master；不是
  publication-time snapshot。
- Node signal 已保存部分 display 字段，但 identity/lineage constraint 仍依赖 formal branch。
- current read DTO 还会丢失部分 V2 lineage；V3 不扩展完整 lineage audit UI，但历史 V2
  已存字段不得因 dual-read 改造进一步丢失。

### 6.3 Miniapp

Miniapp Backend/Frontend 当前把 impact/node/chain ID 声明为必填 UUID，Frontend strict
parser 会拒绝 nullable 或 local identity。页面实际展示的是名称、变量状态、判断、传导和
结论，因此需要真实 DTO/parser/mapping 改动，但不需要 UI 改版。

## 7. Wire contract

### 7.1 Route、version 与 branch discrimination

route 不变：

```http
POST /api/data/v1/research-theme-imports
```

OpenAPI request 使用严格 `oneOf`：

- existing V2 formal request，保持现有 shape 与全部 formal validators；
- V3 analyst snapshot request，顶层必填 `publication_mode=analyst_snapshot`。

`publication_mode` 只是 wire variant discriminator，不表示 Ontology binding 或质量等级。
V2 request 不补该字段；V3 不通过字段缺失或 formal ID 是否为空来猜 branch。

首次成功 `201`；同 publisher、同 `analysis_batch_id`、同 canonical payload 重放 `200`；
相同幂等身份不同 payload `409`；JSON/schema `400`，结构或 Event/Evidence 引用失败 `422`，
body 过大 `413`，认证/授权 `401/403`。POST 预算 15s，scope 继续为
`data.research.import`，RFC 8785 + SHA-256 canonicalization 保持。

### 7.2 Aggregate

V3 request 保留现有：

- `analysis_batch_id`；
- `analysis_as_of`；
- `discovery_window_start`、`discovery_window_end`；
- exactly one `theme`；
- `reasoning_trees[1..N]`。

Theme + 全部 Trees、Events/Evidence associations、impacts、nodes、signals 和 receipt 必须在
同一个 PostgreSQL transaction 内全部提交或全部回滚。

### 7.3 Local identity

- `theme_key`：沿用现有 Theme local key。
- `tree_key`：aggregate 内唯一，pattern
  `^[a-z0-9][a-z0-9._:-]{0,127}$`。
- `node_key`：单个 aggregate 内的分析对象 identity，只用于 Theme impact -> 至少一棵
  Tree Node 的 coverage/reference；不同 presentation occurrence 的 `display_name` 可以不同。
- `signal_key`：Tree Node 内唯一的显示 Signal identity，同一 Node 内不得重复。
- local key 参与 canonical payload/hash，不从 display name 自动生成。
- local key 不是 formal Entity、Variable、Signal、IndustryChain 或 Relation ID，也不写入
  master-data tables。
- local key 的作用域只在当前 publication aggregate 及其明确子作用域内；Data 不跨
  publication 比较 local key、名称或内容。

Tree ID 从 `theme_id + NUL + tree_key` 确定性生成。Node ID 从
`reasoning_tree_id + NUL + node_key` 确定性生成。position 只描述路径顺序，不参与 identity。

## 8. Analyst snapshot objects

### 8.1 Theme

Theme 保留当前 title、conclusion、guidance、risk、time horizon、transmission、attention、
checkpoint 等 narrative 字段。

Theme impact 改为报告 snapshot：

| Field | Rule |
| --- | --- |
| `node_key` | required；必须存在于至少一棵 Tree |
| `display_name` | required；保存发布时名称 |
| `relation_role` | existing enum |
| `impact_direction` | existing enum |
| `impact_summary` | existing nullable display text |
| `display_order` | required、连续且唯一 |

V3 Theme impact 不接受 `chain_node_entity_id` 或其他 formal Entity ID。
`ThemeImpact.display_name` 是 Theme 卡片的短关注对象名；匹配到的
`TreeNode.display_name` 是该 Tree 内的具体推理节点名。两者可以合法不同，Data 只校验
`node_key` coverage，不比较文案。

### 8.2 Theme/Tree Event and Evidence association

Theme 与 Tree 继续保存 Event association；V3 每个 association 增加正式 Evidence 引用：

| Field | Rule |
| --- | --- |
| `event_id` | required Data Event UUID |
| `evidence_ids` | optional；提供时为 1..N 个唯一 Data Evidence UUID |
| `evidence_role` | existing enum |
| `supported_claim` | Theme association 沿用；Tree 可 nullable |
| `display_order` | Tree association required；Theme 沿用当前顺序规则 |

V3 Theme 和每棵 Tree 都至少包含一个正式 Data Event association。Data 验证 Event 存在；
只有 caller 提供 `evidence_ids` 时，Data 才验证 Evidence 存在且属于所声明 Event。Evidence
省略不阻止 snapshot publication，因为 Miniapp 当前展示 Event，Event 自身已由 Data 持有
Evidence。Data 不读取 AgentRun 原文，不判断 Evidence 是否足够或 claim 是否正确。一个
Tree 使用的来源 Event 必须出现在该 Tree association 中；Theme-level association 不能替代
Tree association。

这只是报告来源引用，不是新的事实 registry，也不建立 fact-to-node 或 inference grounding。

### 8.3 Reason Tree

| Field | Rule |
| --- | --- |
| `tree_key` | required、aggregate 内唯一 |
| `display_name` | required；Miniapp Tab/路径显示名 |
| `title` | existing Tree title |
| `display_order` | required、连续且唯一 |
| narrative fields | 保留 one-line conclusion、fact/transmission/impact/boundary/support/counter summary |
| invalidation/checkpoints | 保留现有数组与枚举 |
| events | 见 8.2 |
| nodes | `1..N` |

V3 Tree 不接受 `industry_chain_entity_id`。`display_name` 可以是产业链、宏观传导链、估值链
或商业模式链名称；Data 不分类、不验证其是否是正式 IndustryChain。

### 8.4 Node

| Field | Rule |
| --- | --- |
| `node_key` | required；同一 Tree 唯一 |
| `display_name` | required；保存当前 Tree occurrence 的发布时名称 |
| `position` | required，严格连续 `1..N` |
| `state_summary` | existing nullable display text |
| judgment fields | 保留 impact direction/strength/summary、reasoning basis、evidence gap |
| `incoming_transmission` | root 必须 null；每个非 root 必填 |
| `signals` | `1..5` display snapshots |

V3 Node 不接受 formal Entity/ChainNode ID 或 entity type。路径相邻关系由 Node position 和
非根节点的 incoming transmission 隐式表达：position `n` 的 incoming 永远表示
position `n-1` -> `n`，不新增 Relation identity 或 grounding。

Incoming transmission 保存现有展示字段：

- nullable `title`；
- required `mechanism`；
- nullable `condition_summary`。

Data 只校验 root/non-root 结构、长度和 enum，不检查 Relation/GraphEdge、端点、本体方向、
confidence 或经济学合理性。Data 不从 mechanism 自动生成 title；Theme Analyst 提供什么
presentation snapshot，Data 就保存什么。

### 8.5 Signal display snapshot

每个 Node Signal 是 Miniapp 展示所需的分析师变量状态快照；核心字段直接对应当前页面的
`primarySignal.displaySummary`：

| Field | Rule |
| --- | --- |
| `signal_key` | required；Node 内唯一 |
| `display_summary` | required；直接承接 Analyst `variable_state`，如“完成流片”或“商业化改善但质量未知” |
| `role` | required，existing `primary|supporting|contradicting`；每个 Node 恰好一条 primary，且为第一条 |
| `display_order` | `1..5`，连续且唯一 |
| `variable_name` | nullable；仅在 Analyst 明确给出独立变量名时保存 |
| `direction` | nullable；提供时校验 existing `increase|decrease|mixed|unchanged|uncertain` |

V3 Signal 不接受 VariableDefinition、VariableSignal、Submission、DirectImpact、Relation、
GraphEdge ID，也不要求 formal subject/variable/version/status。Data 不从 `display_summary`
推断 variable name 或 direction，不标准化 variable name，也不判断状态是否正确。没有
direction 不是 `422`。

## 9. Deterministic validation matrix

### 9.1 Data SHALL validate

| Boundary | Validation |
| --- | --- |
| Request | strict JSON/schema/additionalProperties、1 MiB、length、basic enum、UTC、key pattern |
| Idempotency | publisher + analysis_batch_id + canonical payload/hash；same replay、different conflict |
| Aggregate | exactly one Theme、1..N Trees、unique/continuous display order |
| Local identity | tree_key aggregate-local unique；node_key impact coverage；signal_key Node-local unique；不跨 publication 比较 |
| Impact closure | 每个 Theme impact node_key 至少存在于一棵 Tree |
| Path | Node position 连续；root incoming null；每个非 root incoming required |
| Signal | `1..5`、恰好一条 primary 且为第一条、display order；optional direction 有值时才校验 enum |
| Event/Evidence | Event ID 存在；optional Evidence 有值时验证存在且属于 Event；Tree 来源 Event 在 Tree association 中 |
| Transaction | receipt、Theme、Trees、events/evidence、impacts、nodes、signals 全部同事务 |
| Readback | local keys、display snapshot、Event/Evidence、null 和顺序逐字段稳定返回 |

### 9.2 Data SHALL NOT validate for `analyst_snapshot`

- Entity 是否存在、active/approved 或类型匹配；
- Node 是否为正式 IndustryChain member；
- IndustryChain 是否存在；
- VariableDefinition 是否存在、active 或 key/version 匹配；
- VariableSignal 是否 accepted/latest/non-superseded；
- Signal subject 是否等于某个正式 Entity；
- DirectImpact、Submission、Relation/GraphEdge 是否存在或连接相邻节点；
- inference 是否由正式 Signal/Impact grounding；
- confidence、Evidence 是否足够、Theme readiness、投资结论或传导机制是否正确。

上述“不校验”只适用于 `publication_mode=analyst_snapshot`。旧 V2 formal request 继续执行
现有全部 formal validators，不能通过切换字段组合静默降级；V3 strict schema 直接禁止提交
formal ontology IDs。

## 10. Persistence design

后续实现优先扩展现有 aggregate/tree/node/signal/incoming transmission 表和 DTO，不建设第二
套领域模型。最低必要变化：

1. Receipt 保存 `publication_contract_version=3` 与 `publication_mode=analyst_snapshot`，
   receipt tree map 改为/增加 `reasoning_tree_ids_by_tree_key`。
2. Theme impact 增加 `node_key`、`display_name`；V3 行不依赖 formal ChainNode FK，唯一约束
   使用 `(theme_id, node_key)`。
3. Reason Tree 增加 `tree_key`、`display_name`；V3 formal IndustryChain FK 为 null，唯一约束
   使用 `(theme_id, tree_key)`。
4. Node 增加 `node_key`、`display_name`；V3 formal ChainNode FK 为 null；Tree 内 node key 与
   position 分别唯一。
5. Signal 增加/映射 `signal_key`、`display_summary`、nullable `variable_name`、nullable
   direction、role、order；V3 formal lineage columns 全部为 null，不触发 V2 lineage checks。
6. 复用现有 Node incoming title/mechanism/condition columns，不新增 Relation table。
7. 复用现有 Theme/Tree Event association；optional `evidence_ids` 仅在 caller 提供时保存，
   Data Biz 批量验证 Evidence -> Event 归属。可选审计字段不得成为无 Evidence payload 的
   migration/write 门禁，也不得演化为事实/推理 registry。
8. V3 display name/state 直接保存在 publication rows；read 不用当前 master name 覆盖。
9. immutable trigger/constraint 覆盖新增 snapshot 字段。

现有 non-null FK、primary key 和 lineage CHECK 只能对 V3 snapshot 行按 contract version/mode
有条件放松；V1/V2 formal 行和 formal publish branch 的约束不得删除或变弱。业务分支规则
位于 Biz，Postgres 保证事务/不可变/最低约束，HTTP binding 不承载业务判断。

## 11. Read API and OpenAPI

现有读取路径不变：

- `/api/data/v1/research/themes`；
- `/api/data/v1/research/themes/{theme_id}`；
- `/api/data/v1/research/themes/{theme_id}/reasoning-trees`；
- `/api/data/v1/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}`。

V3 Theme/Tree detail readback 至少返回：

- publication contract version/mode；
- theme/tree/node/signal local keys；
- Theme impact、Tree、Node、Signal、incoming transmission 的发布时 display snapshot；
- Theme/Tree Event associations 及 optional Evidence IDs；
- existing narrative、order、timestamps 和 generated IDs。

List/Tree-tab summary 返回现有 UI 所需 title/display name/order/count，不复制完整 Evidence
集合。OpenAPI 必须对 V2 formal 与 V3 snapshot request/read union 定义 exact required/null、
`additionalProperties:false`、enum、pattern、example 和 discriminator，handwritten DTO 同步。

V3 detail request/readback 必须逐字段对称。Data 可以为 V1/V2 继续返回历史 formal IDs 和
已有 lineage；V3 不为 snapshot 伪造这些字段。

## 12. Miniapp impact

### 12.1 Miniapp Backend

- Data port/DTO 支持 V1/V2 formal 和 V3 snapshot dual-read；
- Theme impact matching 从 formal `chain_node_entity_id` 改为 `node_key`；
- Tree tab 使用 `tree_key`/generated reasoning tree ID 与 `display_name`；
- Node/Signal 使用 local key 和 `display_summary` snapshot；optional variable/direction 保真；
- 机械映射 Data errors、timeout 和 null，不执行 Ontology 或投资校验。

### 12.2 Miniapp Frontend

- strict parser/types 接受 V3 exact BFF shape；
- impact intersection 使用 `nodeKey`；
- 页面继续从 primary Signal `displaySummary` 渲染变量状态，并渲染 Tree/Node display name、
  传导、判断和结论；
- 不增加 provisional badge，不改 UI、路由、交互、loading/error/empty 或平台 API。

## 13. Compatibility, rollout and rollback

### 13.1 Dual-read

- V1/V2 formal rows 继续读取 formal ID、现有 display 字段和已有 lineage；
- historical `legacy_snapshot` 缺失 lineage 时保持 explicit null，不补造；
- V1/V2 没有 publication-time name snapshot 时允许 current-master fallback，但必须保留现有
  legacy 兼容语义，不宣称是发布时快照；
- V3 永远读取 publication-time snapshot，不因 master rename 改变；
- 新 V3 payload 不能伪装成或写入 formal branch。

### 13.2 Rollout

1. Data additive DB changes + V1/V2/V3 dual-read，V3 write 暂关闭。
2. Data OpenAPI/DTO/Biz snapshot branch 与 provider/consumer fixtures 上线。
3. Miniapp Backend/Frontend dual-read parser/mapping 上线，UI 不变。
4. Theme Analyst adapter 生成 `analyst_snapshot` request，并以真实 UAT fixture prepare。
5. 开启 V3 write，执行 publish/replay/readback smoke。

回滚只关闭 V3 write/provider；已发布 V3 snapshot 必须继续可读，不删除、不降级、不反向
填充 formal IDs。数据库 additive columns 不在紧急回滚中删除。

## 14. Security and failure semantics

- 认证 scope 保持 `data.research.import` / `data.research.read`。
- Payload 不包含 AgentRun 原文或 Secret；只引用 Data Event/Evidence ID。
- 15s POST 内 transaction commit 或 rollback；unknown result 只重放相同 payload。
- 不支持 partial Tree success；任一 Tree/Node/Event/Evidence 失败则 Theme 和 receipt 全回滚。
- canonical hash 覆盖所有 local key、display snapshot、Event/Evidence 与顺序。
- 错误返回稳定字段路径/error code，不暴露数据库实现。

## 15. Acceptance matrix

| Case | Input/action | Expected |
| --- | --- | --- |
| Legacy read | representative V1/V2 formal + `legacy_snapshot` rows | 全部可读；现有 formal 数据/lineage 不被 V3 破坏 |
| Snapshot publish | no Entity/Variable/Signal/Chain/Relation IDs；formal Event、optional Evidence；one Theme + multiple Trees | `201`；不查询/写入 formal master tables 作为门禁或副作用 |
| Real UAT prepared fixture | 最新 UAT 中一个真实 Theme 经 Theme Analyst presentation preparation 生成 canonical V3 request | fixture 标注每个字段来自 UAT 原字段还是 Analyst preparation；Data 不补语义，且不因本体覆盖不足 `422` |
| Snapshot shape | macro、industry 或 business-model display names | 同一 DTO 成功；Data 不分类 Tree/object/variable |
| Event only | valid Event association, no Evidence IDs | success；Event 正常展示，不因缺 optional Evidence `422` |
| Event/Evidence | optional valid Evidence belongs to Event | success；detail 对称回读 IDs/role/claim/order |
| Bad Evidence | caller 提供的 Evidence 不存在或属于另一个 Event | `422`；零写入；省略 Evidence 合法 |
| Missing impact node | Theme impact node_key 不在任何 Tree | `422` |
| Bad path | position gap、root incoming non-null、non-root incoming missing | `422` |
| Duplicate keys | duplicate tree/node/signal key in defined scope | `422` |
| Atomicity | 最后一棵 Tree 最后一个 Node 结构错误 | Theme、所有 Trees、receipt 全部不存在 |
| Replay | same publisher+batch+canonical payload | `200`；IDs/hash/timestamps 与首次一致，replayed=true |
| Conflict | same publisher+batch，任一 snapshot/Event/Evidence 改变 | `409`；原 aggregate 不变 |
| Readback | GET Theme + Tree detail | local keys、display snapshot、Event/optional Evidence、order/null 与 request 对称 |
| Master rename | V3 publish 后同名 formal master 被修改 | V3 display 不变，因为读取 publication snapshot |
| Miniapp | V3 Data -> BFF -> strict parser -> existing page | 正常展示且无需 formal UUID；UI 无改版 |
| Formal regression | existing V2 formal publish fixtures | strict formal validators 与成功/失败结果保持 |

默认测试 seam：

- Data Biz table tests：strict snapshot schema、key/impact/path closure、Event/Evidence batch
  validation、V2/V3 branch isolation；
- Data PostgreSQL integration：atomicity、immutability、idempotency、V1/V2/V3 dual-read、
  publication-time snapshot；
- Handler/OpenAPI contract tests：oneOf/discriminator、status/error、unknown fields；
- Miniapp Backend contract tests：V1/V2/V3 mapping；
- Miniapp Frontend parser/view-model tests：local key matching 与无 formal UUID；
- Theme Analyst provider fixture：真实 UAT Theme -> Analyst-owned presentation preparation ->
  annotated canonical request -> publish -> readback。

## 16. Implementation split and responsibility

### A. Data API/Biz contract

- OpenAPI/DTO 增加严格 `analyst_snapshot` variant；
- 实现最小 schema、local-key closure、Event/Evidence 和 V2/V3 branch validators；
- 不引入 Ontology binding/domain classification。

### B. Data persistence/readback

- additive migration、nullable/conditional formal columns、local keys/display snapshots；
- 原子 write、receipt V3、dual-read 与 immutable snapshot；
- 不新增 fact/relation/grounding registry。

### C. Miniapp consumer

- BFF port/DTO/mapping 与 Frontend parser/type/view-model 使用 local key/snapshot；
- 不改 UI 或平台行为。

### D. Theme Analyst provider

- 执行 Theme Analyst-owned presentation preparation，在 prepare 前生成 Miniapp 所需的完整
  Theme/Tree/Node/Signal/incoming display contract；
- 对 canonical fixture 逐字段标注“来自 UAT 原字段”或“由 Analyst presentation
  preparation 生成”，Data 不补文案、不推断 enum；
- 补充 Theme/Tree Event associations；Evidence refs 可选；
- 不查找、不伪造、不提交 formal Ontology/Signal/Graph IDs；
- 保存 prepare-only canonical request，并验证真实 UAT round-trip。

交付顺序为 A+B -> C -> D -> smoke。每个 owner 只通过 versioned OpenAPI/fixture 协作，不
共享数据库或 domain package。

## 17. Unresolved product choices

没有阻塞本次最小 Analyst-first snapshot publication 的产品选择。Ontology promotion、绑定
审核、完整分析 lineage UI、Research Thread provenance、data nature、置信度门禁和 Tree
分类都是未来独立需求，不在 V3 中预留字段或行为。

本 Spec 已通过事件推理模型复审并获得用户实施批准；任何超出上述边界的
本体绑定、晋升或新 UI 需求仍需独立 Spec 和授权。
