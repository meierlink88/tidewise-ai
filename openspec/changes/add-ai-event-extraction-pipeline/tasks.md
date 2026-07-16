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

## 1. Event DB contract and mapping Package

- [ ] 1.1 在 `domain.Event` 与关系 domain 类型中先补 contract tests：`fact_payload` JSON object 校验、空 payload 兼容、禁止建议/预测字段、候选审计字段的空值与范围语义。
- [ ] 1.2 在 repository contract/fake tests 中定义 Event payload 读写、扫描映射、既有列表查询兼容，以及 evidence/Tag 候选字段不静默丢失；不得触碰 `event_entity_links`。
- [ ] 1.3 明确 Tidewise DB Tag authority、YAML policy-only 约束与 extraction candidate → deterministic executor → repository 的边界测试；不调用模型、外网或真实数据库，不实现实体匹配/实体候选。
- [ ] 1.4 根据 Proposal Review 逐项决策更新 design/specs/tasks，并形成 future `000019` migration 的精确字段清单、兼容说明和待审批项。

## 2. Additive migration and contract verification Package

- [ ] 2.1 （R2 migration gate）提交 migration preflight、范围/顺序/recovery/before-after assertions/停止条件，确认既有 `raw_documents=407` 且 `events`、`event_sources`、`event_tag_defs`、`event_tag_maps`、`event_entity_links` 均为 0 的基线未漂移；只读确认后保留既有数据。
- [ ] 2.2 编写并评审 `000019` 增量 migration contract tests：`fact_payload`、已批准的 evidence/Tag 候选字段、默认/NULL 语义、仅必要索引、重复执行和非破坏性 SQL；未通过 Review 的字段不得落地。
- [ ] 2.3 在获明确授权后执行 local-only migration，并验证版本、schema、既有行数/约束及 fail-closed/forward-fix 回滚证据；不得执行 seed、业务回填、实体关联、Neo4j 或生产写入。
- [ ] 2.4 运行受影响 backend package suite、architecture/contract tests、OpenSpec strict validate、scoped diff 和 secret scan，记录失败即停止。
