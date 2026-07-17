## ADDED Requirements

### Requirement: 委派式工程变更生命周期
每个正式 change SHALL 按 `Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup` 顺序推进，且 SHALL 不跳过、倒置或由自动化结果替代任何人工门禁。`Merge` 和 `Cleanup` SHALL 作为 PR 交付后的 Leader operational state 跟踪，不得阻塞已归档 change 的 OpenSpec checkbox 完成或要求回写已归档 tasks。

#### Scenario: 正常完成一个正式 change
- **WHEN** 一个工程请求需要修改仓库文件
- **THEN** 工作流依次完成探索、独立任务委派、提案、Leader 提案评审、实施、验证、Leader 验收、规格同步、归档、PR 交付、用户合并和合并后清理

#### Scenario: 前置阶段尚未完成
- **WHEN** 当前阶段所依赖的前置阶段或人工批准尚未完成
- **THEN** Agent 停留在当前门禁并报告所需条件，不继续执行后续阶段

#### Scenario: 归档后的合并与清理状态
- **WHEN** 执行 Agent已完成 Deliver 并向 Leader交付 cleanup handoff，但用户尚未 merge Pull Request
- **THEN** 本 change 的 OpenSpec tasks 保持可完成，Leader 在主 Codex 任务中跟踪用户 Merge 与后续 Cleanup，不回写已归档 tasks

### Requirement: Leader 与执行 Agent 职责分离
开发 Leader SHALL 仅负责 Explore、创建和委派新 Codex 任务、评审、批准、监控、验收及合并后清理，不得执行具体代码、测试、配置、文档或 change 产物实施；执行 Agent SHALL 独立完成 Eino audit、OpenSpec、TDD、实现、验证、Sync、Archive、提交、推送和 Pull Request。

#### Scenario: Leader 启动新 change
- **WHEN** Leader 在主 Codex 任务中确认需要正式仓库变更
- **THEN** Leader 创建并委派独立执行任务，而不在主任务 worktree 中编写具体实现或 change 产物

#### Scenario: 执行 Agent 推进获批 change
- **WHEN** 执行 Agent收到明确的 Proposal Review 或 Apply-final Review 批准
- **THEN** 执行 Agent在自己的任务和 worktree 中继续完成后续实施或交付阶段

### Requirement: 合并后的安全资源清理
Leader SHALL 仅在确认 Pull Request merged 后从主项目清理 change 资源，并 SHALL 依次验证 change worktree clean、change 提交已进入 `origin/main`，再删除 worktree、本地 change 分支、远端 change 分支（若存在），最后执行 worktree prune；任何失败 SHALL 被报告且不得静默视为完成。

#### Scenario: 安全清理已合并 change
- **WHEN** PR 已确认 merged、change worktree clean 且必要提交可从 `origin/main` 到达
- **THEN** Leader 按 worktree、本地分支、可选远端分支、prune 的顺序清理，并报告最终资源状态

#### Scenario: worktree 含有未提交内容
- **WHEN** 合并后检查发现 change worktree 存在 tracked 或 untracked 变更
- **THEN** Leader 停止删除操作并报告工作区状态，不删除 worktree 或分支

#### Scenario: change 提交尚未进入主线
- **WHEN** 无法证明 change 的必要提交已经进入 `origin/main`
- **THEN** Leader 停止清理并报告验证失败，不删除 worktree 或分支

#### Scenario: 清理步骤失败
- **WHEN** 删除 worktree、删除本地或远端分支、或 prune 任一步骤失败
- **THEN** Leader 报告失败命令、错误、已完成步骤和剩余资源，不宣称 Cleanup 完成

## MODIFIED Requirements

### Requirement: 独立的 change 工作空间
每个新 OpenSpec change SHALL 由开发 Leader 使用 Codex `create_thread` 创建独立执行任务和 Desktop-managed worktree；执行任务 SHALL 在写入 change 产物或实现文件前，使用基于最新 `origin/main` 的独立 `codex/<change-name>` feature branch。

#### Scenario: 开始新的 change
- **WHEN** 当前不存在与请求范围匹配的活跃 change、执行任务和 worktree
- **THEN** Leader 使用 `create_thread` 创建并委派独立 Codex 任务及 Desktop-managed worktree，执行 Agent获取 `origin/main`、创建命名分支，并在写入文件前验证任务、worktree 和分支

#### Scenario: Desktop worktree 初始为 detached HEAD
- **WHEN** 新执行任务的 Desktop-managed worktree 没有当前分支
- **THEN** 执行 Agent先确认同名分支不存在或安全匹配，再从最新 `origin/main` 创建并切换 `codex/<change-name>`，不得覆盖已有工作

#### Scenario: 恢复已有 change
- **WHEN** 已有活跃 change 具备匹配的执行任务、分支和 worktree
- **THEN** Leader和执行 Agent继续使用已有隔离资源，不创建重复 change 或覆盖已有修改

### Requirement: 经过评审的提案包
proposal、必要的 design、capability specs 和具备 TDD 结构的 tasks SHALL 在实施开始前全部完成并通过 strict validation，且 SHALL 由开发 Leader 明确完成 Proposal Review；执行 Agent MUST NOT 自行批准该门禁。

#### Scenario: 提案产物准备完成
- **WHEN** `openspec status` 显示所有 apply 前置产物均已完成且 strict validation 成功
- **THEN** 执行 Agent向 Leader 汇总范围、需求、风险、任务、分支和 worktree 状态，并等待 Leader 明确批准后再实施

#### Scenario: 执行 Agent 未获得 Leader 批准
- **WHEN** Leader尚未明确表达 Proposal Review 通过
- **THEN** 执行 Agent不得进入 Apply，且不得以自身判断或验证通过替代批准

### Requirement: 完成前评审与验证
已实现的 change SHALL 通过工程测试、策略检查和严格 OpenSpec 验证，并 SHALL 在同步规格和归档前由开发 Leader明确完成 Apply-final Review 和 Leader Acceptance；执行 Agent MUST NOT 自行批准该门禁。

#### Scenario: 实现任务看似全部完成
- **WHEN** 所有实现任务均被标记为完成
- **THEN** 执行 Agent向 Leader报告完整差异、RED/GREEN/REFACTOR 和验证证据，并等待 Leader明确验收后再执行 Sync 和 Archive

#### Scenario: Leader 要求调整实现
- **WHEN** Leader在 Apply-final Review 中发现范围、实现或证据不满足提案
- **THEN** 执行 Agent继续在同一任务和 worktree 中修正、重新验证并再次提交验收材料

### Requirement: 通过 Pull Request 交付
每个完成且获得 Leader Acceptance 的 change SHALL 由执行 Agent有选择地提交文件、推送至 `codex/<change-name>` 分支，并通过 GitHub Pull Request 交付；Pull Request merge SHALL 始终由用户控制，且 merged 状态 SHALL 成为 Cleanup 的前置条件。执行 Agent在 PR 交付时 SHALL 向 Leader提供 cleanup handoff；该 handoff SHALL 使该 change 的 OpenSpec tasks 在不等待 merge 的情况下完成。

#### Scenario: change 获准交付
- **WHEN** Leader Acceptance、规格同步、归档和最终验证均成功
- **THEN** 执行 Agent推送分支、创建或更新 Pull Request，并向 Leader和用户报告地址、验证证据及已知风险

#### Scenario: Pull Request 尚未合并
- **WHEN** Pull Request 仍为 open、draft、closed-unmerged 或 merged 状态无法确认
- **THEN** Leader不得启动 change worktree 或分支清理，用户仍决定是否以及何时 merge

#### Scenario: 执行 Agent交付 cleanup handoff
- **WHEN** 执行 Agent已创建 Pull Request
- **THEN** 执行 Agent向 Leader提供 PR URL/number、change branch、worktree path、必要 commits，以及 merged 确认、worktree clean、`origin/main` 可达性、删除 worktree/分支和 prune 的清理顺序，使归档前的 OpenSpec tasks 不依赖用户后续 merge
