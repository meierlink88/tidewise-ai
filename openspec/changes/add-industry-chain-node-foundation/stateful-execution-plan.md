# 产业链 Stateful 执行计划

## 1. 当前边界

- 本文记录分层有状态执行顺序与已验收证据；截至当前 checkpoint，Layer 1–6 已按独立授权完成 PostgreSQL Write/Query，Neo4j仍未授权、未访问。
- 可执行 seed：`backend/data/entity_foundation/industry_chains_v1.json`，包含2条industry chain、21个新增chain node、5个既有node profile改进、27个membership、24条canonical topology、首批4条已批准physical constraint及6条已批准且已写PG验收的`mapped_to_sector`；economy、commodity、benchmark关系仍为空。
- Review-only fixture：`backend/data/entity_foundation/review/industry_chain_candidates_v1.json`，剩余11条physical constraint candidate和6条`mapped_to_sector` candidate。它不在`DefaultSeedPaths`，不得传给repository。
- 各层预计数量与执行结果均按 2026-07-13 local PostgreSQL 的实时preflight和写后显式READ ONLY查询校准；每个scope只运行一次，未做幂等重跑，也未访问或重建Neo4j。

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
| 4. Canonical topology（已验收） | 2026-07-13 已单独授权；仅执行一次`-apply-scope industry-chain-topology`并完成只读Query | 实际created 24、updated 0、unchanged 0；未做幂等重跑 | `industry_chain_topology_edges` | 实际24/24 active，AI10/半导体14；ID/tuple唯一，无self、substitutes、反向depends重复或无效端点；membership仍27/27 active | 将本批 24 条 topology 置 inactive；不删除 membership；重新查询 active topology=0。该停用DML未授权、未执行 |
| 5. Physical constraint（首批已验收） | 2026-07-13已单独授权；仅执行一次`-apply-scope industry-chain-physical-constraint`并完成只读Query | 实际created 4、updated 0、unchanged 0；review-only剩余11，未做幂等重跑 | 仅`industry_chain_physical_constraints`；不进入`entity_nodes`或Neo4j | 实际4/4 active approved、`generated_by_ai=true`、P2/P6正确、主体同链active；其他表计数不变 | 将获批4行置inactive；不改topology、不触发Neo4j rebuild。该停用DML未授权、未执行 |
| 6. `mapped_to_sector`（首批已验收） | 2026-07-13已单独授权；仅执行一次`-apply-scope industry-chain-sector-mapping`并完成只读Query；其余6条仍review-only | 实际created6、updated0、unchanged0；未做幂等重跑 | 仅`entity_edges`的6条`mapped_to_sector`；未写其他关系，未创建海外market `covers_sector`中国sector | 实际6/6 active、ID/tuple唯一、provenance与方向正确；`entity_edges`383→389，其他关系类型及相关表计数不变 | 将本批6条edge通过reviewed forward seed置inactive；不改sector主数据；重新投影前查询active edge=0。该停用DML未授权、未执行 |
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
- Layer 2审计时`industry_chains_v1.json`包含23 entities、5 explicit profiles、27 memberships和24 topology，physical constraints当时为0；现已在独立批准后加入首批4条constraint，review fixture始终不在`DefaultSeedPaths`。
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
5. 首批4条physical constraint已完成PG Write与只读Query验收，不得未经独立授权幂等重跑；其余11条继续Review。
6. 首批6条`mapped_to_sector`已完成PG Write与只读Query验收，不得未经独立授权幂等重跑；其余6条仍review-only。
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

## 10. 2026-07-13 Layer 4 Canonical Topology 只读 Preflight 与 Scope 修复

- Git/worktree clean，刷新origin后local/remote HEAD均为`aebd50a207c6b1dfcb19059086c4d6e4680fb4fa`；branch与Desktop-managed worktree正确。
- 显式READ ONLY确认version=14、2 chains active+approved、26 pilot nodes active、27 memberships active（AI12/半导体15）、topology=0、physical constraints=0。首次节点查询因只读SQL将`ion_implantation_equipment`误拼为`ion_implation_equipment`返回25；修正同一查询后为26，数据库无漂移。
- 正式manifest的24条topology逐项审计：ID唯一24、`(chain,from,relation,to)`唯一24，AI10、半导体14；self edge=0、非法relation type=0、`substitutes_for`=0、非同链active membership端点=0、反向`depends_on` canonical重复=0。数据库topology表为空，planned ID/tuple冲突为0。
- 调用链审计确认原checkpoint没有topology scope，默认scope会夹带membership；且repository共享batch validator要求membership与topology同批，直接裁剪topology会被拒绝。
- 已按TDD增加`-apply-scope industry-chain-topology`：只调用`UpsertIndustryChainBatch`，Memberships=0、TopologyEdges=24、PhysicalConstraints=0，且entity/profile/relationships/sector mappings调用均为0。
- Repository在topology-only路径中仍先验证edge与canonical规则；PostgreSQL事务按稳定的`(chain,node)`顺序去重查询端点，并用`SELECT ... FOR SHARE`锁定membership行直至commit。该共享锁允许并发topology校验，但与membership `UPDATE`/`DELETE`及其upsert的`DO UPDATE`冲突，消除校验与topology upsert之间的停用TOCTOU；稳定锁顺序降低多edge并发死锁风险。缺失或inactive端点在首条topology upsert前rollback。Memory repository对已持久化membership map执行等价校验。
- Scope RED证据为缺少`ApplyScopeIndustryChainTopology`导致目标包编译失败；GREEN覆盖scope精确batch、Memory persisted membership成功/缺失拒绝、默认/master/membership不回归和unknown/conflict拒绝。后续并发Review的RED证明原普通`SELECT status`不持有行锁；修正后的测试断言最终SQL含`FOR SHARE`、反向输入仍按稳定端点顺序锁定，以及缺失/inactive端点原子rollback。
- 未来获批单次写入预计created=24、updated=0、unchanged=0，FinalTableImpact仅`industry_chain_topology_edges created=24`。写后只读验收须确认总数/active=24、AI10/半导体14、ID/tuple唯一、端点仍为同链active membership、membership仍27、constraint仍0，其他数据族计数不变。
- 停用方案：通过reviewed forward seed将本批24条topology置inactive并保留审计，不DELETE、不改membership；该DML本轮未授权。
- **该checkpoint状态：** 当时仅完成Layer 4写入边界与preflight，尚未运行entity-seed；实际Write与Query结果见下一节。

## 11. 2026-07-13 Layer 4 Canonical Topology Write 与 Query 验收

- 写前Git门禁：Desktop-managed worktree与branch正确，工作区clean，local/remote HEAD均为获批checkpoint `d06f5d31051d4e1436494176483da7dd1878488e`。
- 写前显式READ ONLY：version=14，2 chains active+approved、26 pilot nodes active、27 memberships active（AI12/半导体15）；topology/constraint均为0，membership无无效端点。无关表基线为`entity_nodes=634`、`industry_chain_profiles=2`、`chain_node_profiles=54`、`entity_edges=383`、`sector_source_mappings=89`。
- Pre-Layer4备份：`/private/tmp/tidewise_local_pre_layer4_20260713T024610Z.dump`，custom format，986,070 bytes，SHA-256 `abe475308950b05b6a275ca4388b0bd0d28b39c3d8c751350c49fef2f198fe95`；`pg_restore --list`成功读取253个TOC entries。
- 唯一写命令为`go run ./cmd/entity-seed -apply-scope industry-chain-topology`，只执行一次；未运行默认scope、Layer2/3 scope、其他seed或幂等重跑。
- Report：TotalEntities=0，scope=`industry-chain-topology`，IndustryChainCounts=`membership 0 / topology 24 / physical_constraint 0`，operation与FinalTableImpact均为created24、updated0、unchanged0，Failed=0；其他数据族统计为空或0。
- 写后显式READ ONLY：topology总数/active=`24/24`，ID与`(chain,from,relation,to)`唯一数均为24；AI链10、半导体链14；self edge、无效/跨链/inactive membership端点、`substitutes_for`和反向`depends_on` canonical重复均为0。
- Membership仍为27/27 active，physical constraints仍为0；无关表仍为`634/2/54/383/89`，与写前完全一致。
- **Layer 4状态：Write与Query验收完成。** Physical constraint、`mapped_to_sector`、economy/commodity/benchmark、Neo4j rebuild及后续阶段均未授权、未执行。

## 12. 2026-07-13 Layer 5 Physical Constraint 首批无状态晋级准备

- 实时Git门禁：Desktop-managed worktree与branch正确，工作区clean，local/remote HEAD均为`c35a3a7e65c3f20d0ecdc6c6068f7953d55e0ef4`。显式READ ONLY确认2 chains active+approved、26 pilot nodes、27 active memberships、24 active topology、physical constraints=0。
- 用户逐项批准4条进入正式seed准备：data center `power_capacity`、AI advanced packaging `packaging_density`、data center `infrastructure_access`、semiconductor advanced packaging `reliability`。全部保持`generated_by_ai=true`、`review_status=approved`、`approved_by_human=true`；后者只转换为执行上下文approval gate，不作为事实字段持久化。
- `infrastructure_access`已校正为IEA Executive Summary P2；`reliability`已校正为TSMC/ECTC具体可靠性论文P6，并将两条`verified_at`更新为2026-07-13复核时间。其余11条仍只存在review-only fixture：需补证9、删除或改写2。
- 已按TDD增加`industry-chain-physical-constraint` scope：batch仅含PhysicalConstraints=4，Profiles/Memberships/TopologyEdges=0，entity/profile/relationship/sector mapping调用均为0；report预计created4且FinalTableImpact仅`industry_chain_physical_constraints`。
- Repository对constraint-only batch先校验显式approval gate，再在PostgreSQL事务内按稳定subject key顺序以`FOR SHARE`锁定已持久化同链active membership或topology edge；锁与subject更新/停用冲突。缺失/inactive subject或identity conflict在任何constraint写入失败时原子rollback；Memory repository执行等价持久化subject校验。
- **该checkpoint状态：** 当时仅完成无状态晋级准备，尚未运行`entity-seed`；实际Write与Query结果见下一节。Physical constraints不投影Neo4j。

## 13. 2026-07-13 Layer 5 首批 Physical Constraint Write 与 Query 验收

- 写前Git门禁：Desktop-managed worktree与branch正确，工作区clean，local/remote HEAD均为获批checkpoint`149408396a30aa4a3b3705f8cfe839cbe3470156`。
- 写前显式READ ONLY：version=14，2 chains active+approved、26 pilot nodes、27 active memberships、24 active topology、constraints=0；4条planned subject均为同链active membership。无关表基线为`entity_nodes=634`、`industry_chain_profiles=2`、`chain_node_profiles=54`、`entity_edges=383`、`sector_source_mappings=89`。
- Pre-constraint备份：`/private/tmp/tidewise_local_pre_constraints_20260713T032130Z.dump`，custom format，989,099 bytes，SHA-256 `9296a1df957449d28376790444509634e6f9e3b99303a314ca9b8eed0728f288`；`pg_restore --list`成功读取253个TOC entries。
- 唯一写命令为`go run ./cmd/entity-seed -apply-scope industry-chain-physical-constraint`，只执行一次；未运行默认scope、其他scope、其他seed或幂等重跑。
- Report：TotalEntities=0，scope=`industry-chain-physical-constraint`，IndustryChainCounts=`membership 0 / topology 0 / physical_constraint 4`，operation与FinalTableImpact均为created4、updated0、unchanged0，Failed=0；FinalTableImpact仅`industry_chain_physical_constraints`。
- 写后显式READ ONLY：constraints total/active=`4/4`、unique IDs=4、unique subject+type=4、approved=4、`generated_by_ai=true`=4；P2/P6及另外两条source_name/source_url/verified_at与正式seed完全一致；无效或inactive subject=0。
- Membership仍27/27 active，topology仍24/24 active；无关表仍为`634/2/54/383/89`。本轮未访问或重建Neo4j，physical constraints继续保持PostgreSQL-only。
- **Layer 5首批状态：Write与Query验收完成。** 其余11条constraint、首批6条`mapped_to_sector` PG Write、其他6条mapping candidate、其他跨实体关系、Neo4j rebuild及后续阶段均未授权、未执行。

## 14. 2026-07-13 Layer 6 `mapped_to_sector` 无状态晋级准备

- 实时Git clean且local/remote HEAD均为`edcedd54ad7d1708ecf3fc378c7d310fdeffe347`；显式READ ONLY确认version14、2 chains、26 unique nodes、27 active memberships、24 active topology、4 active approved constraints、`entity_edges=383`、现有`mapped_to_sector=0`。
- 6条获批mapping的from/to端点均active且类型为`industry_chain|chain_node → sector`，tuple冲突0。其余6条继续留在review-only fixture（2条需补证、4条删除或改写）。
- 正式relationship以固定commit的Layer 6 Review为`source_url`，`source_name/evidence_note`明确登记Tidewise composite curation；chain/node与sector来源是组合证据，S5仅支持三条设备映射的目标板块定义，不冒充外部直接mapping声明。
- 已按TDD增加显式`industry-chain-sector-mapping` scope：只处理6条`mapped_to_sector`，所有无关repository调用为0；Postgres单事务按stable key排序`FOR SHARE`锁定active持久化端点，校验policy并复用relationship upsert，不可变identity冲突或任一失败整批rollback。Memory repository提供等价原子行为。
- 未来获批首次执行预计report为created6/updated0/unchanged0，FinalTableImpact仅`entity_edges`。本checkpoint未运行该scope、无DML、未访问Neo4j。未来必须先独立授权PG Write/Query；通过后再单独授权Neo4j Rebuild/Query。

## 15. 2026-07-13 Layer 6 `mapped_to_sector` Write 与 Query 验收

- 写前Git门禁：Desktop-managed worktree与branch正确，工作区clean，local/remote HEAD均为获批checkpoint`0391999806aab54c7136bd8a479b037c29fb1bcb`。
- 写前显式READ ONLY：version=14，2 chains active+approved、26 pilot nodes active、27 active memberships、24 active topology、4 active approved constraints；`entity_edges=383`、`mapped_to_sector=0`。正式manifest精确6条、review fixture剩6条；12个端点均active且方向为`industry_chain|chain_node → sector`，planned ID/tuple冲突均为0。
- Pre-mapping备份：`/private/tmp/tidewise_local_pre_mapping_20260713T050005Z.dump`，custom format，990,107 bytes，SHA-256 `26e62a90ff6fbafe6e3de44f5aba43947446e41daee3c62e2314c821dbea7bcb`；`pg_restore --list`成功读取253行清单。
- 唯一写命令为`go run ./cmd/entity-seed -apply-scope industry-chain-sector-mapping`，只执行一次；未运行默认scope、其他scope、其他seed或幂等重跑。
- Report：TotalEntities=0，scope=`industry-chain-sector-mapping`，EdgeCounts仅`mapped_to_sector=6`；operation counts=`created6 / updated0 / unchanged0 / failed0 / skipped0`，FinalTableImpact仅`entity_edges created6 / updated0 / unchanged0`。
- 写后显式READ ONLY：`entity_edges=389`，`mapped_to_sector` total/active=`6/6`，unique IDs=6、unique `(from,to,relation_type)`=6；六条stable key、source/evidence/verified_at与正式seed逐项一致，均明确为Tidewise composite curation；12个端点均active，from仅为`industry_chain|chain_node`、to仅为`sector`。
- 其他relation type计数与写前一致：`covers_sector=52`、`has_market=40`、`measures=10`、`member_of=223`、`observes_benchmark=10`、`references=5`、`tracks_index=43`。Membership/topology/constraint仍为`27/24/4`；`sector_source_mappings=89`、`entity_nodes=634`、`industry_chain_profiles=2`、`chain_node_profiles=54`，均与写前一致。
- **Layer 6首批状态：Write与Query验收完成。** 本轮未访问或重建Neo4j；其余6条mapping、其他跨实体关系、Neo4j Rebuild/Query、Sync、Archive与PR均未授权、未执行。
