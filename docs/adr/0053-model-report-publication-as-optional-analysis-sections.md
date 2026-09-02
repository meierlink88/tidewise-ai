---
status: accepted
date: 2026-09-02
issue: 369
supersedes_in_part: 0052-replace-research-theme-with-report-publications.md
---

# 将 Report 发布建模为可选分析 Section 与必选产业链集合

## 背景

`report-publication.v1` 依据首页原型把 Report 固定成地缘政治、宏观经济、产业链和未发布公司
四层，并把首页 `ReportCard` 作为重复值对象持久化。定稿报告基线说明了每个实际分析对象的
summary/detail 内容，却不证明每次推理都会产生地缘政治或宏观经济分析；产品确认当前最小
合法 Report 只要求产业链，公司能力尚无可发布基线。首页还需要展示一份 Report 的全部产业链，
固定卡片子集与领域事实及分页读取都发生冲突。

## 决策

- `report-publication.v2` 的最小闭包是一份元数据和至少一条产业链分析。地缘政治与宏观经济是
  相互独立的可选分析 Section；缺失 Section 不用空对象代替，也不要求另一上层同时存在。
- 公司分析在获得定稿基线前不进入 v2 合同，不发布 `published=false` 占位对象。未来通过新的
  合同版本加入真实公司 Section，并在该版本决定它是必选还是可选。
- 每个存在的上层 Section 显式分为 summary 与 detail；产业链集合中的每条分析同样分为
  summary 与 detail。上层 summary 保存对象级结论、向下传导和反证/边界，detail 保存锚点与
  可空推导步骤；产业链 summary 保存链级结论、状态、窗口、路径、链图及反证/缺口，detail
  保存本期受影响节点。
- Report 不持久化首页 `ReportCard`。Data 从不可变 Report 生成只读投影，Miniapp Backend
  拥有首页 DTO 和分页编排；任何投影都不得成为可独立修改的第二份报告事实。
- 产业链 summary 与 detail 保持在同一 `reports.content JSONB` 不可变快照中；Data 使用
  report-bound cursor 返回有序 summary page，单链 detail 按 key 延迟读取。分页不得先把完整
  Report 解码进应用内存后切片。
- 产业链 summary 持有链图及反证/缺口，detail 持有本期受影响节点；两者显式分离。受影响项以 Report-local node ref 指向链图节点，保存
  有序 effect（维度、方向、信号置信度）、结果、结论性质、传导逻辑、时间窗口、报告置信度与
  Evidence refs。
- Evidence refs 按当前作用域使用 `supports_claim`、`supports_reasoning`、
  `supports_transmission` 或 `direct_target` typed role。只有锚点/产业链节点的
  `direct_evidence` 结论允许使用 `direct_target` 且必须含直接 Evidence；
  `reasoning_hypothesis` 与 `pending_validation` 不得伪装成直接 Evidence。
- `reports` 与 `report_evidence_links` 两表、不可变发布、内容 hash、幂等和 restrictive Evidence
  外键保持不变。v2 通过 forward migration 做零数据切换；发现已有发布行时 fail closed，不转换
  或重写历史 Report。

## 基线约束

2026-09-02 定稿报告包含 54 条产业链和 157 个受影响节点。每条链都有 claim、链状态、链结果、
时间窗口、置信度、路径、链图、受影响节点、反证与 Gap、停止条件；已接受动态传导假设仅在
18 条链出现，因此可空。节点 effect 使用 `UP | DOWN | STABLE` 与
`HIGH | MEDIUM | LOW | UNKNOWN`；结论性质为直接证据、推理假设或待验证。

## 影响

- AgentOS 必须发布 v2 fixed package，不能上传 Markdown 或让 Data 从文案解析结构。
- Data Context、OpenAPI、fixture、Biz validation、SQL read projection 和 Evidence scope 同步升级。
- Miniapp Backend 保持上海当日报告选择，按 Report 独立、完整消费产业链 summary page，并
  通过现有 Home DTO 返回全部卡片；详情和 Evidence 继续按需读取。Frontend 增量分页属于后续
  可独立演进的性能优化，不进入本次领域合同切换。
- v1 的固定四层、持久化卡片和首页产业链子集不再是当前领域规则；历史 ADR 与 migration 仅保留
  为账本记录。
