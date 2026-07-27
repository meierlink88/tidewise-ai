# 行业—概念—产业链—节点关系构建 Spec V1

**状态：** 生效  
**版本：** `entity-relationship-construction-v1`  
**生效日期：** 2026-07-27  
**事实所有者：** Tidewise Data Domain Service  
**当前执行方式：** 由具体 Codex 任务离线构建、校验并生成人工审核包；不自动写入数据库

## 1. 目标

本 Spec 统一以下四类主数据之间的结构关系：

- Industry（行业）；
- Concept（概念）；
- Industry Chain（产业链）；
- Chain Node（产业链节点）。

每次 Codex 关系构建任务必须先读取本 Spec，只能使用本 Spec 已批准的关系类型、方向和
落表语义。当前阶段先生成版本化候选关系包和审核工作簿，用户批准后再进入独立的
local PostgreSQL、UAT 导入任务。

本 Spec 的直接目标是支持：

1. 从产业链查询其关联行业和关联概念；
2. 从产业链查询全部成员节点及节点的链内阶段、位置；
3. 在同一产业链内沿节点边构建结构图和传导路径；
4. 区分发现来源、候选关系、正式结构事实和单次分析推断；
5. 为未来 PostgreSQL 约束和 Neo4j 可重建投影保留稳定关系 code。

## 2. 非目标

V1 不负责：

- 建立覆盖所有 Entity Type 的通用关系本体；
- 自动创建未知关系类型；
- 将 AI 置信度当作正式关系证据；
- 根据链名相似、关键词共现或同一发现批次直接生成正式关系；
- 将 Event 影响、利好/利空、受益/承压或投资建议写入实体结构关系；
- 建立 Concept 与 Industry、Concept/Industry 与 Chain Node 的冗余直连边；
- 建立替代、协同或竞争正式边；这些关系在现有 Schema 支持前只进入待定义队列；
- 在本 Spec 中执行数据库迁移、local/UAT 写入或 Neo4j 投影。

## 3. 权威顺序

发生冲突时，按以下顺序处理：

1. 用户对本批关系的明确审核决定；
2. [产业链—节点协同发现与扩充方法 V1](./2026-07-22-industry-chain-node-coevolution-method-v1.md)；
3. 本 Spec；
4. 当前 PostgreSQL Schema、领域校验和导入合同；
5. 历史试点合同、旧关系和旧数据包。

数据库约束比离线候选格式更窄时，候选关系可以保留，但不得绕过约束直接写库。需要的
Schema 或应用策略变化必须作为独立导入前置任务明确列出。

## 4. 统一语言

### 4.1 产业链

围绕明确目标产出与终端用途，由多个独立经济节点通过投入、组成、技术支撑或依赖形成的
有边界、有方向研究子图。Industry Chain 不是 Industry、Concept 或 Chain Node 的别名。

### 4.2 发现映射

一个 Industry、Concept 或 Chain Node 因子在链名发现中指向某个候选产业链的运行记录。
发现映射用于解释候选来源，不自动等于正式实体关系。

### 4.3 产业链节点归属

某个 Chain Node 被纳入一个特定 Industry Chain 的上下文关系。`upstream`、
`midstream`、`downstream` 和 `position` 都属于这条归属关系，不是节点的全局属性。

### 4.4 产业链图谱边

同一 Industry Chain 的两个 active 成员节点之间，带有明确方向和传导机制的结构关系。
它不是关键词相关、发现映射或单次 Research Anchor 的临时路径。

### 4.5 全局节点关系

不依赖某一条命名产业链语境、可被独立证据支持的稳定 Chain Node 关系。链内图谱边不得
因为出现在一条链中就自动晋级为全局节点关系。

## 5. V1 关系类型注册表

以下 code 是 V1 唯一允许的规范关系类型。关系 code 使用小写 `snake_case`，不得翻译、
缩写、复数化或生成近义变体。

| code | 起点类型 | 终点类型 | 固定方向 | 传递性 | 正式存储 |
|---|---|---|---|---|---|
| `is_subcategory_of` | Industry | Industry | 子行业 → 父行业 | 是 | `industry_profiles.parent_industry_entity_id` |
| `mapped_to_industry` | Industry Chain | Industry | 产业链 → 行业 | 否 | `entity_edges` |
| `mapped_to_concept` | Industry Chain | Concept | 产业链 → 概念 | 否 | `entity_edges` |
| `has_node` | Industry Chain | Chain Node | 产业链 → 成员节点 | 否 | `industry_chain_node_memberships`，类型隐含 |
| `is_subcategory_of` | Chain Node | Chain Node | 子类节点 → 父类节点 | 是 | `chain_node_relations` |
| `input_to` | Chain Node | Chain Node | 投入/供应节点 → 使用或转化节点 | 否 | 链内 `industry_chain_graph_edges`；满足全局门禁时可写 `chain_node_relations` |
| `is_component_of` | Chain Node | Chain Node | 部件节点 → 总成/系统节点 | 默认否 | 链内 `industry_chain_graph_edges`；满足全局门禁时可写 `chain_node_relations` |
| `depends_on` | Chain Node | Chain Node | 依赖方节点 → 被依赖节点 | 否 | 链内 `industry_chain_graph_edges`；满足全局门禁时可写 `chain_node_relations` |

### 5.1 `mapped_to_industry`

表示产业链的稳定经济活动范围与一个已批准 Industry 存在实质覆盖。它是多对多映射，
不表示整条产业链“属于”单一行业，也不表示行业分类树的父子关系。

最低门禁：

- 两端必须解析到 approved、active 的规范实体；
- 产业链 scope、target output、end use 至少一项与行业定义和边界形成明确对应；
- 必须记录 `mapping_reason`；
- 仅有名称相似、同一关键词或模型常识不足以晋级；
- Industry M3 发现映射只能形成候选，需经关系审核后批准。

### 5.2 `mapped_to_concept`

表示产业链的目标产出、应用、技术路线、需求、政策、商业模式或生态范围与一个已批准
Concept 存在实质覆盖。它是多对多映射，不表示 Concept 是产业链成员或上下游节点。

最低门禁：

- 两端必须解析到 approved、active 的规范实体；
- Concept definition、boundary note 与产业链 scope/route/end use 必须有非空对应说明；
- 必须记录 `mapping_reason`；
- 公司名、概念别名或短词命中不能单独成为正式关系；
- Concept M3 发现映射只能形成候选，需经关系审核后批准。

### 5.3 `has_node`

表示 Chain Node 是特定 Industry Chain 的成员。正式记录必须同时包含：

- `industry_chain_entity_id`；
- `chain_node_entity_id`；
- `contextual_stage`：`upstream | midstream | downstream`；
- `position`：正整数；并行节点可以拥有相同 position；
- `inclusion_reason`：说明节点为何属于该链；
- `review_status`；
- `status`。

Node M1 发现映射只证明“该节点可以作为候选链的发现锚点”，不能单独提供
`contextual_stage`、`position` 或完整链边，因此只能转成 membership 候选。

### 5.4 节点图谱边

关系方向严格采用：

| code | 方向 |
|---|---|
| `input_to` | 投入/供应节点 → 使用或转化节点 |
| `is_component_of` | 部件节点 → 总成/系统节点 |
| `depends_on` | 依赖方节点 → 被依赖节点 |

`upstream`/`downstream` 是遍历方向，不是关系类型。每条边必须包含非空
`mechanism`；有成立条件时填写 `condition_note`。

`input_to` 多跳路径只表示存在投入传导，不得把首尾节点写成新的直接 `input_to`。
`depends_on` 多跳路径只表示间接依赖，不得自动生成直接依赖事实。
`is_component_of` 默认不做全局传递闭包；跨层总成关系必须保留中间层，或明确标记为
`compressed_candidate` 并填写 `omitted_step_note`。

## 6. 禁止关系类型

以下类型或表达不得写入正式关系：

- `related_to`、`关联`、`相关`、`其他`；
- `benefits`、`pressures`、`bullish`、`bearish`；
- `upstream_of`、`downstream_of`；
- 未经本 Spec 批准的 `supplies`、`supply_to`、`inputs_to` 等近义变体；
- `substitutes_for`、`competes_with`、`collaborates_with`，直至后续关系 Spec 和
  Schema Delta 明确；
- 由 AI 现场创造、无法映射到本注册表的自由字符串。

无法归类的候选写入 `unmapped_relation_candidates.json`，不得静默降级为
`related_to`。

## 7. 方向、反向查询与多关系规则

1. 每种关系只保存规范方向，不同时保存语义相同的反向边。
2. 查询可以反向遍历，但存储方向不改变。
3. 同一对实体可以存在多个语义不同的关系类型。
4. 同一语义关系不得因不同发现入口重复写入。
5. 不以“两个实体之间只能有一条边”作为唯一性规则。

规范唯一键：

```text
entity_edges:
  from_entity_id + relation_type + to_entity_id

industry_chain_node_memberships:
  industry_chain_entity_id + chain_node_entity_id

industry_chain_graph_edges:
  industry_chain_entity_id + from_chain_node_entity_id
  + relation_type + to_chain_node_entity_id

chain_node_relations:
  from_chain_node_entity_id + relation_type + to_chain_node_entity_id
```

## 8. 关系层级与落表规则

### 8.1 发现层

保存“哪个因子发现了哪个候选链”。它属于运行血缘，不进入正式实体图谱。

### 8.2 候选关系层

由 Codex 生成、尚未通过全部门禁和人工审核的结构候选。候选关系只写输出数据包，不写
PostgreSQL。

### 8.3 正式关系层

通过身份、语义、证据、方向、去重和人工审核的稳定结构事实。正式关系按下表路由：

| 关系 | 正式存储 |
|---|---|
| Industry 层级 | `industry_profiles.parent_industry_entity_id` |
| Industry Chain → Industry/Concept | `entity_edges` |
| Industry Chain → Chain Node | `industry_chain_node_memberships` |
| 链内 Chain Node → Chain Node | `industry_chain_graph_edges` |
| 全局稳定 Chain Node → Chain Node | `chain_node_relations` |

### 8.4 分析推断层

Event 影响、节点状态、Theme/Anchor 传导路径和投资判断属于带分析时点的推断或发布
快照，不得自动回写上述正式关系。

## 9. Codex 任务输入合同

每次构建任务必须冻结并记录：

- `relation_spec_version=entity-relationship-construction-v1`；
- 本 Spec 的相对路径和 SHA-256；
- Industry、Concept、Industry Chain、Chain Node 注册表版本和 SHA-256；
- 输入候选数据包路径和 SHA-256；
- 数据库来源、快照时间和证据截止时间；
- 本批允许构建的关系类型；
- 本批是否只生成候选，或已获得明确关系审核授权。

只允许引用 approved、active 的规范实体。无法解析、已停用、重定向未收敛或类型不符的
端点进入拒绝/未决队列。

## 10. 候选关系最小结构

### 10.1 产业链—行业/概念

```json
{
  "relation_key": "stable-key",
  "from_key": "industry_chain:...",
  "relation_type": "mapped_to_industry",
  "to_key": "industry:...",
  "mapping_reason": "产业链边界与行业定义的具体对应",
  "discovery_refs": [],
  "evidence_ids": [],
  "review_status": "candidate",
  "evidence_status": "inferred_only"
}
```

### 10.2 产业链—节点归属

```json
{
  "relation_key": "stable-key",
  "chain_key": "industry_chain:...",
  "relation_type": "has_node",
  "node_key": "chain_node:...",
  "contextual_stage": "upstream",
  "position": 1,
  "inclusion_reason": "节点的具体投入、产出或交付如何进入本链",
  "discovery_refs": [],
  "evidence_ids": [],
  "review_status": "candidate",
  "evidence_status": "inferred_only",
  "status": "active"
}
```

### 10.3 产业链节点图谱边

```json
{
  "relation_key": "stable-key",
  "chain_key": "industry_chain:...",
  "from_node_key": "chain_node:...",
  "relation_type": "input_to",
  "to_node_key": "chain_node:...",
  "mechanism": "前一节点的具体交付如何进入后一节点",
  "condition_note": null,
  "segment_kind": "direct_candidate",
  "omitted_step_note": null,
  "discovery_refs": [],
  "evidence_ids": [],
  "review_status": "candidate",
  "evidence_status": "inferred_only",
  "status": "active"
}
```

`confidence` 可以作为候选排序辅助字段，但不得替代 `mapping_reason`、`mechanism`、
`evidence_ids` 或人工审核状态，也不进入当前正式关系表。

## 11. 当前 708 条产业链数据包的转换规则

输入：

`outputs/industry-chain-discovery-v2/local-pg-baseline-20260727/`

当前发现映射包含：

| 发现入口 | seed→chain 配对数 | 被触达的唯一产业链数 | 转换结果 |
|---|---:|---:|---|
| Industry M3 | 319 | 312 | `mapped_to_industry` 候选 |
| Concept M3 | 172 | 152 | `mapped_to_concept` 候选 |
| Node M1 | 570 | 563 | `has_node` 候选，待补 stage/position |

第二列统计的是 `seed_key + canonical_chain_key` 配对数；第三列统计的是至少被该入口触达
一次的唯一产业链数。第三列不是“去重后的关系数”，不得用于关系条数或覆盖率校验。

转换规则：

1. 将 `candidate_chain_keys[]` 中每个 chain key 与当前 seed 配对；
2. Industry/Concept 发现映射统一反转为“产业链 → 行业/概念”的规范存储方向；
3. 按规范唯一键去重，多个发现批次合并进 `discovery_refs[]`；
4. 不因同一产业链由 Industry、Concept、Node 多入口共同发现而互相推导额外关系；
5. Node M1 候选在 M2 补齐链成员、阶段、位置前不得批准为 membership；
6. 节点图谱边必须通过 M2 逐跳扩展产生，不能仅根据链名或 Node M1 映射生成。

## 12. 每批构建顺序

1. 读取并校验本 Spec；
2. 冻结四类实体注册表和输入包；
3. 解析规范实体身份与 redirect；
4. 生成并去重产业链—行业、产业链—概念候选；
5. 生成节点归属候选，补充 `contextual_stage`、`position`、`inclusion_reason`；
6. 对同一链 active members 生成逐跳图谱边；
7. 区分链内边与全局稳定节点关系；
8. 执行方向、端点类型、自环、重复、DAG、证据和状态校验；
9. 生成 Excel 审核工作簿；若用户对本批已明确委托 Codex 在完整门禁后直接裁决，可改为
   生成机器可审计的裁决明细，不要求中间候选审核；
10. 用户逐条审核，或依据本批明确委托完成身份、语义、证据、方向和完整性门禁后，冻结
    批准包；
11. 由独立任务先导入 local PostgreSQL，核验后再导入 UAT；
12. PostgreSQL 成为事实源，Neo4j 只从批准事实重建投影。

## 13. 输出合同

每批至少输出：

```text
relationship_build_manifest.json
industry_chain_industry_relations.json
industry_chain_concept_relations.json
industry_chain_node_memberships.json
industry_chain_graph_edges.json
global_chain_node_relations.json
unmapped_relation_candidates.json
relationship_validation_report.json
关系人工审核.xlsx
```

Manifest 必须记录本 Spec 版本与 SHA-256、输入注册表版本、条数、去重数、拒绝数、
未决数、审核状态和是否执行数据库写入。

## 14. 硬性校验

- 起点和终点均存在，类型符合关系注册表；
- 无自环；
- 所有关系类型来自本 Spec；
- 所有方向符合固定方向；
- 所有链内边两端都是同一链的 active membership；
- active 产业链拓扑是 DAG；
- `compressed_candidate` 必须填写 `omitted_step_note`；
- `mapped_to_*` 必须填写 `mapping_reason`；
- `has_node` 必须填写 stage、position、inclusion reason；
- 图谱边必须填写 mechanism；
- 发现映射不得冒充正式关系证据；
- AI 候选不得仅凭模型置信度直接标记 approved；用户对具体批次明确委托直接裁决时，
  仍须逐条通过身份、语义、证据、方向、去重和完整性门禁，并保存可审计裁决依据；
- 同一语义关系不得因多入口或多批次重复；
- 禁止用 `related_to` 吸收无法判断的语义；
- 禁止在证据或机制中写利好、利空、受益、承压、预测和投资建议。

任一硬性校验失败时，该关系不得进入批准包。

## 15. 当前数据库兼容性

无需修改 Entity 主表即可执行候选关系构建。

当前 Schema 已支持：

- `industry_chain_node_memberships`；
- `industry_chain_graph_edges` 的 `input_to | is_component_of | depends_on`；
- `chain_node_relations` 的
  `is_subcategory_of | is_component_of | input_to | depends_on`；
- `entity_edges` 的起点、终点、关系类型和来源字段。

`mapped_to_industry`、`mapped_to_concept` 尚未加入当前
`relationship_policy.go` 的端点策略。现有通用 `entity-seed` 也不支持
Industry Chain、Industry Profile、Concept Profile 及本批多表原子导入；不得只增加两个
关系类型后直接复用该导入器。

正式写入前采用以下最小兼容方案：

- 不修改 `entity_nodes`；
- 为 `industry_chain_node_memberships` 补充非空 `inclusion_reason`，使正式表满足
  5.3 的数据合同；
- 新增一个只服务本关系包、Manifest 驱动、单事务且可重放的窄导入命令；
- 导入顺序固定为 Industry Chain 实体与定义、`mapped_to_*`、memberships、graph edges；
- dry-run 和写后校验必须覆盖输入 SHA、端点状态、唯一键、DAG、零孤链和内容漂移；
- UAT 只能在受控部署环境中使用同一冻结包显式 dry-run 后 apply，不把业务 DML 写进
  Schema migration。

## 16. Neo4j 投影约定

未来投影只读取 PostgreSQL 中 approved/active 的事实：

```text
IndustryChain -[:MAPPED_TO_INDUSTRY]-> Industry
IndustryChain -[:MAPPED_TO_CONCEPT]-> Concept
IndustryChain -[:HAS_NODE {contextual_stage, position}]-> ChainNode
ChainNode -[:INPUT_TO {chain_id, mechanism}]-> ChainNode
ChainNode -[:IS_COMPONENT_OF {chain_id, mechanism}]-> ChainNode
ChainNode -[:DEPENDS_ON {chain_id, mechanism}]-> ChainNode
```

Neo4j 关系类型使用本 Spec code 的大写形式。投影不得反向成为事实源，也不得从无类型
连通查询生成新的正式关系。

## 17. 变更规则

新增或修改关系类型时必须：

1. 给出无法复用现有类型的具体案例；
2. 明确起点、终点、方向、传递性、基数和正式存储；
3. 明确直接事实与多跳推导的区别；
4. 检查当前 Schema 和应用关系策略；
5. 生成人工审核差异；
6. 发布新的 Spec 版本；
7. 旧批次继续引用原版本，不静默套用新语义。

Codex 不得在单次关系构建任务中自行修改本 Spec 并同时使用新类型生成正式关系。
