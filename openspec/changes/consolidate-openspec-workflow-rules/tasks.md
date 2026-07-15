## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review：确认规则职责、长期语义和覆盖矩阵 | R1 | yes | SPEC_SEMANTICS | 仅审阅本 change OpenSpec artifacts；不得 Apply、Sync、Archive、Deliver 或修改业务代码 |
| 2 | Proposal 自检与 checkpoint：运行范围匹配检查并提交可供 Review 的 Proposal | R1 | no | NONE | 仅 OpenSpec artifacts、规则文本和验证证据；允许 scoped commit/push，不创建完成态 PR |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 1 |
| stateful_layers | 0 |
| checkpoints | 1 |
| full_test_runs | 0 |
| continuous_automation_scope | packages:2 |

## 1. 规则职责与长期行为 Package

- [ ] 1.1 盘点 `AGENTS.md`、`openspec/config.yaml`、`.agents/skill-routing.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md` 与主 workflow spec 的重复、冲突和唯一职责边界。
- [ ] 1.2 将最高级硬门与路由、稳定项目背景与 artifact 写作约束、OpenSpec/Git/测试详述分别收敛到约定文件；删除重复的完整操作流程，但保留必要摘要、链接和不可绕过门禁。
- [ ] 1.3 重写主 workflow spec 的受影响 requirements/scenarios，移除本 change、历史行数/压缩率、一次性迁移和旧验收指标，保留长期可验证行为与 OpenSpec delta 完整性。
- [ ] 1.4 建立语义覆盖矩阵，逐项映射 OpenSpec 顺序与人工 Review、Desktop-managed 入口、sequential/parallel 分流、两类 cleanup、风险分级、有状态写、TDD/CI/验证、事实源和安全边界；缺项立即停止。

## 2. 范围验证与 Proposal Checkpoint Package

- [ ] 2.1 运行 `openspec validate consolidate-openspec-workflow-rules --strict`，确认 artifacts、delta spec、Gate Map、Complexity Budget 和链接结构有效。
- [ ] 2.2 运行 `OPENSPEC_TASK_LINT_CHANGE=consolidate-openspec-workflow-rules go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1`，仅验证本 change 的 package/gate/schema 约束。
- [ ] 2.3 运行必要的 workflow architecture targeted checks、`git diff --check`、scope 检查、secret 检查、仓库内链接检查和重复/冲突扫描；记录受影响边界、未验证项与不运行 `go test ./...` 的理由。
- [ ] 2.4 完成 Proposal 自审：复读验证输出，核对需求覆盖、OpenSpec artifacts、scope、secret、链接和非目标；发现阻断项先修复并刷新证据。
- [ ] 2.5 仅暂存当前 change 的 OpenSpec artifacts，创建 `spec: propose consolidate-openspec-workflow-rules` scoped commit，并 push `codex/consolidate-openspec-workflow-rules` 供 Proposal Review；不得创建完成态 PR。
