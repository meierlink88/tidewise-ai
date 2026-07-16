## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review approval before Apply; Apply-final review before Sync | R1 | yes | SPEC_SEMANTICS | Review and approve OpenSpec artifacts, then implementation/test changes limited to this change; no database or graph writes |
| 2 | Separate local PostgreSQL preflight/backup authorization before migration and seed | R2 | yes | DRIFT_RECOVERY | Only local PostgreSQL migration 000020, deterministic Tag seed and import dry-run/verify package; no UAT/prod/shared, Neo4j, or business import without explicit scope |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 1 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---:|---|---:|---|---|---|---|---|---|---|---|
| local-event-import-schema-and-tag-seed | 2 | local | 1 | PostgreSQL migration 000020, receipt schema, deterministic Tag definition seed, schema/count/fixture-hash verification, and explicitly authorized import dry-run only | UAT/prod/shared, Neo4j, event_entity_links, Agent writes, migration/seed before authorization | backup | new:event-import-local-pg-preflight | counts=22 active tags;hash=Tag seed fixture hash;schema=000020+receipt+tag columns/constraints/indexes | PostgreSQL identity, migration version, table/column/constraint/index inventory, raw/event/tag/receipt counts and source master baseline recorded | 000020 applied, 22 active Tags matching fixture, receipt constraints/indexes verified, no unexpected existing-row loss, dry-run has zero writes | Any drift, failed assertion, count/hash mismatch, conflict, timeout, or unauthorized scope stops all remaining layers and invalidates the package |

## 1. Event import and Tag seed implementation Package

- [ ] 1.1 保留真实 reviewed-outbox 顶层 `idempotency_key/raw_documents/event/event_sources/event_tags/review` 和单 Event 结构，定义 current-v0 → required-v1 最小差异、canonical payload hash、Review 状态/时间/evidence/Tag validation，并建立一致的 v0/v1 fixture。
- [ ] 1.2 先写 domain/application tests，再实现 `EventImportService`、状态映射、source resolve、receipt same-hash replay/different-hash conflict 和全量 rollback contract。
- [ ] 1.3 为 repository 增加最小 `DBTX`/transaction runner 与 package repository；复用 RawDocument 能力但不重写既有仓储，补充 Postgres SQL/constraint contract tests。
- [ ] 1.4 增加 `000020` migration SQL：固定 source master manifest/UUID、固定 UUID literal 的 22-tag `INSERT ... ON CONFLICT` seed，以及精确 `is_active/display_order/updated_at`；覆盖 receipt schema/index/constraint、保留既有数据和 forward recovery/down 说明的静态测试。
- [ ] 1.5 增加本地 `event-import` CLI；覆盖单文件/目录、dry-run、strict JSON、exit code、stderr/stdout secret redaction，以及 22 条 Tag 主数据的 active/identity/数量校验 tests。
- [ ] 1.6 按 Proposal Review 批准后的 scoped implementation package 完成 targeted suite、architecture/contract tests、`git diff --check` 和 scope/secret 检查；Apply-final 前评估并执行受影响边界完整验证，因共享 domain/repository、migration、CLI 变更运行 `go test ./...`。

## 2. Local PostgreSQL authorization and verification Package

- [ ] 2.1 在独立 R2 授权前仅准备命令和清单；获得授权后执行 backup、identity/count/hash/schema preflight，记录 `new:event-import-local-pg-preflight` recovery baseline。
- [ ] 2.2 仅在 preflight 通过后执行一次 migration `000020` 与 22-tag idempotent seed；按 `preflight -> Write -> Query/assert` 顺序，任何漂移或断言失败立即停止。
- [ ] 2.3 验证 receipt constraints/indexes/lock semantics、active tag count=22、Tag seed fixture hash、固定 source master、既有 raw/event/tag/event_entity_links 行数保留和 dry-run 零写入；保存 recovery/after assertions，不执行 UAT/prod/shared/Neo4j 或真实业务 import。
