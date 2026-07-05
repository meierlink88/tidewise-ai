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

OpenSpec 内容语言规则：

- 所有 OpenSpec 生成内容默认使用中文。
- 只有 OpenSpec 规范要求保留的框架性文案、固定标题、关键字、命令、文件名、路径、代码标识和协议字段可以保留英文。
- proposal、design、tasks、spec requirements 和 scenarios 的正文、说明、任务描述、风险、取舍、影响范围和验收内容都应使用中文。

标准流程：

```text
Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive
```

各阶段含义：

- Explore：讨论问题、架构、边界和取舍，不直接写实现代码。
- Propose：创建 `openspec/changes/<change-name>/`，生成 proposal、specs、design、tasks。
- Review：人工确认 artifacts 是否符合方向和范围。
- Apply：严格按 tasks 实现代码，并在完成后更新任务状态。
- Validate：运行可用验证命令，检查配置、类型、lint、测试或运行结果。
- Sync：将 delta specs 同步到 `openspec/specs/`，使其成为当前系统事实。
- Archive：完成后归档 change，保留历史决策与实现记录。

## OpenSpec Directory Model

OpenSpec 文件含义：

```text
openspec/
├── config.yaml   # 项目上下文、技术约束和 artifact 规则
├── specs/        # 当前系统已经生效的主规格
└── changes/      # 正在设计或实现的变更
```

每个 change 通常包含：

```text
openspec/changes/<change-name>/
├── proposal.md   # 为什么做、做什么、不做什么、影响范围
├── design.md     # 怎么做、技术选型、架构边界、风险取舍
├── tasks.md      # 可执行实现清单
└── specs/        # 本次变更的 delta requirements
```

主规格 `openspec/specs/` 是系统当前行为和能力的事实来源。新 change 必须基于主规格和现有代码增量设计。

## Iteration Rules

开始任何新 change 前，必须：

- 读取 `openspec/config.yaml`。
- 读取相关主规格 `openspec/specs/**/spec.md`。
- 检查相关已有代码目录和文件。
- 总结当前系统状态，再提出增量方案。
- 优先复用和扩展已有模块，不要创建平行结构。
- 明确本次 change 的 scope、non-goals 和 impact。

实现任何 change 前，必须：

- 读取该 change 的 `proposal.md`、`design.md`、`tasks.md` 和相关 `specs/**/spec.md`。
- 读取受影响的现有代码文件。
- 说明将复用哪些已有页面、组件、services、models、data、store 或配置。
- 严格按照 tasks 顺序执行。
- 完成一个任务后立即把对应 checkbox 从 `- [ ]` 改为 `- [x]`。

实现过程中如果发现设计不匹配，必须：

- 暂停继续编码。
- 说明 design/spec/tasks 与现实代码的冲突。
- 先更新 OpenSpec artifacts 或征求用户确认。
- 不得在 artifacts 过期时继续盲目实现。

完成 change 后，必须：

- 运行适当验证。
- 确认 tasks 全部完成。
- 同步 delta specs 到 `openspec/specs/`。
- 归档 change 到 `openspec/changes/archive/`。

## Git Branch, Worktree, and Commit Workflow

OpenSpec change 是本项目的正式工作单元。Git branch 是 change 的交付边界，commit 是阶段性检查点，worktree 只用于并行隔离。

除项目初始 baseline、紧急小修或用户明确要求直接在 `main` 操作外，正式 OpenSpec change 必须从 `main` 创建独立分支：

```text
codex/<change-name>
```

标准分支流程：

```text
1. 确认 `main` 干净，并从 `main` 开始。
2. 创建或切换到 `codex/<change-name>`。
3. 执行 Explore/Propose，生成或更新 `openspec/changes/<change-name>/` artifacts。
4. 运行 `openspec validate <change-name>`。
5. 提交 propose 检查点，推荐 commit message：`spec: propose <change-name>`。
6. 等待或完成 Review，确认 artifacts 可以进入实现。
7. 执行 Apply，严格按 `tasks.md` 顺序实现。
8. 每完成一组可验证任务后更新 checkbox，并按需要提交阶段性 commit。
9. tasks 全部完成后运行适当验证。
10. Sync delta specs 到 `openspec/specs/`。
11. Archive change 到 `openspec/changes/archive/`。
12. 运行 `openspec validate --all`。
13. 提交 archive 检查点，推荐 commit message：`spec: archive <change-name>`。
14. 通过 PR 或明确确认后合并回 `main`。
```

commit 规则：

- propose artifacts 完整且 `openspec validate <change-name>` 通过后，应提交一次 `spec: propose <change-name>`。
- apply 过程中，完成一组有独立验证意义的任务后可以提交一次，例如 `chore: add backend config foundation` 或 `feat: add backend health endpoints`。
- 不要在 tasks 未更新、验证未运行或 artifacts 明显过期时提交完成态代码。
- sync/archive 完成且 `openspec validate --all` 通过后，应提交一次 `spec: archive <change-name>`。
- commit 前必须检查 `git status --short`，确认没有把 `node_modules`、构建产物、缓存、真实 secret 或无关文件加入提交。

worktree 规则：

- 默认一个 change 使用当前 worktree 加一个独立 branch 即可。
- 当多个 change 并行、某个 change 长期未完成但需要切换任务、或多个 Codex 线程同时工作时，才创建额外 worktree。
- worktree 目录建议放在 `tidewise-ai` 同级目录，并带 `tidewise-ai-wt-<change-name>` 前缀。
- 不要在两个 worktree 中同时修改同一个 OpenSpec change，避免 tasks 状态和 specs delta 冲突。

`main` 规则：

- `main` 应保持可验证、可恢复的稳定状态。
- `main` 可以包含已 propose 但未 apply 的轻量 active change；不应长期包含半实现代码。
- 合并回 `main` 前应确认 OpenSpec 校验通过，且相关 tasks、sync/archive 状态与代码实现一致。

## Anti-Duplication Rules

AI agent 必须在已有实现基础上增量迭代，不得另起一套。

禁止行为：

- 不得重复创建已有页面目录。
- 不得重复创建已有 service、model、store、data 或 config 层。
- 不得绕过已有 service 直接在页面中硬编码数据访问逻辑。
- 不得在未检查现有代码前生成新的平行工程结构。
- 不得因为新 change 而重建已有工程骨架。

如果现有实现不符合新设计，应先更新 design 和 tasks，再执行迁移或重构。

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

后端工程应作为独立 change 设计，例如：

```text
init-backend-skeleton
```

推荐后续目录方向：

```text
backend/api/                # Go API/BFF 服务入口
backend/modules/            # 领域模块
backend/integrations/       # Agent 平台、支付、消息、外部数据源等外部集成
backend/repositories/       # 数据访问层
backend/contracts/          # API 契约定义或生成入口
backend/jobs/               # 异步任务、定时任务、回调处理
backend/config/             # local/uat/prod 非敏感配置模板
backend/internal/config/    # Go 强类型配置加载和校验
infra/                      # 基础设施、容器、部署、数据库迁移和 CI/CD 配置
```

不要在小程序 change 中实现真实后端能力。后端工程、API 契约、数据库、Agent 平台集成和部署拓扑应通过独立 OpenSpec change 设计和实现。

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
