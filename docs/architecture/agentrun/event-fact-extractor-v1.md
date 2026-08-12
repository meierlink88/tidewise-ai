---
status: accepted
---

# AgentRun Event Fact Extractor V1

> `event-fact-extractor.v1` 的 Tag Catalog revision/hash 快照属于历史合同。当前
> `event-fact-extractor.v2` 由 Issue #211 修订为 Data PostgreSQL 唯一当前 Catalog；新执行不再
> 接收、重算或持久化 Catalog revision/hash。V1 其余历史执行语义保持不变。

## 1. Outcome

`event-fact-extractor.v1` 是独立于 Collector 的 Agent Definition。它读取明确
`collector_execution_ids` 对应的完成 Manifest 和 accepted Artifact，形成原子、
可验证的 Event Fact Candidate，并通过 Event Publication V2 发布正式 Event、
Evidence、Tag、Review 与 Collector/Extractor 血缘。

Collector Artifact Publication 完成是正常触发点。周期 Reconciler 只补偿漏触发和
中断恢复，不是业务主调度。Collector 成功不等待 Event 提取完成，Extractor 失败也不
改变 Collector 终态。

本 Spec 细化并取代 `first-real-research-report-mvp-v1.md` 6.1 中把 Event Fact 与
Semantic 合为一个第一里程碑的局部描述；它不修改总 Roadmap。

## 2. Non-goals

V1 不做：

- Mention 到 Entity Resolution；
- Event 到 Entity 或 Chain Node 的正式关联；
- Entity 到 Entity 关系新增、删除或重建；
- Event Variable Signal、Direct Node Impact、产业链传播或 Node State；
- Thesis、Company Assessment、市场方向或投资建议；
- 在 `fact_payload`、摘要或自由文本中隐藏以上语义关联；
- AgentRun 直写 Data PostgreSQL；
- Event/Evidence/Receipt 产品查询 API。

原始 Mention 可以保存在 AgentRun Candidate 或审计 Artifact，但不得解析为正式 Data
Entity ID。上述能力属于后续 `event-semantic.v1`。

## 3. Owner Map

| Concern | Owner |
|---|---|
| Collector Execution、Manifest、完整 Artifact | AgentRun |
| Artifact Ready Signal、Extraction Work Item | AgentRun |
| Extractor Agent Definition/Version/Execution | AgentRun |
| 模型输出、Candidate、校验、审核与发布重试 | AgentRun |
| Event Tag Catalog 主数据与只读 API | Data |
| Event Publication V2 合同 | Data |
| 正式 Event、Evidence、Tag Assignment、Receipt | Data |
| Entity、Chain Node 与正式语义关联 | Data，V1 不修改 |

服务只通过版本化 HTTP 合同协作，不共享数据库凭据、repository、Go domain model 或
实现包。

## 4. Reference Baseline

### 4.1 Eino

- `cloudwego/eino` commit `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`
- `cloudwego/eino-ext` commit `9137edd89e72b72735ede69db1c5ae29178a6e41`
- `cloudwego/eino-examples` commit `171220631fb7068ead50b7cd964b8c471647117d`

采用 typed `compose.Workflow`、稳定节点名、`model.BaseChatModel` 边界和严格结构化
输出。不采用 ADK、多 Agent、模型选 Tool、Eino checkpoint 或在 Eino 节点中发布
Data；调度、持久化、审核等待、HTTP 重试与恢复属于 AgentRun application/platform。

### 4.2 已验证的文件型 Event Extractor

参考任务 `AI 事件提取器` 已在
`agent-raw-ingestion-mvp/.agents/skills/extract-factual-events/` 验证以下规则：

- 每篇文档产生零到多个 Event，零 Event 必须有明确理由；
- 一个 Event 只有一个核心动作和一次状态变化；
- announced、planned、approved、effective、executing、completed、paused、
  cancelled、reported 等生命周期不得合并；
- 报道、预测、观点、传闻和计划必须保留原始认识论模态；
- `occurred_at` 只能来自正文，不得用 `published_at` 或 `collected_at` 冒充；
- Evidence Excerpt 必须是正文连续逐字片段并保持原语言；
- Event 规范标题、摘要和动作使用中文，不能增加原文没有的事实；
- `title_only` 不生成 Event；summary/snippet 只有在证据完整时才可形成 Candidate；
- 程序只召回可能重复的候选对，模型只裁决召回对；
- 生命周期、统计期或明确发生时间不一致的候选强制保持独立；
- Tag 只能来自权威 Catalog；模型不能创造 Tag；
- 模型不决定数据库状态，也不能执行 SQL。

V1 保留这些事实规则，但用 PostgreSQL Work Item、Agent Execution、ArtifactReader 和
Publication Journal 替代 TSV 处理索引、`run.lock`、目录扫描、Heartbeat 和文件
Outbox。

## 5. Exact Trigger

正常触发点是 Collector Artifact Publication 的最终 PostgreSQL 提交：

1. accepted Markdown、Dedup Index 与 Run Artifact Manifest 已发布到最终位置；
2. Manifest 是该 Collector Execution 的完成标记，`results_pending = 0`；
3. Collector Execution 即将在 PostgreSQL 进入终态；
4. 同一提交事务写入一个不依赖 Extractor 配置的 `Artifact Ready Signal`；
5. 提交成功后立即唤醒 in-process dispatcher；
6. dispatcher 为当前启用的 `event-fact-extractor.v1` 创建或复用 Work Item。

只有至少一个 accepted Artifact 才产生可消费 Signal。有 accepted Artifact 的
`succeeded` 或 `partially_succeeded` Collector Execution 均可触发；
`succeeded_no_change` 不触发。

Signal 不携带模型、Prompt、Review Policy 或 Tag Catalog Version，避免 Extractor
配置故障阻止 Collector 提交。dispatcher 在 Collector 事务之外解析当前启用的
Extractor Agent Version。

Reconciler 周期读取未消费 Signal，并核对“完成 Manifest + accepted Artifact +
目标 Agent Version”是否缺少 Work Item，再用相同稳定身份 Upsert。它不遍历 Artifact
目录，不建立第二套定时业务计划。

因此触发策略是：

```text
每次 Collector Artifact Publication 完成后立即触发
+ 周期 Reconciler 补漏
≠ 定时批量扫描后才开始提取
```

## 6. Stable Identity

V1 正常路径一个 Work Item 对应一个 Collector Execution；显式重放接口仍可接受
排序去重后的多个 `collector_execution_ids`。

```text
work_item_key = sha256(canonical_json({
  schema: "event_fact_work.v1",
  collector_execution_ids: sorted_unique_ids,
  extractor_agent_version: "event-fact-extractor.v1"
}))
```

立即 dispatcher 和 Reconciler 使用相同 Key。一个 Work Item 可以经历多个技术
Execution attempt。ArtifactReader 完成整批完整性验证后，为每个 accepted Artifact
建立一个稳定 Artifact Unit：

```text
unit_key = sha256(canonical_json({
  schema: "event_fact_artifact_unit.v1",
  work_item_key,
  artifact_id,
  content_sha256
}))
```

Unit 按稳定 Artifact ID 排序取得 `artifact_ordinal`，同一 Work Item 内依次处理。每个
真实模型运行有独立 Extractor Execution ID，并关联 Unit，快照 Agent Version、
Prompt/Schema hash、Provider/Model 配置、输入 Collector Execution IDs 与后续实际
使用的 Tag Catalog Revision/Hash。

Candidate 使用结构化身份字段进行重复召回：

- actor mentions；
- 单一 action；
- object mentions；
- lifecycle status；
- occurred time 与 precision；
- location；
- reference period。

新 Event Dedupe Key 从规范化身份字段确定性生成。若与 AgentRun 已发布 Canonical
Package 召回为同一事实，则复用既有 Dedupe Key 与不可变核心事实，只追加新 Evidence；
若生命周期、统计期或明确时间不同则强制生成独立 Event。AgentRun 不依赖 Data Event
查询 API 做该召回。

## 7. ArtifactReader

ArtifactReader 从 AgentRun Execution 已登记的 Manifest 引用开始，不扫描
`runs/` 或 `documents/`：

1. 校验 Collector Execution、Agent Definition 和允许终态；
2. 校验 Manifest Schema、Execution ID、Agent Version 与 `results_pending = 0`；
3. 校验 accepted 条目唯一、路径未逃逸 Artifact Root；
4. 校验每个 accepted 文件整体 SHA-256；
5. 严格解析 `connector_result_md.v1` Front Matter；
6. 校验 `quality_status = accepted`；
7. 按 Collector 规则重算正文 `content_sha256` 与 `document_id`；
8. 拒绝同一 Artifact 身份对应不同内容。

下游只收到包含 Artifact ID、来源快照、正文和内容哈希的 typed value；本机路径不得
进入 Workflow 输出、日志、Data 请求或安全错误。

## 8. Typed Workflow

```text
load_verified_artifacts
→ prepare_extraction_input
→ extract_fact_candidates
→ validate_atomic_facts
→ recall_possible_duplicates
→ judge_recalled_pairs
→ classify_with_tag_catalog
→ evaluate_semantic_fidelity
→ build_validated_result
```

模型负责语义理解、原子事实提取、受控 Tag Code 提议和被召回候选对的语义裁决。
程序负责 Schema、证据逐字命中、禁止字段、中文规范字段、时间格式、生命周期强制拆分、
身份、排序、Tag ID 映射、审核状态、Artifact Unit Journal 和所有副作用。

Candidate 可以保存 `actor_mentions`、`action`、`object_mentions`、`change`、
`lifecycle_status`、`time_precision`、`location_mentions`、`reference_period`、
`quantities` 和原始 Mention。Event Publication V2 只接收冻结的核心事实、Evidence、
Tag、Review 和血缘。

## 9. Review Policy

平台保留：

```text
auto_approved
manual_review
rejected
```

也可以为了兼容未来策略保留 `pending_evidence`。V1 不提供人工审核 UI、API 或消费者，
只有 `auto_approved` Candidate 可以发布。

V1 采用包含独立第二次 AI 语义审核的 `auto_only` Policy：

- Artifact 完整、严格 Schema、原子事实、逐字 Evidence、认识论模态、Tag Catalog
  和全部确定性门禁通过；
- 独立语义审核返回 `semantic_pass = true`、`conflict = false` 和非空中文理由；
- 同时满足以上条件才算“提取成功”，确定性写入 `auto_approved` 并允许发布；
- 硬门禁失败写入 `rejected`；
- 语义审核失败或存在冲突写入 `rejected`，不进入 Data，并继续处理下一个 Artifact
  Unit；`manual_review` 仅保留为未来策略枚举，V1 不产生该状态；
- 语义审核置信度作为审计事实保存，V1 不以来源数量、官方性或单独置信度阈值阻止已经
  通过布尔语义门禁的 Candidate。

单一来源、非官方来源、summary 或 snippet 本身不阻止通过；Evidence Grade 只表达
来源覆盖，不决定正式状态。`verified` 只表示 Event 与当前逐字 Evidence 一致。

## 10. Event Tag Catalog API

Data 新增版本化只读合同：

```http
GET /api/data/v1/event-tags?active=true
Authorization: Bearer <service-token>
```

调用主体需要独立只读 scope，例如 `data.event-tags.read`。响应至少包含：

```json
{
  "request_id": "data-...",
  "result": {
    "catalog_revision": "event-tags:...",
    "catalog_hash": "64-char-lowercase-sha256",
    "tags": [
      {
        "id": "uuid",
        "tag_kind": "news_category",
        "code": "technology_industry",
        "name": "科技产业",
        "is_active": true
      }
    ]
  }
}
```

Tag 按 `tag_kind, code, id` 稳定排序。AgentRun 可以条件缓存，但每个 Extractor
Execution 必须快照实际使用的 Revision 与 Hash。YAML/Prompt 只保存分类规则，不能
复制 Tag ID。

Tag API 不可用时，已验证 Candidate 保持在 `awaiting_tag_catalog`，只重试 Catalog
读取和后续分类，不重新读取模型提取结果。

本里程碑不新增 Event、Evidence 或 Receipt 查询 API。Event Publication 成功响应提供
正式身份和 Receipt；完整事务与血缘由 Data provider 集成测试在 Data 边界验证。

## 11. Artifact Unit Publication And Journal

一个 Collector Execution 的全部 accepted Artifact 是父 Extraction Batch/Work Item；
其中每个 accepted Artifact 是一个独立、耐久的 Artifact Unit。Unit 固定执行：

```text
读取与校验一个 Artifact
→ 提取 0..10 个原子 Event
→ Unit 内去重
→ 与已发布 Canonical Event 跨 Unit 去重
→ AI 语义确认
→ 立即发布
```

每个有 Event 的 Unit 只建立一个 Publication Journal：

- 只携带该 Unit 的一个 Artifact；
- 携带该 Artifact 的 Collector Execution 血缘；
- 使用稳定 `artifact_ordinal` 和基于 `unit_key` 的 `package_id`；
- 在首次 POST 前持久化精确请求字节和 SHA-256；
- 独立取得 Data Receipt。

Unit 取得 Receipt 后先更新 AgentRun Canonical Event 缓存，再允许下一个 Unit
领取；因此下一份 Artifact 可以复用既有 Dedupe Key 并只为同一 Event 追加 Evidence。
一个 Artifact 产生超过十个可发布 Event 时整 Unit 拒绝，不截断。父 Work Item
聚合全部 Unit：全成功为 `published`，成功与拒绝/阻塞混合为
`partially_published`。Data 按 Unit 逐份可见，不伪装父批次跨 Unit 原子性。

网络超时、响应丢失、`429` 或 `5xx` 只重发 Journal 中相同字节。`401/403` 阻塞配置，
`409/422` 阻塞合同或事实冲突。未知结果重试可能在 Data 生成多个 Receipt，但自然身份
一致的 Event、Evidence 和 Tag 不重复；模型调用次数不增加。

投递领取使用 Journal `status + attempt_count` 条件更新和有界 `sending` 租约。过期
worker 的成功或失败回调不能覆盖更新 attempt 的结果；Publication 重试保持当前
Unit 在 `ready_to_publish/publishing`，后续 Unit 不得越过它领取，也不重新调用模型。

## 12. State Summary

```text
Artifact Ready Signal:
pending → dispatched

Work Item:
pending → running
        → pending | awaiting_tag_catalog | retry_wait
        → ready_to_publish | publishing
        → published | partially_published | blocked | rejected | no_events

Artifact Unit:
pending → running
        → awaiting_tag_catalog | retry_wait
        → ready_to_publish → publishing → published
        → rejected | blocked | no_events

Extractor Execution:
queued → running → succeeded | succeeded_no_change | failed

Publication Journal:
prepared → sending → acknowledged | retry_wait | blocked
```

`manual_review` 状态枚举仅为未来策略保留，V1 不产生人工等待。发布恢复不占用 Eino
Runner，也不使用 Eino checkpoint。

## 13. Highest Observable Verification

Happy path 使用真实 Collector Artifact Publication、真实 AgentRun PostgreSQL、真实编译
Eino Workflow、计数 Fake ChatModel、真实 Data HTTP 与隔离 Data PostgreSQL：

1. Collector Publication 提交并写 Artifact Ready Signal；
2. 不等待定时器即可创建唯一 Work Item；
3. ArtifactReader 验证真实 Manifest 与 Markdown，并为每个 accepted Artifact 建立
   一个 Unit；
4. 每个 Unit 提取零到十个原子 Event，并对无 Event 文档记录理由；
5. 从真实 Tag Catalog API 读取并快照 Catalog；
6. 每个 Unit 完成校验后立即建立 Journal 并发布；
7. Data 返回 Receipt，AgentRun 更新 Canonical 缓存后才领取下一个 Unit；
8. Data provider 集成断言正式 Event、Evidence、Tag 和完整血缘。

未知结果测试要求：

```text
Data 已提交，但响应被丢弃
→ AgentRun 重启
→ Journal 重发相同 payload bytes
→ Fake Model 总调用次数仍为一次
→ Data 正式事实自然身份唯一
→ 允许产生新的 Receipt
→ Work Item 最终 published
```

还需覆盖漏触发由 Reconciler 补齐、重复 Signal、零 accepted Artifact、Artifact hash/
document identity 漂移、Tag API 不可用、单 Unit 超过十个 Event、Unit 失败不阻塞后续
Unit、跨 Unit 去重，以及 `401/403/409/422` 不重新调用模型。

## 14. Roadmap Dependency Recommendation

不修改 Roadmap，但实现工作应解释为：

```text
OR-22A Event Fact Extractor
→ 正式 Event / Evidence / Tag / Review / 血缘

OR-22B Event Semantic Enricher
→ Mention Resolution
→ Event ↔ Entity / Chain Node
→ Event Variable Signal / Direct Node Impact
```

OR-22B 依赖 OR-22A、Object/Link Contract、Entity Resolution API、Ontology Context
Package 和已审核产业链关系。OR-22A 不提前实现或暗藏这些依赖。

## 15. Frozen Review Decision

V1 保留 Event 提取模型之外的独立 AI 语义审核。它只判断事实表达是否被逐字证据支持、
是否遗漏关键限定条件以及是否存在语义冲突，不选择数据库状态；确定性 Review Policy
拥有最终状态映射。当前不建设人工审核入口。
