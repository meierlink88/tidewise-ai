## Context

当前 `tidewise-ai` 的后端已经具备 Go + Gin API/BFF 骨架、`local/uat/prod` 配置模板、健康检查、`repositories`、`integrations` 和 `jobs` 占位边界，但还没有数据库迁移、PostgreSQL schema、Redis 使用边界实现，也没有 `internal/ingestion` 采集层代码。

外部设计输入 `/Users/meierlink/Documents/股票趋势验证/theme-reasoning-er-diagram.md` 给出了本阶段数据模型和采集层边界：PostgreSQL 保存主数据、事件事实、证据、审核状态和回滚依据；图数据库只作为后续从 PostgreSQL 投影出的查询和可视化设施；采集层只把外部信息源转为可复核、可去重、可清洗的 `RAW_DOCUMENT`，不直接判断利好利空、影响方向、主题贡献、传导强度或预测结论。

三个参考系统提供了不同采集经验：

- `Vibe-Research`：以 `news_sources.json` 和 `newsradar.py` 表达配置型 RSS/Atom 抓取、合规过滤、时间过滤和本地缓存。
- `Vibe-Trading`：以 `agent/backtest/loaders` 表达 loader registry、fallback chain、按 host 限流、RSSHub 路由、Tushare/AKShare SDK、Eastmoney/Yahoo/SEC 等外部通道。
- `Stock`：以脚本形式表达 Eastmoney 行情和新闻抓取、RSS 抓取、网页正文抓取、本地 CSV/JSON 文件输出和概念板块抓取。

本 change 要把这些经验收敛成当前 Go 后端的正式数据库和采集层基础，而不是复制历史系统的 Python 脚本、模拟数据或业务推理逻辑。

## Goals / Non-Goals

**Goals:**

- 建立 PostgreSQL migration 来源，覆盖 ER 设计中的实体、关系、采集源、原始文档、事件、证据、标签和事件实体关联表。
- 建立 repo 内长期保留的版本化 SQL DDL/migration 来源，后续任何数据库结构变更都必须通过增量 migration 表达。
- 建立后端 `internal/ingestion` 代码骨架，承载采集编排、清洗、标准化、去重和写入流程。
- 建立采集源目录、连接器注册、解析器注册、凭证解析、限流、原始对象保存和原始文档写入的可测试接口。
- 第一阶段实现 Go 可直接承载的通道边界：`rss_feed`、`http_eastmoney`、`rsshub_feed`、`web_fetch`、`local_file`。
- 为 `sdk_tushare` 和 `sdk_akshare` 建立配置和接口边界，明确后续通过 Python worker、sidecar 或内部 HTTP wrapper 接入。
- 保持采集层只写 `RAW_DOCUMENT` 和基础采集状态，不把情绪评分、影响方向、传导强度、预测结论写入本阶段数据模型。
- 保证真实密钥不进入 repo、不进入 PostgreSQL，只通过环境变量或部署平台 secret 注入，数据库中最多保存 `credential_ref`。

**Non-Goals:**

- 不实现前端页面，不修改 `frontend/miniapp/src`。
- 不实现 AI 事件抽取、实体链接自动标注、Agent 推理、RAG、报告生成或投资观点生成。
- 不引入独立图数据库或向量数据库；本阶段只保留未来投影边界。
- 不把 Tushare、AKShare Python SDK 直接嵌入 Go 主服务运行时。
- 不接入真实生产 token、cookie、API key、数据库密码或云厂商密钥。
- 不复制 Stock 中的示例新闻、模拟数据或历史输出文件作为生产种子数据。
- 不修改 `/Users/meierlink/Documents/股票趋势验证/*` 下的参考系统源码。

## Decisions

### Decision: PostgreSQL 作为本阶段事实数据主存储

本阶段通过迁移创建 ER 设计中的核心表：

- `entity_nodes`
- `entity_edges`
- `economy_profiles`
- `policy_body_profiles`
- `market_profiles`
- `index_profiles`
- `sector_profiles`
- `chain_node_profiles`
- `company_profiles`
- `security_profiles`
- `instrument_profiles`
- `metric_profiles`
- `commodity_profiles`
- `person_profiles`
- `source_catalogs`
- `raw_documents`
- `events`
- `event_sources`
- `event_tag_defs`
- `event_tag_maps`
- `event_entity_links`

Schema field mapping 以外部 ER 文档为来源，在本 change 内固化为 migration 实现基线。字段名采用 PostgreSQL snake_case。基础类型约定为：`uuid` 映射 PostgreSQL `uuid`，`string` 映射 `text` 或受限 `varchar`，`boolean` 映射 `boolean`，`int` 映射 `integer`，`date` 映射 `date`，`datetime` 映射 `timestamptz`。实现可以补充 `created_at`、`updated_at` 等工程审计字段，但不得删除下列 ER 核心字段。

| 表 | 核心字段 |
|---|---|
| `entity_nodes` | `id` PK, `entity_type`, `layer_code`, `name`, `canonical_name`, `aliases`, `status` |
| `entity_edges` | `id` PK, `from_entity_id` FK -> `entity_nodes.id`, `to_entity_id` FK -> `entity_nodes.id`, `relation_type`, `evidence_note`, `status` |
| `economy_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `country_code`, `currency_code`, `region` |
| `policy_body_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `body_type`, `jurisdiction`, `policy_domain` |
| `market_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `market_type`, `economy_entity_id` FK -> `entity_nodes.id`, `currency_code`, `timezone` |
| `index_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `index_code`, `index_type`, `market_entity_id` FK -> `entity_nodes.id`, `provider`, `currency_code`, `list_date` |
| `sector_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `sector_system`, `sector_code`, `sector_type`, `exchange_scope`, `constituent_count`, `list_date`, `parent_sector_entity_id` FK -> `entity_nodes.id` |
| `chain_node_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `chain_position` |
| `company_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `registration_economy_entity_id` FK -> `entity_nodes.id`, `area`, `industry_name`, `controller_name`, `controller_type` |
| `security_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `ticker`, `symbol`, `exchange`, `market_board`, `security_type`, `issuer_company_entity_id` FK -> `entity_nodes.id`, `list_date`, `delist_date`, `list_status`, `currency_code` |
| `instrument_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `instrument_type`, `underlying_entity_id` FK -> `entity_nodes.id`, `exchange`, `currency_code` |
| `metric_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `metric_type`, `unit`, `frequency` |
| `commodity_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `commodity_type` |
| `person_profiles` | `entity_id` PK/FK -> `entity_nodes.id`, `role_title`, `organization_entity_id` FK -> `entity_nodes.id`, `economy_entity_id` FK -> `entity_nodes.id` |
| `source_catalogs` | `id` PK, `ingest_channel`, `provider_key`, `connector_key`, `parser_key`, `source_type`, `source_name`, `source_url`, `source_level`, `topic_hint`, `route_template`, `code_style`, `auth_required`, `auth_type`, `credential_ref`, `rate_limit_policy`, `usage_policy`, `status` |
| `raw_documents` | `id` PK, `source_id` FK -> `source_catalogs.id`, `ingest_channel`, `source_type`, `source_name`, `source_url`, `source_external_id`, `title`, `content_text`, `raw_object_uri`, `raw_mime_type`, `language`, `published_at`, `collected_at`, `content_hash`, `ingest_status` |
| `events` | `id` PK, `title`, `summary`, `event_time`, `first_seen_at`, `knowable_at`, `event_status`, `fact_status`, `dedupe_key`, `primary_source_id` FK -> `event_sources.id` |
| `event_sources` | `id` PK, `event_id` FK -> `events.id`, `raw_document_id` FK -> `raw_documents.id`, `source_level`, `evidence_excerpt`, `evidence_hash` |
| `event_tag_defs` | `id` PK, `tag_kind`, `code`, `name` |
| `event_tag_maps` | `id` PK, `event_id` FK -> `events.id`, `tag_id` FK -> `event_tag_defs.id`, `assign_source`, `review_status` |
| `event_entity_links` | `id` PK, `event_id` FK -> `events.id`, `entity_id` FK -> `entity_nodes.id`, `entity_role`, `assign_source`, `review_status`, `evidence_note` |

关键约束和索引方向：

- 各 profile 表的 `entity_id` 必须同时作为主键和外键，表达一实体一扩展 profile。
- `entity_edges` 需要支持按 `from_entity_id`、`to_entity_id` 和 `relation_type` 查询。
- `source_catalogs` 需要支持按 `status`、`provider_key`、`ingest_channel`、`connector_key` 查询。
- `raw_documents` 需要支持按 `source_id`、`source_external_id`、`content_hash`、`published_at`、`collected_at` 和 `ingest_status` 查询；幂等约束优先围绕 `source_id + source_external_id` 和 `source_id + content_hash` 设计。
- `events` 需要支持按 `dedupe_key`、`event_time`、`first_seen_at`、`knowable_at`、`event_status`、`fact_status` 查询。
- `event_sources` 需要支持按 `event_id`、`raw_document_id` 和 `evidence_hash` 查询。
- `event_tag_maps` 和 `event_entity_links` 不放样例数据，但需要支持后续抽取、审核和回滚。
- `events.primary_source_id` 与 `event_sources.event_id` 存在插入顺序问题，migration 或 repository 实现需要通过可空字段、延迟更新或事务内两阶段写入处理。

选择理由：

- 主规格已经明确 MVP 阶段 PostgreSQL 是结构化主存储。
- ER 文档明确图数据库只接收 PostgreSQL 正式数据投影。
- 事件事实、证据和审核状态需要可回滚、可审计和可迁移的关系型存储。

备选方案：

- 直接引入图数据库：路径查询更自然，但会提前增加部署和一致性复杂度。
- 先只写 JSON 文件：开发快，但无法支撑后续正式 API、审计、幂等和回滚。

### Decision: 版本化 SQL migration 是 schema 变更唯一来源

本项目不采用运行时根据 Go struct 或 ORM 自动推断数据库结构的方式。数据库表、索引、约束、枚举检查和后续结构调整必须保存在 repo 内的版本化 SQL migration 文件中，并作为 PostgreSQL schema 演进的唯一来源。

默认实现策略：

- migration 目录使用 `backend/migrations/`。
- Go 迁移工具优先采用 `pressly/goose`。
- migration 文件使用 `000001_xxx.sql` 这类递增版本命名，并保留 `-- +goose Up` 和 `-- +goose Down` 段。
- 已经在共享环境或生产环境执行过的 migration 文件不得被重写来表达新 schema；后续变化必须追加新版本 migration。
- `goose_db_version` 或等价版本表记录已执行版本，后端启动时根据版本表检测 pending migrations。
- local 和 uat 默认允许启动时自动执行 pending migrations；prod 也使用同一套迁移文件和检查逻辑，但是否自动执行由部署配置显式控制。
- 启动迁移执行前必须获取 PostgreSQL advisory lock 或迁移工具等价锁，避免多实例同时改 schema。
- 自动迁移只允许执行可审阅的增量 DDL，不允许通过清空数据、重建全库或无保护删除字段来完成升级。

数据保留原则：

- 允许的常规增量包括新增表、新增可空字段、新增带默认值字段、新增索引、新增非破坏性约束和兼容性回填。
- 删除表、删除字段、重命名字段、收紧非空约束、大规模类型变更等破坏性操作必须拆成独立 OpenSpec change，并使用 expand-contract、数据回填、兼容窗口和人工确认。
- 在已有数据环境中执行 migration 时，失败必须中止启动或部署，不得自动 fallback 到清库重建。

选择理由：

- Go 生态更成熟的生产方式是显式 SQL migration，而不是 Java ORM 风格的 `ddl-auto update`。
- 版本化 DDL 能让 Codex、人工 reviewer 和部署流程看到真实 schema 变化。
- 启动检查可以降低本地和测试环境漏跑 migration 的风险，同时通过配置控制生产环境风险。

备选方案：

- ORM 自动迁移：开发早期方便，但容易产生不可审阅、不可回滚或环境不一致的 schema 变化。
- 只提供手工 SQL 文档：可审阅但容易漏执行，无法和启动检查、版本表、CI 验证形成闭环。
- Atlas 或 golang-migrate：也可满足版本化迁移需求，但本阶段 `goose` 的 SQL 文件和 Go 内嵌调用更直接，AI 编码歧义更低。

### Decision: 采集层先以 `RAW_DOCUMENT` 为边界

采集层输出统一原始文档模型，包含来源、外部 ID、标题、正文、原始对象 URI、语言、发布时间、采集时间、内容哈希和状态。后续 AI 抽取再从 `RAW_DOCUMENT` 生成或更新 `EVENT`、`EVENT_SOURCE`、`EVENT_TAG_MAP` 和 `EVENT_ENTITY_LINK`。

选择理由：

- ER 文档明确 `RAW_DOCUMENT` 不直接关联实体，只通过 `EVENT_SOURCE` 作为事件事实证据。
- 采集层的职责是可复核、可去重、可清洗，不负责影响判断和推理结论。
- 把原始材料和事件事实分开，有利于后续人工审核、重新抽取和回滚。

备选方案：

- 采集时直接生成事件和实体关系：短期看似完整，但会混淆采集、抽取、审核和推理边界。
- 只保存事件摘要不保存原文：节省存储，但丢失证据链和复核能力。

### Decision: 连接器按通道和 provider 解耦

采集源目录中的 `ingest_channel` 表达底层技术通道，`provider_key` 表达数据提供方，`connector_key` 表达连接器实现，`parser_key` 表达内容解析方式。

第一阶段通道分组：

| 通道 | 本 change 处理方式 |
|---|---|
| `rss_feed` | Go 直接实现，吸收 Vibe-Research 的配置型 RSS 思路 |
| `http_eastmoney` | Go 直接实现，吸收 Vibe-Research/Vibe-Trading/Stock 的限流、UA、session、字段标准化经验 |
| `rsshub_feed` | Go 直接实现，吸收 Vibe-Trading 的 `route_template`、`code_style`、`RSSHUB_BASE_URL` 和 XML 安全解析思路 |
| `web_fetch` | Go 直接实现，保存 HTML/PDF/网页快照并提取可读正文 |
| `local_file` | Go 直接实现，用于 CSV/JSON/本地历史材料回灌 |
| `sdk_tushare` | 只定义接口和配置，后续通过 Python worker、sidecar 或内部 HTTP wrapper 接入 |
| `sdk_akshare` | 只定义接口和配置，后续通过 Python worker、sidecar 或内部 HTTP wrapper 接入 |

选择理由：

- Go 后端适合承载 HTTP/RSS/文件读取和统一写库。
- Tushare/AKShare 是 Python SDK，直接塞进 Go 主服务会破坏部署和测试边界。
- Vibe-Trading 的 loader registry 可借鉴，但其多数 loader 面向行情 K 线，不等同于事件原文采集。

备选方案：

- 所有采集都用 Python worker：能快速复用 SDK，但会绕开 Go 后端分层和当前配置体系。
- 所有采集都直接写具体 provider 代码：初期快，但后续 provider 和 parser 会失控。

### Decision: 限流和凭证作为基础设施边界

`rate_limiter` 按 `provider_key` 和 `rate_limit_policy` 执行进程内限流，后续可以接 Redis 做分布式限流。`credential_resolver` 根据 `credential_ref` 从环境变量或部署平台 secret 获取凭证，不在数据库或代码中保存真实值。

选择理由：

- Eastmoney 等公开接口存在 IP 限流和封禁风险。
- Tushare、Finnhub、Alpha Vantage 等授权 API 需要严格凭证边界。
- `SOURCE_CATALOG.credential_ref` 只表达引用，不泄露密钥。

备选方案：

- 每个 connector 自己 `sleep`：简单但不可统一测试和运维。
- 把 token 写进配置文件或数据库：方便调试但违反安全边界。

### Decision: 原始对象保存与结构化写入分离

`raw_object_store` 保存原始 HTML、JSON、PDF、CSV 或网页快照，返回 `raw_object_uri`；`raw_document_writer` 只负责根据 `source_external_id`、`content_hash` 和来源信息幂等写入 `RAW_DOCUMENT`。

选择理由：

- 原始对象可能很大，不适合全部塞进结构化表。
- `content_text` 服务全文检索、AI 抽取和证据匹配，`raw_object_uri` 服务审计和复核。
- 幂等写入可以避免重复采集造成事实污染。

备选方案：

- 只写 PostgreSQL 不保存原始对象：实现简单但审计能力不足。
- 原始对象和结构化记录混在一个模块：职责不清，不利于后续替换对象存储。

## Risks / Trade-offs

- [Risk] 第一阶段 schema 覆盖面较大，migration 容易变复杂。→ Mitigation：优先创建稳定基础表和索引，复杂图/向量能力延后，迁移保持可审阅，并通过版本表和启动检查保证环境一致。
- [Risk] 启动自动迁移在生产多实例环境中可能产生并发 DDL 或破坏性误操作。→ Mitigation：迁移执行必须加 PostgreSQL advisory lock，prod 自动执行由配置显式开启，破坏性 migration 必须独立 review 且默认禁止自动执行。
- [Risk] Eastmoney、RSSHub、网页抓取等公开源存在限流、反爬或路由不稳定。→ Mitigation：所有 provider 走统一限流、timeout、UA 和错误状态，不让单源失败中断整体采集批次。
- [Risk] Tushare/AKShare SDK 无法在 Go 主服务内直接运行。→ Mitigation：本 change 只定义 SDK connector 边界和配置，真实 Python worker 接入另开 change。
- [Risk] 采集层可能被误用为分析层。→ Mitigation：spec 明确禁止本阶段写入影响方向、评分、传导强度和预测结论。
- [Risk] 外部设计文档和参考系统路径不在 repo 内，未来可能移动。→ Mitigation：proposal/design 记录路径，后续如需长期保留应另行决定是否加入脱敏设计输入快照。
- [Risk] 真实凭证误提交。→ Mitigation：只提交 placeholder、环境变量名和 `credential_ref`，验证任务必须扫描 secret/token/password。

## Migration Plan

1. 创建 `backend/migrations/`，使用 `goose` 兼容的版本化 SQL migration 文件保留 DDL。
2. 创建核心 schema、索引、唯一约束和必要枚举检查，并通过版本表记录已执行 migration。
3. 扩展后端配置，加入采集、对象存储、限流和启动迁移检查相关非敏感配置。
4. 新增 `internal/ingestion`、`internal/domain`、`internal/repositories` 和 `internal/integrations` 中的采集相关接口和实现。
5. 在 API 启动流程中加入 migration checker/runner，按配置检查并执行 pending migration。
6. 实现 Go 可直接运行的第一批 connector 和 parser，并用测试覆盖标准化、幂等、限流和错误处理。
7. 保持 `sdk_tushare`、`sdk_akshare` 为声明式边界，不接真实 Python SDK。
8. 运行 `go test ./...`、`go vet ./...`、migration 静态/解析验证、`openspec validate init-database-and-ingestion-layer` 和 `openspec validate --all`。

回滚策略：

- schema 变更通过 down migration、兼容迁移或新版本修复 migration 回滚；不得通过清空数据或重建全库回滚。
- 采集层代码可以通过禁用 `source_catalogs.status` 或环境配置关闭。
- 原始对象保存失败不得写入成功状态的 `RAW_DOCUMENT`。

## Open Questions

- 对象存储第一阶段使用本地文件目录、S3 兼容 URI 还是仅接口和测试 fake，需要在 apply 前确认默认实现。
- `SOURCE_CATALOG` 是否需要在本 change 中提供最小 seed 数据，还是只提供 schema 和测试 fixture。
- Redis 限流是否在本 change 中真实接入，还是先提供进程内限流并保留 Redis 边界。
