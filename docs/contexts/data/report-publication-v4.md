# Report publication v4 数据合同

本合同固化已审核的 v7 报告结构。正式接口版本为 `report-publication/v4`；发布信封继续使用 `publisher_report_id` 和 `report`，完整机器合同见 [OpenAPI](../../../data-service/backend/api/data/v1/openapi.yaml)，合成样例见 [publication fixture](../../../data-service/backend/api/data/v1/report/testdata/normalized-publication-request.json)。实现与兼容决策见 [ADR-0061](../../adr/0061-normalize-report-assessments-and-latest-selection.md)。

## 1. 数据层级和身份

报告 → 地缘政治／宏观经济／产业链三个分区 → 推理单元 → 总结、详情。

- 推理单元：地缘和宏观按故事线组织，产业链按 Concept 组织。
- 总结：一句话结论、简洁推导逻辑、影响度、受影响对象引用、Evidence。
- 详情：地缘层可有宏观对象预测；三类单元均可有产业链详情。
- 产业链详情：链级判断、三类推理总结、真实链路图、节点判断、无方向结论空态。
- 对象身份 `source_id` 指冻结实体；报告内部定位使用 `local_key`。节点结果身份包含单元、产业链和节点结果键，不能只按 CND ID 合并。

### 不重复存储总结锚点结果

`summary.affected_refs` 直接引用本单元详情中的判断对象，显示列从其 `assessment` 读取。总结和详情中的锚点方向、条件、周期、置信度、结论因此只维护一次。报告级总结的一句话结论仍是独立的综合判断，不由下层机械投票生成。

锚点范围固定：地缘政治 → 宏观对象或产业链；宏观经济 → 产业链；产业链 → 节点。三类使用相同的引用形状，但允许的目标类型不同。

## 2. 报告与单元字段

以下对象默认所有键必需，未知键禁止；允许空数组与 null 的规则见后文。代码标签由消费方按固定字典显示，除兼容保留的 report_type 外，不重复存储可漂移的 label。

| 路径                                          | 类型                      | 含义与规则                                             |
| --------------------------------------------- | ------------------------- | ------------------------------------------------------ |
| schema_version                                | string                    | 固定 report-publication/v4                             |
| report_type                                   | {code,label}              | 沿用原报告类型                                         |
| generated_at                                  | RFC3339                   | 源报告生成时间，不能把格式修订时间伪装成新的证据时点   |
| timezone                                      | string                    | Asia/Shanghai                                          |
| analysis_window                               | {start,end}               | 本次固定分析输入窗口；不等于未来预测周期               |
| geopolitical_stories                          | Unit[]                    | 地缘故事线单元                                         |
| macroeconomic_stories                         | Unit[]                    | 宏观故事线单元                                         |
| concept_analyses                              | Unit[]                    | 产业链 Concept 单元                                    |
| observations                                  | Observation[]             | 不形成总结锚点的观察，本例保留流动性观察               |
| limitations                                   | string[]                  | 整体范围、推理与来源限制                               |
| Unit.local_key/source_id/title                | string                    | 报告内键、真实故事线或 Concept ID、展示标题            |
| Unit.summary                                  | Summary                   | 总结数据                                               |
| Unit.detail                                   | Detail                    | 详情数据                                               |
| Summary.conclusion                            | string                    | 单元级综合一句话结论                                   |
| Summary.transmission_logic                    | string                    | 简洁箭头因果逻辑；允许多条独立路径，不依赖来源层级渲染 |
| Summary.impact_assessment                     | ImpactAssessment          | 本总结卡片的影响度；不下放为每个链或节点的影响度       |
| Summary.affected_refs                         | AffectedRef[]             | 有序引用详情对象；空数组表示没有受影响对象结论         |
| Summary.evidence_ids                          | Evidence ID[]             | 总结依据                                               |
| Detail.macro_impacts                          | MacroImpact[]             | 仅地缘政治允许非空                                     |
| Detail.industry_chains                        | IndustryChainDetail[]     | 该单元下的链级判断，不自动扩充全库产业链               |
| Observation.local_key/title/text/evidence_ids | string/string/string/ID[] | 观察身份、正文和依据，不与有方向锚点混合               |

### ImpactAssessment

| 字段         | 类型                          | 规则                                                         |
| ------------ | ----------------------------- | ------------------------------------------------------------ |
| level        | high / medium / low / pending | 高影响／中影响／低影响／待评估；表示后果大小，不表示可信程度 |
| rationale    | string                        | 必须说明影响大小的推理与适用范围                             |
| evidence_ids | ID[]                          | 已评级必须非空；待评估允许为空或引用导致待评估判断的来源     |

### AffectedRef

| 字段            | 类型                                                       | 规则                                                                           |
| --------------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------ |
| target_type     | macroeconomic_story / industry_chain / industry_chain_node | 目标对象类型；macroeconomic_story 指宏观真实对象，不意味着复用宏观独立推理结论 |
| local_key       | string                                                     | 详情宏观预测、链详情或节点判断的键                                             |
| chain_local_key | string 或 null                                             | 节点目标必填链键；宏观或链目标固定 null                                        |

## 3. 三类报告共用的产业链详情

| 字段                     | 类型               | 规则                                               |
| ------------------------ | ------------------ | -------------------------------------------------- |
| local_key/source_id/name | string             | 详情键、ICH ID、真实链名                           |
| assessment               | Assessment         | 链级判断元数据；与节点独立，不由节点结果自动求平均 |
| reasoning_summary        | ReasoningSummary   | 对外只呈现推导逻辑、支持、反证三类                 |
| graph                    | Graph              | 冻结真实结构，不能把本次假设写为正式边             |
| affected_nodes           | NodeImpact[]       | 仅本次已形成判断的节点，不等于 graph.nodes 全量    |
| empty_state              | EmptyState 或 null | 无方向节点时显式填写，正常节点结果非空时为 null    |

ReasoningSummary：

- `logic`：链级综合推导逻辑，与 chain.assessment.transmission_logic 必须相同；前者是三类内容展示入口，不单独生成另一套文本。
- `support`：Claim，综合支持该链结论的判断及 Evidence；不按来源逐层列出。
- `objections`：Objections，在“反证”一个展示区内，明确区分反向事实、缓冲、证据缺口和边界。

NodeImpact：`local_key`、`source_id`（CND）、`node_local_key`（对应本链图节点）、`name`、`assessment`、`objections`。节点机制必须解释到该节点的传导，不能复制全链文字代替节点机制。

MacroImpact：`local_key`、`source_id`（MEC）、`name`、`assessment`、`objections`；没有产业链图和节点数组是合法业务差异。

Graph.nodes：`local_key`、`source_id`、`name`。Graph.edges：`from_node_local_key`、`to_node_local_key`、`relation_label`，两端必须在本图内。

## 4. Assessment：宏观对象、链级与节点级共用判断字段

| 字段               | 类型                                     | 含义与空值规则                                                                 |
| ------------------ | ---------------------------------------- | ------------------------------------------------------------------------------ |
| conclusion         | string                                   | 当前对象的推理结论，不与因果过程混用                                           |
| direction          | warming/cooling/diverging/pending        | 升温／降温／分化／不赋方向。升温不自动等于利润增加，以结论文字确定作用变量     |
| conclusion_basis   | reasoning_hypothesis/observation_only    | 推理假设／仅观察；当前样例没有新增“已证实未来结论”                             |
| validation_status  | pending_validation/insufficient_evidence | 条件判断待验证／证据不足而仅观察                                               |
| confidence         | low/medium/high 或 null                  | 对判断的可信程度；仅观察且不发布预测时为 null，本样例有方向结论均低            |
| forecast_window    | ForecastWindow                           | 保留具体预测窗口，不与分析窗口混淆                                             |
| scope              | string                                   | 对应国家、业务、项目或产品的适用范围；三类链详情均必填                         |
| conditions         | string[]                                 | 条件陈述，保留“若／仅限／且／或”语义；多条不同来源路径并不自动构成一个逻辑 AND |
| follow_up          | string[]                                 | 后续验证指标或动作；仅观察对象可由 empty_state.follow_up 提供                  |
| transmission_logic | string                                   | 当前对象的箭头因果推导                                                         |
| evidence_ids       | ID[]                                     | 当前判断的依据集合，不等于所有事件或全量覆盖审计                               |

### ForecastWindow

`kind`：relative（相对时间）、calendar（绝对时间）、stage（业务阶段）、mixed（组合窗口）、not_applicable（无方向预测）。

`description`：保留诸如“数日—3个月”“建设期”“2030—2031年及以后”的完整文字。

`start_at/end_at`：明确到可用时间点时才写 RFC3339；源文未给精确起止则为 null，不从“建设期”“至2027年”猜测年月日。当前样例全部保留 null，避免制造精确性。不得由未来窗口反写 generated_at。

本合同不再把“短期／中期”等无统一阈值的自由标签作为查询枚举，若产品需要按期限筛选，应另定明确阈值后实施。

## 5. 支持、反证与 Evidence

Claim 字段：`text`、`basis`、`evidence_ids`。

- basis=source_fact：源材料直接陈述；必须有 Evidence；不等于外部真实性已独立核验。
- basis=inference：基于来源的推导，Evidence 证明输入，不把推导提升为事实。
- basis=hypothetical_buffer：待验证的缓冲机制，可无 Evidence，必须保留条件语气。

Objections 字段：

| 字段                   | 类型                       | 规则                                                           |
| ---------------------- | -------------------------- | -------------------------------------------------------------- |
| summary                | string                     | 对当前判断的限制性总结，可综合多种限制；分类性质由下面字段明确 |
| counterevidence        | Claim[]                    | 有来源、直接限制该判断或其外推的反向事实；没有则 []            |
| counterevidence_status | identified/none_identified | 与 counterevidence 是否非空一致；没有反证不等于证实            |
| buffers                | Claim[]                    | 已有缓冲或条件性缓冲，basis 区分来源事实与假设                 |
| evidence_gaps          | string[]                   | 待补数据，不伪造 Evidence 引用，不作为负面事实                 |
| scope_limits           | string[]                   | 防止跨国家、主体、产品、时间或因果层级外推的边界               |

本样例“定制短剧仍较高报价”限制全行业统一降价外推，列反向事实；稀土部分批次获批、通道绕行列缓冲；未取得融资合同列缺口。节点条目的缺口可包含核验该节点所需的链级前提，不表示缺失数据已造成节点损失。

Evidence ID 在发布模型中保留；公开读取接口可换成不透明 scope token。支持和反证的 scope 必须各自独立，不能把同一整链 Evidence 集合强行归为每条反证依据。本地 Evidence 目录及全量 Event/Signal 覆盖审计不在发布合同内。服务端解析已有 Evidence，不重复存储采集全文。

## 6. 空态与一致性约束

- EmptyState：`code=observation_only`、`reason`、`follow_up`。
- 有 empty_state 时 affected_nodes=[]，direction=pending、basis=observation_only、validation_status=insufficient_evidence、confidence=null、forecast_window.kind=not_applicable。
- AI1 人形机器人仍保留链图与出货观察，不在 C5 总结中新增方向节点。
- 有方向判断必须有条件、验证路径及依据；已评级影响度必须有依据。
- local_key 在各自容器内唯一；summary refs 必须在本单元解析，不跨单元静默取值。
- 图节点 ID 与名称必须匹配节点判断；边端点必须存在。
- 原有不同故事线对同一节点的条件结论允许不同，禁止按 CND ID 全局覆盖。
- 条件路径不生成新的 Signal 或图谱边，Data Service 不负责合成或补写判断。

## 7. 发布、读取和兼容

- 发布仍使用 `POST /api/data/v1/report-publications`。未知键、缺失必填键、非法空值或跨字段矛盾返回 400；Evidence 不存在返回 422，整个事务回滚。同一发布身份同内容重放返回 200，不同内容返回 409；新发布返回 201。
- 总结、详情和产业链详情沿用 `/reports/{id}/analyses/{kind}` 路由。总结按数据库单元序号分页；详情返回宏观判断及产业链头信息，完整图和节点按链详情端点读取。
- 总结响应增加 `affected_anchors`，从 `affected_refs` 指向的唯一详情判断解析，不另存一份可能漂移的结果。
- `/reports/{id}/home` 返回报告类型、版本、生成时间、时区、分析窗口、观察和限制。
- 读取模型以 `evidence_scope_token` 替换每一处 `evidence_ids`；空依据返回 null。总结、影响度、判断、支持、每条反证和缓冲、观察分别拥有独立作用域。公开读取不返回原始 Evidence ID。
- 报告列表不传版本时覆盖 legacy、v3、v4，按发布时间与 ID 排序；可显式传 `legacy`、`report-publication/v3` 或 `report-publication/v4`。Cursor 绑定版本选择和时间筛选；历史隐式 legacy cursor 不能延续为新版默认全部版本。
- 旧快照、哈希和幂等身份保持原样。新格式使用新的发布身份，不覆盖已发布快照。迁移 86 只扩展 Evidence scope CHECK；不改写业务数据，应用回滚保留此兼容约束。

## 8. 验证与发布边界

审核基线覆盖 14 个总结、23 个链详情、43 个节点、1 个宏观预测及观察空态。仓库提交的合成 fixture 覆盖三类推理单元、6 个链详情、宏观预测、节点、反证、缓冲和无方向空态，不包含真实报告内容。

Data Service 负责结构、引用和 Evidence 完整性校验，不生成推理结论，不从知识图谱补写报告。Miniapp 映射和 UAT 部署是独立交付；需先部署具备 v4 读取能力的服务并适配消费者，再发布 v4 业务报告。
