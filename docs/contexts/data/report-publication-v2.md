# Report Publication v2 领域契约

## 权威与边界

本契约依据 `investment-reasoning-report-2026-09-02-presentation-v2.md` 定稿报告拆解，
由 Data OpenAPI 的 `ReportPublicationRequestV2` 和 Biz validation 执行。发布方必须提交
结构化 fixed package；Data 不解析 Markdown，Miniapp 卡片不是发布事实。

## 报告根

| 字段 | 约束 |
| --- | --- |
| `report_type` | 必填，发布方报告类型 |
| `title` | 必填 |
| `generation_status` | 必填，表达生成结果而非页面状态 |
| `simulation` | 必填布尔值 |
| `generated_at` | 必填 RFC3339 时间 |
| `analysis_window` | 必填 `{started_at, ended_at}`，且 `started_at < ended_at` |
| `timezone` | 必填 IANA 时区名 |
| `provenance` | 必填；包含可空上游报告 ID、冻结源 hash/commit 及必填 template 引用 |
| `statistics` | 必填非负统计；三个结构计数必须与快照闭合 |
| `geopolitics` | 可选 Section，缺失时不发布空对象 |
| `macroeconomics` | 可选 Section，不依赖 `geopolitics` 是否存在 |
| `industry_chains` | 必填非空有序集合；`display_order` 从 1 连续递增 |

v2 不含 `company`、`published_layers` 或 `report_cards`。公司层等定稿基线后
以新合同版本引入。

## 上层 Section

`geopolitics` 与 `macroeconomics` 共用同一结构：

- `key` 必须与 Section 名一致，`title` 必填。
- `summary.claim` 是一句话结论，含 Report-local `key` 与原文 `text`。
- `summary.transmissions[]` 是向下传导逻辑；每项含源 claim、一个或多个目标、
  传导逻辑、关系性质、置信度、状态与 Evidence refs。基线只有文本目标而无
  稳定站内目标时 `ref` 可缺失，不伪造引用。
- `summary.uncertainty` 分别保留反证、Evidence Gap、边界、反转条件和有序检查点；其中反证、
  边界和反转条件必填，Evidence Gap 可空。
- `detail.anchors[]` 是受影响锚点表格的领域表达；含有序 effects、结果、结论性质、
  原文推理、时间窗口、报告置信度、可选源引用与 Evidence refs。
- `detail.reasoning_steps[]` 可为空，不因 Section 存在而强制编造。
- `detail.related_chain_keys[]` 只能指向同一 Report 中存在的产业链。

## 产业链分析

每条 `industry_chains[]` 含 `key`、`display_order`、`name`、`summary` 和 `detail`。

`summary` 字段：

- `claim`：一句话结论；
- `status`、`result`、`confidence`、`time_window`：链级聚合结果；
- `path`：基线中的有序传导路径原文；
- `accepted_hypothesis_summary`：可空，基线 54 条中只有 18 条存在；
- `graph.nodes[]` 与 `graph.edges[]`：该链的结构图，节点不携带本期投资结论；
- `uncertainty.counterevidence_and_gap` 与 `uncertainty.stop_condition`：反证、缺口和停止条件；
- `evidence_refs`：支持链级 claim 的显式引用。

`detail` 字段：

- `node_impacts[]`：非空的本期受影响节点结论，`node_key` 必须指向本链 topology node，
  同一拓扑节点在一条链内最多一个 impact。

## 值对象与 Evidence 规则

- `result.code`: `warming | cooling | diverging | stable | mixed | pending`；单值 code 与中文 label
  固定映射，`mixed` 的 label 保留报告组合结果原文（例如“升温 / 局部稳定”）。
- `confidence.code`: `high | medium_high | medium | low_medium | low`，label 固定映射，
  `score` 可空或位于 `[0,1]`。
- `time_window.horizons`: `immediate | short | medium | long | future` 的非空无重复有序数组；
  `lag` 可空，`label` 保留报告展示原文。
- `effect.direction`: `up | down | stable`；`effect.confidence`: `high | medium | low | unknown`。
- `nature.code`: `direct_evidence | reasoning_hypothesis | pending_validation`。
- Evidence role 限于 `direct_target | supports_claim | supports_reasoning | supports_transmission`。
  锚点/节点 `direct_evidence` 必须含 `direct_target`；另两种 nature 的直接 Evidence 数组必须为空。

地缘政治或宏观经济 Section 一旦存在，summary 至少包含一条向下传导，detail 至少包含一个
受影响锚点；推导步骤仍为可空数组，不能因展示需要生成。Section 不存在时整体省略。

所有数组都必须显式发布为数组而非 `null`；所有 `display_order` 在所属集合内从 1
连续递增。Report-local key 使用 `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`。

## 存储与分页

`reports.content` 保存完整不可变 JSONB，`report_evidence_links` 保存对已发布
Atomic Evidence 的 restrictive 关系。`GET /reports/{report_id}/industry-chains`
按 `display_order` 在 PostgreSQL JSONB 投影中执行 report-bound cursor 分页；不得先把整份
Report 解码到 Go 内存再切片。单链 detail 仍通过 key 按需读取。
列表投影同时从该链的 `detail.node_impacts` 与 `summary.graph.nodes` 在 PostgreSQL 中生成有序
`impact_items`，供 Miniapp 卡片预览；它不是写回 JSONB 的第二份领域事实。
