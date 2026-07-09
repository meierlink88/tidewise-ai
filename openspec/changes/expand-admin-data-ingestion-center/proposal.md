## Why

当前管理后台只有调度器设置，无法直接查看采集到的原始材料、已抽取事件、采集源目录和历史调度执行记录。数据采集链路已经具备 PostgreSQL 表和基础调度记录，本 change 需要把这些已有数据以只读运营后台方式暴露出来，便于后续验证采集质量、排查来源状态和观察事件抽取结果。

## What Changes

- 将当前管理后台 sidebar 中的 `调度器设置` 一级菜单改为 `数据采集中心`。
- 在 `数据采集中心` 内新增 4 个 tab：
  - `原始数据`：展示 `raw_documents` 列表，支持标题模糊搜索、分页，每页 50 条。
  - `全球事件`：展示 `events` 列表，支持事件标题模糊搜索、`event_status`、`fact_status`、`event_time` 时间范围、`first_seen_at` 时间范围筛选，分页每页 50 条。
  - `搜索通道`：展示 `source_catalogs` 列表，支持按 `active`、`inactive`、`disabled` 状态筛选，不分页，不展示 parser 字段。
  - `调度器`：保留现有调度器配置能力，并在同一页面右侧展示最近 50 条调度执行记录。
- 新增只读 Admin API，用于查询原始数据、事件、搜索通道和最近调度执行记录。
- 管理后台继续复用 Admin Token 鉴权、Minimal Dashboard 设计系统、自有 UI primitives 和现有 admin shell。
- 不做编辑、删除、审核、重跑采集、事件详情页、事件抽取、事件关系图谱或新增数据库表。

## Capabilities

### New Capabilities

无。本 change 不引入独立新能力，属于 `admin-console` 的增量扩展。

### Modified Capabilities

- `admin-console`：从单一调度器设置页扩展为 `数据采集中心` 一级菜单，新增原始数据、全球事件、搜索通道和调度执行记录的只读查询展示能力。

## Impact

- 后端：
  - `backend/internal/apps/adminapi/` 增加只读查询路由、请求参数解析和响应 DTO。
  - `backend/internal/repositories/` 增加面向 admin 查询的分页/筛选 repository 方法。
  - 不直接调用采集 connector，不改变 ingestion runtime，不新增 migration。
- 前端：
  - `frontend/admin/src/api/` 增加数据采集中心 API client。
  - `frontend/admin/src/pages/` 增加或重组数据采集中心页面和 4 个 tab。
  - `frontend/admin/src/components/ui/` 可按需要补充 table、tabs、pagination 等 Minimal Dashboard 风格基础组件。
- OpenSpec：
  - 更新 `admin-console` delta spec，明确 `数据采集中心` 的只读查询、分页、筛选和调度记录展示行为。
- 非目标：
  - 不更新 `prototype` 或 `doc`。
  - 不实现小程序端能力。
  - 不实现事件审核、采集源编辑、手动重跑、详情页或 AI 事件抽取流程。
