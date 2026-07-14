# Task 1.18a：source-side taxonomy resolution Review（R0）

## 边界与方法

- 本包逐 code 处理上一包的 13 个 `taxonomy_resolved=false` 候选；不按 canonical 名称批量推定，不修改原始 1,156 条 candidate，也不写 PostgreSQL/Neo4j。
- 证据时间：2026-07-14（Asia/Shanghai）。优先核验工作簿原始 `来源分类` 和来源平台入口；公开页面因动态加载或编码问题无法显示分类文本时，降低置信度或建议排除，绝不默认填值。
- 同花顺的 `gn/detail/code/<code>` 与 `thshy/detail/code/<code>` 是不同的官方分类入口；本次抓取器对这些页面返回 Unicode decoding error，故将 URL route 视为可复现的平台入口证据，而不伪造页面正文。
- 下表中的“单一建议”都是 `proposed_not_approved`，不是生产 taxonomy，也不构成 mapping R2 package 或 Write 授权。

## 逐 code 结论

| 来源 code | 外部名称 / 工作簿分类 | 平台观察与证据 | 单一建议 | 置信度 | 反证或不确定性 |
| --- | --- | --- | --- | --- | --- |
| eastmoney BK0456 | 家用电器 / 行业板块、概念板块 | [东方财富页](https://quote.eastmoney.com/bk/90.BK0456.html)以家用电器呈现，检索到“家用电器行业数据排行”。 | `industry_sector` | 高 | 页面同时有概念资金流导航，但行业数据入口是 code 级更直接分类证据。 |
| eastmoney BK0896 | 白酒 / 概念板块、行业板块 | [东方财富页](https://quote.eastmoney.com/bk/90.BK0896.html)以白酒呈现，检索到“白酒行业数据排行”。 | `industry_sector` | 高 | 历史页面也可能以指数/板块措辞展示；本包只判断外部 taxonomy。 |
| eastmoney BK1029 | 汽车整车 / 概念板块、行业板块 | [东方财富页](https://quote.eastmoney.com/bk/90.BK1029.html)检索到“汽车整车行业数据排行”。 | `industry_sector` | 高 | 通用行情页有概念资金流导航，不能单独构成概念分类反证。 |
| eastmoney BK1115 | 跨境电商 / 概念板块、行业板块 | [东方财富页](https://quote.eastmoney.com/bk/90.BK1115.html)仅给出同名板块和通用行业/概念导航；未发现能区分其与 BK1547 的官方分类元数据。 | `exclude_from_first_batch` | 中 | 与 BK1547 同名且同时存在；不能根据名称、code 位置或通用页面“行业数据排行”强行判行业。 |
| eastmoney BK1305 | 燃料电池 / 行业板块、概念板块 | [东方财富页](https://quote.eastmoney.com/bk/90.BK1305.html)检索结果明确出现“概念资金流”，并以燃料电池主题展示。 | `concept_sector` | 中 | 页面也含通用行业数据模块，未取得独立 taxonomy API 的文本读回，因此不标高。 |
| eastmoney BK1343 | 物业管理 / 行业板块、概念板块 | [东方财富页](https://quote.eastmoney.com/bk/90.BK1343.html)能确认 code/name，但未获得该 code 的官方行业/概念分类文本。 | `exclude_from_first_batch` | 低 | 外部看板把它列为板块不等于来源 taxonomy；本轮不以“物业管理”语义反推行业。 |
| eastmoney BK1547 | 跨境电商 / 概念板块、行业板块 | [东方财富页](https://quote.eastmoney.com/bk/90.BK1547.html)同样只展示同名板块和通用导航；与 BK1115 的同名双 code 不能证明旧新迁移或两种 taxonomy。 | `exclude_from_first_batch` | 中 | 公开新闻关联会指向该 code，但这不是来源 taxonomy 证据；必须等待来源侧明确分类。 |
| ths 300316 | 燃料电池 / 行业板块、概念板块 | 官方 [gn 入口](https://q.10jqka.com.cn/gn/detail/code/300316/)；`gn` 是概念板块入口，工作簿名称与其一致。 | `concept_sector` | 中 | 本次抓取器无法解码正文；该数值与证券代码可重叠，不能将其解释为证券。 |
| ths 300814 | 家用电器 / 行业板块、概念板块 | 官方 [gn 入口](https://q.10jqka.com.cn/gn/detail/code/300814/)；`gn` 分类入口与工作簿名称一致。 | `concept_sector` | 中 | 同上，需保留 route 证据及抓取解码限制。 |
| ths 301564 | 跨境电商 / 概念板块、行业板块 | 官方 [gn 入口](https://q.10jqka.com.cn/gn/detail/code/301564/)；`gn` 分类入口与工作簿名称一致。 | `concept_sector` | 中 | 该结论只针对同花顺此 code，不向东方财富同名 code 外推。 |
| ths 308717 | 物业管理 / 行业板块、概念板块 | 官方 [gn 入口](https://q.10jqka.com.cn/gn/detail/code/308717/)；`gn` 分类入口与工作簿名称一致。 | `concept_sector` | 中 | 页面正文抓取受编码限制；因此不是高置信度。 |
| ths 881125 | 汽车整车 / 概念板块、行业板块 | 官方 [thshy 入口](https://q.10jqka.com.cn/thshy/detail/code/881125/)；公开同花顺行业列表亦将 `881125` 列为汽车整车。 | `industry_sector` | 高 | 本次正文抓取受编码限制，但入口与公开行业列表相互印证。 |
| ths 881273 | 白酒 / 概念板块、行业板块 | 官方 [thshy 入口](https://q.10jqka.com.cn/thshy/detail/code/881273/)；公开同花顺行业列表亦将 `881273` 列为白酒。 | `industry_sector` | 高 | 同上。 |

## BK1115 与 BK1547 的专门结论

两个东方财富 code 都显示为“跨境电商”，但本次可复现的官方行情页都仅提供同名板块和共用导航；没有找到说明它们分别是行业/概念、旧/新 taxonomy 或替代关系的来源侧元数据。它们不能合并为同一 external identity，也不能以其中一个的市场语义替另一个补 taxonomy。因此本包同时建议 `exclude_from_first_batch`，保留原始候选事实，等待东方财富 taxonomy API/目录或人工来源证据。

## 重新计算的 proposed disposition

[proposed taxonomy dispositions](mapping-candidate-artifacts/proposed-taxonomy-dispositions.json) 使用确定性脚本从 c719a5c 原始候选生成，不覆盖原候选。它包含 10 个 `map` 和 3 个 `exclude_from_first_batch` 建议，且所有行均为未批准状态。

| 预期项 | 按建议执行的值 |
| --- | ---:|
| mapping rows | 1,153 |
| eastmoney / ths | 808 / 345 |
| 双来源节点 | 239 |
| external identity 冲突 / deterministic ID 冲突 | 0 / 0 |
| 预期 orphan | 0 |
| proposed mapping digest | `040e81568f90836706a14deeed75eacb819fe037c3c8a586ac9390d66e19908e` |

`entity_id + source_system + taxonomy` 存在 43 个多 code 组，这是本表模型允许的普通索引访问形态，**不是**唯一约束冲突；唯一键仍是 `(source_system, source_taxonomy_type, external_code)`。

即使主对话接受全部 13 条建议，本包仍保持 `ready_for_write=false`：必须先将已批准 disposition 重新写入 candidate 生成输入、重算完整 1,153 条（或主对话修订后的）候选、再单独 Review；之后才可能准备完全独立的 mapping R2 package。

## 结论与下一步

- 推荐：10 条 map、3 条 exclude；其中 5 条高、7 条中、1 条低置信度建议为排除。
- 主对话应逐项或整包确认/调整这 13 个 disposition。任何调整都要重新生成提案 artifact 和 digest。
- 本 R0 checkpoint 不授权 PostgreSQL/Neo4j Write、mapping R2、关系/约束、migration 17、cleanup、Sync、Archive 或 Deliver。
