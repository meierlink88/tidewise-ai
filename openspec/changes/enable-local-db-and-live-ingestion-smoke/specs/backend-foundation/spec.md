## ADDED Requirements

### Requirement: 真实 PostgreSQL 连接边界
系统 SHALL 在 Go 后端提供真实 PostgreSQL 连接边界，使迁移、repository 和本地 smoke 命令可以通过统一 config 和 secret 注入访问数据库。

#### Scenario: 构建本地数据库连接
- **WHEN** 开发者以 `APP_ENV=local` 加载后端配置并提供数据库密码或连接串
- **THEN** 后端必须能够构建 PostgreSQL 连接并完成 ping 验证，且不得从业务模块散落读取环境变量

#### Scenario: 避免提交数据库 secret
- **WHEN** 开发者查看 repo 内配置文件、本地基础设施模板或示例环境文件
- **THEN** 文件中不得包含真实数据库密码、生产连接串、token 或私有凭证

### Requirement: 后端启动迁移检查
系统 SHALL 在 Go API/BFF 启动阶段检查 PostgreSQL migration 状态，并按环境配置决定是否自动应用 pending migration。

#### Scenario: 自动应用 pending migration
- **WHEN** `migration.auto_apply=true` 且数据库存在 pending migration
- **THEN** 后端启动流程必须通过受保护的迁移执行器应用 pending migration，再继续启动服务

#### Scenario: 拒绝未知 schema 启动
- **WHEN** `migration.auto_apply=false` 且数据库存在 pending migration
- **THEN** 后端必须拒绝以未知 schema 继续提供服务，或明确返回不可就绪状态

### Requirement: 数据库迁移命令入口
系统 SHALL 提供独立数据库迁移命令，使开发者和部署流程可以不启动 API 服务也能检查或应用 migration。

#### Scenario: 执行本地建表
- **WHEN** 开发者在 local 环境运行数据库迁移命令
- **THEN** 命令必须使用 repo 内 migration 来源在 PostgreSQL 中创建事件知识相关表，并输出可审阅的迁移结果

#### Scenario: 检查迁移状态
- **WHEN** 开发者以 check-only 模式运行迁移命令
- **THEN** 命令必须报告已应用版本和 pending migration，而不是修改数据库结构
