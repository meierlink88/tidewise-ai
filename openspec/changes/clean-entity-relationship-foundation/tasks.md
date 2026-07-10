## 1. 关系来源 schema 和校验基础

- [x] 1.1 编写 migration 静态测试，覆盖 `entity_edges` 增量增加 `source_name`、`source_url`、`verified_at`，并确认 migration 不包含删除实体、清空业务表或重建全库语句。
- [x] 1.2 编写关系 loader 和 policy validator 测试，覆盖 7 类正式关系的 from/to 实体类型、方向、来源字段、URL、核验时间、自环、重复关系、悬空实体和推理性字段。
- [x] 1.3 编写 relationship repository 和 seed report 测试，覆盖来源字段幂等写入、更新、unchanged、失败和按关系类型统计。
- [x] 1.4 编写 graph relation mapper 测试，确保 7 类正式关系都映射为明确 Neo4j 类型，特别覆盖当前尚未显式映射的 `applies_to`。
- [x] 1.5 实现增量 migration、关系来源模型、policy validator、repository、report 和 graph relation mapper，使上述测试通过。

## 2. 建立空关系基线

- [x] 2.1 编写 seed 文件静态测试，确认实体主数据路径仍完整加载，并且空关系基线不会自动加载原有 78 条历史样例关系。
- [x] 2.2 将关系 seed 收敛到按关系族管理的目录，移除默认旧样例关系并保留可逐批加入的空基线结构。
- [x] 2.3 在 local PostgreSQL 应用增量 migration，验证实体节点、profile、采集和事件相关表数据未丢失。
- [x] 2.4 清空前核验 local 数据库目标和数量，记录 `entity_nodes`、各 profile 与 `entity_edges` 数量，不连接 uat 或 prod。
- [x] 2.5 在事务中清空 local PostgreSQL `entity_edges`，验证关系为零且实体节点与 profile 数量保持不变。
- [x] 2.6 清空 local Neo4j 全部节点和关系数据，验证节点数和关系数均为零，同时约束和索引仍存在。
- [x] 2.7 再次运行 `entity-seed`，验证实体主数据保持幂等且 PostgreSQL 关系仍为零。

## 3. 第一批：联盟组织与国家/经济体关系

- [x] 3.1 基于联盟组织官方或权威来源整理 `member_of` 审阅清单，包含中文名称、实体 key、关系方向、来源名称、来源 URL 和核验时间。
- [x] 3.2 等待用户逐项 review 并明确确认第一批关系清单，未确认前不得写入正式 seed 或数据库。
- [x] 3.3 编写第一批 seed fixture 和校验测试，再将已确认 `member_of` 关系写入正式关系 seed 文件。
- [x] 3.4 运行 `entity-seed` 写入 local PostgreSQL，核验 `member_of` 数量、方向、端点和来源字段。
- [x] 3.5 使用 TDD 将 Neo4j 节点标签收敛为单一 `Entity`，继续通过 `projection_namespace` 隔离本系统投影，重建并核验节点与 `MEMBER_OF` 关系。
- [ ] 3.6 等待用户完成第一批联盟关系图谱验收。

## 4. 第二批：经济体与市场关系

- [ ] 4.1 整理 `has_market` 权威来源审阅清单并等待用户确认。
- [ ] 4.2 先编写 seed fixture 和校验测试，再写入已确认 `has_market` 关系。
- [ ] 4.3 写入 PG、核验关系与来源字段，重建 Neo4j 并完成该批图谱验收。

## 5. 第三批：市场与指数关系

- [ ] 5.1 整理 `tracks_index` 权威来源审阅清单并等待用户确认。
- [ ] 5.2 先编写 seed fixture 和校验测试，再写入已确认 `tracks_index` 关系。
- [ ] 5.3 写入 PG、核验关系与来源字段，重建 Neo4j 并完成该批图谱验收。

## 6. 第四批：公司与证券关系

- [ ] 6.1 整理 `issues` 权威来源审阅清单并等待用户确认。
- [ ] 6.2 先编写 seed fixture 和校验测试，再写入已确认 `issues` 关系。
- [ ] 6.3 写入 PG、核验关系与来源字段，重建 Neo4j 并完成该批图谱验收。

## 7. 第五批：公司与产业链节点关系

- [ ] 7.1 整理 `participates_in` 权威来源审阅清单并等待用户确认，不把板块归类、利好利空或传导判断写成客观关系。
- [ ] 7.2 先编写 seed fixture 和校验测试，再写入已确认 `participates_in` 关系。
- [ ] 7.3 写入 PG、核验关系与来源字段，重建 Neo4j 并完成该批图谱验收。

## 8. 第六批：人物从属关系

- [ ] 8.1 整理 `affiliated_with` 权威来源审阅清单并等待用户确认，关系只表达可核验的当前任职或从属事实。
- [ ] 8.2 先编写 seed fixture 和校验测试，再写入已确认 `affiliated_with` 关系。
- [ ] 8.3 写入 PG、核验关系与来源字段，重建 Neo4j 并完成该批图谱验收。

## 9. 第七批：指标适用关系

- [ ] 9.1 整理 `applies_to` 定义依据和审阅清单并等待用户确认。
- [ ] 9.2 先编写 seed fixture 和校验测试，再写入已确认 `applies_to` 关系。
- [ ] 9.3 写入 PG、核验关系与来源字段，重建 Neo4j 并完成该批图谱验收。

## 10. 最终验证和说明

- [ ] 10.1 增加最终数据一致性检查，核验 repo seed、PostgreSQL active `entity_edges` 和 Neo4j 关系数量及类型一致。
- [ ] 10.2 更新实体 seed 和 Neo4j 本地说明，记录关系 review gate、来源字段、空基线、逐批写入和重建命令。
- [ ] 10.3 运行 `go test ./...`。
- [ ] 10.4 运行 `openspec validate clean-entity-relationship-foundation` 和 `openspec validate --all`。
