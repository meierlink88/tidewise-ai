## ADDED Requirements

### Requirement: 正式研发必须由已安装 Skill 驱动
系统 SHALL 在任务与已安装 Skill 的触发条件匹配时优先调用该 Skill，不得使用临时自定义流程替代已有能力。

#### Scenario: 开始正式功能 change
- **WHEN** 用户提出需要设计和实现的正式工程变更
- **THEN** Agent 必须通过 OpenSpec Explore/Propose skills 建立 change，并按项目 Skill 路由进入后续阶段

#### Scenario: 纯解释性问答
- **WHEN** 用户只要求解释概念且不要求修改工程行为
- **THEN** Agent 不得机械创建 OpenSpec change 或无关长期 artifacts

### Requirement: OpenSpec 必须拥有唯一正式 artifacts
系统 MUST 将 proposal、design、requirements、tasks、当前主规格和归档历史保存在 OpenSpec 目录中，不得默认创建平行的 Superpowers 设计或计划事实来源。

#### Scenario: Brainstorming 完成设计
- **WHEN** `superpowers:brainstorming` 已获得用户设计确认
- **THEN** 设计结果必须进入 OpenSpec `design.md`，不得默认创建 `docs/superpowers/specs/`

#### Scenario: 复杂任务完成计划拆分
- **WHEN** 使用 `superpowers:writing-plans` 的任务拆分方法
- **THEN** 可执行计划必须进入 OpenSpec `tasks.md`，除非用户明确批准，否则不得创建 `docs/superpowers/plans/`

### Requirement: OpenSpec 生命周期必须映射到明确 Skills
系统 SHALL 使用 repo-local OpenSpec skills 驱动 Explore、Propose、Apply、Sync 和 Archive，并使用人工 Review 与 OpenSpec CLI 完成 Review 和 Validate。

#### Scenario: 执行 change 全生命周期
- **WHEN** 正式 change 从需求进入实现和交付
- **THEN** Agent 必须依次使用或遵循 `openspec-explore`、`openspec-propose`、人工 Review、`openspec-apply-change`、Validate、`openspec-sync-specs` 和 `openspec-archive-change`

### Requirement: 工程纪律必须使用对应 Superpowers Skills
系统 MUST 在功能实现、缺陷、完成验证和代码审查场景中调用对应 Superpowers skills，并将结果更新到当前 OpenSpec change。

#### Scenario: 实现功能或修复缺陷
- **WHEN** Apply 阶段涉及功能、bugfix、重构或可观察行为变更
- **THEN** Agent 必须使用 `test-driven-development`，遇到异常时必须先使用 `systematic-debugging`

#### Scenario: 声明完成或执行 Git 交付
- **WHEN** Agent 准备声明完成、commit、push、创建 PR、sync 或 archive
- **THEN** Agent 必须使用 `verification-before-completion` 并读取新鲜验证结果

#### Scenario: 收到代码审查意见
- **WHEN** 用户或 GitHub 提供 review feedback
- **THEN** Agent 必须使用 `receiving-code-review` 验证意见后再修改，不得盲目接受

### Requirement: Git 隔离和收尾必须服从 change 边界
系统 MUST 为正式 OpenSpec change 使用独立 `codex/<change-name>` branch，并只在并行隔离需要时增加 worktree。

#### Scenario: 从最新主分支开始 change
- **WHEN** Agent 准备创建新的正式 OpenSpec change
- **THEN** Agent 必须先更新远端引用，并基于最新 `origin/main` 创建独立 branch 或 worktree，不得使用未经确认的本地 `main` 作为基线

#### Scenario: 串行执行单一 change
- **WHEN** 当前没有其他未完成 change 需要保留工作目录
- **THEN** Agent 必须使用独立 branch，并且不得仅为普通任务额外创建 worktree

#### Scenario: 并行执行 change
- **WHEN** 当前 change 尚未结束且用户批准启动另一个 change
- **THEN** Agent 必须使用 `using-git-worktrees` 创建隔离 worktree，并为新 change 创建独立 branch；在 Codex Desktop 可用时优先创建与独立任务绑定的原生 worktree

#### Scenario: 准备 PR 或 merge
- **WHEN** 实现与测试已经完成
- **THEN** Agent 只有在 OpenSpec sync、archive 和 `validate --all` 完成后才能使用 `finishing-a-development-branch` 进入 PR/merge

### Requirement: 并行 Agent 必须限制共享写状态
系统 SHALL 只对两个以上无共享写状态、无顺序依赖的任务使用并行 Agent，并由主 Agent 统一维护正式 artifacts 和数据库写入。

#### Scenario: 并行只读分析
- **WHEN** 多个来源核验、测试分析或日志排查可以独立完成
- **THEN** Agent 可以使用 `dispatching-parallel-agents`，等待结果后由主 Agent 汇总

#### Scenario: 多 Agent 可能修改同一文件
- **WHEN** 候选子任务会修改同一个 OpenSpec artifact、tasks 文件或数据库状态
- **THEN** Agent 不得并行执行这些写任务

### Requirement: OpenSpec 配置必须兼容当前 schema
系统 MUST 只在 `openspec/config.yaml` 的 `rules` 中使用当前 spec-driven schema 支持的 artifact IDs，并保持工程上下文与已生效架构一致。

#### Scenario: 获取 artifact instructions
- **WHEN** Agent 运行 `openspec instructions <artifact> --change <name> --json`
- **THEN** 命令不得输出未知 artifact rule key 警告，且上下文不得声明与主规格相冲突的过期架构
