# OpenSpec Workflow

OpenSpec 是正式工程 change 的唯一生命周期和 artifacts 来源。正式变更必须先创建 change，再实现代码；Skill 映射见 `.agents/skill-routing.md`，Git 交付见 `.agents/git-workflow.md`。

## Language And Artifact Rules

- OpenSpec 内容默认使用中文；仅固定标题、关键字、命令、路径、代码标识和协议字段保留英文。
- 主规格 `openspec/specs/` 是已生效能力的事实来源；新 change 必须基于主规格和现有代码增量设计。
- Proposal、design、delta specs、tasks 和归档历史都归 OpenSpec 所有，不得建立平行长期事实来源。

## Lifecycle

```text
Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver
```

阶段不得跳过或调换：

1. **Explore**：读取相关主规格、现有 change 和代码，确认当前状态、范围、非目标和复用点；只读探索不得推定实现或写入授权。
2. **Propose**：创建中文 `proposal.md`、`design.md`、delta specs 和 `tasks.md`，明确影响范围、风险和验证方式。
3. **Review**：用户人工确认全部 artifacts。未获明确批准不得进入 Apply；Skill 默认流程或自动化不得替代人工 Review。
4. **Apply**：读取当前 change 全部 artifacts、受影响主规格、相关代码和命中任务规则，严格按 tasks 顺序实施；每完成一项立即更新 checkbox。
5. **Validate**：运行 OpenSpec CLI 与任务相关验证，读取新鲜结果；失败或未验证项必须明确报告。
6. **Sync**：tasks 全部完成且第二次人工 Review 通过后，才可将 delta specs 同步到主规格。
7. **Archive**：Sync 后归档 change，运行 `openspec validate --all` 并提交 scoped archive checkpoint；archive 不等于 delivered。
8. **Deliver**：Archive commit 存在且工作区无当前 change 未提交文件后，才可按 `.agents/git-workflow.md` push、PR/merge 和 cleanup。完成 Deliver 前不得宣称 change 关闭，也不得启动依赖其产物或接续其工作的 sequential successor change；用户明确批准且无依赖、无共享写状态的 independent parallel change 可按 Git 门禁独立启动。

## Review And Stateful Operation Gates

- Propose 后必须停在人工 Review；批准前不得修改生产规则、源码、数据库或图谱。
- Apply 完成后必须提供 scoped diff 与验证证据并再次等待人工 Review；批准前不得 Sync、Archive 或 Deliver。
- 数据库 migration/apply、seed、业务写入、图谱关系写入、图谱投影重建或清理等有状态操作，必须先展示范围、顺序、预期影响和回滚边界，再取得用户明确批准。
- 分层数据或图谱工作必须逐层执行 `Review -> Write -> Rebuild -> Query`；上一层未验收不得进入下一层。
- 只读审计、设计批准、Apply 批准或某一层写入批准都不得被解释为其他有状态操作的授权。

## 风险分级、阶段 Review package 与条件式执行包

本节是 R0—R3、人工 gate、阶段 Review package、候选数据审阅、R2/R3 recovery evidence、条件式执行包和 active change adoption 的详细唯一事实来源。根 `AGENTS.md` 只保留不可绕过摘要；Git、测试与 Skill 文件只维护各自专责规则。

### 风险等级和人工 gate

- R0：文档、调研、只读审计。artifact checkpoint 运行 OpenSpec validate、scoped diff 和 secret 检查。
- R1：源码或测试变更但无有状态写入。除 R0 证据外运行 targeted tests，并在 Apply final 运行受影响交付边界的完整验证。
- R2：migration、seed、本地/UAT 数据变更。必须具备明确授权、只读 preflight、可验证 recovery evidence 和 before/after state assertions。
- R3：生产、不可逆 cleanup、Neo4j rebuild 或敏感部署。必须独立明确授权及备份/恢复或等价灾难恢复证据；R3 不得跨层批量执行，R3 cleanup 必须单独成包。

change 在 Proposal 声明基线风险；阶段或命名操作可以上调，混合 change 按当前操作的最高风险执行。普通 task checkbox 不自动成为人工 gate。快速模式仅适用于 local 的普通 coding：Proposal Review 与 Apply-final Review 是唯一需要停顿并重新验收的 checkpoint；Proposal 批准后的 package 子项、targeted tests、修复和 scoped commit 是连续执行证据。真实 shared 写入、UAT/prod、migration/seed/data repair、secret/权限、回滚与 R2/R3 仍按各自严格 gate 执行。任何人工 Review、Authorization 或 Acceptance gate 必须记录风险等级、风险理由、所需证据、通过后允许的下一步和明确不授权的操作。

### Task Design Efficiency 与机器 schema

新 change 的 `tasks.md` 一级编号只表达内聚交付 package，普通实现、测试、修复、dry-run、validate、diff/secret check、commit 和 push 只能作为 package 内子项。人工 gate 仅允许 Spec/业务语义、R3、Neo4j、UAT/prod/shared、部署/secret/权限、scope/count/hash/schema 漂移或失败恢复、Apply-final、PR merge/cleanup；不得用普通 checkbox 或通用“等待确认”制造人工停顿。

Proposal 的 `## Gate Map` 必须是 `## Why` 后第一个二级 heading，tasks 的 `## Gate Map` 必须是第一个二级 heading。两份 artifact 的表必须逐行一致，固定列为：

```markdown
| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
```

- `Package` 是无前导零正整数，按行顺序与 tasks 的 `## <Package>. <name> Package` 一一对应；linter 只通过 Package ID 关联，不从 Gate 或 scope 文案猜测。
- `Gate` 与 `Allowed Scope` 是单行非空文本；`Risk` 只能是 `R0`、`R1`、`R2`、`R3`；混合 package 使用最高风险。
- `Human` 只能是小写 `yes` 或 `no`。`no` 的 Reason Code 必须为 `NONE`；`yes` 只能使用 `SPEC_SEMANTICS`、`R3_OPERATION`、`NEO4J`、`SHARED_ENV`、`DEPLOYMENT_SECURITY`、`DRIFT_RECOVERY`、`APPLY_FINAL`、`GIT_COMPLETION`。

`## Complexity Budget` 必须紧跟 Gate Map，Proposal 与 tasks 内容逐行一致，固定为：

```markdown
| Key | Value |
|---|---|
| human_gates | <integer> |
| stateful_layers | <integer> |
| checkpoints | <integer> |
| full_test_runs | <integer> |
| continuous_automation_scope | packages:<selector> |
```

五个 key 必须按该顺序各出现一次。前四个值是允许 `0` 的无符号十进制整数；`human_gates` 必须等于 `Human=yes` 行数。selector 使用升序、无重复的 package ID 或闭区间且只能引用 `Human=no` package，例如 `packages:2-4,6`；空范围写 `packages:none`。当全部 package 均为 `Human=yes` 时，`continuous_automation_scope=packages:none` 只表示不存在可跳过人工起始 gate 的 `Human=no` package，不否定 Proposal 批准后 package 子项连续执行。常见 R0/R1 change 通常只有 Proposal Review 与 Apply-final 两个人工 gate、一个 Apply package 和 Apply-final 一次完整验证；这两个 Review 是唯一需要停顿并重新验收的 checkpoint，Archive/Deliver 仅记录生命周期，在输入指纹不变时不新增人工停顿或重复全量验证。超出建议阈值只 warning，不以任意数量覆盖真实风险边界。

当 `stateful_layers=0` 时可以省略 `## Stateful Layer Map`；若保留，只能有固定表头而无数据行。当值大于零时，Proposal 与 tasks 必须在 Complexity Budget 后、首个编号 package 前提供逐行一致的固定表：

```markdown
| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
```

- `Layer` 是唯一 kebab-case；`Package` 只能引用 Gate Map 的 R2/R3 package；`Environment` 只能是 `local`、`shared-local`、`uat`、`prod`；`Order` 在每个 package 内从 1 连续且唯一。
- `Scope`、`Before Assertions`、`After Assertions`、`Stop Conditions` 必须非空；没有排除项时 `Exclusions` 写 `none`。
- `Recovery Evidence` 只能是 `backup` 或 `approved-disposable-recovery`。后者保留 local R2 的既有支持，并仅为 local R3 增加一个窄例外：Gate 必须为 `Risk=R3`、`Human=yes`，Scope 必须包含大小写不敏感且有 ASCII token 边界的 `Neo4j`，并包含同样有边界的 `cleanup`、`rebuild`、`sync` 之一，Before Assertions 必须包含有边界的 `PG` 或 `PostgreSQL` 以及 `baseline`。Layer 名称不得代替 Scope 证明业务范围；这些机器锚点只允许表达 recovery，不构成 R3 执行授权。`Recovery Baseline` 只能是 `new:<kebab-id>` 或 `reuse:<kebab-id>`。
- `reuse` 必须引用同 package、同 Environment 中更早 order 的 `new` baseline，并在 Before Assertions 复验 identity、scope、count、hash、schema；`Expected Counts/Hash/Schema` 固定为 `counts=<value>;hash=<value>;schema=<value>`，不适用项写 `na`。

local R3 的上述窄例外只适用于可由冻结、已验收 PostgreSQL 唯一事实源完整重建的 Neo4j projection cleanup、rebuild 或 sync。每个命名 layer 仍须在运行前取得独立明确授权，并严格逐层执行；任何 drift、失败、超时、冲突、断言失败或人工中止立即停止，未执行 layer 的授权失效。`shared-local`、UAT、prod/shared、生产、非 Neo4j R3、非限定 operation、缺少 PG/PostgreSQL baseline 或无法完整重建的状态必须继续使用 `backup` 或等价正式灾备。本规则不定义或引入 UAT Neo4j recovery、backup、deployment、adoption 或验收能力。

已批准 Spec 且范围精确匹配的 local-only R2 可以在一个独立授权 package 内逐名列出多个 layer。每层严格执行 `preflight -> Write -> Query/assert`；当前层断言全部通过后才自动进入包内已明确命名的下一层，任何漂移、失败、超时、冲突或人工中止立即停止并使未执行层授权失效。同一环境、同一维护窗口且 identity、scope、count/hash/schema 未变化时，后续层可以复验并复用 recovery baseline；不一致时必须停止并重新建立 recovery evidence。该简化不适用于 shared-local、UAT、prod、Neo4j 或 R3。

local 快速模式下的一次精确范围数据写入只允许一次完整 preflight -> 单次 Write -> 一次 verify；它必须在该单次 write scope 的明确授权、recovery evidence 和 before/after assertions 内连续完成。多个命名 R2 layer、shared-local、UAT、prod、Neo4j 或 R3 不得借此合并或跳过既有逐层授权、断言和停止语义。

### Task-design lint 与 legacy baseline

task-design lint 复用 `backend/internal/architecture/` 的 Go 标准库测试模式，不引入新依赖、wrapper 或平行 CI job。确定性违规 fail，包括固定 heading/table schema、枚举、package 映射、预算整数/selector、stateful count/字段/order/recovery、Proposal/tasks 不一致、baseline 格式和 explicit scope；一级 package 数过多、相邻微型 Review/checkpoint 或测试/dry-run/commit/push 疑似被提升为 package 等启发式判断只 warning。

- active mode：`OPENSPEC_TASK_LINT_CHANGE` 未设置时枚举 `openspec/changes/<change>/` 的直接 active 目录；archive 目录始终先排除，再处理 legacy baseline。现有 backend `go test ./...` 自动运行该模式。
- explicit mode：设置 `OPENSPEC_TASK_LINT_CHANGE=<change-name>` 后只校验该单段 kebab-case active change，即使它仍在 baseline；archive、路径分隔符或未知 change 必须 fail，不回退到 active mode。
- 从 `backend/` 运行的精确命令是 `OPENSPEC_TASK_LINT_CHANGE=<change-name> go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1`。

legacy baseline 固定为 repo-local `.agents/openspec-task-lint-baseline.tsv`，UTF-8 header 精确为 `change_name<TAB>reason`。每行只登记本规则 Deliver 时仍 active 的 kebab-case change name 和不含 TAB/CR/LF 的非空单行 reason。该文件语义归本 workflow 所有，只能由 scoped workflow/adoption change 维护，linter 不自动改写。

- active mode 仅在 baseline 名称仍对应 active change 时跳过，并输出 reason 与 adoption 后移除提示；显式 adoption 后必须删除对应行。
- 重复、已归档、未知或 adoption 后残留条目至少 warning，且不产生新的跳过能力；header、空字段、非法 name 或多余列 fail。
- archive 在 baseline 前排除且历史不扫描、不改写。explicit mode 始终绕过 baseline skip，使 legacy change 可以被单独整改和验收。

### 阶段 Review package 与验证选择

同一风险边界内的 contract、实现、测试、dry-run、只读 preflight、diff、候选异常清单和验证结果可以组成一个阶段 Review package。package 必须记录 scope、non-goals、风险等级、证据、未验证项、阻断项和下一步授权边界。Proposal 批准后的内聚 package 及其 scoped commit 是连续执行证据，不得把每个微型 task 自动升级为 commit、push 或人工 Review；连续执行证据必须记录阶段、commit、已通过验证、输入状态指纹、下一步和真实 blocker。Archive/PR 在输入 commit、manifest、schema、baseline、environment 指纹未变化且证据未失败时复用对应证据，只重验变化或失败层。

开发中按当前失败或实现运行 targeted tests；连续 Apply package 在其风险边界内运行一次与整个 package 匹配的验证；Apply final 运行一次受影响交付边界完整验证。任意 change 必须先按真实受影响交付边界选择验证，不得因其是正式 change、共享开发规则或未来可能部署而机械扩大为业务 repo-wide 测试：

- OpenSpec artifacts、workflow 文本、agent rules、architecture test/lint 自身的变更运行对应 OpenSpec/architecture/规则 targeted validation、精确 task-design lint、diff/scope/secret/链接检查，不自动触发业务 Go 全量测试。
- 局部 coding 运行 targeted tests 与受影响 package/module 的完整 suite。
- 数据-only change 运行 manifest/dry-run/preflight/post-write assertions；没有代码影响时不机械运行业务 unit tests。
- 只有修改共享运行时代码、跨模块运行时契约、公共运行时基础设施，或影响边界无法可靠确定时，才运行 repo-wide full validation；Go module 对应 `go test ./...`，前端同理按受影响 workspace。

UAT/prod/shared/stateful 安全门禁不由测试范围优化削弱；R2/R3 的 recovery、授权、before/after assertions 和停止语义保持不变。验证记录必须写明受影响边界、共享 tests 和 repo-wide 判定理由；边界、理由或 suite 不清楚时 fail-closed，必须扩大到 repo-wide full validation 或停止等待澄清。失败与修复留在同一 package，不新增人工 gate；任何失败、环境限制或未验证项必须进入 package，不能用旧日志替代。

运行命令消费的数据不得依赖 active OpenSpec path。实际存在该运行依赖时，R1 Apply 必须将其迁移到稳定 backend `data/` 或 `resource/` 路径，并保留 OpenSpec review snapshot/hash/evidence；若搜索确认不存在 active runtime consumer，则记录检查结果并跳过迁移，不得为满足规则制造副本、runner 或默认 path test。

task agent 在通知主对话验收前必须完成 self-review/code review，复读测试结果并检查 diff、scope、secret 和需求覆盖；发现阻断问题必须先整改并刷新证据。该内部审查不替代 Proposal 后或 Apply 后人工 Review。

### 候选数据审阅

候选数据 package 必须包含生成规则与输入指纹、总体 counts、全量机器校验、异常/冲突清单、宽边界清单、低置信度清单、用户明确指定项、预期动作分类和 fail-closed 条件。普通正常项通过全量机器校验与总体断言后无需机械逐条人工确认；异常、冲突、宽边界、低置信度和用户明确指定项必须逐项审阅。已批准规格要求逐项确认的 final manifest 不得被该策略降级。

### R2 条件式执行包

条件式执行包是独立授权对象，不由普通 Apply、旧批准或上一层批准推定。用户必须一次明确授权包内每个命名操作的环境、顺序、范围、排除范围、recovery evidence、预期 counts、before/after assertions 和停止条件。

每个 R2 层必须逐一选择：

- `backup`：shared local、开发主数据、UAT 或任何不可替代数据必须提供可恢复备份；
- `approved disposable recovery`：仅限用户逐层批准且明确声明 disposable、没有不可替代数据、具备确定性 recreate/reseed 路径的 local/test；必须记录环境身份、声明、重建/重灌命令、预计耗时和验证断言。

当前 tidewise 本地 curated PostgreSQL 不得自动视为 disposable。未逐层声明 recovery evidence、证据不成立、重建/验证失败、范围漂移、断言失败或触发停止条件时必须 fail-closed：不得执行或继续该层，未执行层的剩余授权自动失效，重新执行必须取得新授权。

执行顺序严格为 `preflight -> Write(layer N) -> Query/assert(layer N)`；只有本包内已逐名明确授权的下一层，且当前层全部自动断言通过，才可继续。下一层因执行包中被显式命名而获授权，不是从上一层结果推定。R3 操作不得放入 R2 条件式执行包。

### Active change adoption

本规则 Deliver 后创建的新 change 默认使用新规则。Deliver 时仍 active 的 legacy change 必须登记在 `.agents/openspec-task-lint-baseline.tsv`。active change 不自动重写、不追认历史操作、不取消已开始写操作的验收、不扩大既有授权。

每个 active change 采用新规则前，必须在其 branch 执行 `git fetch origin` 并更新到最新 `origin/main`，检查共享规则和 tasks 冲突，提交仅包含未来 gate 的 scoped workflow-adoption tasks diff，并经用户一次人工 Review；adoption diff 必须删除对应 baseline 行，使后续 active lint 生效。adoption 只能合并尚未开始的未来 gate；未命名的新环境、新层、新范围或 Neo4j rebuild 仍需独立授权。

## Starting Or Continuing A Change

开始新 change 前必须：

- 读取 `openspec/config.yaml`、受影响主规格、相关代码及命中的 `.agents` 规则。
- 确认 scope、non-goals、复用点和影响区域，禁止创建平行结构。
- 通过 `.agents/git-workflow.md` 的 New Change Gate；OpenSpec 不重复维护 branch/worktree 操作细节。

继续已有 change 时只读取：

- 该 change 的 proposal、design、tasks 和 delta specs。
- 受影响主规格和相关代码。
- 由实际生命周期、Git、后端、前端或测试动作命中的对应 `.agents` 文件。

不得因为开始任务而无差别读取全部规则、全部主规格或项目外文档。

## Apply Rules

- 实现前说明复用哪些已有页面、组件、services、models、data、store、repository 或配置。
- 不得在未检查现有实现前生成平行页面、service、model、store、data 或 config 层。
- Go 后端功能、bugfix 或重构必须服从 `.agents/testing-tdd.md`，先测试再实现。
- 若 design/spec/tasks 与代码现实冲突，立即暂停；先更新 artifacts 或征求用户确认，不得在过期设计上继续。
- 用户要求暂停时保留 change 和 tasks 状态，使其可恢复。

## Design Requirements

复杂后端 change 的 `design.md` 必须包含真实边界图示：

- 后端流程、跨模块调用、外部 API、scheduler、connector、事件抽取、图谱写入、异步任务或部署边界：Mermaid sequence diagram。
- 新增核心类型、接口、adapter、repository、service、parser、connector、worker 或跨包依赖：Mermaid class 或 component diagram。
- 简单配置、文案或小范围测试修复可以不画图，但 design 必须说明原因。

## Completion Gates

进入 Sync 前必须同时满足：

- tasks 全部完成并有对应验证证据。
- design、delta specs 和实现一致。
- 用户已完成 Apply 后人工 Review。

进入 Archive 前必须完成 Sync；进入 Deliver 前必须完成 Archive、`openspec validate --all` 和 archive commit。Git 提交、PR、merge、branch/worktree cleanup 的完整操作只在 `.agents/git-workflow.md` 维护。
