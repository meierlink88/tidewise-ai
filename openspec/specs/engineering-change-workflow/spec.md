# Engineering Change Workflow Specification

## Purpose

定义 `tidewise-ai-agentrun` 的正式工程变更如何通过 Eino 三仓审计、OpenSpec、中文优先、TDD、Git worktree、人工评审和 Pull Request 完成可追溯交付。

## Requirements

### Requirement: Eino-first 与 OpenSpec 技能路由
每次仓库写入 SHALL 先完成与任务相关的 `eino-reference-first` 审计，并 SHALL 在探索、提案、实施、规格同步和归档阶段使用对应的仓库内置 OpenSpec skill。

#### Scenario: Agent 收到工程变更请求
- **WHEN** Agent 被要求修改源代码、测试、配置、文档、提示词、skill、脚本、依赖、生成物或工程结构
- **THEN** Agent 在编辑前完成 Eino 三仓审计，并将请求路由到一个命名的 OpenSpec change

### Requirement: 独立的 change 工作空间
每个新 OpenSpec change SHALL 在写入 change 产物或实现文件前，使用基于最新 `origin/main` 创建的独立 `codex/<change-name>` 分支和 worktree。

#### Scenario: 开始新的 change
- **WHEN** 当前不存在与请求范围匹配的活跃 change 和 worktree
- **THEN** Agent 获取 `origin/main`、创建命名分支和独立 worktree，并在写入文件前验证两者

#### Scenario: 恢复已有 change
- **WHEN** 已有活跃 change 具备匹配的分支和 worktree
- **THEN** Agent 继续使用已有工作空间，不创建重复 change

### Requirement: change 产物中文优先
所有 change 产物 SHALL 优先使用中文描述；仅 OpenSpec 固定结构标记、规范关键字、命令、路径、代码标识符、capability 标识及其他不可翻译技术内容保留原始形式。

#### Scenario: 生成 OpenSpec change 产物
- **WHEN** Agent 编写 proposal、design、specs 或 tasks
- **THEN** 固定结构标记和必要技术标识保持规范形式，其余标题与正文使用中文

### Requirement: 经过评审的提案包
proposal、必要的 design、capability specs 和具备 TDD 结构的 tasks SHALL 在实施开始前全部完成，并由用户明确评审通过。

#### Scenario: 提案产物准备完成
- **WHEN** `openspec status` 显示所有 apply 前置产物均已完成
- **THEN** Agent 汇总范围、风险和任务，并等待用户明确批准后再实施

### Requirement: 测试驱动实施
所有行为变更 SHALL 遵循 RED、GREEN 和 REFACTOR；change 的 tasks 文件 SHALL 记录相关失败测试、最小实现和验证命令。

#### Scenario: 实现一项行为需求
- **WHEN** 开始执行实现任务
- **THEN** 相关自动化测试先因预期原因失败，在最小实现后通过，并在重构后通过受影响测试集

### Requirement: 完成前评审与验证
已实现的 change SHALL 通过工程测试、策略检查和严格 OpenSpec 验证，并 SHALL 在同步规格和归档前获得用户明确的完成评审批准。

#### Scenario: 实现任务看似全部完成
- **WHEN** 所有实现任务均被标记为完成
- **THEN** Agent 报告验证证据，并等待用户明确批准后再执行 sync 和 archive

### Requirement: 通过 Pull Request 交付
每个完成的 change SHALL 有选择地提交文件、推送至 change 分支，并在 Agent 报告完成前通过 GitHub Pull Request 交付；合并操作 SHALL 由用户控制。

#### Scenario: change 获准交付
- **WHEN** 完成前评审、规格同步、归档和最终验证均成功
- **THEN** Agent 推送分支、创建或更新 Pull Request，并报告其地址和验证证据

### Requirement: 本地材料不得发布
工作流 SHALL 将密钥、运行产物、操作系统文件和 `.reference/cloudwego/` 排除在提交及 Pull Request 之外。

#### Scenario: Agent 准备提交
- **WHEN** Agent 为基线提交或 change 提交暂存文件
- **THEN** Agent 验证忽略项，并扫描暂存差异中可能存在的凭据内容后再提交
