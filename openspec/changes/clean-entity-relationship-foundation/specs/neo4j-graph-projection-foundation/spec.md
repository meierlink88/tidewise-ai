## ADDED Requirements

### Requirement: 空投影和已审阅关系重建
系统 SHALL 支持清空 local Neo4j 投影数据，并在关系分批审阅后仅从 PostgreSQL 当前事实重建实体图谱。

#### Scenario: 建立空 Neo4j 投影
- **WHEN** 开发者在确认 local 环境后执行关系清洗基线重置
- **THEN** 系统必须允许删除 Neo4j 中全部节点和关系数据，同时保留 database、约束、索引和连接配置

#### Scenario: PG 无关系时重建实体图
- **WHEN** PostgreSQL `entity_edges` 为空并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须只包含可投影实体节点且不包含任何实体关系

#### Scenario: 按已审阅 PG 关系重建
- **WHEN** 某一关系批次已写入 PostgreSQL 并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须移除历史投影关系并只投影当前 active `entity_edges`，不得保留 PostgreSQL 中不存在的关系
