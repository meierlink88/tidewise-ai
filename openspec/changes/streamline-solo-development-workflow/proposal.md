## Why

当前正式研发流程把 targeted test、dry-run、修复、commit 和 push 过度拆成重复 checkpoint，且无状态错误容易触发不必要的全量重验；运行命令还可能依赖会随 OpenSpec archive 移动的 active path。现在需要在不削弱 OpenSpec、Superpowers、TDD、CI 或 UAT/prod 安全门禁的前提下，为单人 local 开发建立可恢复、可复用证据且可量化的快速模式。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review；批准后连续执行内聚 Apply package | R1 | yes | SPEC_SEMANTICS | Proposal 批准后仅限本 change 的 workflow 规则、OpenSpec artifacts、现有 lint 的最小调整和稳定资源路径迁移评估；不得执行数据库/图谱写入、UAT/prod 或修改 backend 业务代码 |
| 2 | Apply-final Review；通过后才可 Sync/Archive/Deliver | R1 | yes | APPLY_FINAL | 仅限受影响 workflow/architecture 规则边界的完整验证、OpenSpec strict、diff/secret 检查及本 change scoped 交付；不得扩大到未批准环境或状态写入 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 0 |
| continuous_automation_scope | packages:none |

说明：`checkpoints=2` 仅指需要停顿并重新验收的 Proposal Review 与 Apply-final Review。Proposal 批准后的 Package 1 子项及其 commit 是连续执行证据；Archive/Deliver 只是生命周期记录，在输入指纹不变时不新增人工停顿或重复全量验证。`continuous_automation_scope=packages:none` 是因为两行 package 均为 `Human=yes`；它不否定 Proposal 批准后 Package 1 内子项连续执行，也不表示可跳过人工起始 gate。

## What Changes

- 建立全局测试选择策略：任意 change 按真实受影响交付边界选择验证；workflow/OpenSpec/architecture 规则类变更 targeted，局部 coding 跑受影响 suite，数据-only 以状态断言验证，只有共享运行时代码/契约/基础设施或边界不清才 repo-wide。
- 明确开发、local、shared-local、UAT/prod 的门禁分流：快速连续模式仅适用于 local；真实共享资源写入、UAT/prod、migration/seed/data repair、secret/权限、回滚仍保留严格 gate，且不因测试范围优化削弱。
- 将 targeted tests、实现、修复、dry-run、验证、commit 和 push 聚合为一个内聚 Apply package；普通 coding、测试、commit、push 和无状态命令错误不单独制造人工 gate。
- 规定 stateful 操作在获准范围内执行一次完整 preflight、单次 Write、一次 post-write verify；状态指纹未变化时，后续 Archive/PR 阶段复用证据，仅重验变化或失败层。
- 增加 checkpoint 记录要求：阶段、commit、已通过验证、状态指纹、下一步和真实 blocker，使恢复从下一步继续；无状态错误不得触发数据库全量重验。
- 将运行命令消费的数据从 active OpenSpec path 解耦：稳定运行资源进入 backend 稳定 data/resource 路径，OpenSpec 仅保存 review snapshot/hash/evidence；现有 merged manifest 迁移限于窄 R1 Apply scope。
- 仅在必要时最小调整现有 task-design lint；默认不新增通用 preflight/apply/verify runner、独立 lint 工具或自动 path test，按 YAGNI 优先使用规则与现有测试。
- 保持 OpenSpec `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver`、Superpowers、TDD/unit tests、CI 和 UAT/prod 安全门禁不变。

## Capabilities

### New Capabilities

- 无：本 change 优化既有研发 workflow，不新增独立产品能力。

### Modified Capabilities

- `skill-driven-development-workflow`: 修改 package/gate、环境分级、证据复用、断点恢复和稳定运行资源路径要求。

## Impact

- 仓库区域：仅涉及 `tidewise-ai` 的 `.agents/` workflow 规则、`openspec/specs/skill-driven-development-workflow/` delta、必要的 `backend/internal/architecture/` 现有 lint/测试最小调整，以及稳定运行资源路径的窄 R1 迁移范围；本 Proposal 阶段不修改这些生产文件。
- 不涉及 `prototype/`；不修改 `doc/`，也不复制 prototype HTML、DOM 或内联脚本。
- 不修改 backend 业务代码、运行时 manifest、数据库、图谱或部署配置；path migration 若经 Review 批准，仅迁移现有 merged manifest 的稳定消费路径，并保留 OpenSpec review snapshot/hash/evidence。
- 兼容性：不改变 OpenSpec 生命周期、对外 API、PostgreSQL/Neo4j 事实源边界、CI/UAT/prod 安全策略或现有命令语义；缺少可验证指纹、发生漂移或验证失败时必须 fail-closed。
- 测试边界：本 change 将全局测试选择策略写入共享 workflow：OpenSpec/workflow/agent/architecture rule 类变更 targeted；局部 coding 跑受影响 suite；数据-only 运行 manifest/dry-run/preflight/post-write assertions；只有共享运行时代码、跨模块运行时契约、公共运行时基础设施或边界不清时才 repo-wide。当前 change 属于第一类，不运行 `go test ./...`；无新增网络、真实凭证或生产数据库依赖。
- 风险等级：基线 R1；本轮不执行 R2/R3 有状态操作。任何 UAT/prod、shared 写入、migration/seed/data repair、secret/权限、回滚或图谱操作必须另行上调风险并取得独立授权。
- 可量化验收：普通 coding 路径最多保留 Proposal 与 Apply-final 两次人工确认；单个 local stateful package 最多一次 preflight、一次 Write、一次 verify；相同输入指纹下 Archive/PR 不重复全量重验；workflow-only 变更的受影响 architecture/workflow 测试、`openspec validate --strict`、task-design lint、`git diff --check`、scope/secret/链接检查全部通过；不新增 runner、独立 lint 工具或 path test。
