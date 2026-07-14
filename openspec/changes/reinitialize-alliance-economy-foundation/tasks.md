# Task Design Efficiency：五个交付 Package

本 change 以 package 作为唯一 task 完成单元，不再把资料准备、技术微步骤或同一风险边界内的验证拆成独立 checkbox。普通 checkbox 只记录 package 状态；真正人工确认仅保留候选业务语义、两个 local R2 授权和 Apply-final Review。当前仍停在 Package 1 的最终联盟 manifest Review。

## Package 1：联盟范围与 Spec Review（R0，人工）

- [x] 1.1 **历史准备已完成**：workflow adoption、A：Schema / Data Contract Review、B provisional alliance candidate draft、CSV 69—85 排除和来源新鲜度整改均已有 checkpoint；已批准字段、identity 契约、68 条 recommendation 与全部空白 final decision 保持不变。
- [ ] 1.2 **最终联盟 Manifest Review**：主对话逐项确认 68 条候选和 10 个现有 active alliance 的最终 `approve/reject/merge/defer` 与 `keep/create/merge/inactivate` 处置，解决来源、identity、alias、summary、category、stale/merge 和关系影响冲突，形成穷尽且带版本/checksum 的 approved alliance manifest。

Acceptance criteria：

- `alliance-candidate-review.md` 的输入指纹、counts、确定性抽样、异常/宽边界和 fail-closed 条件完整；抽样不替代逐项 final manifest。
- 任一 final decision 留空、现有 active alliance 未穷尽或 merge/inactivate 未确认时，Package 1 不得完成。
- Package 1 完成只允许启动 Package 2 的 R0 候选工作；不授权源码、migration、seed、PostgreSQL 或 Neo4j。

## Package 2：Economy 与关系候选 Review（R0，人工）

- [ ] 2.1 **候选 Package Review**：Package 1 通过后，包内连续完成正式成员来源审计、formal-active 成员全集、现有 economy identity/ISO diff、缺失 economy 候选、economy exception/protection manifest，以及穷尽 `member_of` manifest；全部准备完成后由主对话一次审阅业务语义与最终候选。

Acceptance criteria：

- 包内按“批准联盟 → 正式成员来源 → economy diff/补齐 → `member_of`”顺序连续执行，不为资料生成、diff 或技术检查逐项停顿；任一前置断言失败则整个 package fail-closed。
- economy 候选包含规范中英文名/aliases、四类 identity、ISO 或不适用、currency、兼容 region、稳定/拟新增 key 与来源；exception 只含逐项确认的冲突、重复或错误，其他合法 economy 进入保护快照。
- `member_of` 固定 `economy -> alliance_org`，只含 formal active，逐条带端点、来源、核验时间、现有 edge disposition、stale reason 与完整性断言，并穷尽现有 active `member_of`。
- `led_by`、`part_of` 是非阻塞可选附录：证据充分时随本 package 一次 Review；未决或未批准时明确排除出本次 MVP 和后续 R2B，不产生额外 gate。
- Review 通过只冻结 approved economy/relationship manifests；不生成可执行写入授权。

## Package 3：R1 实现与自动技术验收（无有状态写入）

- [ ] 3.1 **Implementation checkpoint**：等待 `refactor-industry-chain-node-foundation` 完成 Deliver 且结果进入最新 `origin/main`，安全更新本 branch 并自动完成 overlap audit；随后按 TDD 一次实现 migration、validator、repository/service、mapping-only seed、dry-run/report 和关系 policy，以及对应测试与只读 preflight，不执行 migration/seed/database write。

Acceptance criteria：

- overlap audit 覆盖共享 entityfoundation seed/repository、migration 序号与 tests；发现 artifact/contract 冲突时回到 R0 Review，不在过期设计上实现。
- RED→GREEN→REFACTOR 覆盖已批准 profile/identity 契约、manifest 穷尽性、稳定 identity、forward convergence、stale/inactive provenance、保护快照、幂等、checksum 漂移与禁止破坏性重置。
- checkpoint 自动运行 targeted tests、受影响交付边界完整 suite、共享 architecture/contract tests、migration 静态验证、OpenSpec strict、diff/scope/secret；repo-wide full test 不作为默认要求，仅在项目规则的实际触发条件命中或边界不清时运行。
- 输出 scoped R1 implementation diff 和验证证据；不设置独立 R1 人工门禁，也不授权 Package 4 的任何 R2。

## Package 4：两个 Local PostgreSQL R2（分别人工授权）

- [ ] 4.1 **R2A `master-data` Review → Write → Query**：主对话一次明确授权目标 local 环境中的必要 schema、alliance 与 economy forward convergence；以 approved manifests/version/checksum、完整 exact diff、范围/排除项、recovery evidence、预计 counts、before/after assertions 和停止条件为执行包，在单事务边界内写入并立即 Query 验收。
- [ ] 4.2 **R2B `relationships` Review → Write → Query**：仅在 R2A Query 验收后，由主对话独立授权 approved `member_of` forward convergence；只有 Package 2 同一 Review 已批准的 `led_by`/`part_of` 才可被明确列入，否则排除。写入后立即 Query active tuple 集合、端点、方向、重复/悬空、stale provenance、官方成员集合与幂等。

Acceptance criteria：

- 两个 R2 各自只有一个人工授权点和一条 `Review → Write → Query` 链；R2A 不再为 schema、alliance、economy重复设门禁，R2B 不被 R2A 或普通 Apply 推定授权。
- curated local PostgreSQL 默认要求可恢复 backup；只有用户逐层明确批准 disposable recovery 时才可替代。未授权范围、checksum/counts/断言漂移或 Query 失败立即停止并使未执行授权失效。
- R2A Query 必须证明 active alliance keys 等于 approved manifest、economy exceptions 精确生效且无关合法 economy key/UUID/status 未变化；R2B Query 必须证明 active relationship tuples 等于 approved manifest。
- 禁止 `TRUNCATE`、无谓词 DELETE、清空重灌、历史 rollback、手工修表或其他未审阅写路径。

## Package 5：Apply-final Review 与 Deliver

- [ ] 5.1 **Apply-final → Sync → Archive → Deliver**：汇总 R1、R2A、R2B 的 scoped evidence，运行受影响交付边界测试、共享 tests、OpenSpec strict、diff/scope/secret 和需求覆盖 self-review；主对话完成唯一 Apply-final 人工 Review 后，依次 Sync、Archive、`openspec validate --all`、archive commit、PR/merge 和按 worktree 所有权 cleanup。

Acceptance criteria：

- Apply-final 之前不得 Sync/Archive/Deliver；普通 checkbox、R2 Query 或旧批准都不能替代该人工 Review。
- 本 change 以 PostgreSQL 事实源验收为完成边界，不执行 Neo4j Review/Rebuild/Query。图投影由后续独立 graph projection change 读取已验收 PostgreSQL facts，不阻塞本 change。
- 未创建完成态 PR、未 merge 或未完成规定 cleanup 时不得宣称 delivered。

## 旧 Task → 新 Package 映射

| 旧 tasks | 新位置 | 状态/效率处理 |
|---|---|---|
| 0.1、1.1—1.4、2.1—2.2 | Package 1.1 | 历史完成证据保留为已完成；workflow adoption 0.2 行政门禁删除，不追认新业务授权。 |
| 2.3 | Package 1.2 | 保留为当前唯一联盟业务人工出口。 |
| 3.1—3.5、4.1—4.3、4.6 | Package 2.1 | 合并为连续候选生成与一次最终业务 Review，不在中间逐项暂停。 |
| 4.4—4.5 | Package 2 可选附录 | 有证据且同包批准则纳入；否则排除，不阻塞 MVP。 |
| 5.1—5.2、6.1—6.6 | Package 3.1 | 合并 overlap audit、TDD 实现和自动技术验收；删除旧 5.3 独立 R1 人工门禁。 |
| 7.1—7.4 | Package 4.1 | schema + alliance + economy 合并为一个 local R2A master-data 包。 |
| 7.5—7.8 | Package 4.2 | `member_of` 与同包获批可选关系合并为一个 local R2B relationship 包。 |
| 8.1—8.3 | Package 5 / 后续 change | Apply-final/Deliver 证据进入 Package 5；Neo4j 全部移出，留给独立 graph projection change。 |
