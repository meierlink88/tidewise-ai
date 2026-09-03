# Report 发布领域契约

## 权威与边界

HTTP 权威是 Data OpenAPI 的 `ReportPublicationRequest`，具体发布形状以
`data-service/backend/api/data/v1/report/testdata/investment-report-publication-request.json`
中从 AgentOS 复制的定稿 fixture 为最高验收基线。AgentOS 发布已完成推理的中文结构化
快照；Data 不解析 Markdown、不补做研究推理，也不从 Event、Signal、Graphiti 或图主数据
重建内容。唯一跨领域身份是 canonical `EVD` Atomic Evidence ID。

## 发布根与幂等

请求严格为 `{publisher_report_id, report}`。Report 根必须包含：

- `report_type={code: investment_reasoning, label: 投研推理报告}`；
- RFC3339 `generated_at`与 `timezone=Asia/Shanghai`；
- 相互独立可选的 `geopolitics` 与 `macroeconomics`；
- 必须非空的 `industry_chains`。

可选上层分析不存在时省略对应字段，不发送空占位。相同 `publisher_report_id` 与相同
canonical report 返回原 `report_id/published_at` 且 `replayed=true`；内容不同则冲突。

## 地缘政治与宏观经济

两个上层对象都是扁平发布快照，包含
`local_key/title/conclusion/result/time_window/confidence/affected_anchors/reasoning_steps/
uncertainty/evidence_refs/downward_transmission`。

- Anchor 包含 `local_key/name/current_state/result/conclusion_basis/validation_status/reasoning/
  time_window/confidence/evidence_refs`。
- Reasoning step 包含 `local_key/input/mechanism/output/confidence/evidence_refs`，顺序即报告顺序。
- Uncertainty 包含四个可空字段：
  `counterevidence/evidence_gap/boundary/reversal_condition`。
- 地缘传导按 `to_macroeconomics` 和 `to_industry_chains` 分组；宏观传导按
  `to_industry_chains` 分组。每组含 `summary/paths`。
- path 包含 `local_key/source_conclusion/targets/transmission_logic/transmission_kind/
  confidence/status`。target 使用 `target_type/target_local_key/target_name/result`，必须在
  同一 Report 中闭合。

## 产业链

每条产业链也是扁平快照，包含
`local_key/name/conclusion/result/time_window/confidence/path_summary/
accepted_hypothesis_summary/nodes/edges/uncertainty/evidence_refs`。

- `path_summary` 和 `accepted_hypothesis_summary` 可空，不得由 Data 或 BFF 补写。
- node 包含 `local_key/name/impact/result/conclusion_basis/validation_status/reasoning/
  time_window/confidence/evidence_refs`；节点 `local_key` 同时是链图端点。
- edge 包含 `from_node_local_key/to_node_local_key/relation_label`，端点必须闭合。
- uncertainty 包含可空 `counterevidence_and_gap/stop_condition`。

Report-local key 只在所属 `report_id` 内有意义，不是正式图对象身份。

## code、label 与 Evidence

- AgentOS 按固定目录同时发布 code 和中文 label；Data 严格校验配对，不生成 label。
- confidence 只有 `code/label`，没有 score。
- `conclusion_basis` 与 `validation_status` 是独立维度。直接证据必须是
  `direct_evidence + confirmed`；推理假设或无方向结论必须是 `pending_validation`。
- 每个 `evidence_refs` 元素包含 `evidence_id` 与有序 role `{code,label}`。role 只允许
  `direct_support/直接依据`、`reasoning_support/推导依据`、`summary_support/核心依据`。
- 直接证据 anchor/node 必须且只能使用 direct support；推理假设 anchor/node 不能携带
  Evidence。层结论、推理步骤与链结论分别使用自己的显式作用域，不从子对象聚合。
- 发布事务先批量校验 unique EVD，再同时写 Report 和 links；任一 EVD 缺失整体回滚。

读取端只获得 report-bound opaque `RPE` scope token。Evidence 查询只返回按 link position
排序的 `published_at/summary/keywords`，不暴露 EVD ID、scope path 或 role 元数据。

## 存储与分页

`reports.report` 是一行不可变 JSONB snapshot；`report_evidence_links` 是唯一跨领域关系表。
产业链列表使用 PostgreSQL `jsonb_array_elements(... WITH ORDINALITY)` 分页，cursor 绑定
report ID 与最后 ordinality，单链详情按 local key 延迟读取。默认页 20，最大页 100。

Migration 000081 是空库 forward-only 切换；遇到已有 Report/link 行以 SQLSTATE 55000 拒绝。

54 链、157 节点、43 个 unique EVD 和 265 条 link 是独立的分页/容量回归基线；
不替代 AgentOS 定稿 fixture 的精确字段契约。
