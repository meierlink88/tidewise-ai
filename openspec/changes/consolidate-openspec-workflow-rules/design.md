## Context

当前规则已经形成分层结构，但根规则、OpenSpec 配置、专责规则和主 workflow spec 仍交叉描述生命周期、Git、验证、gate 和历史迁移。主 workflow spec 还包含只适用于规则重构过程的度量与验收叙述。目标是收敛职责，而不是改变研发门禁或运行时行为。

## Goals / Non-Goals

**Goals:**

- 建立清晰的唯一职责来源：根级硬门与路由、配置背景、OpenSpec 生命周期、Git 交付、测试验证分别归位。
- 统一 gate、package、checkpoint、commit 的词义、生命周期位置和允许动作。
- 以长期可验证 requirement/scenario 表达硬门，保留 OpenSpec、Superpowers、TDD、CI、风险分级、环境隔离、有状态写安全和 cleanup 顺序。
- 在 Apply 设计语义覆盖矩阵、重复/冲突扫描、链接和 schema 检查，确认门禁没有丢失。

**Non-Goals:**

- 不修改 backend 业务逻辑、API、数据库、Neo4j、部署或 CI 实现。
- 不修改 `doc/`、`prototype/`，不复制 prototype 代码或资产。
- 不在本 Proposal 阶段修改长期规则正文，不执行 Apply、Sync、Archive、Deliver 或完成态 PR。
- 不把历史行数、字符数、字节数、压缩率或一次性迁移步骤保留为主 spec 的长期行为；如 Apply 需要采集证据，只放在 change evidence 中。

## Decisions

### 1. 采用职责分层而不是建立新的总规则文件

保留现有文件作为最小入口：`AGENTS.md` 只表达不可绕过硬门和路由；`openspec/config.yaml` 只表达稳定上下文与写作约束；`.agents/openspec-workflow.md` 维护生命周期、人工 gate、package、风险和有状态写；`.agents/git-workflow.md` 维护 branch/worktree、commit、push、PR、merge、cleanup；`.agents/testing-tdd.md` 维护 TDD、测试边界和验证选择；主 spec 只表达可观察、长期可验证行为。这样查询者可按动作加载一份详述来源，避免新增平行事实源。

备选方案是把全部规则合并到一个大文件，或保留现状只加交叉链接。前者破坏按任务路由和维护边界，后者不能消除重复，因此不采用。

### 2. 用统一术语和 Gate Map 表达阶段边界

`gate` 表示必须停顿并取得人工 Review/Authorization/Acceptance 的边界；`package` 表示同一风险边界内可连续完成并一次汇总证据的内聚交付单元；`checkpoint` 表示 package 或生命周期阶段的可审阅证据点；`commit` 表示 scoped Git 状态快照，不自动等同人工 gate。Proposal 与 tasks 使用相同 Gate Map 和 Complexity Budget，普通测试、修复、diff、commit、push 不单独制造 gate。

### 3. 用语义覆盖矩阵替代历史压缩指标

Apply 维护一张覆盖矩阵，将硬门映射到唯一详述来源、根级路由摘要、主 spec 长期 requirement、自动化检查和明确的非目标。矩阵至少覆盖 OpenSpec 顺序与人工 Review、Desktop-managed 入口、sequential/parallel 分流、两类 cleanup、有状态写授权、风险分级、TDD/CI/验证边界、事实源与安全边界。重复扫描和链接检查验证可维护性；不把本 change 的行数或压缩率写入长期 spec。

### 4. 采用范围匹配的 Proposal 验证

Proposal checkpoint 只验证 OpenSpec strict、显式 change 的 task-design lint、必要的 workflow architecture targeted checks、`git diff --check`、scope/secret/link 检查。由于本阶段不修改 backend runtime，`go test ./...` 不在范围内；Apply final 是否运行更广验证由最终受影响交付边界重新判定。

## Risks / Trade-offs

- [语义遗漏] 去重可能误删硬门 → 用覆盖矩阵、现有 architecture/workflow checks 和人工 Proposal Review fail-closed；矩阵缺项不得进入 Apply。
- [职责边界仍有交叉] 短摘要可能重复详述 → 每个操作语义只在一个 `.agents` 专责文件完整出现，其他位置只保留路由或不可绕过摘要。
- [主 spec 过度抽象] 长期 requirement 失去可测试性 → 每个保留 requirement 配套 WHEN/THEN scenario，并保留对规则文本/architecture 检查的可验证锚点。
- [规则与测试契约漂移] 既有 architecture test 仍依赖旧短语 → Apply 仅在确认测试契约需要时同步 targeted assertions，不能降低检查范围；Proposal 阶段记录该风险，不修改测试实现。

## Migration Plan

Apply 按“先建立覆盖矩阵与术语 → 调整四类职责来源 → 重写主 spec → 更新必要检查 → 范围验证”的顺序执行。所有改动均为仓库文本/测试契约变更，无数据库、图谱、部署或运行时迁移；若覆盖矩阵、OpenSpec strict 或 targeted checks 失败，停止并回到 artifact 修订，不进入 Sync。

## Open Questions

- Proposal Review 需要确认：主 spec 中保留的长期 requirements 是否覆盖所有硬门，以及哪些既有 architecture assertions 应继续作为自动化锚点。
