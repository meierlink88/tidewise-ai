## ADDED Requirements

### Requirement: 事件知识 PostgreSQL schema
系统 SHALL 在 PostgreSQL 中保存 MVP 阶段的实体、实体关系、采集源、原始文档、事件事实、事件证据、事件标签和事件实体关联，并将这些结构作为后续事件抽取、审核、图谱投影和 API 查询的事实基础。

#### Scenario: 创建事件知识 schema
- **WHEN** 本 change 的数据库迁移被执行
- **THEN** PostgreSQL 必须具备实体节点、实体关系、各类实体 profile、采集源目录、原始文档、事件事实、事件来源证据、事件标签定义、事件标签关联和事件实体关联表

#### Scenario: 支持图谱投影
- **WHEN** 后续 change 需要构建实体图或事件图
- **THEN** 系统必须能够从 PostgreSQL 中的实体、事件、标签和关系表投影数据，而不是要求采集层直接写入图数据库

### Requirement: 实体和扩展 profile
系统 SHALL 使用统一实体节点表表达经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物，并使用对应 profile 表保存类型专属属性。

#### Scenario: 保存统一实体
- **WHEN** 系统保存公司、证券、板块、指标或政策机构等对象
- **THEN** 必须先保存统一实体节点，并通过实体类型和层级字段表达基础归属

#### Scenario: 保存类型专属属性
- **WHEN** 某类实体具备证券代码、上市市场、注册经济体、政策领域或计价货币等专属属性
- **THEN** 必须保存到对应 profile 表，而不是把所有字段塞进统一实体节点表

### Requirement: 原始文档和事件事实分离
系统 SHALL 将外部采集得到的原始材料保存为 `RAW_DOCUMENT`，并通过 `EVENT_SOURCE` 将原始文档作为事件事实证据，不得让原始文档直接关联实体。

#### Scenario: 采集原始文档
- **WHEN** 采集层从 RSS、HTTP API、RSSHub、网页或本地文件获得外部材料
- **THEN** 系统必须保存原始文档记录，并记录来源、外部 ID、标题、正文、原始对象 URI、发布时间、采集时间、内容哈希和处理状态

#### Scenario: 关联事件证据
- **WHEN** 后续抽取流程从原始文档生成或更新事件事实
- **THEN** 系统必须通过事件来源证据表关联事件和原始文档，并保存来源等级、证据摘录和证据哈希

#### Scenario: 避免原文直连实体
- **WHEN** 原始文档中出现公司、板块、政策机构或指标
- **THEN** 原始文档不得直接写入实体关联表，实体关联必须由后续事件抽取、结构化校验或审核流程生成

### Requirement: 采集阶段不保存推理结论
系统 SHALL 在本阶段数据库 schema 中排除事件评分、主题贡献、传导强度、预测结论、复盘验证和用户预警任务等推理或产品化结论。

#### Scenario: 写入采集结果
- **WHEN** 采集层写入原始文档或采集状态
- **THEN** 数据库不得要求采集层填写利好利空、受益承压、影响强度、传导路径、预测结论或投资建议

#### Scenario: 后续需要分析结论
- **WHEN** 后续 change 需要保存 Agent 分析、主题贡献、传导强度或复盘验证
- **THEN** 必须通过独立 OpenSpec change 定义对应 schema 和安全边界

### Requirement: schema 迁移和回滚
系统 SHALL 为事件知识 schema 提供可审阅、版本化、长期保留的数据库迁移来源，并保证 local、uat、prod 使用同一套迁移定义。

#### Scenario: 执行迁移
- **WHEN** 开发者或部署流程初始化 PostgreSQL
- **THEN** 必须能够从 repo 内的迁移来源创建事件知识 schema

#### Scenario: 保留版本化 DDL 文件
- **WHEN** 本 change 或后续 change 修改数据库表、字段、索引、约束或枚举检查
- **THEN** 必须在 repo 内追加或更新对应阶段允许修改的版本化 SQL migration 文件，并让该文件成为 schema 变化的审阅依据

#### Scenario: 增量演进已有数据
- **WHEN** 数据库已经存在业务数据且需要执行 schema 更新
- **THEN** migration 必须以增量方式演进结构，不得通过清空表、删除全库、重建全库或丢弃既有业务数据来完成升级

#### Scenario: 遵循字段映射
- **WHEN** 本 change 实现 PostgreSQL migration
- **THEN** migration 必须覆盖 `design.md` 中 `Schema field mapping` 列出的 ER 核心字段、主键和外键关系

#### Scenario: 审阅回滚策略
- **WHEN** schema 变更需要回滚或降级
- **THEN** 本 change 必须提供 down migration、兼容迁移说明或明确的回滚策略，且回滚策略不得依赖清空业务数据
