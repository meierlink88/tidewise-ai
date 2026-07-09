## 1. 调度持久化结构

- [x] 1.1 为调度相关 migration 编写静态测试，验证新增表、索引、外键、默认值、非破坏性迁移和敏感字段禁止写入。
- [x] 1.2 新增调度 migration，创建 `ingestion_scheduler_configs`、`ingestion_runs`、`ingestion_run_sources`。
- [x] 1.3 为全局调度配置、调度模式、固定时间、run 记录和 source 结果补充 domain model 测试。
- [x] 1.4 实现调度配置、run 记录和 source 结果的 domain model。

## 2. Repository 调度能力

- [x] 2.1 为 repository 编写测试，覆盖读取默认配置、保存配置、创建 run、写入 source 结果、完成 run 汇总和查询最近 run。
- [x] 2.2 扩展 repository 接口、内存实现和 PostgreSQL 实现，支持调度器和 admin API 需要的读写能力。
- [x] 2.3 编写 PostgreSQL gated 集成测试，验证 migration 后调度表与已有 `source_catalogs`、`raw_documents` 共存且不破坏已有数据。

## 3. Scheduler service

- [x] 3.1 使用 fake clock 和 table-driven tests 编写 `TriggerPlanner` 测试，覆盖 interval、固定时间、跨天、禁用状态、非法配置和至少 5 个固定时间。
- [x] 3.2 实现 `TriggerPlanner`，负责判断是否到期和计算下一次触发时间。
- [x] 3.3 在 `internal/apps/ingestion/scheduler` 使用 fake repository、fake runner 和 fake clock 编写 scheduler service 测试，覆盖未启用、无 active source、成功执行、失败隔离、run 记录汇总和 source filter 传递。
- [x] 3.4 实现 scheduler service，负责读取全局配置、创建 run、按全局过滤条件调用现有 ingestion runtime、写入 source 结果、完成 run 汇总和输出 report。
- [x] 3.5 为并发数、batch size、tick 间隔、超时、应用时区和默认关闭策略编写配置测试。
- [x] 3.6 实现调度器配置加载，确保 local、uat、prod 通过统一 config 读取非敏感配置，敏感信息仍只通过环境变量或 secret 注入。

## 4. 命令入口和本地运行

- [ ] 4.1 为 `cmd/ingestion-scheduler` 编写命令参数测试，覆盖 `-once`、持续模式、tick 间隔、配置刷新、退出信号和 dry run 或等价安全预览参数。
- [ ] 4.2 实现 `cmd/ingestion-scheduler`，通过 `internal/apps/ingestion/scheduler` 组装调度服务，支持单轮运行和持续 worker loop，并在收到取消信号时完成当前轮次后退出。
- [ ] 4.3 保留并验证 `source-ingest` 手动触发路径，确保调度器实现不破坏 provider/channel/source-type 指定采集。

## 5. Admin API

- [ ] 5.1 为 Admin Token middleware 编写 `httptest` 测试，覆盖 token 缺失、token 错误、token 正确和未配置 token 时拒绝访问。
- [ ] 5.2 实现 Admin Token middleware，通过 `ADMIN_API_TOKEN` 校验 `Authorization: Bearer <token>`。
- [ ] 5.3 为 scheduler admin API 编写 `httptest` 测试，覆盖查询配置、保存配置、查询最近 run、非法调度模式、非法固定时间、非法 interval、非法并发和非法 batch size。
- [ ] 5.4 在 `backend/internal/apps/adminapi` 实现 scheduler admin handler、DTO 校验和 repository 调用。
- [ ] 5.5 新增或补齐 `backend/cmd/admin-api` 入口，负责加载配置、组装 repository、注册 admin 路由和启动 HTTP 服务。

## 6. Admin 前端

- [ ] 6.1 初始化 `frontend/admin/` 为 Vite + React + TypeScript + Ant Design 独立 Web 管理后台，不影响 `frontend/miniapp/`。
- [ ] 6.2 为 scheduler settings 页面编写前端测试，覆盖 token header 注入、加载配置、保存配置、interval 模式、固定时间模式和表单校验。
- [ ] 6.3 实现 admin 布局和单一菜单“调度器设置”。
- [ ] 6.4 实现调度器设置页面，支持启停、模式切换、interval、至少 5 个固定时间、并发数、batch size、timeout、source filter、保存和最近运行摘要展示。
- [ ] 6.5 验证 admin 前端本地启动、页面可访问、表单不溢出、请求携带 Admin Token，并记录本地访问方式。

## 7. 文档和验证

- [ ] 7.1 更新 `infra/local/README.md`，补充 migration、admin API、admin 前端、scheduler 单轮、scheduler 持续运行和验证 SQL。
- [ ] 7.2 运行 `go test ./...`，确保单元测试、fixture 测试和 gated 集成测试边界保持通过。
- [ ] 7.3 运行 admin 前端测试和构建验证。
- [ ] 7.4 运行 `openspec validate add-ingestion-scheduler`。
- [ ] 7.5 在本地 PostgreSQL 执行 migration，并配置只启用 AI Web Research 或少量低风险 active source 的调度过滤条件。
- [ ] 7.6 运行 `ingestion-scheduler -once` 或等价命令，验证生成 run 记录、source 结果和 `raw_documents` 幂等写入。
- [ ] 7.7 启动持续调度器，验证 interval 触发一次真实采集后可停止进程，且 run 记录可查询。
- [ ] 7.8 启动 admin API 和 `frontend/admin/`，通过页面读取和保存调度器配置。
