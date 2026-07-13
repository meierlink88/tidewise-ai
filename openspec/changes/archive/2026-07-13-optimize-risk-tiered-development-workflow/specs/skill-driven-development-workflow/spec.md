## ADDED Requirements

### Requirement: 正式研发必须使用统一风险等级
系统 MUST 为每个正式 change 声明 R0—R3 基线风险，并按具体阶段或命名操作上调：R0 为文档、调研和只读审计；R1 为源码或测试变更但无有状态写入；R2 为 migration、seed 或本地/UAT 数据变更；R3 为生产、不可逆清理、Neo4j rebuild 或敏感部署。混合风险 change MUST 按当前操作的最高适用等级执行 gate。

#### Scenario: 静态 migration 代码与实际 apply 风险不同
- **WHEN** Agent 只修改并验证 migration 代码而不执行 apply
- **THEN** 该阶段可以按 R1 管理，但实际在本地或 UAT apply migration 必须上调为 R2

#### Scenario: cleanup 属于高风险操作
- **WHEN** cleanup 被判定为不可逆或高破坏性
- **THEN** 系统必须将其标为 R3 并要求独立授权，不得与普通 schema 或 seed 写入合并

### Requirement: 普通任务不得自动成为人工门禁
系统 SHALL 将普通 task checkbox 作为可验证工作单元，而不是自动人工门禁。人工 gate MUST 标注风险等级、风险理由、所需证据、通过后允许的下一步和明确不授权的操作。

#### Scenario: 完成微型实现任务
- **WHEN** Agent 完成一个不跨越授权边界的微型 task
- **THEN** Agent 必须将其证据汇入所属阶段 Review package，不得仅因 checkbox 完成而要求一次独立人工 Review

#### Scenario: 设置人工 gate
- **WHEN** 某一步需要用户人工 Review、Authorization 或 Acceptance
- **THEN** tasks 必须说明该 gate 所控制的实际风险与授权边界，不能只写通用“等待确认”

### Requirement: 阶段 Review package 必须聚合一致风险边界内的证据
系统 SHALL 允许 contract、实现、测试、dry-run、只读 preflight、diff 和异常清单组成同一阶段 Review package。package MUST 说明 scope、non-goals、风险等级、证据、未验证项、阻断项和下一步授权边界，且不得绕过 Proposal 后或 Apply 后人工 Review。

#### Scenario: R1 阶段形成 Review package
- **WHEN** contract、实现和 targeted tests 已在无状态写入的同一阶段完成
- **THEN** Agent 可以用一个阶段 package 提交验收，而不是为每个微型 task 分别 commit、push 或 Review

#### Scenario: package 涉及状态写入
- **WHEN** package 包含 R2 或 R3 有状态操作
- **THEN** 普通阶段 Review 不得隐含写入授权，Agent 必须另行提交满足对应风险等级的明确授权对象

### Requirement: 验证深度必须随风险与生命周期递增
系统 MUST 在 R0 artifact checkpoint 运行 OpenSpec validate、scoped diff 和 secret 检查；在 R1 checkpoint 增加 targeted tests。Apply final MUST 完整运行受影响 app/module/package 的 suite 与共享 architecture/contract tests；只有共享规则、跨模块契约、公共基础设施或 repo-wide 变更才 MUST 运行 repo-wide full validation。验证选择 MUST 记录受影响交付边界、共享 tests 与 repo-wide 判定理由；边界、理由或 suite 不清楚时 MUST fail-closed，不得自行降级。R2 与 R3 MUST 额外提供执行前后状态断言，且任何失败或未验证项必须明确报告。

#### Scenario: R1 阶段 checkpoint
- **WHEN** Agent 准备提交无有状态写入的阶段 checkpoint
- **THEN** Agent 必须运行与修改范围匹配的 targeted tests，但不需要在每个微型 task 后重复全量验证

#### Scenario: Apply final 验证
- **WHEN** change 完成 Apply 并准备请求 Apply 后人工 Review
- **THEN** Agent 必须运行受影响交付边界的完整 suite、共享 architecture/contract tests、OpenSpec strict validation、diff/scope/secret 检查，并汇总所有 R2/R3 pre/post evidence

#### Scenario: 共享规则 change 的 Apply final
- **WHEN** change 修改共享规则、跨模块契约、公共基础设施或其他 repo-wide 行为
- **THEN** Agent 必须运行 repo-wide full validation；本 workflow change 修改全项目规则与 architecture tests 时必须运行 `go test ./...` 和相关 OpenSpec/规则检查

#### Scenario: 验证边界无法明确
- **WHEN** Agent 无法明确受影响交付边界、完整 suite 或是否触发 repo-wide 条件
- **THEN** Agent 必须 fail-closed，扩大到 repo-wide full validation 或停止等待澄清，不得自行省略测试

#### Scenario: R2 操作断言失败
- **WHEN** R2 命名操作的 post-state、counts、保护或幂等断言任一失败
- **THEN** Agent 必须立即停止，不得继续后续层，也不得用旧验证证据替代失败结果

### Requirement: Task agent 必须先自审再通知主对话
系统 MUST 要求 task agent 在通知主对话验收前完成内部 self-review 或适用的 code review，复读测试结果并执行 diff、scope、secret 与需求覆盖检查；阻断问题 MUST 先自行整改并刷新验证。该自审不得替代用户的 Proposal 或 Apply 后人工 Review。

#### Scenario: 自审发现阻断问题
- **WHEN** task agent 的内部 review 发现规格遗漏、测试失败、越界 diff 或 secret 风险
- **THEN** task agent 必须先整改并重新验证，不能把已知阻断问题直接作为待验收结果交给主对话

#### Scenario: 自审只有非阻断风险
- **WHEN** 自审只发现无法在当前 scope 内消除的非阻断风险
- **THEN** Agent 必须在 Review package 中明确风险与未验证项，再通知主对话验收

### Requirement: 候选数据必须采用规则、抽样和异常聚焦审阅
系统 SHALL 要求规模化候选数据 Review package 提供生成规则与输入指纹、总体 counts、可复现抽样、异常/冲突清单、宽边界清单和 fail-closed 条件。正常项不得被机械要求全部逐条审阅；高风险、宽边界、身份/来源/映射冲突及用户明确指定的清单 MUST 逐项审阅。

#### Scenario: 大量正常候选无冲突
- **WHEN** 候选由固定规则生成且正常项通过总体断言
- **THEN** 用户可以通过生成规则、确定性抽样和 counts 审阅正常项，不需要机械逐条确认全部记录

#### Scenario: 候选存在宽边界或冲突
- **WHEN** 候选包含宽边界语义、身份冲突、来源冲突或映射冲突
- **THEN** 系统必须把这些项列入异常清单并逐项审阅，未决冲突必须阻断后续写入

#### Scenario: 业务契约要求 final manifest 逐项确认
- **WHEN** 已批准规格明确要求某个 final manifest 由用户逐项确认
- **THEN** 抽样策略不得取消该人工决策，只能用于其余规模化正常候选的证据组织

### Requirement: R2 条件式执行包必须逐层显式授权并声明 recovery evidence
系统 SHALL 允许用户在一次明确授权中预授权多个 R2 命名层，但执行包 MUST 逐层列出每个命名操作、环境、顺序、范围、排除范围、recovery evidence、预期 counts、before/after assertions 和停止条件。每层 recovery evidence MUST 明确选择可恢复备份，或经批准的 disposable recovery。disposable recovery 只适用于明确声明为 disposable、没有不可替代数据且具有确定性 recreate/reseed 路径的 local/test，并必须记录环境身份、声明、命令、预计耗时和验证断言；shared local、开发主数据、UAT 或任何不可替代数据 MUST 提供可恢复备份。当前 tidewise 本地 curated PostgreSQL MUST NOT 自动视为 disposable。普通 Apply 批准、旧批准或上一层批准 MUST NOT 被解释为该执行包授权。

#### Scenario: 用户一次授权多个 R2 命名层
- **WHEN** 执行包逐名列出 Layer A 和 Layer B 的全部授权字段、每层 recovery evidence 且用户明确批准整个包
- **THEN** Agent 可以严格执行 `Write Layer A -> Query/assert Layer A -> Write Layer B -> Query/assert Layer B`

#### Scenario: disposable local test 层使用重建证据
- **WHEN** 某 local/test 层被用户逐层批准为 disposable，且环境没有不可替代数据并提供确定性 recreate/reseed 命令、预计耗时和验证断言
- **THEN** Agent 可以以 approved disposable recovery 作为该层 recovery evidence，而不是物理备份

#### Scenario: shared 或不可替代数据层
- **WHEN** R2 层涉及 shared local、开发主数据、UAT 或任何不可替代数据
- **THEN** Agent 必须在该层执行前提供可恢复备份，不得使用 disposable recovery

#### Scenario: 上一层断言通过
- **WHEN** Layer A 的全部自动断言通过且 Layer B 已在同一执行包中被逐名明确授权
- **THEN** Agent 可以进入 Layer B，因为 Layer B 已被显式授权，而不是从 Layer A 的批准推定

#### Scenario: 任一断言、范围或 recovery evidence 失败
- **WHEN** 某层断言失败、实际范围漂移、recovery evidence 不成立、出现冲突或触发停止条件
- **THEN** Agent 必须立即停止，所有未执行层的剩余授权自动失效，重新执行必须取得新授权

#### Scenario: 执行包使用概括性后续范围
- **WHEN** 执行包只写“其余层”“后续数据”或其他未逐名范围
- **THEN** 这些未命名操作不在授权范围内，Agent 不得执行

### Requirement: R3 操作必须保持独立授权和恢复证据
系统 MUST 默认禁止 R3 跨层批量执行。生产、不可逆清理、Neo4j rebuild 和敏感部署 MUST 分别获得独立明确授权；被定义为 R3 的 cleanup MUST 单独成包。R3 MUST 提供备份/恢复或等价灾难恢复证据，且不得使用 disposable recovery 例外。

#### Scenario: PostgreSQL R2 层完成后准备 Neo4j rebuild
- **WHEN** PostgreSQL 各命名层已经通过 Query 验收
- **THEN** Agent 仍必须为 Neo4j rebuild 提交独立 R3 授权，不得从 PostgreSQL 执行包推定

#### Scenario: 生产操作与 UAT 操作相似
- **WHEN** 相同命令已在 UAT 作为 R2 获批并成功
- **THEN** 生产环境仍属于独立 R3 授权对象，UAT 授权不得扩展到生产

### Requirement: Commit 与 Review 必须采用阶段级 checkpoint
系统 SHALL 为内聚且可独立验证的阶段创建 commit/checkpoint，并禁止把每个微型 task 自动升级为 commit、push 或人工 Review。Proposal checkpoint、Apply 阶段 package、Apply final 与 Archive checkpoint仍必须服从现有 Git 和生命周期门禁。

#### Scenario: 同一阶段包含多个微型任务
- **WHEN** 多个 task 属于同一风险边界并共同形成可验证结果
- **THEN** Agent 必须在阶段完成后创建一个 scoped checkpoint，而不是为每个 checkbox 分别 commit 和 push

### Requirement: 新规则必须通过显式 adoption 应用于 active change
系统 MUST 让本规则 Deliver 后创建的新 change 默认使用新流程；active change MUST 保持历史 artifacts 和授权不变，直到其 branch 更新最新 `origin/main`、提交 scoped workflow-adoption tasks diff 并通过一次用户人工 Review。adoption 只能合并未来 gate，不能追认历史操作、取消已开始写操作的验收或扩大既有授权。

#### Scenario: Active change 尚未 adoption
- **WHEN** 本规则已经 Deliver 但某 active change 尚未提交并通过 adoption Review
- **THEN** 该 active change 必须继续按原 tasks 与授权边界执行，不得自动套用新规则

#### Scenario: Adoption 合并未来 R2 gates
- **WHEN** active change 的未开始 schema、seed 和 mapping 层被 scoped tasks diff 逐名组织为 R2 条件包并通过用户 Review
- **THEN** 这些未来层可以使用新条件式执行包，但已开始的写操作仍按原验收完成

#### Scenario: Adoption 试图扩大旧授权
- **WHEN** adoption diff 把既有批准扩展到新环境、新层、新范围或 Neo4j rebuild
- **THEN** 用户必须拒绝该 adoption 范围，系统不得把旧批准解释为新授权

### Requirement: 风险工作流规则必须保持分层单一事实来源
系统 MUST 让根 `AGENTS.md` 只保留简短总原则与路由，并把风险等级、Review package、条件式执行包、候选审阅、自审和 adoption 的详细唯一规则维护在 `.agents/openspec-workflow.md`；Git checkpoint、测试验证和 Skill 路由只在各自专责文件维护，不得复制完整流程。

#### Scenario: Agent 查找条件式执行包规则
- **WHEN** Agent 需要确定 R2 多层执行包的授权与停止语义
- **THEN** `.agents/openspec-workflow.md` 必须提供完整唯一规则，根 `AGENTS.md` 和其他 `.agents` 文件只保留硬门摘要或引用

#### Scenario: Apply 验证规则分层
- **WHEN** 本 change 修改规则正文
- **THEN** 自动化架构测试必须验证 R0—R3、Proposal/Apply Review、R2 逐层显式授权、R3 独立授权、阶段 checkpoint 和 active adoption 边界
