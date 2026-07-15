## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review；批准后连续执行内聚 Apply package | R1 | yes | SPEC_SEMANTICS | 仅限本 change 的 workflow 规则、OpenSpec artifacts、现有 lint 的最小调整和稳定资源路径迁移设计；不得执行 Apply、数据库/图谱写入、UAT/prod 或修改 backend 业务代码 |
| 2 | Apply-final Review；通过后才可 Sync/Archive/Deliver | R1 | yes | APPLY_FINAL | 仅限受影响 workflow/architecture 规则边界的完整验证、OpenSpec strict、diff/secret 检查及本 change scoped 交付；不得扩大到未批准环境或状态写入 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

说明：`checkpoints=2` 仅指需要停顿并重新验收的 Proposal Review 与 Apply-final Review。Proposal 批准后的 Package 1 子项及其 commit 是连续执行证据；Archive/Deliver 只是生命周期记录，在输入指纹不变时不新增人工停顿或重复全量验证。`continuous_automation_scope=packages:none` 是因为两行 package 均为 `Human=yes`；它不否定 Proposal 批准后 Package 1 内子项连续执行，也不表示可跳过人工起始 gate。

## 1. Workflow contract and stable-resource Apply Package

- [ ] 1.1 [RED] 为 package/gate 聚合、local-only 快速模式、stateful 一次 preflight/Write/verify、checkpoint 指纹与证据复用、stable resource path 建立 Go architecture fixture/table-driven tests；确认测试不访问网络、凭证、生产数据库或运行时 manifest。
- [ ] 1.2 [GREEN] 按 delta spec 最小修改 `.agents/openspec-workflow.md`、`.agents/skill-routing.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md` 或根规则中实际命中的重复/冲突条目；保留 OpenSpec 生命周期、Superpowers、TDD、CI、UAT/prod 与 R2/R3 安全门禁，不新增 runner 或独立 lint 工具。
- [ ] 1.3 [GREEN] 仅在真实消费者需要时，将 merged manifest 运行时输入迁移到稳定 backend `data/` 或 `resource/` 路径；OpenSpec change 仅保存 review snapshot/hash/evidence，完成只读路径、hash、schema 和 archive 后消费验证；若无真实 active-path 消费者则移除迁移项。
- [ ] 1.4 [REFACTOR] 运行 targeted architecture/lint tests，修复失败并保持同一 package；将 package scope、non-goals、风险、证据、未验证项、阻断项、停止条件和下一步写入 checkpoint，失败不得新增人工 gate或隐式扩大授权。
- [ ] 1.5 运行 package checkpoint 验证、`git diff --check`、scoped diff/secret 检查，复读结果并完成 self-review；只提交本 change 相关规则、spec、测试和稳定资源文件，不包含 backend 业务代码、数据库/图谱状态、UAT/prod 或 secret。

## 2. Apply-final verification Package

- [ ] 2.1 运行受影响 workflow/architecture 边界的完整验证：`go test ./...`、OpenSpec strict validation、现有 task-design lint、规则/路径/hash 检查及 scoped diff/secret 检查。
- [ ] 2.2 记录 repo-wide full validation 的触发理由、命令、结果、未验证项、输入指纹和 checkpoint 下一步；任一失败、漂移或边界不清立即停止并扩大验证或等待处理。
- [ ] 2.3 完成 Apply-final self-review/code review，确认 proposal、design、delta spec、tasks、实现和验证证据一致；仅在用户通过 Apply-final Review 后执行 Sync、Archive、`openspec validate --all` 和 Deliver。
