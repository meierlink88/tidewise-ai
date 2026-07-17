## Context

仓库已经包含基于 Eino 的 AI 采集器、强制执行的 `eino-reference-first` skill，以及 OpenSpec 1.5.0 生成的五个 Codex skills。目前缺少仓库级技能路由和生命周期规则，尚不能保证这些能力在每次任务中被一致执行。兄弟项目 `tidewise-ai` 提供了经过实践的参考流程，但其中的领域边界和可选 skill 依赖不能原样复制。

该工作流需要覆盖三个规划中的任务 Agent：AI 采集器、AI 事件提取器和 AI 投研报告分析师。本地上游参考仓库、运行密钥和采集结果必须始终排除在 Git 历史之外。

## Goals / Non-Goals

**Goals:**

- 定义从探索到 PR 交付的统一强制生命周期。
- 将每个 change 隔离在 `codex/<change-name>` 分支和独立 worktree 中。
- 将 Eino 三仓审计和 OpenSpec skills 明确为技能路由的强制步骤。
- 要求提供可执行的 TDD 证据，并设置两个人工评审门禁。
- 记录三个规划 Agent 共用的架构边界。
- 除 OpenSpec 固定结构和必要技术标识外，change 产物统一优先使用中文。
- 增加轻量级自动化策略测试，防止核心工作流规则被静默删除。

**Non-Goals:**

- 修改 AI 采集器运行行为。
- 在本 change 中实现 AI 事件提取器或 AI 投研报告分析师。
- 自动执行 GitHub 合并、部署或发布。
- 提交 `.reference/cloudwego/`、`.env` 或运行产物。

## Decisions

### 以仓库指令作为工作流入口

`AGENTS.md` 保持精简，并将 Agent 路由到 `.agents/` 下的专项规则文件。该方式与 `tidewise-ai` 保持一致，同时能突出本工程的 Eino-first 约束。未采用单个超长指令文件，因为生命周期和架构规则的演进频率不同。

### OpenSpec 能力保留在仓库内

`.codex/skills/` 下生成的 OpenSpec skills 和 `openspec/config.yaml` 继续纳入版本控制。Agent 必须根据任务意图使用 `openspec-explore`、`openspec-propose`、`openspec-apply-change`、`openspec-sync-specs` 和 `openspec-archive-change`。不依赖机器全局安装的 skill 内容，从而保证新环境克隆仓库后能够复现工作流。

### change 产物采用中文优先策略

在 `openspec/config.yaml` 和工作流指令中明确：OpenSpec 固定标题、`Requirement`、`Scenario`、`WHEN/THEN`、`SHALL/MUST`、命令、路径、代码标识符和 capability 标识保持规范要求或原始形式，其余由 change 生成的提案、设计、规格和任务描述均使用中文。该策略兼顾 OpenSpec 解析稳定性与团队阅读效率。

### worktree 创建由 Agent 执行并验证

开始新 change 时，Agent 必须获取 `origin/main`，创建或选用 `codex/<change-name>` 分支，并在写入提案或实现文件前进入独立 worktree。优先使用 Codex Desktop 管理的 worktree，`git worktree` 作为明确记录的后备方式。工作流必须记录分支、worktree 和基线状态的验证命令。未引入自定义守护进程或 Git 包装器，以避免不必要的运维复杂度。

### 通过任务结构和策略测试约束 TDD

所有行为变更任务必须声明 RED、GREEN 和 REFACTOR/验证证据。新增 Go 架构策略测试，检查活跃 OpenSpec tasks 是否包含规定的任务结构，并检查工作流文档中的关键约束。这不能完全证明开发过程，但能在不增加运行时依赖的前提下提供快速、可执行的反馈。

### 人工评审是不可跳过的门禁

实施前必须评审 proposal、必要的 design、capability specs 和 tasks；实施完成后必须再次评审，之后才能同步规格、归档并完成 PR 交付。Agent 可以准备材料并报告就绪状态，但不能自行批准评审门禁。该规则延续 `tidewise-ai` 的治理方式。

### PR 是工程交付的终点

测试、OpenSpec 验证、规格同步和归档完成后，Agent 必须有选择地提交文件、推送 change 分支并创建 PR。合并操作始终由用户控制。可以提前创建 Draft PR 便于查看，但 Draft PR 不代表 change 已完成。

## Risks / Trade-offs

- **小改动也存在流程成本** → 探索阶段可以保持轻量，但任何仓库写入仍应进入命名 change，除非符合明确记录的紧急例外。
- **基于指令的 worktree 自动化可能被绕过** → 规定分支/worktree 验证命令，并通过自动化策略测试保护长期规则。
- **人工门禁会中断全自动执行** → Agent 必须明确报告需要批准的内容，并在批准后从原 worktree 继续。
- **策略测试可能过度依赖 Markdown 文案** → 只测试稳定标题、规范关键字和关键约束，不比较完整段落。
- **不同环境的 OpenSpec CLI 版本可能不同** → 在仓库中保存生成的 skills 和配置，并将 OpenSpec 1.5.x 记录为开发前置条件。
- **中英文混排边界可能不一致** → 在配置中列出必须保留原文的结构和技术标识，其余默认中文。

## Migration Plan

1. 增加技能路由、OpenSpec 生命周期、Git/worktree、TDD 和架构边界文档。
2. 更新 `AGENTS.md` 和 OpenSpec 工程上下文，引用上述规则并加入中文优先约束。
3. 先以 RED 阶段加入策略测试，再补充配置使测试进入 GREEN。
4. 执行 Go 测试和严格 OpenSpec 验证。
5. 完成评审后同步 capability spec、归档 change，并通过 PR 交付。

如需回滚，直接回退本 change 即可；AI 采集器运行行为不受影响。

## Open Questions

当前没有未决问题。CI 强制检查和陈旧 worktree 自动清理可以在本地工作流经过实践后另行提出 change。
