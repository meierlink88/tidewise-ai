# AGENTS.md

## Project

观潮家是全球政经事件驱动的市场理解与决策辅助产品，覆盖事件采集、知识图谱、RAG、Agent 推理、板块/资产传导、报告和订阅。

所有投研与 AI 输出必须定位为市场理解和决策辅助，不得表达为直接投资建议。

## Workspace Boundary

Codex Desktop 应以本 `tidewise-ai/` 目录作为源码和 OpenSpec 根目录。

```text
观潮家/
├── tidewise-ai/ # 源码、OpenSpec、工程配置和 agent 规则
├── doc/         # 长期产品、架构、商业和数据模型文档
└── prototype/   # 高保真原型和设计参考
```

- 工程代码、OpenSpec artifacts、自动化脚本和项目规则只进入 `tidewise-ai/`。
- `doc/` 与 `prototype/` 默认只读；除非用户明确要求，不得修改。
- `prototype/` 不是生产源码，不得直接复制其中的 HTML、DOM 操作或内联脚本。

## Rule Priority And Routing

冲突优先级：用户当前明确指令 > `AGENTS.md` 与 `.agents` > 已批准 OpenSpec artifacts > 已安装 Skills > Agent 临时判断。

新任务先读本文件，再按实际动作读取命中的规则；不得无差别读取全部规则、全部主规格或项目外文档。

| 任务 | 必须读取 |
|---|---|
| 正式研发、Skill 选择 | `.agents/skill-routing.md` |
| OpenSpec 生命周期、artifact、审批、sync/archive | `.agents/openspec-workflow.md` |
| branch、worktree、commit、push、PR、merge、cleanup | `.agents/git-workflow.md` |
| Go 后端、API、采集、数据库、integration、部署 | `.agents/backend-boundaries.md` |
| Go 实现、bugfix、重构、测试与验证 | `.agents/testing-tdd.md` |
| 小程序、管理后台、设计系统、前端组件 | `.agents/frontend-boundaries.md` |

继续已有 change 时读取其 proposal、design、tasks、delta specs、受影响主规格、相关代码及命中的规则；只读解释或查询不机械创建 change。

## OpenSpec Hard Gates

OpenSpec 是正式工程 change 的唯一生命周期和 artifacts 来源：

```text
Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver
```

- 阶段不得跳过或调换；完整定义唯一位于 `.agents/openspec-workflow.md`。
- Propose 后必须由用户人工 Review；未获明确批准不得进入 Apply。
- Apply 完成后必须提交 scoped diff 和验证证据，再次等待人工 Review；批准前不得 Sync、Archive 或 Deliver。
- 数据库 migration/apply、seed、业务写入、图谱关系写入、投影重建或清理必须先展示范围和顺序并取得明确批准。
- 分层数据与图谱操作按 `Review -> Write -> Rebuild -> Query` 逐层审批；只读或上一层批准不得推定下一层写入授权。
- Sync、Archive、Deliver 的完成条件及顺序不得削弱；Archive 不等于 change 已交付或关闭。

## Risk-Tiered Development Summary

- 正式 change 必须声明 R0—R3 风险：R0 文档/调研/只读审计，R1 无有状态写入的源码/测试，R2 migration、seed 或 local/UAT 数据变更，R3 生产、不可逆 cleanup、Neo4j rebuild 或敏感部署；具体操作可上调等级。
- 普通 task checkbox 不自动成为人工 gate；详细的风险理由、阶段 Review package、候选审阅、条件式执行包、R2 recovery evidence、R3 独立授权和 active adoption 只在 `.agents/openspec-workflow.md` 维护。
- 新 change 的一级 task 必须表达内聚 package，并使用固定 Gate Map、Complexity Budget 与 task-design lint；机器 schema、legacy baseline 和 explicit lint 接口只在 `.agents/openspec-workflow.md` 维护。
- R2/R3 的有状态操作必须显式授权且 fail-closed；旧批准、普通 Apply 批准或上一层结果不得推定下一层、环境或范围。R3 不得跨层批量执行。
- Apply final 按受影响交付边界运行完整验证和共享 tests；共享规则、跨模块契约、公共基础设施或 repo-wide 变更才运行 repo-wide full validation。

## Git And Desktop Hard Gates

- 新 change 默认按顺序推进；仅用户明确批准、无依赖且无共享文件或数据库写状态的独立并行 change 可同时启动，出现依赖或共享写状态必须暂停并重新排序。
- Desktop 可用时，所有新 change 必须由 Desktop 新任务创建独立受管 worktree，并在其中从最新 `origin/main` 创建或切换 `codex/<change-name>`；不得手工执行 `git worktree add` 或混入其他 change。
- 只有 Desktop 受管机制不可用且用户明确批准 fallback 时，才允许按 `.agents/git-workflow.md` 创建项目自有 worktree。
- Desktop-managed worktree 只能由 Desktop 释放；agent 不得对托管目录执行 `rm` 或 `git worktree remove`。
- PR merge 后必须按 worktree 所有权有序清理远端 branch、worktree/任务和本地 branch；Desktop 未释放时记录待清理状态，不得宣称 cleanup 或 Deliver 完成。
- 两类 cleanup 的完整唯一顺序位于 `.agents/git-workflow.md`。

## Architecture Invariants

- 前端与后端分离；小程序为 `frontend/miniapp/`，采用 Taro + React + TypeScript，面向微信和抖音。
- 管理后台为 `frontend/admin/`，采用 Vite + React + TypeScript；默认设计系统是 repo-local Minimal Dashboard，不新增 Ant Design 依赖。
- 后端为 `backend/` 下单 Go module、模块化单体；`cmd/*` 是进程入口，`internal/apps/*` 是业务子系统。
- PostgreSQL 是实体、事件和关系事实源；Neo4j 仅保存可从 PostgreSQL 重建的图投影。
- Redis 只承担缓存、限流、幂等和短期任务状态；AI/RAG/Prompt 编排优先由外部 Agent 平台承载。
- 正式 API 必须先定义 DTO、错误、分页、时间、ID、枚举和 Agent 回写契约。
- local、uat、prod 使用统一强类型配置；敏感配置只能由环境变量或部署 secret 注入。
- 后端功能、bugfix 和重构默认 TDD；具体测试标准不得从 `.agents/testing-tdd.md` 降级。

## Engineering And Safety

- 基于主规格和现有代码增量迭代；先检查复用点，不得新建平行页面、service、model、store、data、config 或工程骨架。
- 不得把一个 change 的修改、artifact、数据库状态或未提交文件混入另一个 change。
- 小程序只负责展示、交互、轻状态和 API 调用，不得包含服务端执行、数据库访问或模型/支付密钥。
- 前端使用 React/Taro 数据驱动方式；不得使用 `document`、`window`、`innerHTML`、内联 `onclick` 或直接 DOM mutation。
- mock 数据放 dedicated data modules，API 调用经过 services/request 边界。
- 不在源码添加注释，除非用户明确要求。
- 不得提交、写入或打印 secret、token、数据库连接串、模型 API key、支付密钥或个人隐私信息。
- 不 revert 其他用户或 agent 的无关改动，不使用破坏性 Git 命令，除非用户明确授权。

## Completion Discipline

- 声明完成、commit、push、PR、sync 或 archive 前必须运行新鲜验证并读取结果。
- 验证受环境限制时必须报告未验证项和风险，不得用推测替代证据。
