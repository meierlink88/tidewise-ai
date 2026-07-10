## ADDED Requirements

### Requirement: UAT CI/CD 部署边界
系统 SHALL 将 GitHub-hosted runner、GitHub Container Registry、UAT self-hosted runner 和办公室私有网络 UAT 环境定义为相互分离的部署边界。

#### Scenario: 规划 UAT 发布链路
- **WHEN** 开发者规划 backend 或 admin portal 的 UAT 发布
- **THEN** 该发布链路必须区分云端 CI 构建、镜像仓库、内网 runner 部署和 UAT 运行环境
- **AND** 不得要求 GitHub-hosted runner 直接访问办公室内网资源

#### Scenario: 区分小程序发布和服务端部署
- **WHEN** backend 和 admin portal 通过 UAT GitHub Actions 发布
- **THEN** 该流程不得发布微信或抖音小程序
- **AND** 小程序发布必须继续作为独立平台发布流程处理

#### Scenario: 使用 repo 内基础设施边界
- **WHEN** UAT 部署需要 workflow、Dockerfile、compose 模板、env example 或部署说明
- **THEN** 文件必须位于 `.github/workflows/`、`backend/`、`frontend/admin/` 或 `infra/uat/` 对应边界内
- **AND** 不得创建平行工程结构
