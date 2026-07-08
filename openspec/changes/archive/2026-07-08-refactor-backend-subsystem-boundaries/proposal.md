## Why

当前后端已经出现 API、采集、seed、migration、connector、parser、repository 等多类能力，但主规格和代码目录仍以较早期的扁平边界表达，例如 `internal/ingestion`、`internal/jobs` 和全局 `integrations` 混用。随着采集器需要作为 backend 内独立可运行子系统推进，必须先固化 Go 后端单 module、多可部署子系统结构，避免后续 `add-ingestion-scheduler` 在错误边界上继续叠代码。

本 change 的目标是先做架构边界重构，让后续 agent 能清楚判断代码应该放在 `internal/apps/<subsystem>`、共享基础层还是共享 `integrations`。

## What Changes

- 建立 `backend/internal/apps/` 业务子系统目录模型，表达 `miniappapi`、`adminapi` 和 `ingestion` 三类子系统边界。
- 明确 `backend/cmd/*` 是独立进程入口，采集器是 backend 内独立可运行子系统，可作为独立进程或容器 entrypoint 运行，但不是独立 Go module、独立 repo 或独立数据库。
- 将采集子系统边界定义为 `internal/apps/ingestion/*`，包括 scheduler、runtime、sourcecatalog、connectors、parsers 和 health。
- 收窄 `internal/integrations` 语义，只承载跨多个后端子系统复用的外部平台集成，不再作为采集数据源 connector 的默认归属。
- 迁移或增加 package 文档、目录和必要适配层，使现有采集能力可以逐步落入新边界，同时保持现有命令和测试行为不变。
- 增加架构边界测试或静态检查，防止 `integrations` 反向依赖 `apps`，防止 miniapp/admin API 直接依赖采集 connector。
- 更新后端主规格，使 `backend-foundation`、`data-ingestion-layer` 和 `persistence-and-contracts` 与新子系统边界一致。

## Capabilities

### New Capabilities

- `backend-subsystem-boundaries`: 定义后端单 Go module、多可部署子系统、子系统目录归属、共享基础层和依赖方向。

### Modified Capabilities

- `backend-foundation`: 将旧的扁平后端边界更新为 `cmd/*` + `internal/apps/*` + 共享基础层结构。
- `data-ingestion-layer`: 将采集编排、connector、parser、source catalog 和来源健康归属到 `internal/apps/ingestion/*`。
- `persistence-and-contracts`: 明确 migration 和 seed 数据仍由 backend 统一管理，采集器不是独立数据库或独立 module。

## Impact

- 影响源码区域：`backend/cmd/*`、`backend/internal/*`、`backend/data/*`、`backend/migrations/*`。
- 影响规则文件：`.agents/backend-boundaries.md` 已定义目标边界，本 change 将代码和 OpenSpec 主规格对齐该规则。
- 影响后续 active changes：`add-ingestion-scheduler` 应在本 change 完成后按 `internal/apps/ingestion/scheduler` 和 `internal/apps/ingestion/runtime` 实现，不应继续扩大旧 `internal/jobs` 边界。
- 非目标：不实现采集调度器业务逻辑；不新增调度状态表和 run 表；不拆分 Go module、repo 或数据库；不引入队列、分布式锁或多实例调度；不修改 `prototype` 和 `doc`。
