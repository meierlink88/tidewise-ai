## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review：确认规则职责、长期语义和覆盖矩阵 | R1 | yes | SPEC_SEMANTICS | 仅审阅本 change OpenSpec artifacts；该 gate 已由用户明确批准继续，不追认 Apply |
| 2 | R1 Apply package：规则去重、职责归位、主 spec 重写、覆盖矩阵和必要 architecture contract 调整连续完成 | R1 | no | NONE | 仅规则文件、主 workflow spec、architecture workflow contract 和 change evidence；不涉及业务代码、数据库、图谱、部署、doc 或 prototype |
| 3 | Apply-final Review：完成范围匹配验证、scoped diff/证据和 Apply commit/push 后停在人工 Review | R1 | yes | APPLY_FINAL | 仅审阅 Apply 交付边界与新鲜证据；不得 Sync、Archive、Deliver、PR、merge 或 cleanup |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 0 |
| continuous_automation_scope | packages:2 |

## 1. Proposal Review Package

- [x] 1.1 已完成 Proposal artifacts、Gate Map/Complexity Budget、范围和非目标审阅；用户已明确批准继续进入 R1 Apply。
- [x] 1.2 已完成 Proposal strict、精确 task-design lint、workflow targeted checks、`git diff --check`、scope/secret/link 检查和 Proposal checkpoint commit/push；证据属于 Proposal Review，不作为 Apply 待办。

## 2. 规则收敛 Apply Package

- [ ] 2.1 建立并保存语义覆盖矩阵，将 OpenSpec 顺序与人工 Review、Desktop-managed 入口、sequential/parallel 分流、两类 cleanup、风险分级、有状态写、TDD/CI/验证、事实源和安全边界逐项映射到唯一详述来源与自动化锚点。
- [ ] 2.2 收敛 `AGENTS.md`：只保留最高级硬门、规则路由、不可绕过的风险/安全摘要和来源指引；删除生命周期、Git、测试的重复详述。
- [ ] 2.3 收敛 `openspec/config.yaml`：只保留稳定项目背景、语言约束和 artifact 写作约束；删除 workflow、Git、测试和一次性迁移规则。
- [ ] 2.4 收敛 `.agents/skill-routing.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md`：分别保留 Skill 路由、OpenSpec 生命周期/审批、Git/交付、TDD/验证的唯一完整详述，删除跨文件重复并保留必要入口摘要。
- [ ] 2.5 重写主 workflow spec 的受影响 requirements/scenarios，移除本 change、历史行数/压缩率、一次性迁移和旧验收指标，保留长期可验证行为与 delta spec 完整性。
- [ ] 2.6 根据现有 architecture workflow contract 的真实断言更新必要的规则语义锚点；不得删除或削弱 Desktop、fallback、sequential/parallel、cleanup、风险、package、验证和事实源检查。

## 3. Apply-final Review Package

- [ ] 3.1 运行 `openspec validate consolidate-openspec-workflow-rules --strict` 和精确 task-design lint。
- [ ] 3.2 运行受影响 workflow architecture targeted checks、`git diff --check`、scope/secret/link 检查、重复/冲突扫描和覆盖矩阵自审；不运行 `go test ./...`，并记录受影响边界与未验证项。
- [ ] 3.3 完成独立 reviewer 自审：逐项核对覆盖矩阵、唯一职责来源、门禁无降级、scope 和 fresh checks；阻断项必须在本 package 内修复并刷新证据。
- [ ] 3.4 仅暂存本 change 的规则/spec/architecture contract 文件，创建 `spec: apply consolidate-openspec-workflow-rules` scoped Apply commit 并 push；随后停在 Apply-final Review，不执行 Sync、Archive、Deliver、PR、merge 或 cleanup。
