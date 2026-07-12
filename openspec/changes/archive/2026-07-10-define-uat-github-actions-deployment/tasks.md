## 1. 后端镜像与部署入口

- [x] 1.1 确认 `backend/cmd/admin-api`、`backend/cmd/dbmigrate` 和 UAT 配置加载路径满足镜像运行需要，不新增业务逻辑。
- [x] 1.2 新增 backend Dockerfile，构建 `admin-api` 和 `dbmigrate` 可执行文件，并使用非 root 或最小运行时镜像承载服务。
- [x] 1.3 验证 backend 镜像构建命令可以完成，并确认镜像不包含真实 secret。

## 2. 管理后台镜像

- [x] 2.1 确认 `frontend/admin` 构建脚本、测试脚本和 UAT API 入口配置方式。
- [x] 2.2 新增 admin portal Dockerfile 和静态服务配置，承载 `npm run build` 后的 `dist/` 产物。
- [x] 2.3 验证 admin portal 镜像构建命令可以完成，并确认静态产物不包含服务端 secret。

## 3. UAT 基础设施模板

- [x] 3.1 新增 `infra/uat/docker-compose.yaml`，定义 backend、admin portal、网络、端口、环境变量和健康检查。
- [x] 3.2 新增 `infra/uat/.env.example`，列出 UAT 必需变量和占位值，不写入真实密码、token 或连接串。
- [x] 3.3 新增 `infra/uat/README.md`，说明 self-hosted runner labels、GHCR 登录、UAT `.env` 准备、部署、健康检查和基于镜像 tag 的回滚步骤。

## 4. GitHub Actions 工作流

- [x] 4.1 新增 CI workflow，在 GitHub-hosted runner 上运行 backend `go test ./...`、admin portal 测试或类型检查和生产构建。
- [x] 4.2 新增镜像构建和发布步骤，将 backend 与 admin portal 镜像推送到 GHCR，并使用 commit SHA tag。
- [x] 4.3 新增 UAT deploy workflow 或 job，使用 `runs-on: [self-hosted, tidewise, uat]`，拉取镜像、执行 `dbmigrate -apply`、等待 compose healthcheck 成功后在服务容器内运行 HTTP 健康检查。
- [x] 4.4 限制 UAT 部署为手动触发或受控触发，并在 workflow 中避免让 GitHub-hosted runner 访问办公室内网资源。

## 5. 验证与收尾

- [x] 5.1 运行 `openspec validate define-uat-github-actions-deployment`。
- [x] 5.2 运行 backend Go 测试，至少覆盖 `backend/` 下 `go test ./...`。
- [x] 5.3 运行 admin portal 测试、类型检查或构建验证。
- [x] 5.4 对新增 workflow、Dockerfile、compose 和 env example 做 secret 扫描式检查，确认没有真实 secret。
- [x] 5.5 更新 tasks 状态，并在完成实现后运行适当的 Docker/compose 语法、healthcheck 等待或构建验证。
