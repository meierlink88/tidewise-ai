## ADDED Requirements

### Requirement: 事件知识图谱投影来源
系统 SHALL 将 PostgreSQL 中的实体、实体关系、事件、证据、标签和事件实体关联作为 Neo4j 图谱投影的事实来源。

#### Scenario: 投影实体图
- **WHEN** 系统构建 Neo4j 实体图
- **THEN** 必须从 `entity_nodes`、实体 profile 和 `entity_edges` 读取投影来源，而不是由采集层、前端或 Neo4j 手工数据直接生成

#### Scenario: 后续投影事件图
- **WHEN** 后续 change 构建事件图谱
- **THEN** 必须从 `events`、`event_sources`、事件标签和 `event_entity_links` 读取投影来源，并继续保留 PostgreSQL 事件事实和证据链

#### Scenario: 保留原始文档和事件事实分离
- **WHEN** 原始文档中的文本被用于后续事件图谱构建
- **THEN** 原始文档仍然不得直接连接实体图，必须先经过事件抽取、结构化校验和事件实体关联事实表
