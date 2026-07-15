## ADDED Requirements

### Requirement: 运行时资源必须独立于 active OpenSpec 路径

运行命令消费的数据 MUST 位于不会随 OpenSpec archive 移动的稳定 backend `data/` 或 `resource/` 路径；OpenSpec change 目录只能保存 review snapshot、hash 和 evidence，不得作为运行时数据的唯一来源。迁移必须保持单一运行事实源，并在路径、hash 或 schema 验证失败时 fail-closed。

#### Scenario: Archive 后运行资源仍可消费

- **WHEN** change 已归档且运行命令按稳定资源路径启动
- **THEN** 命令必须读取同一份已验证资源，不依赖已移动的 active OpenSpec path

#### Scenario: 稳定资源校验失败

- **WHEN** 稳定资源不存在、hash/schema 与 review evidence 不一致或消费者仍指向 active path
- **THEN** 运行或交付流程必须停止并报告阻断，不得静默读取未验证副本

## MODIFIED Requirements

### Requirement: 普通任务不得自动成为人工门禁

系统 SHALL 将普通 task checkbox 作为 package 内可验证工作单元，而不是自动人工门禁。人工 gate MUST 在 Gate Map 标注风险等级、合法风险原因、所需证据、通过后允许的下一步和明确不授权的操作；一级 task MUST 表达内聚交付 package。普通 coding、测试、修复、commit、push 和无状态命令错误 MUST 在同一风险边界内连续处理，不得仅因操作发生而新增人工 gate；local 快速模式不得外推到 shared/UAT/prod 或其他严格门禁范围。

#### Scenario: 完成微型实现任务

- **WHEN** Agent 完成一个不跨越授权边界的微型 task
- **THEN** Agent 必须将其证据汇入所属 package，不得仅因 checkbox 完成而要求一次独立人工 Review

#### Scenario: 设置人工 gate

- **WHEN** 某一步需要用户人工 Review、Authorization 或 Acceptance
- **THEN** Gate Map 必须说明该 gate 所控制的合法风险原因与授权边界，不能只写通用“等待确认”

#### Scenario: 普通操作被单独包装为 gate

- **WHEN** tasks 将测试、修复、dry-run、validate、diff/secret check、commit、push 或无状态命令错误单独标为人工 gate
- **THEN** task-design lint 必须失败，Agent 必须把该操作合并回所属 package

#### Scenario: 快速模式环境越界

- **WHEN** Agent 试图把 local 快速模式用于真实 shared、UAT/prod、secret/权限、回滚或 migration/seed/data repair
- **THEN** 系统必须保持严格 gate，不能仅凭普通 coding 路径授权继续

### Requirement: 阶段 Review package 必须聚合一致风险边界内的证据

系统 SHALL 允许 contract、实现、测试、修复、dry-run、只读 preflight、validate、diff/secret check、commit/push 和异常清单组成同一风险边界的 package。package MUST 说明 scope、non-goals、风险等级、证据、未验证项、阻断项、停止条件和下一步授权边界，且不得绕过 Proposal 后或 Apply 后人工 Review。普通 local coding 通常 MUST 采用 Proposal Review -> 连续 Apply package -> Apply-final Review；stateful local package 在获准范围内 MUST 采用一次 preflight -> 单次 Write -> 一次 verify。

#### Scenario: R1 阶段形成 package

- **WHEN** contract、实现、targeted tests、修复和 Git 操作属于同一无状态风险边界
- **THEN** Agent 必须用一个 package 提交验收，而不是为每个微型 task 分别 commit、push 或 Review

#### Scenario: package 内测试失败并修复

- **WHEN** package 内 targeted test、validate 或 lint 失败且修复不扩大风险与 scope
- **THEN** Agent 必须在同一 package 内修复并刷新证据，不得为失败与修复新增人工 gate

#### Scenario: local stateful package

- **WHEN** local package 已获得精确范围授权并包含状态写入
- **THEN** Agent 必须执行一次完整 preflight、一次 Write 和一次 post-write verify，失败、漂移或断言不通过时立即停止

#### Scenario: package 涉及严格环境或高风险操作

- **WHEN** package 包含真实 shared、UAT/prod、secret/权限、回滚、migration/seed/data repair 或 R3 操作
- **THEN** 快速模式不得替代对应的严格人工 gate、recovery evidence 和独立授权

### Requirement: 验证深度必须随风险与生命周期递增

系统 MUST 以 package 为单位组织验证：开发中运行与当前工作匹配的 targeted tests；连续 Apply package 在其风险边界内运行一次范围匹配验证；Apply final 运行一次受影响 app/module/package 的完整 suite 与共享 architecture/contract tests。共享规则、跨模块契约、公共基础设施或 repo-wide 变更 MUST 运行 repo-wide full validation。验证选择 MUST 记录受影响交付边界、共享 tests 与 repo-wide 判定理由；边界、理由或 suite 不清楚时 MUST fail-closed。输入 commit、manifest、schema、baseline 和 environment 指纹未变化且证据未失败时，Archive/PR 可以复用对应验证证据；变化或失败时 MUST 只重验受影响层，并在不确定时扩大验证。R2/R3 MUST 额外提供执行前后状态断言。

#### Scenario: 开发中运行 targeted tests

- **WHEN** Agent 在 package 内实现、调试或修复
- **THEN** Agent 可以按需重复 targeted tests，但不得把每次测试运行升级为 checkpoint 或人工 Review

#### Scenario: Apply final 验证

- **WHEN** change 完成 Apply 并准备请求 Apply 后人工 Review
- **THEN** Agent 必须运行一次受影响交付边界完整 suite、共享 architecture/contract tests、OpenSpec strict validation、diff/scope/secret 检查，并汇总所有 R2/R3 pre/post evidence

#### Scenario: 共享 workflow change

- **WHEN** change 修改共享规则、architecture tests 或其他 repo-wide 行为
- **THEN** Apply final 必须运行 `go test ./...` 和相关 OpenSpec/规则检查

#### Scenario: 指纹未变化

- **WHEN** commit、manifest、schema、baseline 和 environment 指纹均与已通过证据一致
- **THEN** Archive/PR 可以复用对应证据，不得无理由重复全量重验

#### Scenario: 指纹变化或证据失败

- **WHEN** 任一输入指纹变化、验证证据失败、范围漂移或环境身份变化
- **THEN** Agent 必须停止复用并重验受影响层；无法明确边界时必须扩大到 repo-wide full validation 或停止等待澄清

### Requirement: Commit 与 Review 必须采用两个停顿型 checkpoint

系统 SHALL 将 Proposal Review 与 Apply-final Review 作为仅有需要停顿并重新验收的 checkpoint，并禁止把每个微型 task 自动升级为 commit、push 或人工 Review。Proposal 批准后的内聚 Apply package 及其 commit MUST 连续执行并作为验证证据，不构成额外人工 checkpoint；Archive/Deliver MUST 仅作为生命周期记录，在输入 commit、manifest、schema、baseline 和 environment 指纹未变化且证据未失败时不得新增人工停顿或重复全量验证。连续执行证据 MUST 记录阶段、commit、已通过验证、输入状态指纹、下一步和真实 blocker。恢复时 MUST 从记录的下一步继续；Git、权限、输出格式等无状态错误不得触发数据库或其他状态层全量重验。两行 package 均为 `Human=yes` 时，`continuous_automation_scope=packages:none` MUST 仅表示没有 `Human=no` package，不得被解释为否定 Proposal 批准后 Package 1 子项的连续执行或跳过人工起始 gate。

#### Scenario: 同一阶段包含多个微型任务

- **WHEN** 多个 task 属于同一风险边界并共同形成可验证结果
- **THEN** Agent 必须在 Proposal 批准后连续完成该 package 并创建 scoped commit 作为证据，而不是为每个 checkbox 分别 commit、push 或新增人工 checkpoint

#### Scenario: 从断点恢复

- **WHEN** checkpoint 已记录下一步、已通过验证和真实 blocker，且无状态输入未变化
- **THEN** Agent 必须从下一步继续，并可以复用对应证据

#### Scenario: 无状态错误恢复

- **WHEN** 仅发生 Git、权限、输出格式或其他无状态命令错误
- **THEN** Agent 不得因此触发数据库或其他状态层的全量重验
