# 全量 chain_node_relations Neo4j Sync R3 Review 包（已 superseded）

> **Superseded / 禁止执行：** 2026-07-15 用户批准继续开展 842-node usable-map additive 分析，既有 100 条不再是 Package 2 最终集合。本包仅针对其中尚未投影的 4 条边，已失效且不得作为任何 R3 授权或执行依据。未来只能在 additive PostgreSQL accepted baseline 验收后重新准备 `usable-map-final-relations-neo4j-sync` 独立 R3 包。

## 状态与结论

- 命名层：`all-chain-node-relations-neo4j-sync`。
- 风险：R3，local Neo4j projection write。
- 当前状态：`superseded_non_executable`，**不可执行、不可申请写入授权**。
- 本 checkpoint 只完成代码审计、PG/Neo4j 只读基线冻结与条件式执行设计；未修改源码，未写 PostgreSQL/Neo4j，task 2.6 保持未完成。
- 已确认的业务 Scope 只允许把 PG accepted baseline 中新增的 3 条 `INPUT_TO` 与 1 条 `DEPENDS_ON` 同步到既有 chain_node 端点，不允许清空、重建节点或重复投影其他关系。
- 现有 `graph-projector project-entities` 没有 relation-only/filter mode，会重新 `MERGE/SET` 全部 981 个节点与全部 233 条关系，并向 PostgreSQL 写一条 projection run 审计记录。因此该入口超出当前批准 Scope，不得执行。

## 环境 identity

| 项目 | 冻结值 |
|---|---|
| PostgreSQL container | `tidewise-local-postgres` / `postgres:16` / image `sha256:be01cf82fc7dbba824acf0a82e150b4b360f3ff93c6631d7844af431e841a95c` / healthy |
| PostgreSQL database | `tidewise_local` / user `tidewise` / PostgreSQL `16.14` |
| Neo4j container | `tidewise-local-neo4j` / `neo4j:5-community` / image `sha256:4bae36aff76271e27fd6a6ed0835413f86a284cd179cfb1cb7d188f5f7533aca` / healthy |
| Neo4j database | `neo4j` / standard / read-write / online / Neo4j `5.26.28 community` |
| Projection namespace | `tidewise` |
| Environment boundary | local only；UAT/prod/shared 与其他 namespace 明确排除 |

## PostgreSQL accepted baseline

| 项目 | 冻结值 |
|---|---|
| Goose | max applied `18`；applied rows `19` |
| active chain_node / profiles | `842 / 842` |
| external identifiers / entity_edges | `1169 / 241` |
| projected nodes | `981`：alliance_org `45`、economy `94`、chain_node `842` |
| in-scope entity_edges | `133`：member_of `133` |
| active chain_node_relations | `100`：is_subcategory_of `95`、is_component_of `1`、input_to `3`、depends_on `1` |
| physical constraints | `0` |
| relation identity SHA-256 | `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` |
| relation content SHA-256 | `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7` |
| chain_node identity/profile MD5 | `d6b53dce56fb5ca72ec77eef816f0a4b` / `2876324fb6bffa41967812702c6bc038` |
| integrity | orphan/invalid endpoint `0`；duplicate tuple `0`；duplicate ID `0`；self-loop `0`；legacy type `0`；incomplete evidence/provenance/verified_at `0` |
| projection audit | `graph_projection_runs=17`；`graph_projection_run_items=2`；latest run 为 `project_entities/succeeded/1210/1210/0/0`，namespace `tidewise` |

当前 projector 的完整 PG source snapshot 为 `981 nodes + 133 entity_edges + 100 chain_node_relations = 1214 rows`。

## Neo4j pre-sync baseline

| 项目 | 冻结值 |
|---|---|
| Tidewise nodes | `981`：alliance_org `45`、economy `94`、chain_node `842` |
| Tidewise relationships | `229`：MEMBER_OF `133`、IS_SUBCATEGORY_OF `95`、IS_COMPONENT_OF `1`、INPUT_TO `0`、DEPENDS_ON `0` |
| Existing relationship identity/direction SHA-256 | `7b6f92c6ad2eeeda9a8067e88204f90bd8d02a449af6e8df08dfe21453120dd1` |
| PG existing-vs-Neo4j row diff | `0`（从 target 排除 3 INPUT_TO + 1 DEPENDS_ON 后共 229 行） |
| Namespace integrity | other namespace nodes/relationships `0/0`；cross-namespace relationships `0` |
| Graph integrity | duplicate entity `0`；duplicate edge `0`；invalid endpoint `0`；legacy/unsupported relation `0` |
| Constraints / indexes | constraints `0`；node/relationship 两个 lookup indexes 均 ONLINE、population `100.0` |

## 仅允许同步的四条关系

| edge_id | from | type | to |
|---|---|---|---|
| `6b1dbf06-8bf8-590c-8d4f-4bc89608fb4f` | 半导体 | DEPENDS_ON | 半导体设备 |
| `8c9f519b-8671-5a94-bf02-9ac09e0f0dcc` | 光伏主材 | INPUT_TO | 光伏电池组件 |
| `54586f0d-a2a9-56c5-90eb-84cd379db8b7` | 半导体材料 | INPUT_TO | 半导体 |
| `6615b7df-17b0-5593-ac2f-fdfc735fe53a` | 锂 | INPUT_TO | 锂电池 |

四条关系的 from/to UUID 均属于冻结的 842 active profiled chain_node；方向保持 PostgreSQL 原始事实。

## Scope 与明确排除

**允许的目标 Scope：**

- 仅 local `neo4j` database、`projection_namespace=tidewise`。
- 只新增上述 3 条 `INPUT_TO` 与 1 条 `DEPENDS_ON`，端点复用既有 chain_node。
- PostgreSQL 是唯一事实源；Neo4j 不反写 PostgreSQL。
- 目标为 981 nodes / 233 relationships：MEMBER_OF `133`、IS_SUBCATEGORY_OF `95`、IS_COMPONENT_OF `1`、INPUT_TO `3`、DEPENDS_ON `1`。
- 目标 relationship identity/direction SHA-256：`9bbd936045d02dca1c21a6ad6e2dcafaef37fa8715d4c5dcd80dc709bc1b6ef2`。

**排除：**

- Neo4j cleanup、`rebuild-entities`、节点创建/重建/重投、已有 229 条关系重投。
- 其他 namespace、UAT、prod、shared。
- PostgreSQL 业务事实写、migration、seed、manual SQL、PG restore。
- 自动 retry、cleanup/rebuild recovery、forward-fix、查询 API、派生关系或 Package 3。

## 现有能力审计与 blocker

| 行为 | 代码证据 | 结论 |
|---|---|---|
| CLI mode | `backend/cmd/graph-projector/main.go` 只有 `check`、`project-entities`、`rebuild-entities` | 无 relation-only sync mode |
| PG read | `ProjectEntities` 固定调用 `ListGraphEntityNodes` 与 `ListGraphEntityEdges` | 固定读取 981 nodes 与全部 233 relationships |
| Neo4j node write | `UpsertEntities` 对全部节点执行 `MERGE` 后 `SET/REMOVE` properties | 会写全部 981 节点，超出当前 Scope |
| Neo4j relationship write | `UpsertRelationships` 按全部 relation types 分组，对每条执行 `MERGE` 后 `SET` | 会重投已有 229 条关系，超出当前 Scope |
| cleanup | 只有 mode=`rebuild_entities` 才调用 `DeleteNamespace` | `project-entities` 不 cleanup，但仍是 full upsert |
| PG audit | 投影开始前 `CreateGraphProjectionRun`，结束时 `CompleteGraphProjectionRun` | 每次命令会新增 1 条 `graph_projection_runs`；成功且无 skip/fail 时不新增 run item |
| failure atomicity | nodes 与各 relation type 是多次 Neo4j write，PG audit 与 Neo4j 不在同一事务 | 失败可能留下部分 projection update；必须停止，不得自动 retry/cleanup/rebuild |

候选命令如下，但在当前 Scope 下**明确禁止执行**：

```sh
APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/graph-projector project-entities
```

若未来独立 Review 扩大 Scope 并批准 full idempotent upsert，成功时精确预期为：

- Neo4j 981 nodes / 233 relationships，全部 1214 source rows 被报告为 projected，skipped/failed `0/0`。
- `graph_projection_runs` 从 `17 -> 18`；latest run 为 mode `project_entities`、status `succeeded`、source/projected `1214/1214`、config namespace `tidewise`。
- `graph_projection_run_items` 保持 `2`。

这些预期只描述代码事实，不构成当前 R3 授权。

## 条件式 preflight

只有 blocker 通过新的 Review 解决后，未来执行前才允许按以下顺序 fresh 复验；任一不一致立即停止：

1. branch/upstream clean，HEAD 等于后续获批的可执行 checkpoint；授权文本精确命名 `all-chain-node-relations-neo4j-sync`。
2. 容器、image、database、namespace 均等于本文；Neo4j database online/read-write，constraints/index metadata 未漂移。
3. PostgreSQL Goose、schema contract、981/133/100 source counts、保护 counts、identity/content hashes、endpoint/integrity 均等于本文。
4. Neo4j pre-sync 必须仍为 981/229，类型、identity/direction hash、namespace 与完整性均等于本文。
5. 只允许一个经 Review 的单次 sync 入口；若入口仍会写节点或已有关系，而授权未显式包含 full upsert，则不得执行。
6. 确认 `graph_projection_runs=17`、run items `2`；若执行入口会写审计，授权必须显式接受精确的 `17 -> 18`。

## 条件式单次 sync 与停止条件

- 当前不存在满足 Scope 的单次 sync 命令，因此本包不提供可执行命令。
- 若后续批准最小 R1 relation-only 实现，必须重新生成 Review 包，冻结新入口、tests、source filter、预期审计和写集合后再申请 R3。
- 若后续改为批准现有 `project-entities` full idempotent upsert，也必须显式扩大 Scope 到 981 nodes + 233 relationships，并接受 1 条 PG audit；本包不能替代该授权。
- 任一命令非零退出、超时、连接中断、结果不确定、部分写、skipped/failed 非零、count/hash/schema/namespace 漂移或 postcondition 失败时立即停止。
- 失败后不得自动 retry、cleanup、rebuild、restore、forward-fix 或执行第二次 projection；只保存脱敏 evidence 并回到 Review。

## 条件式写后 Query / Assert

未来合法执行完成后必须立即只读证明：

- Neo4j nodes 仍为 `981=45/94/842`；relationships 为 `233=133/95/1/3/1`。
- 四个 edge_id 各恰好 1 条，from/to、type、`original_relation_type`、`source=postgres_chain_node_relations` 与 PG 一致。
- 全部 233 关系 identity/direction SHA-256 为 `9bbd936045d02dca1c21a6ad6e2dcafaef37fa8715d4c5dcd80dc709bc1b6ef2`。
- PG accepted relation set 对 Neo4j 的 missing/extra 均为 0；duplicate entity/edge、orphan/invalid endpoint、legacy/unsupported、cross/other namespace 均为 0。
- database、constraints、indexes 与配置不变。
- PostgreSQL 业务事实、Goose、schema、100 relation identity/content hashes 与保护 counts 全部不变。
- projection audit 只允许等于重新 Review 后冻结的精确预期；不得接受额外 run 或 run item。

## 幂等验证

- 不用第二次 projection 验证幂等，因为那是未授权的第二次 Neo4j Write。
- 必须通过写入契约的 identity key 与只读结果验证：关系按 `(edge_id, projection_namespace)` `MERGE`，写后 233 edge_id 唯一、duplicate edge 为 0、target hash 精确一致。
- 若采用未来 relation-only 入口，其 targeted tests 必须先证明重复输入不会生成重复 edge；该 R1 tests 不授权任何 live Neo4j 写入。

## Recovery Evidence

`Recovery Evidence=approved-disposable-recovery`。仅适用于本地 Tidewise Neo4j projection：PostgreSQL accepted baseline 是唯一恢复来源，不创建 Neo4j backup/rollback。

若未来 sync 失败，不得在本授权中自动 cleanup/rebuild；必须停止并由新的独立 Review 决定恢复动作。该语义不适用于 UAT/prod/shared。

## 需要新的人工决定

当前存在确定性 scope blocker，必须选择并重新 Review 后才能申请 R3 执行授权：

1. 保持“只新增 4 条关系”的严格 Scope：先另行批准最小 R1 relation-only sync 入口与 targeted tests，再刷新本 R3 包；或
2. 显式扩大本 R3 Scope：允许现有 `project-entities` 对 981 nodes + 233 relationships 做 full idempotent upsert，并接受 `graph_projection_runs 17 -> 18`。

在该决定前，task 2.6 保持未完成，不得执行任何 Neo4j/PG 写入。

## Checkpoint 验证

- PG/Neo4j baseline、source/target hashes 与现有 229 行 diff 均由 fresh read-only Query 取得。
- `openspec validate rebuild-foundation-graph-and-enrich-chain-data --strict`：通过。
- explicit `TestOpenSpecTaskDesignLint`：通过。
- `git diff --check`、scoped file list 与 secret scan：通过；diff 仅含本 Review artifact。
