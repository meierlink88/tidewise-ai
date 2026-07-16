## Why

当前 Tidewise 没有受支持的 Event Agent reviewed-outbox 导入边界：Agent 不能直接写 SQL，事件、原始文档、证据和 Tag 也无法在一个可审计事务中幂等落库。与此同时，Tag 主数据为空且缺少只读、可复现的 catalog 输出，导致下游无法稳定校验分类输入。现在建立本地 CLI 可复用 application service、receipt 审计和 Tag Catalog 契约，为未来 HTTP 复用留下稳定边界。

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
| local-event-import-schema-and-tag-seed | 2 | local | 1 | PostgreSQL migration 000020, receipt schema, Tag definition seed, schema/count/hash verification, and explicitly authorized import dry-run only | UAT/prod/shared, Neo4j, event_entity_links, Agent writes, migration/seed before authorization | backup | new:event-import-local-pg-preflight | counts=22 active tags;hash=canonical active-tag catalog hash;schema=000020+receipt+tag columns/constraints/indexes | PostgreSQL identity, migration version, table/column/constraint/index inventory, raw/event/tag/receipt counts and source catalog baseline recorded | 000020 applied, 22 active tags, deterministic catalog hash, receipt constraints/indexes verified, no unexpected existing-row loss, dry-run has zero writes | Any drift, failed assertion, hash/count mismatch, conflict, timeout, or unauthorized scope stops all remaining layers and invalidates the package |

## What Changes

- 增加 `EventImportService`，接受 reviewed-outbox package，校验 Review、状态、Tag、证据和时间映射后，在一个 PostgreSQL transaction 中幂等写入 `raw_documents`、`events`、`event_sources`、`event_tag_maps` 和最小 `event_import_receipts`。
- 增加本地 `event-import` CLI：单文件或 outbox 目录输入，支持 `--dry-run`、机器 JSON、明确 exit code；本期不增加 HTTP endpoint。
- 增加 migration `000020`：扩展 `event_tag_defs` 的 active/catalog 字段，增加 Tag assignment 审计字段（沿用已存在字段并补足约束），创建最小 receipt 表、约束和索引；保留既有数据和 `event_entity_links`。
- 以确定性 UUID、英文稳定 `code`、中文 `name` 幂等 seed 22 个 active Tag，并提供只读 Tag Catalog JSON 导出，包含 `catalog_revision`、`catalog_hash`、`generated_at`、`tags`。
- 固化 Tidewise v1 reviewed-outbox 输入契约，明确当前样本 `event_tags=[]`、`review.package_id` 缺失和 event source evidence metadata 不足时的拒绝/适配要求；不修改 Agent。
- **BREAKING**：缺 Review、rejected、pending/manual_review 未满足映射、package/review 不一致、未知或停用 Tag、Tag 数量越界、证据契约缺字段、payload hash 冲突均拒绝且事务回滚。
- 不做实体 linking、`event_entity_links` 写入、Neo4j 投影、采集 Agent、推理/行情预测、前端、部署、UAT/prod/shared 数据操作；不修改 `sibling agent-raw-ingestion-mvp`、`doc` 或 `prototype`。

## Capabilities

### New Capabilities

- `event-import-and-tag-catalog`: Event Agent reviewed-outbox 的 v1 输入契约、事务性幂等导入、receipt 审计、Tag seed 与只读 catalog 导出。

### Modified Capabilities

- None. 现有 `event-knowledge-schema` 的既有事实继续有效，本 change 通过新增 capability 规定导入 adapter 与增量 migration 的边界。

## Impact

- OpenSpec：新增本 change 的 proposal/design/spec/tasks；不修改主规格、doc 或 prototype。
- 后端预期影响：`backend/migrations/000020*`、`backend/internal/domain`、`backend/internal/repositories`、`backend/internal/apps/ingestion` 下的 importer/service、`backend/cmd` 下的本地 CLI、`backend/data` 下的 Tag seed，以及对应测试。
- 数据：PostgreSQL 是唯一事实源；local migration/seed 为 R2，Proposal Review 不授权执行。Neo4j、Redis、UAT/prod/shared 不受影响。
- 依赖方向：CLI → application service → domain/repositories/platform transaction；parser 只读文件且不访问数据库或网络；Agent 仍只产生 reviewed-outbox 文件。
