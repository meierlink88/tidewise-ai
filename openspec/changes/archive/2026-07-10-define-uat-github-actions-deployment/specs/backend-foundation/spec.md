## ADDED Requirements

### Requirement: 后端 UAT 镜像打包
系统 SHALL 为 Go 后端提供 UAT 可部署镜像，使管理后台 API 和数据库 migration 命令可以作为同一提交版本的服务端产物发布。

#### Scenario: 构建后端镜像
- **WHEN** GitHub Actions 构建 backend UAT 镜像
- **THEN** 镜像必须从 `backend/` Go module 构建后端可执行文件
- **AND** 镜像必须包含运行 `admin-api` 所需的产物
- **AND** 镜像必须包含运行数据库 migration 命令所需的产物

#### Scenario: 使用 UAT 配置启动后端
- **WHEN** UAT 环境启动 backend 服务
- **THEN** 服务必须以 `APP_ENV=uat` 加载 UAT 配置
- **AND** 必须通过环境变量或 UAT 未提交配置注入数据库密码和 Admin Token

### Requirement: UAT migration 部署执行
系统 SHALL 在 UAT backend 服务更新前通过版本化后端 migration 命令执行数据库迁移。

#### Scenario: 部署时执行 migration
- **WHEN** UAT 部署 job 更新 backend 服务
- **THEN** 部署 job 必须在启动新服务前运行后端 migration apply 命令
- **AND** migration 必须使用 repo 内版本化 migration 来源

#### Scenario: migration 失败阻止部署
- **WHEN** UAT migration 执行失败
- **THEN** 部署 job 必须失败
- **AND** 不得继续启动或替换 backend 服务

### Requirement: 后端 UAT 健康探针
系统 SHALL 在 UAT 部署流程中验证后端 HTTP 健康检查和就绪检查。

#### Scenario: 验证 UAT backend 存活
- **WHEN** backend UAT 服务启动完成
- **THEN** 部署流程必须调用 `/healthz` 并确认成功响应

#### Scenario: 验证 UAT backend 就绪
- **WHEN** backend UAT 服务启动完成
- **THEN** 部署流程必须调用 `/readyz` 或等价就绪检查并确认成功响应
