## ADDED Requirements

### Requirement: MVP 产业链跨实体关系 policy
系统 SHALL 只为 `scoped_to_economy`、`uses_commodity`、`produces_commodity`、`observed_by_benchmark` 和 `mapped_to_sector` 定义允许端点、方向、来源和语义。

#### Scenario: 连接 economy 与 commodity
- **WHEN** economy scope 或节点商品投入/产出通过 Review
- **THEN** 只允许 `industry_chain → economy` 的 `scoped_to_economy` 以及 `chain_node → commodity` 的 `uses_commodity` / `produces_commodity`

#### Scenario: 连接 benchmark
- **WHEN** benchmark 可客观观察某产业链或节点
- **THEN** 只允许 `industry_chain|chain_node → benchmark` 的 `observed_by_benchmark` 并保存完整 provenance

#### Scenario: mapped_to_sector 不表达身份
- **WHEN** 产业链或节点映射到中国 canonical sector
- **THEN** 必须使用 `mapped_to_sector` 并保存来源；该关系不表达身份、法定覆盖或影响方向

### Requirement: 全球 benchmark 到中国板块的正确路径
系统 SHALL 通过 chain/node 的 benchmark 与 sector 客观映射形成跨市场传导输入，不得伪造海外市场覆盖中国板块。

#### Scenario: 拒绝错误 COVERS
- **WHEN** 候选试图使用海外 `market → 中国 sector` 的 `covers_sector`
- **THEN** relationship policy 必须拒绝该候选

### Requirement: 分层写入门禁
系统 SHALL 将 membership、topology、physical constraint 和每个跨实体关系族分层 Review，并分别按 `Review → Write → Rebuild → Query` 推进。

#### Scenario: 上一层批准不推定下一层
- **WHEN** 任一数据族尚未逐项通过人工 Review
- **THEN** 该数据族不得进入正式 seed、PostgreSQL 或 Neo4j
