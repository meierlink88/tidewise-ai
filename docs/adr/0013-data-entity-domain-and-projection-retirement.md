---
status: accepted
date: 2026-08-12
supersedes_in_part: 0009-event-semantic-qdrant-projection-and-agentrun-retrieval.md, 0010-event-semantic-entity-first-cross-type-resolution.md
superseded_in_part_by: 0014-agentrun-consumes-retained-qdrant-snapshot.md
superseded_in_part_by_2: 0016-tidewise-ai-2-object-schema-and-independent-region.md
superseded_in_part_by_3: 0027-retire-agent-run.md
---

# Data 收敛 Entity 领域并退役 Neo4j/Qdrant 投影

## 背景

Data 的 Entity 类型与规则分散在通用 model、Entity seed、Industry relationship import、
Neo4j graph projection、Qdrant semantic projection 和多个 PostgreSQL wrapper 中。场景 package
与通用模型桶不符合当前 Kratos 领域布局，也让一次性构建能力看起来属于正式 Entity 运行时。

业务已决定由 Tidewise Reason 负责 Entity seed、关系包构建/导入前决策和历史收敛能力。
Data 继续拥有正式 PostgreSQL Entity 事实，但不再负责将 Entity 投影到 Neo4j 或 Qdrant。

## 决策

- Data 使用单数 `entity` 领域统一拥有 Entity ABox、Profile、Relation、Industry Chain
  topology、Entity Type Definition TBox 和 Benchmark Observation 事实。
- Raw Document/Ingest Status 不属于 Entity 或 Event，暂时收敛为独立、明确退役的
  `rawdocument` 兼容领域，并保持现有管理读取 API 合同。
- Data 删除 Entity seed、Industry relationship import、历史收敛批次及其命令、规则、
  Adapter、测试和部署能力；这些能力的未来 owner 是 Tidewise Reason。
- Data 删除 PG→Neo4j 与 PG→Qdrant 的全部投影写入能力、命令、配置和部署门禁。
- Data Runtime Health 不再探测或报告 Neo4j；Admin Runtime Health 同步移除 Neo4j。
- 本决策不创建 Reason→Data Entity publication API。该跨项目合同需要独立设计；Reason
  不得因此直写 Data PostgreSQL。
- 已执行 migration、历史投影/import 表、receipt 和数据保持不变，不执行 destructive DDL。
- AgentRun Qdrant consumer 不在本变更中重写，但依赖 projection freshness 的 Event Semantic
  执行暂停，直到新的 projection owner 与 rollout 合同获批。

## 与既有权威的关系

- 取代 ADR-0009 中 Data-owned PG→Qdrant projector、embedding Port 和 Qdrant writer 的决策。
- 取代 ADR-0010 中“Data-owned projector ownership 不变”的部分；Entity-first PostgreSQL
  最终校验、候选隔离和 Event Semantic 其他合同继续有效。
- `local-industry-graph-projection-v1`、`uat-industry-graph-projection-v1`、
  `event-semantic-qdrant-retrieval-v2` 中 Data projector 部分和
  `uat-event-semantic-qdrant-rollout-v1` 的 projection rollout 部分停止作为当前运行权威。
- PostgreSQL 仍是正式事实源；Industry Chain Graph Edge 仍是 Entity 领域关系事实，而不是
  Neo4j projection artifact。

## 影响

Data image、local Compose 与 UAT CD 不再包含 Entity seed、Industry relationship import、
Industry graph projector 或 Event Semantic projector。Data Server 不再需要 Neo4j driver、
credentials、configuration or lifecycle management。历史 migration ledger 与数据保持可由旧版本
读取，应用 rollback 不运行 down migration，也不触碰外部 Neo4j/Qdrant 状态。
