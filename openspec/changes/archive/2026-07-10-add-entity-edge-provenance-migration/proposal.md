## Why

local PostgreSQL 已使用 `entity_edges` 的来源元数据字段保存经过核验的关系来源，而 UAT 仍停留在缺少这些字段的 schema 版本。直接将 local 数据恢复到 UAT 会丢失关系来源信息或因列不匹配失败，因此需要先以独立、可审阅的增量 migration 对齐 UAT schema。

## What Changes

- 新增版本化 PostgreSQL migration，为 `entity_edges` 追加 `source_name`、`source_url` 和可空的 `verified_at`。
- 为该 migration 增加静态自动化测试，确认字段定义、增量性和非破坏性约束。
- 修复 admin portal 日期筛选测试的时区相关硬编码预期，确保 UAT schema 发布 PR 可在 GitHub UTC CI 环境稳定验证。
- 允许现有 local、uat、prod 数据库通过同一 migration 来源对齐 schema，且不得清空或重建任何业务表。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `event-knowledge-schema`: 为实体关系事实增加最小来源与核验元数据，并规定增量迁移保留既有业务数据。

## Impact

- 影响 `backend/migrations/`、`backend/internal/platform/dbmigration/` 和 `frontend/admin/src/pages/DataIngestionCenter.test.tsx` 的 migration、静态测试与 CI 稳定性测试。
- 影响 UAT 发布流程中的既有 `dbmigrate -apply` 步骤；不修改 API、前端、实体关系 seed、Neo4j 投影、local/UAT 数据导入或 `prototype`、`doc`。
