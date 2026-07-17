## 1. Proposal Review 门禁

- [x] 1.1 执行 Agent向开发 Leader提交 proposal、design、delta spec、tasks、Eino reference audit、风险、worktree/branch 状态和 `openspec validate establish-leader-delegation-workflow --strict` 结果。
- [x] 1.2 等待开发 Leader明确批准 Proposal Review；该 checkbox 只能在收到 Leader批准后由执行 Agent记录，执行 Agent不得自行批准，未批准前不得执行第 2 组及后续任务。

## 2. 工作流策略测试（RED）

- [x] 2.1 先扩展 `internal/architecture/workflow_policy_test.go`，要求策略文件包含完整委派生命周期、Leader/执行 Agent职责、Codex `create_thread`、Desktop-managed worktree、禁止执行 Agent自我批准、用户控制 merge，以及 clean/`origin/main`/远端分支/prune 的 post-merge cleanup 标记。
- [x] 2.2 运行 `go test ./internal/architecture -run '^TestEngineeringWorkflowPolicy$' -count=1`，确认测试仅因当前策略文档缺少目标规则而失败，并在任务证据中记录失败命令、缺失标记和 RED 原因。

## 3. 委派式工程工作流（GREEN）

- [x] 3.1 更新 `AGENTS.md`，将开发 Leader与执行 Agent职责、两道 Leader门禁、用户控制 PR merge 和扩展生命周期写入强制入口，同时保持文件精简并指向专项规则。
- [x] 3.2 更新 `.agents/openspec-workflow.md` 和 `.agents/skill-routing.md`，加入 `Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup`，明确 Leader只编排/评审/验收、执行 Agent端到端实施且不能自我批准。
- [x] 3.3 更新 `.agents/git-workflow.md`，要求 Leader通过 Codex `create_thread` 创建独立执行任务和 Desktop-managed worktree，保留 `codex/<change-name>`，并定义 merged 后从主项目执行 clean、`origin/main` 可达性、worktree、本地/远端分支和 prune 的安全清理顺序及失败报告。
- [x] 3.4 按需要更新 `openspec/config.yaml`，使后续 proposal、design、specs 和 tasks 都明确记录委派职责、两道 Leader门禁、用户 merge 和 Cleanup 约束。
- [x] 3.5 运行 `go test ./internal/architecture -run '^TestEngineeringWorkflowPolicy$' -count=1`，确认最小策略更新满足全部新断言并记录 GREEN 证据。

## 4. 重构与验证（REFACTOR）

- [x] 4.1 在测试保护下消除 `AGENTS.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/skill-routing.md` 和 `openspec/config.yaml` 的重复或职责歧义，不削弱稳定策略标记。
- [x] 4.2 对变更的 Go 文件执行 `gofmt`，再次运行聚焦测试，并运行 `go test ./...` 记录 REFACTOR 与全量验证证据。
- [x] 4.3 运行 `openspec validate establish-leader-delegation-workflow --strict` 和 `git diff --check`，检查变更范围、凭据模式、`.env`、`data/`、`.reference/cloudwego/` 及已有 `adopt-openspec-tdd-workflow` worktree 均未进入 change。

## 5. Leader Acceptance 门禁

- [x] 5.1 执行 Agent向开发 Leader提交完整 diff 范围、需求映射、RED/GREEN/REFACTOR 证据、聚焦与全量测试、strict validation、残余风险和回滚说明；复审后补充 `change worktree clean`、`远端分支` 策略断言及 Leader Review 主体文案修正的验证证据。
- [x] 5.2 等待开发 Leader明确批准 Apply-final Review / Leader Acceptance；该 checkbox 只能在收到 Leader验收后由执行 Agent记录，执行 Agent不得自行批准，未批准前不得执行 Sync、Archive 或 Deliver。

## 6. Sync、Archive 与 Deliver

- [x] 6.1 获得 Leader Acceptance 后，执行 Agent使用 `$openspec-sync-specs` 将 delta spec 同步到 `openspec/specs/engineering-change-workflow/spec.md`，核验未丢失既有 requirements/scenarios。
- [x] 6.2 执行 Agent使用 `$openspec-archive-change` 归档 change，并重新运行策略测试、`go test ./...`、最终 OpenSpec validation 和 `git diff --check`。
- [x] 6.3 执行 Agent仅暂存当前 change 文件，检查 staged diff、密钥和禁止路径后，提交并推送 `codex/establish-leader-delegation-workflow`，创建 base 为 `main` 的 Pull Request，报告地址、验证证据和已知风险；不得 merge PR。
- [x] 6.4 PR 创建后，执行 Agent向 Leader提交 cleanup handoff：PR URL/number、change branch、Desktop worktree path、必要 commits，以及用户 merge 后从主项目执行的 merged 确认、worktree clean、`origin/main` 可达性、删除 worktree、本地/可选远端分支与 prune 的命令/顺序。此 handoff 完成后，本 change 的 OpenSpec checkbox 可全部完成；用户 Merge 与 Leader Cleanup 由主 Codex 任务作为归档后的 operational state 跟踪，不回写已归档 tasks。
