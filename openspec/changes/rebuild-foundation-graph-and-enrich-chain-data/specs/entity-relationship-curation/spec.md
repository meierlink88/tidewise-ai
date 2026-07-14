## ADDED Requirements

### Requirement: 842 节点关系候选按批审批
系统 SHALL 在每个已批准批次的四类关系审计状态完整、候选与异常/冲突全部处置后，才能冻结该批 final 数据，并保持 PostgreSQL Write 与 Neo4j sync 独立授权；批次验收不得替代 842 总目标覆盖验收。

#### Scenario: 分批候选审核
- **WHEN** 某个节点群的关系候选、无关系结论与证据已完整展示
- **THEN** 用户必须审核 identity、端点、类型、方向、证据、不适用/证据不足理由和异常/冲突项
- **AND** 任一分批审核都不得推定 PostgreSQL 或 Neo4j Write 授权

#### Scenario: 当批 final 结果冻结
- **WHEN** 已批准批次范围内节点的四类审计状态全部完整
- **THEN** 系统必须证明当批无待研究项、未处置候选、孤儿端点、重复 tuple 或未解决冲突，才能冻结 final 节点/关系数据

#### Scenario: PostgreSQL 先写并验收
- **WHEN** 当批 final 数据获得独立 local PostgreSQL R2 授权
- **THEN** 系统必须在现有事务边界内原子写入，并立即 Query 覆盖率、端点、tuple、orphan、范围外保护和幂等结果

#### Scenario: Neo4j 后同步并验收
- **WHEN** PostgreSQL 写后 Query 已验收且对应 local Neo4j R3 sync 另行获得授权
- **THEN** projector 必须只从 PostgreSQL accepted baseline 同步批准数据并 Query
- **AND** 不得从 Neo4j 反写 PostgreSQL 或同步未审核数据
