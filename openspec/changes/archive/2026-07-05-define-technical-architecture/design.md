## Context

观潮家当前处于源码工程初始化前的架构定型阶段。`tidewise-ai` 是源码工程根目录和 OpenSpec 根目录，目前已有 OpenSpec 配置和 Codex OpenSpec skills，尚未包含 Taro 小程序代码或 Go 后端代码。

现有方向调整为：前端采用 Taro + React + TypeScript，以同时支持微信和抖音小程序；后端采用 Go + Gin 构建高并发 API/BFF，以模块化单体起步并独立部署；AI 推理、RAG、Agent 工作流编排和 Prompt 编排主要由外部 Agent 平台承载，本工程只负责 Agent 平台 API 调用、回调接收、结构化结果校验、落库和展示。

## Goals / Non-Goals

**Goals:**

- 定义观潮家 MVP 到可演进阶段的前后端分离技术架构。
- 确定前端技术栈为 Taro + React + TypeScript，首批支持微信小程序和抖音小程序。
- 确定后端技术栈为 Go + Gin API/BFF，满足高并发、高扩展性、独立部署和 AI 编程准确率诉求。
- 定义 `frontend/`、`backend/`、`infra/` 三类顶层目录边界。
- 定义后端领域模块、外部 Agent 平台集成、数据访问层、API 契约和异步任务边界。
- 定义小程序端的安全边界：只做展示、交互、轻状态和 API 调用。
- 建立 mock 数据、领域模型和 services 边界，让前端页面后续可以从 mock-first 平滑迁移到 API/BFF。

**Non-Goals:**

- 不在本 change 中实现真实 Go 后端 API、数据库、队列或部署配置。
- 不选择最终云厂商、数据库托管方案、队列产品、向量数据库产品或外部 Agent 平台供应商。
- 不在本工程中实现 AI 推理、RAG 检索、Agent 工作流编排、Prompt 编排或模型调用细节。
- 不实现真实事件采集、知识图谱推理、报告生成、订阅推送、支付或登录能力。
- 不修改 `doc` 或 `prototype` 目录。
- 不完整迁移 prototype 的所有 UI 和交互。

## Decisions

### Decision 1: Use frontend/backend separated monorepo layout

后续源码和工程配置统一放在 `tidewise-ai` 根目录下，但采用前后端分离的目录结构：

```text
tidewise-ai/
├── frontend/
│   └── miniapp/
├── backend/
│   ├── api/
│   ├── config/
│   ├── modules/
│   ├── integrations/
│   ├── internal/
│   │   └── config/
│   ├── repositories/
│   ├── contracts/
│   └── jobs/
├── infra/
├── openspec/
└── .codex/
```

`frontend/miniapp` 放 Taro 跨平台小程序。`backend` 放 Go 后端服务、环境配置、领域模块、外部集成、数据访问、API 契约和异步任务。`infra` 放基础设施配置，不放业务数据访问实现。

理由：
- `frontend` 和 `backend` 让前后端分离、独立部署和职责边界更直观。
- `backend/modules` 承载业务领域模块，避免所有逻辑堆在 API handler 中。
- `backend/config` 承载 local/uat/prod 非敏感配置文件和示例模板。
- `backend/internal/config` 承载 Go 强类型配置加载、合并和启动校验逻辑。
- `backend/integrations` 承载外部 Agent 平台、支付、消息和数据源集成。
- `backend/repositories` 承载数据库和中间件访问层，避免把数据访问误放进 `infra`。
- `backend/contracts` 承载 API 契约定义或生成入口，便于前后端联调和回调接口稳定。
- `infra` 只承载 Docker、部署、数据库迁移、CI/CD、环境模板等基础设施配置。

备选方案：
- `apps/*` + `services/*` + `packages/*`：适合大型 monorepo，但当前阶段对用户来说不如 frontend/backend 直观。
- 多 repo：边界清楚，但 MVP 阶段协作和契约演进成本更高。
- 一次性微服务 repo：过早增加部署、通信、观测和数据一致性成本。

### Decision 2: Use Taro + React + TypeScript for cross-platform miniapp

前端采用 Taro + React + TypeScript，源码位于 `frontend/miniapp`，首批目标产物为微信小程序和抖音小程序。

理由：
- 项目明确需要同时上线微信和抖音小程序，原生微信小程序不再满足多端诉求。
- Taro 官方支持微信、抖音等多个小程序平台，并支持 React/Vue 等开发方式。
- React + TypeScript 有利于组件化、复杂交互状态管理、类型约束和后续 H5/更多端扩展。
- Taro 能让业务层尽量保持跨端一致，平台差异通过适配层处理。

备选方案：
- 原生微信小程序：微信端体验直接，但无法高效复用到抖音小程序。
- uni-app：跨端覆盖广，Vue 生态成熟；若团队更偏 Vue 可以重新评估，但当前默认选择 Taro + React。
- 纯 H5 嵌入：多端复用高，但小程序原生能力、性能和平台审核适配不如小程序框架直接。

### Decision 3: Use Go + Gin for backend API/BFF

后端采用 Go 语言，优先使用 Gin 构建 API/BFF 服务。MVP 阶段采用模块化单体：一个独立部署的 Go 服务，内部按领域模块组织事件、市场、板块、AI 分析结果、报告、订阅、用户、支付回调和 Agent 平台集成。

理由：
- Go 的 goroutine 和 runtime 适合高并发 I/O、外部 API 调用、回调处理和长连接/任务调度场景。
- Go 编译产物简单，容器镜像轻，部署和水平扩展成本低。
- Gin 生态成熟，适合快速构建高性能 HTTP API/BFF。
- Go 的语法小、强类型明确、格式化统一、编译反馈直接，AI 生成代码时更容易通过类型检查、编译错误和单元测试被纠偏。
- Go 工程通常倾向显式错误处理和简单依赖结构，相比高度动态或元编程较多的技术栈，更有利于 AI 编程准确率和代码审查。
- 模块化单体可以先保持开发效率，后续再按容量、部署节奏或团队边界拆成独立服务。

备选方案：
- Node.js/NestJS：TypeScript 全栈体验好，AI 编程表现也较稳定，但 CPU 密集任务和高并发资源效率不如 Go 稳健。
- Java/Spring Boot：企业生态强，但初期工程复杂度、资源占用和启动成本较高。
- Python/FastAPI：AI 生态好，但动态类型和运行时错误更容易放大 AI 生成代码的不确定性；本工程不承载 AI 推理本体，作为高并发 API/BFF 主栈不是首选。

### Decision 4: Treat AI reasoning as external Agent platform integration

AI 推理、RAG 检索、Agent 工作流编排、Prompt 编排、工具调用和模型调用主要由外部 Agent 平台承载，例如 Dify 类平台。本工程不实现完整 AI 推理工作流。

后端负责：
- 调用 Agent 平台 API。
- 管理 Agent 平台调用鉴权、限流、重试、超时和审计。
- 接收 Agent 平台同步响应或异步回调。
- 提供结构化回写 API，让 Agent 平台把分析结果回传给系统。
- 校验结构化结果，落库后提供给前端展示。
- 统一处理投研内容安全边界，避免输出直接投资建议。

理由：
- Agent 工作流、RAG、Prompt 和工具编排变化快，交给专门平台可以降低本工程复杂度。
- 后端统一持有 Agent 平台密钥，避免前端暴露密钥。
- 结构化回写 API 可以让分析结果进入系统数据模型，便于复用、审计和订阅推送。

备选方案：
- 在本工程自建 Agent/RAG 服务：控制力强，但初期复杂度和维护成本过高。
- 前端直接调用 Agent 平台：安全风险高，无法统一鉴权、限流、审计和内容风控。

### Decision 5: Define API contracts as interface agreements, not shared implementation

`backend/contracts` 中的 API 契约用于描述接口约定，而不是业务实现。契约内容包括接口路径、请求参数、响应结构、错误结构、鉴权要求、状态码、分页结构和 Agent 平台回写接口。

契约可以采用 OpenAPI、JSON Schema 或后续选定的 schema 工具表达。若未来需要从契约生成前端类型或 SDK，可以再引入 `packages/contracts` 存放生成物；当前不强制建立 `packages/shared-types`。

理由：
- 前后端分离后，稳定 API 契约比共享 TypeScript 类型更重要。
- 后端选择 Go 后，直接共享 TypeScript 类型不再自然。
- OpenAPI/JSON Schema 更适合作为跨语言契约来源。

备选方案：
- `packages/shared-types`：适合前后端都用 TypeScript 的项目，但本项目后端选择 Go 后不作为默认目录。
- 先写代码再补契约：短期快，但前后端联调和 Agent 回调会更容易漂移。

### Decision 6: Keep storage and async infrastructure as backend boundaries

长期架构允许 PostgreSQL、Neo4j、向量存储、Redis 和队列/任务系统分别承担结构化数据、图关系、语义索引、缓存/会话/热点状态和异步任务。当前阶段只定义边界，不安装或接入这些依赖。

数据访问实现属于 `backend/repositories` 或 `backend/integrations`，不属于 `infra`。`infra` 只放基础设施配置、部署配置、数据库迁移、容器和 CI/CD。

理由：
- 避免源码骨架阶段被具体基础设施选择牵引。
- 清晰区分“访问基础设施的业务代码”和“基础设施配置”。
- 让后续数据库、队列和部署 changes 独立评估具体技术。

备选方案：
- 立即固定所有数据库和队列：会在需求未稳定时增加维护和迁移成本。
- 把数据访问层放进 `infra`：会混淆业务代码和基础设施配置边界。

### Decision 7: Keep frontend and backend independently deployed

前端和后端独立部署。`frontend/miniapp` 分别构建并发布到微信、抖音等小程序平台。`backend` 作为 Go API/BFF 服务部署到服务端环境。异步任务、Agent 回调处理和推送任务可以先与 backend 同进程，后续按容量和可靠性要求拆分为 worker 或独立服务。

理由：
- 前端发布节奏与后端部署节奏不同。
- 小程序平台审核、发布和回滚链路与服务端部署不同。
- Agent 平台回调、支付回调、订阅推送和外部数据处理必须在服务端受控环境执行。

备选方案：
- 前后端同部署单元：不适合小程序发布链路。
- 首版多后端服务：基础设施复杂度过高，MVP 阶段先用模块化单体更稳。

### Decision 8: Use typed backend environment configuration

后端必须标准化支持 `local`、`uat`、`prod` 三类环境。启动时通过 `APP_ENV` 选择环境配置，并加载统一的 Go 强类型 config。

推荐结构：

```text
backend/
├── config/
│   ├── config.local.yaml
│   ├── config.uat.yaml
│   ├── config.prod.yaml
│   └── config.example.yaml
├── internal/
│   └── config/
│       └── config.go
└── deployments/
    └── env/
        ├── local.env.example
        ├── uat.env.example
        └── prod.env.example
```

配置文件只保存非敏感结构化配置，例如服务端口、日志级别、外部接口 base URL、数据库 host/port/name、Redis 地址、限流参数和回调路径。数据库密码、Agent 平台 API key、支付密钥、JWT secret、云厂商密钥等敏感信息必须通过环境变量或部署平台 secret 注入，不得提交到 repo。

业务代码只能依赖统一 config，例如 `config.AgentPlatform.BaseURL`、`config.Database.DSN`、`config.Redis.Address`，不得在业务模块中散落读取环境变量或硬编码 local/uat/prod 分支。

理由：
- local、uat、prod 的外部接口、数据库、Redis、支付回调和 Agent 平台配置天然不同，需要标准化隔离。
- 强类型 config 可以在启动时校验必填字段，减少运行时配置错误。
- secret 与非敏感配置分离可以降低密钥泄露风险。
- 统一配置入口更适合 AI 编程和代码审查，避免环境差异散落在业务代码中。

备选方案：
- 只使用环境变量：部署简单，但大量配置分散且缺少结构化校验。
- 把所有环境配置写入 YAML：读取方便，但容易误提交 secret。
- 业务代码直接读取环境变量：短期省事，但会造成环境逻辑散落和测试困难。

### Decision 9: Keep Taro data access behind services

首阶段 Taro 前端 services 可以读取 mock 数据模块返回结构化结果，后续替换为真实 request 调用。页面和组件不能直接读取 prototype 文件，不能硬编码浏览器 DOM 状态，也不能绕过 service 层访问后端或 mock 数据。

理由：
- 支持前端与后端独立推进。
- 让 API 契约、领域模型和页面数据结构提前稳定。
- 后续接入 Go API/BFF 时尽量不改页面结构。

备选方案：
- 页面直接 import mock 数据：速度快，但后续切 API 时页面改动面过大。
- 在页面 markup 中嵌入 mock 内容：不利于复用、测试和替换。

## Risks / Trade-offs

- [风险] Taro 多端适配仍会遇到微信和抖音平台差异 → 缓解：业务层保持跨端，平台差异放到适配层，并在任务中加入双端构建验证。
- [风险] Go 后端与 TypeScript 前端无法直接共享类型 → 缓解：以 OpenAPI/JSON Schema 作为跨语言 API 契约来源，必要时生成前端类型。
- [风险] 模块化单体可能在复杂度上升后边界变模糊 → 缓解：每个后端能力通过 OpenSpec change 定义模块边界、契约和迁移条件。
- [风险] 外部 Agent 平台能力和稳定性影响核心体验 → 缓解：后端 Agent 集成层必须处理超时、重试、降级、状态追踪和结构化结果校验。
- [风险] 投研/AI 内容可能被用户理解为投资建议 → 缓解：服务端和前端均必须保持决策辅助定位，AI/报告/分析能力不得输出直接买卖建议。
- [风险] 图谱、RAG、采集和报告生成的基础设施选择较多 → 缓解：本 change 只约束能力归属，具体技术选型由后续专项 changes 决定。

## Migration Plan

1. 审阅并确认本 technical architecture change。
2. 创建 Taro + React + TypeScript 前端工程骨架。
3. 创建五个 tab 页面空壳，保证微信和抖音小程序构建目标可识别。
4. 创建前端基础目录和 placeholder 模块，承接后续页面实现。
5. 添加 mock 数据、领域模型和 services 边界的最小样例。
6. 将 `technical-architecture` 和 `mini-program-foundation` delta specs 同步为主规格。
7. 后续通过独立 changes 定义 Go backend skeleton、环境配置加载、API 契约、Agent 平台集成、数据库、队列、图谱、RAG 数据边界和 infra。

回滚策略：删除本 change 新增的 Taro 前端工程文件和目录，并通过后续 OpenSpec change 修改或替换已同步 requirements。

## Open Questions

- Taro 前端使用 React 还是 Vue 作为长期 UI 技术栈；当前默认 React。
- Go API/BFF 是否使用 Gin 作为首选 HTTP 框架；当前默认 Gin。
- API 契约格式采用 OpenAPI 还是 JSON Schema，需要在 `define-api-contracts` change 中决策。
- 外部 Agent 平台优先选择 Dify 还是其他平台，需要在 `define-agent-platform-integration` change 中决策。
- PostgreSQL、Neo4j、向量存储、Redis 和队列的具体产品、托管方式和部署环境，需要在 infra 相关 changes 中决策。
- 管理后台是否进入 MVP 工程范围，需要结合运营和内容审核流程另行确认。
