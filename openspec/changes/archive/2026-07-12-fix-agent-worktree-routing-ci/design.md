## Context

`streamline-agent-rules` 已把 Git/worktree 规则收敛为正确的两条路径：Codex Desktop 可用时必须由 Desktop 新任务创建受管 worktree；只有 Desktop 受管机制不可用且用户明确批准时，才允许 project-owned fallback。精简过程中，`.agents/skill-routing.md` 删除了 `superpowers:using-git-worktrees` 的全部引用，造成 `TestSkillDrivenWorkflowRules` 在 CI 与本地稳定失败。

同一测试还通过若干旧字面短语间接验证生命周期。这些短语不再是当前规则正文的标准表达，因此即使真实语义完整，测试仍可能误报。此 hotfix 只修复规则路由和其架构契约测试，不修改归档历史、其他 active change 或产品运行时代码。

## Goals / Non-Goals

**Goals:**

- 恢复 `superpowers:using-git-worktrees` 的精确条件映射，同时保留 Desktop-managed 强制默认路径。
- 让架构测试直接验证稳定的条件语义与阶段顺序，而不是要求恢复旧措辞。
- 用 TDD 记录现有 CI 失败，并验证最小修复后的 architecture package、全量 Go 测试和 OpenSpec 校验。

**Non-Goals:**

- 不允许在 Desktop 可用时把 `superpowers:using-git-worktrees` 当作默认创建路径。
- 不放宽“Desktop 不可用”和“用户明确批准”两个 fallback 前置条件。
- 不修改 `streamline-agent-rules` 归档历史、`add-market-sector-foundation`、其他 active change、数据库、前端、API、`prototype/` 或 `doc/`。
- 不在本轮 Apply；proposal checkpoint 后等待人工 Review。

## Decisions

### 1. 在 Skill 路由中表达条件映射，不复制完整 Git 流程

`.agents/skill-routing.md` 将说明 Desktop-managed 路径由 Desktop 新任务机制负责；只有 approved fallback 才调用 `superpowers:using-git-worktrees`，完整条件和操作仍唯一引用 `.agents/git-workflow.md`。

备选方案是把完整 Desktop/fallback 流程复制进 Skill 路由。该方案会制造第二份 Git 事实来源，增加规则漂移，因此不采用。

### 2. 测试稳定语义锚点与顺序，不测试过期叙述

`workflow_rules_test.go` 将分别验证：

- Skill 路由包含 Desktop 新任务机制、approved fallback 与 `superpowers:using-git-worktrees` 的条件关系。
- Git 规则同时包含 Desktop 受管强制入口和 fallback 双条件，不出现将“原生 worktree”设为默认路径的契约。
- OpenSpec 规则保持 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver` 顺序，并明确两次人工 Review 门禁。
- Git 规则保持 sequential/parallel 分流、Desktop-managed cleanup、project-owned fallback cleanup 的关键顺序。
- `superpowers:finishing-a-development-branch` 只在 Archive 之后进入 Deliver。

备选方案是仅把旧短语重新写回规则正文以让测试通过。该方案会把测试实现细节反向污染规则，并可能重新暗示默认手工 worktree，因此不采用。

### 3. TDD 以失败契约和最小规则修复为边界

Apply 时先运行现有测试记录 CI RED，再先修改测试以表达新语义并确认它仍因缺少条件映射而 RED；随后只修改 `.agents/skill-routing.md` 和必要的测试 helper/assertions，直到 GREEN。测试是静态规则契约测试，不需要 fixture、fake、外部网络、secret 或数据库。

验证层级为：`go test ./internal/architecture -count=1`、`go test ./... -count=1`、`openspec validate fix-agent-worktree-routing-ci`，Apply 完成后的全局检查再运行 `openspec validate --all`。

## Risks / Trade-offs

- [风险] 只依赖任意关键词仍可能把两个条件拆散后误判为通过 → 使用同一规则段落的组合断言或专用 helper 验证 Desktop 默认与 approved fallback 的关联。
- [风险] 顺序断言过度依赖中文微调 → 选择生命周期阶段名、规则标题和明确操作作为稳定锚点，避免依赖解释性整句。
- [风险] hotfix 与其他 active change 文件重叠 → scoped diff 仅允许本 change artifacts；Apply 预期只触碰 `.agents/skill-routing.md` 和 `backend/internal/architecture/workflow_rules_test.go`，发现重叠立即暂停。
- [权衡] 静态文本测试不能证明 Desktop UI 自身行为 → 本测试的职责仅是防止 repo 规则契约回退，Desktop 机制由产品环境负责。

## Migration Plan

无需数据迁移或运行时部署。Apply 经人工批准后按 RED/GREEN 顺序提交最小规则与测试修改；若验证失败，可撤回当前 change 的未交付修改，不影响产品数据或其他 change。归档与交付必须继续遵循 OpenSpec Review、Sync、Archive、Deliver 门禁。

## Open Questions

无。范围、默认路径、fallback 条件和验证命令均由当前项目规则与本次委派明确确定。
