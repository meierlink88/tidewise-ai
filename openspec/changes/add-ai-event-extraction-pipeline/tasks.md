## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review 后可准备 domain/repository contract 设计与测试；不得执行 migration 或状态写入 | R1 | no | NONE | 仅 `backend/internal/domain`、`backend/internal/repositories` 的 Event 映射契约、fake/contract tests 和本 change artifacts；不调用模型/外网 |
| 2 | 独立批准后执行 fresh preflight/backup → authorized additive migration → read-only schema/count assertions；漂移或失败立即停止 | R2 | yes | DRIFT_RECOVERY | 仅 local PostgreSQL `000019` 增量 migration、migration/domain/repository contract tests；不得执行 seed、回填、Neo4j、采集、worker 或生产写入 |
| 3 | Apply-final Review：核验 package 1/2 scoped diff、测试、migration assertions 和未验证项；通过后才可请求后续生命周期授权 | R2 | yes | APPLY_FINAL | 仅本 change 的 Apply-final 验证与 Review 记录；不得 Sync、Archive、push/PR 或执行未批准的其他状态操作 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 1 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:1 |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---:|---|---:|---|---|---|---|---|---|---|---|
| event-db-migration | 2 | local | 1 | PostgreSQL `tidewise_local` only; migration `000019` for reviewed Event/evidence/Tag columns | seed, backfill, cleanup, entity links, Neo4j, worker, production/shared environments | backup | new:event-db-migration-local | counts=fresh-before;hash=fresh-before;schema=000001-000018 | Fresh PostgreSQL schema/version, table counts, constraints/indexes and backup identity are captured; 407/0 audit values are not reused as live assertions | `000019` is applied once; reviewed schema exists; every affected table row count equals fresh-before; migration version and non-destructive assertions pass | Any schema/hash/count drift, backup failure, migration failure, timeout, assertion failure or manual stop immediately invalidates remaining scope |

## 1. Event DB contract and mapping Package

- [x] 1.1 在 `domain.Event` 与证据/Tag domain 类型中先补 contract tests：`fact_payload` JSON object 校验、空 payload 兼容、禁止建议/预测字段、候选审计字段的空值与范围语义。
- [x] 1.2 在 repository contract/fake tests 中定义 Event payload 读写、扫描映射、既有列表查询兼容，以及 evidence/Tag 候选字段不静默丢失；不得触碰 `event_entity_links`。
- [x] 1.3 明确 Tidewise DB Tag authority、YAML policy-only 约束与 extraction candidate → deterministic executor → repository 的边界测试；不调用模型、外网或真实数据库，不实现实体匹配/实体候选。
- [x] 1.4 根据 Proposal Review 逐项决策更新 design/specs/tasks，并形成 future `000019` migration 的精确字段清单、兼容说明和待审批项。

## 2. Additive migration and contract verification Package

- [x] 2.1 （R2 migration gate）在 fresh preflight/backup 中读取当前 schema/version、受影响表行数、约束/索引与 backup identity；407/0 仅作为 2026-07-16 audit snapshot，不作为未来相等断言，漂移时重新评估兼容性。Fresh before 与 recovery evidence 见 [R2 execution evidence](reviews/event-db-migration-r2-execution.md)。
- [x] 2.2 编写并评审 `000019` 增量 migration contract tests：`fact_payload`、已批准的 evidence/Tag 候选字段、默认/NULL 语义、仅必要索引、重复执行和非破坏性 SQL；未通过 Review 的字段不得落地。
- [x] 2.3 在获明确授权后执行 local-only migration，并验证版本、schema、既有行数/约束及 fail-closed/forward-fix 回滚证据；不得执行 seed、业务回填、实体关联、Neo4j 或生产写入。单次 Apply 与 after assertions 见 [R2 execution evidence](reviews/event-db-migration-r2-execution.md)。
- [x] 2.4 运行受影响 backend package suite、architecture/contract tests、OpenSpec strict validate、scoped diff 和 secret scan，记录失败即停止。统一 migration 目录属于共享运行时契约，本轮已使用 Complexity Budget 中唯一一次 backend `go test ./...` 并通过；其他验证亦全部通过。

## 3. Apply-final Review Package

- [ ] 3.1 汇总 package 1/2 的 scoped diff、fresh preflight/backup、`000019` migration 结果、read-only schema/count assertions、测试输出和未验证项。
- [ ] 3.2 通过 Apply-final Review 前不得 Sync、Archive、push/PR 或执行其他有状态操作。
