# 报告发布 v6：统一推导合同

对应 #498 / ADR-0066。地缘政治、宏观经济、产业链使用同一结构；单条和多条推导只由数组长度决定。当前小程序只选择 v6，旧报告保留归档。没有在线版本转换。

## 业务字段与页面

`story` 指 geopolitical_stories[]、macroeconomic_stories[]、concept_analyses[] 或 industry_chain_analyses[] 中的一项。发布完整定义以 Data OpenAPI 的 UnifiedReport 为准；读取定义为 UnifiedSummaryProjection / UnifiedDetailProjection。

| 字段                                                    | 用途                                              | 使用页面                     | 变化                                                         |
| ------------------------------------------------------- | ------------------------------------------------- | ---------------------------- | ------------------------------------------------------------ |
| story.title                                             | 故事线名称                                        | 首页、详情                   | 复用                                                         |
| story.summary.conclusion                                | 一句话结论                                        | 首页、详情头部               | 复用                                                         |
| story.summary.transmission_logic                        | 故事线推导逻辑，保留箭头与条件原文                | 首页、详情头部               | 复用                                                         |
| story.summary.judgment                                  | 结论成立或变化的业务条件                          | 首页、详情头部               | 新增，可缺省                                                 |
| story.summary.impact_assessment                         | 总体影响程度及解释                                | 首页                         | 复用                                                         |
| story.summary.affected_refs[]                           | 按报告顺序选取需要展示的影响资产                  | 首页                         | 改为 reasoning_local_key + local_key，无 target_type         |
| story.summary.evidence_ids[]                            | 当前结论引用的正式 Evidence                       | 首页证据入口                 | 复用；读出为 evidence_scope_token / evidence_count           |
| story.detail.reasonings[]                               | 一套或多套完整推导                                | 详情                         | 统一替代 macro_impacts / industry_chains                     |
| reasonings[].title                                      | 推导名称，多条时作切换项                          | 详情                         | 统一名称                                                     |
| reasonings[].assessment                                 | 推导结论、方向、范围、时间与条件                  | 详情                         | 复用完整 assessment                                          |
| reasonings[].reasoning_summary.logic                    | 蓝色“关键机制”卡片正文                            | 详情                         | 统一宏观 transmission_logic 与产业链 reasoning_summary.logic |
| reasonings[].reasoning_summary.support                  | 支持判断的原文与证据                              | 详情                         | 复用，可缺省；不得将条件冒充事实证据                         |
| reasonings[].reasoning_summary.objections               | 反证、缓冲、证据缺口和范围限制                    | 详情                         | 复用全部内容                                                 |
| reasonings[].reasoning_blocks[]                         | 由结论与数据组成的推导区块                        | 详情                         | 新增；无量化内容允许空数组                                   |
| reasoning_blocks[].title                                | 区块结论                                          | 详情                         | 新增                                                         |
| reasoning_blocks[].explanation                          | 区块解释、副标题                                  | 详情                         | 新增                                                         |
| reasoning_blocks[].relation_type                        | causal 因果节点 / comparison 并列比较             | 详情                         | 新增；只是展示关系，不证明因果                               |
| reasoning_blocks[].nodes[]                              | 不限数量的指标节点，超出三张可横滑                | 详情                         | 新增                                                         |
| nodes[].name / description                              | 指标卡名称与简要解释                              | 详情                         | 新增                                                         |
| nodes[].metrics[]                                       | 一个节点的一项或多项指标                          | 详情                         | 新增                                                         |
| metrics[].name                                          | 指标名称                                          | 详情                         | 新增                                                         |
| metrics[].value                                         | 可计算的数值，如 9.5                              | 详情                         | 新增，可缺省                                                 |
| metrics[].display_value                                 | 原文区间或定性数值，如“1万至2万元”                | 详情                         | 新增，可缺省；与 value 至少提供一项，展示优先使用该字段      |
| metrics[].unit                                          | 单位，如 %、美元/日                               | 详情                         | 新增；display_value 已含单位时不再拼接                       |
| metrics[].measure_type                                  | level 水平值、change_rate 变化率、count 数量      | 详情                         | 新增                                                         |
| metrics[].period_label                                  | 统计周期，如“近一周”                              | 指标卡底部                   | 新增；不是预计传导周期                                       |
| metrics[].as_of                                         | 数据截止时间                                      | 指标卡底部                   | 新增，可缺省                                                 |
| metrics[].value_nature                                  | observed 观测、forecast 预测、assumption 情景假设 | 详情                         | 新增，避免预测伪装成事实                                     |
| metrics[].evidence_ids[]                                | 指标的正式证据引用                                | 详情证据入口                 | 新增范围；读出沿用 token/count                               |
| reasoning_blocks[].links[]                              | 本区块节点之间明确的连接与标签                    | 详情                         | 新增，可缺省                                                 |
| reasonings[].affected_assets[]                          | 本推导的影响资产，完全使用报告内容                | 首页标签、详情指标卡和详情卡 | 统一旧节点及宏观影响对象                                     |
| affected_assets[].name                                  | 资产名称，“黄金／农业”也可整体作为一个资产        | 首页、详情                   | 复用                                                         |
| affected_assets[].assessment.direction                  | warming / cooling / diverging / pending           | 首页颜色、详情方向           | 复用，不添加增配/减配枚举                                    |
| affected_assets[].assessment.weight_delta_pp            | 配置比例变化的百分点数，例如 4、-3、0             | 详情指标卡                   | 新增；显示 ↑4%、↓3%、—；缺省不是 0，不代表收益率             |
| affected_assets[].assessment.adjustment_purpose         | 调整目的                                          | 详情指标卡底部、详情卡       | 新增，可缺省                                                 |
| affected_assets[].assessment.conclusion                 | 资产判断                                          | 详情卡                       | 复用                                                         |
| affected_assets[].assessment.transmission_logic         | 该资产的调整/影响逻辑                             | 详情卡                       | 复用                                                         |
| affected_assets[].assessment.forecast_window            | 传导时间及描述                                    | 详情卡                       | 复用；非指标统计周期                                         |
| affected_assets[].assessment.conditions                 | 改变判断的业务条件                                | 详情卡                       | 复用                                                         |
| affected_assets[].assessment.follow_up                  | 后续观察事项                                      | 报告详情数据                 | 保留                                                         |
| affected_assets[].objections                            | 资产专属反证、缓冲、缺口与限制                    | 详情卡                       | 复用，不能用整体反证替换                                     |
| reasonings[].graph                                      | 既有图谱节点、关系及范围                          | 详情                         | 可选，保留，不从正式图数据库补写                             |
| affected_assets[].node_local_key                        | 可选的本推导内图谱节点绑定                        | 详情图谱点击                 | 复用；无图谱绑定为空                                         |
| reasoning / asset 的 variable_signals                   | 对应对象的信号原文及采用判断                      | 详情                         | 复用，不混用父级信号                                         |
| reasoning / asset 的 reasoning_sources、judgment_origin | 报告判断来源                                      | 数据追溯                     | 原样保留                                                     |
| reasonings[].empty_state                                | 仅观察状态的说明和后续观察项                      | 详情                         | 原样保留                                                     |
| story.detail.companies / company_analyses               | 原报告已有公司判断                                | 原报告及独立公司读取         | 保留，不强行改造成配置组                                     |

## 读取与存储

- Data Service 继续使用 report_archive 不可变全量快照，以及 report_publications / report_summary / report_detail 分拆读取。无需新增数据库列。
- Miniapp Service ListReports 固定带 schema_version=report-publication/v6，在该集合中选当日最新或最近历史报告。旧报告不会成为新首页回退结果。
- 完整详情一次返回 reasonings[]，资产及推导切换不请求旧 industry-chains 子接口。
- 所有 Evidence 数量由发布服务计算；只对 Evidence ID 检查正式存在性。资产 local_key 只在报告内部解析。
- 旧序列化器只为归档与维护保留，保证原快照哈希和空数组/空引用不变；不在读取链路生成新报告。

## 一次性迁移与发布顺序

1. 确认环境和唯一 source report_id，保存原 JSON、分拆单元、原哈希及源文件 SHA256。禁止运行时自动选择任意 latest。
2. 用 `data-service/backend/scripts/report-unify.py` 显式输入 source、units、source-report-id、全新 publisher-report-id，输出新发布 JSON 与 manifest。脚本不连接数据库。
3. 使用 `go run ./data-service/backend/cmd/report-storage --validate-publication <文件>` 做离线合同校验。
4. 发布 Data Service、Miniapp Service 新版本，按现有认证发布接口提交新 JSON；原报告不删除、不修改。
5. 读回新报告，核对单元/推导/图谱/资产/信号/Evidence 数量、每个首页引用和发布哈希；再次提交应返回同一 report_id。
6. 发布 Miniapp，验证单/多推导、指标横滑、资产切换、证据弹层关闭和旧链接不可误展示为新详情。
7. 回滚采用上一版服务/小程序和保留的旧报告。不得直接 UPDATE 不可变快照绕过触发器。

## 本次转换范围

已对本地明确源报告 RPT13a23a27-a3ce-4fe2-bb84-8588840b195e 离线验证：18 个故事线单元，1 个宏观影响、27 条产业链，63 个首页引用，79 个原节点、50 条图谱边、7 个独立公司判断。整链引用保留对应判断，无配置幅度处不补值。实际 JSON 与 SHA 清单位于工作区外输出目录，不提交业务数据。

目标业务环境尚待确认；以上离线验证不代表已完成环境发布或业务库迁移。

## 地缘政治可选判断字段与故事线证据（#507）

仅 `geopolitical_stories` 的 reasoning/asset assessment 允许 `confidence`、`forecast_window`、`follow_up` 缺省或 null；follow_up 也可为空数组。无窗口的零值对象或 kind=not_applicable、description 为空且日期为空视为缺省。有值的窗口、置信度、观察项仍校验枚举、文本和日期，不能借可选绕过非法值校验。方向、依据、验证状态与业务条件要求保留。宏观、产业及历史 v4/v5 校验不变。

Codex 负责沿同一冻结窗口的故事线→Event→Evidence 关联查询，将去重真实 ID 写入 `summary.evidence_ids`，该故事线数组必须非空。判断、资产、指标可保留真实专属 Evidence，但不要求每项非空，也不把故事线全集复制为每资产支持。来源映射留外部审计；正文无需证据台账。Data 仍验证所有引用存在，并为 summary 投影 evidence_scope_token/evidence_count 供小程序展示；不在接收端查询图谱补造关联。

此为同一 v6 的按板块约束调整，无数据库迁移；先部署 Data 再发布新包。回滚应用后新宽松包会被旧校验拒绝，既有不可变归档保留。代码校验通过不等于 UAT 部署或发布成功。
