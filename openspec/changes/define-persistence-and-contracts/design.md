## Context

当前工程已经完成 Taro 跨平台小程序壳和 Go + Gin API/BFF 后端骨架。前端已有 `frontend/miniapp/src/models`、`data`、`services` 和 mock-first 页面，后端已有 `backend/internal/config`、`internal/http`、`internal/integrations`、`internal/repositories`、`internal/jobs` 等最小边界。

正式模块开发前，系统需要先确定持久化、API 契约、Agent 平台回写、异步任务和前端真实 API 接入规范。否则事件、报告、订阅、AI 分析等模块会在模型、DTO、错误格式、数据访问和任务状态上各自做局部决策，后续难以统一。

本 change 只做架构定义和 OpenSpec 项目上下文修正，不实现真实数据库连接、认证、支付、Agent 调用或业务 API。

## Goals / Non-Goals

**Goals:**

- 明确 MVP 阶段 PostgreSQL 作为结构化主存储。
- 明确 Redis 作为缓存、限流、幂等和短期任务状态基础设施。
- 明确图谱和向量能力在 MVP 阶段不作为独立数据库引入，先通过 PostgreSQL 结构化关系表和外部 Agent 平台结果承载。
- 明确 API 契约作为前端、后端和 Agent 平台回写的共同边界。
- 明确 Go 后端正式模块分层和目录演进方向。
- 明确 Agent 平台调用、回写、鉴权、幂等、校验和落库边界。
- 明确异步任务和调度边界。
- 明确 Taro 小程序从 mock-first service 切换到真实 API 的规则。
- 修正 `openspec/config.yaml` 中已经过期的当前工程状态描述。

**Non-Goals:**

- 不创建真实业务表、迁移脚本或数据库连接实现。
- 不引入真实 PostgreSQL、Redis、队列或部署配置。
- 不实现登录、订阅、支付、报告、事件、图谱或 RAG 业务模块。
- 不实现真实 Agent 平台 API 调用或回调 handler。
- 不把 `../doc` 或 `../prototype` 纳入修改范围。

## Decisions

### Decision: PostgreSQL 作为 MVP 结构化主存储

MVP 阶段使用 PostgreSQL 承载用户、事件、市场指标、板块、资产、订阅、报告、Agent 分析结果、任务记录和基础关系数据。

选择理由：

- 结构化数据和事务需求明确，PostgreSQL 对一致性、索引、JSONB、时间序列扩展和后续分析查询都有较好支撑。
- Go 生态对 PostgreSQL 支持成熟，适合用 `pgx` 或 `database/sql` 构建清晰 repository 边界。
- 对 AI 编程友好：schema、migration、Go struct 和 repository 测试能够提供明确反馈。

备选方案：

- MySQL：成熟但 JSON、复杂查询和未来分析扩展弹性弱于 PostgreSQL。
- MongoDB：文档灵活，但核心业务仍需要较多关系、一致性和查询约束。
- 云厂商专有数据库：早期绑定过重，不利于本地开发和 UAT/生产一致性。

### Decision: Redis 进入 MVP 基础设施，但不承载长期事实数据

Redis 用于缓存、限流、幂等键、临时会话状态和短期任务进度。长期事实数据必须落 PostgreSQL 或外部平台事实源。

选择理由：

- Agent 回写、支付回调、订阅状态刷新和报告生成都需要幂等与短期状态。
- Redis 适合高并发读写和过期键，不适合作为系统长期事实来源。

备选方案：

- 暂不引入 Redis：实现更简单，但幂等、限流和任务状态会分散到数据库或内存中。
- 使用数据库表模拟全部短期状态：一致性强，但高频临时状态会增加主库压力。

### Decision: 图谱和向量能力暂不独立建库

MVP 阶段不直接引入 Neo4j、JanusGraph、Milvus、Weaviate 等独立图数据库或向量数据库。系统先通过 PostgreSQL 保存实体、关系、Agent 分析结果和外部引用；RAG、Prompt 编排、向量检索和复杂推理主要由外部 Agent 平台承载。

选择理由：

- 当前产品优先验证事件理解、报告、订阅和 AI 辅助决策流程，独立图/向量数据库会过早增加部署和数据同步复杂度。
- 外部 Agent 平台已经承担 RAG 和推理编排，本工程更需要稳定的调用、回写和结构化结果边界。

备选方案：

- 立即引入图数据库：表达能力强，但工程和运维复杂度高。
- 立即引入向量数据库：适合大规模检索，但 MVP 阶段可先依赖外部 Agent 平台。

### Decision: API 契约作为独立边界

建立 `backend/contracts` 或等价契约目录作为 Go 后端、Taro 前端和 Agent 平台回写的共同契约来源。契约内容包含 DTO、错误结构、分页、时间、ID、枚举、Agent 回写 payload 和版本约定。

选择理由：

- 当前 `frontend/miniapp/src/models` 是前端展示模型，不能天然代表服务端 API 契约。
- 契约边界能让 AI 编程在新增模块时先对齐请求响应，再分别实现前后端。
- 契约可以先用 Markdown/OpenAPI/JSON Schema 形式描述，后续再决定是否生成 TypeScript 和 Go 类型。

备选方案：

- 只用 Go struct 作为契约：后端方便，但前端和 Agent 平台难以独立审阅。
- 只用前端 TypeScript 类型作为契约：前端方便，但无法覆盖回写、错误、分页和服务端校验。

### Decision: 后端采用模块化单体分层

正式模块开发时，后端在现有 `backend/internal` 基础上演进为清晰分层：

- `internal/http`：路由、handler、middleware、HTTP DTO 映射。
- `internal/application`：用例编排、事务边界、调用 repository 和 integrations。
- `internal/domain`：领域模型、领域规则、枚举和值对象。
- `internal/repositories`：数据访问接口和实现边界。
- `internal/integrations`：Agent 平台、支付、消息、外部数据源等外部系统适配。
- `internal/jobs`：异步任务、定时任务、回调后处理和重试编排。
- `backend/contracts`：API 契约和回写契约来源。

选择理由：

- 模块化单体适合 MVP，避免过早服务拆分。
- 分层边界能减少 handler 直接访问数据库或外部平台的风险。
- 未来可以按领域或容量拆分服务，同时保持 API/BFF 契约稳定。

### Decision: Agent 平台回写必须经过后端契约和幂等边界

Agent 平台可以由后端主动调用，也可以回写后端 API。所有回写必须经过后端鉴权、请求签名或密钥校验、幂等键、schema 校验、状态流转和落库。

选择理由：

- Agent 输出属于决策辅助内容，需要结构化校验和安全定位。
- Agent 回写可能重试或乱序，必须具备幂等和状态约束。
- 前端不能直接持有 Agent 平台凭证或回调入口密钥。

### Decision: 异步任务优先抽象，队列实现延后

正式模块必须识别长时间任务，使用 job 状态模型表达任务生命周期。MVP 可以先用数据库状态和服务端 job runner 表达任务，后续再引入队列。

选择理由：

- 报告生成、Agent 分析、事件采集和通知投递都可能超过一次 HTTP 请求的合理时长。
- 先固定 job 契约和状态，比立即选择具体队列更关键。

备选方案：

- 立即引入消息队列：能力完整，但部署复杂度更高。
- 全部同步 HTTP：实现简单，但用户体验和稳定性较差。

### Decision: 前端 request 从 mock wrapper 演进为环境感知 API 边界

`frontend/miniapp/src/services/request.ts` 后续必须支持 base URL、环境配置、统一错误、鉴权 token 注入、超时、重试策略和 mock/real 切换。页面仍只能依赖 service 模块，不能绕过 service 直接请求后端。

选择理由：

- 当前 mock-first 模式适合原型开发，但正式模块需要稳定 API 入口。
- Taro 小程序需要同时面向微信和抖音，request 封装必须屏蔽平台差异。

## Risks / Trade-offs

- [Risk] PostgreSQL + Redis 同时进入架构会增加本地环境复杂度。→ Mitigation：本 change 只定义边界，后续 `init-persistence-skeleton` 再提供本地启动和配置模板。
- [Risk] 暂不引入独立图/向量数据库可能限制复杂图谱和 RAG 能力。→ Mitigation：MVP 通过 PostgreSQL 关系表和外部 Agent 平台承载，等查询规模和场景明确后再独立 change 引入。
- [Risk] API 契约如果只写文档，可能和代码漂移。→ Mitigation：后续实现 change 必须把契约校验纳入 tasks，并逐步引入 schema 或生成工具。
- [Risk] Agent 回写接口设计不严谨会造成重复落库或错误状态。→ Mitigation：要求回写必须包含幂等键、状态流转和结构化校验。
- [Risk] 后端分层过细可能拖慢 MVP。→ Mitigation：采用模块化单体和最小目录，不提前拆服务，只在模块内保持边界清晰。

## Migration Plan

1. 本 change 先更新 OpenSpec artifacts 和 `openspec/config.yaml` 当前状态描述。
2. Review 通过后，将 delta specs 同步到主规格并归档。
3. 后续创建 `init-persistence-skeleton`，实现 PostgreSQL/Redis 配置、迁移目录、连接初始化和 repository 基础测试。
4. 后续创建 `define-api-contracts` 或在业务模块 change 中先补充契约，再实现真实 API。
5. 后续前端从 mock-first service 渐进切换到真实 API，不一次性替换所有页面。

## Open Questions

- API 契约最终采用 OpenAPI、JSON Schema、Markdown 规范，还是混合方式，需要在实现契约骨架时确认。
- Redis 是否在最早的后端持久化骨架中一起引入，还是先只保留配置和接口，需要结合部署环境确认。
- Agent 平台具体是 Dify、Coze、自研工作流还是其他平台，需要在 Agent 集成 change 中确认具体鉴权和回写协议。
