## Why

当前 PostgreSQL 只有事件、产业链节点和指数主数据，缺少承载外部 Agent 批次研究结果的“研究主题”和“研究锚点”事实结构，也没有支撑观潮家 Miniapp 首页及后续“查看影响路径/锚点层”的只读 API。现在先建立并评审数据、查询应用服务、DTO 和 HTTP 契约，才能在后续独立实现阶段安全提供首页卡片与详情读取，而不把研究结论误建成第三类实体。

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

## What Changes

- 新增研究主题事实结构 `research_themes` 及其 chain node、index、event 三类关系表。
- 新增研究锚点事实结构 `research_anchors` 及其 chain node、index、event 三类关系表。
- 以 `analysis_batch_id` 保存外部 Agent 批次编号；不创建 `research_analysis_runs`、`research_conclusions` 或任何 `research_conclusion_*` 表。
- 明确主题与锚点是两套平行结果，均拥有名称、一句话结论、传导路径、自然语言交易指向及独立关联；主题增加 impact/window 字段，锚点增加 type/importance 字段。
- 新增 Miniapp 只读 API：主题列表/详情、锚点列表/详情四个 GET endpoint，支撑首页卡片和影响路径/锚点层读取。
- 通过 query application service、repository read interface/implementation、DTO 和 handler 组织读取；handler 不访问 DB，查询 PostgreSQL，不查询 Neo4j，列表避免 N+1。
- 复用 `events`、`chain_node_profiles`、`index_profiles`，不复制或初始化指数、事件、产业链节点主数据；删除研究结果只级联其关系表。

## Capabilities

### New Capabilities

- `research-theme-anchor-foundation`: 为研究主题和研究锚点提供 PostgreSQL 事实结构、证据关联边界、Miniapp 只读查询和 HTTP DTO 契约。

### Modified Capabilities

- 无。现有主题主数据 `theme-foundation` 的 entity/theme profile 契约不变；本 change 的 `research_themes` 是研究结果，不是 `entity_type=theme` 主数据。

## Impact

- 规格与设计：`openspec/changes/add-research-theme-anchor-foundation/`。
- 后续实现区域：`backend/migrations`、`backend/internal/domain`、`backend/internal/repositories`、`backend/internal/apps/miniappapi`、`backend/internal/http` 及 API/仓储测试。
- 固定 API 路径：
  - `GET /api/v1/miniapp/research/themes`
  - `GET /api/v1/miniapp/research/themes/{theme_id}`
  - `GET /api/v1/miniapp/research/anchors`
  - `GET /api/v1/miniapp/research/anchors/{anchor_id}`
- 列表默认 `window_hours=24`、`limit=20`，边界分别为 `1..168` 与 `1..50`；主题排序为 `high > focus > watch`、同级 `published_at DESC`、最后 `id ASC`，锚点排序为 `importance` 业务优先级、`published_at DESC`、最后 `id ASC`。
- 发布可见性固定为 `published_at IS NOT NULL` 且 `published_at` 位于请求窗口；列表 cursor 固定 `as_of`，详情也接受同样窗口参数。
- 未来 DTO 示例：主题列表项含 `id/name/one_line_conclusion/impact_level/transmission_path/trading_direction/transmission_stage/next_checkpoint/published_at/affected_chain_nodes/related_indices/supporting_event_count/contradicting_event_count/has_more_detail`；响应含 `window_start/window_end/as_of/theme_count/event_count/items/next_cursor`。
- 小程序前端、Agent 生成/导入写 API、定时任务、Neo4j 查询/变更、分类表、display order、raw document 内容输出均为非目标；原型仅作信息架构参考，不复制 HTML/JS。
