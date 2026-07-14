## ADDED Requirements

### Requirement: 新产业基础图谱可从 PostgreSQL 完整重建
系统 SHALL 只从 PostgreSQL 当前 active 实体与已批准关系重建 Neo4j Tidewise 投影，并支持 `alliance_org`、`economy`、`chain_node`、通用 `entity_edges` 和独立 `chain_node_relations` 的当前模型；不得读取已删除的 sector/industry-chain legacy 表或把 Neo4j 作为事实源。

#### Scenario: 读取当前节点和关系来源
- **WHEN** graph projector 在前置 change 完整 Deliver 后读取最终 PostgreSQL baseline
- **THEN** 节点 source 必须只依赖当前存在的 `entity_nodes` 与必要 profile，关系 source 必须分别读取 active `entity_edges` 和 active `chain_node_relations`
- **AND** 不得读取 `sector_profiles`、`industry_chain_profiles`、`industry_chain_memberships` 或 `industry_chain_topology_edges`

#### Scenario: 投影四类产业节点关系
- **WHEN** PostgreSQL 包含已批准的 active `is_subcategory_of`、`is_component_of`、`input_to` 或 `depends_on` 关系
- **THEN** Neo4j 必须使用固定安全 relationship type 投影并保留 stable ID、原始类型、source table、evidence/provenance、verified time、状态和 namespace
- **AND** 不得恢复 `member_of_chain`、`supplies_to` 或 `substitutes_for` 产业关系

#### Scenario: 自动化验证当前投影来源
- **WHEN** 开发者运行 graph projection repository、mapper、projector 和 CLI 测试
- **THEN** 测试必须证明删除表无引用、四类关系映射正确、inactive/缺失端点 fail-closed、fake writer 重建幂等且普通 suite 不依赖真实 Neo4j

### Requirement: local disposable projection 分层重建
系统 SHALL 在用户明确授权的产品 1.0 探索期 local Neo4j 中，把 Tidewise namespace 视为可由 PostgreSQL 确定性恢复的 disposable projection，并将 cleanup 与 rebuild 作为两个独立 R3 层执行；该规则不得应用于 UAT、prod、shared 或其他 namespace。

#### Scenario: 清理 local Tidewise 投影
- **WHEN** 最终 PostgreSQL baseline 已冻结、环境身份确认为 disposable local Neo4j 且 `local-neo4j-foundation-cleanup` 获得独立 R3 授权
- **THEN** 系统必须只删除 Tidewise 投影节点和关系并保留 database、约束、索引和连接配置
- **AND** 不要求创建 Neo4j backup 或 rollback 副本

#### Scenario: 从已验收 PG baseline 重建
- **WHEN** cleanup Query 已通过、PG baseline identity/scope/count/hash/schema 复验无漂移且 `local-neo4j-foundation-rebuild` 获得下一层独立 R3 授权
- **THEN** 系统必须从 PostgreSQL 全量重建批准范围并输出 PG/Neo4j counts、类型、重复、悬空、failed/skipped 与旧关系残留的一致性报告

#### Scenario: 重建失败保持 fail-closed
- **WHEN** cleanup 后 projector 失败、超时、出现 skipped/failed 或一致性断言不通过
- **THEN** Neo4j 必须保持空、partial 或 stale 状态并停止，不得恢复旧图、手工补图或反写 PostgreSQL

### Requirement: 首批数据只从已验收 PG 同步
系统 SHALL 只在首批 PostgreSQL Write 与 Query 独立验收后，依据冻结的 post-write PostgreSQL accepted baseline 同步 local Neo4j；R2 PostgreSQL 授权不得推定 R3 Neo4j 授权。

#### Scenario: 同步批准首批事实
- **WHEN** 首批 PG accepted baseline 已冻结且 `first-batch-neo4j-sync` 获得单独 R3 授权
- **THEN** projector 必须只同步批准的节点与关系，并通过 Query 证明 Neo4j 没有额外事实、重复、悬空或旧投影残留

#### Scenario: 拒绝未验收或漂移输入
- **WHEN** 首批 PG Query 未验收、manifest/hash/count/schema 漂移或 Neo4j R3 授权缺失
- **THEN** 系统不得执行同步，也不得用旧批准或 PostgreSQL Write 授权替代 Neo4j 授权

### Requirement: typed 关系多跳下游查询
系统 SHALL 保持 Neo4j 原始 relationship 与 PostgreSQL 相同方向，并在下游查询中把 `input_to` 按顺向、`depends_on` 按反向统一解释；`is_subcategory_of` 和 `is_component_of` 不得计入上下游路径。

#### Scenario: 沿 input_to 顺向穿透
- **WHEN** PostgreSQL 保存 `A input_to B` 且查询从 A 向下游遍历
- **THEN** 路径必须从 A 到 B，并返回原始 relation ID、depth 和 evidence

#### Scenario: 沿 depends_on 反向穿透
- **WHEN** PostgreSQL 保存 `A depends_on B` 且查询从 B 向下游遍历
- **THEN** 路径必须从 B 到 A，同时保留原始 PG 边 `A -> B` 的 relation ID、类型和 evidence

#### Scenario: 分类边不进入下游路径
- **WHEN** 候选路径只由 `is_subcategory_of` 或 `is_component_of` 连接
- **THEN** 下游查询不得把该路径计为供应、依赖或影响穿透
