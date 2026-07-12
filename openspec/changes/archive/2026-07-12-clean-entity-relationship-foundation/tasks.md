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
- [x] 3.6 等待用户完成第一批联盟关系图谱验收。

## 4. 第二批：经济体与市场关系

- [x] 4.1 整理现有市场的 `has_market` 权威来源审阅清单并完成用户确认：通过 27 条无歧义关系，暂缓欧洲股票市场、ICE 和 3 条全球聚合市场关系。
- [x] 4.2 整理第一版事件投研所需的补充市场实体审阅清单，覆盖核心主权债券市场、关键商品交易场所、沙特阿拉伯、印度尼西亚和越南股票市场，明确抽象市场与交易场所分类，并等待用户确认。
- [x] 4.3 先编写市场实体和 `has_market` seed fixture、分类校验及关系校验测试，再写入 review 通过的新增市场实体和全部已确认关系。
- [x] 4.4 运行 `entity-seed` 写入 PG，核验新增市场 profile、关系数量、方向和来源字段。
- [x] 4.5 重建 Neo4j，核验 `HAS_MARKET` 投影并完成该批图谱验收。

## 5. 第三批：市场与指数关系

- [x] 5.1 整理 `tracks_index` 权威来源审阅清单；经语义复核后将范围收紧为正式指数，并登记 10 个延后到 benchmark change 的价格、收益率和参考利率概念。
- [x] 5.2 先编写市场、指数和 `tracks_index` seed fixture、归属及 benchmark 排除测试，再写入 review 通过的 43 个正式指数和 43 条关系。
- [x] 5.3 对 local PG 精确移除 10 个误归类 index 及对应 `tracks_index`，重新运行 `entity-seed`，核验 43 个指数、43 条关系、方向和来源字段。
- [x] 5.4 重建 Neo4j，核验 43 条 `TRACKS_INDEX` 投影并完成该批图谱验收。

## 6. 后续关系优先级调整

- [x] 6.1 完成投研优先级复核，保留 `issues` 候选清单但不写入数据，并将 `issues`、`participates_in`、`affiliated_with`、`applies_to` 延后到 benchmark 与市场产业传导基础完成之后。

## 7. 最终验证和说明

- [x] 7.1 增加最终数据一致性检查，核验 repo seed、PostgreSQL active `entity_edges` 和 Neo4j 关系数量及类型一致。
- [x] 7.2 更新实体 seed 和 Neo4j 本地说明，记录关系 review gate、来源字段、空基线、逐批写入、暂缓关系和重建命令。
- [x] 7.3 运行 `go test ./...`。
- [x] 7.4 运行 `openspec validate clean-entity-relationship-foundation` 和 `openspec validate --all`。
