## ADDED Requirements

### Requirement: 市场板块图谱投影边界
系统 SHALL 从 PostgreSQL 当前事实投影市场板块实体和已审阅客观关系，并保持 Neo4j 为可重建查询视图。

#### Scenario: 投影板块实体
- **WHEN** graph projector 读取 active `sector` 实体和 profile
- **THEN** Neo4j 必须创建统一 `Entity` 节点并保留 `projection_namespace`、`entity_key`、`entity_type=sector`、名称、规范名称、状态和可查询的板块分类属性

#### Scenario: 从 sector profile 投影分类属性
- **WHEN** PostgreSQL active sector 的 `sector_profiles.classification_code` 属于已批准枚举
- **THEN** repository 必须通过现有实体节点 source 读取真实 profile 值，mapper 和 writer 必须将其写为 Neo4j `classification_code` 属性，不得从 entity key 推导或创建平行 profile 节点

#### Scenario: 缺失或非法板块分类 fail-closed
- **WHEN** active sector 缺失 `sector_profiles`、`classification_code` 为空或值不属于已批准枚举
- **THEN** projector 必须拒绝该 sector 节点投影并在既有 run item/report 中记录 failed；依赖该节点的关系按既有缺失端点策略记录 skipped，不得静默推导

#### Scenario: 非板块节点不保留分类属性
- **WHEN** graph projector 写入任意非 sector 实体
- **THEN** 参数不得携带 sector `classification_code`，writer 必须清除同一 namespace 节点可能残留的该属性

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

#### Scenario: convergence 后重建图谱
- **WHEN** PostgreSQL 已完成 sector convergence 并进行下一次实体图重建
- **THEN** Neo4j 只能投影 52 个 active canonical sector，不得保留 60 个 inactive legacy sector 节点或指向它们的 active 关系

## MODIFIED Requirements

### Requirement: 实体节点图谱投影
系统 SHALL 只将 PostgreSQL 当前 `status='active'` 的 `entity_nodes` 投影为 Neo4j 当前查询视图；inactive 或 merged 实体保留在 PostgreSQL 事实源和审计链中，不进入重建后的当前投影。

#### Scenario: 仅投影 active 实体节点
- **WHEN** 图谱投影 source 同时包含 active、inactive 或 merged 状态的任意实体类型
- **THEN** repository 和 projector 必须只把 active 实体计入 source rows 并写入统一 `Entity` 标签和指定 `projection_namespace`

#### Scenario: 不投影 inactive legacy sector
- **WHEN** convergence 将 60 个 legacy sector 标记为 inactive
- **THEN** 重建后的 Neo4j 不得包含这些 legacy sector，且不得通过 sector 特例绕过适用于所有实体类型的 active 状态边界

### Requirement: 实体关系图谱投影
系统 SHALL 只投影 PostgreSQL 当前 `status='active'` 且起点、终点实体均为 active 的 `entity_edges`，并保持关系类型安全、可审计和可重建。

#### Scenario: 排除 inactive 关系或端点
- **WHEN** entity edge 自身 inactive，或任一端点实体为 inactive、merged 或不存在
- **THEN** repository 不得把该 edge 计入 projection source rows，projector 也不得写入对应 Neo4j 关系
