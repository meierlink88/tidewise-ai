## Context

当前事件 schema 已有 `raw_documents`、`events`、`event_sources`、`event_tag_defs`、`event_tag_maps`，`000019` 已冻结 `fact_payload`、evidence relation/supports fields、evidence hash 唯一性和 tag confidence。现有 `RawDocumentRepository` 的 PostgreSQL 实现直接使用 `*sql.DB`，多个 repository method 无法共享同一 transaction。已只读核对 sibling `tidewise_adapter.py` 与 `event-package-contract.md`：真实 outbox 顶层为 `idempotency_key`、`raw_documents`、单个 `event`、`event_sources`、`event_tags`、`review`；Event 原字段包含 `dedupe_key/title/factual_summary/occurred_at/fact_status/event_status/fact_payload`，source 使用 `document_id`，Review 使用 `review_id/decision/event_status/fact_status/evidence_grade/reasons/component_versions`，Tag 使用 `tag_id/tag_kind/tag_code/confidence/review_status`。current-v0 存在 `event_tags=[]`、review 缺 `package_id`、source 缺 required evidence metadata。因此 importer 必须保留原字段、只增补 required-v1 字段并 fail closed，而不是发明新 transport 或反向修改 Agent。

本 change 是 R2 基线 change：代码和测试可在 Proposal Review 后实施；local PostgreSQL 的 migration、seed、实际 import 和 verify 仍需独立授权包。本阶段不执行任何有状态操作。

## Goals / Non-Goals

**Goals:**

- 提供可复用的 `EventImportService`，本地 CLI 是本 change 唯一 adapter；HTTP 不在本 change 范围内。
- 固定 resolve source master manifest `tidewise:agent:event-reviewed-outbox`，按 source loader 真实规则 `NormalizeUUID(manifest_id)` 固定为 `cd209afe-2ea9-54b8-bdd7-db64eebf0d71`；`000020` 插入/校验该 active source，并在 `source_config.manifest_identity` 保存 manifest identity。Agent payload 不提供数据库 source ID/code；raw document 保存真实 `source_name`、`source_url`，`ingest_channel` 使用该 source master 的固定值。
- 在单 transaction 中完成 raw document、event、source evidence、Tag map 和 receipt 的幂等导入。
- 让 `auto_approved` 映射为 `confirmed+verified`，`pending_evidence/manual_review` 映射为 `candidate+unverified`；`rejected`、缺 Review、错配及契约失败均拒绝。
- 将下表两套确定的 active Tag 主数据以 migration 内固定 UUID literal + `INSERT ... ON CONFLICT (tag_kind, code) DO UPDATE` 幂等 seed 写入 PostgreSQL，供事件 CLI 做 active、数量和 identity 校验；不提供独立 Tag Catalog export。
- 提供 migration forward recovery/down 兼容策略、TDD 边界、机器 JSON/exit code 和事件 CLI 冻结命令示例。

**Non-Goals:**

- 不改 Agent、agent-raw-ingestion-mvp、已有 outbox producer、采集 connectors、推理流程、预测或投资建议表达。
- 不做 HTTP endpoint、实体识别/linking、`event_entity_links`、Neo4j、Redis、前端、部署、UAT/prod/shared、真实 local migration/seed/import。
- 不猜测未冻结 source evidence 字段；输入不足直接返回结构化 contract error。
- 不删除或重建既有数据，不回滚 `000019`，不修改 `event_entity_links`。

## Decisions

### 1. 输入以真实 reviewed-outbox 顶层结构为唯一边界

CLI 读取单个 JSON 文件或目录中的 `.json` 文件，先解析为 DTO，再由 domain validator 进行确定性校验。每个 package 只允许一个 Event；v1 保留真实顶层结构：

```json
{
  "idempotency_key": "pkg-2026-0001",
  "package_id": "pkg-2026-0001",
  "raw_documents": [{"document_id":"sha256:07d59ce...","source_name":"真实来源","source_url":"https://example.com/article","title":"...","content_text":"...","content_level":"summary","published_at":"2026-07-15T01:36:49Z","collected_at":"2026-07-16T07:03:49Z","content_hash":"cc3d..."}],
  "event": {"dedupe_key":"event:v1:...","title":"...","factual_summary":"...","occurred_at":null,"fact_status":"verified","event_status":"confirmed","fact_payload":{}},
  "event_sources": [{"document_id":"sha256:07d59ce...","evidence_excerpt":"...","source_url":"https://example.com/article","supports_fields":["title","factual_summary"],"source_level":"secondary","content_level":"summary","evidence_hash":"sha256:..."}],
  "event_tags": [{"tag_id":"b0fe1994-0db2-526c-a57f-97fa73c1b595","tag_kind":"news_category","tag_code":"geopolitics","confidence":0.98,"review_status":"approved","assignment_reason":"...","assign_source":"ai"}],
  "review": {
    "review_id": "review-0001",
    "package_id": "pkg-2026-0001",
    "decision": "auto_approved",
    "event_status": "confirmed",
    "fact_status": "verified",
    "evidence_grade": "single_source",
    "reasons": ["..."],
    "component_versions": {"review_policy":"event_review_policy.v2","reviewer":"event-reviewer.v2"}
  }
}
```

current-v0 → required-v1 的最小差异：保留 Agent 原顶层和原字段；只补顶层 `package_id`、`review.package_id`，以及 required evidence metadata `supports_fields/source_level/content_level/evidence_hash`。`factual_summary` 不改名为 `summary`；Event 不新增 transport `event_id`，DB event UUID 由 importer 对 `dedupe_key` 确定性派生；source 保留 `document_id` transport token（例如 `sha256:...`），importer 映射到 DB raw UUID；Review 保留 `decision/evidence_grade/reasons/component_versions`，不新增 `review_decision`，`reviewer` 已由 `component_versions` 表达；Tag 保留 `tag_id/tag_kind/tag_code/confidence/review_status`，required-v1 仅新增 `assignment_reason/assign_source`，不改名为 `code/name`。current-v0 的 `event_tags=[]` 仍因本 change 的 news_category 1..2 规则被拒绝。

要求：`review.package_id` 必须存在且等于顶层 `package_id`；`review_id`、decision、evidence_grade、reasons、component_versions 必须可审计；payload hash 为 canonical UTF-8 JSON SHA-256。`event_tags` 必须包含 1..2 条 `news_category`，`index_category` 允许 0..3。未知/停用 Tag、重复 `tag_kind/tag_code`、重复 event/source identity、Tag ID/kind/code 不一致均拒绝。`decision=rejected`、缺 Review 或 decision/status 不匹配拒绝；完整且映射匹配的 `pending_evidence`/`manual_review` 直接按 candidate/unverified 导入，不增加第二次人工批准条件。

### 2. 状态与时间映射冻结

- `auto_approved` → `event_status=confirmed`、`fact_status=verified`。
- `pending_evidence` 或 `manual_review` → `event_status=candidate`、`fact_status=unverified`。
- `rejected` → reject package，不写 `events`。
- Agent 不提供 `event_id`；importer 使用既有确定性 UUID helper 对 `dedupe_key` 派生 DB event UUID，重复 dedupe_key 必须命中同一 Event。
- `event_time=occurred_at`；`first_seen_at` 为文档最早 `collected_at`；`knowable_at` 为最早 `published_at`，若无 published 则回退最早 `collected_at`。
- `evidence_relation` 默认 `supports`，但 new v1 fixture 必须显式发送冻结字段；`supports_fields` 非空，`evidence_hash`、`source_level`、`content_level` 必须存在并与映射一致。

### 3. 事务与幂等

引入最小 `DBTX` 接口（`ExecContext`、`QueryContext`、`QueryRowContext`）和 transaction runner；Postgres repositories 接受 DBTX，保留现有非事务调用兼容，不重写全仓储。service 在一个 transaction 内按 source resolve → raw upsert → event upsert → event source upsert → tag map upsert → receipt insert/return 顺序执行。数据库约束是最终防线，任何 error rollback 全 package。

`event_import_receipts` 冻结为以下最小 schema：`id UUID PRIMARY KEY`；`idempotency_key TEXT NOT NULL UNIQUE`；`package_id TEXT NOT NULL`；`review_id TEXT NOT NULL`；`review_decision VARCHAR(32) NOT NULL CHECK (review_decision IN ('auto_approved','pending_evidence','manual_review'))`；`payload_hash CHAR(64) NOT NULL`；`event_id UUID NOT NULL REFERENCES events(id)`；`raw_document_ids UUID[] NOT NULL CHECK (cardinality(raw_document_ids)>=1)`；`event_source_ids UUID[] NOT NULL CHECK (cardinality(event_source_ids)>=1)`；`event_tag_map_ids UUID[] NOT NULL CHECK (cardinality(event_tag_map_ids) BETWEEN 1 AND 5)`；`review_metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(review_metadata)='object')`；`imported_at TIMESTAMPTZ NOT NULL DEFAULT now()`。一 package/一 event 使用单个 `event_id`，raw/source/tag 结果分别使用数组保存 IDs，以保留多 raw document、多证据/多 Tag 的明确形状；v1 validator 要求至少一个 raw document、至少一个 news Tag、最多五个总 Tag。`event_id` 使用 SQL FK；PostgreSQL 不对数组元素建立 FK，service 必须在同一 transaction 内逐个验证 raw/source/tag IDs 且拒绝重复或不存在的结果 ID。建立 `ux_event_import_receipts_idempotency_key`（唯一）、`idx_event_import_receipts_event_id`、`idx_event_import_receipts_package_id`、`idx_event_import_receipts_imported_at`。

并发语义：事务开始后先执行 `pg_advisory_xact_lock(hashtextextended(idempotency_key, 0))`，再 `SELECT ... FOR UPDATE` receipt；存在且 hash 相同则返回旧 result IDs，存在且 hash 不同则返回 conflict，二者都不修改旧记录；不存在才执行业务 upsert 和 receipt insert。数据库唯一约束是最终防线，唯一冲突只能按同一 key 重新读取并应用上述 replay/conflict 规则；所有业务结果与 receipt 同一 transaction 提交。

### 4. migration 与 Tag seed

`000020` 仅增量增加 `event_tag_defs.is_active`、`display_order`、`updated_at`（必要的 description 不增加），补充 assignment `assign_source`/`review_status`/confidence/reason 的一致性约束，创建 receipt 表与索引；不动 `event_entity_links`。down 采用 forward recovery：不删除已有 Tag/receipt 业务数据；若需降级，恢复 migration 前备份或由后续 reviewed forward migration 兼容，禁止 destructive down。

Tag seed 使用固定 UUID literal、英文稳定 code、中文 name，22 条全部 active。UUID literal 由固定 namespace `6ba7b810-9dad-11d1-80b4-00c04fd430c8` 与 UUIDv5 name `event_tag:<tag_kind>:<code>` 预先计算并写入 migration/fixture；Apply 不得重新决定。`000020` 执行固定 `INSERT ... ON CONFLICT (tag_kind, code) DO UPDATE SET name=EXCLUDED.name,is_active=true,display_order=EXCLUDED.display_order,updated_at=now()`，但仅在 preflight 已断言每个 tuple 不存在或既有 UUID 完全一致后执行；不同 UUID 必须停止，不能由 ON CONFLICT 静默保留旧 ID。未知既有 Tag 不删除。seed fixture 的排序、数量和 hash 只用于 migration/seed verification，不形成独立 catalog export 产品。

冻结 Tag fixture（`display_order` 从 1 开始）：

| tag_kind | display_order | id | code | 中文 name |
|---|---:|---|---|---|
| news_category | 1 | `b0fe1994-0db2-526c-a57f-97fa73c1b595` | geopolitics | 地缘政治 |
| news_category | 2 | `b1a5438f-6e81-55e7-8ecb-33230b9ae965` | macroeconomy | 宏观经济 |
| news_category | 3 | `19fb07c0-aed3-5a1a-99b4-bba004cf2d00` | monetary_policy | 货币政策 |
| news_category | 4 | `80f6cb51-38ed-5fcc-8037-3aff25d1b767` | fiscal_trade | 财政贸易 |
| news_category | 5 | `06d1e3f4-ba81-5903-80d0-daabb27421af` | usd_fx | 美元汇率 |
| news_category | 6 | `80155a2e-33a9-545a-b57e-7bb253af699d` | commodities | 大宗商品 |
| news_category | 7 | `2b775f7a-24de-5b44-9fef-dd18f7480148` | market_indices | 指数行情 |
| news_category | 8 | `79b73443-5cc4-589b-9dd0-720d2af61e14` | executive_commentary | 高层评论 |
| news_category | 9 | `7947aa41-be9c-52ea-816e-8513b6c18d7d` | capital_markets | 资本市场 |
| news_category | 10 | `22a5afc5-20ed-55ce-bf77-54c26bbcc6ea` | technology_industry | 科技产业 |
| index_category | 1 | `173cabde-c2bf-5cdc-a026-08cd52a953f0` | macro_economic_index | 宏观经济指数 |
| index_category | 2 | `71e1deff-56b8-5f70-88ae-fcd4e267c429` | inflation_price_index | 通胀物价指数 |
| index_category | 3 | `d9a25979-00e6-5fe4-8807-4ac455d275cd` | interest_credit_index | 利率与信用指数 |
| index_category | 4 | `896f457d-3c40-5bad-bb91-3c7f196287c5` | fx_index | 外汇汇率指数 |
| index_category | 5 | `87de7402-7632-5a61-8f16-1432f9112c7e` | equity_broad_index | 股票宽基指数 |
| index_category | 6 | `22bf6fe5-7b11-5e80-abfa-430713657426` | industry_theme_index | 行业主题指数 |
| index_category | 7 | `ba56c6f1-2dfb-5f4c-a769-b95570e0a830` | commodity_index | 大宗商品指数 |
| index_category | 8 | `d4616900-4234-578b-9f35-2364c1009634` | market_sentiment_index | 市场情绪指数 |
| index_category | 9 | `b67b9650-7460-5708-9c10-089d566682b0` | stock_trading_data | 个股与成交数据 |
| index_category | 10 | `4f9ffa47-39c7-5a86-90a4-5ad06d91de4b` | futures_contract | 期货合约品种 |
| index_category | 11 | `e95a831e-f852-5838-a739-dbc59726a059` | fund_etf_index | 基金与 ETF 指数 |
| index_category | 12 | `6b2cf910-6aa3-5f8d-8016-8e9c0c4a2b09` | options_derivatives | 期权与衍生品 |

### 5. 组件与依赖方向

```text
cmd/event-import
  -> apps/ingestion/eventimport (service + file adapter)
  -> domain/eventimport (DTO, validation, canonical hash, mapping)
  -> repositories (DBTX-backed package repository)
  -> platform/database transaction runner
```

`cmd` 只解析 flags、装配依赖和输出 JSON；file adapter 只读文件；domain 不访问 DB/网络；repository 不解析 JSON、不读取环境变量；Tag seed 不暴露独立运行时导出命令。

### 6. 固定 Agent source master

版本化 source catalog manifest identity 固定为 `tidewise:agent:event-reviewed-outbox`，source loader 对 manifest `id` 直接执行 `NormalizeUUID(source.ID)`，因此固定 UUID 为 `cd209afe-2ea9-54b8-bdd7-db64eebf0d71`。`000020` 使用固定 UUID literal 插入/校验该 `status=active` source master，并在 `source_config JSONB` 写入 `{"manifest_identity":"tidewise:agent:event-reviewed-outbox"}`；其 `ingest_channel=agent_reviewed_outbox`、`provider_key=tidewise`、`connector_key=local_file`、`parser_key=event_reviewed_outbox`、`source_type=event_agent_reviewed_outbox`。importer 启动时按固定 UUID 查询并拒绝缺失/停用 master。Agent payload 不提供 `source_catalog_code`、数据库 `source_id` 或 source master identity；`raw_documents` 保留 payload 的真实 `source_name`、`source_url`，`ingest_channel` 写入固定 source master 值。

R2 preflight 必须断言：固定 UUID 对应的既有 row 只能是该 manifest identity/用途；若相同 manifest identity/用途已有其他 UUID，或固定 UUID 已被其他来源占用，立即停止，不执行 migration/seed。

### 7. CLI / JSON / exit code contract

冻结命令：

```text
event-import --file ./reviewed-outbox.json --dry-run --json
event-import --dir ./reviewed-outbox --json
```

成功 JSON 冻结为稳定对象：`{"ok":true,"mode":"dry-run|import","result":{"package_count":1,"packages":[{"package_id":"...","payload_hash":"sha256:<64 lowercase hex>","receipt_id":"...","event_id":"...","raw_document_ids":[],"event_source_ids":[],"event_tag_map_ids":[],"counts":{"raw_documents":1,"events":1,"event_sources":1,"event_tags":1,"receipts":1}}]},"errors":[]}`；目录仍使用同一对象形状，仅 `package_count` 与 `packages` 长度变化。失败 JSON：`{"ok":false,"error":{"code":"...","message":"...","details":[]}}`。DB `payload_hash` 存 64 位小写 hex，不含 `sha256:`；CLI 展示可加 `sha256:` 前缀。stdout 只输出机器 JSON，stderr 仅允许非敏感诊断；不得输出 DSN、token、cookie、API key 或完整凭据。exit code：0 success、2 input/contract validation、3 conflict/idempotency、4 database/transaction failure、5 CLI/config/I/O。

payload hash canonicalization 冻结为：对象 key 按 Unicode code point 升序递归排序；数组保持输入顺序；字符串使用 UTF-8；JSON 输出不含无意义空白；数字以 `UseNumber`/等价 exact decimal 语义解析，不经过 binary float，不接受 NaN/Infinity，并用保持原始数值语义的最短十进制表示；同一输入不得因 key 顺序、空白或语言 runtime 改变 hash。

### 8. Verification boundary

Proposal checkpoint 只运行 `openspec validate ... --strict`（按 CLI 实际支持参数）、精确 task-design lint、`git diff --check`、scope/secret 检查。Apply final 运行 migration static tests、domain/repository/service/CLI/Tag-seed tests、architecture/contract tests，并因修改共享 domain/repository、migration 与 CLI 边界评估运行 `go test ./...`；真实 PostgreSQL 只在独立 R2 授权包内执行 preflight/migration/seed/verify。

## Risks / Trade-offs

- [真实 outbox 不满足 v1] → 明确拒绝并输出缺失字段清单；发布 Agent 后续适配契约，不在本 change 猜测或修改 Agent。
- [receipt 与业务写入不一致] → receipt 与所有结果写在同一 transaction，冲突先锁定同 key receipt，任何错误 rollback。
- [Tag seed 与既有空/脏数据漂移] → migration 前逐条 preflight 断言 22 个 `(tag_kind,code)` tuple 不存在或 UUID 完全一致；不同 UUID、同 UUID 不同 identity 或 fixture hash 漂移立即停止，SQL 不静默接受漂移。
- [source master 错配] → 固定 manifest identity 和 UUID，importer 只 resolve 该 active master，不从 payload 任意创建 source。
- [现有 RawDocumentRepository 难以事务化] → 最小 DBTX/transaction runner 与 package repository 适配，不大范围重写原仓储。

## Migration Plan

Proposal Review 不授权 Apply 或数据库操作。批准后，代码/test package 先完成并自测；local R2 package 另行取得明确授权后执行：backup → preflight fixed source UUID/manifest identity/用途 → preflight 22 Tag tuples/UUID/fixture hash → identity/count/hash/schema → 单次 `000020` migration（source master + Tag seed + receipt）→ schema/count/hash verify → dry-run/receipt assertions。任一断言失败立即停止，使用 backup recovery 或 reviewed forward-fix；不执行 destructive down、不触及 UAT/prod/shared 或 Neo4j。

## Open Questions

- Agent 侧何时补齐 `review.package_id` 与 event source 的 `supports_fields/source_level/content_level/evidence_hash`，由后续 Agent change 决定；本 change 在契约缺失时保持 fail closed。
- `current-v0` reviewed-outbox fixture 仍需由 Agent 后续 change 补齐 required-v1 字段；本 change 只实现严格拒绝和结构化适配错误，不修改 Agent。
