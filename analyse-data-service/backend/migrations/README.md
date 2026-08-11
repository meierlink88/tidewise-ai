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

实体基础库 seed 使用 repo 内版本化 JSON 文件：

```text
analyse-data-service/backend/data/entity_foundation/
```

本地执行实体 seed 前，应先执行 migration：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm data-migrate
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm \
  --entrypoint /usr/local/bin/entity-seed data
```

`analyse-data-service/backend/cmd/entity-seed` 默认读取 `data/entity_foundation` 下的实体 seed 文件，并输出 JSON report。重复执行同一组 seed 应保持幂等，report 中应主要体现为 `unchanged`，不应新增重复实体、重复 profile 或重复关系。

关系 seed 按 `data/entity_foundation/relationships/` 下的关系族文件管理。任何关系批次都必须先完成人工 review，再进入正式 JSON、PostgreSQL 和 Neo4j；候选审阅清单本身不代表已批准数据。当前已批准并写入的关系族为：

- `member_of`：223 条。
- `has_market`：40 条。
- `tracks_index`：43 条，只连接正式编制指数。

`issues`、`participates_in`、`affiliated_with` 和 `applies_to` 当前保持空 seed。它们在 benchmark 与市场、板块、商品、产业链传导基础完成前不得提前写入。

可用以下 SQL 快速核验初始化范围：

```sql
SELECT entity_type, COUNT(*) FROM entity_nodes GROUP BY entity_type ORDER BY entity_type;
SELECT relation_type, COUNT(*) FROM entity_edges GROUP BY relation_type ORDER BY relation_type;
SELECT COUNT(*) FROM alliance_org_profiles;
SELECT COUNT(*) FROM sector_profiles WHERE snapshot_date IS NOT NULL OR rank_snapshot > 0;
```

历史基础实体图一致性核验分三层执行：

1. repo seed：实体 seed 测试必须通过，并核对各关系文件数量。
2. PostgreSQL：只统计 active `entity_edges`，当前应为 `member_of=223`、`has_market=40`、`tracks_index=43`，合计 306。
3. 历史 Neo4j：旧 `projection_namespace=tidewise` 曾以 548 个 `Entity` 节点和 306
   条关系作为 seed 基准。它不是当前产业图的验收口径。

PostgreSQL 查询：

```sql
SELECT relation_type, COUNT(*)
FROM entity_edges
WHERE status = 'active'
GROUP BY relation_type
ORDER BY relation_type;

SELECT COUNT(*) AS active_entity_count
FROM entity_nodes
WHERE status = 'active';
```

Neo4j 查询：

```cypher
MATCH (n:Entity {projection_namespace: 'tidewise'})
RETURN count(n) AS entity_count;

MATCH (:Entity {projection_namespace: 'tidewise'})-[r]->(:Entity {projection_namespace: 'tidewise'})
RETURN type(r) AS relation_type, count(r) AS relation_count
ORDER BY relation_type;
```

任一层数量不一致时，不得手工修改 Neo4j；应先修正 repo seed 或 PostgreSQL
事实。旧通用投影器保持退役。

Data 不再提供 `source-seed` 或维护 `source_catalogs`，既有 Event Publication 仍只通过
`POST /api/data/v1/reviewed-event-imports` 连同轻量 Event 证据原子接纳；历史
`raw_documents.content_text` 继续可读，但 V2 新记录不保存正文。ADR-0011 另行建立与该旧链路
隔离的 Evidence 体系：完整材料进入 `raw_evidences`，不复用 `raw_documents` 或
`event_sources`。

## Neo4j 图谱投影

PostgreSQL 仍然是实体、关系、事件和证据链的事实源。Neo4j 只是从 PostgreSQL
派生的可重建查询视图，不是事实源。旧通用 `graph-projector` 仍然退役。

产业链关系 V1 提供 local-only
`analyse-data-service/backend/cmd/industry-graph-projector`。它读取精确的 approved import
receipt，在一个 `REPEATABLE READ READ ONLY` PostgreSQL 快照中构建语义集合，与冻结
CSV 逐项比较后，才在 Neo4j 显式事务中替换固定
`projection_namespace=tidewise-industry-v1`。

当前冻结验收口径：

- 节点：4,449（Industry 512、Concept 180、Industry Chain 708、Chain Node 3,049）；
- 关系：7,867（`MAPPED_TO_INDUSTRY` 716、`MAPPED_TO_CONCEPT` 521、`HAS_NODE`
  3,350、`INPUT_TO` 1,537、`IS_COMPONENT_OF` 704、`DEPENDS_ON` 404、
  `IS_SUBCATEGORY_OF` 635）；
- 孤立节点、重复关系键、自环和缺失链身份均为 0；
- 相同 package 重放必须返回 `unchanged=true`。

运行方式与独立 Cypher 核验见 `infra/local/README.md`；完整合同见
`docs/architecture/local-industry-graph-projection-v1.md`。

`graph_projection_runs` 和 `graph_projection_run_items` 用于审计每次实体图投影的输入数量、成功数量、跳过数量、失败数量和错误摘要。常用核验 SQL：

```sql
SELECT id, projection_type, mode, status, started_at, finished_at,
       source_row_count, projected_count, skipped_count, failed_count, error_summary
FROM graph_projection_runs
ORDER BY started_at DESC
LIMIT 5;

SELECT run_id, item_type, item_key, status, error_message
FROM graph_projection_run_items
ORDER BY created_at DESC
LIMIT 20;
```

这些表在 V1 仅保留历史投影审计记录；local-only 产业图 projector 不写入它们，运行
结果通过 JSON 中的 package SHA、计数、语义指纹及 `applied/unchanged` 状态审计。
