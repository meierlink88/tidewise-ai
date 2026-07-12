## Why

当前根 `AGENTS.md` 同时承担项目总纲、完整生命周期、Git 流程、前后端细节和上下文清单，已达到 218 行、11,308 字节，并与 `.agents/skill-routing.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md` 重复。重复规则增加每次任务的无差别读取成本，也使同一硬门在多处演化时产生冲突或遗漏风险，因此需要在不削弱工程约束的前提下建立清晰的规则分层与单一事实来源。

## What Changes

- 将根 `AGENTS.md` 定位为约 90–110 行、5–6 KB 的稳定入口，只保留项目总纲、目录边界、规则优先级与按任务路由、生命周期硬门、架构不变量和通用安全规则。
- 将 OpenSpec 生命周期的完整阶段说明统一归属 `.agents/openspec-workflow.md`，Skill 映射统一归属 `.agents/skill-routing.md`，分支、worktree、commit、PR、merge 与交付清理统一归属 `.agents/git-workflow.md`。
- 删除根文件中“每次读取全部规则”的 `Useful Context Files` 模式，改为先读根文件，再按任务类型读取对应 `.agents` 规则、change artifacts、相关主规格与代码。
- 去除跨文件重复说明和无约束价值的技术背景，但通过规则覆盖矩阵确保关键硬门均保留且只有一个详述来源。
- 在 Apply 阶段以行数、字节数、重复/冲突扫描、链接有效性和规则覆盖矩阵验证精简没有造成规则丢失。
- 本 change 的 proposal checkpoint 只新增 `openspec/changes/streamline-agent-rules/` artifacts；不修改 `AGENTS.md`、`.agents/*.md`、源码、`prototype` 或 `doc`。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `skill-driven-development-workflow`：增加分层规则所有权、按任务读取、关键硬门覆盖与可验证精简要求，避免根规则和细分规则互相复制。

## Impact

- Apply 预计修改 `AGENTS.md`、`.agents/skill-routing.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`，必要时仅为消除重复而小幅调整其他 `.agents/*.md` 的交叉引用。
- 不改变 OpenSpec artifact 路径、Skill 触发条件、Git 分支命名、数据库或运行时行为，不新增依赖。
- 不触碰 `add-market-sector-foundation`、`add-ai-event-extraction-pipeline` 或任何其他 active change/worktree。
- `prototype` 仅保留为不可复制的参考边界，`doc` 不更新。
