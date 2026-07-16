## ADDED Requirements

### Requirement: Reviewed-outbox import contract

系统 SHALL 接受版本化的 reviewed-outbox package，并在导入前校验 `package_id`、`review_id`、`review.package_id`、`review_decision`、review metadata、`idempotency_key` 和 payload hash；Agent 不得通过 SQL 直接写入 Tidewise。

#### Scenario: 接受契约完整的 package
- **WHEN** package 含匹配的 package/review identity、有效 review metadata、有效 document/event/source/tag DTO
- **THEN** service 生成 canonical payload hash 并进入事务导入流程

#### Scenario: 拒绝当前不兼容 outbox
- **WHEN** `event_tags=[]` 缺少必需 `news_category`、review 缺 `package_id`、或 evidence 缺 `supports_fields/source_level/evidence_hash/content_level`
- **THEN** service 返回结构化 contract error，不写任何 PostgreSQL 业务表，并报告适配要求

### Requirement: Review state mapping and validation

系统 SHALL 将 `auto_approved` 映射为 `confirmed+verified`，将 `pending_evidence` 或 `manual_review` 映射为 `candidate+unverified`；`rejected`、缺 Review、review/package 错配和未满足人工审核条件的 package MUST 被拒绝。

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

系统 SHALL 仅解析稳定 seed 的 ingestion source catalog，不动态创建 source master，并保留真实 `source_name`、`source_url`、`ingest_channel`；`event_time` MUST 来自 `occurred_at`，`first_seen_at` 来自最早 `collected_at`，`knowable_at` 来自最早 `published_at`，缺失时回退 `collected_at`。

#### Scenario: source 已 seed
- **WHEN** package 的 source catalog code/ID 命中 active 稳定 source
- **THEN** raw document 引用该 source，并保留输入的来源展示字段

#### Scenario: source 未知或停用
- **WHEN** source catalog 不存在、非 active 或仅能通过任意 source_name 动态创建
- **THEN** service 拒绝 package 且不创建 source master

### Requirement: Controlled tag assignment

系统 SHALL 以 `event_tag_defs` active rows 作为 Tag 权威，校验 `kind/id/code` 一致性、重复 assignment、confidence、assignment_reason、assign_source 和 review_status；`news_category` 必须 1..2，`index_category` 允许 0..3。

#### Scenario: 接受有效 Tag assignments
- **WHEN** Tag 为 active、kind/code/id 一致、数量合法，且 AI/rule assignment 含 confidence 与非空 assignment_reason
- **THEN** event_tag_maps 持久化 assignment provenance 与 review status

#### Scenario: 拒绝未知停用或数量非法 Tag
- **WHEN** Tag 未知/停用、kind/id/code 不一致、重复，或数量超出范围
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

系统 SHALL 保存最小 `event_import_receipts` 审计记录，包括 `idempotency_key`、`package_id`、`review_id`、`review_decision`、`payload_hash`、result IDs、`imported_at` 和 review metadata，并对 idempotency key 建立唯一约束。

#### Scenario: 记录成功结果
- **WHEN** package transaction 成功
- **THEN** receipt 与所有业务结果同事务提交，并可返回 raw/event/source/tag result IDs

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

#### Scenario: seed two Tag dimensions
- **WHEN** migration/seed package 在 PostgreSQL 执行且不存在冲突的既有主数据
- **THEN** 数据库存在 22 条 active Tag，两个 `tag_kind` 的 code/name/UUID 与 fixture 一致

#### Scenario: repeat seed
- **WHEN** 相同 seed 被重复执行
- **THEN** 既有 Tag 不重复，声明字段保持一致，未声明既有 Tag 不被删除
