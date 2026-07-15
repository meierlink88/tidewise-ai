# usable-map 最终关系 Neo4j Sync R3 Review 包

## 状态与授权边界

- 命名层：`usable-map-final-relations-neo4j-sync`。
- 风险：R3，local Neo4j disposable projection cleanup + full rebuild sync。
- 当前状态：`prepared_for_r3_review`；task 2.9 保持未完成。
- 本 checkpoint 只完成代码审计、PostgreSQL/Neo4j 只读基线冻结与执行设计；未执行 Neo4j cleanup、rebuild、sync，未写 PostgreSQL，未创建 backup，未修改 schema 或源码。
- 本包不构成 R3 执行授权。只有后续授权精确命名本层、环境、入口与完整 Scope 后，才允许执行一次。
- 旧 checkpoint `e8b2658aa42c28863b7e82b848b1ef6434773d2f` 及 `reviews/all-chain-node-relations-neo4j-sync-r3.md` 已 superseded，禁止调用、恢复或作为授权依据。

## 同步策略结论

现有 `project-entities` 会对全部节点和关系执行 `MERGE/SET`，但不会删除 PostgreSQL target 中不存在的 extra/legacy edge，因此不能单独证明 Neo4j 最终集合与 PG 完全一致。

本包选择现有 `rebuild-entities` 作为唯一入口：该命令先通过 `DeleteNamespace` 删除 local Tidewise namespace，再读取最终 PG projection snapshot 并全量重建。这样 after 集合只可能来自本包冻结的 PG baseline，能同时消除任何执行前 extra/legacy edge；不新增 relation-only mode、framework、service 或源码改造。

该入口会：

1. 在 PostgreSQL 创建一条 `graph_projection_runs` running audit；
2. 删除 local Neo4j `projection_namespace=tidewise` 的全部 `Entity` 节点及其关系；
3. 从 PG 读取并写入 981 个节点、133 条 `entity_edges` 和 212 条 `chain_node_relations`；
4. 将 audit 完成态写为 `rebuild_entities/succeeded`。

Neo4j 写入和 PostgreSQL audit 不在同一事务。任何失败都必须停止，不得自动 retry、再次 cleanup/rebuild、改用 `project-entities` 或 forward-fix。

## 环境 identity

| 项目 | 冻结值 |
|---|---|
| Git checkpoint | R2 accepted commit `c0c4f9dc8dabe5257cd71522a64124502a0cbaf1`；本 R3 Review checkpoint 待提交 |
| PostgreSQL container | `tidewise-local-postgres` / `postgres:16` / image `sha256:be01cf82fc7dbba824acf0a82e150b4b360f3ff93c6631d7844af431e841a95c` / healthy |
| PostgreSQL database | `tidewise_local` / user `tidewise` / PostgreSQL `16.14 (Debian 16.14-1.pgdg13+1)` |
| Neo4j container | `tidewise-local-neo4j` / `neo4j:5-community` / image `sha256:4bae36aff76271e27fd6a6ed0835413f86a284cd179cfb1cb7d188f5f7533aca` / healthy |
| Neo4j database | `neo4j` / `online` / `read-write` / primary writer |
| Projection namespace | `tidewise` |
| Environment boundary | local only；其他 namespace、shared-local、UAT、prod/shared 全部排除 |

## PostgreSQL accepted baseline

| 项目 | 冻结值 |
|---|---|
| Goose | max applied `18`；applied rows `19` |
| projected nodes | `981`：alliance_org `45`、economy `94`、chain_node `842` |
| node identity SHA-256 | `e17c1fb837a946ed0dc3ea0efc999d03b936046061cecc0423e2b449041a9800` |
| in-scope entity_edges | `133`：member_of `133` |
| active chain_node_relations | `212`：is_subcategory_of `108`、is_component_of `3`、input_to `93`、depends_on `8` |
| chain relation identity SHA-256 | `2c90c52397149b9bcecd338b8d9c9acb66087395441f8f197d4045b39e0f746b` |
| chain relation content SHA-256 | `f37f3ceae11712606682ff7301e0920967cf635f1b4476861088974f6f324bac` |
| target graph relationship identity/direction SHA-256 | `89278c6fb69420b133d00514566c7efd79e71aef9d4fc14e099d41b31e082f25` |
| relation schema columns / constraints MD5 | `30989050ddac02d7b70f0eeb8c510d19` / `a3779c06528cfb2fbf469d7ced849199` |
| chain relation integrity | orphan/invalid endpoint、duplicate ID/tuple、self-loop、illegal/legacy type 均为 `0` |
| projection source rows | `1326 = 981 nodes + 133 entity_edges + 212 chain_node_relations` |
| projection audit | `graph_projection_runs=17`；`graph_projection_run_items=2`；latest=`project_entities/succeeded/1210/1210/0/0`，namespace `tidewise` |

关系 identity/direction canonical 行固定为 `edge_id|from_entity_id|NEO4J_RELATION_TYPE|to_entity_id`，按 `edge_id` 升序、UTF-8、LF 分隔且末尾无额外 LF。`member_of`、`is_subcategory_of`、`is_component_of`、`input_to`、`depends_on` 分别映射为 `MEMBER_OF`、`IS_SUBCATEGORY_OF`、`IS_COMPONENT_OF`、`INPUT_TO`、`DEPENDS_ON`。

842 个 active chain_node endpoint 与已批准 baseline 集合精确一致；212 条 chain relation 的两端全部属于该集合。

受保护 PG 范围：

| 集合 | Count | Full-row MD5 |
|---|---:|---|
| entity_nodes 全表 | 1387 | `7222adbd427a00756fdf6b1108cb664c` |
| active chain_node subset | 842 | `cca5eca3f360b1d95340130652beab52` |
| chain_node_profiles | 842 | `0ecad0af7035e81f1e63c0cd8510d790` |
| entity_external_identifiers | 1169 | `791ed08c3486b13b8d362247db539502` |
| entity_edges | 241 | `df46fa3c6170c9f9beabc0b27ceedacf` |
| chain_node_physical_constraints | 0 | `d41d8cd98f00b204e9800998ecf8427e` |

## Neo4j pre-sync baseline

| 项目 | 冻结值 |
|---|---|
| Tidewise nodes | `981`：alliance_org `45`、economy `94`、chain_node `842` |
| node identity SHA-256 | `e17c1fb837a946ed0dc3ea0efc999d03b936046061cecc0423e2b449041a9800`，与 PG target 一致 |
| Tidewise relationships | `229`：MEMBER_OF `133`、IS_SUBCATEGORY_OF `95`、IS_COMPONENT_OF `1`、INPUT_TO `0`、DEPENDS_ON `0` |
| current relationship identity/direction SHA-256 | `75140d54a4bef43c5e9c5778b162f3b9d84b09dbf6d0c456036346986adbcd30` |
| target diff | missing `116`、extra `0`；target `345`、current `229` |
| namespace integrity | other namespace nodes `0`；cross-namespace relationships `0` |
| graph integrity | duplicate entity `0`；duplicate edge `0`；legacy/unsupported relation `0` |
| constraints / indexes | constraints `0`；lookup indexes `index_343aff4e`、`index_f7700477` 均 `ONLINE/100.0` |

当前 Neo4j 只投影 96 条 chain relation；PG 已接受 212 条。缺失的 116 条按类型为 IS_SUBCATEGORY_OF `13`、IS_COMPONENT_OF `2`、INPUT_TO `93`、DEPENDS_ON `8`。

## 精确 Scope 与排除

**唯一允许 Scope：**

- 仅 local 容器 `tidewise-local-neo4j`、database `neo4j`、`projection_namespace=tidewise`。
- 使用冻结 PG accepted baseline 执行一次 Neo4j sync；实现方式是现有 `rebuild-entities` 的 namespace cleanup + full rebuild。
- cleanup 只删除 Tidewise namespace 的 `Entity` 节点及其关系；database、constraints、indexes、configuration 和其他 namespace 必须保留。
- rebuild 只写 active alliance_org/economy/chain_node、端点均在该节点集合内的 133 条 entity_edges 与 212 条 chain_node_relations。
- 接受且只接受一条 PG projection audit metadata：runs `17 -> 18`；业务事实不得变化。

**明确排除：**

- `project-entities`、旧 4-edge 包、旧 checkpoint `e8b2658`、manual Cypher、第二次 projection 或自动 retry。
- PostgreSQL 业务写、seed、migration、restore 或 Neo4j backup/rollback。
- blocked/rejected 候选、physical constraints、market/index/benchmark、事件、观测、推理、派生关系或查询 API。
- 其他 namespace、shared-local、UAT、prod/shared、Package 3。

## Recovery Evidence

`Recovery Evidence=approved-disposable-recovery`。PostgreSQL frozen accepted baseline 是唯一恢复来源；local Neo4j 是 disposable projection，不创建 Neo4j backup/rollback。

若执行失败，必须保留实际 partial/empty 状态并立即停止报告；本授权不得自动 retry、再次 cleanup/rebuild、切换入口或恢复。任何恢复动作必须重新 Review 并取得新的明确 R3 授权。

## Fresh preflight

未来执行前必须逐项重新验证，任一不一致立即停止：

1. HEAD/upstream/clean 等于本 Review checkpoint；授权精确命名 `usable-map-final-relations-neo4j-sync`、local 环境、唯一入口和 cleanup+rebuild Scope。
2. PostgreSQL/Neo4j container image、health、database、namespace 与本文一致；Neo4j database 仍 online/read-write，constraints/index metadata 未漂移。
3. PG Goose、schema MD5、981/133/212 source counts、842 endpoint scope、chain identity/content hash、target graph hash、完整性与受保护表 hashes 全部一致。
4. Neo4j pre-sync 必须仍为 981/229、类型分布与 current hash 一致；missing/extra=`116/0`，other/cross namespace、duplicate、legacy 均为 0。
5. `graph_projection_runs/items=17/2`，latest run 与本文一致。
6. 确认未调用旧包、`project-entities`、manual Cypher 或其他写入口；凭据仅从 local 进程环境注入且不得打印。

## 唯一执行入口

全部 preflight 与新的明确 R3 授权通过后，只允许从 `backend/` 单次执行：

```sh
APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/graph-projector rebuild-entities
```

不得增加参数、拆分为 manual cleanup + project、改用 `project-entities`、执行第二次命令或自动 retry。非零退出、超时、连接中断或结果不确定均立即停止。

CLI 必须精确报告：mode=`rebuild_entities`、status=`succeeded`、source_rows=`1326`、projected=`1326`、skipped=`0`、failed=`0`。

## 写后 Query / Assert

单次命令成功后必须立即只读证明：

1. Neo4j Tidewise nodes=`981=45/94/842`，node identity SHA-256=`e17c1fb837a946ed0dc3ea0efc999d03b936046061cecc0423e2b449041a9800`。
2. Neo4j relationships=`345=133 MEMBER_OF / 108 IS_SUBCATEGORY_OF / 3 IS_COMPONENT_OF / 93 INPUT_TO / 8 DEPENDS_ON`。
3. relationship identity/direction SHA-256=`89278c6fb69420b133d00514566c7efd79e71aef9d4fc14e099d41b31e082f25`；PG target 对 Neo4j missing/extra=`0/0`。
4. duplicate entity/edge、invalid endpoint、legacy/unsupported、other namespace、cross-namespace relationships 全部为 `0`。
5. database 仍 online/read-write；constraints=`0`；两个 lookup indexes 的 name/state/type/population metadata 不变。
6. PostgreSQL chain relations 仍为 `212=108/3/93/8`，identity/content hashes、Goose、schema 与所有受保护表 count/hash 不变。
7. `graph_projection_runs=18`、items=`2`；latest mode=`rebuild_entities`、status=`succeeded`、source/projected=`1326/1326`、skipped/failed=`0/0`、namespace=`tidewise`。

不用第二次 projection 验证幂等。全量 target hash、edge_id 唯一、missing/extra=0 和 writer 的 `(edge_id, projection_namespace)` identity contract 作为只读幂等证据。

## Stop Conditions

以下任一情况立即停止，不得自动恢复：环境、HEAD、namespace、PG/Neo4j count/hash/schema、842 endpoint、backup exemption、audit baseline 或 index metadata 漂移；旧包或错误入口被调用；命令非零、超时、连接不确定或疑似部分写；CLI source/projected/skipped/failed 与预期不符；任一写后 Query/assert 失败。

本包完成后只允许等待独立 Review 与后续明确 R3 授权；不得执行 Neo4j、进入 Package 3、Sync、Archive、PR 或 cleanup。
