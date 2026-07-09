## 1. OpenSpec Review

- [ ] 1.1 确认 proposal、design、delta spec 和 tasks 覆盖本次只读管理后台范围。
- [ ] 1.2 运行 `openspec validate expand-admin-data-ingestion-center` 并修正 artifact 问题。

## 2. Backend TDD

- [ ] 2.1 为 `GET /admin/raw-documents` 编写 Admin API 测试，覆盖 Admin Token、分页默认值、标题搜索和空结果。
- [ ] 2.2 为 `GET /admin/events` 编写 Admin API 测试，覆盖 Admin Token、分页默认值、标题搜索、`event_status`、`fact_status`、`event_time` 范围和 `first_seen_at` 范围。
- [ ] 2.3 为 `GET /admin/source-catalogs` 编写 Admin API 测试，覆盖 Admin Token、状态筛选和不返回 parser 字段。
- [ ] 2.4 为 `GET /admin/scheduler/runs?limit=50` 补充测试，确认返回最近 50 条执行记录且统计语义为调度轮次内 source 结果。

## 3. Backend Implementation

- [ ] 3.1 增加 admin 查询 filter、result 和 DTO，统一分页响应结构。
- [ ] 3.2 在 repository 层实现原始数据分页查询、全球事件分页查询和搜索通道状态查询。
- [ ] 3.3 在 `backend/internal/apps/adminapi` 增加原始数据、全球事件和搜索通道路由。
- [ ] 3.4 复用现有调度器 run 查询接口，确保前端可以稳定获取最近 50 条执行记录。
- [ ] 3.5 运行后端相关包测试和 `go test ./...`。

## 4. Frontend TDD And Verification

- [ ] 4.1 为管理后台 API client 或页面行为补充可自动化测试，覆盖四个 tab 的请求参数和基础渲染。
- [ ] 4.2 为分页、状态筛选、事件筛选和 source parser 不展示行为补充测试或可重复验证步骤。

## 5. Frontend Implementation

- [ ] 5.1 将 sidebar 中 `调度器设置` 改为 `数据采集中心`。
- [ ] 5.2 新增 `数据采集中心` 页面，包含 `原始数据`、`全球事件`、`搜索通道`、`调度器` 四个 tab。
- [ ] 5.3 增加 Minimal Dashboard 风格 `Tabs`、`DataTable`、`Pagination` 等必要自有 UI 组件。
- [ ] 5.4 实现原始数据列表，支持标题搜索、分页和空状态。
- [ ] 5.5 实现全球事件列表，支持标题搜索、四类筛选、分页和空状态。
- [ ] 5.6 实现搜索通道列表，支持状态筛选，不分页，不展示 parser。
- [ ] 5.7 调整调度器 tab 为左侧配置、右侧最近 50 条执行记录列表。

## 6. Final Validation

- [ ] 6.1 运行 `openspec validate expand-admin-data-ingestion-center`。
- [ ] 6.2 运行后端测试、前端测试和构建验证。
- [ ] 6.3 在本地管理后台人工验证四个 tab、分页、筛选、保存调度器配置和执行记录展示。
- [ ] 6.4 更新 tasks 完成状态，准备 sync specs、archive 和 PR。
