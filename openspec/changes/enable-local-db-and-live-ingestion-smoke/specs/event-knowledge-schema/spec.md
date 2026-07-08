## ADDED Requirements

### Requirement: 真实数据库 schema 创建验证
系统 SHALL 能够把 repo 内事件知识 migration 应用到真实 PostgreSQL，并通过可重复验证确认关键表、索引和迁移版本存在。

#### Scenario: 创建关键表
- **WHEN** 开发者对空的 local PostgreSQL 执行事件知识 migration
- **THEN** 数据库必须创建 `source_catalogs`、`raw_documents`、`events`、`event_sources`、`entity_nodes` 和实体关系相关表

#### Scenario: 记录迁移版本
- **WHEN** migration 成功执行
- **THEN** 数据库必须记录已应用 migration 版本，使后续重复执行不会重复创建表或清空数据

#### Scenario: 保留已有数据增量更新
- **WHEN** 数据库已经存在采集源或原始文档数据并再次执行迁移检查
- **THEN** 系统不得清空、重建或丢弃已有业务数据

### Requirement: 并发迁移保护
系统 SHALL 在真实 PostgreSQL migration 执行时使用 advisory lock、迁移工具锁或等价机制，避免多个服务实例同时执行 DDL。

#### Scenario: 多实例同时启动
- **WHEN** 多个后端进程同时发现 pending migration
- **THEN** 只有一个进程可以执行 DDL，其余进程必须等待、跳过或失败并给出明确错误
