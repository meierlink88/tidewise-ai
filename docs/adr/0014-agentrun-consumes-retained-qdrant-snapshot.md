---
status: accepted
date: 2026-08-12
supersedes_in_part: 0013-data-entity-domain-and-projection-retirement.md
---

# AgentRun 继续消费保留的 Qdrant Entity 快照

## 背景

ADR-0013 退役 Data-owned PostgreSQL→Qdrant projector 后，同时暂停了 AgentRun Event
Semantic worker。进一步核实表明，Event Semantic 最终事实始终写入 Data PostgreSQL；Qdrant
只保存供 Entity 候选召回使用的 `entity_semantic_v1` 快照。AgentRun 的 Qdrant reader、
embedding、候选校验与 Event Semantic workflow 均未删除，外部 Qdrant collection 也不属于
应用发布或回退的 mutation 范围。

业务接受在新 projection owner 落地前继续使用 UAT 已存在的 Qdrant 快照，并明确承担新增或
修改 Entity 不会进入召回目录的已知限制。

## 决策

- AgentRun Server 启动时恢复 Event Semantic worker，并执行一次初始通知；既有周期 reconcile、
  Work Item、Data API、Eino workflow、模型和审核合同不变。
- AgentRun 继续只读消费现有 `entity_semantic_v1` collection。Qdrant health/readiness、point
  provenance、projection version、embedding model 与 fingerprint 校验保持不变。
- Data-owned Entity/Variable Definition projector、embedding Port、Qdrant writer、collection
  rebuild 和 rollout gate 继续退役；本决策不恢复任何 Data→Qdrant 或 Data→Neo4j 写入。
- Event Semantic Submission、Review 与最终结果继续由 Data PostgreSQL 权威保存，不写入
  Qdrant 或 Neo4j。
- retained snapshot 之后新增、修改或退役的 Entity 可能无法被召回，或仍以旧 identity 内容被
  召回。该限制只能由后续批准的新 projection owner 与版本化 rollout 合同消除。

## 影响

当前完整采集链可继续执行：采集与 Event Fact 发布、Event Semantic 提炼、外部 Research
Analysis Context 消费及 Theme/Reason Tree 发布。API、Schema、migration、Agent Version、
Prompt、模型和外部基础设施数据均不改变。

应用回退到 ADR-0013 对应版本会重新暂停 worker；回退不修改 PostgreSQL、Qdrant 或 Neo4j。
