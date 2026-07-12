## 1. Apply 前 Review 与范围冻结

- [ ] 1.1 由用户先 Review 并批准 10 张核心表的字段、主外键、枚举、唯一约束、索引和事实/观察/推理分层，重点确认产业链指标定义、binding 与现有 `metric_profiles` 分离，再 Review 链范围、节点粒度、跨实体关系、typed observation 范围和 stateful 操作门禁；未批准不得进入后续任务
- [ ] 1.2 在 `candidate-review.md` 整理 AI 算力基础设施、半导体制造、机器人三条首批试点，每链 10–20 节点、现有 33 节点复用/改进/新增判断、三链去重后约 30–50 节点与权威来源；将新能源汽车/储能、创新药/生物制造列为第二批且不进入本 change seed
- [ ] 1.3 为三条试点逐项整理 `market story → system change → required parts → layers → scarce constraints → evidence → risks/falsification` 映射，明确哪些进入主数据/observation，哪些只属于未来 reasoning result
- [ ] 1.4 对 membership、topology、economy/commodity/benchmark/sector/metric 关系分别生成逐项 Review 清单，记录文件所有权和后续 `Review → Write → Rebuild → Query` 顺序

## 2. PostgreSQL schema 与领域模型（TDD）

- [ ] 2.1 先在 `backend/migrations/` 增加失败的静态测试，逐字段覆盖 `industry_chain_profiles`、`chain_node_profiles` 增量字段、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_constraints`、`industry_chain_metric_definitions`、`industry_chain_metric_bindings`、`observation_records` 和两个 typed observation 表的 FK、枚举、partial unique index、幂等、索引、非破坏性与回滚约束
- [ ] 2.2 运行 migration 目标测试确认 RED，再追加单一增量 migration 与安全 down/兼容策略使测试 GREEN；不得执行 migration apply
- [ ] 2.3 先在 `backend/internal/domain/` 增加 table-driven tests，覆盖 `EntityTypeIndustryChain`、profile、membership、topology、constraint、industry chain metric definition/binding、observation 类型的合法与非法状态
- [ ] 2.4 实现最小领域类型与 validator 使目标包测试通过，并执行 REFACTOR 保持现有实体 API 兼容

## 3. Entity foundation loader 与 repository（TDD）

- [ ] 3.1 先在 `backend/internal/apps/entityfoundation/seed/` 增加 loader/fixture 失败测试，覆盖产业链 profile、改进 chain node profile、membership、topology、来源、重复、自环、非成员端点和禁止推理字段
- [ ] 3.2 扩展 manifest、loader、paths 和 validator 使测试通过，不加入未经 Review 的正式产业链 seed
- [ ] 3.3 先增加 MemoryRepository 与 PostgresRepository 失败测试，覆盖产业链定义、membership、topology 的原子幂等 upsert、report 统计和回滚等价性
- [ ] 3.4 实现 repository 接口、SQL 与 report 聚合使测试通过，connector、cmd 和 handler 不得绕过 repository
- [ ] 3.5 先增加跨实体 relationship policy 失败测试，覆盖 `scoped_to_economy`、`uses_commodity`、`produces_commodity`、`observed_by_benchmark`、`represented_by_sector` 的端点和方向，并拒绝产业链专用 `measured_by`
- [ ] 3.6 实现 relationship policy 与 provenance 校验，明确拒绝海外 `market -> 中国 sector` 的错误 `covers_sector`

## 4. Typed observation repository 契约（TDD）

- [ ] 4.1 先在 `backend/internal/repositories/` 增加 metric definition/binding 与 node/flow observation 失败测试，覆盖通用 metric 可选桥接、subject type、chain scope、approved binding、envelope + typed row 原子写入、质量状态、revision 和幂等更新
- [ ] 4.2 实现 MemoryRepository typed observation writer/query，使无 EAV 字段的目标测试通过
- [ ] 4.3 增加 PostgresRepository SQL/扫描与可重复 integration 边界测试，使用本地测试数据库或 fake，不连接生产数据库
- [ ] 4.4 实现 PostgresRepository typed observation writer/query，并保持 ingestion 只通过结构化 contract 交接；本 change 不新增真实 connector

## 5. Neo4j active-only 投影（TDD）

- [ ] 5.1 先在 `backend/internal/repositories/` 和 `backend/internal/apps/graphprojection/` 增加失败测试，覆盖 approved active industry chain、active membership/topology、产业链跨实体关系与 observation 排除
- [ ] 5.2 扩展 graph source DTO/query、mapping 枚举和 projector，使测试验证统一 `Entity` 标签、`projection_namespace`、缺失端点跳过与 active-only 行为
- [ ] 5.3 使用 fake graph writer 验证全球 benchmark 经 chain/node 到中国 sector 的客观路径，且不生成海外 market `COVERS_SECTOR` 中国板块

## 6. 已批准 seed 与有状态分层门禁

- [ ] 6.1 仅在用户批准链和节点清单后新增 `industry_chains.json`、改进 `chain_nodes.json` 与 membership seed，并运行 seed 文件/loader 测试；不得执行数据库写入
- [ ] 6.2 仅在用户逐层批准 topology 与各跨实体关系族后新增对应版本化 seed 文件，并运行 relationship policy 与 seed report 测试
- [ ] 6.3 展示 migration apply 的范围、顺序、预期影响和回滚边界，取得单独明确批准后才执行 schema 写入并查询验证
- [ ] 6.4 对 master seed、membership、topology 和每个跨实体关系族分别执行 `Review → Write → Rebuild → Query`；每层都在用户验收后才能推进下一层
- [ ] 6.5 仅在 PostgreSQL 对应层验收且用户单独批准后执行 Neo4j rebuild，查询 active 节点、关系枚举、namespace、无 observation 投影和无错误跨市场 COVERS

## 7. 完整验证与 Apply 后 Review

- [ ] 7.1 运行 migration、domain、entityfoundation、repository、graphprojection 目标包测试并读取新鲜结果
- [ ] 7.2 在 `backend/` 运行 `go test ./...`，确认没有真实网络、生产数据库或 secret 依赖
- [ ] 7.3 运行 `openspec validate add-industry-chain-node-foundation`、`git diff --check` 和 scoped `git status --short`
- [ ] 7.4 提交 scoped Apply diff、测试证据、数据库/Neo4j 实际状态与未执行项，等待用户第二次人工 Review；批准前不得 Sync、Archive 或 Deliver
