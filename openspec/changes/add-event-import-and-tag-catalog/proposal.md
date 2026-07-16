## Why

当前 Tidewise 没有受支持的 Event Agent reviewed-outbox CLI 入库边界：Agent 不能直接写 SQL，事件、原始文档、证据和 Tag 也无法在一个可审计事务中幂等落库；确定的 22 条 Event Tag 主数据也尚未落库。现在建立 CLI、可复用 application service、receipt 审计和稳定 Tag seed，支撑事件入库与两套 Tag 主数据这两个产品交付目标。

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
| local-event-import-schema-and-tag-seed | 2 | local | 1 | PostgreSQL migration 000020, fixed source master seed, deterministic Tag definition seed, schema/count/fixture-hash verification, and explicitly authorized import dry-run only | UAT/prod/shared, Neo4j, event_entity_links, Agent writes, migration/seed before authorization | backup | new:event-import-local-pg-preflight | counts=active_tags:22,source_master:1;hash=Tag seed fixture hash;schema=000020+receipt+tag columns/constraints/indexes | PostgreSQL identity, migration version, table/column/constraint/index inventory, raw/event/tag/receipt counts and fixed source master baseline recorded | 000020 applied, fixed source master active, 22 active Tags matching fixture, receipt constraints/indexes verified, no unexpected existing-row loss, dry-run has zero writes | Any drift, failed assertion, count/hash mismatch, conflict, timeout, or unauthorized scope stops all remaining layers and invalidates the package |

## What Changes

- 增加 `EventImportService`，接受 reviewed-outbox package，校验 Review、状态、Tag、证据和时间映射后，在一个 PostgreSQL transaction 中幂等写入 `raw_documents`、`events`、`event_sources`、`event_tag_maps` 和最小 `event_import_receipts`。
- 增加本地 `event-import` CLI：单文件或 outbox 目录输入，支持 `--dry-run`、机器 JSON、明确 exit code；本期不增加 HTTP endpoint。
- 增加 migration `000020`：固定 seed 一条 `source_catalogs` active source master（`source_config.manifest_identity`），为 `event_tag_defs` 增加精确的 `is_active`、`display_order`、`updated_at` 字段，增加 Tag assignment 审计字段（沿用已存在字段并补足约束），用固定 UUID literal + `INSERT ... ON CONFLICT` 内置 seed 22 条 Tag，创建最小 receipt 表、约束和索引；保留既有数据和 `event_entity_links`。
- 以确定性 UUID、英文稳定 `code`、中文 `name` 幂等 seed 22 个 active Tag：`news_category` 10 条、`index_category` 12 条；Tag 主数据是事件 CLI 校验和入库的必要支撑。
- 冻结 Agent source master manifest identity `tidewise:agent:event-reviewed-outbox`，按 source loader 真实规则 `NormalizeUUID(manifest_id)` 得到固定 UUID `cd209afe-2ea9-54b8-bdd7-db64eebf0d71`；`000020` 固定 seed 该 active source 并在 `source_config.manifest_identity` 保存 manifest identity；importer 固定 resolve 该 source，不要求 payload 提供数据库 source ID/code，但保留输入真实 source fields。
- 固化 Tidewise v1 reviewed-outbox 输入契约：保留真实顶层 `idempotency_key`、`raw_documents`、单个 `event`、`event_sources`、`event_tags`、`review` 结构和 Agent 原字段，只增补严格校验字段；明确 current-v0 到 required-v1 的最小差异；不修改 Agent。
- **BREAKING**：缺 Review、rejected、decision/status 不匹配、package/review 不一致、未知或停用 Tag、Tag 数量越界、证据契约缺字段、payload hash 冲突均拒绝且事务回滚；完整且映射匹配的 `pending_evidence`/`manual_review` 不需要第二次人工批准即可按候选状态导入。
- 不做独立 Tag Catalog export 产品、HTTP API、实体 linking、`event_entity_links` 写入、Neo4j 投影、采集 Agent、推理/行情预测、前端、部署、UAT/prod/shared 数据操作；不修改 `sibling agent-raw-ingestion-mvp`、Agent、`doc` 或 `prototype`。

## Capabilities

### New Capabilities

- `event-import-and-tag-catalog`: Event Agent reviewed-outbox CLI 入库、事务性幂等支撑、receipt 审计和两套 Event Tag 主数据 seed。

### Modified Capabilities

- None. 现有 `event-knowledge-schema` 的既有事实继续有效，本 change 通过新增 capability 规定导入 adapter 与增量 migration 的边界。

## Impact

- OpenSpec：新增本 change 的 proposal/design/spec/tasks；不修改主规格、doc 或 prototype。
- 后端预期影响：`backend/migrations/000020*`、`backend/internal/domain`、`backend/internal/repositories`、`backend/internal/apps/ingestion` 下的 importer/service、`backend/cmd` 下的本地 CLI、`backend/data` 下的 Tag seed，以及对应测试。
- 数据：PostgreSQL 是唯一事实源；local migration/seed 为 R2，Proposal Review 不授权执行。Neo4j、Redis、UAT/prod/shared 不受影响。
- 依赖方向：CLI → application service → domain/repositories/platform transaction；parser 只读文件且不访问数据库或网络；Agent 仍只产生 reviewed-outbox 文件。Tag seed 只作为事件 CLI 校验的必要主数据支撑，不形成独立导出产品。
