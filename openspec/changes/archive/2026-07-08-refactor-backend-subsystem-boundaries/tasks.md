## 1. 架构边界测试和目录骨架

- [x] 1.1 编写后端架构边界测试，验证 `internal/integrations` 不得 import `internal/apps/*`，API 子系统不得直接 import 采集 connector，connector/parser 不得越界访问数据库或外部环境。
- [x] 1.2 建立 `backend/internal/apps/` 目录和 `miniappapi`、`adminapi`、`ingestion` package 文档，说明各子系统职责。
- [x] 1.3 建立 `backend/internal/apps/ingestion/` 下的 `scheduler`、`runtime`、`sourcecatalog`、`connectors`、`parsers`、`health` 目录或 package 文档。

## 2. 采集子系统迁移

- [x] 2.1 为采集 runtime 边界编写测试，验证现有多来源执行、失败隔离和 provider 限流语义在新 package 下保持不变。
- [x] 2.2 将现有 `internal/jobs` 中采集执行相关逻辑迁移或适配到 `internal/apps/ingestion/runtime`，保持 `source-ingest` 行为兼容。
- [x] 2.3 为采集 source catalog 边界编写测试，验证 seed、校验、统计和 report 行为在新 package 下保持不变。
- [x] 2.4 将现有 `internal/sourcecatalog` 迁移或适配到 `internal/apps/ingestion/sourcecatalog`。
- [x] 2.5 为采集 connector/parser 注册边界编写测试，验证 RSS、RSSHub、网页、本地文件和 Eastmoney JSON parser 仍能注册和解析 fixture。
- [x] 2.6 将只服务采集链路的 connector/parser 从全局 `integrations` 迁移或适配到 `internal/apps/ingestion/connectors` 和 `internal/apps/ingestion/parsers`。

## 3. 命令入口和共享层对齐

- [x] 3.1 更新 `cmd/source-ingest`、`cmd/source-seed` 和 `cmd/ingest-smoke` 的 import 和依赖组装，使其通过新 ingestion 子系统边界运行。
- [x] 3.2 检查 `cmd/api`、`cmd/dbmigrate`、`cmd/entity-seed` 的 import，确保不会依赖采集 connector 或 parser 私有包。
- [x] 3.3 清理或保留薄兼容层，并在 package 文档中说明旧包迁移方向，避免后续新代码继续写入旧边界。

## 4. OpenSpec 和规则对齐

- [x] 4.1 更新必要的主规格或本 change delta specs，使 `backend-foundation`、`data-ingestion-layer` 和 `persistence-and-contracts` 描述与 `internal/apps/*` 边界一致。
- [x] 4.2 检查 `.agents/backend-boundaries.md` 与实际代码目录是否一致，如发现偏差先更新 artifacts 或规则再继续实现。
- [x] 4.3 在 `add-ingestion-scheduler` 的 design/tasks 中记录它依赖本 change 完成后再按新 `internal/apps/ingestion` 边界实现。

## 5. 验证

- [x] 5.1 运行后端架构边界测试，确认禁止依赖方向被自动化覆盖。
- [x] 5.2 运行 `go test ./...`，确保目录迁移不改变现有行为。
- [x] 5.3 运行 `openspec validate refactor-backend-subsystem-boundaries`。
- [x] 5.4 运行 `openspec validate --all`，确认新增 change 与现有 active changes 和主规格兼容。

## 6. 兼容层清理

- [x] 6.1 增加架构测试，验证旧 `internal/jobs`、`internal/sourcecatalog` 和采集用途的旧 `internal/integrations` 不再作为 Go package 存在。
- [x] 6.2 将 `internal/jobs` 中采集 runtime 和 smoke runner 的真实实现迁入 `internal/apps/ingestion/runtime`，并删除旧 `jobs` package。
- [x] 6.3 将 `internal/sourcecatalog` 的真实实现和测试迁入 `internal/apps/ingestion/sourcecatalog`，并删除旧 `sourcecatalog` package。
- [x] 6.4 将 `internal/integrations` 中采集 connector/parser 的真实实现和测试迁入 `internal/apps/ingestion/connectors` 与 `internal/apps/ingestion/parsers`，并删除旧采集 integrations package。
- [x] 6.5 更新 import、package 文档和架构测试，使新代码不再依赖旧兼容层。
- [x] 6.6 重新运行 `go test -count=1 ./...`、`openspec validate refactor-backend-subsystem-boundaries` 和 `openspec validate --all`。

## 7. 采集核心包收敛

- [x] 7.1 增加架构测试，验证 `internal/apps/ingestion/core` 存在，旧 `internal/ingestion` 不再作为 Go package 存在。
- [x] 7.2 将旧 `internal/ingestion` 的核心接口、注册表、限流、凭证、原始文档写入和测试迁入 `internal/apps/ingestion/core`。
- [x] 7.3 更新 `cmd/source-ingest`、采集 runtime、connector、parser 和测试 import，使它们引用 `internal/apps/ingestion/core`。
- [x] 7.4 删除旧 `internal/ingestion` package，并确认 `backend/internal` 下不再存在同名顶层目录。
- [x] 7.5 重新运行 `go test -count=1 ./...`、`openspec validate refactor-backend-subsystem-boundaries` 和 `openspec validate --all`。

## 8. 实体基础库 seed 子系统收敛

- [x] 8.1 增加架构测试，验证 `internal/apps/entityfoundation/seed` 存在，旧 `internal/entityseed` 不再作为 Go package 存在。
- [x] 8.2 将 `internal/entityseed` 的实体基础库 seed loader、service、repository 和测试迁入 `internal/apps/entityfoundation/seed`。
- [x] 8.3 更新 `cmd/entity-seed` import，使实体初始化命令通过 `entityfoundation/seed` 子系统边界运行。
- [x] 8.4 更新 `.agents/backend-boundaries.md`、本 change design 和 delta specs，明确实体基础库 seed 不归入 ingestion 子系统。

## 9. 数据库 migration runner 命名收敛

- [x] 9.1 增加架构测试，验证 `internal/platform/dbmigration` 存在，旧 `internal/migrations` 不再作为 Go package 存在。
- [x] 9.2 将 `internal/migrations` 的 Go migration runner、source、service、runtime、PostgreSQL store 和测试迁入 `internal/platform/dbmigration`。
- [x] 9.3 更新 `cmd/api` 和 `cmd/dbmigrate` import，使它们引用 `internal/platform/dbmigration`。
- [x] 9.4 更新 `.agents/backend-boundaries.md`、本 change design 和 delta specs，明确 `backend/migrations` 是 SQL 文件来源，`internal/platform/dbmigration` 是执行器代码。

## 10. 收敛后验证

- [x] 10.1 重新运行 `go test -count=1 ./...`。
- [x] 10.2 重新运行 `openspec validate refactor-backend-subsystem-boundaries`。
- [x] 10.3 重新运行 `openspec validate --all`。

## 11. 数据库连接基础设施收敛

- [x] 11.1 增加架构测试，验证 `internal/platform/database` 存在，旧 `internal/database` 不再作为 Go package 存在。
- [x] 11.2 将 `internal/database` 的 PostgreSQL 连接基础设施迁入 `internal/platform/database`。
- [x] 11.3 更新 `cmd/source-ingest`、`cmd/source-seed`、`cmd/ingest-smoke`、`cmd/entity-seed` 和 `internal/platform/dbmigration` import，使它们引用 `internal/platform/database`。
- [x] 11.4 更新 `.agents/backend-boundaries.md`、`AGENTS.md`、本 change design 和 delta specs，明确 `platform/database` 是数据库连接基础设施边界。
- [x] 11.5 重新运行 `go test -count=1 ./...`、`openspec validate refactor-backend-subsystem-boundaries` 和 `openspec validate --all`。
