## ADDED Requirements

### Requirement: 投影静态产业链骨架
系统 SHALL 从 PostgreSQL 投影 approved active industry chain、active chain node、membership、topology、physical constraint 和已审阅 active `entity_edges`，并保持统一 `Entity` 与 `projection_namespace`。

#### Scenario: 投影 membership 与最小拓扑
- **WHEN** PostgreSQL 包含 active membership 和 reviewed active `supplies_to`、`depends_on`、`substitutes_for`
- **THEN** projector 必须映射稳定关系并保留原始 ID、relation type、来源、状态和 namespace

#### Scenario: 投影 approved 物理约束
- **WHEN** physical constraint 同时 active、approved 且主体可投影
- **THEN** projector 必须提供可查询的物理约束投影；candidate 或 reviewed 未 approved 记录不得投影

### Requirement: 排除 observation 与未审阅候选
系统 SHALL 不投影当前 change 范围外的 observation 平台或未进入 active PostgreSQL 事实的候选。

#### Scenario: 无 observation schema
- **WHEN** graph projector 为本 change 重建图谱
- **THEN** source 和 mapping 不得依赖 industry chain metric definitions/bindings、observation records 或 node/flow observations

### Requirement: 正确跨市场路径
系统 SHALL 投影已审阅 benchmark 与 sector 映射，使全球 benchmark 经 chain/node 到中国 sector，并拒绝海外 market 错误覆盖。

#### Scenario: 验证客观路径
- **WHEN** PostgreSQL 包含已审阅 `observed_by_benchmark` 和 sector mapping
- **THEN** Neo4j 必须形成经 chain/node 的路径，且不得生成海外 market `COVERS_SECTOR` 中国 sector

### Requirement: 静态投影自动化验证
系统 SHALL 使用 repository source、mapping、projector 和 fake graph writer 测试验证 active-only、approved-only、拓扑枚举、缺失端点跳过和 observation 排除。

#### Scenario: 运行投影测试
- **WHEN** 开发者运行目标包测试和 `go test ./...`
- **THEN** 测试必须不连接真实 Neo4j 并全部通过
