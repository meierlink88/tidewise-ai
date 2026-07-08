## ADDED Requirements

### Requirement: 本地持久化 smoke 链路
系统 SHALL 提供本地可重复运行的持久化 smoke 链路，覆盖数据库连接、migration、采集源 seed、真实采集、原始文档入库和幂等复跑验证。

#### Scenario: 完成本地持久化闭环
- **WHEN** 开发者按本地说明配置 PostgreSQL 并运行迁移和采集 smoke
- **THEN** 系统必须能在 local 数据库中看到采集源记录、原始文档记录和迁移版本记录

#### Scenario: 复跑 smoke
- **WHEN** 开发者在已有 smoke 数据的 local 数据库上再次运行采集 smoke
- **THEN** 系统必须保持幂等，不得因为同一来源同一文档重复创建多条事实基础记录

### Requirement: 本地基础设施配置边界
系统 SHALL 将本地 PostgreSQL 运行所需的非敏感配置和示例模板保存在 repo 内，并将真实 secret 留给环境变量或未提交文件。

#### Scenario: 查看本地配置模板
- **WHEN** 开发者查看本地数据库或 smoke 运行模板
- **THEN** 模板必须说明需要的变量名和用途，但不得包含真实密码、真实 token 或生产连接串

#### Scenario: 切换 local、uat、prod
- **WHEN** 后续环境需要执行同一套 migration
- **THEN** 系统必须通过环境配置和 secret 注入切换连接目标，而不是修改 migration 文件或业务代码
