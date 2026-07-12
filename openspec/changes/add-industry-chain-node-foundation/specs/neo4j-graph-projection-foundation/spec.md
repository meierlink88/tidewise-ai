## ADDED Requirements

### Requirement: 投影产业链定义与稳定拓扑
系统 SHALL 从 PostgreSQL 投影 active 且 approved 的 `industry_chain`、active `chain_node`、active membership 和已审阅 active topology，并保持统一 `Entity` 标签与 `projection_namespace`。

#### Scenario: 投影产业链节点
- **WHEN** graph projector 读取 active approved industry chain 和 active chain node
- **THEN** Neo4j 必须以统一 `Entity` 节点保存稳定身份和可查询 profile 属性，不得创建平行标签事实源

#### Scenario: 投影链内成员和拓扑
- **WHEN** PostgreSQL 包含 active membership 与 active reviewed topology
- **THEN** projector 必须映射链内成员与已批准拓扑枚举，并保留原始 ID、relation type、来源、状态和 namespace

### Requirement: 投影已审阅产业链跨实体关系
系统 SHALL 只投影 PostgreSQL 中 active 且端点均 active 的产业链跨实体 `entity_edges`，并安全映射关系枚举。

#### Scenario: 投影 benchmark 与 sector 路径
- **WHEN** PostgreSQL 包含已审阅 `observed_by_benchmark` 和 `represented_by_sector`
- **THEN** Neo4j 必须形成经产业链或节点连接 benchmark 与 sector 的客观路径，不得生成海外市场覆盖中国板块的关系

#### Scenario: 排除未审阅候选
- **WHEN** 候选链、节点、topology 或跨实体关系未进入 active PostgreSQL 事实
- **THEN** Neo4j 不得包含该候选或由其推导的关系

### Requirement: Observation 不进入 Neo4j 投影
系统 SHALL 将产业链时序 observation 保留在 PostgreSQL，不把每个 observation 创建为 Neo4j 节点或关系。

#### Scenario: 重建包含 observation 的数据库
- **WHEN** PostgreSQL 包含任意数量产业链 node 或 flow observations
- **THEN** graph projector 必须忽略 observation rows，并只投影稳定定义和关系

### Requirement: 产业链投影自动化验证
系统 SHALL 通过 repository source、mapping 和 projector 测试验证 active-only、枚举映射、缺失端点跳过和 observation 排除。

#### Scenario: 运行图投影测试
- **WHEN** 开发者运行目标包测试和 `go test ./...`
- **THEN** 测试必须在不连接真实 Neo4j 的条件下验证产业链投影行为，并全部通过
