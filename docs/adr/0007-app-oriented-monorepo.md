---
status: accepted
date: 2026-07-25
---

# 按应用垂直组织 Tidewise AI Monorepo

## 背景与结果

Tidewise AI 已形成 Miniapp、Admin Portal 与 Data Domain Service 三个独立运行边界，
后续还会把 AgentRun 纳入同一仓库。原有 `src/frontend` 与 `src/backend` 按技术栈横向
分组，使同一应用的 Frontend、Backend、构建和交付资产分散，也让未来 AgentRun 的
ownership 难以从仓库结构直接识别。

仓库改为按应用垂直组织，取消顶层 `src/`：

```text
miniapp/
├── frontend/
└── backend/

admin-portal/
├── frontend/
└── backend/

analyse-data-service/
└── backend/

agent-run/
└── backend/       # 未来纳入，不在本次迁移中创建空目录或导入实现
```

`analyse-data-service` 是 Data Domain Service 的规范工程应用名和目录名；领域术语仍为
**Data Domain Service**，不得用工程目录名替换领域语言。为保持本次迁移的运行兼容性，
现有 Compose service、镜像和 binary 标识 `data` / `data-service` 暂不更名。

本决策取代 ADR-0001。仓库继续保留一个 repository 和一个 Go module；本次不改变
API、数据库 schema、业务数据、认证、运行时或部署拓扑。

## Owner Map

| 应用                 | 用户界面 owner          | API owner                      | 事实与持久化 owner                              | 允许的调用                                        |
| -------------------- | ----------------------- | ------------------------------ | ----------------------------------------------- | ------------------------------------------------- |
| Miniapp              | `miniapp/frontend`      | `miniapp/backend`              | BFF 不拥有 Data 事实                            | Frontend → Miniapp Backend → Data REST API        |
| Admin Portal         | `admin-portal/frontend` | `admin-portal/backend`         | BFF 不拥有 Data/AgentRun 事实                   | Frontend → Admin Backend → Data/AgentRun REST API |
| Analyse Data Service | 无                      | `analyse-data-service/backend` | 投研领域事实、PostgreSQL、migration、Neo4j 投影 | 通过版本化 Data REST API 提供能力                 |
| AgentRun             | 后续决定                | `agent-run/backend`            | Agent、Source、Artifact、Execution 与运行状态   | 通过版本化执行/发布 API 与 Data 协作              |

共仓不改变边界。不同 Backend 不得 import 彼此实现、共享 Repository/Go model/数据库
凭据或直接访问对方数据库。

## 合同与目标落位

- Provider OpenAPI 继续位于所属 Backend；HTTP path、method、DTO、status、error、
  Request ID、时间、顺序、空值、分页、鉴权、超时、重试和幂等语义保持不变。
- Miniapp 与 Admin Portal Backend 使用完整 Kratos Application Layout；
  两者分别使用 `api/miniapp/v1` 与 `api/admin/v1`，并各自拥有 `cmd/server`、
  `configs` 与 `internal/{conf,biz,data,service,server}`。
- Data 在其 Kratos 迁移任务前保留现有内部层名。
- Data-owned migration、seed、版本化 data 和数据库适配器跟随
  `analyse-data-service/backend`。
- 跨 Data、BFF 与 Miniapp 的冻结 JSON fixture 位于根 `testdata/`；它是合同测试资产，
  不是共享业务实现。
- 仓库不提供跨应用共享的运行时 Go 实现。配置、HTTP envelope、API 文档、健康检查和
  Server 构造分别由所属 Backend 持有；即使实现相似，也不得通过根级 Go package
  建立应用间源码依赖。
- 应用自身的架构测试跟随所属 Backend；跨应用依赖、仓库结构、CI/CD 和部署资产门禁
  位于 `scripts/ci/repository-contracts`，作为仓库治理工具而非运行时源码。
- `docs`、`infra`、`scripts`、`.github`、`.agents` 与 `.codex` 是仓库级治理和编排
  目录；应用专属 Dockerfile、配置和脚本仍归应用自己。

根 `go.mod` 使用 module `github.com/meierlink88/tidewise-ai`。npm 根 workspace 只协调
`miniapp/frontend` 与 `admin-portal/frontend`。由于根 `node_modules` 可能包含第三方 Go
源码，仓库级 Go 命令必须显式枚举三个 Backend 与
`scripts/ci/repository-contracts`，不使用会误扫 JavaScript 依赖的裸
`go test ./...` 或 `go vet ./...`。

## 参考与取舍

采用 Kratos 官方 Application Layout 的单应用内部职责，以及官方“大仓模式”中一个仓库
包含多个 Application 的做法。不复制 `go-kratos/examples` 按技术主题组织的目录，也不
引入其示例中的 Protobuf、gRPC、Wire、注册中心或远程配置。

Miniapp Frontend 继续采用 Taro 4 + React + TypeScript，微信优先并保留抖音兼容。
Taro 官方 examples 只作为构建和平台行为证据，不作为 Tidewise 顶层 Monorepo 模板。

不采用以下方案：

- 继续保留 `src/backend` 与 `src/frontend`；
- 创建 `apps/` 或 `services/` 二级聚合目录；
- 为旧路径保留软链接、wrapper、复制目录或永久双入口；
- 在本次目录迁移中同时拆分 Go module；
- 建立任何跨应用共享的运行时 Go package，包括所谓“无业务语义”的 platform helper。

## 安全、失败与兼容

目录迁移不得改变 Secret 来源、服务 Token、CORS、认证主体、数据库凭据或错误清洗。
Frontend 继续只持有自身 BFF 所需凭据，AgentRun Service Token 不进入浏览器。

迁移一次切换所有源码、CI/CD、Docker、Compose、开发命令与文档引用。旧镜像可以作为
运行回滚点；源码不维护旧路径兼容层。迁移失败时回退整个变更，不运行数据库 rollback，
因为本次不修改 schema 或数据。

## 验证

- 仓库治理测试验证应用目录存在、`src/` 与根 `internal/` 不存在、只有一个根
  `go.mod`，并禁止跨应用实现 import。
- Provider OpenAPI、consumer client 与冻结 fixture 合同测试保持通过。
- Go format/vet/test/race、三个 Backend binary 与三个 Backend image 构建通过。
- Miniapp/Admin test、typecheck、lint/build 与 Admin image 构建通过。
- local/UAT Compose config 通过；Miniapp→Data provider-consumer smoke 覆盖成功路径和
  Data 不可用的安全失败路径。
