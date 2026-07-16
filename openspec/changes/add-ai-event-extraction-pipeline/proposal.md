## Why

现有 PostgreSQL 已有事件、raw document 证据和 Tag 基础表，也已有 `domain.Event` 与事件查询映射，但无法承载原子 Event 的事实载荷及最小可审计的证据、Tag 归因。原 Proposal 把 AI 流水线、任务编排和采集触发混入同一 change，超出当前批准范围；本次只修订为 Event DB 契约与后端映射准备，为后续独立的 extraction/worker change 提供稳定边界。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review 后可准备 domain/repository contract 设计与测试；不得执行 migration 或状态写入 | R1 | no | NONE | 仅 `backend/internal/domain`、`backend/internal/repositories` 的 Event 映射契约、fake/contract tests 和本 change artifacts；不调用模型/外网 |
| 2 | Apply 前必须单独批准并完成 migration preflight；Apply-final 前核验兼容性与 recovery evidence | R2 | yes | DRIFT_RECOVERY | 仅未来增量 PostgreSQL migration、migration/domain/repository contract tests；保留既有数据；不得写 Neo4j、采集、worker 或生产环境 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 1 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:1 |

## What Changes

- 收敛本 change 到 Event DB 契约：保留 `events`、`event_sources`、`event_tag_maps`，为 `events` 规划 `fact_payload JSONB` 的增量 schema 变更；现有 `event_entity_links` 保持原样，不触碰。
- 为证据和 Tag 评估最小审计增量：`event_sources` 的 `evidence_relation`、`supports_fields`；`event_tag_maps` 的 `confidence`、`assignment_reason`。这些字段须在 Review 中逐项决定，不能视为预批准。
- 使 Tidewise DB 成为 Tag 主数据唯一权威；YAML 只作为策略/映射输入，不作为运行时 Tag 主数据来源。
- 补齐未来实现所需的 domain 类型、验证规则、repository 接口/扫描映射及 migration contract tests；普通测试使用 fake，不访问模型、外网或真实数据库。
- 明确跨文档确定性召回与模型裁决、extraction candidate JSON、唯一 Event package、执行器事务/证据/幂等等属于后续 change 的输入契约，不在本 change 实现。

## Capabilities

### New Capabilities

无。本 change 只修改既有 `event-knowledge-schema` 的 Event DB 契约。

### Modified Capabilities

- `event-knowledge-schema`：增加 `fact_payload` 与最小证据/Tag 归因字段的 Review 候选，并保持原文、事件、证据链分层。

## Impact

- 规划影响 `backend/migrations`、`backend/internal/domain`、`backend/internal/repositories` 及对应 migration/domain/repository contract tests；本轮只修改 OpenSpec artifacts。
- 只读审计基线：local `tidewise_local` 当前 `raw_documents=407`，`events`、`event_sources`、`event_tag_defs`、`event_tag_maps`、`event_entity_links` 均为 0；四表 schema 来自 migration `000001`，最新 migration 为 `000018`。现有 `events`、`event_sources`、`event_tag_maps` 与 `domain.Event`/Admin `ListEvents` 的字段映射按该基线保留。
- 兼容既有 `events`、`event_sources`、`event_tag_maps` 数据；`event_entity_links` 保持原样并留给未来独立 change。未来 migration 必须为编号 `000019` 的增量、可回滚/兼容降级且禁止清空数据。
- 明确不影响 `backend/internal/apps/ingestion`、采集 connector、AI/LLM client、worker/job/run、实体 seed、实体匹配/候选/写入、Neo4j、frontend、`prototype/` 和 `doc/`；不创建 API、采集 Agent、Event Agent、影响推理或行情预测。
- 未来 Apply 的 migration 是 R2 操作，必须有明确 migration gate、只读 preflight、recovery evidence、before/after assertions 和停止条件；本 Proposal checkpoint 本身为 R0。
