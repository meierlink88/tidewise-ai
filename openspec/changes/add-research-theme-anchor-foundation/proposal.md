## Why

当前 PostgreSQL 只有事件、产业链节点和指数主数据，缺少承载外部 Agent 批次研究结果的“研究主题”和“研究锚点”事实结构，导致一句话结论、传导路径、交易指向及其证据关系没有稳定落点。现在先建立并评审最小 schema 契约，才能在后续独立 change 中安全实现 Agent 回写、查询或展示，而不把研究结论误建成第三类实体。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review：确认研究语义、字段、受控枚举与非目标 | R0 | yes | SPEC_SEMANTICS | 仅审阅和修订本 change 的 proposal、design、delta spec、tasks；不得执行 Apply、migration、数据库写入、Sync 或 Archive |
| 2 | Apply 前的本地 migration 写入授权 | R2 | yes | DRIFT_RECOVERY | 仅在 Package 1 获批后，按下方 Stateful Layer Map 对 local PostgreSQL 执行一次获批的 forward-only migration；不得触及 UAT/prod、seed、Neo4j 或业务数据 |
| 3 | Apply-final Review：实现、测试与 schema 契约验收 | R1 | yes | APPLY_FINAL | 仅审阅已完成的 migration/domain/repository/tests 及 fresh verification；不得代替 Sync、Archive、Deliver 或 PR 授权 |

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
| research-theme-anchor-local-migration | 2 | local | 1 | 仅在 Package 1 通过且取得独立授权后，对指定 local PostgreSQL 执行新增 8 张研究主题/锚点表的单个 forward-only migration，并完成 schema assertions | UAT、prod、shared-local、seed、业务写入、Neo4j、索引主数据灌入和 active change 的数据/部署操作 | backup | new:research-theme-anchor-local-backup | counts=review-gated;hash=review-gated;schema=review-gated | PostgreSQL identity、migration version、目标库 backup 可恢复性、8 张表不存在且 `events`/`chain_node_profiles`/`index_profiles` FK 目标存在已确认 | 8 张表、FK、复合唯一键、CHECK、级联删除边界与索引通过 SQL assertions；被引用主数据行数不变 | identity、scope、backup、schema 或断言漂移，migration 非幂等/失败，或发现非 local 写入立即停止 |

## What Changes

- 新增研究主题事实结构 `research_themes` 及其 chain node、index、event 三类证据/关联表。
- 新增研究锚点事实结构 `research_anchors` 及其 chain node、index、event 三类证据/关联表。
- 以 `analysis_batch_id` 保存外部 Agent 批次编号；不创建 `research_analysis_runs`、`research_conclusions` 或任何 `research_conclusion_*` 表。
- 明确主题与锚点是两套平行结果，均拥有名称、一句话结论、传导路径、交易指向及独立关联；主题增加 impact/window 字段，锚点增加 type/importance 字段。
- 复用 `events`、`chain_node_profiles`、`index_profiles`，不复制或初始化指数、事件、产业链节点主数据；删除研究结果只级联其关系表。
- 在 delta spec 中固定 UUID/FK、审计字段、非空校验、时间窗口、复合唯一性、索引和受控枚举候选集合，供 Proposal Review 决定。

## Capabilities

### New Capabilities

- `research-theme-anchor-foundation`: 为研究主题和研究锚点提供 PostgreSQL 事实结构、证据关联边界和外部批次回写契约。

### Modified Capabilities

- 无。现有主题主数据 `theme-foundation` 的 entity/theme profile 契约不变；本 change 的 `research_themes` 是研究结果，不是 `entity_type=theme` 主数据。

## Impact

- 规格与设计：`openspec/changes/add-research-theme-anchor-foundation/`，后续只影响 PostgreSQL migration、共享 domain/repository 及其测试边界。
- 数据库：后续获批 Apply 才新增 8 张表；PostgreSQL 仍是事实源，Neo4j 不变。
- 业务 API、Agent 提取/报告编排、定时任务、小程序页面、查询 API、seed 数据和 `prototype/`、`doc/` 均为非目标。
- 与 active change `migrate-uat-to-linux-amd64` 不共享文件、运行环境、数据库写状态或执行顺序；本 change 不修改其 artifacts、workflow、UAT 配置或部署状态。
