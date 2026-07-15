# AGENTS.md

## Project

观潮家是全球政经事件驱动的市场理解与决策辅助产品。所有投研与 AI 输出只能表达市场理解和决策辅助，不得表达为直接投资建议。

## Workspace Boundary

Codex Desktop 应以本 `tidewise-ai/` 目录作为源码和 OpenSpec 根目录。工程代码、OpenSpec artifacts、自动化脚本和项目规则只进入本目录；`doc/` 与 `prototype/` 默认只读，prototype 不是生产源码，不得复制其中的 HTML、DOM 操作或内联脚本。

## Rule Priority And Routing

冲突优先级：用户当前明确指令 > 本文件与 `.agents` > 已批准 OpenSpec artifacts > 已安装 Skills > Agent 临时判断。

新任务先读本文件，再按动作读取专责规则；不得无差别读取全部规则或主规格。

| 任务 | 唯一详述来源 |
|---|---|
| 正式研发与 Skill 选择 | `.agents/skill-routing.md` |
| OpenSpec 生命周期、artifact、审批和风险 | `.agents/openspec-workflow.md` |
| branch、worktree、commit、push、PR、merge、cleanup | `.agents/git-workflow.md` |
| Go 实现、重构、测试与验证 | `.agents/testing-tdd.md` |
| 后端、前端或部署领域边界 | 对应 `.agents/*-boundaries.md` |

## OpenSpec Hard Gates

OpenSpec 是正式工程 change 的唯一生命周期和 artifacts 来源：

```text
Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver
```

- 阶段不得跳过或调换；完整定义唯一位于 `.agents/openspec-workflow.md`。
- Proposal Review 未通过不得 Apply；Apply-final Review 未通过不得 Sync、Archive 或 Deliver。
- 数据库 migration/apply、seed、业务写入、图谱关系写入、投影重建或清理必须按层展示范围、顺序、recovery、断言和停止条件，并取得明确授权。
- 正式 change 声明 R0—R3 风险；普通 task checkbox 不自动成为人工 gate；Gate Map、Complexity Budget、package、条件式执行包和 task-design lint 服从 `.agents/openspec-workflow.md`。

## Git And Desktop Hard Gates

- 新 change 必须在 Desktop-managed worktree 中从最新 `origin/main` 创建或切换 `codex/<change-name>`；不得手工创建等价 worktree。
- 仅在 Desktop 机制不可用且用户明确批准 fallback 时，才允许 project-owned worktree。
- Desktop-managed worktree 只能由 Desktop 释放；agent 不得手工删除托管 worktree。
- PR merge 后按 worktree 所有权执行 branch、Desktop 任务、worktree 释放验证和本地 branch cleanup；完整顺序唯一位于 `.agents/git-workflow.md`。

## Architecture And Safety Invariants

- 前后端分离；后端为 `backend/` 下单 Go module；PostgreSQL 是事实源，Neo4j 是可重建投影，Redis 只承担缓存、限流、幂等和短期任务状态。
- 正式 API 先定义 DTO、错误、分页、时间、ID、枚举和 Agent 回写契约；敏感配置只能由环境变量或部署 secret 注入。
- Go 后端功能、bugfix 和重构采用 TDD；测试边界和验证深度唯一服从 `.agents/testing-tdd.md`。
- 不修改无关 change，不写入或打印 secret、token、连接串、模型 key 或个人隐私；不执行未授权的数据库、图谱或部署操作。

## Completion Discipline

声明完成、commit、push、PR、sync 或 archive 前必须运行新鲜验证并读取结果；验证受限时必须报告未验证项和风险。
