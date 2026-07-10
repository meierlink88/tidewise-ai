## 1. Migration 测试先行

- [x] 1.1 在 `backend/internal/platform/dbmigration/migrations_test.go` 增加 `entity_edges` 来源字段 migration 的静态测试，覆盖字段定义、`IF NOT EXISTS`、非破坏性 SQL 和无 secret。
- [x] 1.2 运行 dbmigration 定向测试，确认在 migration 文件加入前测试失败。

## 2. 版本化 schema 实现

- [x] 2.1 新增 `backend/migrations/000007_add_entity_edge_provenance.sql`，以增量 DDL 追加 `source_name`、`source_url` 和 `verified_at`。
- [x] 2.2 运行 dbmigration 定向测试，确认 migration 静态测试通过。

## 3. 验证与 UAT schema 发布

- [x] 3.1 运行 `go test ./...`、`openspec validate add-entity-edge-provenance-migration` 和 migration secret/破坏性语句检查。
- [ ] 3.2 通过 PR 合并后触发现有 UAT workflow，确认 `dbmigrate -apply` 应用 `000007` 且 backend/admin 健康检查通过。
- [ ] 3.3 在 UAT 只读核验 `entity_edges` 三个字段和 migration 版本存在；不在本 change 中恢复 local 业务数据。
