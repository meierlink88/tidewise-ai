## 1. 采集源清单和持久化设计

- [x] 1.1 为 `source_config` migration 编写静态测试，验证 `source_catalogs` 增量字段、默认值和非破坏性迁移要求。
- [x] 1.2 新增 `source_config` 的 PostgreSQL migration。
- [x] 1.3 为 `SourceCatalog` 模型和 repository 编写测试，覆盖 `source_config` seed、scan、upsert 和空配置默认值。
- [x] 1.4 扩展 `SourceCatalog`、`PostgresRepository` 和内存 repository，使来源配置可以完整读写。
- [x] 1.5 为来源目录统计编写 repository/service 测试，覆盖按 provider、通道、来源类型、用途和状态统计。
- [x] 1.6 实现来源目录统计能力，用于 seed report 和后续管理查询。

## 2. 非 SDK 来源 seed 数据

- [x] 2.1 设计 repo 内来源清单格式，并编写 loader/validator 测试，覆盖必填字段、重复 ID、无效 URL、无效 connector/parser、敏感字段禁止写入。
- [x] 2.2 新增 Vibe-Research 内容来源清单，导入 `backend/news_sources.json` 中 108 条 RSS 配置，并在测试中验证 URL 去重后为 106 个唯一 URL。
- [x] 2.3 新增 Vibe-Trading 行情/provider 来源清单，导入 `registry.py` 中排除 `auto` 和 SDK-only loader 后的非 SDK loader source，并在清单中说明排除口径。
- [x] 2.4 新增 Stock 非 SDK 来源清单，导入新闻网页、东方财富股票/指数/板块、本地历史文件等可治理来源，并在清单中说明实际数量和统计口径。
- [x] 2.5 新增内容/事件类来源清单分组，覆盖 RSS、网页新闻、RSSHub route 和本地历史材料回灌。
- [x] 2.6 新增行情类来源清单分组，覆盖 Eastmoney、Sina、Tencent、Yahoo、Stooq、Finnhub、FMP、Tiingo、Alpha Vantage、OKX、CCXT 等 provider 或 endpoint 元信息。
- [x] 2.7 新增板块类来源清单分组，覆盖 Stock 的 Eastmoney 概念板块、预定义重点板块和指数来源元信息。
- [x] 2.8 新增本地回灌类来源清单分组，表达 CSV、JSON、文本等历史材料导入来源。
- [x] 2.9 实现来源清单 loader 和 validator。
- [x] 2.10 为 seed service 编写 fake repository 测试，覆盖幂等 upsert、错误中断、统计 report、禁用来源保留和多类型来源分类。
- [x] 2.11 实现 seed service，将来源清单写入 `source_catalogs` 并输出分类统计。
- [x] 2.12 新增 `cmd/source-seed` 或等价命令入口，支持读取默认清单并写入当前环境 PostgreSQL。
- [x] 2.13 为 seed report 编写测试，验证输出 Vibe-Research 108/106、Vibe-Trading 非 SDK loader source 实际数量、Stock 非 SDK 来源实际数量、总来源数量、类型分布、provider 分布、状态分布和 SDK 排除口径。

## 3. 内容和 HTTP connector 边界

- [x] 3.1 为内容来源 connector/parser 编写 fixture 测试，覆盖 RSS、RSSHub、网页文本和本地文件来源。
- [x] 3.2 确认或补齐内容来源 connector/parser，使内容类来源可以真实进入 `raw_documents`。
- [x] 3.3 为 Eastmoney HTTP 行情和板块 connector/parser 编写 fixture 测试，覆盖字段映射、代码列表、空响应、限流错误和解析失败。
- [x] 3.4 实现 Eastmoney HTTP 行情和板块 connector/parser 的最小可运行边界，优先覆盖 Stock 已验证的股票列表、指数、概念板块和板块 K 线来源。

## 4. 多来源并发采集

- [x] 4.1 为 `IngestionJob` 编写并发测试，使用 fake connector/parser/writer 验证多个来源可以并发执行。
- [x] 4.2 为单源失败隔离编写测试，验证一个来源失败不会阻断其他来源，并正确汇总 report。
- [x] 4.3 为 provider 限流边界编写测试，验证并发 worker 仍按 `provider_key` 调用统一 rate limiter。
- [x] 4.4 扩展 `IngestionJob` 配置，增加可配置并发数、来源类型过滤和固定 worker pool。
- [x] 4.5 更新 smoke runner 或任务入口，使默认并发保持保守值，且可以在测试中设置为 1 保持串行兼容。

## 5. 真实本地验证和文档

- [x] 5.1 更新本地数据库说明，补充 migration、source seed、来源统计和多来源采集 smoke 的运行命令。
- [x] 5.2 运行 `go test ./...`，确保单元测试和 gated 集成测试边界保持通过。
- [x] 5.3 运行 `openspec validate seed-researched-source-catalogs`。
- [x] 5.4 在本地 PostgreSQL 执行 migration 和 source seed，验证 `source_catalogs` 中新增全部调研来源数量、类型分布、用途、状态和关键字段。
- [x] 5.5 验证本地 seed report 明确输出 Vibe-Research 108 条配置/106 个唯一 URL、Vibe-Trading 非 SDK loader source 实际数量、Stock 非 SDK 来源实际数量、总来源数量及 SDK 排除口径说明。
- [x] 5.6 使用少量内容来源执行显式 smoke，验证多来源 report、写入数量、重复写入和错误输出符合预期；Eastmoney HTTP 来源已完成 connector/parser fixture 验证，真实公网端点在当前网络返回空响应，记录为外部连通性阻塞并交由后续调度器和来源健康检查 change 继续验证。
