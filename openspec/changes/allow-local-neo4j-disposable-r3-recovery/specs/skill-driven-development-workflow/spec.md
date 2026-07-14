## MODIFIED Requirements

### Requirement: Stateful Layer Map 必须完整映射有状态层
系统 MUST 在 `stateful_layers>0` 时要求 Proposal 与 tasks 在 Complexity Budget 后、首个编号 package 前提供内容一致的 `## Stateful Layer Map`。固定列 MUST 依次为 `Layer`、`Package`、`Environment`、`Order`、`Scope`、`Exclusions`、`Recovery Evidence`、`Recovery Baseline`、`Expected Counts/Hash/Schema`、`Before Assertions`、`After Assertions`、`Stop Conditions`。`Layer` MUST 是唯一 kebab-case；`Package` MUST 引用 Risk 为 R2/R3 的 Gate Map package；`Environment` MUST 是 `local`、`shared-local`、`uat` 或 `prod`；`Order` MUST 在 package 内从 1 连续且唯一；空 exclusions MUST 写 `none`。Recovery Evidence MUST 是 `backup` 或 `approved-disposable-recovery`，Recovery Baseline MUST 是 `new:<kebab-id>` 或引用同环境更早 order 的 `reuse:<kebab-id>`；expected state MUST 同时提供 `counts=<value>;hash=<value>;schema=<value>`，不适用值写 `na`。其余范围与断言字段 MUST 单行非空。

`approved-disposable-recovery` MUST 保留对符合既有 disposable 条件的 local R2 layer 的支持。它也 MUST 仅在以下条件全部满足时允许 local R3 layer 使用：Layer 或 Scope 以具有 ASCII token 边界的 `Neo4j` 与 `cleanup`、`rebuild` 或 `sync` 中一个封闭 operation token 共同明确为 Neo4j projection operation，不要求额外出现字面量 `projection`；Before Assertions 以同样有边界的 `PG` 或 `PostgreSQL` 与 `baseline` 明确引用事实源 baseline；上述匹配 MUST 大小写不敏感且不得把 cleanupSuffix、resync 或 neo4jBackup 视为合法 token；PostgreSQL 是冻结、已验收且可完整重建该投影的唯一事实源；Gate 的 Risk 为 R3 且 Human 为 yes；Before/After Assertions、Stop Conditions 与 expected state 完整；用户在运行前对该命名 layer 单独明确授权。该 recovery 只表达恢复策略，不构成操作授权。`shared-local`、UAT、prod/shared、生产、非 Neo4j R3、非 projection、operation 不在封闭集合或无法从 PostgreSQL 完整重建的状态 MUST 拒绝 disposable recovery，并要求 `backup` 或等价正式灾备。Neo4j R3 的独立授权、逐层 `preflight -> Write -> Query/assert`、失败立即停止、未执行授权失效和禁止跨层批量 MUST 保持不变。本 requirement 不定义或引入 UAT Neo4j recovery、backup、deployment、adoption 或验收能力。

当 `stateful_layers=0` 时系统 MUST 允许省略 Stateful Layer Map；若仍提供该表，则 MUST 使用固定表头且不得包含数据行。linter MUST 校验数据行数等于预算值，并只通过 Package ID 关联 layer 与 package。

#### Scenario: 无有状态层省略 Stateful Layer Map
- **WHEN** Complexity Budget 声明 `stateful_layers=0`
- **THEN** Proposal 与 tasks 可以省略 Stateful Layer Map，且 lint 不得要求空占位表

#### Scenario: 多层条件式执行包完整映射
- **WHEN** 一个 R2 package 包含两个已命名 local layer
- **THEN** Stateful Layer Map 必须有两行、引用同一 Package、使用连续 Order 1 和 2，并完整提供环境、范围、排除、recovery、expected state、前后断言和停止条件

#### Scenario: local Neo4j R3 disposable projection 合法表达
- **WHEN** 一个 local R3 layer 的 Layer 或 Scope 以有边界 token 明确包含 Neo4j 与 `cleanup`、`rebuild` 或 `sync`，Before Assertions 以有边界 token 引用已冻结验收且可完整重建投影的 PG/PostgreSQL baseline，Gate Human 为 yes，且全部字段与命名 layer 独立授权完整
- **THEN** task-design lint 必须允许 `approved-disposable-recovery`，但该结果不得被解释为已授权执行该 layer

#### Scenario: disposable R3 环境或状态越界
- **WHEN** `approved-disposable-recovery` 用于 shared-local、UAT、prod/shared、生产、非 Neo4j R3、非 projection、非 cleanup/rebuild/sync 或未引用 PG/PostgreSQL baseline 的 layer
- **THEN** task-design lint 必须 fail-closed，并要求 backup 或等价正式灾备

#### Scenario: local R2 disposable recovery 保持兼容
- **WHEN** 一个 local R2 layer 满足既有 approved disposable recovery 条件
- **THEN** task-design lint 必须继续允许该 recovery，不得因新增 Neo4j R3 例外改变原有行为

#### Scenario: Stateful layer 无法对应 package
- **WHEN** layer 引用不存在的 Package，或引用 Risk 为 R0/R1 的 package
- **THEN** task-design lint 必须 fail，不得根据 layer 名称或 scope 文案猜测归属

#### Scenario: 复用 recovery baseline
- **WHEN** 某行使用 `reuse:<baseline-id>`
- **THEN** 同一 Environment 的更早 Order 必须已有 `new:<baseline-id>`，before assertions 必须复验 identity、scope、count/hash/schema；否则 lint 必须 fail

#### Scenario: R3 recovery 不降低执行门禁
- **WHEN** 同一 package 声明多个获准表达 disposable recovery 的 local Neo4j R3 layer
- **THEN** 每个命名 layer 仍必须分别取得明确授权并逐层执行，任何失败或中止必须停止且不得把授权推定到下一层
