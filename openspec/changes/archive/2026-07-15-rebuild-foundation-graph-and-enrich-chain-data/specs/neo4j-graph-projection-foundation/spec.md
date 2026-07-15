## ADDED Requirements

### Requirement: 当前实体模型基础投影
系统 SHALL 从 PostgreSQL 当前 active `entity_nodes`、`entity_edges` 和 `chain_node_relations` 重建 Neo4j 基础投影，不得读取已删除的旧产业表或旧关系类型。

#### Scenario: 投影当前节点集合
- **WHEN** graph projector 读取最终 PostgreSQL baseline
- **THEN** 系统必须仅投影 active `alliance_org`、`economy`、`chain_node`
- **AND** 不得因关系端点而扩张到 market、index、benchmark 或其他实体类型

#### Scenario: entity_edges 两端过滤
- **WHEN** projector 读取一条 active `entity_edges`
- **THEN** 只有 from/to 两端都属于本次已投影节点集合时才能投影该关系
- **AND** 任一端不在集合内时必须跳过该关系，不得生成孤儿边

#### Scenario: 投影 chain_node_relations
- **WHEN** projector 读取 active `chain_node_relations`
- **THEN** 系统必须投影两端均存在且 active 的 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`
- **AND** 关系类型与方向必须保持 PostgreSQL 原始事实

#### Scenario: additive 关系只从最终 PostgreSQL baseline 同步
- **WHEN** usable-map additive 关系已完成 PostgreSQL 写后验收并另行获得 local Neo4j R3 授权
- **THEN** 系统必须按该 R3 包冻结的最终 PostgreSQL accepted baseline 同步全部批准关系
- **AND** 不得执行已 superseded 的局部 4-edge sync 包，不得同步 blocked/rejected 候选

### Requirement: local disposable Neo4j 分层重建
系统 SHALL 把 local Tidewise Neo4j 作为可由已验收 PostgreSQL projection baseline 重建的 disposable projection，并将 cleanup 与 rebuild 作为两个独立 R3 授权层；不得要求或宣称 Neo4j backup/rollback。

#### Scenario: 独立 cleanup
- **WHEN** local 环境、Tidewise namespace 和 PG baseline 已冻结且 cleanup 获得独立 R3 授权
- **THEN** 系统必须只清空 Tidewise 节点与关系并 Query 为零，同时保留 database、约束、索引、配置和 PostgreSQL

#### Scenario: 独立 rebuild 与分类验收
- **WHEN** cleanup 已验收、PG baseline 未漂移且 rebuild 获得下一层独立 R3 授权
- **THEN** 系统必须分别输出 alliance_org、economy、chain_node 节点 counts 和各 relation type/count
- **AND** missing、duplicate、orphan、legacy 必须均为 0
