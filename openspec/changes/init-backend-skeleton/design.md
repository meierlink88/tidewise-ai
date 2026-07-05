## Context

观潮家当前已经完成 Taro 跨平台小程序骨架和总体技术架构固化。主规格要求后端采用 Go + Gin API/BFF，源码位于 `backend/`，并通过统一强类型 config 支持 `local`、`uat`、`prod` 三类环境。当前仓库尚无后端代码，因此需要先建立一个小而稳的后端基线，避免后续业务 API、Agent 平台回调、数据访问和异步任务在没有边界的情况下分散落地。

本 change 只影响 `tidewise-ai` 源码工程内的 `backend/`、根工程脚本和 OpenSpec artifacts，不修改 `../doc` 或 `../prototype`。

## Goals / Non-Goals

**Goals:**

- 创建可编译、可测试、可本地启动的 Go + Gin 后端骨架。
- 建立 `backend/cmd/api`、`backend/internal/config`、`backend/internal/http` 等基础分层。
- 提供 `/healthz` 和 `/readyz` 最小健康检查，便于本地、UAT 和生产部署探针验证。
- 提供 `backend/config/config.local.yaml`、`config.uat.yaml`、`config.prod.yaml` 和 `.env.example`，只保存非敏感配置或占位说明。
- 建立统一强类型 config，集中加载 `APP_ENV`、配置文件和敏感环境变量。
- 为后续 API 契约、Agent 平台集成、回调处理、数据访问和异步任务预留清晰目录边界。

**Non-Goals:**

- 不实现真实事件、报告、订阅、支付、认证、图谱、RAG 或 Agent 平台业务 API。
- 不连接 PostgreSQL、Neo4j、Redis、向量库、队列或外部 Agent 平台。
- 不提交任何真实密钥、token、数据库密码、Agent 平台 API key、支付密钥或生产连接串。
- 不修改 Taro 小程序页面功能，不把后端实现细节暴露给前端页面。
- 不修改 `../doc` 和 `../prototype`。

## Decisions

### Decision: 后端采用独立 Go module

`backend/` 使用独立 `go.mod`，模块名建议为 `github.com/Fission-AI/tidewise-ai/backend` 或本仓库实际远端路径对应名称。这样可以让后端依赖、编译、测试和部署与前端 Node/Taro 工具链解耦。

备选方案是把 Go module 放在 repo 根目录。该方案会让根目录同时承载 Node workspace 和 Go module，短期方便但边界较混乱，不利于后续前后端独立 CI/CD。

### Decision: API 入口放在 `backend/cmd/api`

Go 可执行入口放在 `backend/cmd/api/main.go`，应用组装逻辑保持薄入口。HTTP 路由、config、健康检查和未来领域模块放入 `backend/internal/*`，避免跨包无序依赖。

备选方案是把入口直接放在 `backend/main.go`。该方案更简单，但后续增加 worker 或 migration 命令时扩展性较弱。

### Decision: HTTP 层先提供健康检查和版本化 API 分组

本 change 只落地 `/healthz`、`/readyz` 和预留 `/api/v1` 分组。`/healthz` 用于进程存活，`/readyz` 用于配置加载和依赖就绪状态。MVP 阶段还没有数据库或外部依赖，因此 readiness 可以先验证配置对象有效。

备选方案是同时创建多个业务 endpoint。该方案会过早绑定 API 形态，容易绕过后续 API 契约 change。

### Decision: 配置文件与 secret 分离

`backend/config/config.<env>.yaml` 只保存端口、日志级别、base URL、超时、回调路径等非敏感值。数据库密码、Agent 平台 API key、JWT secret、支付密钥等只允许通过环境变量或部署平台 secret 注入。`backend/internal/config` 输出统一 config 对象，并在启动阶段校验必填字段。

备选方案是所有配置都用环境变量。该方案部署简单，但配置结构分散，AI 编程和代码审查时更容易遗漏字段。另一个备选方案是所有配置都写入 YAML，该方案容易误提交 secret。

### Decision: 先保留 integration/repository/job 目录边界，不实现真实能力

本 change 可以创建空的或最小占位包，例如 `backend/internal/integrations`、`backend/internal/repositories`、`backend/internal/jobs`，用于表达未来扩展方向。真实 Agent 平台、数据库、队列和 worker 逻辑必须由后续独立 changes 定义。

备选方案是不创建这些边界目录。该方案更干净，但后续 AI 编程容易把外部调用或数据访问直接写进 handler。

## Risks / Trade-offs

- [Risk] 骨架过早包含过多抽象，拖慢开发节奏 → Mitigation: 本 change 只保留最小入口、config、HTTP 和健康检查，业务能力通过后续 changes 增量添加。
- [Risk] 环境配置示例被误认为可放真实 secret → Mitigation: 示例文件只使用占位值，并在 README 或配置说明中明确 secret 只能通过环境变量或 secret manager 注入。
- [Risk] `/readyz` 初期没有真实依赖检查，价值有限 → Mitigation: 先验证配置有效，后续引入数据库、缓存、队列或 Agent 平台后再扩展 readiness。
- [Risk] Go module 路径与最终 GitHub 远端不一致 → Mitigation: 采用当前仓库远端可识别路径；若远端变化，通过独立 change 或直接机械修正 module path。

## Migration Plan

1. 在 `backend/` 下创建 Go module、入口、internal 包、配置模板和测试。
2. 在根 `package.json` 增加可选后端脚本，便于从仓库根运行后端测试或格式化。
3. 本地运行 `go test ./...`、`go test ./...` 之外的配置校验，以及可用的 `go vet` 或格式化检查。
4. 后续 change 在该骨架上添加 API 契约、真实 handler、Agent 平台集成、数据访问和部署配置。

Rollback 方式为移除 `backend/` 新增代码和根脚本变更，不影响已存在 Taro 前端骨架。

## Open Questions

- Go module 路径是否最终绑定为 GitHub 仓库路径，需要在首次提交远端稳定后确认。
- 配置库使用标准库 + YAML 解析，还是引入 `viper`，实现时可在 tasks 中以最小依赖为优先。
