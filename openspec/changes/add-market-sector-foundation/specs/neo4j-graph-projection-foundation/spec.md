## ADDED Requirements

### Requirement: 市场板块图谱投影边界
系统 SHALL 从 PostgreSQL 当前事实投影市场板块实体和已审阅客观关系，并保持 Neo4j 为可重建查询视图。

#### Scenario: 投影板块实体
- **WHEN** graph projector 读取 active `sector` 实体和 profile
- **THEN** Neo4j 必须创建统一 `Entity` 节点并保留 `projection_namespace`、`entity_key`、`entity_type=sector`、名称、规范名称、状态和可查询的板块分类属性

#### Scenario: 投影市场覆盖板块关系
- **WHEN** PostgreSQL 包含 active `covers_sector` 实体关系
- **THEN** Neo4j 必须将其映射为 `COVERS_SECTOR`，并保留 edge ID、原始 relation type、来源、状态、更新时间和投影命名空间

#### Scenario: 投影板块 benchmark 跟踪关系
- **WHEN** PostgreSQL 包含 active `tracked_by_benchmark` 实体关系
- **THEN** Neo4j 必须将其映射为 `TRACKED_BY_BENCHMARK`，并保留 edge ID、原始 relation type、来源、状态、更新时间和投影命名空间

#### Scenario: 不投影来源映射
- **WHEN** PostgreSQL 包含 `sector_source_mappings`
- **THEN** graph projector 不得把 source mapping 创建为 Neo4j 节点或关系，除非后续 change 明确扩展投影边界

#### Scenario: 不投影候选清单
- **WHEN** 板块候选尚未通过 Review 或尚未写入 active PostgreSQL 事实
- **THEN** graph projector 不得在 Neo4j 创建对应节点或关系

#### Scenario: 不投影推理结论
- **WHEN** 后续事件推理生成板块影响评分、传导强度、受益承压或预测结论
- **THEN** 这些内容不得通过实体基础图投影为 Neo4j 的基础实体关系
