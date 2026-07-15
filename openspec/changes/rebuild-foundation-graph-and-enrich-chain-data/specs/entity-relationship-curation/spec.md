## ADDED Requirements

### Requirement: 842 节点 usable-map additive 候选审批
系统 SHALL 保留既有 100 条 accepted baseline，并在 842 个既有 chain_node 之间发现和双遍审核 additive 四类关系候选；只有 842/842 节点均参加发现与检查、候选与异常/冲突全部处置后才能冻结 additive final 数据，并保持 PostgreSQL Write 与 Neo4j sync 独立授权。

#### Scenario: 分批候选审核
- **WHEN** 某个节点群的关系候选与证据已完整展示
- **THEN** 用户必须审核 identity、端点、类型、方向、机制、条件、证据 tier、反例和异常/冲突项
- **AND** 任一分批审核都不得推定 PostgreSQL 或 Neo4j Write 授权

#### Scenario: 全量 final 结果冻结
- **WHEN** 842/842 节点均已参加候选发现和两遍独立检查
- **THEN** 系统必须证明既有 100 条逐行保留，且无未处置候选、孤儿端点、重复 tuple、同机制重复或未解决冲突，才能冻结 additive final 关系数据
- **AND** 无关系事实的节点允许保持无边，不得为覆盖率强制登记关系

#### Scenario: PostgreSQL 先写并验收
- **WHEN** additive final 数据获得独立 local PostgreSQL R2 授权
- **THEN** 系统必须只向 `chain_node_relations` additive 写入，并立即 Query 既有 100 条保护、端点、tuple、orphan、节点主数据保护和幂等结果

#### Scenario: Neo4j 后同步并验收
- **WHEN** PostgreSQL additive 写后 Query 已验收且对应 local Neo4j R3 sync 另行获得授权
- **THEN** projector 必须只从最终 PostgreSQL accepted baseline 同步批准数据并 Query
- **AND** 不得从 Neo4j 反写 PostgreSQL 或同步未审核数据
