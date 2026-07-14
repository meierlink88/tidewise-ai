## Gate Map

| Gate | Risk | Human | Reason | Allowed Scope |
|---|---|---|---|---|
| Proposal Review | R0 proposal / R1 change | yes | `SPEC_SEMANTICS`：确认任务包装、lint 与共享规则设计 | 仅进入第 2 个 R0/R1 Apply package；不授权 R2/R3、数据库、Neo4j、部署、Sync、Archive 或 Deliver |
| R0/R1 Apply package | R1 | no | 同一无状态风险边界内连续执行 | 完成测试先行、规则/lint、targeted 验证、修复和一个 scoped checkpoint |
| Apply-final Review | R1 | yes | `APPLY_FINAL`：共享规则准备同步 | 审阅通过后才允许 Sync、Archive、Deliver；不授权任何有状态操作 |

## Complexity Budget

- 人工 gate：2。
- Stateful layers：0。
- Checkpoint：2（当前 Proposal checkpoint、一个 R0/R1 Apply package checkpoint）。
- 完整测试：1 次（Apply-final 的受影响边界完整验证）。
- 连续自动执行范围：Proposal Review 通过后，第 2 个 package 内的测试、实现、lint、targeted 验证、修复、diff/secret check、commit/push 连续执行，直到 Apply-final Review。

## 1. Proposal Review Package

- [ ] 1.1 完成中文 proposal、design、delta spec 与 tasks，运行 `openspec validate optimize-openspec-task-packaging --strict`、scoped diff/secret/所有权检查并创建、推送 `spec: propose optimize-openspec-task-packaging` checkpoint；主对话人工确认 Gate Map、复杂度预算、合法 gate、R2 baseline 复用、lint fail/warning 边界、active change 不 adoption 与全部 non-goals 后，才允许进入第 2 个 package。

## 2. R0/R1 Apply Package

- [ ] 2.1 测试先行：在 `backend/internal/architecture/testdata/task_design/` 增加合规和违规 tasks fixture，并先用失败测试覆盖缺 Gate Map/复杂度预算、非法人工 gate、stateful 字段缺失、archive 排除、explicit scope，以及微型 Review/checkpoint/commit/push 只 warning 的边界。
- [ ] 2.2 在 `.agents/openspec-workflow.md` 落实一级 package、Gate Map、复杂度预算、合法人工 gate、package 验证、候选全量机器校验与例外人工审阅、local-only R2 多层执行和 recovery baseline 复验；仅在必要时对根 `AGENTS.md` 保留短摘要，不复制详细流程。
- [ ] 2.3 复用 `backend/internal/architecture/` 的 Go 标准库测试模式实现 task-design lint：确定性违规 fail、启发式复杂度 warning，仅检查 active 或显式 change，排除 archive 并保护 legacy active baseline；通过现有 backend `go test ./...` 最小接入 CI，仅在现有入口不足时增加薄 wrapper，不新增依赖或平行 job。
- [ ] 2.4 完成 targeted `go test ./internal/architecture -run TaskDesign -count=1`、当前 change 显式 lint、OpenSpec strict validation、规则链接/重复检查、`git diff --check`、scope/secret 检查和 self-review；失败与修复留在本 package 内，确认未修改 active change、业务源码、数据库或部署状态。
- [ ] 2.5 汇总一个 R1 package evidence，记录受影响边界、fixture、fail/warning 结果、legacy active baseline、未验证项和阻断项；创建并推送一个 scoped Apply package checkpoint，不为 2.1—2.4 分别制造 Review、commit 或 push。

## 3. Apply-final Review Package

- [ ] 3.1 运行一次受影响边界完整验证：backend `go test ./...`、共享 architecture/contract tests、`openspec validate optimize-openspec-task-packaging --strict`、规则链接/重复检查、`git diff --check`、scope/secret 检查；复读新鲜结果并确认 stateful layers 为 0、无 R2/R3 状态变化。
- [ ] 3.2 提交 scoped diff、完整验证、self-review 和第 2 个 package checkpoint 证据，等待主对话 Apply-final 人工 Review；通过前不得 Sync、Archive、Deliver、创建完成态 PR，或修改任何 active change artifacts/tasks。
