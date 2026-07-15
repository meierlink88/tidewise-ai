# usable-map additive 关系 PostgreSQL Write R2 execution 证据

## 状态与授权边界

- 命名操作：`usable-map-additive-relations-postgres-write`。
- 环境：local 容器 `tidewise-local-postgres`、数据库 `tidewise_local`、PostgreSQL `16.14`。
- 执行前 HEAD/upstream：`654be674ceef680ed511ab7d052679b21c0bf913`；worktree clean。
- 本层只执行一次 additive relation Write；未执行手工 SQL、第二次 Write、restore、Neo4j、R3 或 Package 3。
- local development PostgreSQL 凭据仅在进程环境中注入，未打印、记录或写入 artifact。

## 写前连续 stateful preflight

所有断言在同一连续 preflight 中逐项通过：

| 项目 | 实际值 |
|---|---|
| Database / server / Goose | `tidewise_local` / PostgreSQL `16.14` / max applied `18`、applied rows `19` |
| relation columns MD5 | `30989050ddac02d7b70f0eeb8c510d19` |
| relation constraints canonical MD5 | `a3779c06528cfb2fbf469d7ced849199` |
| schema contract | `7/2/1/1/7/2/1/4/3/0` |
| 写前 relations | `100 = 95 is_subcategory_of / 1 is_component_of / 3 input_to / 1 depends_on` |
| accepted identity / content SHA-256 | `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` / `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7` |
| active endpoints / profiles / scope 外 | `842 / 842 / 0`；artifact 与 PG endpoint-set SHA 一致 |
| duplicate ID / tuple、self-loop、illegal、incomplete、orphan、inactive | 全部 `0` |
| 写前 dry-run | `created=112 / updated=0 / unchanged=100`；目标类型 `108/3/93/8` |

constraints hash 使用冻结原始口径，仅覆盖 `chain_node_relations`：

```sql
SELECT md5(COALESCE(string_agg(concat_ws('|',conname,contype,pg_get_constraintdef(oid,true)),E'\n' ORDER BY conname),''))
FROM pg_constraint
WHERE conrelid='chain_node_relations'::regclass;
```

写前 recovery evidence 保持不变：

| 项目 | 实际值 |
|---|---|
| Backup path | `/private/tmp/tidewise_local_pre_usable_map_additive_relations_postgres_write_20260715T044701Z.dump` |
| Bytes / mode | `1090701` / `0600` |
| SHA-256 | `098f07196f06a07a6f827748233080bfb28cbd4a97d145685f29a11dfd41611b` |
| Partial | 不存在 |

## 唯一 Write

从 `backend/` 单次执行：

```sh
APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/entity-seed \
  -chain-node-relation-manifest ../openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-usable-map-r0/additive-final-candidate-manifest.json \
  -chain-node-relation-approved-data-write
```

CLI exit `0`，精确报告：

```json
{"created":112,"updated":0,"unchanged":100,"by_relation_type":{"depends_on":8,"input_to":93,"is_component_of":3,"is_subcategory_of":108}}
```

未 retry、未执行第二次 Write、未 restore 或 forward-fix。

## 写后 Query / Assert

| 项目 | 实际值 |
|---|---|
| relations | `212 = 108 is_subcategory_of / 3 is_component_of / 93 input_to / 8 depends_on` |
| accepted 100 identity / content SHA-256 | `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` / `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7` |
| additive 112 identity / content SHA-256 | `1a25a8e742b5034c82e5b730f06ff101fdfeeb79727c275389c32b002c04189f` / `637e7520aaa248640636019b1a2ceb4de380b513b642da48acb63550bf5147f1` |
| combined 212 identity / content SHA-256 | `2c90c52397149b9bcecd338b8d9c9acb66087395441f8f197d4045b39e0f746b` / `f37f3ceae11712606682ff7301e0920967cf635f1b4476861088974f6f324bac` |
| duplicate ID / tuple、self-loop、illegal、incomplete、orphan、inactive、baseline 外 endpoint | 全部 `0` |
| post-write dry-run | `created=0 / updated=0 / unchanged=212`；类型 `108/3/93/8` |

受保护范围写前写后精确一致：

| 集合 | Count | Full-row MD5 |
|---|---:|---|
| entity_nodes 全表 | 1387 | `7222adbd427a00756fdf6b1108cb664c` |
| active chain_node subset | 842 | `cca5eca3f360b1d95340130652beab52` |
| chain_node_profiles | 842 | `0ecad0af7035e81f1e63c0cd8510d790` |
| entity_external_identifiers | 1169 | `791ed08c3486b13b8d362247db539502` |
| entity_edges | 241 | `df46fa3c6170c9f9beabc0b27ceedacf` |
| chain_node_physical_constraints | 0 | `d41d8cd98f00b204e9800998ecf8427e` |

Goose 仍为 max applied `18` / applied rows `19`；未修改 schema、节点、profile、external identifier、entity edge 或 physical constraint。

## 下一步边界

task 2.8 已完成。本 checkpoint 只等待独立 R2 验收；不得据此自动执行 Neo4j sync、R3、Apply-final、Sync、Archive 或 Deliver。
