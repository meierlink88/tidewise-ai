## Why

当前工程已经具备 Go 后端、Web 管理后台、UAT 配置模板和本地 PostgreSQL 基础设施，但还没有统一的 GitHub Actions 打包、镜像发布和 UAT 部署机制。为了让 `backend` 和 `frontend/admin` 能通过可重复、可审计的流水线进入办公室私有网络 UAT 环境，需要把 CI、镜像构建、self-hosted runner 部署、环境变量和健康检查纳入正式工程边界。

## What Changes

- 新增 GitHub Actions CI 工作流，用 GitHub-hosted runner 对后端和管理后台执行轻量自动化验证。
- 新增 backend Docker image 构建边界，打包 Go 后端可运行进程，并保留 migration 命令在部署流程中执行。
- 新增 admin portal Docker image 构建边界，将 Vite 构建产物作为静态 Web 管理后台服务。
- 新增 GitHub Container Registry 发布约定，使 UAT 部署使用明确版本镜像，而不是在 UAT 机器上临场编译。
- 新增 UAT self-hosted runner 部署工作流，由办公室私有网络内的 runner 拉取镜像、执行 migration、启动服务并进行健康检查。
- 新增 `infra/uat` 部署模板和说明，约束 UAT 环境变量、secret 注入、compose 服务边界、runner label 和手动触发方式。
- 不发布小程序，不修改 `prototype`，不搭建办公室网络/VPN，不引入 Kubernetes，不实现 prod 部署。

## Capabilities

### New Capabilities

- `uat-github-actions-deployment`: 定义 backend 和 admin portal 通过 GitHub Actions 构建镜像、发布到 GHCR，并由 UAT self-hosted runner 部署到办公室私有网络 UAT 环境的能力。

### Modified Capabilities

- `technical-architecture`: 补充 UAT CI/CD 部署边界，明确 GitHub-hosted runner、self-hosted runner、镜像仓库、UAT 环境和小程序发布之间的职责分离。
- `backend-foundation`: 补充后端在 UAT 部署流程中的镜像打包、migration 执行、secret 注入和健康检查要求。
- `admin-console`: 补充管理后台在 UAT 部署流程中的静态构建、容器化服务和后端 API 访问配置要求。

## Impact

- 影响源码区域：`.github/workflows/`、`backend/`、`frontend/admin/`、`infra/uat/`、`openspec/changes/define-uat-github-actions-deployment/`。
- 影响部署系统：GitHub Actions、GitHub Container Registry、UAT self-hosted runner、办公室私有网络 UAT 服务器或 UAT runner 机器。
- 影响配置和安全边界：新增 GitHub Secrets、UAT `.env` 模板和 runner label 约定；不得提交真实数据库密码、Admin Token、registry token 或生产连接串。
- 不影响区域：`frontend/miniapp/` 小程序发布流程、`../prototype` 原型目录、`../doc` 项目文档目录、prod 部署。
