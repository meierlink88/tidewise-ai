# Task Design Efficiency：五个交付 Package

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


本 change 以 package 作为唯一 task 完成单元，顶层 checkbox 只表达内聚交付状态。真正人工 gate 仅保留候选业务语义、R3 local cleanup、R2 latest manifest rebuild 和 Apply-final。Package 3.1 已完成 R1 实现与自动技术验收，但不授权任何数据库操作。

## 1. 联盟范围与 Spec Review Package

- [x] 1.1 **历史准备已完成**：workflow adoption、旧 A contract、旧 68 条 provisional draft、来源整改和 task packaging checkpoint 保留为历史证据；旧候选、recommendation、网页核验和 categories 契约已被新 Excel 真值源 supersede。
- [x] 1.2 **最终联盟 Manifest Review**：主对话已于 2026-07-14 批准 `联盟组织列表1.0.xlsx` 全部 45 条候选、四字段、两个 U+200C normalization、9 keep + 36 create；`approved-alliance-manifest.md` v1 canonical checksum 为 `4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a`。

Acceptance criteria：

- 当前联盟业务输入只有名称、缩写、核心主导方和核心影响范围说明；名称只进入 `entity_nodes`，profile 不含 name/categories。
- Package 1 完成只允许进入 R0 候选工作，不授权源码或任何数据库写入。

## 2. Economy 与关系候选 Review Package

- [x] 2.1 **候选 Package Review**：主对话已于 2026-07-14 批准 79 条 economy target 与 133 条 formal-active `economy -> alliance_org member_of`；economy checksum `95613a931adf3d7231cbb1d311e5051f3695d9da40c60bbeeccb39d006118cb3`，member checksum `c3d652571fa93307088633cbfe06dce18b1257d8ac2b3ee2ae88c5a27e69fcf7`。

Acceptance criteria：

- 45/79/133 是 latest approved rebuild target；旧 223 条 keep/preserve/proposed-inactivate disposition 只保留为历史候选证据，不再作为 importer、cleanup 或 rebuild 的执行输入。
- `member_of` 只含 formal active，方向固定 economy → alliance；21 个非正式 membership model 不生成关系。
- `led_by`、`part_of`、`participates_in`、`signatory_to` 全部排除出本 change。

## 3. R1 最小实现与自动技术验收 Package

- [x] 3.1 **Implementation checkpoint**：基于最新 `origin/main` 完成 overlap audit，并按 TDD 实现 alliance 三字段最小 migration/domain/repository 适配、冻结 45/79/133 artifact、一个 change-specific importer、最小 preflight/post-query/assertion 与 targeted tests；未 apply migration、执行 seed 或写数据库。

Acceptance criteria：

- 不新增 `economy_profiles.identity_kind`、区域/货币 schema 规则、全局 `entity_nodes.entity_key` 唯一索引、平行 profile 表、通用 manifest/service/policy/mapping framework 或复杂 dry-run/report。
- importer 优先复用现有 entity-seed/repository，只接受本 change 冻结 artifact；覆盖字段、count/checksum、端点、精确 scope、原子性或明确 fail-closed 两阶段、幂等和漂移停止条件。
- 完成只读 dependency audit：枚举目标表、FK、`entity_edges` relation type/count；静态 fixture 基线至少记录 10 alliance、50 economy、223 `member_of` 与 40 `economy -> market has_market`。真实 local counts 必须在未来只读 preflight 刷新。
- economy 与 market/index/benchmark/industry chain/company/person 等跨域事实必须形成“删除并丢弃”或“保留/重建”候选；主对话未决定时阻断 Package 4.1，不得自行判断。
- 运行 targeted tests、受影响 backend suite、共享 architecture/contract/migration tests、OpenSpec strict、task-design lint、diff/scope/secret；输出 scoped R1 Review package。Package 3.1 完成不授权 Package 4。

## 4. Local PostgreSQL Cleanup 与 Rebuild Package

- [x] 4.1 **R3 scoped local cleanup Review → Write → Query**：2026-07-14 第二次、唯一获授权的 CLI 执行已完成。fresh preflight 匹配 `tidewise_local`、Goose=17、10/10/50/50/223/40、dependency checksum `c312731a72705ccf293ee3a24a58ed0aa07fe099375e12345402695600de0b9b`、protected checksum `32dc4eaf1132a6a18d5850b0cfd19ab0536931652557b384c0a6edaf67992cdf` 与 frozen manifest。删除精确为 223 `member_of`、10 profile、10 alliance；post Query 为 alliance/profile/member_of=0/0/0、economy/profile=50/50、`has_market`=40、protected checksum 不变、Goose=17。脱敏证据见 `package-4-1-execution-evidence.md`；不授权 migration、4.2 或任何下一层。
- [x] 4.2 **R2 latest manifest rebuild Review → Write → Query**：主对话已单独授权并完成冻结 manifest 的 45 alliance/profile、79 target economy/profile 与 133 formal-active `member_of` 重建；随后已获授权的同 artifact 幂等验证返回 `entity/profile/member_writes=0/0/0`。Query 确认 15 non-target economy/profile 与跨域事实保持，集合精确、端点 active、无孤儿/重复/方向 mismatch。脱敏证据见 `package-4-2-execution-evidence.md`。

Acceptance criteria：

- 4.1 与 4.2 是两个 fail-closed 独立授权包；普通 Apply、旧批准、R1 或上一包成功均不能推定下一包 Write 授权。
- 50 个 existing economy/profile 与所有非 `member_of` 跨域事实是简单 scope exclusion；35 existing target 只在 4.2 原位 upsert，44 missing target 才创建，15 non-target 永不进入 manifest convergence。
- 环境不是 local、真实 count/hash 与 Review package 漂移、FK/跨域处置未决、cleanup zero assertion 或 rebuild post-query 失败时立即停止。
- 旧 223 disposition、preserve 断言、recovery evidence、backup/restore rehearsal 和 forward convergence 要求均已废止；不得实现它们的替代通用系统。
- Neo4j、UAT、prod、shared 及未授权跨域事实不在任何执行包中。

## 5. Apply-final Review 与 Deliver Package

- [ ] 5.1 **Apply-final → Sync → Archive → Deliver**：Apply-final Review package 已准备，脱敏证据见 `package-5-apply-final-review.md`；仍等待主对话人工 Review。通过后才可依次 Sync、Archive、`openspec validate --all`、archive commit、PR/merge 和按 worktree 所有权 cleanup；本 checkpoint 不授权任何后续生命周期操作。

Acceptance criteria：

- Apply-final 前不得 Sync/Archive/Deliver；未完成 PR/merge/cleanup 时不得宣称 delivered。
- PostgreSQL exact Query 是本 change 完成边界；Neo4j 由后续独立 graph projection change 处理。

## 旧 Task → 新 Package 映射

| 旧 tasks / 语义 | 新位置 | 状态/效率处理 |
|---|---|---|
| 0.1、1.1—1.4、2.1—2.3 | Package 1 | 历史证据与 45 条人工批准保持完成。 |
| 3.1—4.6 | Package 2 | 79/133 业务批准保持完成；旧 223 disposition 执行语义被本 amendment supersede。 |
| 5.1—6.6 | Package 3.1 | 收缩为 change-specific R1；删除通用 service/policy/mapping/dry-run/report 与独立 R1 人工门禁。 |
| 旧 7.1—7.8 / R2A、R2B convergence | Package 4.1、4.2 | 改为 R3 scoped local cleanup 与 R2 latest manifest rebuild，各自独立授权；删除 backup/recovery/preserve。 |
| 8.1—8.3 | Package 5 / 后续 change | Apply-final/Deliver 进入 Package 5；Neo4j 全部移出。 |
