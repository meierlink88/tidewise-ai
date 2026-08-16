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
- `000047_add_raw_evidence_categories.sql`：新增受控 Evidence Category 目录与 Raw Evidence 多标签关系，并初始化 11 个稳定内容分类。
- `000048_replace_alliance_with_organizations.sql`：协调式破坏性切换，创建独立 Organization、三类可维护目录、Domain Tag 与 Country 成员历史关系，扩展正式 Event Object 引用，并完整退役旧 `alliance_org` 身份及其依赖聚合。目录数据不在 migration 中 seed，需在迁移后运行 `organization-catalog-publish`；回滚必须恢复切换前 PostgreSQL 快照与上一版应用。
- `000049_retire_benchmark_metric_and_graph_projection.sql`：协调式破坏性切换，删除 Benchmark、Metric Entity 及其依赖的语义和研究聚合，移除其 profile/observation 表与遗留图投影账本。发布前必须停止写入并取得 PostgreSQL 快照；回滚必须恢复快照与上一版应用。
- `000050_unify_domain_object_ids.sql`：零兼容窗口身份切换，将 Entity、Entity Relation、Country、Region、Organization 与 Evidence 领域对象统一为“领域前缀 + canonical lowercase UUID”，并将 Country code 收敛为 ISO 3166-1 alpha-2。发布前必须停止写入并取得 PostgreSQL 快照；回滚必须恢复快照与上一版应用。
- `000051_enforce_business_primary_key_ids.sql`：把其余独立业务、关系和回执主键及传递外键统一为已注册前缀 ID，移除两个历史序列。
- `000052_migrate_embedded_business_ids.sql`：同步改写数组及研究回执 JSON map 中保存的业务 ID。
- `000053_unify_organization_evidence_primary_keys.sql`：为四张 Organization/Evidence 目录与关系表增加
  Data Service 生成的前缀 UUID `id` 主键，并将 Raw Evidence 与 Atomic Evidence 主键列改名为 `id`。

`000047` 对“目录数据独立发布”规则采用限域例外：Data Evidence 是 owner，且这 11 个固定
分类是本次 Raw Evidence API 与外键同时生效所必需的合同数据，因此随 additive schema
一次安装，范围不扩展到普通 seed、历史回填或可运营目录。该例外不引入 Secret 或外部数据；
由全账本 smoke 和真实 PostgreSQL API seam 校验。回滚使用 reviewed forward repair，未来若
分类获得独立管理生命周期，则由新的受控目录发布合同替代该例外。

`000047` 的发布顺序固定为“受控 migration → 校验目录→新 Data Service”。由于
UAT risk manifest 将它标记为 `mixed`，普通系统 Deploy 不得自动执行；操作员必须先
确认 PostgreSQL 恢复点，用本次候选 Data 镜像的正式 `/usr/local/bin/dbmigrate`
在独立、可审计的发布动作中先执行 check-only，确认 `000047` 是唯一 pending
Data migration 后执行 `-apply`。执行后必须确认 ledger 为 `47`、目录恰有
11 行且所有 description 非空，然后才允许普通 Deploy 发布新应用。旧应用不读写新表，
因此可与已应用的 `000047` 共存；新应用回退时保留 schema、目录和已发布关系，
不执行 down migration。如受控 migration 未通过，不发布新应用，并依恢复点或新的
reviewed forward repair 处理，不在原 migration 上就地修改。

Entity seed、关系包及其导入/投影执行能力已按
`docs/adr/0013-data-entity-domain-and-projection-retirement.md` 从 Data 退役；历史 PostgreSQL
schema migration 继续保留，但本目录不再声明 repo seed 或图投影验收合同。

`000049` 是零兼容窗口的协调式破坏性发布，已由 Issue #237 授权。操作员必须先停止所有
Data 写入者、确认 PostgreSQL 恢复点并以候选 Data 镜像执行正式 `dbmigrate` 的 check-only。
确认 `000049` 是唯一 pending migration 后才可执行 `-apply`；随后确认 ledger 为 `49`、五张
退役表均不存在且 `entity_nodes` 不含 `benchmark` 或 `metric`。只有这些确认完成后才可部署
不再识别两个 Entity Type 的应用版本。应用回退不得运行 down migration，必须将数据库快照与
上一版应用一同恢复，或执行经过审阅的前向修复。

`000050` 是 Issue #241 授权的零兼容窗口切换。操作员必须停止 Data 及所有
上游写入者，确认 PostgreSQL 恢复点，并用候选 Data 镜像执行 `dbmigrate`
check-only。只有在 `000050` 为唯一 pending migration 时才可人工确认并执行
`-apply`。执行后必须确认 ledger 为 `50`，相关主键与外键均满足领域前缀
UUID 合同，Country 为 201 条 ISO alpha-2 code，且无孤儿引用，才可发布新应用。
旧应用不兼容新身份；回退不得运行 down migration，必须同时恢复数据库快照与
上一版应用，或使用经审阅的前向修复 migration。

`000051`、`000052` 是 Issue #244 对同一停写窗口的扩展。check-only 必须确认它们按序
紧随 `000050`；执行后确认 ledger 为 `52`、独立业务主键均为注册前缀加 UUID、无默认值或
序列且外键无孤儿。应用与数据库必须一起发布；回退恢复迁移前快照和上一版应用。

`000053` 是 Issue #251 授权的零兼容窗口切换。本需求明确不迁移旧数据；迁移在需要
新 ID 的四张表存在行时 fail closed，不回填、不删除。执行后确认 ledger 为 `53`、六张目标表
的唯一主键列均为 `id`，且分别满足 `OCA`/`ODT`/`ODL`/`RAW`/`EVD`/`RCL` 前缀合同。

Data 不再提供 `source-seed` 或维护 `source_catalogs`，既有 Event Publication 仍只通过
`POST /api/data/v1/reviewed-event-imports` 连同轻量 Event 证据原子接纳；历史
`raw_documents.content_text` 继续可读，但 V2 新记录不保存正文。ADR-0011 另行建立与该旧链路
隔离的 Evidence 体系：完整材料进入 `raw_evidences`，不复用 `raw_documents` 或
`event_sources`。
