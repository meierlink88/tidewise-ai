## Context

当前 `000001_init_event_knowledge_schema.sql` 已建立 `events`、`event_sources`、`event_tag_defs`、`event_tag_maps`、`event_entity_links`，其中 `events` 只有标题、摘要、时间、状态、`dedupe_key` 和主证据指针；`domain.Event` 与 `ListEvents` 也只映射这些字段。`raw_documents` 是证据镜像，Event 必须通过 `event_sources` 追溯原文。只读 local 基线为 `tidewise_local`：`raw_documents=407`，其余五个相关事件/Tag/实体链接表均为 0；最新 migration 为 `000018`。

已批准语义是：Event 是可独立验证且引起现实对象或关键变量状态变化的最小事实单元；每篇 Markdown 先形成 0..N 原子 Event extraction candidate，跨文档确定性召回与模型裁决形成唯一 Event package。模型不直写 DB，确定性执行器负责证据、合并、Tag、幂等和事务。这个 change 只准备 DB/domain/repository 契约，不实现上述执行链。

## Goals / Non-Goals

**Goals:**

- 设计 `fact_payload JSONB` 的持久化、domain 和 repository 映射边界，保持既有 Event 字段兼容。
- 逐项评估证据关系、支持字段和 Tag 归因的最小增量字段。
- 保持 Tag 主数据只来自 Tidewise DB；YAML 只描述策略，不成为 Tag authority。
- 通过增量 migration contract、domain validation 和 repository fake/SQL mapping tests 固化契约。
- 为未来 extraction candidate、唯一 Event package 和确定性执行器提供不含模型依赖的写入边界。

**Non-Goals:**

- 不实现采集 Agent、Event Agent、LLM/model client、prompt、worker、job/run、重试/回放或采集后触发。
- 不新增 `schema_version`、`fact_schema_version`、`dedupe_version`、`supersedes_event_id`、`event_extraction_jobs`、`event_extraction_runs`、`event_relations`。
- 不修改或写入现有 `event_entity_links`，不实现实体匹配、实体候选、实体关联或其测试；这些内容留给未来独立 change。
- 不新增事件评分、预测、利好利空、传导强度、影响推理或直接投资建议字段。
- 不在本 change 预批准除 `fact_payload` 外的 `events` 字段；事件类型、地域等如需增加必须另列 Review 决策并提供字段语义、约束、兼容和回滚论证。
- 不改采集职责、实体主数据 seed、Neo4j、前端/API、部署或真实 PostgreSQL/Neo4j 写入。

## Decisions

### 1. 只增加事实载荷，不拆分事实 schema

`events.fact_payload JSONB NOT NULL DEFAULT '{}'::jsonb` 是当前唯一预设的新增 `events` 字段。它承载 Event-specific 原子事实键值，必须保持可审计、可 JSON 解码，并拒绝投资建议/预测类字段。MVP 不增加 schema version 字段；payload 的键策略由后续 extraction contract 另行定义。`summary`、时间、状态、`dedupe_key` 等既有列继续作为查询和兼容基础。

### 2. 关系表只接受最小审计增量，逐项 Review

以下是实现前必须逐项确认的候选映射，不是本 Proposal 的默认批准：

| Table | Candidate columns | Purpose | Required decision |
|---|---|---|---|
| `event_sources` | `evidence_relation`, `supports_fields` | 说明证据与 Event 的关系及支持的字段集合 | 是否允许空值/枚举或 JSON，哈希与唯一性如何兼容 |
| `event_tag_maps` | `confidence`, `assignment_reason` | 保存 Tag 置信度与确定性分配理由 | 数值范围、空值兼容、Tag 状态和 DB authority |

保留现有表和唯一约束：`event_sources` 仍连接 Event/raw document，`event_tag_maps` 仍连接 Event/DB Tag；`event_entity_links` 保持现有 schema 原样，不引入 `event_relations`。

### 3. 兼容、索引与回滚

未来 migration 必须使用 `ADD COLUMN IF NOT EXISTS` 或等价可重复增量语义；新增列优先 nullable 或带无损默认值，不重写既有事实。只为实际查询/唯一性证明必要的字段建立索引，避免为 JSONB 或低选择性字段过早建索引。回滚优先采用 forward compatibility（停止写新列、旧代码继续读既有列）；若必须 down migration，必须证明无数据丢失且先处理新增值，不得清空业务表。

### 4. Domain/repository 映射

`domain.Event` 增加 `FactPayload` 的明确类型边界（推荐 `map[string]any` 或等价不可变 JSON value），验证空 payload 合法、非对象/不可编码 payload 拒绝，并复制/序列化时避免共享可变引用。PostgreSQL repository 的 insert/update/scan contract 必须覆盖 payload；现有 admin list DTO 可保持兼容，除非 Review 明确要求查询返回 payload。关系候选字段在对应 domain 类型与 repository contract 中逐项映射，禁止静默丢弃。

### 5. 写入边界与安全

模型或 extractor 只产生 candidate JSON；确定性执行器（后续 change）负责通过 repository 完成 evidence、合并、Tag、幂等与事务。Tag 只能引用 Tidewise DB 中的 `event_tag_defs`；YAML 仅用于策略配置。任何模型输出的买入卖出、涨跌预测、利好利空、传导强度、评分或投资建议不得进入 `fact_payload` 或关系字段。Event Agent MVP 不处理实体匹配或实体关联。

## Risks / Trade-offs

- [Risk] JSONB 灵活性可能隐藏键语义漂移。→ Mitigation：MVP 不加 schema version；后续 candidate/extraction contract 必须定义允许键、拒绝键和结构校验，contract tests 固化边界。
- [Risk] 新增 nullable 关系字段可能导致旧记录缺少审计信息。→ Mitigation：保留旧字段语义，新增列只由新执行器写入；Review 明确空值含义和查询过滤。
- [Risk] migration 与旧 binary 并行部署产生读写不兼容。→ Mitigation：先 additive migration，再启用新列写入；migration gate 要求 before/after schema assertions 和兼容性检查。
- [Risk] 事件字段继续扩张造成事实与推理混杂。→ Mitigation：除 `fact_payload` 外其他 `events` 字段逐项 Review；明确排除预测、评分、传导和投资建议。

## Migration Plan

本轮不执行 migration。未来 Apply package 必须在 R2 migration gate 下按 `preflight -> additive migration -> schema/data assertions` 执行：确认 migration 版本、目标表/列/索引、既有行数和约束快照；仅追加字段/约束；验证既有数据可读、新字段默认/空值语义、重复执行幂等和 down/兼容策略。任一漂移、失败、超时、断言失败或发现需要写业务数据时立即停止；不执行 seed、回填、清理、Neo4j 或生产写入。

## Open Questions

- `event_sources.evidence_relation` 是否采用受控短文本枚举，`supports_fields` 是否采用 `TEXT[]`/JSONB，及其空值和验证规则。
- 三组关系候选字段的置信度精度/范围、assignment/match 取值集合与外键可空性。
- `fact_payload` 是否仅接受 JSON object、是否允许空 object，以及后续允许键集合由哪个 extraction contract change 定义。
- `event_type_code`/`action_code` 是否使用受控 Tag 或 `fact_payload`（当前 defer）；`lifecycle_status` 与现有 `event_status` 重叠（当前 defer）；`reference_period_start/end` 先放 `fact_payload`（当前 defer）；`last_seen_at` 可由 `event_sources` 聚合（当前 defer）；`event_time_precision` 是唯一具有独立事实语义且无法从 timestamp 恢复的候选，须由用户 Review 决定。
- 是否需要为新增字段建立索引；默认 YAGNI，除非有具体查询契约和选择性证据。

## Read-only Audit Baseline

本轮仅记录主对话提供的只读审计结果，不执行数据库查询或写入：local `tidewise_local` 当前 `raw_documents=407`；`events`、`event_sources`、`event_tag_defs`、`event_tag_maps`、`event_entity_links` 均为 0。四表 schema 来自 `000001_init_event_knowledge_schema.sql`，最新 migration 为 `000018`；现有 `events` 仅有 `title/summary/event_time/first_seen_at/knowable_at/event_status/fact_status/dedupe_key/primary_source_id`，`event_sources` 有 `source_level/evidence_excerpt/evidence_hash`，`event_tag_maps` 有 `assign_source/review_status`。现有 `domain.Event` 和 Admin `ListEvents` 映射这些字段，尚无 Event 写 repository。
