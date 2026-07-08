## MODIFIED Requirements

### Requirement: 事件知识 PostgreSQL schema
系统 SHALL 在 PostgreSQL 中保存 MVP 阶段的实体、实体关系、采集源、原始文档、事件事实、事件证据、事件标签和事件实体关联，并将这些结构作为后续事件抽取、审核、图谱投影和 API 查询的事实基础。

#### Scenario: 创建事件知识 schema
- **WHEN** 本 change 的数据库迁移被执行
- **THEN** PostgreSQL 必须具备实体节点、实体关系、联盟组织 profile、各类既有实体 profile、采集源目录、原始文档、事件事实、事件来源证据、事件标签定义、事件标签关联和事件实体关联表

#### Scenario: 支持图谱投影
- **WHEN** 后续 change 需要构建实体图或事件图
- **THEN** 系统必须能够从 PostgreSQL 中的实体、事件、标签和关系表投影数据，而不是要求采集层直接写入图数据库

### Requirement: 实体和扩展 profile
系统 SHALL 使用统一实体节点表表达联盟组织、经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物，并使用对应 profile 表保存类型专属属性。

#### Scenario: 保存统一实体
- **WHEN** 系统保存联盟组织、经济体、公司、证券、板块、指标或政策机构等对象
- **THEN** 必须先保存统一实体节点，并通过实体类型和层级字段表达基础归属

#### Scenario: 保存类型专属属性
- **WHEN** 某类实体具备组织代码、证券代码、上市市场、注册经济体、政策领域或计价货币等专属属性
- **THEN** 必须保存到对应 profile 表，而不是把所有字段塞进统一实体节点表

## ADDED Requirements

### Requirement: 联盟组织实体
系统 SHALL 支持把 `OPEC+`、`G7`、`WTO` 等跨经济体联盟组织、国际组织或多边协调机制保存为独立实体类型，并与国家政策机构、经济体和市场实体区分。

#### Scenario: 保存联盟组织
- **WHEN** 系统初始化或保存联盟组织实体
- **THEN** 必须在 `entity_nodes` 中保存 `entity_type=alliance_org`、`layer_code=alliance` 的实体节点，并在联盟组织 profile 表中保存组织简称、组织类型、主要领域、影响范围和官网地址

#### Scenario: 区分联盟组织和政策机构
- **WHEN** 实体表示跨多个经济体的国际组织、联盟、论坛或规则协调机制
- **THEN** 系统必须使用联盟组织实体表达，而不是把该实体保存为单一经济体下的政策机构
