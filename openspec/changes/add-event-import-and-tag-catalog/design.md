## Context

当前事件 schema 已有 `raw_documents`、`events`、`event_sources`、`event_tag_defs`、`event_tag_maps`，`000019` 已冻结 `fact_payload`、evidence relation/supports fields、evidence hash 唯一性和 tag confidence。现有 `RawDocumentRepository` 的 PostgreSQL 实现直接使用 `*sql.DB`，多个 repository method 无法共享同一 transaction。真实 Agent outbox 样本存在三项兼容差异：`event_tags=[]`、`review.package_id` 缺失、`event_sources` 缺 `supports_fields/source_level/evidence_hash/content_level`（只有 `raw_documents.content_level`）。因此 importer 必须 fail closed，并发布 Tidewise v1 输入契约，而不是猜测或反向修改 Agent。

本 change 是 R2 基线 change：代码和测试可在 Proposal Review 后实施；local PostgreSQL 的 migration、seed、实际 import 和 verify 仍需独立授权包。本阶段不执行任何有状态操作。

## Goals / Non-Goals

**Goals:**

- 提供可复用的 `EventImportService`，本地 CLI 是本 change 唯一 adapter；HTTP 不在本 change 范围内。
- 以稳定 source catalog 记录解析 Agent ingestion source，不动态创建任意 source master；保留真实 `source_name`、`source_url`、`ingest_channel`。
- 在单 transaction 中完成 raw document、event、source evidence、Tag map 和 receipt 的幂等导入。
- 让 `auto_approved` 映射为 `confirmed+verified`，`pending_evidence/manual_review` 映射为 `candidate+unverified`；`rejected`、缺 Review、错配及契约失败均拒绝。
- 将两套确定的 active Tag 主数据以稳定 seed 写入 PostgreSQL，供事件 CLI 做 active、数量和 identity 校验；不提供独立 Tag Catalog export。
- 提供 migration forward recovery/down 兼容策略、TDD 边界、机器 JSON/exit code 和事件 CLI 冻结命令示例。

**Non-Goals:**

- 不改 Agent、agent-raw-ingestion-mvp、已有 outbox producer、采集 connectors、推理流程、预测或投资建议表达。
- 不做 HTTP endpoint、实体识别/linking、`event_entity_links`、Neo4j、Redis、前端、部署、UAT/prod/shared、真实 local migration/seed/import。
- 不猜测未冻结 source evidence 字段；输入不足直接返回结构化 contract error。
- 不删除或重建既有数据，不回滚 `000019`，不修改 `event_entity_links`。

## Decisions

### 1. 输入以 reviewed-outbox package 为唯一边界

CLI 读取单个 JSON 文件或目录中的 `.json` 文件，先解析为 DTO，再由 domain validator 进行确定性校验。v1 顶层字段为：

```json
{
  "package_id": "pkg-2026-0001",
  "review": {
    "review_id": "review-0001",
    "package_id": "pkg-2026-0001",
    "review_decision": "auto_approved",
    "reviewed_at": "2026-07-16T01:02:03Z",
    "reviewer": {"kind": "agent", "id": "event-agent"}
  },
  "idempotency_key": "agent:event-agent:review-0001",
  "documents": [{"source_catalog_code":"agent-ingestion-default","source_name":"...","source_url":"...","ingest_channel":"agent_reviewed_outbox","source_external_id":"...","title":"...","content_text":"...","content_level":"full","published_at":"...","collected_at":"...","content_hash":"sha256:..."}],
  "events": [{"dedupe_key":"...","title":"...","summary":"...","occurred_at":"...","fact_payload":{},"tags":[{"kind":"news_category","code":"geopolitics","confidence":0.98,"assignment_reason":"...","assign_source":"ai","review_status":"approved"}],"sources":[{"document_ref":"...","evidence_excerpt":"...","evidence_relation":"supports","supports_fields":["title","summary"],"source_level":"primary","content_level":"full","evidence_hash":"sha256:..."}]}]
}
```

要求：`review.package_id` 必须存在且等于顶层 `package_id`；`review_id`、decision、review metadata 必须可审计；payload hash 为 canonical UTF-8 JSON SHA-256。`event_tags` 缺少 `news_category` 或 1..2 条越界拒绝；`index_category` 允许 0..3。未知/停用 Tag、重复 kind/code、重复 event/source identity、ID/code/kind 不一致均拒绝。`review_decision=rejected`、缺 Review、或 pending/manual_review 未通过显式 review 不写入。

### 2. 状态与时间映射冻结

- `auto_approved` → `event_status=confirmed`、`fact_status=verified`。
- `pending_evidence` 或 `manual_review` → `event_status=candidate`、`fact_status=unverified`。
- `rejected` → reject package，不写 `events`。
- `event_time=occurred_at`；`first_seen_at` 为文档最早 `collected_at`；`knowable_at` 为最早 `published_at`，若无 published 则回退最早 `collected_at`。
- `evidence_relation` 默认 `supports`，但新 v1 输入必须显式发送冻结字段；`supports_fields` 非空，`evidence_hash`、`source_level`、`content_level` 必须存在并与映射一致。

### 3. 事务与幂等

引入最小 `DBTX` 接口（`ExecContext`、`QueryContext`、`QueryRowContext`）和 transaction runner；Postgres repositories 接受 DBTX，保留现有非事务调用兼容，不重写全仓储。service 在一个 transaction 内按 source resolve → raw upsert → event upsert → event source upsert → tag map upsert → receipt insert/return 顺序执行。数据库约束是最终防线，任何 error rollback 全 package。

`event_import_receipts` 以 `idempotency_key` 唯一；相同 key+payload_hash 返回原 receipt/result IDs，不重复写；相同 key+不同 hash 返回 conflict，且不修改旧结果。receipt 保存 `package_id`、`review_id`、`review_decision`、`payload_hash`、result IDs、`imported_at` 和 review metadata JSONB；不保存凭据。

### 4. migration 与 Tag seed

`000020` 仅增量增加 `event_tag_defs.is_active`、`display_order`、`updated_at`（必要的 description 不增加），补充 assignment `assign_source`/`review_status`/confidence/reason 的一致性约束，创建 receipt 表与索引；不动 `event_entity_links`。down 采用 forward recovery：不删除已有 Tag/receipt 业务数据；若需降级，恢复 migration 前备份或由后续 reviewed forward migration 兼容，禁止 destructive down。

Tag seed 使用固定 UUID（由 `tag_kind:code` 的版本化 UUID 派生且写入 fixture）、英文稳定 code、中文 name，22 条全部 active：`news_category` 10 条、`index_category` 12 条。upsert 只更新已声明的 name/display_order/active，不删除未知既有 Tag。seed fixture 的排序、数量和 hash 只用于 migration/seed verification，不形成独立 catalog export 产品。

### 5. 组件与依赖方向

```text
cmd/event-import
  -> apps/ingestion/eventimport (service + file adapter)
  -> domain/eventimport (DTO, validation, canonical hash, mapping)
  -> repositories (DBTX-backed package repository)
  -> platform/database transaction runner
```

`cmd` 只解析 flags、装配依赖和输出 JSON；file adapter 只读文件；domain 不访问 DB/网络；repository 不解析 JSON、不读取环境变量；Tag seed 不暴露独立运行时导出命令。

### 6. CLI / JSON / exit code contract

冻结命令：

```text
event-import --file ./reviewed-outbox.json --dry-run --json
event-import --dir ./reviewed-outbox --json
```

成功 JSON：`{"ok":true,"mode":"dry-run|import","package_id":"...","payload_hash":"sha256:...","result":{"event_ids":[],"raw_document_ids":[],"receipt_id":"..."},"errors":[]}`；失败 JSON：`{"ok":false,"error":{"code":"...","message":"...","details":[]}}`。stdout 只输出机器 JSON，stderr 仅允许非敏感诊断；不得输出 DSN、token、cookie、API key 或完整凭据。exit code：0 success、2 input/contract validation、3 conflict/idempotency、4 database/transaction failure、5 CLI/config/I/O。

### 7. Verification boundary

Proposal checkpoint 只运行 `openspec validate ... --strict`（按 CLI 实际支持参数）、精确 task-design lint、`git diff --check`、scope/secret 检查。Apply final 运行 migration static tests、domain/repository/service/CLI/Tag-seed tests、architecture/contract tests，并因修改共享 domain/repository、migration 与 CLI 边界评估运行 `go test ./...`；真实 PostgreSQL 只在独立 R2 授权包内执行 preflight/migration/seed/verify。

## Risks / Trade-offs

- [真实 outbox 不满足 v1] → 明确拒绝并输出缺失字段清单；发布 Agent 后续适配契约，不在本 change 猜测或修改 Agent。
- [receipt 与业务写入不一致] → receipt 与所有结果写在同一 transaction，冲突先查 receipt，任何错误 rollback。
- [source master 错配] → 只接受已 seed 的稳定 source catalog code/ID；不动态创建 source。
- [Tag seed 与既有空/脏数据漂移] → migration 前 preflight 记录 count/hash/schema；seed 只 upsert 22 个声明 Tag，未知既有行不删除；数量/hash/assertion 失败立即停止。
- [Tag seed 漂移] → 固定 UUID、code/name、数量、display_order 与 fixture hash；migration/seed verification 失败立即停止，不提供独立 catalog export 来扩大范围。
- [现有 RawDocumentRepository 难以事务化] → 最小 DBTX/transaction runner 与 package repository 适配，不大范围重写原仓储。

## Migration Plan

Proposal Review 不授权 Apply 或数据库操作。批准后，代码/test package 先完成并自测；local R2 package 另行取得明确授权后执行：backup → preflight identity/count/hash/schema → 单次 `000020` migration → 22-tag seed → schema/count/hash verify → dry-run/receipt assertions。任一断言失败立即停止，使用 backup recovery 或 reviewed forward-fix；不执行 destructive down、不触及 UAT/prod/shared 或 Neo4j。

## Open Questions

- Agent 侧何时补齐 `review.package_id` 与 event source 的 `supports_fields/source_level/content_level/evidence_hash`，由后续 Agent change 决定；本 change 在契约缺失时保持 fail closed。
- 稳定 ingestion source catalog 的最终 seed code/UUID 需在 Apply 前从现有 source catalog fixture 选择并写入 design/task 的实现注释；不允许运行时动态创建。
