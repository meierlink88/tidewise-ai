# Report 发布领域契约

## 权威与边界

HTTP 权威是 Data OpenAPI 的 `ReportPublicationRequest`。AgentOS 发布已经完成推理的结构化中文
快照；Data 不解析 Markdown、不补做研究推理，也不通过 Event、Signal、Graphiti、IndustryChain
或 ChainNode 主数据重建内容。唯一跨领域身份是 canonical `EVD` Atomic Evidence ID。

## 发布根与幂等

请求只允许：

```json
{
  "publisher_report_id": "AgentOS 的重试稳定报告身份",
  "report": {
    "generated_at": "RFC3339",
    "geopolitics": null,
    "macroeconomics": null,
    "industry_chains": []
  }
}
```

JSON 实际不存在的可选 Section 应省略，不发送空占位。`industry_chains` 必须非空。相同
`publisher_report_id` 与相同 canonical report 返回原 `report_id/published_at` 且
`replayed=true`；不同内容返回冲突。API 不接收或返回 `contract_version/content_hash`。

## 上层 Section

`geopolitics`、`macroeconomics` 相互独立可选，均含：

- `title`；
- `summary`：`conclusion/result/confidence/time_window/downward_transmission/uncertainty/evidence_ids`；
- `detail`：`affected_anchors/reasoning_steps`，两者显式为数组且可为空。

Transmission 包含 `local_key/source_conclusion/targets/transmission_logic/transmission_kind/
confidence/status`，没有 Evidence 和 per-transmission boundary。target 只能引用同一 Report 的
Section、anchor、industry chain 或 affected node。

Uncertainty 保留四个独立可空字段：`counterevidence/evidence_gap/boundary/reversal_condition`。

Anchor 包含 `local_key/name/current_state/result/conclusion_basis/validation_status/
transmission_logic/time_window/confidence/evidence_ids`。Reasoning step 包含
`input/mechanism/output/reasoning_type/confidence/evidence_ids`；数组顺序就是推理步骤顺序。

## 产业链

每项包含 `local_key/name/summary/detail`：

- summary：`conclusion/status/result/confidence/time_window/path/graph/
  counterevidence_and_gap/stop_condition/evidence_ids`；
- graph node：`local_key/name`；edge：`from_node_key/to_node_key/relation`，端点必须闭合；
- detail：非空 `affected_nodes`；每项包含独立 impact local key、graph `node_local_key`、name、
  impact、result、conclusion basis、validation status、transmission logic、window、confidence 和
  Evidence IDs。

Report-local key 只在所属 `report_id` 内有意义，不是正式图对象身份。

## code、label 与 Evidence

- AgentOS 同时发布 code 和中文 label；Data 校验已知结构映射，不生成 label。
- `conclusion_basis`: `direct_evidence/直接证据` 或 `reasoning_hypothesis/推理假设`，可空。
- `validation_status`: `pending_validation/待验证`，可空，不能代替 conclusion basis。
- 只有 direct-evidence anchor/node 必须且允许带 Evidence IDs；推理假设即使待验证也没有依据入口。
- summary 和 reasoning-step Evidence 是各自显式作用域，不能从子对象聚合。
- 发布事务先批量验证 unique EVD，再同时写 Report 与全部 links；任一 EVD 缺失则整体回滚。

读取端返回 nullable opaque `evidence_scope_token (RPE...)`。调用
`GET /reports/{report_id}/evidences?scope_token=...` 时，Data 在 report 内解析 token，并只返回按
`position` 排列的 `published_at/summary/keywords`，不暴露 EVD ID、scope path 或 link 元数据。

## 存储与分页

`reports` 是一行不可变 JSONB snapshot；`report_evidence_links` 是唯一关系表。产业链列表使用
PostgreSQL `jsonb_array_elements(... WITH ORDINALITY)` 直接分页，不先解码整份 Report。cursor
绑定 report ID 与最后 ordinality；单链详情按 report-local key 延迟读取。默认页 20，最大页 100。

Migration 000081 是空库、forward-only 切换；遇到已有 Report/link 行时以 SQLSTATE 55000 拒绝。

## 定稿规模基线

以 `investment-reasoning-report-2026-09-02-presentation-v2.md` 的实际表格为准：54 条产业链、
157 个受影响节点、43 个唯一 EVD。保留上层 summary/anchor/reasoning-step 以及产业链
summary/affected-node 的显式 Evidence 作用域后，共产生 265 条 Report-Evidence link。
测试 fixture 使用等规模、确定性内容锁定这些结构计数，不把 AgentOS 的中文报告正文复制进 Data。
