# Apply-final Review

Review 时间：2026-07-16。基线 `origin/main@ecb7ad1`，Apply HEAD `da4b731`；merge-base 与 `origin/main` 一致，worktree 在 Review 前干净。

## Scope

- Package 1：Event `fact_payload`、evidence/Tag attribution 的 domain 与 in-memory repository contract、幂等和 Tag authority tests。
- Package 2：统一 migration `000019_add_event_fact_contract.sql`、dbmigration contract tests、local-only Apply 与 recovery/post assertions。
- 删除原 active change 中 Agent extraction、ingestion 和通用 persistence delta specs，使 change 收敛到已批准的数据库结构与相关映射代码。
- 未修改 `event_entity_links`、Agent/worker、ingestion、API DTO、实体、Neo4j、frontend、deployment、`doc/` 或 `prototype/`。

## Stateful Evidence

- Before/backup、单次 Apply、after assertions 和验证方法修正见 [R2 execution evidence](event-db-migration-r2-execution.md)。
- Apply-final fresh read-only DB：local PostgreSQL `16.14`，Goose `19/20`；counts `407/0/0/0/0/0`；目标列 `5`、validated constraints `3`、unique evidence index `1`。
- Backup SHA-256 重新核验为 `f35ee7a7588ce86acbc19cd7f256653d501daca413d9e301ce16da249b6a7768`。
- 未执行第二次 migration、restore、forward-fix、seed、回填、实体关联或 Neo4j 写入。

## Fresh Verification

- `go test ./internal/domain ./internal/repositories ./internal/apps/adminapi ./internal/platform/dbmigration ./internal/architecture -count=1`：通过。
- Package 2 完成前已按共享 migration 边界运行一次 `go test ./... -count=1`：通过。
- `openspec validate add-ai-event-extraction-pipeline --strict`：通过。
- `git diff origin/main...HEAD --check`、scoped diff 与新增 secret scan：通过。

## Decision

Apply-final 通过。实现符合已批准 scope，R2 migration 与兼容断言完整，无未解决测试失败或数据漂移。后续 Sync、Archive、Deliver、push/PR 仍须按独立生命周期授权推进。
