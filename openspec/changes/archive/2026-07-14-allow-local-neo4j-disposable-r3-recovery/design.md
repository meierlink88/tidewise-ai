## Context

PostgreSQL 是项目实体、事件和关系的唯一事实源，Neo4j 只保存可从 PostgreSQL 完整重建的图投影。现行 `.agents/openspec-workflow.md` 的 R3 总则允许“备份/恢复或等价灾难恢复证据”，但 Stateful Layer Map 与 `task_design_lint.go` 只接受 local R2 的 `approved-disposable-recovery`，形成规则文字与机器 contract 的不一致。

只读复验 `rebuild-foundation-graph-and-enrich-chain-data` 得到六条错误：Proposal 与 tasks 各有三个 local Neo4j R3 layer 被同一 `stateful-recovery` 分支拒绝，未发现其他 task-design error。原 Layer 名称与 Scope 合起来表达了 Neo4j cleanup/rebuild/sync 语义，但 Scope 本身尚未完整包含批准规则要求的 `Neo4j` 与英文 operation token，不能由 Layer 名称代替；Before Assertions 已引用冻结或验收后的 PG baseline，且现有 schema 已强制 Recovery Baseline、Expected Counts/Hash/Schema、Before/After Assertions 与 Stop Conditions 非空或格式化。

本 change 是共享 workflow/lint 的 R1 修改，不执行 R3 操作。由于只扩展既有 Stateful Layer Map 的一个 recovery 判定，不涉及后端运行时流程、跨模块调用、外部 API、数据模型或部署边界，design 不增加 Mermaid 图。

## Goals / Non-Goals

**Goals:**

- 保留 local R2 对 `approved-disposable-recovery` 的原有支持。
- 仅允许满足全部条件的 local Neo4j R3 projection layer 使用该 recovery：Scope 是 `cleanup`、`rebuild` 或 `sync`；PG baseline 已冻结、验收且可完整重建投影；字段完整；命名 layer 在运行前取得独立明确授权。
- 对 shared-local、UAT、prod/shared、非 Neo4j R3、非 projection 操作、不可从 PG 完整重建的状态继续 fail-closed。
- 不降低 R3 独立授权、逐层执行、失败停止、禁止跨层批量及其他环境正式灾备要求。
- 让受阻 change 在规则 Deliver 后无需伪造 backup，并通过其自身 Review 将三个 Stateful Layer Map Scope 补成明确机器锚点后解除 lint 根因。

**Non-Goals:**

- 不授权或执行任何 Neo4j cleanup、rebuild、sync、数据库写入或 projection 操作。
- 不修改受阻 change artifacts、业务源码、CI、数据库、Neo4j、`doc/` 或 `prototype/`。
- 不增加 Stateful Layer Map 列、环境枚举、recovery 枚举、依赖、通用 policy engine 或 workflow framework。
- 不把 PostgreSQL、其他数据库、缓存、文件、对象存储或不可完整重建状态定义为 disposable R3。

## Decisions

### 1. 在现有 recovery 分支增加最小合取谓词

`approved-disposable-recovery` 的合法性定义为：

```text
existingLocalR2
OR
(
  Environment == local
  AND Gate.Risk == R3
  AND Gate.Human == yes
  AND Scope 明确包含 Neo4j
  AND Scope 明确包含 cleanup、rebuild、sync 之一
  AND Before Assertions 明确引用 PG 或 PostgreSQL baseline
)
```

匹配对大小写不敏感；Scope 中有 ASCII token 边界的 `Neo4j` 与 `cleanup`、`rebuild`、`sync` 之一共同编码“Neo4j projection operation”，不额外要求 Scope 补写字面量 `projection`。Layer 名称不参与 eligibility，不得用命名掩盖非 Neo4j 或危险 Scope。Before Assertions 中有边界的 `PG`/`PostgreSQL` 与 `baseline` 是事实源机器锚点。三个 operation token 是封闭集合，不接受 cleanupSuffix、resync、neo4jBackup、泛化的 migrate/delete/write/restore 或任意 R3 文案。现有通用校验继续要求 layer/package 映射、local 环境、唯一 layer、连续 order、Recovery Baseline、Expected Counts/Hash/Schema、Before/After Assertions、Stop Conditions 及 Proposal/tasks 一致。

这些静态谓词只判断 artifact 是否有资格表达 recovery，不代表操作已获授权。`.agents/openspec-workflow.md` 继续要求 PostgreSQL 是冻结、已验收、可完整重建该投影的唯一事实源，并要求用户在运行前对每个命名 R3 layer 单独明确授权；同一 package 中存在多个 layer 也不得合并授权。任何 drift、失败、超时、断言失败或人工中止立即停止，未执行 layer 不继承授权。

选择该方案，是因为 Scope 是 Stateful Layer Map 中唯一表达业务范围的字段，不能从 Layer 命名推定。替代方案 A 是放开全部 local R3，范围过宽，会错误覆盖 PostgreSQL cleanup 和不可替代状态；替代方案 B 是新增 `State Kind`、`Source Of Truth`、`Operation` 等列，结构更强但会扩大 schema、fixture 和所有 active change adoption，超出本次单根因修复。

### 2. 允许/禁止矩阵保持 fail-closed

| Environment | Risk | Scope / Recovery 条件 | 结果 |
|---|---|---|---|
| local | R2 | 满足现有 disposable local/test 规则 | 允许，行为不变 |
| local | R3 | Neo4j projection cleanup + PG baseline + 完整字段 + 命名 layer 独立授权 | 允许 |
| local | R3 | Neo4j projection rebuild + PG baseline + 完整字段 + 命名 layer 独立授权 | 允许 |
| local | R3 | Neo4j projection sync + PG baseline + 完整字段 + 命名 layer 独立授权 | 允许 |
| shared-local / uat / prod / shared | R2 或 R3 | 任意 disposable 声明 | 禁止，必须 backup 或等价正式灾备 |
| local | R3 | PostgreSQL cleanup、业务数据写入或其他非 Neo4j 状态 | 禁止 |
| local | R3 | Neo4j 非 projection 状态，或 operation 不在 cleanup/rebuild/sync | 禁止 |
| local | R3 | 未引用 PG baseline、PG 不是唯一事实源或不能完整重建 | 禁止 |
| local | R3 | Human 不是 yes、字段不完整、未逐层命名授权或跨层批量 | 禁止或不得执行 |

### 3. TDD 与验证边界

Apply 必须先扩充 `task_design_lint_test.go` 的 table-driven cases/fixture，记录新允许案例和每个负向边界的 RED 结果，再最小修改 `task_design_lint.go` 进入 GREEN。正例至少覆盖 local R2 regression 与 Scope 显式锚定的 local Neo4j R3 cleanup/rebuild/sync；反例至少覆盖 shared-local、uat、prod、Layer 合法但 Scope 非 Neo4j/非允许 operation、缺 PG baseline 和非 Human gate。

workflow 文本与主规格 delta 描述相同合取条件。修改共享规则和 architecture tests，因此 Apply final 运行 `go test ./internal/architecture` 与 backend `go test ./...`；另外运行 OpenSpec strict、当前 change explicit task-design lint、镜像受阻 change adoption-ready Scope 的 regression fixture、`git diff --check`、scope/status 和 secret pattern 检查。受阻 branch 的真实 explicit lint 只能在本规则 Deliver、该 branch 更新到最新 `origin/main` 并经 Review 补齐 Scope token 后运行，不能在本 change 中用跨 branch 修改伪造 adoption。测试不访问网络、数据库或 Neo4j，不新增依赖。

### 4. 受阻 change 的 adoption

本 change 只有在 Proposal Review、Apply、Apply-final Review、Sync、Archive、Deliver 全部完成并进入 `origin/main` 后，受阻 change 才能在自己的 Desktop-managed worktree 执行 adoption：

1. `git fetch origin` 并更新到最新 `origin/main`，检查共享 workflow/lint 文件没有冲突或漂移。
2. 在受阻 change 自己的 Proposal/tasks 中，将三个 Scope 分别补成明确有边界的 `Neo4j cleanup`、`Neo4j rebuild`、`Neo4j sync` 机器锚点，并连同已失效 blocker 叙述形成 scoped adoption diff 等待 Review；本 change 不代改 artifacts。
3. adoption Review 通过后运行 explicit lint，证明六条同根因错误消失且没有新增错误；该 lint 结果不追认既有授权，也不执行任何 layer。
4. cleanup、rebuild、sync 仍分别按命名 layer 重新展示 preflight、范围、断言和 stop conditions，并分别取得 R3 明确授权。

## Risks / Trade-offs

- [Scope 使用机器锚点，表达灵活性较低] → 固定封闭 token，宁可对含糊文案或仅 Layer 名合法的情况 fail-closed，不从命名或任意自然语言猜测。
- [静态 lint 不能证明 PG 实际可重建或用户已授权] → lint 只验证 artifact 资格；实际可重建性由冻结 baseline、只读 preflight、逐层 Review/Authorization 和写后 Query 证明。
- [共享 workflow 修改影响所有 change] → 保留默认拒绝，只新增合取式窄例外，并运行 repo-wide backend 验证与正反矩阵。
- [受阻 change 在本规则 Deliver 前误继续] → design 明确 adoption 顺序；Proposal checkpoint 不解除 blocker，也不构成 Apply 或 R3 授权。

## Migration Plan

1. Proposal checkpoint 后停止，等待人工 Review；批准前不修改 workflow/lint/tests。
2. Apply 阶段按 TDD 完成 tests、最小 lint 判定与 workflow 文本，刷新完整验证证据并再次等待人工 Review。
3. Apply-final Review 通过后才 Sync、Archive、Deliver。
4. 规则进入最新 `origin/main` 后，由受阻 change 独立 adoption；回滚本规则只需回退 workflow/lint/spec 的窄例外，不涉及数据恢复，因为本 change 无有状态写入。

## Open Questions

无。若 Review 要求新增 schema 列、扩大 operation 集合或覆盖其他 disposable R3 状态，应另行扩大 proposal，而不是在 Apply 中自行延伸。
