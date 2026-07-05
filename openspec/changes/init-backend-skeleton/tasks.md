## 1. Artifact Review

- [x] 1.1 审阅 `proposal.md`，确认本 change 只涉及 `backend/` 后端骨架、根工程脚本和 OpenSpec artifacts，不修改 `../doc` 或 `../prototype`。
- [x] 1.2 审阅 `design.md`，确认 Go module、`cmd/api`、`internal/config`、HTTP 健康检查、环境配置和 secret 边界符合主规格。
- [x] 1.3 审阅 `specs/backend-foundation/spec.md` 和 `specs/technical-architecture/spec.md`，确认每条 requirement 都可验证，并使用正确 scenario 格式。

## 2. Go Module and Project Layout

- [x] 2.1 创建 `backend/go.mod`，声明 Go module 和 Go 版本。
- [x] 2.2 添加 Gin、YAML 解析和必要测试依赖，并保持依赖最小化。
- [x] 2.3 创建 `backend/cmd/api`、`backend/internal/config`、`backend/internal/http`、`backend/internal/integrations`、`backend/internal/repositories`、`backend/internal/jobs` 和 `backend/config` 目录。
- [x] 2.4 在根工程脚本中增加后端测试、格式化或本地启动命令。

## 3. Configuration Foundation

- [x] 3.1 创建 `backend/config/config.local.yaml`、`config.uat.yaml` 和 `config.prod.yaml`，只包含非敏感配置和占位 URL。
- [x] 3.2 创建 `backend/config/.env.example`，列出 `APP_ENV` 和未来 secret 注入变量名，不包含真实值。
- [x] 3.3 实现 `backend/internal/config` 强类型 config 结构、环境枚举、默认值、YAML 加载和环境变量合并逻辑。
- [x] 3.4 实现配置校验，未知 `APP_ENV`、缺失必填配置或格式错误必须返回明确错误。
- [x] 3.5 添加 config 单元测试，覆盖 local/uat/prod 加载、未知环境、缺失必填配置和 secret 占位边界。

## 4. HTTP API Skeleton

- [x] 4.1 创建 `backend/cmd/api/main.go`，完成配置加载、Gin engine 初始化和服务启动。
- [x] 4.2 创建 HTTP router 构造函数，集中注册 `/healthz`、`/readyz` 和预留 `/api/v1` 分组。
- [x] 4.3 实现 `/healthz` 结构化响应，表达服务存活状态。
- [x] 4.4 实现 `/readyz` 结构化响应，表达配置加载和基础就绪状态。
- [x] 4.5 添加 HTTP handler 单元测试，覆盖健康检查状态码和响应结构。

## 5. Boundary Placeholders

- [x] 5.1 为 `integrations`、`repositories` 和 `jobs` 添加最小包边界或占位文件，避免真实业务能力直接写入入口或 handler。
- [x] 5.2 确认占位边界不实现真实 Agent 平台调用、数据库连接、Redis、队列、RAG、图谱或支付逻辑。
- [x] 5.3 确认前端 `frontend/miniapp` 不因本 change 直接依赖后端内部模块。

## 6. Validation

- [x] 6.1 运行 `go test ./...`，并修复所有失败。
- [x] 6.2 运行可用的 Go 格式化、`go vet` 或等价验证命令，并记录不可用工具。
- [x] 6.3 运行 `openspec validate init-backend-skeleton`，并修复所有验证错误。
- [x] 6.4 扫描 `backend/` 和配置文件，确认不存在真实密钥、token、数据库密码、Agent 平台凭证、支付密钥或生产连接串。
- [x] 6.5 验证本 change 没有修改 `../doc` 或 `../prototype`。
- [x] 6.6 验证根工程已有 Taro 前端脚本仍保留，且后端脚本不破坏前端 workspace。
