## Why

当前工程规则同时在 `AGENTS.md`、`openspec/config.yaml`、`.agents` 专责文件和主 workflow spec 中重复描述，查找路径、维护边界和 gate 语义容易漂移。现在前置 workflow change 已交付，适合在不降低门禁的前提下建立长期、可验证且职责单一的规则结构。

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

## What Changes

- 将 `AGENTS.md` 收敛为最高级硬门、规则路由和不可绕过摘要。
- 将 `openspec/config.yaml` 收敛为稳定项目背景、语言和 artifact 写作约束。
- 使 `.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md` 分别成为生命周期/审批、Git/交付、测试/验证的唯一详述来源。
- 将主 workflow spec 收敛为长期可验证行为，移除本 change、历史行数/压缩率、一次性迁移和旧验收指标。
- 统一 gate、package、checkpoint、commit 的定义和引用关系，并维护硬门语义覆盖矩阵。
- 保留 OpenSpec、Superpowers、TDD、CI、风险分级、local/UAT/prod/shared、有状态写安全、PostgreSQL/Neo4j 事实源和 Desktop cleanup 语义。
- 已完成的 Proposal 自检证据归属于 Package 1，不作为 Apply 待办；Apply package 批准后连续完成规则实施与必要 contract 调整。
- Apply-final 只执行受影响边界的 targeted verification、scoped diff/证据和安全检查；不运行 `go test ./...`。

## Capabilities

### New Capabilities

无。本 change 不引入新的运行时能力。

### Modified Capabilities

- `skill-driven-development-workflow`：重组规则职责并保留长期可验证的研发、审批、验证和交付行为。

## Impact

- 受影响仓库区域仅为 `AGENTS.md`、`openspec/config.yaml`、`.agents/skill-routing.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md`、`openspec/specs/skill-driven-development-workflow/spec.md` 及本 change artifacts。
- 不修改 `backend/` 业务逻辑、数据库/Neo4j 数据、部署、`doc/` 或 `prototype/`。
- 不新增依赖、API、运行时数据路径或 CI job；Apply 阶段如需调整 architecture/workflow 检查，仅限既有检查与本规则契约一致。
