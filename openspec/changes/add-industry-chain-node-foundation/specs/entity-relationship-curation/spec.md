## ADDED Requirements

### Requirement: 产业链跨实体关系 policy
系统 SHALL 为 `scoped_to_economy`、`uses_commodity`、`produces_commodity`、`observed_by_benchmark` 和 `represented_by_sector` 定义允许的 from/to 实体类型、方向、来源和语义；产业链指标适用关系必须进入专用 binding 表而不是 `entity_edges`。

#### Scenario: 连接产业链与经济体
- **WHEN** economy-scope 产业链关系通过 Review
- **THEN** 只允许 `industry_chain -> economy` 的 `scoped_to_economy`，并要求与 profile scope 一致

#### Scenario: 连接节点与商品
- **WHEN** 节点投入或产出商品关系通过 Review
- **THEN** 只允许 `chain_node -> commodity` 的 `uses_commodity` 或 `produces_commodity`，且两种语义不得互换

#### Scenario: 连接 benchmark 与 sector
- **WHEN** 产业链或节点的 benchmark 和市场映射通过 Review
- **THEN** 系统必须使用 `observed_by_benchmark` 或 `represented_by_sector` 的明确方向并保存完整 provenance，且不得用 `measured_by` 连接产业链专用指标

### Requirement: 全球 benchmark 到中国板块的正确传导路径
系统 SHALL 通过产业链或节点的客观映射连接全球 benchmark 与中国 canonical sector，不得伪造海外市场覆盖中国板块的事实。

#### Scenario: 构建跨市场传导输入
- **WHEN** 全球 benchmark 被用于分析中国板块
- **THEN** 客观图路径必须经 `observed_by_benchmark` 与 `represented_by_sector` 连接产业链或节点，推理方向由事件与 observation 决定

#### Scenario: 拒绝错误 COVERS
- **WHEN** 关系候选试图用海外 `market -> 中国 sector` 的 `covers_sector` 表达传导
- **THEN** relationship policy 必须拒绝该候选

### Requirement: 产业链关系分层写入门禁
系统 SHALL 将 membership、topology 和每个跨实体关系族分层 Review，并分别按 `Review → Write → Rebuild → Query` 推进。

#### Scenario: 未批准关系不得写入
- **WHEN** 任一关系族尚未获得逐项人工 Review
- **THEN** 该关系族不得进入正式 seed、PostgreSQL 或 Neo4j，且上一层批准不得推定本层授权
