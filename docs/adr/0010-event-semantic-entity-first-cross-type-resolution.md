---
status: accepted
date: 2026-08-02
supersedes_in_part: 0009-event-semantic-qdrant-projection-and-agentrun-retrieval.md
---

# Event Semantic 采用 Entity-first 跨类型解析与候选级隔离

## 背景

V2 让 Generator 同时提取 Mention、预测 Entity Type、分配角色并生成 Signal。exact 和
vector retrieval 随后以模型预测类型做硬过滤；一旦类型预测错误，PostgreSQL/Qdrant 中已经
存在的正确 Entity 也无法进入候选。V2 还把候选内容错误当成整个 JSON 输出合同错误，任一
Mention、Selection 或 Signal 失败可能删除同 Event 的其他合法事实。

## 决策

- Stage A 只提取 `raw mention + evidence_ids`，不输出 Entity Type、角色、VariableSignal 或
  Measurement。
- AgentRun 的 Qdrant exact/vector 均跨 Entity Type、按 Event 批量召回；0009 中的
  per-mention typed filter 被本决策取代。Data-owned projector、AgentRun Eino Embedder、
  PostgreSQL 权威性和批量查询所有权继续沿用 0009。
- Selector 只能从当前 Mention 的正式候选中选择 `entity_id | no_match`，并按照候选实际携带
  的 Entity Type Definition 分配 `entity_role`。Entity Type 来自候选，最终由 Data PostgreSQL
  复核。
- Data 在原 `entity_type_definitions` 表上工程化扩展 `name_zh`、`name_en`、
  `business_definition`、`inclusion_criteria`、`exclusion_criteria` 和
  `event_link_allowed`。既有 active definitions 的首批内容通过 forward migration 一次性人工
  编写回填；不建设自动生成、Curator、管理 UI 或持续同步功能。
- Entity Resolution 后，AgentRun 按正式 Entity Type 从 pinned complete Variable Definition
  directory 确定性筛选适用定义，再由独立 Signal Stage 生成 VariableSignal 与可选自然语言
  Measurement。EventEntityLink 不要求存在 Signal。
- 可解析 JSON envelope 内的 Mention、Selection、Signal、Review item 按候选隔离；一个候选
  错误不能撤销其他合法事实。只有 JSON envelope 一次 repair 后仍不可解析、Context/transport
  或 unknown-outcome 等 Execution 级错误才能终止整个 Event。
- Mention 可以出现在 Event title/summary，或任一所引 Evidence title/excerpt。只在 Event 中
  出现时必须保留 primary supporting Evidence lineage；Evidence ID 仍必须属于该 Event。
- AgentRun 持久化有界 per-stage mention/candidates/selection/variable/violation/isolation audit，
  供同批约 100 Event 验收和错误归因。

## 影响

Agent Version 升级为 `event-semantic-enricher.v3`，Context manifest 升级为
`event-semantic-context-manifest.v3`。V2 Submission 与历史 DirectImpact 数据继续保留，但新
V3 运行不读取 Direct Target、Transmission Rule 或 EntityRelation 传导，不生成 DirectImpact、
Theme、Reason Tree 或投资判断。Qdrant collection、embedding provider 和 PG→Qdrant 一次性
projector 的所有权不变。
