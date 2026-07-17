## MODIFIED Requirements

### Requirement: 正式研发必须由 OpenSpec 和项目原生规则驱动

系统 SHALL 在任务与已安装 Skill 的触发条件匹配时优先调用可用 Skill，但正式 change 必须通过 OpenSpec Explore/Propose/Apply/Sync/Archive 路由；TDD、系统化调试、完成前验证、Git 隔离和交付清理不得要求安装 Superpowers。

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
- **THEN** OpenSpec artifacts、当前主规格和项目规则仍是唯一可审阅事实来源，且不存在 `docs/superpowers` 平行来源要求

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

### Requirement: 工作流架构测试必须验证当前规则语义

系统 SHALL 通过自动化检查验证生命周期顺序、人工 Review、Desktop-managed 默认入口、approved fallback 双条件、sequential/parallel 分流、两类 cleanup、package/gate schema、验证范围和 Superpowers 依赖解除，不得依赖已被当前规则替代的历史短语。

#### Scenario: 规则契约被削弱
- **WHEN** 规则删除 Desktop 强制受管、fallback 双条件、cleanup 顺序、Gate Map、验证边界或原生 TDD/debug/verification 约束
- **THEN** 对应 architecture contract 或 task-design lint 必须失败

#### Scenario: Superpowers 强制引用残留
- **WHEN** active 项目规则、主 workflow spec 或 workflow architecture assertions 仍包含 `superpowers:*`、`docs/superpowers` 或必须安装 Superpowers 的约束
- **THEN** workflow architecture assertion 必须失败并阻止 Apply-final Review

#### Scenario: 正向硬门保持
- **WHEN** active 项目规则移除 Superpowers 强制路由
- **THEN** 检查仍必须确认 OpenSpec 顺序、两个人工 Review、TDD、debug diagnosis、fresh verification、Desktop worktree、branch/cleanup、风险/事实源/安全边界均存在
