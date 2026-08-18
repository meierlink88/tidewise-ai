---
status: superseded
superseded_by: 0027-retire-agent-run.md
date: 2026-08-01
supersedes: 0008-bounded-data-owned-event-semantic-resolution.md
---

# Event Semantic 使用 Data-owned Qdrant 投影与 AgentRun 直接召回

## 背景

Data-owned 有界锚点解析仍把 ChainNode route、anchor、relationship 和 transmission
带入 Event Semantic，并不能通用覆盖 Entity 的简称、别名和语义变体。Event Semantic
V2 只需生成 EventEntityLink、Event-native VariableSignal 及可选的 Evidence-grounded
自然语言 Measurement；跨实体传导和投资推理属于 Theme Analyst。

## 决策

- PostgreSQL 仍是唯一正式事实源。Data Service 拥有一次性、可重复执行的
  PG→Qdrant 全量 projector，分别重建 `entity_semantic_v1` 和
  `variable_definition_semantic_v1`。Qdrant 只是可重建召回投影。
- Data Biz 只定义普通批量 embedding Port；Data adapter 使用普通
  OpenAI-compatible HTTP 客户端调用 embedding 服务并写 Qdrant。Data 代码和
  可部署二进制不得引入 Eino/eino-ext。
- AgentRun 直接查询 Qdrant，不经 Data 搜索 API，也不连 Data PostgreSQL。
  AgentRun composition root 必须优先构造 eino-ext OpenAI-compatible Embedder 并以
  Eino `embedding.Embedder` 接口注入 Qdrant adapter；AgentRun 不重复实现
  embedding HTTP wire 协议。
- 官方 Qdrant Retriever 为单 query 接口，无法满足每 Event 多 mention、每 mention
  Entity Type filter 和一次 query batch。因此 AgentRun 保留薄的 Qdrant adapter：
  每 Event 一次 `EmbedStrings` 和一次 Qdrant batch query，映射候选白名单并允许
  `no_match`。
- Data projector 与 AgentRun 查询调用同一独立 DashScope OpenAI-compatible
  embedding 服务，冻结 `text-embedding-v4`、1024 维 dense float、cosine。
  Data 不通过 AgentRun 获取 embedding，AgentRun 不通过 Data 获取 embedding，Qdrant
  不执行 embedding 模型。
- 新 Event Semantic 不读取 Direct Target、Transmission Rule 或 EntityRelation 传导，
  不产生 DirectImpact。历史表和已接受数据不删除。
- Measurement 是 VariableSignal 的可选一对多自然语言子对象，只包含
  `measurement_text + evidence_ids`。Data 保留空白、长度、数量、Evidence 和引用
  完整性校验，不做数值、单位、范围或归一化校验。

## 影响

迁移只放宽新 v2 Measurement 写入并增动态 TBox 字段，不破坏历史数据。
投影必须在 AgentRun V2 启用前完成；混合版本或 rebuild 失败期间暂停
Event Semantic。实时同步、CDC、调度和生产 Qdrant 运维不在本决策范围。
