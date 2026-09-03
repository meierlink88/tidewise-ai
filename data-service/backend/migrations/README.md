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
- `000063_add_ministries_and_institutions.sql`：新增独立 `ministries` 与 `institutions` 事实表、
  MIN/INS 身份、Country/Organization XOR 归属、受控枚举和公开 Data Adapter 持久化基础。
- `000064_add_narrative_blueprints.sql`：新增独立 `geopolitic_rivalries` 与 `macro_economics`
  静态叙事蓝图、GPR/MEC 身份和受控类型/生命周期枚举；不增加 Storyline 或外部对象关系。
- `000068_add_storyline_domain_codes.sql`：为确认为空的 `storyline_domains` 增加全局唯一、
  非空且格式受控的机器 `code`；目录事实不在 migration 中 seed，需随后运行
  `storyline-domain-catalog-publish -file /app/initdata/storyline-domains-v1.json`。
- `000069_move_industry_chain_mappings_to_typed_links.sql`：删除已确认隔离的模拟晶圆测试夹具，
  将正式 IndustryChain–Industry 与 IndustryChain–Concept 映射完整迁移到两张 typed Link 表，
  保留 ERL 身份/端点/创建时间，并禁止通用 `entity_edges` 再写入两种保留关系类型。
- `000070_retire_legacy_industry_chain_tables.sql`：删除不再由当前应用拥有的全局 ChainNode
  Relation、Physical Constraint 和 Industry relationship import receipt 表及遗留回执触发函数；
  Industry Chain Graph Edge 与 Membership 保持为当前拓扑和归属事实。
- `000071_retire_entity_identifiers_and_redirects.sql`：删除无当前应用 owner 的通用 Entity
  External Identifier 与 Entity Redirect 表，移除 Redirect 专属函数，并重建剩余 Data Object
  引用保护函数以继续覆盖 Entity Relations 和 typed IndustryChain Links。
- `000074_rebuild_atomic_evidence_business_semantics.sql`：零兼容把 Atomic Evidence 重建为最小完整
  业务命题，将 Keywords 从 Raw Evidence 移至 Evidence，并以主体、动作、对象、阶段、情态、时间、
  辖区、原因、执行方式、指标和归因组成当前语义合同。
- `000075_rebuild_event_business_semantics.sql`：零兼容把 Event 升级为事件级完整业务命题，情态与
  occurrence/announcement/effectiveness 时间归入 `semantic`，增加原因、执行方式和指标，并移除
  Event wire 顶层的重复 modality 与时间字段。
- `000076_relax_event_metric_storage_constraint.sql`：修复 `000075` 对所有非空 Event metrics 的
  错误拒绝；数据库保留 Event 核心语义和指标数组/对象外形，具体指标属性由 typed HTTP 与 Biz
  边界校验。
- `000078_retire_research_theme_reason_tree.sql`：Issue #367 授权的高风险零兼容退役，按依赖顺序
  删除 Research Theme/Reason Tree 九表、其中全部数据、九个不可变 trigger 与专属函数；不创建
  Report schema，也不使用 `CASCADE`。

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

已完成 Source 所有权切换的环境可显式运行 `source-research-rss-initialize`，以 Git 审查过的
目录补齐全球投研 RSS Source。该命令只插入缺失的 dynamic/rss/generic_rss Source，重复执行
保留既有运营配置；同 code 存在但协议身份不兼容时原子失败。它不属于 migration 或普通 Deploy。

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

`000064` 是 Issue #300 的 additive、forward-only GeopoliticRivalry 与 MacroEconomic
persistence 基础。操作员使用候选 Data 镜像执行 check-only，确认它是唯一 pending
migration 后 apply，并验证 ledger 为 `64`、两个空表的 GPR/MEC identity、native enum、
必填/可空文本、默认 ACTIVE 状态、稳定列表索引与时间默认值满足合同。旧应用不消费新增
结构，可以与已应用 schema 共存；应用回退保留新增空结构。若必须移除数据库结构，恢复
migration 64 前快照或使用另行审阅的 forward repair，不运行 down migration。Storyline、
外部对象关系、OpenSPG、初始化数据、Biz/API wiring 与 UI 均需后续独立发布。

`000065` 是 Issue #302 的 additive、forward-only StorylineDomain 与
StorylineDomainTactic persistence 基础。操作员使用候选 Data 镜像执行 check-only，确认
它是唯一 pending migration 后 apply，并验证 ledger 为 `65`、两个空表的 SLD/SDT identity、
StorylineDomain native category enum、默认 active、Tactic 全局唯一机器 key、必填文本、
稳定列表索引与时间默认值满足合同。两个对象当前没有外键或彼此关系；旧应用不消费新增结构，
可以与已应用 schema 共存。应用回退保留新增空结构；若必须移除数据库结构，恢复 migration 65
前快照或使用另行审阅的 forward repair，不运行 down migration。关系、初始化数据、Biz/API
wiring、OpenSPG 与 UI 均需后续独立发布。

`000066` 是 Issue #304 的 additive、forward-only Storyline 与 StorylineEventLink
persistence 基础。操作员使用候选 Data 镜像执行 check-only，确认它是唯一 pending migration
后 apply，并验证 ledger 为 `66`、两个空表的 STL/SLE identity、Storyline 类型与唯一锚点、
native lifecycle/alignment enum、数值边界、restrictive Event 外键、唯一端点和时间默认值满足
合同。首末 Event 时间不持久化，只由 Data Adapter 从关联 Event 的非空 `occurred_at` 计算。
旧应用不消费新增结构，可以与已应用 schema 共存；应用回退保留新增空结构。若必须移除数据库
结构，恢复 migration 66 前快照或使用另行审阅的 forward repair，不运行 down migration。
Biz/API wiring、初始化数据、StorylineDomain 关系、对账历史、unlink/delete、OpenSPG 与 UI
均需后续独立发布。

`000067` 是 Issue #306 的零兼容协调切换。操作员必须停止 Data 及直接写入者、确认 PostgreSQL
恢复点，并用候选镜像执行 check-only；只有确认 `000067` 是唯一 pending migration 后才 apply。
迁移前会拒绝无法表示的 Company code/name/aliases、重复 code、不能精确唯一匹配 Industry name
的旧行业标签，以及仍通过未批准通用 Entity 关系引用 Company 的事实。执行后确认 ledger 为
`67`，Company 数量、UUID 后缀、已知名称/别名/经营区域/状态/时间戳和注册 Country 保持一致，
Company–Industry 端点正确，Company shadow Entity 为零，`company_profiles` 不存在，且
Storyline/Security 不再含 Company 引用列。controller 字段和两类明确取消的关系不会保留。
旧应用不兼容新 schema；回滚必须同时恢复 migration 67 前快照和上一版应用，不运行 down migration。

`000068` 是 Issue #308 的 StorylineDomain 目录发布前置 schema。操作员先停止任何直接写入者、
确认 `storyline_domains` 为空并保留 PostgreSQL 恢复点，再以候选 Data 镜像执行 check-only，
确认它是唯一 pending migration 后 apply；非空表会 fail closed，不从既有名称或分类猜测 code。
迁移后使用同一候选/发布镜像运行
`storyline-domain-catalog-publish -file /app/initdata/storyline-domains-v1.json`，确认 ledger 为
`68`、目录共 35 条且均 active、code 全局唯一、GEOPOLITICAL/MACRO/INDUSTRY/CORPORATE
数量分别为 7/12/8/8，并抽查
`scope_definition = description`。旧应用没有 StorylineDomain 运行时 wiring，可与新 schema 和
目录共存；应用回退保留 schema 与目录，不运行 down。失败时保持旧应用并使用恢复点或另行审阅的
forward repair。

`000069` 是 Issue #330 的零兼容协调切换。操作员必须停止 Data 及直接写入者、确认 PostgreSQL
恢复点，并用候选镜像执行 check-only；只有确认它是唯一 pending migration 后才 apply。迁移
会 fail closed 校验端点类型、active 状态、创建/更新时间一致性、ERL 格式、重复端点、模拟夹具独占性和剩余产业链
Industry 覆盖，然后删除完整模拟晶圆夹具并移动全部正式 mapping。执行后确认 ledger 为 `69`、
两张 Link 表与迁移前正式端点集合完全一致、`entity_edges` 不含保留 mapping 类型、模拟夹具
不存在，且每条剩余 IndustryChain 至少关联一个 Industry。旧应用不兼容新物理存储；回滚必须
同时恢复 migration 69 前快照和上一版应用，不运行 down migration。

`000070` 是 Issue #332 的高风险破坏性退役。操作员必须停止 Data 及直接写入者、确认当前
PostgreSQL 恢复点，并用候选镜像执行 check-only；只有确认它是唯一 pending migration 后才
apply。执行前记录 IndustryChain、ChainNode、Membership 与 Graph Edge 数量；执行后确认 ledger
为 `70`、`chain_node_relations`、`chain_node_physical_constraints`、
`industry_relationship_import_receipts` 及 `prevent_industry_relationship_import_receipt_mutation`
函数均不存在，且四类当前事实数量未变化。旧表数据不会转换为 Graph Edge；回滚必须同时恢复
migration 70 前快照和上一版应用，不运行 down migration。DDL 期间不允许新旧 Data binary
承载 mixed traffic；校验完成后只由候选 binary 恢复流量。上一版应用不访问三类退役对象，物理
schema 兼容，但单独回退应用无法恢复已删除的历史数据。

`000071` 是 Issue #334 的高风险破坏性退役。操作员必须停止 Data 及直接写入者、确认当前
PostgreSQL 恢复点，并用候选镜像执行 check-only；只有确认它是唯一 pending migration 后才
apply。执行前记录两张退役表及保留 Data Object/Relation 表数量；执行后确认 ledger 为 `71`、
`entity_external_identifiers`、`entity_redirects`、`validate_entity_redirect` 与
`protect_profiled_entity_identity` 均不存在，保留的 owner delete/truncate guard 可执行且不再
引用退役表，保留事实数量未变化。DDL 期间不允许新旧 Data binary 承载 mixed traffic；校验后
只由候选 binary 恢复流量。上一版应用没有两张表的运行时 Adapter/API，物理 schema 兼容，但
单独回退应用无法恢复已删除行；完整回滚必须同时恢复 migration 71 前快照和上一版应用。

`000072` 是 Issue #336 的高风险破坏性拓扑事实收敛。操作员必须停止 Data 及直接写入者、
取得 PostgreSQL 恢复点，并以候选镜像确认它是唯一 pending migration。执行前记录 Membership
与 Graph Edge 行数、端点集合并确认时间顺序有效；执行后确认 ledger 为 `72`、两张表只包含
ADR-0046 定义的保留列、行数与端点不变、replacement indexes 存在且 cycle guard 拒绝成环写入。
该迁移删除旧 review/lifecycle/provenance/evidence/explanation 值，Data Research Graph 合同同时
切换至 V2，禁止新旧 Data/Miniapp binary mixed traffic。完整回滚必须同时恢复 migration 72 前
快照和上一版应用，不运行 down migration。

`000073` 是 Issue #339 的高风险 Event 合同切换。操作员必须停止 Data、
Reasoning 及所有 Event 直接写入者，取得 PostgreSQL 恢复点，并确认 `events`、
`event_evidence_links`、`event_actor_links` 与 `event_asset_links` 均为空。该迁移会
fail closed 拒绝任何历史 Event，因为旧 5W1H JSON 不能无损推导出新的事件身份语义。
用候选镜像确认它是唯一 pending migration 后才可 apply；执行后确认 ledger 为
`73`、`event_publication_receipts` 约束完整，并使用新 Data/Reasoning binary 发布一条
Event 验证原子 Event + Evidence Link + Receipt。新旧 binary 不得 mixed traffic。完整
回滚必须恢复 migration 73 前恢复点并同时回退 Data/Reasoning，不运行 down migration。

`000074` 是 Issue #351 的高风险 Atomic Evidence 合同切换。操作员必须停止 Data、AgentOS
及所有 Evidence/Event 写入者，取得 PostgreSQL 恢复点，并确认 Raw Evidence、Atomic Evidence、
Event 及其 Evidence/Actor/Asset Link 与 Publication Receipt 均为空。迁移会 fail closed 拒绝任何
历史链路，因为旧 5W1H 无法无损转换为最小完整业务命题；不得在 migration 内静默删除历史事实。
确认它是唯一 pending migration 后才可 apply。执行后确认 ledger 为 `74`、Raw Evidence 不再拥有
`keywords`、Atomic Evidence 拥有 `keywords` 和新 `semantic` 约束，再用匹配的新 Data/AgentOS
发布一个含多指标业务命题的 Evidence。新旧 binary 不得 mixed traffic；完整回滚必须恢复 migration
74 前恢复点并同时回退 Data/AgentOS，不运行 down migration。

`000075` 是 Issue #359 的高风险 Event 合同切换。操作员必须停止 Data、Admin、AgentOS 及所有
Event 写入者，取得 PostgreSQL 恢复点，并确认 Event、Event Evidence/Actor/Asset Link 与 Event
Publication Receipt 均为空。迁移会 fail closed 拒绝旧七键 Event semantic，不在 migration 中
猜测或清理历史事实。确认它是唯一 pending migration 后才可 apply；执行后确认 ledger 为 `75`、
Event semantic 十键和嵌套时间约束生效，再按 Data provider → Admin consumer → AgentOS producer 的
顺序发布匹配二进制。新旧 binary 不得 mixed traffic；完整回滚必须恢复 migration 75 前恢复点并
同时回退三个应用，不运行 down migration。

`000076` 是 Issue #361 的向后兼容约束修复。它不转换或删除 Event，只重建
`chk_events_semantic`，删除重复且错误的深层 metric 属性检查，同时保留 Event 核心语义、时间投影和
metrics 数组/对象外形。迁移前后 Data API 与 Biz 合同不变，旧版与新版 Data binary 均可写入；正常
Schema migration 路径即可执行，回滚使用恢复点或后续 forward repair，不恢复 `000075` 的错误约束。

`000077` 是 Issue #363 的向前兼容 Event 时间扩展。它不转换或删除历史 Event，只允许
`semantic.time.observed_at` 作为第四类时间锚点；既有四键业务时间 JSON 继续合法，observed-only Event
使用五键 JSON。先发布兼容的 Admin/Data binary，再启用 AgentOS observed-only 写入。回滚优先回退
AgentOS 写入，再回退应用；Schema 保持兼容或通过后续 forward repair，不运行 down migration。

`000078` 是 Issue #367 的高风险破坏性退役。操作员必须停止旧 AgentOS publisher、Data、Miniapp
及所有直接写入者，取得并确认 PostgreSQL 恢复点，用候选 Data 镜像执行 check-only，并确认当前
版本为 `77` 且唯一 pending migration 为 `78`。执行前记录九张退役表各自行数；apply 后确认 ledger
为 `78`，九表、九个 trigger 与 `prevent_research_publication_mutation` 均不存在，同时 Research
Graph、Event、Raw Evidence 与 Atomic Evidence 表及事实保持不变。该阶段不创建 Report 表，Report
schema 由后续 migration 独立安装。新旧 binary 不得 mixed traffic；完整回滚必须恢复 migration 78
前快照并同时回退应用与 publisher，不运行 down migration。

`000079` 是 Issue #367 的不可变 Report 发布 schema。它仅新建 `reports` 与
`report_evidence_links` 两张表；Report 全量 typed JSONB 快照只持久化一次，其显式
Evidence ID 引用通过受约束的 Link 表关联现有 `evidences`。上线前必须停止写入、
取得恢复点，确认 ledger 为 `78` 且 `79` 是唯一 pending migration；apply 后确认
两表列集合、RPT/RPE/EVD 身份约束、两类唯一约束、反向索引、RESTRICT FK 和禁止
UPDATE/DELETE/TRUNCATE 的 statement trigger，再使用候选 Data binary 完成首次发布、安全重放与
Evidence 读取验证。新旧 publisher/Data/Miniapp binary 不得 mixed traffic。该迁移为
forward-only；完整回滚必须恢复 migration 79 前快照并同时回退相关应用，不运行
down migration。

`000080` 是 Issue #369 的 `report-publication.v2` 零数据前向切换。执行前必须确认
`reports` 与 `report_evidence_links` 都为空；migration 会在发现任一已发布事实时 fail closed，
不会转换、覆盖或删除不可变 v1 Report。它把外部幂等身份列重命名为
`publisher_report_id`，将合同固定为 v2，并把 Evidence scope 收敛到 Section summary、锚点、
推导步骤、传导、产业链 summary 和链节点影响。完整回滚必须恢复 migration 80 前快照并同步
回退 Data、Miniapp 与 AgentOS publisher，不运行 down migration。

`000081` 是 Issue #369 的最终 Report 发布合同切换。它只允许在 `reports` 与
`report_evidence_links` 均为空时执行，删除 `contract_version`，将 `content` 收敛为
`report` immutable JSONB，并将 Evidence 关联收敛为 `scope_path + position`。它不转换或
覆盖已发布报告；发现任何历史行就 fail closed。完整回滚必须恢复 migration 81 前
已确认的 RDS 恢复点，并同步回退 Data、Miniapp 与 AgentOS publisher，不运行 down migration。
