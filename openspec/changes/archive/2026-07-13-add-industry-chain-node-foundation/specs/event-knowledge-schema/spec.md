> **SUPERSEDED — DO NOT SYNC：** 本 delta spec 的4表 + profile扩展结构不再是目标 schema；已应用的000014与现有PG facts由后续 change 通过 forward migration 接管。

## ADDED Requirements

### Requirement: 静态产业链 PostgreSQL schema
系统 SHALL 通过非破坏性增量 migration 创建 `industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints` 并扩展 `chain_node_profiles`，保留全部既有业务数据。

#### Scenario: 创建静态结构
- **WHEN** migration 被执行
- **THEN** PostgreSQL 必须按 design field mapping 创建字段、FK、枚举检查、唯一约束、索引、自环约束和节点/边恰一主体约束

#### Scenario: 保存 AI 来源 provenance
- **WHEN** 经人工 Review 批准的 physical constraint 最初由 AI 生成
- **THEN** `industry_chain_physical_constraints.generated_by_ai` 必须保存为 true，且 migration 不得创建 `approved_by_human`、approval gate 或审批平台表

#### Scenario: 兼容已有节点
- **WHEN** migration 应用于已有 33 个 `chain_node` 的数据库
- **THEN** 必须保留 UUID、stable key、名称和既有 `chain_position`，不得删除、复制或隐式重分类节点

#### Scenario: 非破坏性升级
- **WHEN** 已有实体、关系、benchmark observation、事件或采集数据的数据库应用 migration
- **THEN** migration 不得删除、清空或重建任何既有业务表或数据

### Requirement: Observation schema 排除
系统 SHALL 将产业链 metric definitions/bindings、通用 observation governance、node/flow observations 和 writer/query 移到后续独立 change。

#### Scenario: 验证当前 migration scope
- **WHEN** 开发者检查当前 change migration
- **THEN** migration 不得创建 `industry_chain_metric_definitions`、`industry_chain_metric_bindings`、`observation_records`、`industry_chain_node_observations` 或 `industry_chain_flow_observations`

### Requirement: 静态 schema 自动化验证
系统 SHALL 提供 migration 静态测试及 domain/repository 测试，验证静态表约束、幂等与回滚边界。

#### Scenario: 运行后端验证
- **WHEN** 开发者验证产业链 schema
- **THEN** migration 测试、目标包测试和 `go test ./...` 必须通过，普通单元测试不得依赖真实外部网络或生产数据库
