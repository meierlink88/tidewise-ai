# Report 故事线与 Concept 发布合同

## 所有权与版本

Data Service 接收 AgentOS 已生成的结构化快照，不解析 Markdown，不生成结论、合并故事线、
推断 Concept 成员或查询图主数据补写内容。仍使用 `reports.report` JSONB 与
`report_evidence_links`，不新增页面表。只有 Evidence ID 建立跨领域外键。

`POST /api/data/v1/report-publications` 保持 `{publisher_report_id, report}` 根结构。
新报告必须声明 `schema_version: report-publication/v3`；未声明版本仍按原有扁平合同处理。
不接受混合新旧字段或未知版本。旧合同的 canonical hash、重放身份与已存快照保持不变。

本合同的可执行样例为
`data-service/backend/api/data/v1/report/testdata/story-concept-publication-request.json`。
样例是合成合同数据，包含两个地缘故事线、空宏观集合、一个 Concept 与两条产业链；
其中 source_id 是示例来源标识，不声明真实图对象存在。它不是已经接入的 AgentOS 输出。

## 根结构

- `report_type`、`generated_at`、`timezone=Asia/Shanghai` 沿用原语义。
- `analysis_window: {start,end}` 是有序 RFC3339 观察窗口，与节点预测周期 `time_window` 独立。
- `geopolitical_stories[]`、`macroeconomic_stories[]`、`concept_analyses[]` 必须显式提供数组，
  单组可以为空，但三组合计至少一个分析单元。故事线独立发布，不要求产业链板块非空。
- 数组顺序是作者发布顺序；Data 不排序或重新分组。相同来源故事线/Concept 在同组只出现一次。

## 分析单元

三个集合都使用 `AnalysisUnit`：

| 字段                         | 语义                                                         |
| ---------------------------- | ------------------------------------------------------------ |
| `local_key`                  | 报告内唯一、稳定的分析身份                                   |
| `source_id`                  | 来源故事线或 Concept 的 ID 快照，最长 200 字符；没有实时关联 |
| `title`                      | 故事线或 Concept 名称快照                                    |
| `summary.conclusion`         | 一句话综合结论，详情复用                                     |
| `summary.transmission_logic` | 卡片级传导摘要，由发布者编写                                 |
| `summary.anchor_keys[]`      | 有序引用本单元内的影响项，不复制影响判断                     |
| `summary.evidence_refs[]`    | 显式核心依据 `summary_support`                               |
| `detail.reasoning_steps[]`   | 整体逻辑的有序 input/mechanism/output 步骤及推导依据         |
| `detail.affected_anchors[]`  | 地缘的宏观/产业链锚点，宏观的产业链锚点；Concept 为空        |
| `detail.uncertainty`         | 反证、证据缺口、边界、反转条件，四个字段均可为 null          |
| `detail.industry_chains[]`   | 故事线可包含受影响产业链详情；Concept 至少一条链             |

故事线的 `anchor_keys` 仅引用自身 `detail.affected_anchors`，不能引用链内节点。
地缘锚点只允许 macro_anchor（宏观故事线）或 industry_chain；宏观锚点只允许 industry_chain。
故事线内每条链的 source_id/name 必须与自身某个 industry_chain 锚点一致。允许空链数组，
Data 不为未提供详情的锚点补图。结构图不是故事线到产业链的既有因果边。
新增合成样例 `data-service/backend/api/data/v1/report/testdata/story-chain-publication-request.json`
同时包含地缘、宏观的链图、节点落点与推理 Evidence。

Concept 的 `anchor_keys` 引用本 Concept 各链的 `affected_nodes`。
同一真实节点在不同链中有独立影响项与 local key，不能通过名称自动去重或覆盖判断。
Concept 综合结论由 AgentOS 发布，不从链结果取平均或拼接生成。

## 产业链因果分析

每条链包含 `local_key/source_id/name/conclusion/transmission_logic/reasoning_steps/graph/
affected_nodes/uncertainty/evidence_refs`。同一个分析单元内同一个 source_id 只能出现一次，
表达一个产业链对应一个详情分组。不同故事线或 Concept 可以包含相同来源产业链的独立分析快照。

- `graph.nodes` 只包含 local_key/source_id/name；`graph.edges` 沿用显式有向边合同。
  边端点须在本链闭合，不允许重复边或自环。
- `reasoning_steps` 表达本次因果推导，结构图不等于本次影响已经传到所有节点。
- `affected_nodes` 与完整拓扑节点分开；其中 `node_local_key` 引用本链图节点，
  source_id/name 必须与该节点一致。未受影响的拓扑节点不需要补写影响结论。
- 推导内容不足时允许空 steps/graph/affected_nodes 数组，并在 uncertainty 明确表达限制；
  Data 不补造缺失内容，也不承诺已有每个 Tab 的完整节点推导。

## 锚点与节点影响

影响项包含 `local_key/target_type/source_id/node_local_key/name/impact/result/conclusion_basis/
validation_status/reasoning/transmission_signal/conditions/follow_up/time_window/confidence/evidence_refs`。

`target_type` 使用 macro_anchor / industry_chain / industry_chain_node 及匹配中文 label。
故事线锚点 `node_local_key` 为 null；链内影响必须是 industry_chain_node 并引用本链节点。
`impact` 是受影响结论，`reasoning` 是为什么影响到这里；`transmission_signal` 可为 null。
`conditions` 和 `follow_up` 是显式字符串数组，允许为空，不由 Data 补写。

直接证据要求 `direct_evidence + confirmed` 且至少一个 `direct_support` 引用。
推理假设只能是 `pending_validation`，可用 `reasoning_support` 引用上游依据，不能标作
直接证据；无方向结论必须使用 pending 结果。其余 code/label 目录沿用原合同。
引用 Evidence 不会把推理假设升级为已经发生的事实。

## Evidence 与身份

所有 local key 在一份 Report 内唯一，长度不超过既有 ReportLocalKey 限制。
summary 引用只在当前单元闭合，链节点引用只在当前链闭合。source_id 仅用于来源追溯，
不建立到 Concept、IndustryChain、ChainNode 或故事线表的外键。

Event、Signal 来源和全量覆盖审计保留在 AgentOS，不发布到 Data；各推理实际引用的 Evidence ID 按作用域发布。

发布事务先批量检查所有 unique EVD，然后原子写入 Report 与全部作用域关系。
任何 Evidence 不存在则整体回滚。相同 publisher_report_id 同内容重放，异内容冲突。
事务中先匹配已存内容 hash，再对新发布执行当前校验，保证历史快照不因新锚点规则失去原样重放能力。

Evidence 路径示例：

- `geopolitical_stories/<key>/summary/evidence_refs`
- `geopolitical_stories/<key>/detail/affected_anchors/<key>/evidence_refs`
- `geopolitical_stories/<key>/detail/industry_chains/<key>/affected_nodes/<key>/evidence_refs`
- `macroeconomic_stories/<key>/detail/industry_chains/<key>/reasoning_steps/<key>/evidence_refs`
- `concept_analyses/<key>/detail/industry_chains/<key>/affected_nodes/<key>/evidence_refs`
- `concept_analyses/<key>/detail/industry_chains/<key>/reasoning_steps/<key>/evidence_refs`

Migration 000085 仅扩展 scope_type CHECK：story_summary、concept_summary、chain_reasoning_step，
保留旧 scope 类型与全部历史数据、外键、顺序约束和不可变触发器。
读取只返回 report-bound opaque Evidence scope token，不暴露 EVD ID、role 或 scope_path。

## 读取 API

- `GET /reports?schema_version=report-publication/v3`：显式列出新版本报告；摘要包含版本和观察窗口。
  省略参数只列旧合同报告，避免当前 Miniapp 在未升级时选中无法读取的新报告。
  报告列表 cursor 同时绑定 schema_version 和时间筛选。
- `GET /reports/{report_id}/analyses/{kind}?limit=&cursor=`：按地缘故事线、宏观故事线或 Concept
  分页，kind 使用根集合名；默认 20、最大 100。返回总结、显式锚点和链数量，不返回链全文。
- `GET /reports/{report_id}/analyses/{kind}/{analysis_key}`：单元总结、整体推导、锚点详情和链头列表。
  链头只包含 local_key/source_id/name/conclusion，用于后续产品 Tab。
- `GET /reports/{report_id}/analyses/{kind}/{analysis_key}/industry-chains/{chain_key}`：读取指定故事线或 Concept 下的单条链，严格限定报告、分组、单元和链。
- `GET /reports/{report_id}/concept-analyses/{concept_key}/industry-chains/{chain_key}`：按需读取一条链。
- `GET /reports/{report_id}/evidences?scope_token=`：复用现有 Evidence 读取接口。

路径均位于 `/api/data/v1`。新读取接口使用现有 `data.reports.read` 权限与 5 秒预算；
发布沿用 `data.reports.publish`、20 秒预算与 1 MiB body 限制。
SQL 在 JSONB 内分页，cursor 绑定 Report、kind 和最后序号；不会解码整份 Report 后分页。
空分析集合返回空数组，未知报告/单元/链返回现有稳定 404 分类，非法 cursor 返回 400。
旧版 layer/industry-chain 接口不负责把新版本分析转成旧形状。

## 发布顺序与回退

先部署扩展约束和支持双合同的 Data，再由后续 AgentOS/Miniapp 工作采纳新合同。
本次不发布业务报告、不升级 AgentOS、不调整 Miniapp、不部署 UAT。
旧报告不迁移、不覆盖，新报告不会自动生成旧版副本。

回退应用前停止新版本发布；若已有新版本报告，需要保留能读取新版本的服务。
不能将旧二进制无法解码新快照描述为完整回滚。扩展 CHECK 保持向前兼容，无需撤销，
不可变数据纠错继续发布新的 publisher_report_id。

故事线链详情扩展沿用 v3 字段与已有 Evidence scopes，无新 migration。报告列表的 industry_chain_count 仍只统计 Concept 板块，避免传导链重复计数。发布扩展快照前先更新 Data；回退必须保留故事线链详情读取能力。
