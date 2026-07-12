## MODIFIED Requirements

### Requirement: 事件知识 PostgreSQL schema
系统 SHALL 在 PostgreSQL 中保存 MVP 阶段的实体、可审计实体关系、采集源、原始文档、事件事实、事件证据、事件标签和事件实体关联，并将这些结构作为后续事件抽取、审核、图谱投影和 API 查询的事实基础。

#### Scenario: 创建事件知识 schema
- **WHEN** 本 change 的数据库迁移被执行
- **THEN** PostgreSQL 必须具备实体节点、实体关系、联盟组织 profile、各类既有实体 profile、采集源目录、原始文档、事件事实、事件来源证据、事件标签定义、事件标签关联和事件实体关联表

#### Scenario: 保存实体关系来源元数据
- **WHEN** 系统保存经过审阅的实体关系
- **THEN** `entity_edges` 必须保存来源名称、来源 URL 和核验时间，并继续保存证据说明、状态和更新时间

#### Scenario: 增量升级关系 schema
- **WHEN** 已有 PostgreSQL 应用实体关系来源字段 migration
- **THEN** migration 必须以增量方式增加字段，不得删除实体节点、实体 profile、采集数据、事件数据或其他现有业务数据

#### Scenario: 支持图谱投影
- **WHEN** 后续 change 需要构建实体图或事件图
- **THEN** 系统必须能够从 PostgreSQL 中的实体、事件、标签和关系表投影数据，而不是要求采集层直接写入图数据库
