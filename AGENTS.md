# AGENTS.md

## Project Overview

观潮家是一个全球政经事件驱动的市场理解与决策辅助产品。系统目标是通过事件采集、知识图谱、RAG、Agent 推理、板块/资产传导分析、报告生成和订阅能力，帮助用户理解宏观事件、产业事件、市场指标、板块和企业之间的关系。

产品输出定位为决策辅助与市场理解，不应表达为直接投资建议。

## Workspace Boundary

当前 Codex Desktop 工作区应打开本目录：

```text
/Users/meierlink/Documents/david/创业项目/观潮家/tidewise-ai
```

本目录是源码工程根目录，也是 OpenSpec 根目录。所有工程代码、OpenSpec 变更、主规格、Codex OpenSpec skills、项目级 agent 规则都应以本目录为工作根。

上级目录结构用途如下：

```text
观潮家/
├── tidewise-ai/ # 源码工程根目录，OpenSpec 根目录，Codex Desktop 工作区根目录
├── doc/        # 项目文档空间，放商业计划、产品设计、架构文档、数据模型等
└── prototype/  # 原型设计目录，放高保真原型和设计参考
```

目录使用规则：

- `tidewise-ai`：只放工程源码、OpenSpec artifacts、工程配置、自动化脚本和 agent 规则。
- `doc`：只放长期项目文档、商业资料、产品资料、架构资料和数据模型资料。
- `prototype`：只放原型图、高保真设计和交互参考，不作为生产源码。
- 不得把 `prototype` 中的 HTML、DOM 操作或内联脚本直接搬进生产代码。
- 除非用户明确要求，不要修改 `doc` 和 `prototype`。

## OpenSpec Methodology

本项目严格按照 OpenSpec 方法论执行工程开发。正式工程变更必须先创建 OpenSpec change，再实现代码。

OpenSpec 细则见 `.agents/openspec-workflow.md`。处理任何 OpenSpec change 前必须读取该文件。

## Agent Rule Index

`AGENTS.md` 保留项目总纲、硬规则和规则路由。细分工程规则放在 `.agents/` 目录，agent 必须按任务类型读取对应文件。

当前细分规则：

- 处理 OpenSpec change 生命周期、artifact、sync 或 archive 时，必须读取 `.agents/openspec-workflow.md`。
- 处理 branch、worktree、commit、push 或 PR 时，必须读取 `.agents/git-workflow.md`。
- 处理 Go 后端、API、采集器、数据库、integration 或部署边界时，必须读取 `.agents/backend-boundaries.md`。
- 处理 Go 后端实现、bugfix、重构或验证时，必须读取 `.agents/testing-tdd.md`。

后续可以继续拆分：

- `.agents/frontend-boundaries.md`：前端、小程序、设计稿转实现边界。
- `.agents/design-to-frontend.md`：设计系统 + HTML 设计稿 -> 前端实现流程。

## Superpowers Integration

本项目可以使用 Superpowers plugin 作为工程执行辅助机制。OpenSpec 是 change 生命周期、系统事实和正式 artifacts 的来源，Superpowers 只用于补强具体执行方法。

固定使用原则：

- OpenSpec 负责 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive`。
- Superpowers 负责需求澄清、TDD、系统化调试、完成前验证、代码审查、分支收尾等执行纪律。
- GitHub plugin 负责 GitHub 侧的 PR、CI、review comments、push 和协作发布动作。
- 如果 Superpowers 默认规则与 `AGENTS.md`、OpenSpec artifacts 或项目 Git 规则冲突，必须以 `AGENTS.md` 和 OpenSpec 为准。
- Superpowers 不应默认产生独立于 OpenSpec 的长期 artifacts；正式 change artifacts 仍只以 `openspec/changes/<change-name>/` 和 `openspec/specs/` 为准。
- 只有当任务特别复杂且用户明确同意时，才允许额外创建 `docs/superpowers/plans/` 下的计划文件。

推荐节点映射：

- Explore 阶段可以使用 `superpowers:brainstorming` 辅助澄清问题、范围和取舍。
- Apply 阶段涉及后端功能、bugfix、重构或行为变更时，必须使用 `superpowers:test-driven-development` 辅助测试先行。
- 遇到缺陷、测试失败或行为异常时，必须使用 `superpowers:systematic-debugging` 辅助排查，不得直接猜测式修改。
- 在声明任务完成、提交、push、创建 PR 或进入 archive 前，必须使用 `superpowers:verification-before-completion` 辅助完成新鲜验证。
- 完成主要功能、准备合并或用户要求 review 时，可以使用 `superpowers:requesting-code-review` 辅助审查。
- 需要并行 change、长任务隔离或多个 Codex 线程协作时，可以使用 `superpowers:using-git-worktrees` 辅助判断是否创建额外 worktree，但 worktree 命名和 OpenSpec change 边界仍遵守本文件规则。

## Iteration Rules

新 change 必须基于主规格和现有代码增量设计，优先复用已有模块，不得创建平行结构。详细 Explore、Propose、Apply、Validate、Sync、Archive 规则见 `.agents/openspec-workflow.md`。

复杂后端 change 的 `design.md` 必须包含 Mermaid sequence diagram 和 class/component diagram，具体规则见 `.agents/openspec-workflow.md`。

## Git Branch, Worktree, and Commit Workflow

OpenSpec change 是本项目的正式工作单元。Git branch 是 change 的交付边界，commit 是阶段性检查点，worktree 只用于并行隔离。详细规则见 `.agents/git-workflow.md`。

## Anti-Duplication Rules

AI agent 必须在已有实现基础上增量迭代，不得另起一套。

禁止行为：

- 不得重复创建已有页面目录。
- 不得重复创建已有 service、model、store、data 或 config 层。
- 不得绕过已有 service 直接在页面中硬编码数据访问逻辑。
- 不得在未检查现有代码前生成新的平行工程结构。
- 不得因为新 change 而重建已有工程骨架。

如果现有实现不符合新设计，应先更新 design 和 tasks，再执行迁移或重构。

## Testing And TDD Workflow

本项目后端研发默认采用 TDD 测试先行。涉及 Go 后端实现、bugfix、重构或验证时，必须读取 `.agents/testing-tdd.md`。

## Engineering Architecture

项目采用前后端分离架构。

前端：

- MVP 首端为跨平台小程序，首批目标至少包含微信小程序和抖音小程序。
- 前端采用 Taro + React + TypeScript 技术栈。
- 前端源码位于 `frontend/miniapp/`。
- 前端通过各小程序平台分别发布。
- 小程序端只负责展示、交互、轻状态和 API 调用。
- 小程序端不得保存模型密钥、Agent 平台密钥、支付密钥、数据库连接或后端凭证。

后端：

- 后端采用 Go 语言技术栈，优先使用 Go + Gin 构建高并发 API/BFF 服务。
- 后端技术选型同时要求适合 AI 编程协作：语言和框架应具备强类型、清晰工程约定、直接编译反馈、易测试和低歧义代码风格。
- 后端源码位于 `backend/`。
- API/BFF、领域模块、Agent 平台集成、数据访问、订阅推送、支付回调等能力作为服务端应用独立部署。
- MVP 阶段结构化主存储采用 PostgreSQL，缓存、限流、幂等和短期任务状态采用 Redis。
- MVP 阶段暂不直接引入独立图数据库或向量数据库；初始图谱关系优先通过 PostgreSQL 结构化关系表达，RAG、向量召回和 Prompt 编排优先由外部 Agent 平台承载。
- 正式业务 API 开发前必须先定义 API 契约边界，覆盖请求响应 DTO、错误结构、分页、时间、ID、枚举和 Agent 回写 payload。
- 数据采集层属于后端 ingestion 边界，支持自研爬虫脚本采集和外部 Agent API 采集结果接入两种路径。采集结果必须经过来源追踪、去重、清洗、标准化、质量标记和结构化校验后再进入存储边界。
- AI 推理、RAG、Agent 工作流编排和 Prompt 编排主要由外部 Agent 平台承载，本工程只负责 API 调用、回调接收、结构化结果校验、落库和展示。
- MVP 阶段后端采用模块化单体，后续再按容量、部署和团队边界拆分为独立服务。

部署：

- frontend 和 backend 分离部署。
- frontend 分别发布到微信、抖音等小程序平台。
- backend API/BFF 和异步任务能力部署到服务端环境。

## Frontend Direction

MVP 阶段前端采用：

```text
Taro + React + TypeScript
```

选择 Taro 是因为项目需要同时支持微信和抖音小程序，并为后续 H5 或更多小程序平台保留空间。Taro 官方支持微信、抖音等多个小程序平台，且支持 React/Vue 等现代前端开发体验；本项目默认采用 React 技术路线。

跨平台小程序源码位于：

```text
frontend/miniapp/
```

如果某个既有 OpenSpec artifact 中仍出现 `apps/miniprogram/`、`miniprogram/` 或原生小程序结构，应在实现前调整为 `frontend/miniapp/` 和 Taro 工程结构。

## Backend Direction

后端采用单 Go module、多可部署子系统结构。可部署进程放在 `backend/cmd/*`，业务子系统应用逻辑放在 `backend/internal/apps/*`，共享基础层放在 `backend/internal/domain`、`backend/internal/repositories`、`backend/internal/config`、`backend/internal/integrations` 和 `backend/internal/platform`。

处理任何后端 change 前，必须读取 `.agents/backend-boundaries.md`，并按其中的子系统归属、integration 边界、connector 归属和依赖方向执行。

不要在小程序 change 中实现真实后端能力。后端工程、API 契约、数据库、采集层、Agent 平台集成和部署拓扑应通过独立 OpenSpec change 设计和实现。

后端必须标准化支持 local、uat、prod 三类环境配置。配置文件可以保存非敏感结构化配置，数据库密码、Agent 平台 API key、支付密钥、JWT secret 等敏感信息必须通过环境变量或部署平台 secret 注入，不得提交到 repo。业务代码只能依赖统一的 Go 强类型 config，不得散落读取环境变量或硬编码环境差异。

## Code Style

- 不要在源码中添加注释，除非用户明确要求。
- 前端优先使用 TypeScript 类型表达数据结构和契约。
- 后端优先使用 Go 类型表达服务端领域模型、请求响应结构和数据访问结构。
- 前端页面逻辑应采用 Taro/React 数据驱动方式实现，使用组件 props、state/hooks、条件渲染和列表渲染表达交互状态。
- 禁止在生产小程序代码中使用 `document`、`window`、`innerHTML`、内联 `onclick` 或浏览器 DOM 操作。
- mock 数据应放在 dedicated data modules，不要散落在页面 markup 中。
- API 调用必须经过 services/request 边界。

## Security

- 不得提交、写入或打印密钥、token、数据库连接串、模型 API key、支付密钥或个人隐私信息。
- 小程序端不得包含服务端执行逻辑、RAG 编排、Agent 调度、Agent 平台密钥或数据库访问。
- 投研和 AI 分析内容必须保持决策辅助定位，不得表达为直接投资建议。

## Useful Context Files

新会话或新任务开始时，应优先读取：

```text
AGENTS.md
.agents/openspec-workflow.md
.agents/git-workflow.md
.agents/backend-boundaries.md
.agents/testing-tdd.md
openspec/config.yaml
../doc/architecture.md
```

如果处理已有 change，还应读取：

```text
openspec/changes/<change-name>/proposal.md
openspec/changes/<change-name>/design.md
openspec/changes/<change-name>/tasks.md
openspec/changes/<change-name>/specs/**/*.md
```

如果处理已完成能力，还应读取：

```text
openspec/specs/**/*.md
```
