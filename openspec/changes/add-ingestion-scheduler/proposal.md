## Why

当前采集层已经具备 `source_catalogs`、手动 `source-ingest`、active source 批量选择、多来源并发采集、provider 限流、AI Web Research connector 和原始文档幂等入库能力，但还缺少长期运行的调度控制面。现在仍需要人工运行命令才能触发采集，无法通过系统配置让采集器按全局策略持续运行。

本 change 要把“人工触发采集”升级为“可配置、可观察、可管理的后端采集调度”。调度器先只负责全局调度策略，不在本阶段做每个 source 独立 cron 配置；采集执行继续复用现有 active source 运行时和并发能力。与此同时，需要新增一个独立 Web 管理后台 `frontend/admin/`，本阶段只提供调度器设置菜单，为后续采集源管理、原始数据查询和事件列表管理留下入口。

## What Changes

- 新增后端采集调度器，支持全局 interval 模式和固定时间模式。
- interval 模式支持按分钟或小时配置循环执行，例如每 15 分钟、每 2 小时触发一次。
- 固定时间模式支持配置至少 5 个每日固定触发时间，例如 09:00、12:00、15:00、18:00、21:00；调度器进程持续驻留，到点触发，执行完成后等待下一次触发。
- 调度器只负责全局触发策略、运行状态、失败隔离和 run 记录；实际采集必须复用现有 `IngestionJob`、connector、parser、writer、provider rate limiter 和并发控制。
- 调度执行时从 `source_catalogs` 选择 active source，可按全局过滤条件限制 provider、channel 或 source type；默认不支持按单个 active source 调度，也不重新定义每个 source 的独立调度频率。
- 新增调度配置持久化，保存全局启用状态、调度模式、interval、固定时间列表、并发数、batch size、超时、过滤条件和最近运行摘要。
- 新增采集运行记录，保存每次调度 run、每个 source 的执行结果、写入数量、重复数量、失败原因和耗时。
- 新增 `backend/cmd/ingestion-scheduler` 进程入口，支持持续运行和单轮验证。
- 新增 `backend/cmd/admin-api` 或复用既有 admin API 边界，为管理后台提供调度器配置查询、保存、启停和最近运行状态接口。
- 新增 `frontend/admin/`，采用 Vite + React + TypeScript + Ant Design，实现独立 Web 管理后台；本阶段只实现“调度器设置”菜单。
- 管理后台 MVP 使用 Admin Token 鉴权：后端通过 `ADMIN_API_TOKEN` 校验 `Authorization: Bearer <token>`；真实 token 不得提交到 repo、数据库或前端源码。
- 更新本地运行说明，覆盖 migration、admin API、admin 前端、scheduler 启动、单轮运行和验证 SQL。

## Capabilities

### New Capabilities

- `ingestion-scheduler`: 定义全局采集调度器、调度配置、运行记录、失败隔离、持续进程和单轮验证。
- `admin-console`: 定义独立 Web 管理后台、Admin Token 鉴权和调度器设置页面。

### Modified Capabilities

- `data-ingestion-layer`: 从“可手动多来源采集”扩展为“可由全局调度器触发多来源采集”，要求调度器复用现有 active source 并发运行时。
- `persistence-and-contracts`: 增加调度配置、调度运行记录和管理后台 API 契约的 PostgreSQL 与接口边界要求。

## Impact

- 影响源码区域：`backend/internal/apps/ingestion/scheduler`、`backend/internal/apps/adminapi`、`backend/internal/repositories`、`backend/internal/domain`、`backend/internal/config`、`backend/cmd`、`backend/migrations`、`frontend/admin`、`infra/local/README.md`。
- 影响数据结构：需要新增非破坏性 migration，为全局调度配置和采集运行结果提供持久化结构。
- 影响运行方式：新增后台调度进程入口和独立 admin Web 前端，但不改变现有小程序 API/BFF 进程和手动 `source-ingest` 命令。
- 影响外部系统：调度器会按配置持续访问 active source，必须保留保守并发、provider 限流、超时、失败隔离和可关闭开关。
- 非目标：不引入 Kafka、Temporal、Airflow、CronJob 平台或独立任务队列；不实现 per-source 独立调度 UI；不实现 SDK 类 worker；不实现事件抽取、AI tag、实体关联或图谱写入；不实现原始数据列表和事件列表页面；不修改 `prototype` 和 `doc` 目录。
