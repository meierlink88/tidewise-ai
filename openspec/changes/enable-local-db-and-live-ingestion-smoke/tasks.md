## 1. 配置和数据库连接

- [x] 1.1 为数据库 config 扩展编写 Go 测试，覆盖 local/uat/prod 非敏感字段、password 环境变量注入、连接串覆盖和 secret 不落配置文件。
- [x] 1.2 实现数据库 config 扩展、连接串构建、连接池参数和 ping 边界，保持业务模块只依赖统一 config。
- [x] 1.3 更新 `backend/config/config.local.yaml`、`config.uat.yaml`、`config.prod.yaml` 的非敏感数据库字段，并保留真实 password/token 的环境变量注入方式。

## 2. 迁移执行器和启动检查

- [x] 2.1 为 migration service 编写测试，覆盖 check-only、auto apply、pending migration 报告、advisory lock 调用和错误路径。
- [x] 2.2 引入 PostgreSQL driver、goose 或等价迁移执行依赖，并实现真实 migration service。
- [x] 2.3 增加 `backend/cmd/dbmigrate` 命令，支持检查和应用 repo 内 `backend/migrations`。
- [x] 2.4 将 API 启动流程接入 migration 检查，并按 `migration.auto_apply` 决定自动执行或拒绝未知 schema 启动。

## 3. PostgreSQL repository 和稳定 UUID

- [x] 3.1 为稳定 UUID 生成编写 table-driven tests，覆盖合法 UUID 透传、非 UUID 候选 ID 映射、source/external ID/hash 幂等生成。
- [x] 3.2 实现采集源和原始文档写库所需的稳定 UUID helper。
- [x] 3.3 为 PostgreSQL repository 编写测试或显式集成测试边界，覆盖 active sources 查询、source seed/upsert、raw document 幂等写入、重复检测和状态更新。
- [x] 3.4 实现 `source_catalogs` 和 `raw_documents` 的 PostgreSQL repository 最小闭环。

## 4. 真实采集 smoke

- [ ] 4.1 为 smoke runner 编写测试，使用 `httptest` 模拟公开 RSS/Atom 来源，验证 source seed、connector/parser/job/writer 编排和结构化 report。
- [ ] 4.2 实现 smoke runner，使其可配置公开来源、最大采集条数、provider/source 过滤和输出 JSON report。
- [ ] 4.3 增加 `backend/cmd/ingest-smoke` 命令，使用真实 PostgreSQL repository 和现有 ingestion job 执行本地采集入库。
- [ ] 4.4 确认 smoke 失败时返回明确错误，不写入伪造文档，也不把失败标记为成功。

## 5. 本地基础设施和运行说明

- [ ] 5.1 增加本地 PostgreSQL 运行模板或说明，明确必需环境变量、数据库名称、用户、password 注入和不提交 `.env.local` 的规则。
- [ ] 5.2 增加 smoke 运行说明，包含迁移命令、采集 smoke 命令、复跑幂等验证和常见失败排查。
- [ ] 5.3 检查新增模板、说明和测试 fixture，确认没有真实 secret、生产连接串、个人 token 或私有凭证。

## 6. 验证和 OpenSpec

- [ ] 6.1 运行 `go test ./...`，确认默认测试不依赖真实 PostgreSQL 或真实网络。
- [ ] 6.2 在具备 local PostgreSQL 的环境下运行 migration 命令，确认关键表和 migration 版本记录真实存在。
- [ ] 6.3 在具备 local PostgreSQL 和网络访问的环境下运行真实采集 smoke，确认至少 1 条真实原始文档写入 `raw_documents`。
- [ ] 6.4 复跑真实采集 smoke，确认重复文档不会产生无意义重复记录。
- [ ] 6.5 运行 `openspec validate enable-local-db-and-live-ingestion-smoke` 和 `openspec validate --all`。
