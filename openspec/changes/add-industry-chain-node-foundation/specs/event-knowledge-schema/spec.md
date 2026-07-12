## ADDED Requirements

### Requirement: 产业链 PostgreSQL schema
系统 SHALL 通过非破坏性增量 migration 增加 `industry_chain_profiles`、改进 `chain_node_profiles`，并创建 `industry_chain_memberships`、`industry_chain_topology_edges` 与 `industry_chain_constraints`，保留全部既有实体、关系和业务数据。

#### Scenario: 创建产业链结构
- **WHEN** 产业链 schema migration 被执行
- **THEN** PostgreSQL 必须按 `design.md` 的 Schema field mapping 创建 profile、membership、topology、constraint 的字段、外键、枚举检查、唯一约束、状态和来源字段

#### Scenario: 兼容已有产业链节点
- **WHEN** migration 应用于已有 33 个 `chain_node` 的数据库
- **THEN** 必须保留节点 UUID、stable key、名称与既有 `chain_position`，不得删除、复制或隐式重分类节点

### Requirement: Observation governance 与 typed schema
系统 SHALL 通过增量 migration 创建 `observation_records`、`industry_chain_node_observations` 和 `industry_chain_flow_observations`，并保证一个 envelope 只能对应符合其 `observation_type` 的 typed row。

#### Scenario: 创建 observation schema
- **WHEN** migration 被执行
- **THEN** PostgreSQL 必须创建来源、时间、修订、质量、幂等约束和 typed 外键，并拒绝缺少有效 metric 或领域端点的 validated observation

#### Scenario: 非破坏性升级
- **WHEN** 已有 benchmark observations、事件或采集数据的数据库应用 migration
- **THEN** migration 不得删除、清空或重建任何既有表或业务数据

### Requirement: 结构性约束 schema
系统 SHALL 使用 `industry_chain_constraints` 保存节点或拓扑边上的结构性约束类型、形成机制、替代路径、扩产路径、来源和 Review 状态，不得将当前瓶颈结论或评分混入稳定主数据。

#### Scenario: 创建节点级约束
- **WHEN** 已审阅供应商集中、认证周期、扩产周期、纯度、良率、专用设备、know-how 或监管约束
- **THEN** constraint 必须只引用同链一个节点，并保存合法 constraint type、机制与完整来源

#### Scenario: 创建边级约束
- **WHEN** 已审阅基础设施连接、投入流量或替代路径约束
- **THEN** constraint 必须只引用同链一个 topology edge，不得同时引用节点

#### Scenario: 拒绝推理字段
- **WHEN** constraint 写入包含 severity、score、benefit、pressure、prediction 或当前瓶颈结论
- **THEN** domain validator 和 repository 必须拒绝该写入

### Requirement: 产业链 schema 自动化验证
系统 SHALL 提供 migration 静态测试和可重复的 repository/domain 测试，验证产业链约束、幂等与回滚边界。

#### Scenario: 运行后端验证
- **WHEN** 开发者验证产业链 schema
- **THEN** 相关 migration 测试、目标包测试和 `go test ./...` 必须通过，且普通单元测试不得依赖真实外部网络或生产数据库
