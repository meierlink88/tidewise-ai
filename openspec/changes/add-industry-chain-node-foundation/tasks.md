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

- [ ] 5.1 仅在用户批准两条链和节点清单后新增 industry chain、chain node 改进与 membership seed，并运行 seed/loader 测试；不得执行数据库写入
- [ ] 5.2 仅在用户逐项批准 topology、physical constraint 和各跨实体关系族后新增版本化 seed 文件并运行 policy/report 测试
- [ ] 5.3 展示 migration、master seed、membership、topology、physical constraint、每个关系族和 Neo4j rebuild 的范围、顺序、影响及回滚边界，并分别取得明确 stateful 授权
- [ ] 5.4 按层执行 `Review → Write → Rebuild → Query`，每层由用户验收后才能推进下一层

## 6. 完整验证与 Apply 后 Review

- [ ] 6.1 运行 migration、domain、entityfoundation、repository、graphprojection 目标包测试和 `go test ./...`
- [ ] 6.2 运行 `openspec validate add-industry-chain-node-foundation`、`git diff --check` 和 scoped `git status --short`
- [ ] 6.3 提交 scoped Apply diff、验证证据、实际数据库/Neo4j 状态与未执行项，等待用户第二次人工 Review；批准前不得 Sync、Archive 或 Deliver
