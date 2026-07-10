## ADDED Requirements

### Requirement: 管理后台 UAT 静态打包
系统 SHALL 为 `frontend/admin` 提供 UAT 可部署静态 Web 产物，使管理后台可以作为独立 admin portal 服务发布。

#### Scenario: 构建管理后台产物
- **WHEN** GitHub Actions 构建 admin portal UAT 镜像
- **THEN** workflow 必须在 `frontend/admin/` 下完成依赖安装、测试或类型检查和生产构建
- **AND** 镜像必须承载构建后的静态 Web 产物

#### Scenario: 不包含服务端 secret
- **WHEN** 开发者检查 admin portal 构建产物、Dockerfile 或 UAT 配置
- **THEN** 文件中不得包含真实 Admin Token、数据库密码、Agent API key 或其他服务端 secret

### Requirement: 管理后台 UAT API 访问配置
系统 SHALL 允许 UAT admin portal 访问 UAT backend Admin API，并保持前端静态产物与服务端 secret 隔离。

#### Scenario: 配置 UAT Admin API 入口
- **WHEN** UAT admin portal 服务启动
- **THEN** 服务必须能够指向 UAT backend Admin API 入口
- **AND** 该配置不得要求把 Admin Token 写入前端源码

#### Scenario: 验证管理后台入口
- **WHEN** UAT admin portal 服务启动或更新完成
- **THEN** 部署流程必须检查 admin portal HTTP 入口返回成功响应
