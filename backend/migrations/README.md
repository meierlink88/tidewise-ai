# Database Migrations

本目录保存 PostgreSQL schema 的版本化 DDL，是数据库结构演进的工程来源。

规则：

- migration 文件使用 Goose 兼容 SQL，文件名格式为 `000001_name.sql`。
- 已在共享环境或生产环境执行过的 migration 不得重写；后续 schema 变化必须追加新版本 migration。
- migration 必须保留 `-- +goose Up` 和 `-- +goose Down` 段。
- 自动升级只能执行可审阅的增量 DDL，不得通过清空表、重建全库或丢弃业务数据完成升级。
- 破坏性结构调整必须拆成独立、可审阅的开发任务，并包含兼容窗口、数据回填和人工确认。

初始 schema 的 down 段不自动删除业务表。需要回滚初始 schema 时，应通过已审阅的前向修复 migration 或数据库备份恢复执行，避免在有数据环境中误删事实基础。

本地执行 migration 时，优先使用后端命令入口：

```bash
cd backend
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/dbmigrate -apply
```

只检查 pending migration 时不加 `-apply`。

当前实体基础库相关 migration：

- `000001_init_event_knowledge_schema.sql`：创建事件知识 schema、实体节点、实体关系和各类 profile 表。
- `000002_add_alliance_org_profiles.sql`：补充联盟组织 profile 表。
- `000003_add_sector_seed_snapshot_fields.sql`：补充板块初始化热度快照字段 `rank_snapshot` 和 `snapshot_date`。
- `000004_add_source_catalog_source_config.sql`：为 `source_catalogs` 补充 `source_config` JSONB 扩展配置字段。
- `000005_add_ingestion_scheduler.sql`：创建采集调度器配置、执行批次和 source 级执行记录表。
- `000006_add_graph_projection_runs.sql`：补充 `entity_nodes.entity_key`，创建 Neo4j 图谱投影 run 和明细记录表。
- `000015_refactor_industry_chain_node_phase_a.sql`：以人工授权门禁执行旧产业结构受控 cleanup，并收敛最小 chain_node/theme profile。
- `000016_add_entity_external_identifiers.sql`：新增通用实体外部标识表、外部 identity 唯一约束与实体侧查询索引；不包含任何 mapping 数据。

实体基础库 seed 使用 repo 内版本化 JSON 文件：

```text
backend/data/entity_foundation/
```

本地执行实体 seed 前，应先执行 migration：

```bash
cd backend
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/dbmigrate -apply
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/entity-seed
```

`cmd/entity-seed` 默认读取 `data/entity_foundation` 下的实体 seed 文件，并输出 JSON report。重复执行同一组 seed 应保持幂等，report 中应主要体现为 `unchanged`，不应新增重复实体、重复 profile 或重复关系。

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

最终实体图一致性核验分三层执行：

1. repo seed：实体 seed 测试必须通过，并核对各关系文件数量。
2. PostgreSQL：只统计 active `entity_edges`，当前应为 `member_of=223`、`has_market=40`、`tracks_index=43`，合计 306。
3. Neo4j：重建结果必须为 548 个 `Entity` 节点和 306 条关系，且关系类型计数与 PostgreSQL 完全一致。

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

任一层数量不一致时，不得手工修改 Neo4j；应先修正 repo seed 或 PostgreSQL 事实，再运行 `graph-projector rebuild-entities`。

采集源目录 seed 使用 repo 内版本化 JSON 文件：

```text
backend/data/source_catalogs/
```

本地执行采集源 seed 前，应先执行 migration：

```bash
cd backend
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/dbmigrate -apply
APP_ENV=local DATABASE_PASSWORD=<local-password> go run ./cmd/source-seed
```

`cmd/source-seed` 会输出来源统计 report，重复执行应按稳定来源 ID 幂等 upsert，不应创建重复 `source_catalogs` 记录。常用核验 SQL：

```sql
SELECT COUNT(*) FROM source_catalogs;
SELECT provider_key, COUNT(*) FROM source_catalogs GROUP BY provider_key ORDER BY COUNT(*) DESC, provider_key;
SELECT source_type, COUNT(*) FROM source_catalogs GROUP BY source_type ORDER BY source_type;
SELECT status, COUNT(*) FROM source_catalogs GROUP BY status ORDER BY status;
```

## Neo4j 图谱投影记录

PostgreSQL 仍然是实体、关系、事件和证据链的事实源。Neo4j 只保存从 PostgreSQL 派生的图谱查询视图；Neo4j 数据清空后，应通过 `graph-projector` 从 PostgreSQL 重新投影。

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

本地执行实体图投影前，应先完成 migration、实体 seed，并启动 Neo4j：

```bash
cd backend
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> go run ./cmd/dbmigrate -apply
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> go run ./cmd/entity-seed
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./cmd/graph-projector check
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./cmd/graph-projector project-entities
```

需要清理并重建本系统命名空间下的实体图时使用：

```bash
APP_ENV=local DATABASE_PASSWORD=<local-postgres-password> NEO4J_USERNAME=<local-neo4j-user> NEO4J_PASSWORD=<local-neo4j-password> go run ./cmd/graph-projector rebuild-entities
```
