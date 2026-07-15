# usable-map 最终关系 Neo4j Sync R3 execution 证据

## 授权与执行边界

- 命名层：usable-map-final-relations-neo4j-sync，仅限 local 容器 tidewise-local-neo4j、database neo4j、projection_namespace=tidewise。
- 执行前 HEAD：bb3ced229280958cb12042863e86a68e09373bce，branch/upstream clean。
- 唯一入口：从 backend 单次运行 APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/graph-projector rebuild-entities。
- local PostgreSQL 与 Neo4j 凭据仅由冻结容器 metadata 在当前进程内桥接给单次 child process；未回显、未记录值、未写入文件或 artifact。
- 未调用 project-entities、旧 e8b2658/旧 4-edge 包、manual Cypher、第二次执行或自动 retry；未进入 Package 3。
- recovery=approved-disposable-recovery，以冻结 PostgreSQL accepted baseline 为唯一恢复来源；未创建 Neo4j backup/rollback。

## 写前连续 stateful preflight

全部检查逐项 PASS，最终结果为 PRECHECK=PASS failures=0：

| 检查 | 写前实际值 |
|---|---|
| PostgreSQL migration/schema | Goose 18 / 19 rows；columns MD5 30989050ddac02d7b70f0eeb8c510d19；constraints MD5 a3779c06528cfb2fbf469d7ced849199 |
| 投影节点 | 981 = alliance_org 45 / economy 94 / chain_node 842 |
| entity_edges | 133 |
| chain_node_relations | 212 = is_subcategory_of 108 / is_component_of 3 / input_to 93 / depends_on 8 |
| relation identity SHA-256 | 2c90c52397149b9bcecd338b8d9c9acb66087395441f8f197d4045b39e0f746b |
| relation content SHA-256 | f37f3ceae11712606682ff7301e0920967cf635f1b4476861088974f6f324bac |
| target graph SHA-256 | 89278c6fb69420b133d00514566c7efd79e71aef9d4fc14e099d41b31e082f25 |
| PostgreSQL audit | graph_projection_runs=17，run_items=2；latest project_entities succeeded 1210/1210/0/0 |
| Neo4j | 981 nodes / 229 relationships；PG→Neo missing/extra=116/0 |
| Neo4j完整性 | duplicate/orphan/legacy/other/cross namespace=0；database online/read-write；constraints=0；lookup indexes ONLINE |

## 单次执行结果

唯一入口仅调用一次，CLI 返回：

    run=3623f679-6312-5c84-a7af-80b868f9e576 status=succeeded source_rows=1326 projected=1326 skipped=0 failed=0

未发生非零退出、超时、连接不确定或第二次写入。

## 写后 Query/assert

写后只读检查全部 PASS，最终结果为 POSTFLIGHT=PASS failures=0：

| 检查 | 写后实际值 |
|---|---|
| Neo4j nodes | 981 = alliance_org 45 / economy 94 / chain_node 842 |
| Neo4j relationships | 345 = MEMBER_OF 133 / IS_SUBCATEGORY_OF 108 / IS_COMPONENT_OF 3 / INPUT_TO 93 / DEPENDS_ON 8 |
| Neo4j target SHA-256 | 89278c6fb69420b133d00514566c7efd79e71aef9d4fc14e099d41b31e082f25 |
| PG↔Neo diff | missing=0 / extra=0 |
| Neo4j完整性 | duplicate/orphan/legacy/other/cross namespace=0 |
| Neo4j基础设施 | database online/read-write；constraints=0；两个 lookup indexes metadata 不变 |
| PostgreSQL业务事实 | nodes=981、entity_edges=133、chain_node_relations=212=108/3/93/8，identity/content hashes 不变 |
| PostgreSQL schema/保护范围 | Goose、schema MD5 与受保护表 count/full-row MD5 全部不变 |
| PostgreSQL audit | graph_projection_runs=18，run_items=2；latest rebuild_entities succeeded 1326/1326/0/0，namespace=tidewise |

## 结论

task 2.9 的 local Tidewise namespace cleanup + final accepted baseline 全量 rebuild 已按单次 R3 授权完成并验收。旧 checkpoint e8b2658 与旧 4-edge 包未调用；本 checkpoint 停止等待独立验收，不进入 Package 3。
