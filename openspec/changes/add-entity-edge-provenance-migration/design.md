## Context

local PostgreSQL 已应用 `000007_add_entity_edge_provenance.sql`，`entity_edges` 包含关系来源名称、来源 URL 和核验时间；UAT 仅应用到 `000006_add_graph_projection_runs.sql`。local 数据恢复到 UAT 前必须先对齐此 schema，且不能将未完成的实体关系清理 change 一并发布。

## Goals / Non-Goals

**Goals:**

- 通过单个版本化 migration 为 `entity_edges` 增加来源与核验字段。
- 保证 local、uat、prod 重复执行 migration 时不清空关系或其他业务数据。
- 以静态 Go 测试验证字段、非破坏性 SQL 约束和 migration 文件格式。

**Non-Goals:**

- 不写入、清空或导入任何业务数据。
- 不修改实体关系 seed、关系校验、Neo4j 投影、API 或前端。
- 不合并或完成 `clean-entity-relationship-foundation` 的其他任务。

## Decisions

### 使用独立 `000007` 增量 migration

新增 `000007_add_entity_edge_provenance.sql`，以 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` 追加字段。该文件在已部署的 `000006` 之后执行，能被现有 `dbmigrate -apply` 和 UAT GitHub Actions workflow 自动识别。

不直接在 UAT PostgreSQL 手工执行 DDL，因为 repo migration 必须是 local、uat、prod 的唯一审阅来源。

### 采用向后兼容字段定义

- `source_name TEXT NOT NULL DEFAULT ''`
- `source_url TEXT NOT NULL DEFAULT ''`
- `verified_at TIMESTAMPTZ`，允许为空

默认值允许既有 `entity_edges` 行保留原值并完成升级；可空核验时间表达尚未完成核验的历史关系。字段不增加外键、非空回填或数据重写，避免扩大锁表与发布风险。

### 测试先行并复用 migration 静态测试边界

先在 `backend/internal/platform/dbmigration/migrations_test.go` 中添加失败断言，再添加 SQL migration。测试验证 migration 文件、字段片段、`IF NOT EXISTS`、禁止破坏性 DDL 和不含 secret；完成时运行 `go test ./...`。

### 以测试输入生成日期筛选预期

admin portal 的 `datetime-local` 输入会按测试运行时区转换为 ISO 时间。测试必须从同一输入值通过 `new Date(value).toISOString()` 生成断言，不能硬编码某一时区的 UTC 偏移结果。该修复只稳定测试，不改变页面或 API 的日期转换行为。

## Risks / Trade-offs

- [UAT migration 失败导致部署停止] → 既有 workflow 必须在启动新 backend 前执行 migration，失败时保持服务不替换；执行前保留 UAT PostgreSQL 备份。
- [未来未来源关系的空字符串语义不清] → 本 change 只保证 schema 兼容；关系 review 与数据质量规则仍由独立 change 定义。
- [local 与 UAT 数据导入混入 schema 发布] → 本 change 明确不执行数据导入；schema 合并和 UAT 部署成功后再单独执行备份、导出和恢复操作。

## Migration Plan

1. 合并此 change 后触发现有 UAT workflow，构建带 `000007` 的 backend 镜像。
2. workflow 的 `dbmigrate -apply` 在启动新 backend 前追加三个字段并记录版本。
3. 核验 UAT `goose_db_version` 与 `entity_edges` 列定义，再开始 local 数据恢复。
4. 若 migration 或部署失败，停止数据恢复；通过 UAT 部署前备份恢复数据库。该 migration 不提供删除列的 down 操作，避免回滚时丢失已写入的来源元数据。

## Open Questions

- 无。数据导入的备份位置、导出格式、恢复顺序和逐表行数核验将在 schema 对齐后作为独立运行操作执行。
