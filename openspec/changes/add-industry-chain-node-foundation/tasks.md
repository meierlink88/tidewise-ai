## 1. Apply 前 Review 与候选冻结

- [x] 1.1 由用户 Review 并批准 4 张新表、`chain_node_profiles` 增量字段、三类 topology、13 类 physical constraint、`mapped_to_sector`、两条试点范围和 stateful 操作门禁；2026-07-12 主对话正式确认进入 Apply 候选冻结阶段
- [x] 1.2 在 `candidate-review.md` 整理 AI 算力基础设施 12 节点与半导体制造 15 节点，对现有 33 节点给出复用/改进/新增判断，去重为 26 节点并登记来源与证据缺口；第二批链不进入本 change seed
- [x] 1.3 分别生成 membership、canonical topology、physical constraint、economy/commodity/benchmark/sector 关系 Review 清单；按 Review 修正封装方向、以 IC design 衔接 EDA 与制造、移除孤立 electric_power membership，并记录后续 `Review → Write → Rebuild → Query` 顺序

## 2. 静态 schema 与领域模型（TDD）

- [x] 2.1 先在 `backend/migrations/` 增加失败的静态测试，逐字段覆盖 `industry_chain_profiles`、`chain_node_profiles` 增量、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints` 的 FK、枚举、唯一、自环、恰一主体、索引和非破坏性约束
- [x] 2.2 运行 migration 目标测试确认 RED，再追加单一增量 migration 与安全回滚策略使测试 GREEN；不得执行 migration apply
- [x] 2.3 先在 `backend/internal/domain/` 增加 table-driven tests，覆盖 industry chain profile、membership、active/inactive topology 端点、canonical edge 去重、physical constraint 同链主体、人工批准 gate、合法/非法类型、方向、状态和禁止推理字段
- [x] 2.4 实现最小领域类型与 validator 使目标包测试通过，并保持既有 `chain_node` 身份兼容

## 3. Loader、repository 与关系 policy（TDD）

- [x] 3.1 先在 `backend/internal/apps/entityfoundation/seed/` 增加 loader/fixture 失败测试，覆盖 profile、membership、topology、physical constraint 的来源、重复、自环、inactive/非成员端点、AI candidate 未授权 approved、人工批准后保留 AI provenance 和禁止非物理/推理字段
- [x] 3.2 扩展 manifest、loader、paths 和 validator 使测试通过，不加入未经 Review 的正式试点 seed
- [x] 3.3 先增加 MemoryRepository 与 PostgresRepository 失败测试，覆盖 4 张新表和 profile 增量的原子幂等 upsert、不可变 identity、report 统计、关联校验与回滚等价性
- [x] 3.4 实现 repository 接口、共享 batch validator、identity conflict guard、SQL、AI provenance 持久化、按 chain/node/topology edge 批量查询 physical constraints 与 report 聚合使测试通过
- [x] 3.5 先增加 relationship policy 失败测试并实现 `scoped_to_economy`、`uses_commodity`、`produces_commodity`、`observed_by_benchmark`、`mapped_to_sector`；拒绝海外 market `COVERS_SECTOR` 中国板块

## 4. Neo4j active-only 静态投影（TDD）

- [x] 4.1 先在 repository 与 `backend/internal/apps/graphprojection/` 增加失败测试，覆盖 approved active industry chain、active membership/topology、已审阅 entity_edges、缺失端点跳过，并断言 physical constraints 不进入 graph source
- [x] 4.2 扩展 graph source DTO/query、mapping 和 projector，使统一 `Entity`、`projection_namespace` 与 active-only 测试通过；不得引入 physical constraint 或 observation 投影
- [x] 4.3 使用 fake graph writer 验证全球 benchmark 经 chain/node 到中国 sector 的客观路径，且不生成海外 market 覆盖中国 sector 的关系

## 5. 两条试点与有状态分层门禁

- [x] 5.1 用户已批准两条链、26 个去重节点与 membership；已新增 `industry_chains_v1.json` 并完成 seed/loader/report 测试，未执行数据库写入
- [x] 5.2a 用户已批准 canonical topology 方向；已在版本化可执行 seed 中准备 24 条 topology 并完成 policy/report 测试，无 `substitutes_for`
- [ ] 5.2b 仅在用户逐项批准后，才将 15 条 physical constraint candidate 或 12 条 `mapped_to_sector` candidate 从 review-only fixture 晋级为可执行 seed；economy/commodity/benchmark 保持空
- [x] 5.3 已在 `stateful-execution-plan.md` 展示 migration、master seed、membership、topology、physical constraint、`mapped_to_sector`、Neo4j rebuild/query 的范围、顺序、预计统计、验证与回滚边界；所有 stateful 操作仍需逐层明确授权
- [x] 5.4a 2026-07-13 在用户单独授权后，为 local PostgreSQL 创建并校验 pg_dump 备份，仅执行 `000014`，并以只读 Query 验收 version=14、4 张空新表、profile 增量列、约束/索引、33 个既有节点和 planned ID 零冲突
- [x] 5.4b Layer 2 preflight 发现默认 seed 会夹带后续层后，按 RED→GREEN 增加显式 `industry-chain-master` scope、CLI 冲突校验和 operation/final-table 双口径 report；测试证明跳过 industry batch 与无关数据族，本步骤未执行 DML
- [x] 5.4c 2026-07-13 在独立授权、备份和写前只读门禁后，仅执行一次 `industry-chain-master` scope；report与只读Query确认2 chain、26 node及最终表级23/5、2/0、21/5，后续表和关系仍为0
- [x] 5.4d Layer 3只读preflight确认2 chain/26 node active、27个membership ID与tuple无冲突；按RED→GREEN增加显式 `industry-chain-membership` scope，测试证明batch仅含27 memberships且其他数据族为空，本步骤未执行DML
- [x] 5.4e 2026-07-13在独立授权、备份和写前只读门禁后，仅执行一次`industry-chain-membership`；report与只读Query确认27/27 active、12/15、ID/tuple唯一、共享节点两链，其他表不变
- [ ] 5.4f 进入Layer 4 topology独立preflight与必要scope实现；Layer 3验收不得推定topology Write授权

## 6. 完整验证与 Apply 后 Review

- [ ] 6.1 运行 migration、domain、entityfoundation、repository、graphprojection 目标包测试和 `go test ./...`
- [ ] 6.2 运行 `openspec validate add-industry-chain-node-foundation`、`git diff --check` 和 scoped `git status --short`
- [ ] 6.3 提交 scoped Apply diff、验证证据、实际数据库/Neo4j 状态与未执行项，等待用户第二次人工 Review；批准前不得 Sync、Archive 或 Deliver
