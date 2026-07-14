## ADDED Requirements

### Requirement: 当前实体模型基础投影
系统 SHALL 从 PostgreSQL 当前 active `entity_nodes`、`entity_edges` 和 `chain_node_relations` 重建 Neo4j 基础投影，不得读取已删除旧产业表或旧关系类型。

#### Scenario: 投影当前节点和关系
- **WHEN** graph projector 读取最终 PostgreSQL baseline
- **THEN** 系统必须投影 active `alliance_org`、`economy`、`chain_node` 及 PG 已批准关系
- **AND** 不得依赖 `sector_profiles`、`industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`、`member_of_chain`、`supplies_to` 或 `substitutes_for`

#### Scenario: targeted tests 验证最小适配
- **WHEN** 开发者运行当前 query、mapper、projector 和 CLI targeted tests
- **THEN** 测试必须覆盖当前 source、四类产业关系、namespace cleanup/rebuild、run counts 与失败处理

### Requirement: local disposable Neo4j 分层重建
系统 SHALL 把 local Tidewise Neo4j 作为可由已验收 PostgreSQL projection baseline 重建的 disposable projection，并将 cleanup 与 rebuild 作为两个独立 R3 授权层；不得要求或宣称 Neo4j backup/rollback。

#### Scenario: 独立 cleanup
- **WHEN** local 环境、Tidewise namespace 和 PG baseline 已冻结且 cleanup 获得独立 R3 授权
- **THEN** 系统必须只清空 Tidewise 节点与关系并 Query 为零，同时保留 database、约束、索引、配置和 PostgreSQL

#### Scenario: 独立 rebuild 与分类计数
- **WHEN** cleanup 已验收、PG baseline 未漂移且 rebuild 获得下一层独立 R3 授权
- **THEN** 系统必须分别输出 alliance_org、economy、chain_node 节点 counts 和各 relation type/count
- **AND** missing、duplicate、orphan、legacy 必须均为 0

### Requirement: 首批 PG-first 同步和 typed Cypher 验收
系统 SHALL 只在首批 PostgreSQL Write/Query 验收后，以单独 R3 授权同步对应 Neo4j 投影；多跳能力只通过验收 Cypher 验证，不得新增查询 API、service、推理引擎或派生关系。

#### Scenario: 从 PG accepted baseline 同步
- **WHEN** first-batch PG accepted baseline 已冻结且 Neo4j sync 获得独立 R3 授权
- **THEN** projector 必须只同步批准 manifest 的 chain_node 与关系，并证明 counts/type 一致及 missing/duplicate/orphan/legacy=0

#### Scenario: 验收 typed 多跳方向
- **WHEN** 从指定 chain_node 执行验收 Cypher
- **THEN** `input_to` 必须顺向、`depends_on` 必须反向组合，并返回 depth、path 与 evidence
- **AND** `is_subcategory_of` 与 `is_component_of` 不得计入上下游路径
