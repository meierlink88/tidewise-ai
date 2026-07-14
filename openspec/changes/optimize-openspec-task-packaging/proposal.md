## Why

现有风险分级规则已经说明普通 task checkbox 不自动成为人工 gate，但 Propose 阶段仍缺少可执行的任务包装约束与自动校验，导致合理范围被拆成测试、dry-run、checkpoint、commit/push 等微型停顿。需要把任务重新组织为内聚交付 package，并用轻量 lint 阻止可确定的退化，同时保留全部高风险硬门禁。

## Gate Map

| Gate | 风险类型 | 人工 | 原因 | 通过后允许范围 |
|---|---|---|---|---|
| Proposal Review | R0 proposal / R1 change | 是 | `SPEC_SEMANTICS`：确认任务包装规则、lint 边界、复杂度预算与共享规则影响 | 仅允许进入一个 R0/R1 Apply package；不授权数据库、Neo4j、部署、Sync、Archive 或 Deliver |
| R0/R1 Apply package | R1 | 否 | 同一无状态风险边界内连续完成规则、测试、lint 与验证 | 允许完成包内实现、targeted 验证和一个 scoped checkpoint；失败立即留在包内整改 |
| Apply-final Review | R1 | 是 | `APPLY_FINAL`：共享工作流规则准备同步到主规格 | 仅在审阅通过后允许 Sync、Archive、Deliver；不授权任何 R2/R3 操作 |

## What Changes

- 规定 OpenSpec 一级 task 必须表达内聚交付 package，而不是单个操作步骤；Proposal 与 tasks 必须提供 Gate Map 和复杂度预算。
- 限定人工 gate 的合法原因，并禁止把普通源码实现、测试/修复、dry-run、validate、diff/secret check、commit/push 单独升级为人工 gate。
- 规范已批准 Spec 下精确匹配的 local-only R2 条件式执行包：包内可以逐名列出多个顺序层，每层严格执行 `preflight -> Write -> Query/assert`，只有当前层通过才自动进入已明确授权的下一层；漂移或失败立即停止。
- 允许同一环境、同一维护窗口且基础状态未变化时复验并复用 recovery baseline，避免为每层重复制造完整 backup package；不降低 recovery evidence 与 fail-closed 要求。
- 将验证组织为 package：开发中 targeted、package checkpoint 一次匹配验证、Apply-final 一次受影响边界完整验证。
- 将候选数据审阅统一为“全量机器校验 + 异常、冲突、宽边界、低置信度和用户明确指定项人工审阅”。
- 在现有 Go 工作流架构测试模式上设计 task-design lint，不引入新依赖；确定性违规 fail，启发式复杂度 warning，仅检查 active changes 或显式传入 change，并提供合规/违规 fixture。
- 明确新规则只默认适用于本 change Deliver 后创建的新 change；不自动 adoption 或改写任何 active change。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `skill-driven-development-workflow`: 增加 Propose 阶段任务包装、Gate Map、复杂度预算、合法人工 gate、条件式执行包复验、package 验证、候选审阅与 task-design lint 的行为要求。

## Complexity Budget

- 预计人工 gate：2 个（Proposal Review、Apply-final Review）。
- 预计 stateful layers：0；本 change 不执行 R2/R3 操作。
- 预计 checkpoint：2 个（Proposal checkpoint、一个 R0/R1 Apply package checkpoint）。
- 预计完整测试次数：1 次（Apply-final 的受影响边界完整验证）；开发中只运行 targeted tests，package checkpoint 运行一次匹配验证。
- 可连续自动执行范围：Proposal Review 通过后，测试先行、规则实现、lint/fixture、targeted 验证、修复、diff/secret check、checkpoint 与 push 在一个 R0/R1 Apply package 内连续执行，直到 Apply-final Review。

## Impact

- 当前 Proposal checkpoint 为 R0；未来 Apply 最高为 R1，因为可能修改共享规则、`backend/internal/architecture/` 下的 lint/测试 fixture 和 `.github/workflows/ci.yml` 的最小接入点，但不执行有状态写入。
- 当前 artifacts 仅位于 `openspec/changes/optimize-openspec-task-packaging/`。未来 Apply 才可能修改 `AGENTS.md`、`.agents/openspec-workflow.md`、必要的 lint 脚本/测试/fixture 与 CI；不修改业务源码、API、前端、migration、seed、PostgreSQL、Neo4j 或部署状态。
- `prototype/` 只读且不是输入；`doc/` 不更新；不引入第三方依赖。
- 不重写 OpenSpec CLI，不扫描或改写 archive 历史，不自动改写 active change，不修改 `refactor-industry-chain-node-foundation` 的 worktree、tasks、artifacts 或数据库状态。
- 不削弱 R3、Neo4j、UAT/prod/shared、部署/secret/权限、漂移/失败恢复、Apply-final、PR merge/cleanup 硬门禁，也不调整任何业务功能。
