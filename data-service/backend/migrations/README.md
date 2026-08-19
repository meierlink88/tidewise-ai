# Database Migrations

本目录保存 PostgreSQL schema 的版本化 DDL，是数据库结构演进的工程来源。

规则：

- migration 文件使用 Goose 兼容 SQL，文件名格式为 `000001_name.sql`。
- 已在共享环境或生产环境执行过的 migration 不得重写；后续 schema 变化必须追加新版本 migration。
- migration 必须保留 `-- +goose Up` 和 `-- +goose Down` 段。
- 自动升级只能执行可审阅的增量 DDL，不得通过清空表、重建全库或丢弃业务数据完成升级。
- 破坏性结构调整必须拆成独立、可审阅的开发任务，并包含兼容窗口、数据回填和人工确认。

初始 schema 的 down 段不自动删除业务表。需要回滚初始 schema 时，应通过已审阅的前向修复 migration 或数据库备份恢复执行，避免在有数据环境中误删事实基础。

本地 migration 只从 Data image 运行。规范 npm 启动命令会先构建候选 Data image，再通过
临时 Compose run 执行 migration，成功后才启动 Service；也可以显式运行同一命令：

```bash
npm run runtime:migrate:data
```

只检查 pending migration 时，显式运行同一模板并以 `--` 覆盖默认的 `-apply`：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm --build data-migrate --
```

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
- `000054_add_organization_function_primary_id.sql`：为 Organization Function 目录补充 Data Service
  生成的 `OFN` 前缀 UUID `id` 主键，并将 `code` 保留为唯一业务键。
- `000055_simplify_atomic_evidence_semantics.sql`：零兼容删除历史 Atomic Evidence，删除顺序、
  双层 Source 与 Expression 身份列，并以独立 `summary` 和严格单层 5W1H `semantic JSONB`
  建立当前 Evidence 合同；Raw Evidence 保留。
- `000056_make_industry_concept_independent.sql`：将 profiled Industry/Concept 无损迁移为独立
  `industry`/`concept` 事实表，回填名称和别名，移除退役分类/边界字段，保留全部 object-aware
  引用后删除对应 shadow Entity；无 profile 的 legacy Entity 不变。
- `000057_make_chain_node_industry_chain_independent.sql`：将 profiled ChainNode/IndustryChain
  无损迁移为独立 `chain_node`/`industry_chain` 事实表，保留全部图谱、Research 与多态引用。
- `000058_assign_independent_object_id_prefixes.sql`：保留四类独立对象的 canonical UUID 后缀，
  将 Industry、Concept、ChainNode、IndustryChain 的历史 `ENT` 前缀分别切换为
  `IND`、`CON`、`CND`、`ICH`，并同步改写全部直接、多态和 JSONB 引用。
- `000059_retire_data_event_semantics.sql`：删除 Data-owned Event Semantic/Variable Signal
  能力和持久化，并在语义表之前删除依赖它们的 formal Research 行；保留
  `analyst_snapshot` Research 及 Atomic Evidence `semantic`。
- `000060_rebuild_event_domain.sql`：Issue #277 授权的零兼容切换，删除旧 Event 与
  依赖 Research 事实，退役轻量 Raw Document、Event Source、Event Tag 和 Publication
  Receipt，并围绕 Event、Atomic Evidence Link、Actor Link 和 Asset Link 重建持久化。
- `000061_add_sources.sql`：创建当前 Data-owned `sources` 表、固定身份保护、单一 active
  web-search 约束和稳定读取索引。它不复用已退役 `source_catalogs` 语义，不 seed
  也不从 AgentOS 数据库读取事实。
- `000062_add_subdivisions.sql`：新增严格从属于 Country 的独立 `subdivisions` 事实表、
  `SUB` 身份约束、Country 内 local code 组合唯一与四值 PostgreSQL 原生类型；不 seed、
  不建立 Region 或 Organization 关系，也不增加运行时 API wiring。

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

`000054` 是 Issue #253 对 Organization Function 遗漏身份的 forward-only 修正。迁移不回填、
不删除数据；`organizations`、Organization Domain Tag Link、Domain Tag 或 Function 任一非空时
均 fail closed。发布前停止 Data 写入并确认 PostgreSQL 恢复点；先确认 Organization 事实和
Domain Tag Link 为空，再由经审阅的操作按外键依赖顺序清空 Domain Tag、Function 两类可重复
发布目录，Category 不需清空。若环境已有 Organization 事实，本次无回填合同不允许直接升级；
必须恢复/重建经审阅的空 Organization 环境，或另行设计迁移，不能由部署脚本删除事实。随后以
候选镜像执行 check-only 并确认 `000054` 是唯一 pending migration，再执行 `-apply`。迁移后运行
`organization-catalog-publish`，确认 ledger 为 `54`、7 条 Function 均满足 `OFN` 前缀合同，
且 Category、Function、Domain Tag 分别为 4、7、21 条，然后部署新应用。旧应用不兼容必填
Function `id`；回滚必须同时恢复切换前快照和上一版应用，不运行 down migration。

`000055` 是 Issue #255 授权的零兼容破坏性切换，也是“schema migration 不清理事实”的限域
例外：旧双层 Evidence 无法无歧义转换为新的单层 5W1H 合同，需求明确要求全部删除且不保留。
操作员必须先停止 Data Evidence 写入者、确认 PostgreSQL 恢复点，并以候选 Data 镜像执行
`dbmigrate` check-only；只有确认 `000055` 是唯一 pending migration 后才执行 `-apply`。
迁移后确认 ledger 为 `55`、`evidences` 为空、`raw_evidences` 保持原数量、Evidence 表只保留
正式身份、Raw Evidence 外键、`is_split`、`summary`、`semantic` 和 `created_at`，随后一起发布
新 Data Service 与新发布方。旧应用和旧 payload 不兼容；回滚必须同时恢复切换前快照与
上一版应用，不运行 down migration。

`000056` 是 Issue #258 授权的零兼容协调切换。操作员必须停止 Data 及上游写入者、确认
PostgreSQL 恢复点，并用候选 Data 镜像执行 check-only；只有确认 `000056` 是唯一 pending
migration 后才执行 `-apply`。执行后必须确认 ledger 为 `56`，`industry_profiles` 与
`concept_profiles` 不存在，独立 `industry`/`concept` 的 ID、name、aliases、保留字段、时间戳
及映射/外部标识/Redirect/Event Semantic 引用与迁移前完全一致，并确认已迁移对象在
`entity_nodes` 中没有 shadow row。无法表达的旧状态、去除分类版本后的重复代码或未支持的
typed Entity 引用会在删除前 fail closed。旧应用不兼容新表；回滚必须同时恢复迁移前快照和
上一版应用，不运行 down migration。

`000058` 是 Issue #265 授权的零兼容身份切换。操作员必须停止 Data 及全部上游写入者、
确认 PostgreSQL 恢复点，并用候选镜像 check-only；只有确认 `000058` 是唯一 pending
migration 后才执行 apply。执行后确认 ledger 为 `58`，四类 owner 表分别只含
`IND`/`CON`/`CND`/`ICH`，所有 UUID 后缀、事实数量和内容保持一致，直接外键、多态引用、
Research 与 JSONB 回执无旧身份或孤儿。旧应用不兼容新前缀；回滚必须同时恢复 migration
58 前快照和上一版应用，不运行 down migration。

`000059` 是 Issue #267 授权的零兼容破坏性切换。操作员必须停止 Data 与相关上游写入者、
确认 PostgreSQL 恢复点，并用候选镜像 check-only；只有确认 `000059` 是唯一 pending
migration 后才执行 apply。执行后确认 ledger 为 `59`，所有 Event Semantic、Variable
Signal、Direct Impact 与相关 definition/policy/catalog 表均不存在，formal/lineage Research
行已删除，既有 `analyst_snapshot` 可重放、读取且引用完整，Atomic Evidence `semantic`
保持原合同。旧应用不兼容新 schema；回滚必须同时恢复 migration 59 前快照和上一版应用，
不运行 down migration。

`000060` 是 Issue #277 授权的零兼容破坏性切换，也是“migration 不删除业务事实”
规则的限域例外。旧 Event 无法在不编造 semantic、modality 和 lifecycle 的前提下转换，
需求明确要求删除且不保留。操作员必须停止 Data 及直接依赖写入者，确认 PostgreSQL
恢复点，并以候选镜像 check-only；只有确认 `000060` 是唯一 pending migration 后才通过
`data_60_cutover` 执行 apply。执行后确认 ledger 为 `60`，旧五类表不存在，新四表的列、
约束与前缀身份正确，且旧 Event 与依赖 Research 数据为空。旧应用不兼容新 schema；
回滚必须同时恢复 migration 60 前快照与上一版应用，不运行 down migration。

`000061` 是 Issue #286 的 additive、forward-only Source 所有权切换基础。操作员先停止
AgentOS/Admin 上的 Source 管理变更，分别取得 Data PostgreSQL 和 AgentOS Source 数据的
恢复点，然后用候选镜像执行 check-only，确认 `000061` 是唯一 pending migration 后
才 apply。迁移只能通过完整 export 文件运行 `source-import -file ...`；新鲜环境则运行
`source-initialize`。两个命令都是独立人工发布动作，不属于 migration 或普通 Deploy。
发布后确认 ledger 为 `61`、Source 总数与导出一致、不超过 200、最多一个 active
web search，并用正式 token 读取完整管理集合与 active snapshot。新 Data 可与尚未消费
新 API 的旧 Admin Backend/AgentOS 共存，但不允许双写；AgentOS 切换由其独立交付完成。
回滚 Data 应用时保留表与已导入数据，不运行 down；外部消费方切换后的回滚必须停止
新 workflow/管理写入并按 ADR-0031 恢复切换前 AgentOS 快照，不使用部分快照或反向同步。

`000062` 是 Issue #293 的 additive、forward-only Subdivision persistence 基础。操作员使用
候选 Data 镜像执行 check-only，确认它是唯一 pending migration 后 apply，并验证 ledger 为
`62`、空表字段/组合唯一/FK/enum/时间默认值满足合同。旧应用不消费新增结构，可以与已应用
schema 共存；应用回退保留新增空结构。若必须移除数据库结构，恢复 migration 62 前快照或
使用另行审阅的 forward repair，不运行 down migration。Subdivision 初始化、API wiring、
Organization 总部行政区集成与任何事实写入均需后续独立发布。

`000063` 是 Issue #298 的 additive、forward-only Ministry 与 Institution persistence 基础。
操作员使用候选 Data 镜像执行 check-only，确认它是唯一 pending migration 后 apply，并验证
ledger 为 `63`、两个空表的 MIN/INS identity、Country/Organization owner XOR、
`is_supranational` 一致性、restrictive FK、表内 code unique、native enum、nullable 字段与
时间默认值满足合同。旧应用不消费新增结构，可以与已应用 schema 共存；应用回退保留新增空
结构。若必须移除数据库结构，恢复 migration 63 前快照或使用另行审阅的 forward repair，
不运行 down migration。初始化数据、Biz/API wiring、Event Actor wiring 与新关系均需后续
独立发布。
