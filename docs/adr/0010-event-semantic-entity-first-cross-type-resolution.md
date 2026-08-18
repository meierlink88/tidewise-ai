---
status: accepted
date: 2026-08-02
supersedes_in_part: 0009-event-semantic-qdrant-projection-and-agentrun-retrieval.md
superseded_in_part_by: 0027-retire-agent-run.md
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
- 模型 Stage 顶层数组是严格必填 envelope；`null`、`{}`、缺字段、`null` 数组或错误类型不得被
  Go 零值解释为空结果，一次 repair 后仍非法才构成 terminal model-contract failure。
- Selector 与独立 Review 只接受正式 canonical name、name 或 aliases 提供的 identity；不通过
  删除名称后缀、字符串包含或其他手写简称规则扩展 identity。只有唯一 exact identity 被主
  Selector 拒绝时执行一次独立二次复核，vector Top-1 不自动接受。
- Role 明确区分声明来源、行动主体、自身状态主题、行动对象、直接受影响实体与背景；被通牒、
  被调查或作为措施对象的实体不得误标为 `event_subject`。
- 顶层 Stage envelope 继续严格；数组 item 独立按固定 DTO 解析，单个非法 item 只记录并隔离，
  不触发 whole-Event repair 或删除同数组合法候选。
- AgentRun retrieval 与 Data projection HTTP Adapter 保留 `context.Canceled` 和
  `context.DeadlineExceeded`，不将取消或超时包装成普通可重试 RemoteError。
- AgentRun 不维护手写简称/全称 identity guard；唯一 canonical/alias exact identity 来自正式
  投影，其他候选由 Selector 与独立 AI Reviewer 判断是否同一业务对象。
- AgentRun 对每个 Qdrant 外层 point ID 与 payload Entity ID、source identity、projection version、
  embedding model 与 content fingerprint fail closed；payload 不重复保存 `point_id`。缺少 Embedding
  Secret 时必须在进程启动配置阶段失败。
- `mention_not_entity` 只表示真正的日期、数值、状态、行为、报告或会议；ABox 缺失、TBox 未开放
  与 retrieval/model miss 必须在验收中分别归因。

## 影响

Agent Version 升级为 `event-semantic-enricher.v3`，Context manifest 升级为
`event-semantic-context-manifest.v3`。V2 Submission 与历史 DirectImpact 数据继续保留，但新
V3 运行不读取 Direct Target、Transmission Rule 或 EntityRelation 传导，不生成 DirectImpact、
Theme、Reason Tree 或投资判断。Qdrant collection、embedding provider 和 PG→Qdrant 一次性
projector 的所有权不变。
