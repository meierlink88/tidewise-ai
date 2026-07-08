## Context

当前后端已经具备 Go module、`cmd/api`、`cmd/source-ingest`、`cmd/source-seed`、`internal/ingestion`、`internal/jobs`、`internal/integrations`、`internal/sourcecatalog` 等目录。随着数据采集器开始具备独立调度器、数据源配置、connector/parser、原始数据抓取和落库能力，继续把它表达为普通 backend 模块会让边界变模糊。

我们已经确认目标不是把采集器拆成独立仓库、独立 Go module 或独立数据库，而是把它定义为 backend 内的独立可运行子系统：同一套 backend 代码、同一个 Go module、统一 migration 和共享基础层，但可以构建并运行成独立进程或容器 entrypoint。

本 change 要先完成后端子系统边界重构，为后续 `add-ingestion-scheduler` 提供稳定落点。它不实现调度器业务逻辑，也不引入新的调度表。

## Goals / Non-Goals

**Goals:**

- 建立 `backend/internal/apps/*` 子系统目录边界。
- 将小程序 API、未来管理后台 API 和采集器分别表达为 `miniappapi`、`adminapi`、`ingestion` 子系统。
- 将采集 runtime、sourcecatalog、connector、parser、health 等归属到 `internal/apps/ingestion/*`。
- 收窄 `internal/integrations` 的职责，只用于 Dify、支付、对象存储、短信、第三方登录等跨子系统共享外部平台集成。
- 保持现有命令、测试和数据库行为不变。
- 增加 package 文档或架构边界测试，使后续 agent 能明确代码归属和禁止依赖方向。
- 更新 OpenSpec 主规格中的后端边界事实。

**Non-Goals:**

- 不实现 `add-ingestion-scheduler` 的调度状态、run 记录、失败退避或持续 worker loop。
- 不新增 PostgreSQL schema。
- 不拆分 Go module、repo、数据库或独立发布版本。
- 不引入队列、事件流、分布式锁、数据库 leasing 或多实例采集调度。
- 不修改 `prototype` 和 `doc`。
- 不重写业务逻辑；本 change 只做结构迁移、命名对齐、适配和边界验证。

## Decisions

### Decision: 使用单 Go module、多可部署子系统结构

后端继续保持一个 Go module。所有可运行进程放在 `backend/cmd/*`，例如 `miniapp-api`、`admin-api`、`ingestion-scheduler`、`source-ingest`、`source-seed`、`entity-seed`、`dbmigrate`。业务子系统代码放在 `backend/internal/apps/*`。

采集器可以独立部署为 `ingestion-scheduler` 进程或容器 entrypoint，但它仍与其他后端子系统共享同一个 Go module、统一 migration、domain、repository、config 和 platform 基础设施。

备选方案是创建 `backend/ingestion` 独立工程根。该方案更像多 module 结构，但当前不需要独立版本化、独立数据库或独立发布生命周期，会提前引入构建、迁移和契约同步复杂度。

### Decision: 采集 connector/parser 归属于 ingestion 子系统

RSS、Eastmoney、RSSHub、web_fetch、local_file 等采集数据源适配器放入 `internal/apps/ingestion/connectors` 和 `internal/apps/ingestion/parsers`。它们强依赖 source catalog、source_config、限流、parser 和 raw document 语义，不应放在全局 `integrations`。

`internal/integrations` 只承载跨多个子系统复用的外部平台能力，例如 Dify 或外部 Agent 平台、支付、对象存储、短信、第三方登录和推送。

备选方案是继续让所有外部访问都放在 `internal/integrations`。该方案短期省事，但会让采集数据源 connector 与跨子系统平台集成混在一起，后续 agent 难以判断代码归属。

### Decision: 迁移采用兼容式搬迁

实现时优先建立新目录、移动或复制最小必要 package，并更新 import。为了降低风险，允许在迁移过程中保留薄适配层，但最终不应继续让新代码依赖旧 `internal/jobs` 或旧采集 connector 归属。

`cmd/source-ingest`、`cmd/source-seed`、`cmd/ingest-smoke` 等命令的外部行为保持兼容。后续可以通过单独 change 改名 `cmd/api` 为 `cmd/miniapp-api` 或引入 `cmd/admin-api`，本 change 只在不破坏现有命令的前提下建立目标目录。

备选方案是一次性重命名所有命令和移动所有包。该方案目录最干净，但会扩大改动面，与当前多个 active changes 更容易冲突。

### Decision: 通过架构边界测试约束依赖方向

新增轻量测试或静态检查，验证关键禁止依赖：

- `internal/integrations` 不得 import `internal/apps/*`。
- `internal/apps/miniappapi` 和 `internal/apps/adminapi` 不得 import `internal/apps/ingestion/connectors`。
- connector 不得直接访问 repository 或 database。
- parser 不得访问 database、环境变量或外部网络。

备选方案是只依赖 `.agents/backend-boundaries.md`。文档对 agent 很有帮助，但无法防止后续人为或 AI 误改。架构测试可以让边界变成可验证约束。

### Decision: 实体基础库 seed 归属 entityfoundation 子系统

`entity-seed` 不是数据采集器的一部分。它负责将 `backend/data/entity_foundation` 中的经济体、联盟组织、政策机构、市场、指数、板块、产业链节点、公司、证券、指标、商品、KOL 和实体关系等主数据初始化到 PostgreSQL。因此 seed 实现归入 `internal/apps/entityfoundation/seed`，命令入口继续保留在 `cmd/entity-seed`。

备选方案是把实体 seed 放入 `internal/apps/ingestion`。该方案会把“外部数据采集”和“系统基础实体主数据初始化”混在一起，后续 agent 容易误判采集器拥有实体体系，不采用。

### Decision: migration runner 归属 platform

`backend/migrations` 继续作为统一 SQL migration 文件来源。原 `internal/migrations` 是 Go 运行时执行器，不是 migration 文件目录，因此迁移到 `internal/platform/dbmigration`，避免和 SQL 目录产生语义冲突。

备选方案是保留 `internal/migrations`。该方案技术上可运行，但会让 repo 中同时存在两个 `migrations` 目录，后续 agent 和开发者容易把 DDL 文件来源与执行器代码混淆，不采用。

### Decision: 数据库连接基础设施归属 platform

原 `internal/database` 只负责 PostgreSQL 驱动注册、连接创建、连接池配置和连通性检查，属于跨子系统共享的运行平台能力。因此迁移到 `internal/platform/database`，并由命令入口、migration runner 和需要组装 repository 的进程引用。

领域数据访问不随本次迁移进入 platform。`internal/repositories` 继续表示业务 repository 边界，避免 `platform/database` 变成业务 SQL 和领域查询的堆放位置。

## Risks / Trade-offs

- [Risk] 目录迁移可能触碰多个 active changes。→ Mitigation：只做边界重构，不实现 scheduler；迁移后运行 `go test ./...` 和 OpenSpec 校验。
- [Risk] 旧包和新包短期并存导致混乱。→ Mitigation：tasks 明确最终期望和临时适配边界，完成后不让新代码继续依赖旧采集归属。
- [Risk] 架构测试过于严格影响合理复用。→ Mitigation：只检查关键禁止依赖，不检查所有 import。
- [Risk] `cmd/api` 改名可能破坏现有启动脚本。→ Mitigation：本 change 默认不强制重命名既有命令；如需改名，单独评估并保留兼容说明。
- [Risk] OpenSpec 主规格仍引用旧 `internal/ingestion`。→ Mitigation：本 change 通过 delta specs 修改主规格事实。

## Migration Plan

1. 建立 `backend/internal/apps/` 目录和各子系统 package 文档。
2. 将采集相关应用逻辑迁入或适配到 `backend/internal/apps/ingestion/*`。
3. 将采集 connector/parser 归属从全局 `integrations` 迁到 ingestion 子系统，保留必要兼容适配。
4. 更新命令入口 import，使 `cmd/source-ingest`、`cmd/source-seed`、`cmd/ingest-smoke` 通过新边界组装依赖。
5. 增加架构边界测试。
6. 更新 OpenSpec 主规格。
7. 运行 `go test ./...`、相关命令 smoke 或构建检查、`openspec validate refactor-backend-subsystem-boundaries`。

回滚策略：如果迁移出现问题，可以保留新 package 文档和 OpenSpec 设计，暂时恢复命令入口 import 到旧包；因为本 change 不修改数据库结构，不涉及数据回滚。

## Open Questions

- 是否在本 change 中重命名 `cmd/api` 为 `cmd/miniapp-api`？默认建议不重命名，避免影响已有启动方式；后续 API 子系统成型时再单独处理。
- 是否立即迁移所有 connector/parser？默认建议迁移当前已实现的采集 connector/parser，SDK worker 后续独立 change 再纳入。
