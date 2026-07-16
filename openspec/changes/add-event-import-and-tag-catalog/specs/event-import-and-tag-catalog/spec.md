## ADDED Requirements

### Requirement: Reviewed-outbox import contract

系统 SHALL 接受真实 reviewed-outbox 顶层结构：`idempotency_key`、`package_id`、`raw_documents`、单个 `event`、`event_sources`、`event_tags` 和 `review`；导入前 SHALL 校验 `package_id`、`review_id`、`review.package_id`、`review_decision`、review metadata、`idempotency_key` 和 payload hash；Agent 不得通过 SQL 直接写入 Tidewise，且 importer 不得改造为 `documents`/`events[]` 嵌套 transport。

#### Scenario: 接受契约完整的 package
- **WHEN** package 含匹配的 package/review identity、有效 review metadata、有效 document/event/source/tag DTO
- **THEN** service 生成 canonical payload hash 并进入事务导入流程

#### Scenario: 拒绝 current-v0 outbox
- **WHEN** current-v0 的 `event_tags=[]`、review 缺 `package_id`、或 evidence 缺 `supports_fields/source_level/evidence_hash/content_level`
- **THEN** service 返回结构化 contract error，不写任何 PostgreSQL 业务表，并报告适配要求

#### Scenario: 接受 required-v1 最小增补
- **WHEN** 保留 current-v0 顶层结构，只补齐顶层/package review identity、evidence metadata、Tag identity/provenance 和单个 event contract
- **THEN** service 按 required-v1 fixture 解析，不要求 Agent 提供数据库 source ID/code

### Requirement: Review state mapping and validation

系统 SHALL 将 `auto_approved` 映射为 `confirmed+verified`，将 `pending_evidence` 或 `manual_review` 映射为 `candidate+unverified`；`rejected`、缺 Review、review/package 错配和 decision/status 不匹配 MUST 被拒绝。完整 Review、映射匹配且其余 contract 通过时，不得增加第二次人工批准条件。

#### Scenario: 导入自动批准事实
- **WHEN** review decision 为 `auto_approved` 且证据与 Tag 校验通过
- **THEN** event 保存为 `event_status=confirmed`、`fact_status=verified`

#### Scenario: 导入待补证据候选
- **WHEN** review decision 为 `pending_evidence` 或 `manual_review` 且 package 满足 v1 contract
- **THEN** event 保存为 `event_status=candidate`、`fact_status=unverified`

#### Scenario: 拒绝 rejected 或缺 Review
- **WHEN** review decision 为 `rejected`，或 review 缺失/错配
- **THEN** service 返回拒绝结果且 raw/event/source/tag/receipt 不产生部分写入

### Requirement: Transactional idempotent persistence

系统 SHALL 在单一 PostgreSQL transaction 中按 source resolve、raw document upsert、event dedupe upsert、event source upsert、Tag map upsert 和 receipt 写入完成 package；任何失败 MUST 全量 rollback。

#### Scenario: 重复 package 同 hash
- **WHEN** 相同 `idempotency_key` 再次携带相同 payload hash
- **THEN** service 返回原 receipt 和相同 result IDs，不重复创建业务记录

#### Scenario: 重复 key 不同 hash
- **WHEN** 已存在的 `idempotency_key` 携带不同 payload hash
- **THEN** service 返回 conflict exit code/错误，不修改原 receipt 或业务结果

#### Scenario: 中途失败
- **WHEN** raw、event、source、Tag map 或 receipt 任一步骤失败
- **THEN** transaction rollback，数据库中不得留下该 package 的部分写入

### Requirement: Source and time mapping

系统 SHALL 固定解析 manifest identity `tidewise:agent:event-reviewed-outbox` 对应的 active source master UUID `b780800a-bd93-529c-b75a-ff2c4b16a0e5`，不要求 Agent payload 提供数据库 source ID/code，不动态创建 source master，并保留真实 `source_name`、`source_url`、`ingest_channel`；`event_time` MUST 来自 `occurred_at`，`first_seen_at` 来自最早 `collected_at`，`knowable_at` 来自最早 `published_at`，缺失时回退 `collected_at`。

#### Scenario: fixed source master 已 seed
- **WHEN** 固定 manifest identity 对应的 active source master 存在
- **THEN** raw document 引用该 source，并保留输入的来源展示字段

#### Scenario: fixed source master 未知或停用
- **WHEN** 固定 source master 不存在、非 active，或实现试图通过 payload source name 动态创建
- **THEN** service 拒绝 package 且不创建 source master

### Requirement: Controlled tag assignment

系统 SHALL 以 `event_tag_defs` active rows 作为 Tag 权威，校验 `tag_kind/tag_id/code/name` 一致性、重复 assignment、confidence、assignment_reason、assign_source 和 review_status；`news_category` 必须 1..2，`index_category` 允许 0..3。

#### Scenario: 接受有效 Tag assignments
- **WHEN** Tag 为 active、tag_kind/tag_id/code/name 一致、数量合法，且 AI/rule assignment 含 confidence 与非空 assignment_reason
- **THEN** event_tag_maps 持久化 assignment provenance 与 review status

#### Scenario: 拒绝未知停用或数量非法 Tag
- **WHEN** Tag 未知/停用、tag_kind/tag_id/code/name 不一致、重复，或数量超出范围
- **THEN** package 被拒绝且事务回滚

### Requirement: Evidence contract

系统 SHALL 对新 evidence 要求 `evidence_relation` 默认/限制为 `supports`、`contradicts` 或 `context`，并冻结 `supports_fields`、`evidence_hash`、`source_level` 与 `content_level` 的输入映射；不完整或不一致的 evidence MUST 被拒绝。

#### Scenario: 保存 supports evidence
- **WHEN** source 提供非空 supports_fields、source_level、content_level、evidence_hash 和 evidence_excerpt
- **THEN** event_sources 保存 supports evidence，重复三元组幂等

#### Scenario: 拒绝猜测字段
- **WHEN** source 仅提供当前样本的 document_id/source_url/evidence_excerpt，缺少冻结归因字段
- **THEN** importer 返回 contract error，不推断字段值、不写入 Agent 或 Tidewise 业务数据

### Requirement: Import receipt audit

系统 SHALL 保存最小 `event_import_receipts` 审计记录：`id UUID PRIMARY KEY`、`idempotency_key TEXT NOT NULL UNIQUE`、`package_id TEXT NOT NULL`、`review_id TEXT NOT NULL`、`review_decision VARCHAR(32) NOT NULL CHECK (review_decision IN ('auto_approved','pending_evidence','manual_review'))`、`payload_hash CHAR(64) NOT NULL`、`event_id UUID NOT NULL REFERENCES events(id)`、`raw_document_ids UUID[] NOT NULL CHECK (cardinality(raw_document_ids)>=1)`、`event_source_ids UUID[] NOT NULL CHECK (cardinality(event_source_ids)>=1)`、`event_tag_map_ids UUID[] NOT NULL CHECK (cardinality(event_tag_map_ids) BETWEEN 1 AND 5)`、`review_metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(review_metadata)='object')` 和 `imported_at TIMESTAMPTZ NOT NULL DEFAULT now()`；必须建立 event/package/imported_at 查询索引。一 package/一 event 时以单个 `event_id` 和三个结果 ID 数组保存 raw/source/tag IDs；数组元素由 service 在同一 transaction 内逐个验证，拒绝不存在或重复 ID。

#### Scenario: 记录成功结果
- **WHEN** package transaction 成功
- **THEN** receipt 与所有业务结果同事务提交，并可返回 raw/event/source/tag result IDs

#### Scenario: 同 key 同 hash 并发重放
- **WHEN** 并发请求使用相同 `idempotency_key` 和 `payload_hash`
- **THEN** transaction 以 `pg_advisory_xact_lock(hashtextextended(idempotency_key, 0))` 和 receipt `SELECT ... FOR UPDATE` 串行化，后到请求返回相同 receipt/result IDs，不重复写入

#### Scenario: 同 key 不同 hash 冲突
- **WHEN** 已存在 key 的请求携带不同 `payload_hash`
- **THEN** service 返回 conflict，不修改原 receipt 或任何业务结果

### Requirement: Event import CLI contract

系统 SHALL 支持单文件、outbox 目录、`--dry-run` 和机器 JSON；事件 CLI 必须使用已确定的 22 条 Tag 主数据进行 active、identity 和数量校验；exit code MUST 区分成功、输入校验、幂等冲突、数据库事务失败和 CLI/I/O 失败，且 stdout/stderr 不得打印凭据或连接串。

#### Scenario: dry-run
- **WHEN** 使用 `event-import --file <path> --dry-run --json`
- **THEN** CLI 返回将要写入的 IDs/counts/hash JSON，不执行 PostgreSQL write

#### Scenario: 失败机器输出
- **WHEN** 输入、冲突、数据库或 I/O 失败
- **THEN** CLI 输出固定 error JSON、对应非零 exit code，并隐藏 secrets

### Requirement: Deterministic Event Tag seed

系统 SHALL 将确定的 `news_category` 10 条和 `index_category` 12 条 Tag 主数据以稳定 UUID、英文 code、中文 name 幂等 seed 到 `event_tag_defs`；`news_category` assignment 必须为 1..2 条，`index_category` assignment 必须为 0..3 条。

固定 UUID namespace 为 `6ba7b810-9dad-11d1-80b4-00c04fd430c8`，UUIDv5 name 为 `event_tag:<tag_kind>:<code>`，下表是 required-v1 fixture 的完整冻结内容；`000020` 使用固定 UUID literal + `INSERT ... ON CONFLICT (tag_kind, code)`，Apply 不得临时决定。

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

#### Scenario: seed two Tag dimensions
- **WHEN** migration/seed package 在 PostgreSQL 执行且不存在冲突的既有主数据
- **THEN** 数据库存在 22 条 active Tag，两个 `tag_kind` 的 code/name/UUID 与 fixture 一致

#### Scenario: repeat seed
- **WHEN** 相同 seed 被重复执行
- **THEN** 既有 Tag 不重复，声明字段保持一致，未声明既有 Tag 不被删除
