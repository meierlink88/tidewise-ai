---
status: accepted
date: 2026-08-11
supersedes_in_part: 0005-agentrun-owns-source-and-raw-artifacts.md
superseded_in_part_by:
  - 0021-simplify-atomic-evidence-semantics.md
  - 0028-rebuild-event-domain-around-atomic-evidence.md
---

# Data 拥有 Raw Evidence 与原子 Evidence 发布

## 背景

ADR-0005 将完整原始 Artifact 排他地交给 AgentRun，并禁止 Data 保存正文或接纳尚未
产生 Event 的采集材料。AgentRun 进入退役方向后，该边界会让正式原始材料和后续投研
事实失去长期领域 owner，也迫使执行系统持有 Data 应当收敛的权威数据。

## 决策

- Data Domain Service 拥有完整 `Raw Evidence`、从其清洗得到的原子 `Evidence`、自然
  身份、结构校验、不可变发布和 PostgreSQL 事务。发布成功只返回正式事实 ID，不创建
  独立发布回执或审计记录。
- 采集与清洗执行方只通过 Data 的版本化 REST API 发布，不得直写 Data PostgreSQL；
  Data 不反向调用或依赖 AgentRun/AgentOS 的代码、数据库、Artifact 或运行状态。
- Raw Evidence 与 Evidence 分两阶段发布。采集完成先发布 Raw Evidence；清洗完成后再
  为既有 Raw Evidence 一次发布完整的 `1..N` Evidence 集合。零 Evidence 只表示两阶段
  之间的等待状态，不是正式清洗终态。
- Evidence 是原子事实。`1:1` 表示 Raw Evidence 未拆分，`1:N` 表示每条 Evidence 都由
  拆分产生；该血缘由 Data 根据整包基数派生并持久化。
- Evidence Group 不是领域实体；不创建 Group 表、生命周期、合并机制或 API。Atomic
  Evidence 的当前语义与身份合同由 ADR-0021 收敛。
- 新 `raw_evidences`、`evidences` 与既有 Event Publication 的 `raw_documents`、
  `event_sources` 完全隔离，不复用身份、外键或业务语义。

## 影响

ADR-0005 中“AgentRun 独占完整原始 Artifact、Data 不保存正文且不接纳未产生 Event 的
Artifact”不再适用于新的 Raw Evidence/Evidence 体系。ADR-0005 对既有 Event
Publication、轻量 `raw_documents` 与历史兼容的决定继续有效，直到独立的 Event/AgentRun
退役工作明确替代它们。本决策不迁移 AgentRun 本地旧表或数据，也不在本次交付中删除
AgentRun 运行时代码。
