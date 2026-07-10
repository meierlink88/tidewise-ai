## ADDED Requirements

### Requirement: UAT GitHub Actions 发布流水线
系统 SHALL 通过 GitHub Actions 提供 UAT 发布流水线，使 backend 和 admin portal 可以从同一提交构建镜像、发布镜像，并由办公室私有网络内的 self-hosted runner 部署到 UAT 环境。

#### Scenario: 构建并发布 UAT 镜像
- **WHEN** UAT 发布流水线被手动触发或由受控分支触发
- **THEN** GitHub Actions 必须构建 backend 和 admin portal 镜像
- **AND** 必须将镜像推送到配置的容器镜像仓库
- **AND** 镜像 tag 必须能追溯到触发发布的 Git commit

#### Scenario: 通过 UAT self-hosted runner 部署
- **WHEN** 镜像构建和发布成功
- **THEN** 部署 job 必须运行在带有 `self-hosted`、`tidewise`、`uat` labels 的 runner 上
- **AND** 部署 job 必须在办公室私有网络内拉取镜像并启动 UAT 服务

#### Scenario: 云端 runner 不直接访问 UAT 内网
- **WHEN** GitHub-hosted runner 执行 CI 或镜像构建 job
- **THEN** 该 job 不得要求直接访问办公室内网数据库、Redis、UAT 服务或部署服务器

### Requirement: UAT 部署前验证
系统 SHALL 在 UAT 部署前执行轻量自动化验证，确保 backend 和 admin portal 在干净 CI 环境中可测试、可构建。

#### Scenario: 后端 CI 验证
- **WHEN** UAT 发布流水线执行 backend CI
- **THEN** workflow 必须在 `backend/` 下运行 Go 自动化测试
- **AND** 测试不得依赖真实外部 API key、真实办公室数据库或生产服务

#### Scenario: 管理后台 CI 验证
- **WHEN** UAT 发布流水线执行 admin portal CI
- **THEN** workflow 必须在 `frontend/admin/` 下运行测试或类型检查
- **AND** 必须完成生产构建验证

### Requirement: UAT 部署健康检查
系统 SHALL 在 UAT 部署后执行可观察的健康检查，确认 backend 和 admin portal 已经可访问。

#### Scenario: 后端健康检查
- **WHEN** UAT backend 服务启动或更新完成
- **THEN** 部署流程必须检查 backend 的 `/healthz`
- **AND** 必须检查 backend 的 `/readyz` 或等价就绪端点

#### Scenario: 管理后台健康检查
- **WHEN** UAT admin portal 服务启动或更新完成
- **THEN** 部署流程必须检查 admin portal HTTP 入口返回成功响应

#### Scenario: 健康检查失败
- **WHEN** 任一 UAT 健康检查失败
- **THEN** GitHub Actions 部署 job 必须失败并保留可审阅日志

### Requirement: UAT secret 隔离
系统 SHALL 保证 UAT 发布过程中真实 secret 只通过 GitHub Secrets、self-hosted runner 环境或 UAT 未提交配置注入，不得进入 repo。

#### Scenario: 查看 UAT 模板
- **WHEN** 开发者查看 repo 中的 UAT workflow、compose、env example 或说明文件
- **THEN** 文件中不得包含真实数据库密码、Admin Token、GHCR token、Agent API key 或生产连接串

#### Scenario: 部署时注入 secret
- **WHEN** UAT self-hosted runner 执行部署
- **THEN** backend 必须通过环境变量或 UAT 本地未提交 `.env` 获取数据库密码和 Admin Token
- **AND** 管理后台静态产物不得包含服务端 secret

### Requirement: UAT 回滚边界
系统 SHALL 为 UAT 部署提供基于镜像 tag 的回滚边界，使失败发布可以恢复到上一组已知镜像。

#### Scenario: 使用上一版本镜像回滚
- **WHEN** UAT 发布后发现服务不可用或健康检查失败
- **THEN** 运维人员必须能够通过上一组 backend 和 admin portal 镜像 tag 恢复 UAT compose 服务

#### Scenario: 回滚不修改代码
- **WHEN** 执行 UAT 回滚
- **THEN** 回滚不得要求修改源码、migration 文件或已提交 workflow
