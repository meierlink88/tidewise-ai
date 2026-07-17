## Why

当前工程正从实验性的 AI 采集器演进为长期维护的多 Agent 工程，但尚未强制执行统一的规格说明、测试、隔离开发和评审流程。现在建立规范，可以避免 AI 采集器、AI 事件提取器和 AI 投研报告分析师在后续开发中出现流程与架构分化。

## What Changes

- 将仓库内置的 OpenSpec skills 和变更产物设为所有工程改动的必经流程。
- 在提案和实施前强制执行 `$eino-reference-first`，并保持 `.reference/cloudwego/` 只读且不进入版本控制。
- 要求每个 OpenSpec change 使用基于最新 `origin/main` 的独立 `codex/<change-name>` 分支和 worktree。
- 建立提案评审、完成前评审、TDD（`RED -> GREEN -> REFACTOR`）、验证、规格同步、归档和 PR 交付规则。
- 增加适用于三个规划 Agent 的技能路由和工程架构边界。
- 规定 change 产物除 OpenSpec 固定结构标记、规范关键字和不可翻译的技术标识外，均优先使用中文描述。

## Capabilities

### New Capabilities

- `engineering-change-workflow`：定义仓库改动必须遵循的 OpenSpec、Eino 审计、中文优先、TDD、worktree、验证和 PR 生命周期。

### Modified Capabilities

无。

## Impact

- 修改仓库 Agent 指令、OpenSpec 配置和 `.agents/` 工作流文档。
- 影响后续所有源代码、测试、配置、提示词、skill、脚本、依赖、文档和工程结构变更。
- 不改变 AI 采集器运行行为，也不增加生产依赖。
