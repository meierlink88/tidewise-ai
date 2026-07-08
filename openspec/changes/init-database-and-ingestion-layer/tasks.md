## 1. Artifact Review

- [x] 1.1 审阅 `proposal.md`，确认本 change 范围是数据库构建和采集层编码，不实现前端页面、AI 推理、图数据库或向量数据库。
- [x] 1.2 审阅 `design.md`，确认 ER 字段级 schema mapping、三个参考系统采集通道、SDK 边界、限流、凭证、TDD 测试先行和原始文档职责符合当前架构。
- [x] 1.3 审阅 `specs/event-knowledge-schema/spec.md` 和 `specs/data-ingestion-layer/spec.md`，确认新增 capability 能约束后续实现。
- [x] 1.4 审阅 `specs/backend-foundation/spec.md`、`specs/persistence-and-contracts/spec.md` 和 `specs/technical-architecture/spec.md`，确认 delta requirements 不冲突。
- [x] 1.5 确认 apply 阶段执行顺序为先写 Go 单元测试或 fixture，再写生产实现，最后运行对应包测试和全量测试。

## 2. Database Migration Foundation

- [x] 2.1 在 `backend/migrations/` 建立 `goose` 兼容的版本化 SQL migration 目录，并确认 DDL 文件作为 schema 演进唯一来源。
- [x] 2.2 先编写 migration 文件解析、版本顺序和关键 DDL 结构测试，再新增 PostgreSQL schema migration。
- [x] 2.3 为 `Schema field mapping` 中的关键字段添加主键、外键、唯一约束、状态字段、时间字段、内容哈希索引和常用查询索引。
- [x] 2.4 提供 down migration、兼容回滚说明或可审阅的回滚策略，并禁止通过清空数据或重建全库完成升级和回滚。
- [x] 2.5 先编写启动迁移检查/runner 的单元测试，覆盖无 pending migration、有 pending migration、禁用自动执行、迁移失败和并发锁场景。
- [x] 2.6 实现启动迁移检查/runner 边界，能够读取已执行版本、检测 pending migrations，并按配置执行非破坏性增量迁移。

## 3. Backend Config And Domain Models

- [x] 3.1 扩展 `backend/internal/config` 和 `backend/config/config.*.yaml`，加入采集、对象存储、限流和迁移相关非敏感配置，包括是否允许启动自动执行 pending migration。
- [x] 3.2 保持真实数据库密码、API key、token、cookie、RSSHub base URL 密钥和云服务密钥只通过环境变量或部署平台 secret 注入。
- [x] 3.3 先编写配置校验和领域模型测试，覆盖缺失配置、非法状态、时间字段、哈希字段和敏感配置不落盘。
- [x] 3.4 新增领域模型和值对象，按 `Schema field mapping` 表达实体、采集源、原始文档、事件、事件证据、标签和事件实体关联。

## 4. Repository Layer

- [x] 4.1 先编写 repository 接口行为测试或 fake 测试，覆盖 `source_catalogs` 读取、`raw_documents` 幂等写入、重复数据和采集状态更新。
- [x] 4.2 实现基于 `source_id`、`source_external_id` 和 `content_hash` 的幂等写入策略。
- [x] 4.3 添加 repository 测试或可替换 fake，确保 ingestion 层不直接执行 SQL。
- [x] 4.4 确认 handler、main 入口和前端代码不直接访问数据库。

## 5. Ingestion Core

- [x] 5.1 先为 `backend/internal/ingestion` 编写 registry、parser、credential resolver、rate limiter、raw object store 和 raw document writer 的单元测试。
- [x] 5.2 实现 `source_registry`，按 `status`、`provider_key`、`ingest_channel` 选择采集源。
- [x] 5.3 实现 `connector_registry` 和 `parser_registry`，支持连接器、解析器注册、查找和未注册错误处理。
- [x] 5.4 实现 `credential_resolver`，根据 `credential_ref` 解析环境变量或 secret 引用，但不打印真实凭证。
- [x] 5.5 实现 `rate_limiter`，按 `provider_key` 和 `rate_limit_policy` 执行进程内限流，并保留 Redis 分布式限流扩展点。
- [x] 5.6 实现 `raw_object_store` 接口和本地开发默认实现，保存原始 HTML、JSON、XML、PDF、CSV 或文本对象并返回 `raw_object_uri`。
- [x] 5.7 实现 `raw_document_writer`，统一处理清洗正文、内容哈希、幂等写入、状态和错误记录。

## 6. First Connector And Parser Set

- [x] 6.1 先为每个 connector 和 parser 准备 fixture 与单元测试，不访问真实外部站点。
- [x] 6.2 实现 `rss_feed` connector 和 `rss_item` parser，覆盖标准 RSS/Atom 标题、链接、摘要、发布时间和正文候选内容。
- [x] 6.3 实现 `http_eastmoney` connector 和 Eastmoney JSON parser，统一 User-Agent、timeout、限流和错误处理。
- [x] 6.4 实现 `rsshub_feed` connector，支持 `RSSHUB_BASE_URL`、`route_template`、`code_style`、timeout 和安全 XML 解析。
- [x] 6.5 实现 `web_fetch` connector 和 HTML/text parser，保存原始对象并提取可读正文。
- [x] 6.6 实现 `local_file` connector 和 CSV/JSON/text parser，用于本地历史材料回灌。
- [x] 6.7 为 `sdk_tushare` 和 `sdk_akshare` 建立声明式 connector stub，明确需要后续 Python worker、sidecar 或内部 HTTP wrapper 执行真实 SDK。

## 7. Jobs And Operational Boundary

- [ ] 7.1 在 `backend/internal/jobs` 中新增采集任务入口或任务接口，用于按采集源批量执行 ingestion。
- [ ] 7.2 添加采集任务错误分类、重试建议和部分失败处理，确保单一来源失败不阻断整批任务。
- [ ] 7.3 确认采集层不生成影响方向、评分、传导强度、预测结论或投资建议。
- [ ] 7.4 添加最小 seed 或 fixture 策略，用于本地验证采集源目录和连接器注册，不包含真实密钥。

## 8. Validation

- [ ] 8.1 运行 `openspec validate init-database-and-ingestion-layer`，修复所有 OpenSpec 错误。
- [ ] 8.2 在 `backend/` 下运行 `go test ./...`。
- [ ] 8.3 在 `backend/` 下运行 `go vet ./...`。
- [ ] 8.4 运行迁移解析、版本顺序、schema 或 SQL 静态验证命令，确认 migration 不依赖清空数据或重建全库。
- [ ] 8.5 扫描本 change 和新增代码，确认没有真实 secret、token、cookie、数据库密码、API key 或生产连接串。
- [ ] 8.6 运行 `openspec validate --all`，确认本 change 与主规格关系可验证。
- [ ] 8.7 运行 `git status --short`，确认提交范围只包含本 change 的 OpenSpec artifacts 和预期后端实现文件。
- [ ] 8.8 确认所有新增后端功能点都有对应 Go 单元测试或明确标记的集成测试，且测试全部通过后才进入 sync/archive。
