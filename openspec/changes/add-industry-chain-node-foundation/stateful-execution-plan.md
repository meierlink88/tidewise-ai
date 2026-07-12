# 产业链 Stateful 执行计划

## 1. 当前边界

- 本文只定义后续有状态执行顺序，本 checkpoint 不执行任何命令写入 PostgreSQL 或 Neo4j。
- 可执行 seed：`backend/data/entity_foundation/industry_chains_v1.json`，包含 2 条 industry chain、21 个新增 chain node、5 个既有 node profile 改进、27 个 membership、24 条 canonical topology；不含 physical constraint、`mapped_to_sector`、economy、commodity 或 benchmark 关系。
- Review-only fixture：`backend/data/entity_foundation/review/industry_chain_candidates_v1.json`，包含 15 条 physical constraint candidate 和 12 条 `mapped_to_sector` candidate。它不在 `DefaultSeedPaths`，不得传给 repository。
- 以下预计数量已按 2026-07-13 local PostgreSQL 只读 preflight 校准；本轮未运行 migration apply、seed、DML 或 Neo4j rebuild。

## 2. 2026-07-13 只读 Preflight Evidence

- local 配置指向 `localhost:5432/tidewise_local`，数据库容器处于 healthy；审计查询均在显式 `BEGIN TRANSACTION READ ONLY` 中执行。
- `goose_db_version` 当前已应用版本为 `13`，`000014_add_industry_chain_foundation.sql` 未应用。
- `industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints` 均不存在；`chain_node_profiles` 也不存在 `node_category`、`definition`、`unit_of_analysis`、`granularity_note`。未发现 000014 部分应用或同名结构漂移。
- PostgreSQL 当前有 33 个 `chain_node`，且 33 个均为 active。5 个复用节点均存在且 profile 可关联：

| stable key | UUID | 当前 `chain_position` | 当前名称 | 当前 aliases |
|---|---|---|---|---|
| `chain_node:power_grid` | `3bd72a86-f31f-5f88-b0fc-22d4a6a4f784` | `infrastructure` | 电网 | 空 |
| `chain_node:data_center` | `255fc68e-9254-5016-b3a2-3c728a047ba9` | `infrastructure` | 数据中心 | 空 |
| `chain_node:gpu` | `8a21c5bf-454a-5ebc-9a60-e8b85b2a3696` | `core_component` | GPU | 空 |
| `chain_node:eda` | `4e9c51af-2487-5b98-b8bb-44b3fa086c9b` | `core_software` | EDA软件 | 空 |
| `chain_node:lithography_machine` | `dea1dd40-973d-5431-8612-a01a45395a0b` | `core_equipment` | 光刻机 | 空 |

- 21 个 planned 新 node stable key 均未命中 `entity_nodes`；2 个 planned `industry_chain` stable key 均未命中，数据库中也不存在其他 `industry_chain` 实体。因此 master 层不存在 stable key 抢占冲突。
- membership/topology 新表尚不存在，planned 27 个 membership ID 与 24 个 topology ID 的数据库冲突检查为 `not applicable`；Layer 1 后、Layer 3/4 写入前必须再次只读核对 ID 与不可变端点。
- 现有 5 个复用实体的 aliases 为空，而版本化 seed 已为其登记英文 alias；因此 Layer 2 的最终表级影响从“仅 profile 更新”校准为 `entity_nodes` 另有 5 行 updated。其 UUID、stable key、状态和中文 canonical name 不变。

## 3. 2026-07-13 Layer 1 执行与 Query 验收

- 执行前备份：`/private/tmp/tidewise_local_pre_000014_20260712T165928Z.dump`，custom format，大小 959,309 bytes，SHA-256 `8c4a25c7c001aae6cbb35e6f4e7915454b4db1df3635ebeb9cc6931218709371`；`pg_restore --list` 成功读取 212 个 TOC entries。该路径属于本机临时备份，进入后续生产化执行前仍需按环境备份策略另行保存。
- 仅执行项目标准 `APP_ENV=local ... go run ./cmd/dbmigrate -apply`；runner 报告只应用 `000014_add_industry_chain_foundation.sql`，数据库从 version 13 升至 14。未运行 `entity-seed` 或其他业务 DML。
- migration 后的只读 Query 验收全部在显式 `BEGIN TRANSACTION READ ONLY` 中完成：version=14 且 000014 已 applied；4 张新表全部存在；`chain_node_profiles` 的 4 个增量列和 `generated_by_ai BOOLEAN NOT NULL DEFAULT false` 存在。
- 4 张新表共核对到 32 个 PK/UNIQUE/FK/CHECK constraints 和 12 个索引；包含 chain/profile FK、membership 唯一键、topology 自环与三类关系 CHECK、physical constraint 恰一主体与13类枚举 CHECK，以及 node/edge partial indexes。
- `industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints` 行数均为 0；既有 `chain_node` 仍为 33/33 active，5 个复用 stable key 的 UUID/status 与 preflight 完全一致。
- 新表为空，因此 planned 27 个 membership ID 和 24 个 topology ID 的冲突命中均为 0。Layer 3/4 实际写入前仍保留二次只读 identity/endpoint 检查。
- **Layer 1 状态：Write 与 Query 验收完成。** Layer 2 及以后仍未授权、未执行。

## 4. 分层执行矩阵

| 层 | 前置条件与独立授权 | 预计 created / updated / unchanged | 影响范围 | 验证查询 | 回滚或停用方案 |
|---|---|---|---|---|---|
| 1. Migration（已验收） | 2026-07-13 已单独授权并完成备份、Write、只读 Query 验收 | 实际：migration applied 1；新表 created 4；`chain_node_profiles` 列 added 4；业务行 created/updated 0 | `industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints`、`chain_node_profiles` | 实际 version=14；4 表、字段、32 个 constraints、12 个 indexes、`generated_by_ai` 均通过；既有 33 个 chain node 与5个复用UUID/status不变 | 本项目 Down 为 reviewed no-op；如需撤销须停止后续写入并通过新的 reviewed forward migration 处理，不手工 DROP |
| 2. Chain / node master | 层 1 验收；用户批准 2 条 chain、21 个新增 node、5 个复用实体 alias 与 profile 改进 | 最终表级差异：`entity_nodes` created 23、updated 5、既有其余 28 unchanged；`industry_chain_profiles` created 2；`chain_node_profiles` created 21、updated 5；幂等重跑本批 51 个目标行应 unchanged | `entity_nodes`、`industry_chain_profiles`、`chain_node_profiles` | 按 stable key 查询 2 chain + 26 pilot node；核对 5 个复用 UUID 不变、英文 aliases、definition、category、unit、granularity、review status；运行 seed report | 在不删除既有 5 个 node 的前提下，将本批新增 23 个实体置 inactive；5 个 aliases/profile 改进需用 reviewed forward seed 恢复旧值；不物理删除 |
| 3. Membership | 层 2 验收；用户批准两链 12/15 membership | created 27；重复执行 unchanged 27 | `industry_chain_memberships` | 按 chain 查询 active membership，断言 AI=12、semiconductor=15、共享 `advanced_packaging` 两条 membership、stage order 唯一稳定 | 将本批 27 条 membership 置 inactive；保留审计行；后续 rebuild 前重新查询 active 数量 |
| 4. Canonical topology | 层 3 验收；用户批准 10 条 AI + 14 条 semiconductor topology 及 evidence note 中的缺口 | created 24；重复执行 unchanged 24 | `industry_chain_topology_edges` | 按 chain/relation type 查询；断言无 self edge、端点均为同链 active membership、无 `substitutes_for`、无反向 `depends_on` 重复 | 将本批 24 条 topology 置 inactive；不删除 membership；重新查询 active topology=0 |
| 5. Physical constraint | 15 条 candidate 逐项补齐权威技术证据并取得人工 Review；每条写入前单独确认 approval gate；当前未授权 | 当前可执行 0；Review-only candidate 15；未来数量以逐项批准结果为准 | 仅 `industry_chain_physical_constraints`；不进入 `entity_nodes` 或 Neo4j | 按 chain/node/topology edge 查询；断言 `review_status=approved`、`generated_by_ai=true`、主体同链 active、constraint type 属于 13 类 | 将获批写入行置 inactive；不改 topology；不触发 Neo4j rebuild。未批准 candidate 始终留在 Review fixture |
| 6. `mapped_to_sector` | 12 条 candidate 逐项确认分析映射、来源和端点；当前未授权。economy/commodity/benchmark 清单保持空 | 当前可执行 0；Review-only candidate 12；未来数量以逐项批准结果为准 | 经批准后仅 `entity_edges` 的 `mapped_to_sector`；不得创建海外 market `covers_sector` 中国 sector | 查询 relation type、from/to stable key、source；断言无 economy/commodity/benchmark 新关系、无海外 market `COVERS_SECTOR` | 将本批获批 entity edge 置 inactive；不改 sector 主数据；重新投影前查询 active edge=0 |
| 7. Neo4j rebuild | 层 2–4 PostgreSQL 验收完成；若层 6 有获批关系则先完成其 PG 验收；用户单独批准 rebuild。Physical constraint 不构成前置写入 | 预计新增 Entity 23（2 chain + 21 new node）；新增关系 51（27 membership + 24 topology）；既有 5 node 为 upsert unchanged；若层 6 未批准则 sector mapping 0 | Neo4j `projection_namespace=tidewise` 下统一 `Entity` 与 membership/topology/已审阅 entity edges；排除 physical constraints 和 observations | 查询 2 chain、26 pilot node、27 membership、24 topology；验证 benchmark/sector 路径仅在已批准 edge 存在时成立；断言无 constraint 节点/边、无海外 market 覆盖中国 sector | 从 PostgreSQL active facts 重新 rebuild；若需撤销，先按对应 PG 层置 inactive并验收，再单独授权 rebuild；不得直接手工删局部图 |
| 8. Query 验收 | 每次 Write 或 Rebuild 后另行授权只读验收；不得用上一层授权推定 | 只读，created/updated=0 | PostgreSQL report/query 与 Neo4j read query | 对照每层预计数量、stable key、状态、来源、ID 和路径；差异立即停止下一层 | 无写状态；记录差异并回到对应层 Review，不自动修复 |

## 5. 必须逐层取得的用户授权

1. 执行 migration。（2026-07-13 已授权、执行并通过只读 Query 验收）
2. 写入 chain / node master。
3. 写入 membership。
4. 写入 canonical topology。
5. 逐项批准并写入 physical constraint；当前 15 条全部未授权。
6. 逐项批准并写入 `mapped_to_sector`；当前 12 条全部未授权。
7. Neo4j rebuild。
8. 每层 Query 验收。

任何一层只能按 `Review → Write → Rebuild → Query` 的适用顺序推进；上一层批准不推定下一层。若 preflight 实际数量或 identity 与本文预计不一致，必须停止并更新计划后重新 Review。
