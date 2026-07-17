## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review：确认研究语义、字段、受控枚举、发布可见性与 API 契约 | R0 | yes | SPEC_SEMANTICS | 仅审阅和修订本 change 的 proposal、design、delta spec、tasks；不得执行 Apply、migration、数据库写入、Sync 或 Archive |
| 2 | Apply 前的本地 migration 写入授权 | R2 | yes | DRIFT_RECOVERY | 仅在 Package 1 获批后，按下方 Stateful Layer Map 对 local PostgreSQL 执行一次获批的 forward-only migration；不得触及 UAT/prod、seed、Neo4j 或业务数据 |
| 3 | Apply-final Review：数据读取实现、Miniapp API、测试与 schema 契约验收 | R1 | yes | APPLY_FINAL | 仅审阅已完成的 migration/domain/repository/query service/handler/DTO/tests 及 fresh verification；不得代替 Sync、Archive、Deliver 或 PR 授权 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 1 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| research-theme-anchor-local-migration | 2 | local | 1 | 仅在 Package 1 通过且取得独立授权后，对指定 local PostgreSQL 执行新增 8 张研究主题/锚点表的单个 forward-only migration，并完成 schema assertions | UAT、prod、shared-local、seed、业务写入、Neo4j、索引主数据灌入和 active change 的数据/部署操作 | backup | new:research-theme-anchor-local-backup | counts=review-gated;hash=review-gated;schema=review-gated | PostgreSQL identity、migration version、目标库 backup 可恢复性、8 张表不存在且 `events`/`chain_node_profiles`/`index_profiles` FK 目标存在已确认 | 8 张表、FK、复合唯一键、CHECK、级联删除边界、发布可见性查询索引通过 SQL assertions；被引用主数据行数不变 | identity、scope、backup、schema 或断言漂移，migration 非幂等/失败，或发现非 local 写入立即停止 |

## 1. Proposal Review Package

- [x] 1.1 审阅并确认 `research_themes`、`research_anchors` 及 6 张关联表的精确字段、UUID/FK、审计字段、索引、复合主键和删除边界。
- [x] 1.2 逐项决策 `impact_level`、`transmission_stage`、`relation_role`、`importance`、`evidence_role`、`impact_direction`、`anchor_type` 候选集合，以及发布可见性和详情窗口语义。
- [x] 1.3 逐项确认四个 Miniapp GET path、window_hours/limit 边界、DTO 字段、Event 脱敏、空数组、not found、错误 JSON 和稳定 cursor 语义。
- [x] 1.4 确认 non-goals、active change 隔离、PostgreSQL-only、无 research conclusion/run/display order/Neo4j 结构的边界，并记录 Review 结论后再开始 Apply。

## 2. Schema And Read Repository Package

- [x] 2.1 先编写 schema contract、repository read model 和迁移解析测试，覆盖 8 张表、必填文本、受控 CHECK、时间窗口、发布可见性、FK 删除行为、复合唯一键和无 `display_order`。
- [x] 2.2 编写单个 forward-only `backend/migrations` migration，按 design 创建 8 张表、审计字段、明确 FK、索引和 Proposal Review 已确认的受控集合；不插入主数据。
- [x] 2.3 在 `internal/domain`/`internal/repositories` 中实现研究读取 read model、`ResearchReadRepository` interface 和 PostgreSQL 查询，使用批量聚合/CTE 避免列表 N+1，Event 计数按 event_id 去重，index 缺失返回空集合。
- [x] 2.4 为 repository 写 fake/SQL mock 或可重复集成边界测试，覆盖空数据、指数为空、去重计数、稳定排序、分页 keyset、未发布过滤、详情 not found 和不读取 raw document content。
- [ ] 2.5 在获批的 local R2 layer 内执行 preflight → migration → schema assertions，确认既有 events、chain_node_profiles、index_profiles 计数不变；不得执行 UAT/prod/Neo4j/seed/业务写入。

## 3. Miniapp Read API And Final Verification Package

- [x] 3.1 先编写 `internal/apps/miniappapi` query application service、cursor codec、DTO 和 handler 的 Go 测试，覆盖非法参数、非法 cursor、空数据、稳定排序、分页边界、not found、未发布不可见、关联数据和 repository failure。
- [x] 3.2 实现 query application service：校验 24/1..168 window_hours、20/1..50 limit、固定 as_of、opaque keyset cursor、主题/锚点 DTO 映射和统一错误分类；handler 不得访问 DB。
- [x] 3.3 实现四个 GET routes，并通过 `internal/http`/`cmd/api` 依赖注入 repository/service；返回 RFC3339 UTC、`[]` 空集合、主题/锚点批次摘要和不含 raw document content 的 Event 摘要。
- [x] 3.4 运行受影响 backend persistence/miniappapi 边界的完整 suite、精确 OpenSpec task-design lint、`openspec validate add-research-theme-anchor-foundation --strict`、`git diff --check` 和 secret/scope 检查。
- [x] 3.5 完成 self-review：逐条核对 proposal/design/spec/tasks 与实现一致，复读 fresh verification 输出，记录未验证项/风险；提交 scoped checkpoint 后等待 Apply-final Review。
