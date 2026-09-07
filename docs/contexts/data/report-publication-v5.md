# Report publication v5 数据合同

本合同固化 AgentOS PR #210 中用户确认的 v8 报告，正式发布标识为 `report-publication/v5`。归档报告保留原 `v5-draft` 标识，发布者构造发布包时显式改为正式版本，不改写定稿原件。结构真源为 Data OpenAPI 的 SignalReport，合成 fixture 为 `data-service/backend/api/data/v1/report/testdata/signal-publication-request.json`。

## 发布与所有权

沿用 `POST /api/data/v1/report-publications` 和 `{publisher_report_id, report}` 信封。Data 严格校验、持久化、读取发布者结论；不生成结论、不连接实时图谱、不按名称补造实体。Event/Signal ID 是发布者的来源快照，只有 Evidence ID 必须在 Data 现存。报告之外的 Event/Signal 全量覆盖审计不发布。

根保留 v4 五类基础字段和总结详情结构，新增必填数组 `industry_chain_analyses`、`company_analyses`。前者使用相同单元结构，source_id 为 ICH；后者直接保存公司判断（COM），无需伪造 Concept 或链归属。两者都属于产业链分析范围。

## 实体信号和判断

| 位置                                                         | 内容                                           |
| ------------------------------------------------------------ | ---------------------------------------------- |
| 单元 `detail.variable_signals`                               | 当前地缘、宏观、Concept 或产业链单元自身的信号 |
| `detail.macro_impacts[].variable_signals`                    | 各宏观影响锚点的自身信号                       |
| `detail.industry_chains[].variable_signals`                  | 各产业链自身信号                               |
| `detail.industry_chains[].affected_nodes[].variable_signals` | 各节点自身信号                                 |
| `detail.companies[].variable_signals`                        | 详情中的各公司自身信号                         |
| `company_analyses[].variable_signals`                        | 独立公司判断的自身信号                         |

信号行必填 `variable_id`、`variable_name`、`signal_id`、`signal`、`source_direction`、`adoption`、`qualification`、`event_ids`、`evidence_ids`。变量、信号身份是 canonical UUID；Event 使用 EVT，Evidence 使用 EVD。不从变量名反推身份。

人类展示为“变量｜方向｜信号”；方向取 UP/DOWN/STABLE/MIXED/UNKNOWN，分别表示上升、下降、不变、分化、未明确。`signal` 保留来源记载文本，采用限定和溯源可折叠。方向是原始信号方向，不等同于预测结论方向；无信号不能冒充 STABLE。

各单元、宏观、链、节点、公司必填 `judgment_origin` 与 `reasoning_sources`。前者为 direct 或 inferred；direct 必须有自身可采用信号，inferred 的自身信号数组必须为空。后者含有序唯一 `signal_ids`、`event_ids` 与 `upstream_refs`；信号 ID 与自身行顺序一致，信号的 Event ID 必须纳入判断来源。上游引用的 local_key 和 entity_id 必须在同一单元判断范围闭合，不能自指；独立公司判断引用范围仅自身，因此须有 Event 来源。可选 mechanism、condition 非空。

Assessment 复用 v4 字段；v5 可给出有证据、有条件、置信度与跟进项的 pending 有界判断，不能以 observation_only 代替判断。链图 scope 固定 assessed_nodes_only，节点集合必须与节点判断一一匹配，empty_state 必须为 null。不校验发布者推理是否在真实世界成立，不声称能仅凭本发布包证明全量输入覆盖。

## 读取、分页与 Evidence

沿用 `/reports/{report_id}/analyses/{kind}`、`/{analysis_key}` 及 `/industry-chains/{chain_key}`；kind 新增 industry_chain_analyses、company_analyses。总结列表不返回 variable_signals。单元详情返回自身信号、宏观和公司判断，链详情按需读取并返回链和节点信号。独立公司列表与详情采用 `{schema_version, company}`；列表省略公司 variable_signals 和 reasoning_sources，详情完整返回。公司组无链详情，返回 404。

所有 Evidence 数组沿用 report-bound opaque scope token + evidence_count 投影；新增信号及公司 Evidence 同事务校验、建 link 和统计。信号 scope 使用数组序号，公司及实体 scope 使用所属容器 local_key。事件、信号、变量 ID 仍作为来源字段返回。读取不会查 Event 重算 Evidence。

默认报告列表包括 legacy、v3、v4、v5，仍按 published_at DESC、id ASC 排序；显式版本筛选和 cursor 绑定保持不变。产业链总数包含 concept_analyses 与 industry_chain_analyses 中的链，不把公司判断计为产业链。

## 验证与发布

缺失、未知键和非法 null 返回 400；语义不合法及缺失 Evidence 返回 422，且不留下报告或链接。新发布 201，同身份同 canonical 内容重放 200，不同内容 409。现有认证、20 秒发布预算及事务边界不变。

复用 reports.report JSONB、evidence_counts 及 normalized_report_evidence scope，无新 migration。旧版本不填默认信号字段，不修改历史 JSON、hash 或重放身份。先人工合并并部署 Data，再本地验收发布与回读，之后更新 UAT；本变更不直接发布定稿报告。Miniapp 需独立适配 v5，服务成功发布不代表旧客户端已支持。发布后回退应用必须保留 v5 读取能力，不执行数据库回退或改写报告。
