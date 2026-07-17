# Skill-Driven Development Workflow Specification

## Purpose

定义观潮家正式研发 change 如何使用 OpenSpec、项目原生 TDD、Git 隔离、风险 gate、验证和交付规则完成可审阅的工程生命周期。
## Requirements
### Requirement: 生产小程序视觉事实源路由

系统 SHALL 在 `.agents/frontend-boundaries.md` 中规定生产小程序页面的 visual/interaction source 路由：已批准 change 指定的 page-level canonical prototype 拥有页面视觉裁决权，旧 design skill 只保留历史和基础 token/component 参考。

#### Scenario: change 指定 canonical 页面

- **WHEN** design 指定 prototype 路径、版本指纹和视觉验收范围
- **THEN** Agent 必须以该页面最终渲染裁决页面效果，不能让旧 design skill 的冲突规则覆盖它

#### Scenario: 没有指定 canonical 页面

- **WHEN** 设计 change 没有 page-level canonical source
- **THEN** Agent 可以读取旧 design skill 作为参考，但不得宣称其为当前生产页面事实源

#### Scenario: 使用原型作为生产输入

- **WHEN** Agent 将 canonical prototype 转译为 Taro/React
- **THEN** prototype 保持只读，生产源码只提炼必要 tokens/primitives/compositions 和经授权资产，不复制 HTML、DOM、内联脚本或整套设计库

### Requirement: 正式研发必须由 OpenSpec 和项目原生规则驱动

系统 SHALL 在任务与已安装 Skill 的触发条件匹配时优先调用可用 Skill，但正式 change 必须通过 OpenSpec Explore/Propose/Apply/Sync/Archive 路由；TDD、系统化调试、完成前验证、Git 隔离和交付清理不得要求安装外部 plugin。

#### Scenario: 开始正式 change
- **WHEN** 用户提出需要设计和实现的正式工程变更
- **THEN** Agent 必须建立 OpenSpec change，读取命中的项目规则，并按生命周期进入下一阶段；没有 Superpowers plugin 也不得改变该入口

#### Scenario: Superpowers 未安装
- **WHEN** Superpowers plugin 不可用
- **THEN** OpenSpec、项目原生 TDD/debug/verification 规则、Codex Desktop worktree 机制和 GitHub/`gh` 交付路径仍可完成对应工程动作

#### Scenario: 纯解释性问答
- **WHEN** 用户只要求解释概念且不要求修改工程行为
- **THEN** Agent 不得机械创建 OpenSpec change 或无关长期 artifacts

### Requirement: OpenSpec 必须拥有唯一正式 artifacts

系统 MUST 将 proposal、design、delta specs、tasks、当前主规格和归档历史保存在 OpenSpec 目录中；不得创建平行的 Superpowers 设计、计划或执行事实来源，也不得要求 Superpowers 作为 OpenSpec 生命周期前提。

#### Scenario: 设计与计划完成
- **WHEN** 任意辅助讨论或计划方法形成设计与执行计划
- **THEN** 结果必须进入当前 change 的 `design.md` 或 `tasks.md`

#### Scenario: plugin 被卸载
- **WHEN** Superpowers plugin 被卸载
- **THEN** OpenSpec artifacts、当前主规格和项目规则仍是唯一可审阅事实来源，且不存在平行 plugin 文档来源要求

### Requirement: OpenSpec 生命周期必须映射到明确 Skills

系统 SHALL 依次遵循 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver`，并使用人工 Review 与 OpenSpec CLI 维护阶段边界。

#### Scenario: Proposal Review 未通过

- **WHEN** proposal、design、delta specs 或 tasks 尚未获得用户人工确认
- **THEN** Agent 不得进入 Apply，也不得用 Skill 默认流程替代人工 Review

#### Scenario: Apply-final Review 未通过

- **WHEN** Apply 已完成但 scoped diff 或验证证据尚未获得人工确认
- **THEN** Agent 不得进入 Sync、Archive、Deliver、PR 或 merge

### Requirement: Agent 规则必须采用分层单一事实来源

系统 MUST 将规则职责分层：`AGENTS.md` 只提供最高级硬门与路由；`openspec/config.yaml` 只提供稳定项目背景、语言和 artifact 写作约束；`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md` 分别提供 OpenSpec 生命周期/审批、Git 交付、测试/验证的唯一完整详述；主 workflow spec 只保留长期可验证行为。其他文件可以保留不可绕过摘要或链接，但不得复制另一职责来源的完整操作流程。

#### Scenario: 查找 OpenSpec 生命周期规则

- **WHEN** Agent 需要确定阶段顺序、人工 Review、风险 gate 或有状态操作授权
- **THEN** Agent 可以从 `AGENTS.md` 路由到 `.agents/openspec-workflow.md`，并以该文件作为唯一完整操作来源

#### Scenario: 查找 Git 交付规则

- **WHEN** Agent 需要创建 branch/worktree、提交、推送、合并或清理 change
- **THEN** Agent 可以从 `AGENTS.md` 路由到 `.agents/git-workflow.md`，并以该文件作为唯一完整操作来源

#### Scenario: 查找测试与验证规则

- **WHEN** Agent 需要选择 TDD、targeted tests、architecture checks 或完整验证范围
- **THEN** Agent 可以从任务路由进入 `.agents/testing-tdd.md`，并以该文件作为测试/验证唯一完整操作来源

#### Scenario: 查找项目上下文与 artifact 写作约束

- **WHEN** OpenSpec CLI 生成 artifact instructions
- **THEN** `openspec/config.yaml` 提供上下文与写作约束，不承载生命周期、Git 或测试操作详述

### Requirement: Git 隔离和收尾必须服从 change 边界

系统 MUST 为正式 change 使用独立 `codex/<change-name>` branch；Codex Desktop 可用时必须通过 Desktop 新任务创建受管 worktree 并从最新 `origin/main` 开始。Desktop 不可用时，仅在用户明确批准后按项目原生 fallback 规则使用 project-owned worktree。Deliver 必须保留 branch、PR、merge 和 cleanup 的完整顺序，不得依赖 Superpowers skill。

#### Scenario: 启动新 change
- **WHEN** Agent 准备创建正式 OpenSpec change
- **THEN** 必须先 `git fetch origin`，确认 worktree clean，并在 Desktop-managed worktree 中从 `origin/main` 创建或切换匹配 branch

#### Scenario: Desktop 不可用
- **WHEN** Desktop 受管机制不可用且用户明确批准 fallback
- **THEN** Agent 才可以按 `.agents/git-workflow.md` 的项目原生规则使用 project-owned worktree；没有 Superpowers plugin 不得放宽授权条件

#### Scenario: cleanup
- **WHEN** PR 已合并且默认分支已验证包含最终 archive commit
- **THEN** Agent 必须按 worktree 所有权执行远端 branch、Desktop 任务、托管 worktree 释放验证和本地 branch cleanup

### Requirement: 正式研发必须使用统一风险等级

系统 MUST 为每个 change 声明 R0—R3 基线风险，并按具体阶段或命名操作上调：R0 为文档/调研/只读审计，R1 为无状态写入的源码或测试变更，R2 为 migration/seed/local 或 UAT 数据变更，R3 为生产、不可逆 cleanup、Neo4j rebuild 或敏感部署。

#### Scenario: 实际操作风险上调

- **WHEN** 静态代码 change 进入 migration apply、数据写入、shared/UAT/prod 或 R3 操作
- **THEN** Agent 必须按实际最高风险执行对应人工 gate、recovery、before/after assertions 和停止语义

### Requirement: 规则精简必须保留关键工程硬门

系统 MUST 在分层规则与主 workflow spec 中继续明确约束 OpenSpec 唯一生命周期、Codex Desktop 受管任务/worktree 强制入口、change branch/worktree 隔离、人工 Review 与有状态操作审批、Sync/Archive/Deliver 顺序、按 worktree 所有权交付清理、数据事实源和通用安全边界；去重不得降低任何一项门禁。

#### Scenario: Proposal 未获人工确认

- **WHEN** change 的 proposal artifacts 尚未获得人工确认
- **THEN** Agent 不得进入 Apply，也不得以自动化或 Skill 默认流程替代确认

#### Scenario: 执行有状态操作

- **WHEN** Agent 准备写数据库、图谱关系或重建图投影
- **THEN** 必须逐层展示范围、顺序、recovery、断言和停止条件并取得明确授权

#### Scenario: Desktop-managed change 完成交付清理

- **WHEN** PR 已合并且默认分支已验证包含最终 archive commit
- **THEN** Agent 必须按 worktree 所有权执行远端 branch、Desktop 任务、托管 worktree 释放验证和本地 branch 的规定顺序

#### Scenario: 处理生产资料

- **WHEN** Agent 修改生产代码或生成投研与 AI 分析内容
- **THEN** 不得复制 prototype 代码、提交或打印 secret，也不得把内容表达为直接投资建议

### Requirement: 研发工作流术语与阶段证据必须统一

系统 MUST 统一 `gate`、`package`、`checkpoint` 和 `commit` 的定义：gate 是人工 Review/Authorization/Acceptance 边界，package 是同一风险边界内可连续完成的内聚交付单元，checkpoint 是可审阅证据点，commit 是 scoped Git 状态快照。普通 task、测试、修复、diff、commit 和 push 不得单独制造人工 gate；Proposal 与 tasks 的 Gate Map、Complexity Budget 和 package 映射 MUST 一致。

#### Scenario: R1 快速模式

- **WHEN** change 只涉及 local 无状态 coding、规则或测试
- **THEN** Proposal Review 与 Apply-final Review 是唯一人工 checkpoint，中间 package 子项连续执行并汇总证据

#### Scenario: 普通任务在 package 内完成

- **WHEN** Agent 完成不跨越授权边界的实现、测试、修复、diff 或 commit
- **THEN** 证据归入所属 package，不因 checkbox 或单次操作新增人工 gate

#### Scenario: Gate Map 与 tasks 不一致

- **WHEN** Proposal 与 tasks 的 package、gate、risk、human、reason code 或 allowed scope 任一映射不一致
- **THEN** task-design lint 或 Proposal 自检必须失败并阻止进入下一阶段

#### Scenario: 阶段 checkpoint 交付审阅

- **WHEN** package 达到其验证条件并形成 scoped commit
- **THEN** checkpoint 必须记录 scope、风险、证据、未验证项、阻断项和下一步授权边界

#### Scenario: package 内失败修复

- **WHEN** targeted test、validate 或 lint 失败且修复不扩大 scope 或风险
- **THEN** 在同一 package 内修复并刷新证据，不新增人工 gate

### Requirement: 长期工作流行为必须可验证

系统 MUST 通过 OpenSpec strict、规则/architecture targeted checks、精确 task-design lint、scope/secret/link 检查和语义覆盖矩阵验证规则分层与硬门完整性。主 workflow spec 只保留稳定 requirement/scenario，不固化特定 change 的过程约束与历史验收内容。

#### Scenario: Proposal checkpoint 验证范围匹配

- **WHEN** change 只涉及 OpenSpec artifacts、workflow 文本、agent rules 或 workflow architecture assertions
- **THEN** Proposal 阶段运行范围匹配检查，不机械运行 `go test ./...`，并记录未授权的 Apply/Sync/Archive/Deliver 操作

#### Scenario: 硬门覆盖矩阵完整

- **WHEN** Agent 准备请求 Proposal Review 或 Apply 后 Review
- **THEN** 覆盖矩阵必须映射 OpenSpec 顺序、人工 Review、Desktop 入口、sequential/parallel 分流、两类 cleanup、风险/有状态写、TDD/CI/验证、事实源和安全边界，缺项即 fail-closed

#### Scenario: 主 spec 保持长期行为

- **WHEN** change 已归档且规则进入后续研发
- **THEN** 主 workflow spec 仍能用稳定 requirement/scenario 和自动化检查验证行为，不依赖本 change 的历史叙述或一次性数字

### Requirement: 阶段 Review package 必须聚合一致风险边界内的证据

系统 SHALL 允许 contract、实现、测试、dry-run、只读 preflight、diff、验证、commit 和 push 在同一风险边界内组成 package；package 必须说明 scope、non-goals、风险、证据、未验证项、阻断项、停止条件和下一步授权边界。

#### Scenario: R2/R3 package

- **WHEN** package 包含 shared/UAT/prod、R2 或 R3 有状态操作
- **THEN** 普通 Apply gate 不得隐含写入授权，必须有对应 recovery evidence、明确范围和独立授权

### Requirement: 验证深度必须随风险与生命周期递增

系统 MUST 按真实受影响交付边界选择验证：workflow 文本、agent rules、OpenSpec artifacts 和 architecture test/lint 变更运行 OpenSpec strict、精确 task-design lint、workflow targeted checks、diff/scope/secret/link 检查；局部 coding 运行 targeted tests 与受影响 suite；只有共享运行时代码、跨模块契约、公共基础设施或无法判定边界时运行 repo-wide full validation。

#### Scenario: workflow rule change

- **WHEN** change 只修改 workflow 文本、agent rules、OpenSpec artifacts 或 workflow architecture assertions
- **THEN** Apply-final 运行范围匹配的 workflow/architecture checks 和共享 contract checks，不机械运行 `go test ./...`

#### Scenario: 边界无法判定

- **WHEN** 受影响交付边界或完整 suite 无法可靠确定
- **THEN** Agent 必须 fail-closed，扩大验证或停止等待澄清

### Requirement: 工作流架构测试必须验证当前规则语义

系统 SHALL 通过自动化检查验证生命周期顺序、人工 Review、Desktop-managed 默认入口、approved fallback 双条件、sequential/parallel 分流、两类 cleanup、package/gate schema、验证范围和外部 plugin 依赖解除，不得依赖已被当前规则替代的历史短语。

#### Scenario: 规则契约被削弱
- **WHEN** 规则删除 Desktop 强制受管、fallback 双条件、cleanup 顺序、Gate Map、验证边界或原生 TDD/debug/verification 约束
- **THEN** 对应 architecture contract 或 task-design lint 必须失败

#### Scenario: 外部 plugin 强制引用残留
- **WHEN** active 项目规则、主 workflow spec 或 workflow architecture assertions 仍包含外部 plugin 强制路由、平行 plugin 文档来源或必须安装可选 plugin 的约束
- **THEN** workflow architecture assertion 必须失败并阻止 Apply-final Review

#### Scenario: 正向硬门保持
- **WHEN** active 项目规则移除外部 plugin 强制路由
- **THEN** 检查仍必须确认 OpenSpec 顺序、两个人工 Review、TDD、debug diagnosis、fresh verification、Desktop worktree、branch/cleanup、风险/事实源/安全边界均存在

### Requirement: 运行时资源必须独立于 active OpenSpec 路径

运行命令消费的数据 MUST 位于稳定 backend `data/` 或 `resource/` 路径；OpenSpec change 目录只能保存 review snapshot、hash 和 evidence。路径、hash 或 schema 校验失败时必须 fail-closed。

#### Scenario: Archive 后运行资源仍可消费

- **WHEN** change 已归档
- **THEN** 运行命令仍从稳定资源路径读取已验证数据，不依赖 active OpenSpec path
