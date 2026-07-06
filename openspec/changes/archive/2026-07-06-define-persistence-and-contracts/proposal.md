## Why

当前工程已经具备 Taro 小程序壳和 Go API/BFF 后端骨架，但正式模块开发前仍缺少持久化、API 契约、Agent 平台回写、异步任务和前端真实 API 接入的统一架构约定。若现在直接开发事件、报告、订阅或 AI 分析模块，容易出现前后端 DTO 不一致、数据访问边界分散、Agent 回写结构不稳定和后续迁移成本偏高的问题。

## What Changes

- 明确 MVP 阶段结构化主存储、缓存、图谱/向量能力、迁移和数据访问边界。
- 定义前后端 API 契约体系，包括 DTO、错误、分页、时间、ID、Agent 回写结构和契约存放位置。
- 明确 Go 后端模块分层，包括 HTTP、应用服务、领域模型、repository、外部集成和 job 边界。
- 明确数据采集层边界，包括自研爬虫脚本采集和外部 Agent API 采集结果接入两种路径。
- 明确 Agent 平台调用与回写模式，包括后端主动调用、回调接收、鉴权、幂等、校验和落库边界。
- 明确采集数据清洗、标准化、入库和后续进入关系型数据库、向量数据库或图谱数据库的演进边界。
- 明确异步任务与调度边界，包括报告生成、事件采集、Agent 分析、通知推送等能力的同步/异步判断规则。
- 明确 Taro 小程序从 mock-first service 迁移到真实 API 的 request、错误处理、鉴权和 mock 切换规范。
- 更新过期的 OpenSpec 项目上下文，避免继续描述“尚未包含 Taro 小程序壳或 Go 后端骨架”。
- 不在本 change 中实现真实业务模块、数据库连接、迁移脚本、认证登录、支付、Agent 平台真实调用或前端真实接口切换。

## Capabilities

### New Capabilities

- `persistence-and-contracts`: 定义持久化、缓存、迁移、API 契约、数据采集、Agent 回写和异步任务的正式模块开发前置架构能力。

### Modified Capabilities

- `technical-architecture`: 补充数据库/缓存/图谱/向量、API 契约、数据采集、Agent 回写和异步任务的主架构要求。
- `backend-foundation`: 补充 Go 后端模块分层、数据访问、迁移、契约、数据采集和异步任务边界要求。
- `mini-program-foundation`: 补充 Taro 小程序真实 API 接入、request 封装、错误处理和 mock/real 切换要求。

## Impact

- 影响 OpenSpec artifacts 和主规格：`openspec/changes/define-persistence-and-contracts/`、`openspec/specs/**/spec.md`。
- 影响项目上下文：`openspec/config.yaml` 需要修正当前工程状态描述。
- 影响未来代码区域的约定：`backend/internal`、`backend/config`、未来 `backend/contracts`、未来 `backend/internal/domain`、未来 `backend/internal/application`、未来 `backend/internal/ingestion`、未来 `backend/internal/repositories`、未来 `backend/internal/jobs`、`frontend/miniapp/src/services` 和 `frontend/miniapp/src/models`。
- 不修改 `../doc` 或 `../prototype`，它们只作为后续产品和架构参考输入。
- 不引入真实 secret、生产连接串、Agent 平台密钥、数据库密码或支付密钥。
