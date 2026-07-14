## Why

现有仓库 fixture 基线包含 10 个 `alliance_org`、50 个 economy、223 条 active `member_of`，联盟 profile 仍使用 `org_code/org_type/primary_domain/scope_region/official_url`。用户已经批准新的业务基线：45 个 alliance、79 个 economy 和 133 条 formal-active `member_of`。本 change 只服务 local PostgreSQL 探索环境的一次性、可审阅、可重放重建，不建设通用数据导入产品。

联盟候选唯一真值源仍为 `联盟组织列表1.0.xlsx` SHA-256 `ac0d953c0cd93596fe6bf8a70541bbe658620e75d38a9b3178980071b2cdc102` 的首个 sheet `联盟组织`、范围 `A1:K51`。旧 `表格_20260713.csv` 的 68 条候选及其网页核验已经 superseded，只保留为 Git 历史。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Alliance manifest semantics Review | R0 | yes | SPEC_SEMANTICS | 仅确认 45 条联盟与四字段；不授权 economy、关系、源码或数据库 |
| 2 | Economy and relationship semantics Review | R0 | yes | SPEC_SEMANTICS | 仅确认 79 economy 与 133 member_of；不授权源码或数据库 |
| 3 | R1 implementation package | R1 | no | NONE | 完成 change-specific 源码、测试与只读审计；不授权任何写入 |
| 4 | Local cleanup and rebuild execution | R3 | yes | R3_OPERATION | 4.1 R3 cleanup 与 4.2 R2 rebuild 必须各自明确授权；全部现有 economy 与未授权跨域事实已确认为 scope exclusion |
| 5 | Apply-final Review | R1 | yes | APPLY_FINAL | Review 通过后才允许 Sync、Archive、Deliver；不新增任何数据库或 Neo4j 授权 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 4 |
| stateful_layers | 0 |
| checkpoints | 5 |
| full_test_runs | 2 |
| continuous_automation_scope | packages:3 |


## What Changes

- **BREAKING**：将 `alliance_org_profiles` 最小目标收敛为 `entity_id`、`abbreviation`、`leadership_summary`、`influence_scope_summary`。名称与规范名称只进入 `entity_nodes.name/canonical_name`；非空缩写按既有识别惯例派生为 alias。
- 不保存 `categories`、大类、子类、成员数、全球占比、约束力级别、影响力评级或其他 sheet；不新增实体标签机制。
- 保持 `economy_profiles` 现有 `country_code/currency_code/region` 三字段，不新增 `identity_kind`、区域/货币规则或全局 `entity_key` 唯一约束。79 条 approved economy 以冻结 artifact 的既有 identity 表达；EU/global 等聚合不得替代国家成员事实。
- 冻结 approved data artifact：45 alliance、79 economy、133 条 `economy -> alliance_org` formal-active `member_of`。旧 223 条 disposition/preserve 策略不再是执行契约。
- 实现只包含必要 alliance schema/domain/repository 最小适配，以及一个仅服务本 change 的 importer；优先复用现有 entity-seed/repository，单事务或明确 fail-closed 两阶段、幂等、可重放。
- local 探索环境允许对联盟、经济体及其关系做精确 scoped cleanup 后按最新 manifest 重建。该破坏性豁免不适用于 UAT、prod 或 shared，也不构成当前写入授权。
- R1 必须先完成只读 dependency audit，枚举表、FK、关系类型/count 和跨域事实。仓库 fixture 已知除 `member_of` 外还有 40 条 `economy -> market` 的 `has_market`；已确认全部现有 economy 及未授权跨域事实保留，若出现其他 alliance incident edge 或审计漂移则阻断 cleanup。
- Package 4 分为两个独立人工授权包：4.1 R3 scoped local cleanup；4.2 R2 latest manifest rebuild。无 backup、rollback 或恢复演练要求，但每包仍须展示环境、范围、count/hash、顺序、断言和停止条件。
- cleanup 后必须验证 alliance/profile/member_of=0、economy/profile=50 与跨域保护 hash 不变；rebuild 后必须精确验证 45/79/133、15 个 non-target economy/profile 保留、端点完整、无孤儿、无重复、方向正确和幂等复跑。
- PostgreSQL 是唯一事实源和完成边界；Neo4j 不在本 change。

## Capabilities

### New Capabilities

- `alliance-economy-foundation`: 定义联盟四业务字段、approved 45/79/133 manifest，以及 local scoped cleanup/rebuild 的强门禁和完整性断言。

### Modified Capabilities

- `event-knowledge-schema`: 收敛联盟 entity/profile 字段，不改变 economy profile schema。
- `entity-foundation-seeds`: 定义 change-specific importer、scoped cleanup/rebuild、幂等与 exact Query。
- `entity-relationship-curation`: 定义 formal-active `member_of` 重建边界，并禁止从“核心主导方”文本自动生成 `led_by`。

## Impact

- R1 预计只影响 `backend/migrations/`、`backend/data/entity_foundation/`、`backend/internal/apps/entityfoundation/seed/`、`backend/internal/repositories/`、最小 CLI 入口及对应测试；不得引入通用 manifest framework、service 层、relationship policy engine、复杂 dry-run/report 或任意实体 mapping framework。
- R1 只生成源码、测试、冻结 artifact 和只读 Review package；不得 apply migration、执行 seed、写 PostgreSQL、连接/写 Neo4j、创建 backup 或自动进入 Package 4。
- 不修改源工作簿、`prototype/`、项目外 `doc/`、UAT、prod 或 shared 环境。
