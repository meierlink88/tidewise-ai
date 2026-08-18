---
status: accepted
date: 2026-08-18
issue: 267
supersedes: 0008-bounded-data-owned-event-semantic-resolution.md, 0010-event-semantic-entity-first-cross-type-resolution.md
supersedes_in_part: 0009-event-semantic-qdrant-projection-and-agentrun-retrieval.md, 0019-database-independent-domain-object-identities.md
---

# 退役 Data Event Semantic 与 formal Research

## 背景

Data 曾拥有 Event Semantic 的候选、租约、审核、Submission、Resolution、Variable
Definition/Signal 与 Direct Impact 持久化，并让 Research formal publication 和 Analysis
Context 依赖这些事实。这组能力不再属于 Data 的当前事实边界；继续保留会使已退役合同、
数据库和运行命令成为虚假的系统依赖。

## 决策

- Data 删除全部 Event Semantic API、scope、Biz/Data/Service wiring、运行命令、测试资产与
  当前 OpenAPI/Object ID 宣言。
- Forward-only migration `000059` 在停写和快照保护窗口中删除 Event Semantic、Variable
  Signal、Direct Impact 及相关 definition/policy/catalog 表。历史 migration 不改写。
- migration 在删除语义表前删除 Research formal/lineage 行及其引用，不把它们转换为
  `analyst_snapshot`。Research 唯一当前发布模式为 `analyst_snapshot`，其 aggregate-local
  key、display snapshot、Event/Evidence 校验和读取行为保持不变。
- Event Publication、Event Evidence Record/Link、Raw Evidence、Atomic Evidence 继续属于
  Data；Atomic Evidence 的严格单层 `semantic` JSONB 合同不变。
- 共享 Event Semantic fixture 可以作为 AgentRun 或历史合同资产保留，但不得重新成为
  Data runtime、API 或 persistence 的验收依赖。

## 发布与回滚

这是零 mixed-version 窗口的破坏性切换。发布前停止 Data 及相关上游写入，取得 PostgreSQL
恢复点，用候选 Data 镜像 check-only 并确认 `000059` 是唯一 pending migration，再执行
apply 并发布匹配应用。回滚不得运行 down migration；必须同时恢复 migration 59 前快照与
上一版应用，或发布经审阅的 forward repair。
