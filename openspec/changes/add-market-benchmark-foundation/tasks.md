## 1. Benchmark schema 与领域类型

- [ ] 1.1 编写 migration 静态测试，覆盖 `benchmark_profiles`、`benchmark_observations`、外键、幂等唯一约束、时间索引、quality status 和非破坏性 SQL。
- [ ] 1.2 编写领域模型与实体类型测试，覆盖 `EntityTypeBenchmark`、benchmark profile 字段、observation quality status 和合法枚举。
- [ ] 1.3 实现增量 migration、benchmark 领域类型和 schema，使对应测试通过。
- [ ] 1.4 在 local PostgreSQL 应用 migration，验证既有 548 个实体、306 条关系及采集和事件表数据未丢失。

## 2. 实体 Seed 与关系策略

- [ ] 2.1 编写 loader、profile extractor、路径和 report 测试，覆盖 benchmark seed、可空 official series code、profile 数量及幂等写入。
- [ ] 2.2 编写关系策略测试，覆盖 `market -> observes_benchmark -> benchmark`、`benchmark -> measures -> metric`、`benchmark -> references -> commodity/instrument` 的方向、端点、来源和推理字段拒绝。
- [ ] 2.3 编写 graph relation mapper 测试，覆盖 `OBSERVES_BENCHMARK`、`MEASURES` 和 `REFERENCES` 显式映射。
- [ ] 2.4 实现 benchmark profile repository、seed report、关系策略和图关系映射，使对应测试通过。

## 3. Benchmark Observation Repository

- [ ] 3.1 编写 observation repository 单元测试，覆盖创建、同来源同时间幂等更新、不同来源共存、非法 benchmark 类型、质量状态和查询排序。
- [ ] 3.2 实现 benchmark observation 领域模型与 PostgreSQL repository，使对应测试通过。
- [ ] 3.3 增加明确测试，保证 graph projector 的 source rows 和 Neo4j 写入不包含 observation。

## 4. 首批 Benchmark 审阅

- [ ] 4.1 审计 `metric:fear_index` 的 repo 与 local PG 引用，形成迁移为 `metric:implied_volatility` 的精确清理方案。
- [ ] 4.2 整理首批 10 个 benchmark 审阅清单，逐项包含名称、key、benchmark type、官方 series code 或空值、provider、期限、标的、币种、单位、频率和权威来源。
- [ ] 4.3 整理三类关系审阅清单，明确每个 benchmark 对应的 market、metric、commodity 或 instrument。
- [ ] 4.4 等待用户明确确认 benchmark 定义、metric 迁移和关系清单；确认前不得写入正式 seed、PostgreSQL 或 Neo4j。

## 5. 首批 Seed 与 PostgreSQL 验收

- [ ] 5.1 先编写首批 benchmark、metric 迁移和关系 seed fixture 测试，再写入用户确认的数据。
- [ ] 5.2 在事务中迁移 `metric:fear_index` 为 `metric:implied_volatility`，核验不存在悬空引用或重复 VIX 语义。
- [ ] 5.3 运行 `entity-seed` 写入 local PostgreSQL，核验 10 个 benchmark profile、三类关系数量、方向、来源字段和幂等 report。
- [ ] 5.4 验证 `benchmark_observations` 保持为空，确认本 change 没有伪造或导入实时行情值。

## 6. Neo4j 重建与验收

- [ ] 6.1 重建 local Neo4j，核验 benchmark 节点、三类关系和 PG active 事实一致。
- [ ] 6.2 验证 Neo4j 不包含 observation 节点、不恢复 10 个旧 index 节点，并继续使用单一 `Entity` 标签和 `projection_namespace=tidewise`。
- [ ] 6.3 等待用户完成 benchmark 图谱验收。

## 7. 最终验证与说明

- [ ] 7.1 更新实体 seed、migration 和图投影本地说明，记录 benchmark 语义边界、review gate、PG/Neo4j 职责和后续采集边界。
- [ ] 7.2 增加 repo seed、PostgreSQL 和 Neo4j 最终一致性检查。
- [ ] 7.3 运行 `go test ./...`。
- [ ] 7.4 运行 `openspec validate add-market-benchmark-foundation` 和 `openspec validate --all`。

