# local Neo4j 基础投影 cleanup R3 授权包

## Review 状态

- Layer：`local-neo4j-foundation-cleanup`
- 风险：R3
- 状态：**等待用户明确授权，尚未执行**
- 准备日期：2026-07-14
- 对应 task：1.3；本授权包只完成只读 baseline 与执行计划准备，task 1.3 仍未完成。

本包申请的唯一授权是：Neo4j cleanup 仅清空批准的 local `projection_namespace=tidewise` 节点与关系。它不授权 `rebuild-entities`、`project-entities`、PostgreSQL 写入、Neo4j rebuild/sync、Package 2、UAT、prod 或 shared 操作。

## 唯一 Scope 与恢复语义

| 项目 | 冻结结论 |
|---|---|
| Environment | local；PostgreSQL 容器 `tidewise-local-postgres`，Neo4j 容器 `tidewise-local-neo4j` |
| Neo4j target | database `neo4j`，`projection_namespace=tidewise` |
| Cleanup scope | 仅删除 `(:Entity {projection_namespace: 'tidewise'})` 及其关系 |
| Preserved | Neo4j database、constraint、index、configuration；PostgreSQL 全部数据；其他 namespace |
| Recovery Evidence | `approved-disposable-recovery` |
| Recovery source | 本文冻结的 PostgreSQL projection baseline |
| Neo4j backup/rollback | 不创建、不要求、不宣称 |
| Rebuild | 本包禁止；cleanup 验收后必须另行取得 `local-neo4j-foundation-rebuild` R3 授权 |

cleanup 使用 `Neo4jGraphWriter.DeleteNamespace` 已有 Cypher 的同义单次入口。`graph-projector rebuild-entities` 会把 cleanup 与 rebuild 合并，因此本包禁止调用该命令。

## PostgreSQL frozen projection baseline

以下查询在 `BEGIN READ ONLY` 中执行。MD5 仅作为确定性 drift fingerprint，不作为安全散列或备份。

### 环境与 schema

| 项目 | 冻结值 |
|---|---|
| Container image | `postgres:16` / `sha256:be01cf82fc7dbba824acf0a82e150b4b360f3ff93c6631d7844af431e841a95c` |
| Container state | `running / healthy` |
| Database / user | `tidewise_local / tidewise` |
| PostgreSQL | `16.14 (Debian 16.14-1.pgdg13+1)` |
| Goose max applied | `18` |
| Applied version rows | `19` |
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

`input_to` 与 `depends_on` 当前均为 0，因此没有伪造空类型 hash。`entity_edges` 另有 108 条 active 且证据完整、但端点超出本次三类节点边界的关系；它们明确不进入重建 source。

### PG integrity

| Assertion | Frozen result |
|---|---:|
| projected `entity_key` duplicate | 0 |
| active `entity_edges` orphan endpoint | 0 |
| in-scope `entity_edges` duplicate tuple | 0 |
| active `chain_node_relations` orphan/invalid endpoint | 0 |
| active `chain_node_relations` duplicate tuple | 0 |

执行前与执行后都必须在显式 read-only transaction 中复算上述 count/hash/integrity，并逐项等于冻结值。任何差异均视为 PG baseline drift。

## local Neo4j frozen baseline

### 环境 identity

| 项目 | 冻结值 |
|---|---|
| Container image | `neo4j:5-community` / `sha256:4bae36aff76271e27fd6a6ed0835413f86a284cd179cfb1cb7d188f5f7533aca` |
| Container state | `running / healthy` |
| Neo4j | `5.26.28 community` |
| Database | `neo4j / standard / read-write / online` |
| Runtime cypher-shell | `/var/lib/neo4j/bin/cypher-shell`；不在默认 `PATH` |
| `backend/config/config.local.yaml` SHA-256 | `21cedf54b77c3a8b794e0fb8af4c5a2a2b38189fa1d260f5ecee9c3701e86a47` |
| `infra/local/docker-compose.neo4j.yaml` SHA-256 | `ca344992e16fd973c3bdc8ce1c9b18e7b3291e5fd7e0471bf1ec98caaea58c30` |

### Tidewise namespace nodes

| Entity type | Count |
|---|---:|
| alliance_org | 10 |
| benchmark | 10 |
| chain_node | 54 |
| commodity | 45 |
| company | 77 |
| economy | 50 |
| index | 43 |
| industry_chain | 2 |
| instrument | 4 |
| market | 47 |
| metric | 43 |
| person | 30 |
| policy_body | 30 |
| sector | 52 |
| security | 77 |
| **Total** | **574** |

### Tidewise namespace relationships

| Relationship type | Count |
|---|---:|
| COVERS_SECTOR | 52 |
| DEPENDS_ON | 2 |
| HAS_MARKET | 40 |
| MAPPED_TO_SECTOR | 6 |
| MEASURES | 10 |
| MEMBER_OF | 223 |
| MEMBER_OF_CHAIN | 27 |
| OBSERVES_BENCHMARK | 10 |
| REFERENCES | 5 |
| SUPPLIES_TO | 22 |
| TRACKS_INDEX | 43 |
| **Total** | **440** |

这些值证明当前 Neo4j 是旧广域投影，不是本文 PG frozen baseline 的当前三类实体投影；cleanup 的目的正是建立独立的空 local projection 状态，rebuild 不属于本授权。

### Neo4j boundary 与 preserved metadata

| Assertion | Frozen result |
|---|---:|
| other namespace nodes | 0 |
| other namespace relationships | 0 |
| Tidewise non-`Entity` nodes | 0 |
| cross-namespace attached relationships | 0 |
| Tidewise relationship invalid endpoint | 0 |
| duplicate `entity_id` | 0 |
| duplicate `edge_id` | 0 |
| legacy `TidewiseEntity` label | 0 |
| constraints | 0 |
| indexes | 2 |

必须保留的两个 ONLINE lookup indexes：

| Name | Type | Entity type | State | Population |
|---|---|---|---|---:|
| `index_343aff4e` | LOOKUP | NODE | ONLINE | 100.0 |
| `index_f7700477` | LOOKUP | RELATIONSHIP | ONLINE | 100.0 |

## 执行前 preflight

获得用户对本命名 layer 的明确授权后，仍必须按以下顺序重新只读复验：

1. `git status --short` 为空，HEAD 与本授权包 checkpoint 一致。
2. 两个容器的 name、image digest、running/healthy 状态等于本文。
3. PostgreSQL read-only 查询的 environment、Goose、schema hash、source count/hash、endpoint/orphan/duplicate 全部等于本文。
4. Neo4j database 仍为 `neo4j / read-write / online`，Tidewise 仍为 574 nodes / 440 relationships，分类 counts 与本文一致。
5. other namespace、cross-namespace、invalid endpoint、duplicate、legacy 均为 0。
6. constraints 仍为 0；两个 lookup indexes 的 name/type/entityType/state/population 逐项不变；两个非敏感配置文件 hash 不变。
7. 确认当前授权文本只命名 `local-neo4j-foundation-cleanup`，未同时授权 rebuild。

PostgreSQL preflight 使用以下精确 read-only SQL；输出必须逐项等于本文 frozen baseline：

```sh
docker exec -i tidewise-local-postgres psql -X -U tidewise -d tidewise_local -v ON_ERROR_STOP=1 -P pager=off -At -F '|' <<'SQL'
BEGIN READ ONLY;

SELECT 'pg_identity', current_database(), current_user,
       current_setting('server_version'),
       COALESCE(inet_server_addr()::text, 'local-socket'),
       COALESCE(inet_server_port()::text, 'local-socket');

SELECT 'goose', MAX(version_id) FILTER (WHERE is_applied),
       COUNT(*) FILTER (WHERE is_applied)
FROM goose_db_version;

SELECT 'schema_columns_hash',
       md5(COALESCE(string_agg(concat_ws('|', table_name, ordinal_position::text,
           column_name, data_type, udt_name, is_nullable, COALESCE(column_default, '')),
           E'\n' ORDER BY table_name, ordinal_position), ''))
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name IN ('entity_nodes', 'entity_edges', 'chain_node_relations');

SELECT 'schema_constraints_hash',
       md5(COALESCE(string_agg(concat_ws('|', c.conrelid::regclass::text,
           c.conname, c.contype, pg_get_constraintdef(c.oid, true)),
           E'\n' ORDER BY c.conrelid::regclass::text, c.conname), ''))
FROM pg_constraint c
WHERE c.conrelid IN ('entity_nodes'::regclass, 'entity_edges'::regclass,
                     'chain_node_relations'::regclass);

WITH projected_nodes AS (
    SELECT * FROM entity_nodes
    WHERE status = 'active'
      AND entity_type IN ('alliance_org', 'economy', 'chain_node')
)
SELECT 'node', entity_type, COUNT(*),
       md5(COALESCE(string_agg(jsonb_build_object(
           'id', id, 'entity_key', entity_key, 'entity_type', entity_type,
           'layer_code', layer_code, 'name', name, 'canonical_name', canonical_name,
           'aliases', to_jsonb(aliases), 'status', status, 'updated_at', updated_at
       )::text, E'\n' ORDER BY id), ''))
FROM projected_nodes
GROUP BY entity_type
UNION ALL
SELECT 'node', 'total', COUNT(*),
       md5(COALESCE(string_agg(jsonb_build_object(
           'id', id, 'entity_key', entity_key, 'entity_type', entity_type,
           'layer_code', layer_code, 'name', name, 'canonical_name', canonical_name,
           'aliases', to_jsonb(aliases), 'status', status, 'updated_at', updated_at
       )::text, E'\n' ORDER BY id), ''))
FROM projected_nodes
ORDER BY 2;

WITH scoped_edges AS (
    SELECT e.*
    FROM entity_edges e
    JOIN entity_nodes f ON f.id = e.from_entity_id
    JOIN entity_nodes t ON t.id = e.to_entity_id
    WHERE e.status = 'active' AND f.status = 'active' AND t.status = 'active'
      AND f.entity_type IN ('alliance_org', 'economy', 'chain_node')
      AND t.entity_type IN ('alliance_org', 'economy', 'chain_node')
      AND e.source_name <> '' AND e.source_url <> '' AND e.verified_at IS NOT NULL
)
SELECT 'entity_edge', relation_type, COUNT(*),
       md5(COALESCE(string_agg(jsonb_build_object(
           'id', id, 'from', from_entity_id, 'to', to_entity_id,
           'type', relation_type, 'evidence', evidence_note,
           'source_name', source_name, 'source_url', source_url,
           'verified_at', verified_at, 'status', status, 'updated_at', updated_at
       )::text, E'\n' ORDER BY id), ''))
FROM scoped_edges
GROUP BY relation_type
UNION ALL
SELECT 'entity_edge', 'total', COUNT(*),
       md5(COALESCE(string_agg(jsonb_build_object(
           'id', id, 'from', from_entity_id, 'to', to_entity_id,
           'type', relation_type, 'evidence', evidence_note,
           'source_name', source_name, 'source_url', source_url,
           'verified_at', verified_at, 'status', status, 'updated_at', updated_at
       )::text, E'\n' ORDER BY id), ''))
FROM scoped_edges
ORDER BY 2;

WITH scoped_relations AS (
    SELECT r.*
    FROM chain_node_relations r
    JOIN entity_nodes f ON f.id = r.from_chain_node_entity_id
    JOIN entity_nodes t ON t.id = r.to_chain_node_entity_id
    WHERE r.status = 'active' AND f.status = 'active' AND t.status = 'active'
      AND f.entity_type = 'chain_node' AND t.entity_type = 'chain_node'
      AND r.relation_type IN ('is_subcategory_of', 'is_component_of', 'input_to', 'depends_on')
)
SELECT 'chain_relation', relation_type, COUNT(*),
       md5(COALESCE(string_agg(jsonb_build_object(
           'id', id, 'from', from_chain_node_entity_id, 'to', to_chain_node_entity_id,
           'type', relation_type, 'mechanism', mechanism, 'condition', condition_note,
           'evidence', evidence_note, 'provenance', provenance,
           'verified_at', verified_at, 'status', status, 'updated_at', updated_at
       )::text, E'\n' ORDER BY id), ''))
FROM scoped_relations
GROUP BY relation_type
UNION ALL
SELECT 'chain_relation', 'total', COUNT(*),
       md5(COALESCE(string_agg(jsonb_build_object(
           'id', id, 'from', from_chain_node_entity_id, 'to', to_chain_node_entity_id,
           'type', relation_type, 'mechanism', mechanism, 'condition', condition_note,
           'evidence', evidence_note, 'provenance', provenance,
           'verified_at', verified_at, 'status', status, 'updated_at', updated_at
       )::text, E'\n' ORDER BY id), ''))
FROM scoped_relations
ORDER BY 2;

SELECT 'integrity', 'duplicate_projected_entity_key', COALESCE(SUM(n - 1), 0)
FROM (SELECT entity_key, COUNT(*) n FROM entity_nodes
      WHERE status = 'active' AND entity_type IN ('alliance_org', 'economy', 'chain_node')
      GROUP BY entity_key HAVING COUNT(*) > 1) d;

SELECT 'integrity', 'entity_edge_orphan_endpoint', COUNT(*)
FROM entity_edges e
LEFT JOIN entity_nodes f ON f.id = e.from_entity_id
LEFT JOIN entity_nodes t ON t.id = e.to_entity_id
WHERE e.status = 'active' AND (f.id IS NULL OR t.id IS NULL);

SELECT 'integrity', 'entity_edge_in_scope_duplicate_tuple', COALESCE(SUM(n - 1), 0)
FROM (
    SELECT e.from_entity_id, e.to_entity_id, e.relation_type, COUNT(*) n
    FROM entity_edges e
    JOIN entity_nodes f ON f.id = e.from_entity_id
    JOIN entity_nodes t ON t.id = e.to_entity_id
    WHERE e.status = 'active' AND f.status = 'active' AND t.status = 'active'
      AND f.entity_type IN ('alliance_org', 'economy', 'chain_node')
      AND t.entity_type IN ('alliance_org', 'economy', 'chain_node')
      AND e.source_name <> '' AND e.source_url <> '' AND e.verified_at IS NOT NULL
    GROUP BY e.from_entity_id, e.to_entity_id, e.relation_type
    HAVING COUNT(*) > 1
) d;

SELECT 'integrity', 'chain_relation_orphan_or_invalid_endpoint', COUNT(*)
FROM chain_node_relations r
LEFT JOIN entity_nodes f ON f.id = r.from_chain_node_entity_id
LEFT JOIN entity_nodes t ON t.id = r.to_chain_node_entity_id
WHERE r.status = 'active'
  AND (f.id IS NULL OR t.id IS NULL OR f.status <> 'active' OR t.status <> 'active'
       OR f.entity_type <> 'chain_node' OR t.entity_type <> 'chain_node');

SELECT 'integrity', 'chain_relation_duplicate_tuple', COALESCE(SUM(n - 1), 0)
FROM (
    SELECT from_chain_node_entity_id, to_chain_node_entity_id, relation_type, COUNT(*) n
    FROM chain_node_relations
    WHERE status = 'active'
    GROUP BY from_chain_node_entity_id, to_chain_node_entity_id, relation_type
    HAVING COUNT(*) > 1
) d;

COMMIT;
SQL
```

只读 Neo4j preflight 使用容器内 `NEO4J_AUTH` 拆分 username/password，不输出凭证：

```sh
docker exec -i tidewise-local-neo4j sh -lc '
auth="$NEO4J_AUTH"
user="${auth%%/*}"
pass="${auth#*/}"
exec /var/lib/neo4j/bin/cypher-shell -u "$user" -p "$pass" -d neo4j --format plain
' <<'CYPHER'
CALL dbms.components() YIELD name, versions, edition RETURN name, versions[0], edition;
SHOW DATABASES YIELD name, type, access, currentStatus WHERE name = 'neo4j' RETURN name, type, access, currentStatus;
MATCH (n {projection_namespace: 'tidewise'}) RETURN count(n) AS node_count;
MATCH (n:Entity {projection_namespace: 'tidewise'}) RETURN n.entity_type AS entity_type, count(*) AS count ORDER BY entity_type;
MATCH ()-[r]->() WHERE r.projection_namespace = 'tidewise' RETURN type(r) AS relationship_type, count(*) AS count ORDER BY relationship_type;
MATCH (n) WHERE n.projection_namespace IS NOT NULL AND n.projection_namespace <> 'tidewise' RETURN count(n) AS other_namespace_nodes;
MATCH ()-[r]->() WHERE r.projection_namespace IS NOT NULL AND r.projection_namespace <> 'tidewise' RETURN count(r) AS other_namespace_relationships;
MATCH (n:Entity {projection_namespace: 'tidewise'})-[r]-(m) WHERE coalesce(m.projection_namespace, 'missing') <> 'tidewise' OR coalesce(r.projection_namespace, 'missing') <> 'tidewise' RETURN count(DISTINCT r) AS cross_namespace_relationships;
SHOW CONSTRAINTS YIELD name RETURN count(name) AS constraint_count;
SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties, state, populationPercent RETURN name, type, entityType, labelsOrTypes, properties, state, populationPercent ORDER BY name;
CYPHER
```

## 唯一 cleanup 命令

**以下命令当前不得执行。只有用户明确授权 `local-neo4j-foundation-cleanup` 后，且全部 preflight 通过时，才允许执行一次。**

```sh
docker exec tidewise-local-neo4j sh -lc '
auth="$NEO4J_AUTH"
user="${auth%%/*}"
pass="${auth#*/}"
exec /var/lib/neo4j/bin/cypher-shell -u "$user" -p "$pass" -d neo4j --format plain "MATCH (entity:Entity {projection_namespace: '\''tidewise'\''}) DETACH DELETE entity"
'
```

该命令只有一个写语句，不包含 `CREATE`、`MERGE`、`SET`、`DROP`、constraint/index/database/configuration 操作，也不调用 projector。预期影响是删除本文冻结的 574 个 Tidewise `Entity` 节点及其 440 条同 namespace 关系。

## cleanup 后 Query / assert

cleanup 命令成功返回后只允许执行以下只读验收，不得顺带 rebuild：

```cypher
MATCH (n {projection_namespace: 'tidewise'})
RETURN count(n) AS tidewise_nodes;

MATCH ()-[r]->()
WHERE r.projection_namespace = 'tidewise'
RETURN count(r) AS tidewise_relationships;

MATCH (n)
WHERE n.projection_namespace IS NOT NULL
  AND n.projection_namespace <> 'tidewise'
RETURN count(n) AS other_namespace_nodes;

MATCH ()-[r]->()
WHERE r.projection_namespace IS NOT NULL
  AND r.projection_namespace <> 'tidewise'
RETURN count(r) AS other_namespace_relationships;

SHOW DATABASES YIELD name, type, access, currentStatus
WHERE name = 'neo4j'
RETURN name, type, access, currentStatus;

SHOW CONSTRAINTS YIELD name
RETURN count(name) AS constraint_count;

SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties, state, populationPercent
RETURN name, type, entityType, labelsOrTypes, properties, state, populationPercent
ORDER BY name;
```

必须同时满足：

- `tidewise_nodes=0`、`tidewise_relationships=0`。
- other namespace nodes/relationships 仍为 0。
- database 仍为 `neo4j / standard / read-write / online`。
- constraints 仍为 0；两个 lookup indexes 仍 ONLINE 且 metadata 不变。
- 两个非敏感配置文件 hash 不变，容器 image identity 不变。
- 重新执行完整 PG read-only baseline，所有 count/hash/schema/integrity 逐项不变。
- 未产生 Neo4j backup/rollback，未执行 rebuild/project/sync，未创建 graph projection run。

## Stop Conditions

任一条件成立即 fail-closed；cleanup 不得执行，若已执行则停止在空投影验收状态并回到主对话：

- 环境不是本文冻结的 local 容器/database/namespace，或 container image/config hash 漂移。
- PostgreSQL identity、Goose、schema、任一 source count/hash、endpoint/orphan/duplicate 发生漂移。
- Neo4j database 非 online，Tidewise preflight counts/type 漂移，出现其他 namespace、cross-namespace、non-Entity、invalid endpoint、duplicate 或 legacy 状态。
- constraint/index metadata 漂移。
- 没有收到只针对 `local-neo4j-foundation-cleanup` 的明确 R3 授权，或授权文本同时夹带 rebuild/sync。
- cleanup 命令非零退出、超时、连接中断或结果不确定；不得重试，重新执行必须再次授权。
- 任一 postcondition 失败；不得在本包中用 rebuild 修复。

## 用户授权请求

请仅在接受上述 environment、scope、预期删除量、`approved-disposable-recovery`、无 Neo4j backup/rollback、preflight/postconditions 与停止条件时，明确授权：

> 授权执行 `local-neo4j-foundation-cleanup`，仅清空 local Neo4j 的 Tidewise namespace，并按本授权包 Query 验收；不授权 rebuild、sync、PostgreSQL 写入或 Package 2。
