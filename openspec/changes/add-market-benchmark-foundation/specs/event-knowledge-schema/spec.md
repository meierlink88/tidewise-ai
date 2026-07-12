## ADDED Requirements

### Requirement: Benchmark PostgreSQL schema
系统 SHALL 通过增量 migration 增加 benchmark profile 和 observation schema，并保留全部既有实体、关系、采集和事件数据。

#### Scenario: 创建 benchmark profile
- **WHEN** benchmark schema migration 被执行
- **THEN** PostgreSQL 必须创建 benchmark profile 表并保存类型、官方 series code、provider、期限、标的 symbol、币种、单位、频率和来源 URL

#### Scenario: 创建 benchmark observations
- **WHEN** benchmark schema migration 被执行
- **THEN** PostgreSQL 必须创建 observation 表、benchmark 与时间索引、来源与时间幂等约束和质量状态约束

#### Scenario: 非破坏性升级
- **WHEN** 已有业务数据的数据库应用 benchmark migration
- **THEN** migration 不得删除、清空或重建既有实体、profile、关系、采集、事件或投影运行记录

#### Scenario: Observation 引用 benchmark
- **WHEN** 系统保存 benchmark observation
- **THEN** observation 必须引用 `entity_type=benchmark` 的有效实体，并拒绝引用 index、metric、commodity 或 instrument

