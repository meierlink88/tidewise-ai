## 1. TDD RED：锁定当前失败与新语义契约

- [x] 1.1 在 `backend/` 运行 `go test ./internal/architecture -run TestSkillDrivenWorkflowRules -count=1`，记录现有 CI 因缺少 `superpowers:using-git-worktrees` 映射而失败的 RED 证据
- [x] 1.2 先更新 `backend/internal/architecture/workflow_rules_test.go`，用条件语义断言替换“tasks 全部完成后”“Sync delta specs”“Archive change”“原生 worktree”等旧字面断言
- [x] 1.3 新测试必须覆盖 OpenSpec 阶段顺序与人工 Review、Desktop-managed 默认入口、approved fallback 双条件、sequential/parallel 分流、两类 cleanup 顺序，以及 Archive 后才使用 `superpowers:finishing-a-development-branch`
- [x] 1.4 运行 `go test ./internal/architecture -run TestSkillDrivenWorkflowRules -count=1`，确认新语义测试仍因 `.agents/skill-routing.md` 缺少 approved fallback Skill 映射而 RED

## 2. 最小规则修复与 GREEN

- [x] 2.1 在 `.agents/skill-routing.md` 增加条件映射：Desktop-managed 使用 Desktop 新任务机制；只有 Desktop 机制不可用且用户明确批准 fallback 时才使用 `superpowers:using-git-worktrees`
- [x] 2.2 保持 `.agents/git-workflow.md` 为完整 Git/worktree 流程的唯一事实来源，不复制流程、不削弱 Desktop 强制受管与 fallback 双条件
- [x] 2.3 运行 `go test ./internal/architecture -run TestSkillDrivenWorkflowRules -count=1`，确认目标测试 GREEN
- [x] 2.4 运行 `go test ./internal/architecture -count=1`，确认 architecture package GREEN

## 3. 全量验证与 Apply 后人工 Review

- [x] 3.1 在 `backend/` 运行 `go test ./... -count=1` 并读取完整结果
- [x] 3.2 在仓库根目录运行 `openspec validate fix-agent-worktree-routing-ci` 与 `openspec validate --all`，确认 change 和全局 artifacts 均有效
- [x] 3.3 运行 `git diff --check`、检查 scoped diff 与 `git status --short`，确认仅包含本 change artifacts、`.agents/skill-routing.md` 和 `backend/internal/architecture/workflow_rules_test.go`，未触碰归档历史或其他 active change
- [x] 3.4 提交 Apply scoped diff 与 RED/GREEN/全量验证证据，停止并等待第二次人工 Review；未获批准不得 Sync、Archive 或 Deliver

## 4. Review 后生命周期

- [x] 4.1 仅在 Apply 后人工 Review 明确批准后，使用 `openspec-sync-specs` 将本 delta 增量同步到现有 `skill-driven-development-workflow` 主规格，不复制 capability
- [x] 4.2 Sync 后使用 `openspec-archive-change` 归档本 change，运行 `openspec validate --all` 并创建 `spec: archive fix-agent-worktree-routing-ci` checkpoint
- [ ] 4.3 Archive 完成后才使用 `superpowers:finishing-a-development-branch` 进入 Deliver，并按 Desktop-managed cleanup 顺序完成远端 branch、Desktop 任务/worktree 与本地 branch 清理
