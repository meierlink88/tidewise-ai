## ADDED Requirements

### Requirement: Worktree Skill 路由必须保留 Desktop 与 fallback 条件语义
系统 MUST 将 Codex Desktop 新任务机制作为 Desktop 可用时创建受管 worktree 的唯一默认入口，并且仅在 Desktop 受管机制不可用且用户明确批准 fallback 时，才将 `superpowers:using-git-worktrees` 路由到 project-owned fallback worktree。

#### Scenario: Desktop 可用时启动 change
- **WHEN** Agent 在 Codex Desktop 可用的环境中启动正式 OpenSpec change
- **THEN** Skill 路由必须要求通过 Desktop 新任务机制创建受管 worktree，不得调用 `superpowers:using-git-worktrees` 将手工或 project-owned worktree 作为等价默认路径

#### Scenario: 使用 approved fallback
- **WHEN** Codex Desktop 受管机制不可用且用户已经明确批准 fallback
- **THEN** Skill 路由必须允许调用 `superpowers:using-git-worktrees`，并继续服从 `.agents/git-workflow.md` 对路径、所有权、branch 和 scoped changes 的约束

#### Scenario: fallback 条件不完整
- **WHEN** Desktop 受管机制仍可用或用户尚未明确批准 fallback
- **THEN** Agent 不得调用 `superpowers:using-git-worktrees` 创建 project-owned worktree

### Requirement: 工作流架构测试必须验证当前规则语义
系统 SHALL 通过自动化架构测试验证 OpenSpec 生命周期顺序与人工 Review、Desktop-managed 默认入口、approved fallback 双条件、sequential/parallel 分流、两类 cleanup 顺序以及 Archive 后进入 finishing branch，不得依赖已被当前规则替代的旧叙述性短语。

#### Scenario: 规则语义完整
- **WHEN** 开发者运行 `go test ./internal/architecture -count=1`
- **THEN** 测试必须验证当前工作流契约并通过，且不得要求恢复默认手工 worktree 路径

#### Scenario: fallback Skill 映射被删除
- **WHEN** `.agents/skill-routing.md` 不再包含 approved fallback 对 `superpowers:using-git-worktrees` 的条件映射
- **THEN** 架构测试必须失败并指出缺失的 worktree Skill 路由语义

#### Scenario: Desktop 强制受管或 fallback 双条件被削弱
- **WHEN** 规则允许在 Desktop 可用时绕过 Desktop 新任务，或在未同时满足 Desktop 不可用与用户明确批准时创建 project-owned fallback worktree
- **THEN** 架构测试必须失败
