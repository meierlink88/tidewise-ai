## ADDED Requirements

### Requirement: 有限产业链候选分层审批
系统 SHALL 只处理已经业务范围 Review 和 final 候选 Review 批准的有限 chain_node 与静态关系，并保持 PostgreSQL Write 与 Neo4j sync 独立授权。

#### Scenario: final 候选审核
- **WHEN** 有限首批范围的节点/关系候选和证据已完整展示
- **THEN** 用户必须审核范围、identity、端点、类型、方向、证据和异常/冲突项，再冻结 final 候选
- **AND** 候选冻结不得推定 PostgreSQL 或 Neo4j Write 授权

#### Scenario: PostgreSQL 先写并验收
- **WHEN** final 候选获得独立 local PostgreSQL R2 授权
- **THEN** 系统必须在现有事务边界内原子写入，并立即 Query 端点、tuple、orphan、范围外保护和幂等结果

#### Scenario: Neo4j 后同步并验收
- **WHEN** PostgreSQL 写后 Query 已验收且对应 local Neo4j R3 sync 另行获得授权
- **THEN** projector 必须只从 PostgreSQL accepted baseline 同步批准数据并 Query
- **AND** 不得从 Neo4j 反写 PostgreSQL 或扩大范围
