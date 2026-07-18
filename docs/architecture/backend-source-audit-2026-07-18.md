# Backend Source Architecture Audit

## 结论

当前架构不是错误方向，而是只完成了运行边界，尚未完成源码 ownership 收敛：

- Data、Miniapp、Admin Portal 已有独立 binary、Dockerfile 和 REST 调用边界。
- Miniapp/Admin 没有直接 import Data Service 实现，运行依赖方向基本正确。
- 大部分业务实现仍位于旧 `internal/*` 路径，配置、测试和历史工具继续保护过渡结构。
- 全量 Go 测试当前通过，主要风险是长期维护成本和下一次开发继续放错位置，而不是现有测试已经失效。

本次审计未修改业务源码、数据库、migration、seed 或运行环境。

## 主要发现

### 1. 两代目录同时存在

Miniapp 和 Admin 的启动入口位于 `services/*`，但 application/transport 仍位于：

- `internal/apps/miniappapi`
- `internal/http`
- `internal/apps/adminapi`

Data 的 domain、repositories、seed、event import、source catalog 和 graph projection 仍位于：

- `internal/domain`
- `internal/repositories`
- `internal/apps/entityfoundation`
- `internal/apps/ingestion/eventimport`
- `internal/apps/ingestion/sourcecatalog`
- `internal/apps/graphprojection`

PostgreSQL、migration、Neo4j 等 Data-only adapter 位于共享 `internal/platform`。这些代码应移动，不应复制。

### 2. 配置仍是单体配置

`internal/config.Config` 同时暴露 Agent、Database、Redis、Neo4j、Migration、Object Store、Rate Limit、Security 和全部 secret。Data 当前仍用这个大配置，Miniapp/Admin 也复用其中类型和路径解析。

其中 Agent Platform、Redis、Object Store、全局 Rate Limit、JWT/Payment/Cloud secret 当前没有生产消费者，却仍是 Data 配置必填项。根 `backend/config` 与三个 service config 形成两套执行事实；service 目录又只提供 local 模板，无法完整表达独立 UAT/prod 配置。

应保留通用 YAML/环境解析机制，但三个 Service 必须定义自己的强类型配置。

### 3. 采集实现是明确死代码

`internal/apps/ingestion/connectors`、`parsers`、`core` 与 `internal/platform/promptstore` 没有任何生产 command 或 service 调用，共约 4,700 行源码和测试。它们只互相引用，README 也标记为临时保留。

已确认删除这些实现、专属测试和采集 prompt。保留 Source Catalog 主数据/API、Raw Document/Event Import API、receipt、幂等和事务能力。

### 4. Entity seed 混入大量一次性变更执行器

`internal/apps/entityfoundation/seed` 约 13,200 行，长期 seed 能力与以下一次性能力混在同一 command/package：

- first batch workbook/mapping
- sector convergence
- alliance/economy cleanup and rebuild
- frozen chain-node relation review/write contract
- Phase A change-specific preflight

其中 chain-node relation 生产代码仍内嵌已不存在的 `openspec/changes/...` 路径和 review 状态，违反生产代码不依赖历史 change artifact 的原则。

已确认删除门槛：先证明全新空库可以由当前 migration 与正式 canonical seed 完整重建，再删除一次性 flags、实现和专属测试。长期 loader、校验、幂等、事务和正式 seed 测试必须保留。

### 5. BFF application 与 transport 混合

Miniapp research package 同时包含 use case、DTO mapping、Gin route 和 handler。Admin router 同时包含 DTO、鉴权、查询解析、Data client 编排、错误映射、健康检查和已退役 scheduler endpoint。

应拆为各自 `usecase/` 与 `transport/`。已退役 scheduler 的 410 compatibility endpoint 没有 Frontend 调用，应在确认外部消费者为零后删除。

### 6. Data HTTP handler 过深

`services/data/internalapi/handler.go` 约 700 行，同时处理认证、raw import、event import、research、admin query、source metadata、DTO 和错误 envelope。它已经归 Data，但目录和文件责任不清晰。

应移动到 `services/data/transport/internalapi`，按 auth、raw import、event import、research、admin query、source metadata 和 response 拆文件。handler 只调用 use case port。

### 7. Raw Document 存在旧写路径

通用 `internal/repositories/raw_document.go` 的生产 `UpsertRawDocument` 已没有调用。当前 Raw Document 入库走 `services/data/rawimport/postgresstore`，后者提供 transaction、identity lock、receipt 和 whole-batch 原子性。

应删除旧直接写接口和专属测试，保留 Admin read query 与 raw import transaction store。测试需要数据时使用明确 fixture/helper，不应保留无生产语义的写 API 只为测试造数据。

### 8. HTTP client 基础机制重复

Miniapp/Admin 两个 Data HTTP client 各约 370 至 400 行，request ID、认证、timeout、retry、安全错误解析和 envelope 解码高度重复，业务 path/DTO 不同。

consumer-owned port、DTO 和业务方法必须继续分开。可以提取一个无业务语义的 `internal/platform/serviceclient`，只承载 HTTP 机制；若提取后接口没有明显变小，则宁可保留局部重复，避免共享业务 client。

### 9. Architecture tests 保护的是过渡结构

现有测试明确要求 `internal/apps/*`、connector/parser 和旧 platform package 存在，并使用 transitional allowlist。它们与 service-boundary、command-import 测试存在重叠。

应以最终规则替换，而不是简单删除：

- 只允许三个 Service 拥有业务代码。
- BFF 禁止 import Data implementation 和数据库 adapter。
- 不同 Service 禁止方法级调用。
- platform 禁止 import Service 业务包。
- 禁止旧 `internal/apps/domain/repositories/http/config` 复活。
- 保留 Docker、CI、OpenAPI、security、migration 和 service asset tests。

### 10. Miniapp Frontend 仍是 mock-only

Admin Frontend 正确调用 Admin Application Backend Service。Miniapp Frontend 没有调用 Miniapp Application Backend Service，当前页面通过 Frontend-owned mock 工作。

本次不接真实 API。处理规则为：

- 保留仍被页面和明确开发场景使用的 mock，移动到 `mocks/` 或 `devdata/`。
- 删除完全未使用的 `mockGraph`、market/report service/model，以及未使用 barrel files。
- `market-card`、`event-card`、`sector-card`、`insight-panel`、`confidence-bar`、`tag-list`、`empty-state` 当前没有页面调用，应结合设计确认后删除，或由页面采用，不能继续两套实现并存。
- 真实 BFF 接入单独作为产品集成任务。

### 11. UAT 编排仍是旧单体

local compose 已有 Data、Miniapp、Admin Portal 三服务；UAT compose 仍使用单一 `backend` 服务。这不阻塞本次纯源码治理，但与目标部署模型不一致，应在 UAT 重建任务中修正。

## 重复与测试数据结论

- 没有发现可以按文件重复直接批量删除的业务测试。
- 重复主要来自两套 BFF config/client/health 机制和 transitional architecture tests，应在抽取机制或迁移 ownership 时合并。
- 唯一 tracked backend testdata 是 `testdata/event-import/reviewed-outbox-v1.json`，被多个 Event Import contract tests 共享，应保留。
- 五个空 relationship JSON 内容相同但业务关系类型不同，当前是稳定 fixture 路径，不按字节重复删除。
- Miniapp `config/dev.ts` 与 `config/prod.ts` 内容相同，但属于构建工具约定入口，除非构建配置同步调整，否则保留。

## 建议执行顺序

### Package 1: 冻结最终边界

- 接受本次 Context/ADR。
- 将 architecture tests 改为最终路径和禁止依赖规则。
- 保持 API 与运行行为不变。

### Package 2: 收敛两个 Application Backend Service

- 移动并拆分 Miniapp/Admin usecase 与 transport。
- 删除 compatibility server wrapper 和 retired scheduler endpoints。
- 建立 service-owned config，提取通用 configfile/server 机制。
- 保持两个 consumer-owned Data clients 和 HTTP contract。

### Package 3: 收敛 Data Domain Service

- 按 usecase、domain、repositories、adapters、transport 移动 Data code。
- 拆分 Data handler 和 781 行通用 domain model 文件。
- 把 database、migration、graphdb 从共享 platform 移入 Data adapters。
- 删除旧 `internal/apps/domain/repositories/http/config`。

### Package 4: 删除死代码和一次性能力

- 删除 connector/parser/core/promptstore、采集 prompt 和专属测试。
- 删除旧 Raw Document 直接写路径。
- fresh database 执行 migration、canonical seed、查询断言和图投影 dry-run。
- 验证通过后删除历史 entity-seed modes 与专属测试。

### Package 5: Frontend 死代码清理

- 将仍使用的 mock 收敛到 Frontend-owned mock 目录。
- 删除无调用 service/model/mock/component/barrel。
- 不接入真实 API，不改变当前页面数据来源。

### Package 6: 最终验证

- 三个 Backend Service 分别 build/test。
- 全量 Go tests、Frontend tests、OpenAPI drift 和 architecture tests 通过。
- 三个 Docker image 独立构建。
- local compose 只读 smoke 验证服务间 REST 依赖。
- 输出 UAT compose 重建的独立待办，不在本次执行部署。

## 完成标准

- `src/backend/internal` 只保留没有业务语义的 `platform`。
- 三个 Backend Service 的业务源码全部位于自己的 service 目录。
- Frontend 不直接访问 Domain Service。
- 不同 Backend Service 之间没有 Go implementation import，只通过 REST contract。
- BFF 配置不包含 PostgreSQL、migration 或 Neo4j 字段。
- Data 配置不再要求没有消费者的 Agent/Redis/Object Store/JWT/Payment/Cloud 字段。
- Tidewise 仓库不存在无运行入口的采集实现。
- fresh database 能由 migration 和 canonical seed 重建。
- transitional architecture tests、compatibility paths 和历史 OpenSpec runtime reference 全部消失。
- 外部 API、数据库 schema 和现有业务数据不因目录治理发生变化。

## 实施结果

Issue #40 已完成以下源码治理：

- Miniapp、Admin Portal 的 use case、transport 与配置已迁入各自 Service。
- Data 的 domain、use case、repository、adapter、transport 与配置已迁入 Data Service。
- `src/backend/internal` 只保留无业务语义的 platform 机制与 architecture tests。
- Miniapp/Admin 不直接 import Data 实现；服务间继续通过各自维护的 REST client 协作。
- Data HTTP handler 已按认证、导入、研究查询、管理查询、来源元数据和通用响应拆分。
- 旧采集 connector、parser、prompt、promptstore、退役 scheduler 路由及专属测试已删除。
- 旧 Raw Document 直接写入 repository 已删除，受控 import 的事务、幂等和 receipt 路径保留。
- Miniapp 仍使用的 mock 已迁入 Frontend-owned `mocks/`，无调用的组件、mock、model、service 和 barrel 已删除。
- 根 `backend/config` 已删除，三个 Service 使用自己的强类型配置；共享目录只保留配置解析与 HTTP server 等无业务机制。

两个 BFF 的 Data HTTP client 保持独立。它们共享的主要是实现细节，而业务 DTO、path、分页和错误语义不同；本次不为减少表面重复而引入共享业务 client。

### Fresh database 门槛结果

在独立的本地临时 PostgreSQL 数据库上，migration 000001 至 000022 全部成功。随后执行正式 `entity-seed` 时，现有 canonical fixture `alliance_org:opec_plus` 不满足 migration 000018 的 `chk_alliance_org_profiles_influence_scope` 约束，证明当前仓库还不能从空库完整重建现有实体基线。

因此，历史 entity-seed mode、专属校验和其中的历史 change contract 暂不删除。先修复 canonical entity fixture、补齐当前 842 个 chain node 的可执行正式基线，并再次完成 fresh database migration + seed + 查询断言后，才能移除这些工具。此次验证只使用独立临时数据库，没有修改既有 local、UAT、prod 或 shared 数据。
