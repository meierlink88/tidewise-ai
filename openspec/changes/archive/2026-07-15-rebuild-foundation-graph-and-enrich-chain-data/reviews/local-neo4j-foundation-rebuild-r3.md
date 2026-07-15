# local Neo4j 基础投影 rebuild R3 授权包

## Review 状态

- Layer：`local-neo4j-foundation-rebuild`
- 风险：R3
- 状态：**已获独立授权，单次执行并验收通过**
- 准备日期：2026-07-14
- 执行时间：2026-07-15T00:38:50+0800（PG audit `started_at=2026-07-14T16:38:50.513807Z`）
- 对应 task：1.4；已完成 rebuild 与 Query 验收。

本包获得的唯一授权是：在已验收为空的 local `projection_namespace=tidewise` 中，使用现有 `graph-projector project-entities` 从本文冻结的 PostgreSQL projection baseline 投影 981 个节点与 229 条关系，并立即 Query 验收。

本包不授权再次 cleanup、`rebuild-entities`、重复执行、自动重试、Package 2、PostgreSQL 业务数据写入、Neo4j sync、UAT、prod 或 shared 操作。

## 唯一 Scope 与恢复语义

| 项目 | 冻结结论 |
|---|---|
| Environment | local；PostgreSQL 容器 `tidewise-local-postgres`，Neo4j 容器 `tidewise-local-neo4j` |
| Neo4j target | database `neo4j`，`projection_namespace=tidewise` |
| Rebuild scope | 从冻结 PG baseline 投影 active `alliance_org`、`economy`、`chain_node`，以及端点均在该集合内的 `entity_edges` 与 active 四类 `chain_node_relations` |
| Projector entry | 从 `backend/` 运行现有 `graph-projector project-entities`；Neo4j 已为空，因此禁止使用会再次 cleanup 的 `rebuild-entities` |
| Preserved | PostgreSQL 业务事实；Neo4j database、constraint、index、configuration；其他 namespace |
| Recovery Evidence | `approved-disposable-recovery` |
| Recovery source | 本文冻结且已验收的 PostgreSQL projection baseline |
| Neo4j backup/rollback | 不创建、不要求、不宣称 |
| Failure handling | 停止并报告；不得自动 cleanup、retry、再次 project 或 forward-fix |

### PostgreSQL 写入边界的精确说明

现有 projector 在每次投影时按主规格写 `graph_projection_runs` 审计记录，并只在 skipped/failed 时写 `graph_projection_run_items`。因此本命名层的“不得写 PostgreSQL”精确解释为：不得写 `entity_nodes`、`entity_edges`、`chain_node_relations` 或其他业务事实；允许且预期现有 projector 写一条投影运行审计记录。

若授权要求 PostgreSQL **任何表均零写入**，则现有入口无法满足，必须停止并回到 Review；不得绕过审计、临时改源码或手工直写 Neo4j。

## PostgreSQL frozen projection baseline

以下结果于 2026-07-14 在 `BEGIN READ ONLY` 中新鲜复算。MD5 仅作为确定性 drift fingerprint，不作为安全散列或备份。

### 环境与 schema

| 项目 | 冻结值 |
|---|---|
| Container image | `postgres:16` / `sha256:be01cf82fc7dbba824acf0a82e150b4b360f3ff93c6631d7844af431e841a95c` |
| Container state | `running / healthy` |
| Database / user | `tidewise_local / tidewise` |
| PostgreSQL | `16.14 (Debian 16.14-1.pgdg13+1)` |
| Goose max applied / rows | `18 / 19` |
| Relevant columns hash | `d46931e02d32458cd96840499a554793` |
| Relevant constraints hash | `0d38e4ff10b291e5c557f6f49899ba49` |

### Projection source counts 与 hash

| Source | Type | Count | Hash |
|---|---|---:|---|
| `entity_nodes` | `alliance_org` | 45 | `d88821f15290c49a95143d2d559b92b0` |
| `entity_nodes` | `economy` | 94 | `513a4a89d9f324a8ee68af3c94868191` |
| `entity_nodes` | `chain_node` | 842 | `0467f0fe0d5628936671a05304821e34` |
| `entity_nodes` | total | 981 | `e70a01e4c7fb7b3f020aa5713771e273` |
| `entity_edges` | `member_of` | 133 | `f34b720ad336bf3b7bda8cd30789e68e` |
| `entity_edges` | total | 133 | `f34b720ad336bf3b7bda8cd30789e68e` |
| `chain_node_relations` | `is_subcategory_of` | 95 | `583dc30085b1de70179de94cf8e4f7b5` |
| `chain_node_relations` | `is_component_of` | 1 | `9ce49fff74ca75c627999157ec463573` |
| `chain_node_relations` | total | 96 | `7ff0433678e0f40f271bdd29ad57edef` |

`input_to=0`、`depends_on=0`。另有 108 条 active 且证据完整、但端点超出三类节点集合的 `entity_edges`；它们明确不进入 source。预期 `source_rows=981+133+96=1210`。

229 条关系按 `edge_id|from_id|to_id|original_relation_type|source` 排序后的统一 identity/direction hash 为 `f9884df7f3b67ec50c8c74cd0f312bed`；`entity_edges` 与 `chain_node_relations` 两个 source 之间的重复 edge ID 为 0。

### PG integrity 与审计基线

| Assertion | Frozen result |
|---|---:|
| projected `entity_key` duplicate | 0 |
| active `entity_edges` orphan endpoint | 0 |
| in-scope `entity_edges` duplicate tuple | 0 |
| active `chain_node_relations` orphan/invalid endpoint | 0 |
| active `chain_node_relations` duplicate tuple | 0 |
| cross-source duplicate edge ID | 0 |
| `graph_projection_runs` rows | 16 |
| latest projection `started_at` | `2026-07-13 05:44:57.341717+00` |
| `graph_projection_run_items` rows | 2 |

执行前必须重新运行 cleanup 授权包中的完整 PostgreSQL read-only SQL，并额外复验审计基线。任一 count、hash、schema、endpoint、duplicate、orphan、active/source 边界或审计基线漂移都必须停止。

## local Neo4j empty baseline

| 项目 | 冻结值 |
|---|---|
| Container image | `neo4j:5-community` / `sha256:4bae36aff76271e27fd6a6ed0835413f86a284cd179cfb1cb7d188f5f7533aca` |
| Container state | `running / healthy` |
| Neo4j | `5.26.28 community` |
| Database | `neo4j / standard / read-write / online` |
| Global nodes / relationships | `0 / 0` |
| Tidewise nodes / relationships | `0 / 0` |
| Other namespace nodes / relationships | `0 / 0` |
| Constraints | 0 |
| `backend/config/config.local.yaml` SHA-256 | `21cedf54b77c3a8b794e0fb8af4c5a2a2b38189fa1d260f5ecee9c3701e86a47` |
| `infra/local/docker-compose.neo4j.yaml` SHA-256 | `ca344992e16fd973c3bdc8ce1c9b18e7b3291e5fd7e0471bf1ec98caaea58c30` |

必须保留的两个 ONLINE lookup indexes：

| Name | Type | Entity type | State | Population |
|---|---|---|---|---:|
| `index_343aff4e` | LOOKUP | NODE | ONLINE | 100.0 |
| `index_f7700477` | LOOKUP | RELATIONSHIP | ONLINE | 100.0 |

## 执行前 preflight

取得仅针对本命名 layer 的明确 R3 授权后，仍必须按顺序重新只读复验：

1. Git worktree clean，HEAD 与本 Review checkpoint 一致；branch/upstream 同步。
2. 两个容器的 name、image digest、running/healthy 状态与本文一致。
3. PG identity、Goose、schema hash、source count/hash、active/source 边界、endpoint/orphan/duplicate 与审计基线逐项一致；repo migration latest 仍为 18，确保 `auto_apply` 不会应用 migration。
4. Neo4j 必须仍为 global 0/0、Tidewise 0/0、other namespace 0/0；database online/read-write；constraints=0；两个 lookup indexes 与配置 hash 不变。
5. 运行现有 targeted tests；任何失败都停止，不得执行投影。
6. 确认三个凭证环境变量已注入但不输出值：`DATABASE_PASSWORD`、`NEO4J_USERNAME`、`NEO4J_PASSWORD`。
7. 确认授权只命名 `local-neo4j-foundation-rebuild`，且接受本层预期的一条 PG projection audit metadata 写入；没有同时授权 cleanup、retry、sync 或 Package 2。

只读技术检查：

```sh
cd backend
go test ./internal/repositories ./internal/apps/graphprojection ./cmd/graph-projector -count=1
```

PG 与 Neo4j 的精确只读查询复用 [cleanup R3 授权包](local-neo4j-foundation-cleanup-r3.md) 中的 SQL/Cypher；Neo4j 预期值改为本文的空库基线。

## 唯一执行入口

**以下命令已在本 layer 的独立授权下单次执行。当前授权已消耗，不得再次执行；重试或第二次 project 必须重新 Review 与授权。**

```sh
cd backend
test -n "$DATABASE_PASSWORD"
test -n "$NEO4J_USERNAME"
test -n "$NEO4J_PASSWORD"
APP_ENV=local go run ./cmd/graph-projector project-entities
```

必须使用 `project-entities`：当前 Neo4j 已为空，该模式只 upsert 本次 source。禁止 `rebuild-entities`，因为后者会再次调用 `DeleteNamespace`，构成未授权的新 cleanup。

预期 CLI 摘要：

```text
status=succeeded source_rows=1210 projected=1210 skipped=0 failed=0
```

run ID 动态生成，不预先固定。命令只允许运行一次；非零退出、超时、连接中断或输出不确定时不得重试。

## rebuild 后 Query / assert

单次命令成功后立即只读验收：

```cypher
MATCH (n:Entity {projection_namespace: 'tidewise'})
RETURN n.entity_type AS entity_type, count(*) AS count ORDER BY entity_type;

MATCH ()-[r]->()
WHERE r.projection_namespace = 'tidewise'
RETURN type(r) AS relationship_type, count(*) AS count ORDER BY relationship_type;

MATCH (a:Entity {projection_namespace: 'tidewise'})-[r]->(b:Entity {projection_namespace: 'tidewise'})
RETURN r.edge_id AS edge_id, a.entity_id AS from_id, b.entity_id AS to_id,
       r.original_relation_type AS original_relation_type, r.source AS source
ORDER BY edge_id, source;

MATCH (n:Entity {projection_namespace: 'tidewise'})
WITH n.entity_id AS entity_id, count(*) AS n
WHERE n > 1 RETURN count(*) AS duplicate_entity_id_groups;

MATCH ()-[r]->()
WHERE r.projection_namespace = 'tidewise'
WITH r.edge_id AS edge_id, count(*) AS n
WHERE n > 1 RETURN count(*) AS duplicate_edge_id_groups;

MATCH (a)-[r]->(b)
WHERE r.projection_namespace = 'tidewise'
  AND (a.projection_namespace <> 'tidewise' OR b.projection_namespace <> 'tidewise')
RETURN count(r) AS invalid_endpoint_relationships;

MATCH (n:Entity {projection_namespace: 'tidewise'})
WHERE NOT n.entity_type IN ['alliance_org', 'economy', 'chain_node']
RETURN count(n) AS unsupported_entity_nodes;

MATCH ()-[r]->()
WHERE r.projection_namespace = 'tidewise'
  AND NOT type(r) IN ['MEMBER_OF', 'IS_SUBCATEGORY_OF', 'IS_COMPONENT_OF', 'INPUT_TO', 'DEPENDS_ON']
RETURN count(r) AS unsupported_or_legacy_relationships;

MATCH (n:TidewiseEntity) RETURN count(n) AS legacy_label_nodes;

SHOW DATABASES YIELD name, type, access, currentStatus
WHERE name = 'neo4j' RETURN name, type, access, currentStatus;

SHOW CONSTRAINTS YIELD name RETURN count(name) AS constraint_count;

SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties, state, populationPercent
RETURN name, type, entityType, labelsOrTypes, properties, state, populationPercent ORDER BY name;
```

必须同时满足：

- 节点：`alliance_org=45`、`economy=94`、`chain_node=842`，合计 981。
- 关系：`MEMBER_OF=133`、`IS_SUBCATEGORY_OF=95`、`IS_COMPONENT_OF=1`，合计 229；`INPUT_TO=0`、`DEPENDS_ON=0`。
- 上述有向关系明细必须与 PG 的 229 条统一 source 明细逐行一致；按相同字段和顺序计算的 identity/direction hash 必须为 `f9884df7f3b67ec50c8c74cd0f312bed`。missing/invalid endpoint、duplicate entity ID、duplicate edge ID、unsupported entity/relation、legacy label 全部为 0。
- other namespace 仍为 0；database、constraints、两个 lookup indexes 与配置 hash 不变。
- 完整 PG business baseline 的 count/hash/schema/integrity 全部不变。
- `graph_projection_runs` 恰好从 16 增至 17；最新 run 必须为 `mode=project_entities`、`status=succeeded`、`source_row_count=1210`、`projected_count=1210`、`skipped_count=0`、`failed_count=0`、`config_summary.namespace=tidewise`。
- 成功路径不应新增 run item，因此 `graph_projection_run_items` 仍为 2；若新增 item 或存在 skipped/failed，验收失败并停止。

Neo4j writer 使用 `MERGE`，targeted tests 已覆盖幂等 upsert 契约；本次单次执行后以 duplicate=0 验收。不得为“实库幂等测试”自动执行第二次，因为第二次仍是新的 Neo4j 写入并会新增 PG audit run，需另行授权。

## Stop Conditions

任一条件成立即 fail-closed：

- 环境不是本文冻结的 local container/database/namespace，或 image/config/index/constraint 漂移。
- PostgreSQL identity、Goose、schema、source count/hash、active/source 边界、endpoint/orphan/duplicate 或 audit baseline 漂移。
- Neo4j 执行前不再严格为空 0/0，或 database 非 online/read-write。
- targeted tests 失败、凭证变量缺失，或未收到只针对本 layer 的明确 R3 授权。
- 授权不接受既有 PG projection audit metadata 写入，却要求继续使用现有 projector。
- CLI 非零、超时、连接中断、结果不确定、status 非 succeeded，或 source/projected/skipped/failed 不等于 1210/1210/0/0。
- 任一节点/关系 count、方向、missing、duplicate、orphan、legacy、database/index/constraint 或 PG business baseline postcondition 失败。
- PG audit 增量不是精确一条成功 run，或出现新的 run item。

触发后立即停止并报告。不得自动 cleanup、retry、再次 project、改源码、手工补图或进入 Package 2。

## 脱敏 execution evidence

### 授权与 fresh preflight

- 用户授权只命名 `local-neo4j-foundation-rebuild`，接受现有 projector 新增一条 PG projection audit metadata；明确不授权 cleanup、retry、sync、PG 业务写或 Package 2。
- 执行前 Git HEAD 与远端均为 `6db12638228dbdab5ce2315a9ff249e421f6ec84`，worktree clean，branch 为 `codex/rebuild-foundation-graph-and-enrich-chain-data`。
- PostgreSQL/Neo4j container name、image digest、running/healthy 状态及两个非敏感配置 hash 与 frozen baseline 一致；repo migration latest 为 18。
- PG fresh read-only preflight 为 Goose 18 / 19 applied rows；schema hash、981 nodes、133 entity_edges、96 chain_node_relations、229 关系 direction hash 与全部 integrity 结果逐项一致；`graph_projection_runs=16`、`run_items=2`。
- Neo4j fresh read-only preflight 为 global/Tidewise `0/0`，other namespace `0/0`；database online/read-write，constraints=0，两个 lookup indexes ONLINE 且 metadata 一致。
- targeted repository、graphprojection、graph-projector tests 全部通过后才执行投影。

### 唯一写操作

- 只从 `backend/` 单次执行 `APP_ENV=local go run ./cmd/graph-projector project-entities`；凭证从 local runtime 注入且未输出。
- CLI 退出码为 0，输出：`run=d65e5f38-7de9-5ab2-b222-7d968d2b6ad6 status=succeeded source_rows=1210 projected=1210 skipped=0 failed=0`。
- 未调用 `rebuild-entities`，未执行 `DeleteNamespace`、cleanup、retry、第二次 project、sync 或 Package 2。

### 写后 Neo4j Query/assert

| Assertion | Result |
|---|---|
| global / Tidewise nodes | `981 / 981` |
| node types | `alliance_org=45; economy=94; chain_node=842` |
| global / Tidewise relationships | `229 / 229` |
| relationship types | `MEMBER_OF=133; IS_SUBCATEGORY_OF=95; IS_COMPONENT_OF=1; INPUT_TO=0; DEPENDS_ON=0` |
| other namespace nodes / relationships | `0 / 0` |
| duplicate entity / edge groups | `0 / 0` |
| invalid endpoint | 0 |
| unsupported entity / relationship | `0 / 0` |
| legacy `TidewiseEntity` | 0 |
| missing/non-active node / relationship | `0 / 0` |
| database | `neo4j / standard / read-write / online` |
| constraints | 0 |
| lookup indexes | `index_343aff4e NODE ONLINE 100.0; index_f7700477 RELATIONSHIP ONLINE 100.0` |

PG 与 Neo4j 分别导出 229 条 `edge_id|from_id|to_id|original_relation_type|source` 脱敏 identity/direction 行，逐行完全一致；两侧 MD5 均为 `f9884df7f3b67ec50c8c74cd0f312bed`。首次辅助 hash 命令曾因把记录分隔符编码为字面量 `\\n` 得到非规范值；只读逐行诊断确认这是校验命令格式问题，不是数据差异，期间未执行任何写入或重试。

### 写后 PostgreSQL Query/assert

- 业务事实仍为 Goose 18 / 19 applied rows；columns hash `d46931e02d32458cd96840499a554793`、constraints hash `0d38e4ff10b291e5c557f6f49899ba49`。
- 三类节点、entity_edges、chain_node_relations 的 count/hash 及 duplicate/orphan/endpoint integrity 全部与 frozen baseline 一致。
- `graph_projection_runs` 精确从 16 增至 17；最新 run 为 `d65e5f38-7de9-5ab2-b222-7d968d2b6ad6 / entity_graph / project_entities / succeeded / source=1210 / projected=1210 / skipped=0 / failed=0 / namespace=tidewise`。
- `graph_projection_run_items` 仍为 2，没有新增 skipped/failed item。

### 下一授权边界

task 1.4 到此完成。当前 checkpoint 不准备或执行 Package 2；必须停止，由项目经理主对话独立验收后再逐层派发下一步。

### Checkpoint 验证

- `openspec validate rebuild-foundation-graph-and-enrich-chain-data --strict`：通过。
- explicit task-design lint：通过。
- `go test ./internal/repositories ./internal/apps/graphprojection ./cmd/graph-projector -count=1`：通过。
- `git diff --check`、scope、whitespace、secret 检查：通过；diff 仅本 Review artifact 与当前 `tasks.md`。

## 本次授权记录

本次执行采用并严格遵守以下授权文本；该授权已随单次投影执行而消耗：

> 授权执行 `local-neo4j-foundation-rebuild`，仅在 local Neo4j 空 Tidewise namespace 中使用现有 `graph-projector project-entities` 从本文冻结 PG baseline 单次投影并 Query；接受现有 projector 写一条 PG projection audit metadata；不授权 cleanup、retry、sync、PG 业务写入或 Package 2。
