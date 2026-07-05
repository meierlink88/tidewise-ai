# AGENTS.md

## Project Overview

观潮家是一个全球政经事件驱动的市场理解与决策辅助产品。系统目标是通过事件采集、知识图谱、RAG、Agent 推理、板块/资产传导分析、报告生成和订阅能力，帮助用户理解宏观事件、产业事件、市场指标、板块和企业之间的关系。

产品输出定位为决策辅助与市场理解，不应表达为直接投资建议。

## Workspace Boundary

当前 TRAE 工作区应打开本目录：

```text
/Users/meierlink/Documents/david/创业项目/观潮家/tidewise-ai
```

本目录是源码工程根目录，也是 OpenSpec 根目录。所有工程代码、OpenSpec 变更、主规格、TRAE skills、项目级 agent 规则都应以本目录为工作根。

上级目录结构用途如下：

```text
观潮家/
├── tidewise-ai/ # 源码工程根目录，OpenSpec 根目录，TRAE 工作区根目录
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

- MVP 首端为微信小程序。
- 前端通过微信小程序平台发布。
- 小程序端只负责展示、交互、轻状态和 API 调用。
- 小程序端不得保存模型密钥、支付密钥、数据库连接或后端凭证。

后端：

- API/BFF、AI 推理、RAG、知识图谱、数据采集、订阅推送、支付回调等能力作为服务端应用独立部署。
- MVP 阶段可采用模块化单体，后续再拆分为独立服务。

部署：

- frontend 和 backend 分离部署。
- 微信小程序 frontend 发布到微信小程序平台。
- backend API/BFF 和 intelligence services 部署到服务端环境。

## Frontend Direction

MVP 阶段前端采用：

```text
原生微信小程序 + TypeScript + WXSS
```

当前不引入 Taro、uni-app 或其他跨平台框架。项目应通过端无关 API、稳定 DTO、清晰 models、services 和设计系统为未来跨平台预留能力。

微信小程序源码建议位于：

```text
apps/miniprogram/
```

如果某个既有 OpenSpec artifact 中仍出现 `miniprogram/` 作为根目录，应在实现前确认是否需要调整为 `apps/miniprogram/`，避免未来 monorepo 演进时产生迁移成本。

## Backend Direction

后端工程应作为独立 change 设计，例如：

```text
init-backend-skeleton
```

推荐后续目录方向：

```text
apps/api/                  # API/BFF 服务
services/intelligence/     # 后续 AI 推理服务
services/ingestion/        # 后续数据采集服务
services/graph/            # 后续图谱服务
services/rag/              # 后续 RAG 服务
packages/shared-types/     # 前后端共享类型
packages/api-contracts/    # API 契约
infra/                     # 数据库、容器、部署配置
```

不要在小程序 change 中实现真实后端能力。后端工程、API 契约、数据库、AI 服务和部署拓扑应通过独立 OpenSpec change 设计和实现。

## Code Style

- 不要在源码中添加注释，除非用户明确要求。
- 优先使用 TypeScript 类型表达数据结构和契约。
- 页面逻辑应通过 data/setData、bindtap、wx:for、wx:if 等小程序数据驱动方式实现。
- 禁止在生产小程序代码中使用 `document`、`window`、`innerHTML`、内联 `onclick` 或浏览器 DOM 操作。
- mock 数据应放在 dedicated data modules，不要散落在页面 markup 中。
- API 调用必须经过 services/request 边界。

## Security

- 不得提交、写入或打印密钥、token、数据库连接串、模型 API key、支付密钥或个人隐私信息。
- 小程序端不得包含服务端执行逻辑、RAG 编排、Agent 调度或数据库访问。
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
