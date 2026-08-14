# Database Migrations

本目录保存 PostgreSQL schema 的版本化 DDL，是数据库结构演进的工程来源。

规则：

- migration 文件使用 Goose 兼容 SQL，文件名格式为 `000001_name.sql`。
- 已在共享环境或生产环境执行过的 migration 不得重写；后续 schema 变化必须追加新版本 migration。
- migration 必须保留 `-- +goose Up` 和 `-- +goose Down` 段。
- 自动升级只能执行可审阅的增量 DDL，不得通过清空表、重建全库或丢弃业务数据完成升级。
- 破坏性结构调整必须拆成独立、可审阅的开发任务，并包含兼容窗口、数据回填和人工确认。

初始 schema 的 down 段不自动删除业务表。需要回滚初始 schema 时，应通过已审阅的前向修复 migration 或数据库备份恢复执行，避免在有数据环境中误删事实基础。

本地 migration 只从 Data image 运行。正常 Local Compose 启动会先自动执行同一 one-shot
service；也可以显式运行：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm data-migrate
```

只检查 pending migration 时不加 `-apply`。

当前实体基础库相关 migration：

- `000001_init_event_knowledge_schema.sql`：创建事件知识 schema、实体节点、实体关系和各类 profile 表。
- `000002_add_alliance_org_profiles.sql`：补充联盟组织 profile 表。
- `000003_add_sector_seed_snapshot_fields.sql`：补充板块初始化热度快照字段 `rank_snapshot` 和 `snapshot_date`。
- `000004_add_source_catalog_source_config.sql`：历史上为 `source_catalogs` 补充 `source_config`；相关 Source 控制面已由 `000029` 退出 Data。
- `000005_add_ingestion_scheduler.sql`：历史上创建 Data 采集调度与执行表；相关控制面已由 `000029` 退出 Data。
- `000006_add_graph_projection_runs.sql`：补充 `entity_nodes.entity_key`，创建 Neo4j 图谱投影 run 和明细记录表。
- `000015_refactor_industry_chain_node_phase_a.sql`：以人工授权门禁执行旧产业结构受控 cleanup，并收敛最小 chain_node/theme profile。
- `000016_add_entity_external_identifiers.sql`：新增通用实体外部标识表、外部 identity 唯一约束与实体侧查询索引；不包含任何 mapping 数据。
- `000029_add_event_publication_v2.sql`：保留历史 Event/证据正文与关联，把新 `raw_documents` 收敛为轻量 Event 证据记录，新增 V2 Receipt，并退出 Data 内旧 Source、采集运行及 V1 Import Receipt 结构。
- `000042_add_evidence_publications.sql`：新增与 Event Publication 隔离的 `raw_evidences`、原子 `evidences` 和两类不可变 Receipt；完整原文与 Keywords 先发布，清洗后的完整 `1..N` Evidence 集合后发布。
- `000043_add_evidence_creation_times.sql`：为新建 Raw Evidence 和 Atomic Evidence 增加 Data 数据库生成的内部创建时间；历史行保持空值且不回填。
- `000044_remove_evidence_publication_receipts.sql`：历史上物理删除两张 Evidence 发布回执表及其专用不可变函数；该一次性清理入口已退役，migration ledger 继续保留。
- `000045_add_regions_and_remove_entity_type_definitions.sql`：Tidewise AI 2.0 一次性切换，删除数据库 Entity Type Definition，并创建独立 `regions` 事实表与 `region_type` 枚举。该 migration 为 forward-only；回滚必须恢复切换前 PostgreSQL 快照，不运行 down migration。
- `000046_replace_economy_with_countries.sql`：Tidewise AI 2.0 一次性破坏性切换，创建独立 `countries` 与 `country_region_links`，把活动 Country 引用改为稳定 Country ID，清除指向旧 Economy 身份的 Entity/Event 关系，并删除 `economy_profiles` 与 Economy Entity 行。该版本经 ADR-0017 明确豁免兼容窗口；发布前停止写入并取得 PostgreSQL 快照，回滚时必须同时恢复快照和上一版应用。

Entity seed、关系包及其导入/投影执行能力已按
`docs/adr/0013-data-entity-domain-and-projection-retirement.md` 从 Data 退役；历史 PostgreSQL
schema migration 继续保留，但本目录不再声明 repo seed 或图投影验收合同。

Data 不再提供 `source-seed` 或维护 `source_catalogs`，既有 Event Publication 仍只通过
`POST /api/data/v1/reviewed-event-imports` 连同轻量 Event 证据原子接纳；历史
`raw_documents.content_text` 继续可读，但 V2 新记录不保存正文。ADR-0011 另行建立与该旧链路
隔离的 Evidence 体系：完整材料进入 `raw_evidences`，不复用 `raw_documents` 或
`event_sources`。
