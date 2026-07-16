## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review approval before Apply; Apply-final review before Sync | R1 | yes | SPEC_SEMANTICS | Review and approve OpenSpec artifacts, then implementation/test changes limited to this change; no database or graph writes |
| 2 | Separate local PostgreSQL preflight/backup authorization before migration and seed | R2 | yes | DRIFT_RECOVERY | Only local PostgreSQL 000020 migration/seed, synthetic repository fixture write/cleanup assertions, and import dry-run/verify; no UAT/prod/shared, Neo4j, or business import |

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
| local-event-import-schema-and-tag-seed | 2 | local | 1 | PostgreSQL 000020 migration, fixed source/Tag seed, schema/count/hash verification, synthetic repository fixture write/cleanup assertions, and authorized import dry-run only | UAT/prod/shared, Neo4j, event_entity_links, Agent writes, real business import, migration/seed/fixture write before authorization | backup | new:event-import-local-pg-preflight | counts=frozen_tags:22,source_master:formula,receipt_pre_dry_run:0;hash=Tag seed fixture hash;schema=000020+receipt+tag columns/constraints/indexes | PostgreSQL identity/version/pending set, inventory, source/Tag drift, and structured before counts recorded | Formula counts, fixed source, 22 frozen active Tags, schema, synthetic fixture cleanup, and dry-run zero writes verified | Any drift, failed assertion, count/hash mismatch, conflict, timeout, cleanup failure, or unauthorized scope stops all remaining layers and invalidates the package |

## 1. Event import and Tag seed implementation Package

- [x] 1.1 保留真实 reviewed-outbox 顶层 `idempotency_key/raw_documents/event/event_sources/event_tags/review`、单 Event 和 Agent 原字段（`factual_summary/decision/document_id/tag_code`），定义 current-v0 → required-v1 最小差异、Event UUID 派生、canonical payload hash、Review 状态/时间/evidence/Tag validation，并建立一致的 v0/v1 fixture。
- [x] 1.2 先写 domain/application tests，再实现 `EventImportService`、共享 deterministic Plan、状态映射、source resolve、receipt same-hash replay/different-hash conflict、结果 ID 非空/唯一校验和全量 rollback contract。
- [x] 1.3 为 repository 增加最小 `DBTX`/transaction runner 与 package repository；复用 RawDocument 能力但不重写既有仓储，补充 Postgres SQL/constraint contract tests。
- [x] 1.4 增加 `000020` migration SQL：固定 UUID literal 插入/校验 source master（`source_config.manifest_identity`）和 22-tag `INSERT ... ON CONFLICT` seed，以及精确 `is_active/display_order/updated_at`；覆盖 receipt schema/index/constraint、Tag UUID drift preflight、保留既有数据和 forward recovery/down 说明的静态测试。
- [x] 1.5 增加本地 `event-import` CLI；冻结 `--file/--dir` 互斥输入、覆盖 dry-run、stable result object、deterministic IDs/counts/hash、strict JSON、exit code、stderr/stdout secret redaction、DB hex/CLI `sha256:` hash 表示，以及 22 条 Tag 主数据的 active/identity/数量校验 tests。
- [x] 1.6 按 Proposal Review 批准后的 scoped implementation package 完成 targeted suite、architecture/contract tests、`git diff --check` 和 scope/secret 检查；Apply-final 前评估并执行受影响边界完整验证，因共享 domain/repository、migration、CLI 变更运行 `go test ./...`。

## 2. Local PostgreSQL authorization and verification Package

- [ ] 2.1 在独立 R2 授权前仅准备命令和清单（见 `r2-review-package.md`、`r2-preflight.sql`、`r2-baseline.sql`）；获得授权后执行 backup、dbmigrate JSON pending-set、identity/count/hash/schema preflight，记录 `new:event-import-local-pg-preflight` recovery baseline。
- [ ] 2.2 仅在 preflight 通过后执行一次 migration `000020` 与 22-tag idempotent seed；按 `preflight -> Write -> Query/assert` 顺序，任何漂移或断言失败立即停止。
- [ ] 2.3 在受控 R2 PostgreSQL 环境执行 synthetic repository fixture write/cleanup assertions：UUID[] scan/write、advisory-lock replay/conflict、transaction rollback、VerifyReceiptResults missing/cross-event、schema compatibility；再运行 `r2-postflight.sql` 和 dry-run 零写入复验。保存 recovery/after assertions，不执行 UAT/prod/shared/Neo4j 或真实业务 import。
