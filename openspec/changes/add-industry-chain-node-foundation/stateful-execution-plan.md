# 产业链 Stateful 执行计划

## 1. 当前边界

- 本文只定义后续有状态执行顺序，本 checkpoint 不执行任何命令写入 PostgreSQL 或 Neo4j。
- 可执行 seed：`backend/data/entity_foundation/industry_chains_v1.json`，包含 2 条 industry chain、21 个新增 chain node、5 个既有 node profile 改进、27 个 membership、24 条 canonical topology；不含 physical constraint、`mapped_to_sector`、economy、commodity 或 benchmark 关系。
- Review-only fixture：`backend/data/entity_foundation/review/industry_chain_candidates_v1.json`，包含 15 条 physical constraint candidate 和 12 条 `mapped_to_sector` candidate。它不在 `DefaultSeedPaths`，不得传给 repository。
- 所有预计数量均以当前 repo seed 与空的产业链新表为基线；执行前必须先做只读 preflight，若数据库已有同 ID 数据则把对应 created 调整为 updated/unchanged，并重新提交用户 Review。

## 2. 分层执行矩阵

| 层 | 前置条件与独立授权 | 预计 created / updated / unchanged | 影响范围 | 验证查询 | 回滚或停用方案 |
|---|---|---|---|---|---|
| 1. Migration | 用户单独批准执行 `000014_add_industry_chain_foundation.sql`；确认备份、当前 migration version=13、无同名表/约束漂移 | migration 1；新表 4；`chain_node_profiles` 新增列 4；业务行 0 | `industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints`、`chain_node_profiles` | 查询 migration version；检查 4 表、字段、FK、CHECK、索引和 `generated_by_ai`；确认既有 33 个 chain node 数量与 UUID 不变 | 本项目 Down 为 reviewed no-op；失败时事务回滚。成功后若需撤销，停止后续写入并通过新的 forward migration 处理，不手工 DROP |
| 2. Chain / node master | 层 1 验收；用户批准 2 条 chain、21 个新增 node 与 5 个 profile 改进 | `entity_nodes`: created 23；`industry_chain_profiles`: created 2；`chain_node_profiles`: created 21、updated 5；重复执行预计 unchanged 51 | `entity_nodes`、`industry_chain_profiles`、`chain_node_profiles` | 按 stable key 查询 2 chain + 26 pilot node；核对名称、aliases、definition、category、unit、granularity、review status；运行 seed report | 在不删除既有 5 个 node 的前提下，将本批新增 23 个实体置 inactive；5 个 profile 改进需用 reviewed forward seed 恢复旧值；不物理删除 |
| 3. Membership | 层 2 验收；用户批准两链 12/15 membership | created 27；重复执行 unchanged 27 | `industry_chain_memberships` | 按 chain 查询 active membership，断言 AI=12、semiconductor=15、共享 `advanced_packaging` 两条 membership、stage order 唯一稳定 | 将本批 27 条 membership 置 inactive；保留审计行；后续 rebuild 前重新查询 active 数量 |
| 4. Canonical topology | 层 3 验收；用户批准 10 条 AI + 14 条 semiconductor topology 及 evidence note 中的缺口 | created 24；重复执行 unchanged 24 | `industry_chain_topology_edges` | 按 chain/relation type 查询；断言无 self edge、端点均为同链 active membership、无 `substitutes_for`、无反向 `depends_on` 重复 | 将本批 24 条 topology 置 inactive；不删除 membership；重新查询 active topology=0 |
| 5. Physical constraint | 15 条 candidate 逐项补齐权威技术证据并取得人工 Review；每条写入前单独确认 approval gate；当前未授权 | 当前可执行 0；Review-only candidate 15；未来数量以逐项批准结果为准 | 仅 `industry_chain_physical_constraints`；不进入 `entity_nodes` 或 Neo4j | 按 chain/node/topology edge 查询；断言 `review_status=approved`、`generated_by_ai=true`、主体同链 active、constraint type 属于 13 类 | 将获批写入行置 inactive；不改 topology；不触发 Neo4j rebuild。未批准 candidate 始终留在 Review fixture |
| 6. `mapped_to_sector` | 12 条 candidate 逐项确认分析映射、来源和端点；当前未授权。economy/commodity/benchmark 清单保持空 | 当前可执行 0；Review-only candidate 12；未来数量以逐项批准结果为准 | 经批准后仅 `entity_edges` 的 `mapped_to_sector`；不得创建海外 market `covers_sector` 中国 sector | 查询 relation type、from/to stable key、source；断言无 economy/commodity/benchmark 新关系、无海外 market `COVERS_SECTOR` | 将本批获批 entity edge 置 inactive；不改 sector 主数据；重新投影前查询 active edge=0 |
| 7. Neo4j rebuild | 层 2–4 PostgreSQL 验收完成；若层 6 有获批关系则先完成其 PG 验收；用户单独批准 rebuild。Physical constraint 不构成前置写入 | 预计新增 Entity 23（2 chain + 21 new node）；新增关系 51（27 membership + 24 topology）；既有 5 node 为 upsert unchanged；若层 6 未批准则 sector mapping 0 | Neo4j `projection_namespace=tidewise` 下统一 `Entity` 与 membership/topology/已审阅 entity edges；排除 physical constraints 和 observations | 查询 2 chain、26 pilot node、27 membership、24 topology；验证 benchmark/sector 路径仅在已批准 edge 存在时成立；断言无 constraint 节点/边、无海外 market 覆盖中国 sector | 从 PostgreSQL active facts 重新 rebuild；若需撤销，先按对应 PG 层置 inactive并验收，再单独授权 rebuild；不得直接手工删局部图 |
| 8. Query 验收 | 每次 Write 或 Rebuild 后另行授权只读验收；不得用上一层授权推定 | 只读，created/updated=0 | PostgreSQL report/query 与 Neo4j read query | 对照每层预计数量、stable key、状态、来源、ID 和路径；差异立即停止下一层 | 无写状态；记录差异并回到对应层 Review，不自动修复 |

## 3. 必须逐层取得的用户授权

1. 执行 migration。
2. 写入 chain / node master。
3. 写入 membership。
4. 写入 canonical topology。
5. 逐项批准并写入 physical constraint；当前 15 条全部未授权。
6. 逐项批准并写入 `mapped_to_sector`；当前 12 条全部未授权。
7. Neo4j rebuild。
8. 每层 Query 验收。

任何一层只能按 `Review → Write → Rebuild → Query` 的适用顺序推进；上一层批准不推定下一层。若 preflight 实际数量或 identity 与本文预计不一致，必须停止并更新计划后重新 Review。
