# Task 1.18：external identifier mapping candidate Review（R0）

## 范围与边界

- 本包只生成候选与机器校验，未执行 PostgreSQL/Neo4j Write、migration、seed、mapping Write、relation/constraint、R3 cleanup 或任何 UAT/prod/shared 操作。
- 它不是 `phase-a-external-identifier-mapping` R2 authorization package；未提供 Write 命令、session token、recovery authorization 或写后 Query 授权。
- 1.17 已在证据更正后获技术验收。主对话 fresh 只读读回与本包开始时的 read-only assertion 都确认：`tidewise_local`、Goose=16、active `chain_node=842`、`chain_node_profiles=842`、`entity_external_identifiers=0`、profile orphan=0。

## 输入、生成与可审阅产物

| 项目 | 值 |
| --- | --- |
| 工作簿 | `产业链节点候选-稳定节点宽口径筛选与合并.xlsx` / Sheet「原名保留明细」 |
| 工作簿 SHA-256 | `4201d4181be3a4cfe844280a6da536096af252bf2a47e10d9d8b1ecc54eb6a1b` |
| 绑定 manifest | [final-seed-candidate-manifest.json](final-seed-candidate-artifacts/final-seed-candidate-manifest.json) |
| manifest SHA-256 | `2bf669d24b2dd09929665f32a7c89727212dba634d93ebd78dc2845114fe7b13` |
| 生成规则 | `first-batch-mapping-review-v1`；对 16 个已批准 commodity→产业重命名显式使用 manifest `renamed_from` 一对一绑定，绝不复用 commodity 实体 |

- [1,156 条逐行候选](mapping-candidate-artifacts/external-identifier-mapping-candidates.json)：每条含 target `entity_id`/key/canonical、来源系统、外部代码/原名、来源分类、taxonomy 状态、候选 UUID（仅已消歧项）和预期动作。
- [全量机器校验与审阅清单](mapping-candidate-artifacts/external-identifier-mapping-validation.json)：输入指纹、counts、唯一性、绑定、阻断项、79 宽边界、低置信度与用户指定项。
- [生成器](mapping-candidate-artifacts/generate_mapping_candidate_review.py)只读取工作簿和已批准 manifest；输出稳定 JSON。工作簿 relationship 的绝对 `xl/...` target 已有显式路径归一化回归检查，未访问数据库。

## 总体机器校验

| 检查 | 结果 |
| --- | ---:|
| 逐行候选 | 1,156 |
| 东方财富 / 同花顺 | 811 / 345 |
| 同时两来源的 canonical 节点 | 241 |
| 可进入 taxonomy Review 的候选 | 1,143 |
| taxonomy 未消歧阻断项 | 13 |
| `(source_system, external_code)` 重复 | 0 |
| 已消歧 `(source_system, taxonomy, external_code)` 重复 | 0 |
| manifest 目标绑定缺失 / 预期 orphan | 0 / 0 |
| 宽边界且具 mapping 的节点 | 79 |
| 用户指定 commodity→产业重命名所涉 mapping 行 | 20 |

当前 `entity_external_identifiers=0`，所以 1,143 条已消歧候选在**未来独立、fresh snapshot 的 R2 package**中预期为 `created`。13 条未决项为 `blocked_taxonomy_review`；本包 `ready_for_write=false`，不能把 1,143 与 13 拆开写入，也不能以默认 taxonomy 或代码前缀补全。

低置信度清单为空，但这不是语义背书：输入工作簿与批准 manifest 没有“置信度”字段，本包不从名称、来源平台或 taxonomy 推断低置信度。79 条宽边界节点的完整清单在 validation JSON；其存在只提升人工审阅优先级，不改变 mapping taxonomy。

## 13 条必须逐项确认的 taxonomy Spec

| canonical | 来源 | code | 外部名称 | 工作簿来源分类 | 候选 taxonomy | 影响 | 推荐处置 |
| --- | --- | --- | --- | --- | --- | ---: | --- |
| 家用电器 | eastmoney | BK0456 | 家用电器 | 行业板块、概念板块 | industry_sector / concept_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 白酒 | eastmoney | BK0896 | 白酒 | 概念板块、行业板块 | concept_sector / industry_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 汽车整车 | eastmoney | BK1029 | 汽车整车 | 概念板块、行业板块 | concept_sector / industry_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 跨境电商 | eastmoney | BK1115 | 跨境电商 | 概念板块、行业板块 | concept_sector / industry_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 燃料电池 | eastmoney | BK1305 | 燃料电池 | 行业板块、概念板块 | industry_sector / concept_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 物业管理 | eastmoney | BK1343 | 物业管理 | 行业板块、概念板块 | industry_sector / concept_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 跨境电商 | eastmoney | BK1547 | 跨境电商 | 概念板块、行业板块 | concept_sector / industry_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 燃料电池 | ths | 300316 | 燃料电池 | 行业板块、概念板块 | industry_sector / concept_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 家用电器 | ths | 300814 | 家用电器 | 行业板块、概念板块 | industry_sector / concept_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 跨境电商 | ths | 301564 | 跨境电商 | 概念板块、行业板块 | concept_sector / industry_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 物业管理 | ths | 308717 | 物业管理 | 行业板块、概念板块 | industry_sector / concept_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 汽车整车 | ths | 881125 | 汽车整车 | 概念板块、行业板块 | concept_sector / industry_sector | 1 | 回来源侧逐代码确认；确认前阻断 |
| 白酒 | ths | 881273 | 白酒 | 概念板块、行业板块 | concept_sector / industry_sector | 1 | 回来源侧逐代码确认；确认前阻断 |

这 13 行均保留原始 `external_name`，但不生成确定性 external identifier ID；因为 ID 的 identity 包含 taxonomy。主对话应逐项确定 taxonomy，或明确从首批 mapping 范围排除；本 change 不会自行选择。

## 停止条件与下一步

- 任何输入 hash、1,156/811/345/241 counts、目标绑定、唯一性或 taxonomy 状态漂移，均应重新生成 R0 package 并停止，而不是修补候选数据。
- 仅在 13 条 taxonomy 获人工 Spec 处置、全量 candidate 再生成并 Review 通过后，才能另行准备 `phase-a-external-identifier-mapping` R2 package；该 package 必须重新做 fresh snapshot、recovery evidence、preflight、Write 和 Query/assert。
- 本包不授权 mapping Write、主题、产业关系、physical constraint、migration 17、Neo4j 或后续生命周期。
