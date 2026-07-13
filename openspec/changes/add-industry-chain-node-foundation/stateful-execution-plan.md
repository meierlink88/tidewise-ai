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
| 2. Chain / node master（已验收） | 2026-07-13 已单独授权；仅执行 `-apply-scope industry-chain-master` 一次并完成只读 Query 验收 | 实际最终表级：`entity_nodes` created 23、updated 5；`industry_chain_profiles` created 2；`chain_node_profiles` created 21、updated 5。operation report 为 created 46、updated 15、unchanged 0 | `entity_nodes`、`industry_chain_profiles`、`chain_node_profiles` | 实际2 chain active+approved；26 pilot node active且 aliases/profile完整；5个复用UUID/status/中文名不变；membership/topology/constraint与试点跨实体关系均为0 | 在不删除既有 5 个 node 的前提下，将本批新增 23 个实体置 inactive；5 个 aliases/profile 改进需用 reviewed forward seed 恢复旧值；不物理删除 |
| 3. Membership（已验收） | 2026-07-13 已单独授权；仅执行一次 `-apply-scope industry-chain-membership` 并完成只读Query | 实际created 27、updated 0、unchanged 0；未做幂等重跑 | `industry_chain_memberships` | 实际27/27 active且ID/tuple唯一；AI=12、semiconductor=15；`advanced_packaging`两链各一条；端点全active；topology/constraint为0 | 将本批27条membership置inactive；保留审计行；后续rebuild前重新查询active数量。该停用DML未授权、未执行 |
| 4. Canonical topology | 层 3 验收；用户批准 10 条 AI + 14 条 semiconductor topology 及 evidence note 中的缺口 | created 24；重复执行 unchanged 24 | `industry_chain_topology_edges` | 按 chain/relation type 查询；断言无 self edge、端点均为同链 active membership、无 `substitutes_for`、无反向 `depends_on` 重复 | 将本批 24 条 topology 置 inactive；不删除 membership；重新查询 active topology=0 |
| 5. Physical constraint | 15 条 candidate 逐项补齐权威技术证据并取得人工 Review；每条写入前单独确认 approval gate；当前未授权 | 当前可执行 0；Review-only candidate 15；未来数量以逐项批准结果为准 | 仅 `industry_chain_physical_constraints`；不进入 `entity_nodes` 或 Neo4j | 按 chain/node/topology edge 查询；断言 `review_status=approved`、`generated_by_ai=true`、主体同链 active、constraint type 属于 13 类 | 将获批写入行置 inactive；不改 topology；不触发 Neo4j rebuild。未批准 candidate 始终留在 Review fixture |
| 6. `mapped_to_sector` | 12 条 candidate 逐项确认分析映射、来源和端点；当前未授权。economy/commodity/benchmark 清单保持空 | 当前可执行 0；Review-only candidate 12；未来数量以逐项批准结果为准 | 经批准后仅 `entity_edges` 的 `mapped_to_sector`；不得创建海外 market `covers_sector` 中国 sector | 查询 relation type、from/to stable key、source；断言无 economy/commodity/benchmark 新关系、无海外 market `COVERS_SECTOR` | 将本批获批 entity edge 置 inactive；不改 sector 主数据；重新投影前查询 active edge=0 |
| 7. Neo4j rebuild | 层 2–4 PostgreSQL 验收完成；若层 6 有获批关系则先完成其 PG 验收；用户单独批准 rebuild。Physical constraint 不构成前置写入 | 预计新增 Entity 23（2 chain + 21 new node）；新增关系 51（27 membership + 24 topology）；既有 5 node 为 upsert unchanged；若层 6 未批准则 sector mapping 0 | Neo4j `projection_namespace=tidewise` 下统一 `Entity` 与 membership/topology/已审阅 entity edges；排除 physical constraints 和 observations | 查询 2 chain、26 pilot node、27 membership、24 topology；验证 benchmark/sector 路径仅在已批准 edge 存在时成立；断言无 constraint 节点/边、无海外 market 覆盖中国 sector | 从 PostgreSQL active facts 重新 rebuild；若需撤销，先按对应 PG 层置 inactive并验收，再单独授权 rebuild；不得直接手工删局部图 |
| 8. Query 验收 | 每次 Write 或 Rebuild 后另行授权只读验收；不得用上一层授权推定 | 只读，created/updated=0 | PostgreSQL report/query 与 Neo4j read query | 对照每层预计数量、stable key、状态、来源、ID 和路径；差异立即停止下一层 | 无写状态；记录差异并回到对应层 Review，不自动修复 |

## 5. 2026-07-13 Layer 2 实时只读 Preflight

### 5.1 Git 与数据库现状

- 刷新 origin 后，Desktop-managed worktree 仍为 `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai`，branch 为 `codex/add-industry-chain-node-foundation`；local HEAD 与 remote HEAD 均为 `f85d18734633a10238b971fdde5b28bcec31879c`，工作区 clean，无 handoff 后漂移。
- 所有数据库审计均在显式 `BEGIN TRANSACTION READ ONLY` 中执行。实时 migration version=14，4 张产业链表存在且行数仍分别为 `0/0/0/0`。
- `chain_node` 仍为 33/33 active。5 个复用 stable key 的 UUID、status、中文名称与 Layer 1 前一致；aliases 仍为空，新增 profile 字段 `node_category/definition/unit_of_analysis/granularity_note` 仍为空，因此这 5 个实体与 profile 确实需要 Layer 2 update。
- 2 个 planned `industry_chain` 和 21 个 planned 新 `chain_node` stable key 均未命中 `entity_nodes`；数据库仍没有任何 `industry_chain` 实体。

### 5.2 实时预计影响

最终表级差异仍为：

| 表 | created | updated | unchanged |
|---|---:|---:|---:|
| `entity_nodes` | 23 | 5 | 非试点既有 28 个 chain node 保持不变 |
| `industry_chain_profiles` | 2 | 0 | 0 |
| `chain_node_profiles` | 21 | 5 | 非试点既有 profile 保持不变 |

若实现严格的 master-only service scope，并沿用当前 manifest 中“实体内 profile 后再应用5个显式 profile改进”的顺序，预计 seed report 写操作为 `created=46`、`updated=15`、`unchanged=0`：23 次 entity create + 23 次 profile create；5 次复用 entity alias update + 5 次既有 inline profile update + 5 次显式 profile update。该 report 是写操作次数，不等于最终表级差异；实现后必须用测试固定两种统计口径。

### 5.3 当前执行路径阻断

- `backend/cmd/entity-seed/main.go` 只提供 `seed-dir`、inactive 和 sector convergence 参数，总是加载 `DefaultSeedPaths` 后调用 `Service.Apply`，没有 industry-chain 数据族选择边界。
- `industry_chains_v1.json` 当前同时包含 23 entities、5 explicit profiles、27 memberships 和24 topology；physical constraints 与 relationships 为0，review fixture 不在 `DefaultSeedPaths`。
- `Service.Apply` 写完所有 entity/profile 后，只要 manifest 含 membership/topology 就立即调用 `UpsertIndustryChainBatch`。因此运行现有标准 `entity-seed` 会在 Layer 2 master 后继续写 Layer 3 membership 与 Layer 4 topology，违反逐层授权。
- 2026-07-13 已按 TDD 增加合规的 `-apply-scope industry-chain-master` 边界：复用 `DefaultSeedPaths`、完整 manifest validation、现有 repository/report，只筛选本批2个 chain与26个 membership endpoint node 的 entity/profile，并跳过 `UpsertIndustryChainBatch`、relationships、sector mappings 和其他实体数据族。默认无 scope 行为保持兼容；未知 scope 或与 sector convergence 冲突的组合会被拒绝。
- Scope 修复的自动化测试已验证精确选择28个实体、33次 profile operations、不调用 industry batch、不包含 review fixture，并区分 operation counts 与 final table impact。本轮只实现和验证代码，未运行 `entity-seed`，未产生任何 DML。
- **当前结论：原执行路径阻断已由无状态代码关闭，但仍须先完成本 checkpoint 的代码 Review，并再次实时只读 preflight，才能申请 Layer 2 写入授权。**

已完成的最小实现调整复用现有 manifest、validator、repository 与 report，没有创建平行 seed 机制：

1. entity-seed application/service 已增加显式 `industry-chain-master` scope；从已验证 manifest 的 industry-chain membership endpoints 推导本批2个 chain与26个node key，只处理这些 entity/profile，并明确跳过 membership、topology、physical constraints、relationships 和其他实体数据族。
2. 命令边界为 `go run ./cmd/entity-seed -apply-scope industry-chain-master`；它是未来获批后的唯一 Layer 2 写入口，本 checkpoint 严禁执行。无 scope 保持既有全量行为兼容；fake repository 测试断言从未调用 `UpsertIndustryChainBatch`。
3. 在未来进入 Layer 3/4 前，再以同一 `ApplyScope` 最小枚举扩展 membership-only/topology-only；本轮未实现，master-only 不构成后续层授权。

### 5.4 Layer 2 获批写入后的只读验收 SQL 边界

- 查询2个 chain stable key，断言 `entity_type=industry_chain`、active、profile approved；查询26个试点 node，断言21 created、5 UUID/stable key/status/中文名不变且英文 aliases/profile更新。
- 分表对账 `entity_nodes`、`industry_chain_profiles`、`chain_node_profiles` 的最终 created/updated；同时核对 seed report 的 operation counts，不混淆两种口径。
- 断言 `industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints` 仍为0，并按本批 key 核对没有新增 `mapped_to_sector`、economy、commodity 或 benchmark edge。
- 幂等重跑预期本批目标 unchanged，但第二次执行仍是独立 DML，必须再次取得明确授权后才能验证。

停用方案保持不变：不删除既有5个节点；通过 reviewed forward seed 将本批新增23个实体置 inactive，并用 reviewed forward seed 恢复5个复用实体原 aliases/profile。不得手工 DELETE 或依赖 migration Down。

## 6. 2026-07-13 Layer 2 Write 与 Query 验收

- 写前 Git 门禁：刷新 origin 后，Desktop-managed worktree、branch、local/remote HEAD 均匹配获批 checkpoint `5fc8a90c98c809259c2ddba6152dd721502cc83f`，工作区 clean。
- 写前数据库门禁：显式 READ ONLY 确认 version=14、4 张产业链表行数 `0/0/0/0`、chain node 33/33 active、5个复用UUID/status/中文名不变、23个 planned master stable key冲突为0。
- Pre-Layer2 备份：`/private/tmp/tidewise_local_pre_layer2_20260713T015540Z.dump`，custom format，979,104 bytes，SHA-256 `2a547922d53a10b9d1bbf610e3aa3d14fd3b274f81b72270695e28ddf4467268`；`pg_restore --list` 成功读取242个 TOC entries。
- 唯一写命令为 `go run ./cmd/entity-seed -apply-scope industry-chain-master`，只执行一次；未运行无 scope entity-seed、其他 seed 或幂等重跑。
- Report：scope=`industry-chain-master`，TotalEntities=28（2 chain +26 node），operation counts=`created 46 / updated 15 / unchanged 0 / failed 0`；FinalTableImpact=`entity_nodes 23/5/0`、`industry_chain_profiles 2/0/0`、`chain_node_profiles 21/5/0`；IndustryChainCounts 与 EdgeCounts 均为空。
- 写后显式 READ ONLY 验收：`entity_nodes=634`，结合 report 的23 created 对应写前基线611；chain node=54/54 active；2 chain 均 active+approved；26/26 pilot node 存在、含英文 aliases 且4个 profile 增量字段完整；5个复用UUID/stable key/status/中文名保持不变。
- `industry_chain_profiles=2`，`industry_chain_memberships=0`、`industry_chain_topology_edges=0`、`industry_chain_physical_constraints=0`；试点新 chain/node 上 `mapped_to_sector/scoped_to_economy/uses_commodity/produces_commodity/observed_by_benchmark` edge 为0；planned 27 membership与24 topology冲突命中均为0。
- **Layer 2 状态：Write 与 Query 验收完成。** Layer 3 membership、Layer 4 topology、candidate 与 Neo4j 均未授权、未执行。

## 7. 必须逐层取得的用户授权

1. 执行 migration。（2026-07-13 已授权、执行并通过只读 Query 验收）
2. 写入 chain / node master。（2026-07-13 已授权、执行并通过只读 Query 验收）
3. 写入 membership。
4. 写入 canonical topology。
5. 逐项批准并写入 physical constraint；当前 15 条全部未授权。
6. 逐项批准并写入 `mapped_to_sector`；当前 12 条全部未授权。
7. Neo4j rebuild。
8. 每层 Query 验收。

任何一层只能按 `Review → Write → Rebuild → Query` 的适用顺序推进；上一层批准不推定下一层。若 preflight 实际数量或 identity 与本文预计不一致，必须停止并更新计划后重新 Review。

## 8. 2026-07-13 Layer 3 Membership 只读 Preflight 与 Scope 修复

- Git/worktree clean，刷新 origin 后 local/remote HEAD均为 `341a2f52d90ec190227af8b20e8e87a34376c5e5`；Desktop-managed worktree与branch正确。
- 显式READ ONLY确认version=14，2个chain均active+approved，26个pilot node均active；membership/topology/physical constraint行数仍为0。
- Versioned manifest含27个active memberships，ID唯一数27、`(industry_chain_key, chain_node_key)`唯一数27；AI链12、半导体链15，`advanced_packaging`有两条不同chain membership。数据库表为空，因此planned ID与unique tuple冲突均为0；全部chain/node endpoints已存在且active。
- 调用链审计确认checkpoint `341a2f5` 仅有master scope；默认scope会把27 memberships与24 topology放入同一batch，因此不得用于Layer 3。
- 已按TDD增加 `-apply-scope industry-chain-membership`：复用 `DefaultSeedPaths`、完整manifest validation、现有service/repository/report；scope后不含entity/profile/relationships/sector mappings，只调用一次 `UpsertIndustryChainBatch`，其中Memberships=27、TopologyEdges=0、PhysicalConstraints=0。
- RED证据为缺少 `ApplyScopeIndustryChainMembership` 导致目标包编译失败；GREEN覆盖精确batch、无关repository调用为0、默认行为不回归、unknown/sector-convergence冲突拒绝。预计未来获批首次执行report为created=27、updated=0、unchanged=0，FinalTableImpact仅 `industry_chain_memberships created=27`。
- **当前状态：Layer 3写入边界已无状态实现，但未运行entity-seed、未产生DML。** 必须先完成代码Review并重新实时只读preflight，再单独授权命令 `go run ./cmd/entity-seed -apply-scope industry-chain-membership`。

未来写后验收：按chain断言active membership为12/15、总数27、共享`advanced_packaging`两条；所有端点active且identity与manifest一致；topology/constraint仍为0，entity/profile/relationships不变。停用方案为通过reviewed forward seed将本批27行置inactive并保留审计，不DELETE；该方案本轮未授权。

## 9. 2026-07-13 Layer 3 Membership Write 与 Query 验收

- 写前Git门禁：刷新origin后，Desktop-managed worktree与branch正确，工作区clean，local/remote HEAD均为已验收membership scope checkpoint `7f599297af6e62938cace27685edb7c0eaee1032`。
- 写前显式READ ONLY：version=14，2 chains active+approved，26 pilot nodes active；membership/topology/constraint均为0；planned endpoints无inactive，ID/tuple冲突为0。无关表基线为`entity_nodes=634`、`industry_chain_profiles=2`、`chain_node_profiles=54`、`entity_edges=383`、`sector_source_mappings=89`。
- Pre-Layer3备份：`/private/tmp/tidewise_local_pre_layer3_20260713T021205Z.dump`，custom format，983,891 bytes，SHA-256 `e33834390fadbcbc1273a3a1818ce835d12f61295e62efd23f21e0f8dd900b80`；`pg_restore --list`成功读取242个TOC entries。
- 唯一写命令为`go run ./cmd/entity-seed -apply-scope industry-chain-membership`，只执行一次；未运行默认scope、其他seed或幂等重跑。
- Report：TotalEntities=0，scope=`industry-chain-membership`，IndustryChainCounts=`membership 27 / topology 0 / physical_constraint 0`，operation与FinalTableImpact均为created27、updated0、unchanged0，Failed=0；entity/profile/edge/mapping counts为空或0。
- 写后显式READ ONLY：membership总数=27、active=27、unique IDs=27、unique `(chain,node)`=27；AI链12、半导体链15；`advanced_packaging`恰有2条membership且属于2条不同chain；inactive/invalid endpoints=0。
- Topology=0、physical constraints=0；无关表仍为`634/2/54/383/89`，与写前完全一致。
- **Layer 3状态：Write与Query验收完成。** Layer 4 topology、candidate、跨实体关系与Neo4j均未授权、未执行。
