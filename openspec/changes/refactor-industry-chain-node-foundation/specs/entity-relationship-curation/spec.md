## ADDED Requirements

### Requirement: 产业节点关系独立策展
系统 SHALL 将 chain_node 静态关系与通用 `entity_edges` 分离，在独立 `chain_node_relations` 中按四类语义逐层 Review、Write 和 Query。

#### Scenario: Review 产业节点候选边
- **WHEN** 系统准备 `is_subcategory_of`、`is_component_of`、`input_to` 或 `depends_on` 候选边
- **THEN** 清单必须包含中文端点名称、entity key、方向、mechanism、condition、evidence、来源 URL 和核验时间

#### Scenario: 未批准候选不得写入
- **WHEN** 某条候选边或旧 edge 映射尚未获得明确人工确认
- **THEN** 系统不得将其写入正式 seed、PostgreSQL 或 Neo4j

## MODIFIED Requirements

### Requirement: 关系批次写入和图谱重建
系统 SHALL 将已审阅的通用 `entity_edges` 关系先写入 PostgreSQL，再按其 change 契约从 PostgreSQL 重建 Neo4j，不允许直接手工维护 Neo4j 关系事实；产业节点关系例外地写入独立 `chain_node_relations`，且本 change 只执行 PostgreSQL Write 与 Query，不执行 Neo4j rebuild。

#### Scenario: 写入已审阅关系批次
- **WHEN** 某一通用关系族通过 review 并执行实体 seed
- **THEN** 系统必须幂等 upsert 该批 `entity_edges`，并输出 created、updated、unchanged、failed 和按关系类型统计

#### Scenario: 从 PG 重建 Neo4j
- **WHEN** 某一通用关系批次的 PG 验收通过并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须包含全部可投影实体节点和当前 PG 中 active 的已审阅关系，且不得包含 PG 中不存在的历史关系

#### Scenario: 产业节点关系仅验收 PostgreSQL
- **WHEN** 本 change 的 chain_node relation 批次通过 Review 并获得 Write 授权
- **THEN** 系统必须幂等写入 `chain_node_relations` 并执行写后 Query
- **AND** 不得运行 Neo4j rebuild、清理或直接图写入

#### Scenario: 自动化验证关系清洗能力
- **WHEN** 开发者运行后端测试和 OpenSpec 校验
- **THEN** migration、loader、关系策略、repository、report 和 graph projection 边界必须具备自动化测试，且 `go test ./...` 与 OpenSpec 全局校验必须通过
