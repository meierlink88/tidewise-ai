---
status: superseded
date: 2026-07-31
superseded_by: 0009-event-semantic-qdrant-projection-and-agentrun-retrieval.md
---

# Event Semantic 使用 Data-owned 有界锚点解析

> 2026-08-01：本决策已被 ADR-0009 整体取代。历史表和旧 Submission 保留，
> 新 Event Semantic V2 不再调用路由、锚点、Direct Target 或 Transmission Rule。

## 背景

Event Semantic 原实现把完整 Entity/EntityRelation Catalog 固定在 Context Lease，并整体
发送给模型。响应接近服务 1 MiB 上限，ABox 随主数据增长而无界，同时 Submission 预检
依赖这份全量 snapshot。提高响应上限、Wiki/Catalog、搜索中间件或 AgentRun 直连数据库
都不能修复 ownership 与有界性问题。

## 决策

- Context Lease 只持久化轻量 `context_manifest`：Event/Evidence 身份与指纹、Lease 身份和
  Ontology/Policy/EntityType/Variable/Rule/ResolutionRoute 版本引用；不包含 Evidence 摘录、
  完整 TBox 对象或完整 ABox。Context API 按 pinned identities 组装有界响应。
- Data 通过版本化 REST API 提供 ChainNode 的 route、正式 Industry/Concept anchor 和
  reachable candidate 三段有界读取，事实来源仍是现有 PostgreSQL 正式表。
- 模型只选择上一段 Data 响应中的 ID。Data 为候选生成路径 receipt；AgentRun 不能创建
  Entity、Alias、Relation 或路径。
- Data 在 Submission 事务中重算实际选中路径。路径漂移返回
  `EVENT_SEMANTIC_CONTEXT_DRIFT` 可重试冲突，只持久化通过核验的选中 binding。
- 空候选是 unresolved；不进入 Submission 的 EntityLink/Signal，不标记 Event 数据不合规。
- AgentRun 使用有限 DAG：每个 mention 最多一次 route、一次 anchor page、一次 candidate
  page；不开放递归检索或模型控制循环。
- Industry anchor page 在已选 L1 分区内直接暴露存在正式映射、可到达 approved ChainNode 的
  后代叶级锚点，因此真实 L3 mapping 可在固定调用预算内到达。分页由 PostgreSQL 使用稳定
  keyset 与 `LIMIT page_size + 1` 完成，不在内存中加载或指纹化全量 ABox。

## 影响

迁移 `000035` 以 additive forward migration 增加 manifest 与 selected binding；不修改已
进入环境的 `000032`，也不回写历史 snapshot。Data provider 必须先部署，AgentRun consumer
随后部署；混合版本窗口保持 Event Semantic 暂停。Wire 响应继续执行 1 MiB 上限，fixture
目标低于 100 KB。

本决策不引入 Wiki、GitHub Catalog、向量库、搜索中间件、AgentRun 数据库/Neo4j 连接，
也不修改 Theme/Reason Tree。
