## ADDED Requirements

### Requirement: Proposal 必须声明任务包装契约
系统 MUST 要求正式 change 的 Proposal 与 tasks 开头提供可机器读取且内容一致的 Gate Map 与 Complexity Budget。Proposal 的 `## Gate Map` MUST 是 `## Why` 后第一个二级 heading，tasks 的 `## Gate Map` MUST 是第一个二级 heading；其固定列 MUST 依次为 `Package`、`Gate`、`Risk`、`Human`、`Reason Code`、`Allowed Scope`。`Package` MUST 是与 `## <Package>. <name> Package` 一级 heading 一一对应的不带前导零正整数；`Risk` MUST 精确为 `R0`、`R1`、`R2` 或 `R3`；`Human` MUST 是小写 `yes` 或 `no`；`Human=yes` 的 Reason Code MUST 精确为 `SPEC_SEMANTICS`、`R3_OPERATION`、`NEO4J`、`SHARED_ENV`、`DEPLOYMENT_SECURITY`、`DRIFT_RECOVERY`、`APPLY_FINAL` 或 `GIT_COMPLETION`，`Human=no` MUST 使用 `NONE`。linter MUST 通过 Package ID 关联 Gate Map 与一级 package，不得从自然语言猜测。

Complexity Budget MUST 紧跟 Gate Map，固定为 `Key`、`Value` 两列并依次包含 `human_gates`、`stateful_layers`、`checkpoints`、`full_test_runs`、`continuous_automation_scope`。前四个值 MUST 是允许零值的无符号十进制整数；连续执行范围 MUST 使用 `packages:<selector>`，selector 只能引用存在且 `Human=no` 的升序无重复 package ID/闭区间，空范围为 `packages:none`。Proposal 与 tasks 的两张机器表 MUST 经空白规范化后逐行相同。

#### Scenario: 创建新的 R0 或 R1 change
- **WHEN** Agent 为无有状态写入的正式 change 编写 Proposal 与 tasks
- **THEN** artifacts 必须按固定 heading、表头、枚举、整数和 selector schema 提供一致的 Gate Map 与 Complexity Budget，并把实现、测试、修复、验证和 Git 操作组织为对应 package 子项

#### Scenario: Gate Map 与一级 package 对应
- **WHEN** tasks 包含编号为 1、2、3 的三个一级 package
- **THEN** Gate Map 必须按相同顺序各包含唯一 Package 1、2、3，且 linter 必须对缺失、重复、多余或顺序不一致 fail

#### Scenario: Complexity Budget 可机器解析
- **WHEN** Agent 声明零个 stateful layer、两个人工 gate 和 package 2 连续自动执行
- **THEN** 固定值必须分别为 `stateful_layers=0`、`human_gates=2`、`continuous_automation_scope=packages:2`，不得混入单位或说明文字

#### Scenario: 复杂度预算超过常见阈值
- **WHEN** Proposal 声明的人工 gate、checkpoint 或完整测试次数高于同风险 change 的建议阈值但字段完整且风险理由合法
- **THEN** lint 必须输出 warning 供 self-review，不得仅因启发式数量阻断 proposal

### Requirement: Stateful Layer Map 必须完整映射有状态层
系统 MUST 在 `stateful_layers>0` 时要求 Proposal 与 tasks 在 Complexity Budget 后、首个编号 package 前提供内容一致的 `## Stateful Layer Map`。固定列 MUST 依次为 `Layer`、`Package`、`Environment`、`Order`、`Scope`、`Exclusions`、`Recovery Evidence`、`Recovery Baseline`、`Expected Counts/Hash/Schema`、`Before Assertions`、`After Assertions`、`Stop Conditions`。`Layer` MUST 是唯一 kebab-case；`Package` MUST 引用 Risk 为 R2/R3 的 Gate Map package；`Environment` MUST 是 `local`、`shared-local`、`uat` 或 `prod`；`Order` MUST 在 package 内从 1 连续且唯一；空 exclusions MUST 写 `none`。Recovery Evidence MUST 是 `backup` 或 `approved-disposable-recovery`，Recovery Baseline MUST 是 `new:<kebab-id>` 或引用同环境更早 order 的 `reuse:<kebab-id>`；expected state MUST 同时提供 `counts=<value>;hash=<value>;schema=<value>`，不适用值写 `na`。其余范围与断言字段 MUST 单行非空。

当 `stateful_layers=0` 时系统 MUST 允许省略 Stateful Layer Map；若仍提供该表，则 MUST 使用固定表头且不得包含数据行。linter MUST 校验数据行数等于预算值，并只通过 Package ID 关联 layer 与 package。

#### Scenario: 无有状态层省略 Stateful Layer Map
- **WHEN** Complexity Budget 声明 `stateful_layers=0`
- **THEN** Proposal 与 tasks 可以省略 `## Stateful Layer Map`，且 lint 不得要求空占位表

#### Scenario: 多层条件式执行包完整映射
- **WHEN** 一个 R2 package 包含两个已命名 local layer
- **THEN** Stateful Layer Map 必须有两行、引用同一 Package、使用连续 Order 1 和 2，并完整提供环境、范围、排除、recovery、expected state、前后断言和停止条件

#### Scenario: Stateful layer 无法对应 package
- **WHEN** layer 引用不存在的 Package，或引用 Risk 为 R0/R1 的 package
- **THEN** task-design lint 必须 fail，不得根据 layer 名称或 scope 文案猜测归属

#### Scenario: 复用 recovery baseline
- **WHEN** 某行使用 `reuse:<baseline-id>`
- **THEN** 同一 Environment 的更早 Order 必须已有 `new:<baseline-id>`，before assertions 必须复验 identity、scope、count/hash/schema；否则 lint 必须 fail

### Requirement: 人工 gate 必须使用限定风险原因
系统 MUST 只允许以下人工 gate 原因：Spec/业务语义、R3、Neo4j、UAT/prod/shared、部署/secret/权限、scope/count/hash/schema 漂移或失败恢复、Apply-final、PR merge/cleanup。普通源码实现、测试/修复、dry-run、validate、diff/secret check、commit/push MUST NOT 单独成为人工 gate。

#### Scenario: 普通实现与验证属于同一 package
- **WHEN** Agent 在同一 R0/R1 风险边界内执行源码实现、测试、dry-run、validate、diff/secret check、commit 或 push
- **THEN** 这些步骤必须作为 package 内子项连续执行，不得分别请求人工 Review

#### Scenario: Gate Map 使用非法原因
- **WHEN** 人工 gate 的原因不是规范允许的语义、安全、环境、漂移恢复、Apply-final 或 Git 完成边界
- **THEN** task-design lint 必须失败并指出该 gate 的非法原因

### Requirement: Task-design lint 必须可靠且保持轻量
系统 SHALL 复用 `backend/internal/architecture/` 的 Go 标准库测试模式实现无新依赖的 task-design lint。active mode MUST 在 `OPENSPEC_TASK_LINT_CHANGE` 未设置时扫描 active changes；explicit mode MUST 在该变量设置为单段 kebab-case change name 时只检查指定 active change并忽略 baseline 跳过。Proposal/package checkpoint 的精确 explicit 命令 MUST 是从 `backend/` 运行 `OPENSPEC_TASK_LINT_CHANGE=<change-name> go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1`；现有 CI 的 `go test ./...` MUST 自动运行 active mode。

legacy baseline MUST 使用 repo-local `.agents/openspec-task-lint-baseline.tsv`，固定 UTF-8 TSV header 为 `change_name<TAB>reason`，只列规则 Deliver 时仍 active 的 kebab-case change name 与非空且不含 TAB/CR/LF 的单行 reason。其语义 MUST 归 `.agents/openspec-workflow.md` 所有，并只能由 scoped OpenSpec workflow/adoption change 维护；linter MUST NOT 自动改写。archive MUST 在 baseline 前自动排除；每个仍有效的 active skip MUST 输出包含 reason 与 adoption 移除提示的 warning；已归档、未知、重复或 adoption 后未移除的条目 MUST 至少 warning 且不得产生新的静默跳过能力；非法 header、空字段、非法 name 或多余列 MUST fail。

lint MUST 校验固定 Markdown schema、Gate Map/package 对应、Complexity Budget、Stateful Layer Map、合法人工 gate 与两份 artifacts 一致性；可静态可靠判断的违规 MUST fail，启发式复杂度 MUST 只 warning。fixture MUST 覆盖合规 zero-stateful、合规 multi-layer、schema/mapping 违规、baseline stale/duplicate/unknown/archived、explicit mode 与 warning。

#### Scenario: tasks 缺少确定性必填结构
- **WHEN** active 或显式传入 change 的 artifacts 缺固定 Gate Map/Complexity Budget schema、合法人工 gate 原因、package 对应，或 stateful package 缺固定 layer 字段
- **THEN** lint 必须失败并返回包含 artifact、section 与字段的可定位错误

#### Scenario: tasks 疑似过度拆分
- **WHEN** lint 发现重复微型 Review/checkpoint 或把测试、dry-run、commit/push 疑似提升为一级 package，但无法仅靠结构可靠判定违规
- **THEN** lint 必须输出 warning 和复核依据，不得使验证失败

#### Scenario: lint 运行作用域
- **WHEN** CI 运行 active mode 或开发者明确传入一个 change
- **THEN** active mode 必须排除 archive 并仅跳过 baseline 中仍 active 的 change，explicit mode 必须只校验指定 change且不得因 baseline 跳过

#### Scenario: baseline 包含过期或重复条目
- **WHEN** baseline 的 change 已归档、目录未知、重复，或已 adoption 但条目未移除
- **THEN** lint 必须输出 warning 并停止对无效条目提供跳过能力，不能无限静默忽略

#### Scenario: explicit mode 指向 archive 或未知 change
- **WHEN** `OPENSPEC_TASK_LINT_CHANGE` 包含 archive、路径分隔符或不存在的 change name
- **THEN** lint 必须 fail，不得扩大扫描范围或回退到 active mode

#### Scenario: lint 接入现有 CI
- **WHEN** 仓库现有 backend CI 运行 `go test ./...`
- **THEN** task-design lint 必须随工作流架构测试执行，不新增第三方依赖或平行 CI job

## MODIFIED Requirements

### Requirement: 普通任务不得自动成为人工门禁
系统 SHALL 将普通 task checkbox 作为 package 内可验证工作单元，而不是自动人工门禁。人工 gate MUST 在 Gate Map 标注风险等级、合法风险原因、所需证据、通过后允许的下一步和明确不授权的操作；一级 task MUST 表达内聚交付 package。

#### Scenario: 完成微型实现任务
- **WHEN** Agent 完成一个不跨越授权边界的微型 task
- **THEN** Agent 必须将其证据汇入所属 package，不得仅因 checkbox 完成而要求一次独立人工 Review

#### Scenario: 设置人工 gate
- **WHEN** 某一步需要用户人工 Review、Authorization 或 Acceptance
- **THEN** Gate Map 必须说明该 gate 所控制的合法风险原因与授权边界，不能只写通用“等待确认”

#### Scenario: 普通操作被单独包装为 gate
- **WHEN** tasks 将测试、修复、dry-run、validate、diff/secret check、commit 或 push 单独标为人工 gate
- **THEN** task-design lint 必须失败，Agent 必须把该操作合并回所属 package

### Requirement: 阶段 Review package 必须聚合一致风险边界内的证据
系统 SHALL 允许 contract、实现、测试、修复、dry-run、只读 preflight、validate、diff/secret check、commit/push 和异常清单组成同一风险边界的 package。package MUST 说明 scope、non-goals、风险等级、证据、未验证项、阻断项、停止条件和下一步授权边界，且不得绕过 Proposal 后或 Apply 后人工 Review。

#### Scenario: R1 阶段形成 Review package
- **WHEN** contract、实现和 targeted tests 已在无状态写入的同一阶段完成
- **THEN** Agent 必须用一个 package 提交验收，而不是为每个微型 task 分别 commit、push 或 Review

#### Scenario: package 内测试失败并修复
- **WHEN** package 内 targeted test、validate 或 lint 失败且修复不扩大风险与 scope
- **THEN** Agent 必须在同一 package 内修复并刷新证据，不得为失败与修复新增人工 gate

#### Scenario: package 涉及状态写入
- **WHEN** package 包含 R2 或 R3 有状态操作
- **THEN** 普通阶段 Review 不得隐含写入授权，Agent 必须另行提交满足对应风险等级的明确授权对象

### Requirement: 验证深度必须随风险与生命周期递增
系统 MUST 以 package 为单位组织验证：开发中运行与当前工作匹配的 targeted tests；package checkpoint 运行一次范围匹配验证；Apply final 运行一次受影响 app/module/package 的完整 suite 与共享 architecture/contract tests。只有共享规则、跨模块契约、公共基础设施或 repo-wide 变更才 MUST 运行 repo-wide full validation。验证选择 MUST 记录受影响交付边界、共享 tests 与 repo-wide 判定理由；边界、理由或 suite 不清楚时 MUST fail-closed。R2 与 R3 MUST 额外提供执行前后状态断言，任何失败或未验证项必须明确报告。

#### Scenario: 开发中运行 targeted tests
- **WHEN** Agent 在 package 内实现、调试或修复
- **THEN** Agent 可以按需重复 targeted tests，但不得把每次测试运行升级为 checkpoint 或人工 Review

#### Scenario: R1 package checkpoint
- **WHEN** Agent 准备提交无有状态写入的 package checkpoint
- **THEN** Agent 必须运行一次与整个 package 范围匹配的验证，不需要在每个微型 task 后重复完整验证

#### Scenario: Apply final 验证
- **WHEN** change 完成 Apply 并准备请求 Apply 后人工 Review
- **THEN** Agent 必须运行一次受影响交付边界完整 suite、共享 architecture/contract tests、OpenSpec strict validation、diff/scope/secret 检查，并汇总所有 R2/R3 pre/post evidence

#### Scenario: 共享规则 change 的 Apply final
- **WHEN** change 修改共享规则、跨模块契约、公共基础设施或其他 repo-wide 行为
- **THEN** Agent 必须运行 repo-wide full validation；本 workflow change 修改全项目规则与 architecture tests 时必须运行 `go test ./...` 和相关 OpenSpec/规则检查

#### Scenario: 验证边界无法明确
- **WHEN** Agent 无法明确受影响交付边界、完整 suite 或是否触发 repo-wide 条件
- **THEN** Agent 必须 fail-closed，扩大到 repo-wide full validation 或停止等待澄清，不得自行省略测试

#### Scenario: R2 操作断言失败
- **WHEN** R2 命名操作的 post-state、counts、保护或幂等断言任一失败
- **THEN** Agent 必须立即停止，不得继续后续层，也不得用旧验证证据替代失败结果

### Requirement: 候选数据必须采用全量机器校验和异常聚焦审阅
系统 SHALL 要求规模化候选数据 package 提供生成规则与输入指纹、总体 counts、全量机器校验、异常/冲突清单、宽边界清单、低置信度清单、用户明确指定项和 fail-closed 条件。正常项不得被机械要求全部逐条审阅；异常、冲突、宽边界、低置信度及用户明确指定的清单 MUST 逐项人工审阅。

#### Scenario: 大量正常候选通过全量校验
- **WHEN** 候选由固定规则生成且全部正常项通过机器校验与总体断言
- **THEN** 用户可以审阅生成规则、counts 和例外清单，不需要机械逐条确认全部正常记录

#### Scenario: 候选存在异常或低置信度
- **WHEN** 候选包含异常、冲突、宽边界或低置信度项
- **THEN** 系统必须把这些项列入人工清单逐项审阅，未决项必须阻断后续写入

#### Scenario: 用户指定人工审阅项
- **WHEN** 用户明确指定某些候选或类别必须人工确认
- **THEN** 系统必须将其加入人工清单，不得因机器校验通过而跳过

#### Scenario: 业务契约要求 final manifest 逐项确认
- **WHEN** 已批准规格明确要求某个 final manifest 由用户逐项确认
- **THEN** 全量机器校验不得取消该人工决策，只能用于其余正常候选的证据组织

### Requirement: R2 条件式执行包必须逐层显式授权并声明 recovery evidence
系统 SHALL 允许用户在一次明确授权中预授权多个 Spec 已批准且精确匹配的 local-only R2 命名层，但执行包 MUST 逐层列出每个命名操作、环境、顺序、范围、排除范围、recovery evidence、预期 counts、before/after assertions 和停止条件。每层 MUST 严格执行 `preflight -> Write -> Query/assert`；只有当前层全部断言通过，才可自动进入包内已逐名授权的下一层。每层 recovery evidence MUST 明确选择可恢复备份或经批准的 disposable recovery。同一环境、同一维护窗口且基础状态未变化时，可以复验并引用同一 recovery baseline；复验不一致 MUST 立即停止。shared local、开发主数据、UAT 或任何不可替代数据 MUST 提供可恢复备份，当前 tidewise 本地 curated PostgreSQL MUST NOT 自动视为 disposable。普通 Apply 批准、旧批准或上一层批准 MUST NOT 被解释为该执行包授权。

#### Scenario: 用户一次授权多个 local-only R2 命名层
- **WHEN** 执行包逐名列出 Layer A 和 Layer B 的全部授权字段、每层 recovery evidence 且用户明确批准整个包
- **THEN** Agent 可以严格执行 `preflight A -> Write A -> Query/assert A -> preflight B -> Write B -> Query/assert B`

#### Scenario: 同一维护窗口复用 recovery baseline
- **WHEN** 下一命名层与上一层处于同一环境和维护窗口，且环境身份、scope、count/hash/schema 的复验与 recovery baseline 一致
- **THEN** Agent 可以引用已验证 baseline，不需要为该层重新制造完整 backup package

#### Scenario: recovery baseline 复验漂移
- **WHEN** 环境身份、scope、count/hash/schema 与已验证 recovery baseline 不一致
- **THEN** Agent 必须立即停止，未执行层授权失效，并重新建立 recovery evidence 后请求必要授权

#### Scenario: disposable local test 层使用重建证据
- **WHEN** 某 local/test 层被用户逐层批准为 disposable，且环境没有不可替代数据并提供确定性 recreate/reseed 命令、预计耗时和验证断言
- **THEN** Agent 可以以 approved disposable recovery 作为该层 recovery evidence，而不是物理备份

#### Scenario: shared 或不可替代数据层
- **WHEN** R2 层涉及 shared local、开发主数据、UAT 或任何不可替代数据
- **THEN** Agent 必须在该层执行前提供可恢复备份，不得使用 baseline 简化取消 recovery evidence 或使用 disposable recovery

#### Scenario: 上一层断言通过
- **WHEN** Layer A 的全部自动断言通过且 Layer B 已在同一执行包中被逐名明确授权
- **THEN** Agent 可以进入 Layer B，因为 Layer B 已被显式授权，而不是从 Layer A 的批准推定

#### Scenario: 任一断言、范围或 recovery evidence 失败
- **WHEN** 某层断言失败、实际范围漂移、recovery evidence 不成立、出现冲突或触发停止条件
- **THEN** Agent 必须立即停止，所有未执行层的剩余授权自动失效，重新执行必须取得新授权

#### Scenario: 执行包使用概括性后续范围
- **WHEN** 执行包只写“其余层”“后续数据”或其他未逐名范围
- **THEN** 这些未命名操作不在授权范围内，Agent 不得执行

### Requirement: 新规则必须通过显式 adoption 应用于 active change
系统 MUST 让本规则 Deliver 后创建的新 change 默认使用新流程；active change MUST 保持历史 artifacts、任务包装和授权不变，并在 `.agents/openspec-task-lint-baseline.tsv` 中以 change name 和 reason 显式登记，直到其 branch 更新最新 `origin/main`、提交 scoped workflow-adoption tasks diff并通过一次用户人工 Review。adoption diff MUST 移除对应 baseline 行，使后续 active lint 生效。adoption 只能合并未来 gate，不能追认历史操作、取消已开始写操作的验收或扩大既有授权。

#### Scenario: Active change 尚未 adoption
- **WHEN** 本规则已经 Deliver 但某 active change 尚未提交并通过 adoption Review
- **THEN** 该 active change 必须继续按原 tasks 与授权边界执行，并只能通过 baseline 中可见的有效行被 active lint 跳过；explicit lint 仍可用于审阅

#### Scenario: Adoption 合并未来 R2 gates
- **WHEN** active change 的未开始 schema、seed 和 mapping 层被 scoped tasks diff 逐名组织为 R2 条件包并通过用户 Review
- **THEN** 这些未来层可以使用新条件式执行包，adoption diff 必须移除 baseline 行，但已开始的写操作仍按原验收完成

#### Scenario: Adoption 试图扩大旧授权
- **WHEN** adoption diff 把既有批准扩展到新环境、新层、新范围或 Neo4j rebuild
- **THEN** 用户必须拒绝该 adoption 范围，系统不得把旧批准解释为新授权
