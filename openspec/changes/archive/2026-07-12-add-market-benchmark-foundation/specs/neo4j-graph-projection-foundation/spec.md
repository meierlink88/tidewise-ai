## ADDED Requirements

### Requirement: Benchmark 图谱投影边界
系统 SHALL 从 PostgreSQL 投影 benchmark 定义节点及已审阅客观关系，并将时序 observation 保留在 PostgreSQL。

#### Scenario: 投影 benchmark 定义
- **WHEN** graph projector 读取 active benchmark 实体和 profile
- **THEN** Neo4j 必须创建统一 `Entity` 节点并保存 benchmark 类型、provider、期限、标的 symbol、币种、单位和频率

#### Scenario: 投影 benchmark 关系
- **WHEN** PostgreSQL 包含 active `observes_benchmark`、`measures` 或 `references` 关系
- **THEN** Neo4j 必须分别映射为 `OBSERVES_BENCHMARK`、`MEASURES` 和 `REFERENCES`

#### Scenario: 不投影 observation
- **WHEN** PostgreSQL 包含任意数量 benchmark observations
- **THEN** graph projector 不得为逐时点观测创建节点或关系，投影 source row count 也不得包含 observation 行

