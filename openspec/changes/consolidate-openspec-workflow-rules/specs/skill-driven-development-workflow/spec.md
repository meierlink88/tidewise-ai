## MODIFIED Requirements

### Requirement: Agent 规则必须采用分层单一事实来源

系统 MUST 将规则职责分层：`AGENTS.md` 只提供最高级硬门与路由；`openspec/config.yaml` 只提供稳定项目背景、语言和 artifact 写作约束；`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md` 分别提供 OpenSpec 生命周期/审批、Git 交付、测试/验证的唯一完整详述；主 workflow spec 只保留长期可验证行为。其他文件可以保留不可绕过摘要或链接，但不得复制另一职责来源的完整操作流程。

#### Scenario: 查找 OpenSpec 生命周期规则

- **WHEN** Agent 需要确定阶段顺序、人工 Review、风险 gate 或有状态操作授权
- **THEN** Agent 可以从 `AGENTS.md` 路由到 `.agents/openspec-workflow.md`，并以该文件作为唯一完整操作来源

#### Scenario: 查找 Git 交付规则

- **WHEN** Agent 需要创建 branch/worktree、提交、推送、合并或清理 change
- **THEN** Agent 可以从 `AGENTS.md` 路由到 `.agents/git-workflow.md`，并以该文件作为唯一完整操作来源

#### Scenario: 查找测试与验证规则

- **WHEN** Agent 需要选择 TDD、targeted tests、architecture checks 或完整验证范围
- **THEN** Agent 可以从任务路由进入 `.agents/testing-tdd.md`，并以该文件作为测试/验证唯一完整操作来源

#### Scenario: 查找项目上下文与 artifact 写作约束

- **WHEN** OpenSpec CLI 生成 artifact instructions 或 Agent 需要稳定项目背景
- **THEN** `openspec/config.yaml` 提供上下文与写作约束，不承载生命周期、Git 或测试操作详述

### Requirement: 规则精简必须保留关键工程硬门

系统 MUST 在分层规则与主 workflow spec 中继续明确约束 OpenSpec 唯一生命周期、Codex Desktop 受管任务/worktree 强制入口、change branch/worktree 隔离、人工 Review 与有状态操作审批、Sync/Archive/Deliver 顺序、按 worktree 所有权交付清理、数据事实源和通用安全边界；去重不得降低任何一项门禁。

#### Scenario: Proposal 未获人工确认

- **WHEN** change 的 proposal artifacts 尚未获得人工确认
- **THEN** Agent 不得进入 Apply，也不得以自动化或 Skill 默认流程替代确认

#### Scenario: 执行有状态操作

- **WHEN** Agent 准备写入数据库、图谱关系或重建图投影
- **THEN** Agent 必须按层展示范围、顺序、recovery、断言和停止条件并取得明确授权

#### Scenario: Desktop-managed change 完成交付清理

- **WHEN** PR 已合并且默认分支已验证包含最终 archive commit
- **THEN** Agent 必须按 worktree 所有权执行远端 branch、Desktop 任务、托管 worktree 释放验证和本地 branch 的规定顺序

### Requirement: 研发工作流术语与阶段证据必须统一

系统 MUST 统一 `gate`、`package`、`checkpoint` 和 `commit` 的定义：gate 是人工 Review/Authorization/Acceptance 边界，package 是同一风险边界内可连续完成的内聚交付单元，checkpoint 是可审阅证据点，commit 是 scoped Git 状态快照。普通 task、测试、修复、diff、commit 和 push 不得单独制造人工 gate；Proposal 与 tasks 的 Gate Map、Complexity Budget 和 package 映射 MUST 一致。

#### Scenario: 普通任务在 package 内完成

- **WHEN** Agent 完成不跨越授权边界的实现、测试、修复、diff 或 commit
- **THEN** 证据归入所属 package，不因 checkbox 或单次操作新增人工 gate

#### Scenario: Gate Map 与 tasks 不一致

- **WHEN** Proposal 与 tasks 的 package、gate、risk、human、reason code 或 allowed scope 任一映射不一致
- **THEN** task-design lint 或 Proposal 自检必须失败并阻止进入下一阶段

#### Scenario: 阶段 checkpoint 交付审阅

- **WHEN** package 达到其验证条件并形成 scoped commit
- **THEN** checkpoint 必须记录 scope、风险、证据、未验证项、阻断项和下一步授权边界

### Requirement: 长期工作流行为必须可验证

系统 MUST 通过 OpenSpec strict、规则/architecture targeted checks、精确 task-design lint、scope/secret/link 检查和语义覆盖矩阵验证规则分层与硬门完整性。主 workflow spec 不得把某个 change 的历史行数、字符数、字节数、压缩率、一次性迁移步骤或旧验收指标作为长期行为要求。

#### Scenario: Proposal checkpoint 验证范围匹配

- **WHEN** change 只涉及 OpenSpec artifacts、workflow 文本、agent rules 或 workflow architecture assertions
- **THEN** Proposal 阶段运行范围匹配检查，不机械运行 `go test ./...`，并记录未授权的 Apply/Sync/Archive/Deliver 操作

#### Scenario: 硬门覆盖矩阵完整

- **WHEN** Agent 准备请求 Proposal Review 或 Apply 后 Review
- **THEN** 覆盖矩阵必须映射 OpenSpec 顺序、人工 Review、Desktop 入口、sequential/parallel 分流、两类 cleanup、风险/有状态写、TDD/CI/验证、事实源和安全边界，缺项即 fail-closed

#### Scenario: 主 spec 保持长期行为

- **WHEN** change 已归档且规则进入后续研发
- **THEN** 主 workflow spec 仍能用稳定 requirement/scenario 和自动化检查验证行为，不依赖本 change 的历史叙述或一次性数字

## REMOVED Requirements

### Requirement: 规则精简必须可量化验证

**Reason**: 历史行数、字符数、字节数和压缩率属于本次规则重构的阶段证据，不是长期 workflow 行为；保留为主 spec requirement 会把一次性迁移指标固化为未来门禁。

**Migration**: Apply 阶段如需审阅精简效果，将前后统计、重复/冲突扫描和覆盖矩阵保存于当前 change 的 Review package/evidence；长期行为改由“长期工作流行为必须可验证”约束。
