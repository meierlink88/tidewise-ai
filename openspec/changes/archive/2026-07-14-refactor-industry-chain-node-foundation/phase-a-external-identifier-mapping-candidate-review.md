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
| 生成规则 | `first-batch-mapping-review-v2-user-verified-taxonomy-expansion`；对 16 个已批准 commodity→产业重命名显式使用 manifest `renamed_from` 一对一绑定，绝不复用 commodity 实体 |

- [1,169 条最终逐行候选 manifest](mapping-candidate-artifacts/external-identifier-mapping-candidates.json)：每条含 target `entity_id`/key/canonical、来源系统、外部代码/原名、单一 taxonomy、确定性 UUID 与预期动作。
- [全量机器校验与审阅清单](mapping-candidate-artifacts/external-identifier-mapping-validation.json)：输入指纹、counts、三元唯一性、确定性 ID、绑定、13 个双 taxonomy code、79 宽边界、低置信度与用户指定项。
- [生成器](mapping-candidate-artifacts/generate_mapping_candidate_review.py)只读取工作簿和已批准 manifest；输出稳定 JSON。工作簿 relationship 的绝对 `xl/...` target 已有显式路径归一化回归检查，未访问数据库。

## 总体机器校验

| 检查 | 结果 |
| --- | ---:|
| 逐行候选 | 1,169 |
| 东方财富 / 同花顺 | 818 / 351 |
| 同时两来源的 canonical 节点 | 241 |
| 单 taxonomy 来源代码 / 双 taxonomy 来源代码 | 1,143 / 13 |
| `source_system + external_code` 跨 taxonomy 重复 | 13（预期） |
| `(source_system, taxonomy, external_code)` 重复 | 0 |
| 确定性 UUID 重复 | 0 |
| manifest 目标绑定缺失 / 预期 orphan | 0 / 0 |
| 宽边界且具 mapping 的节点 | 79 |
| 用户指定 commodity→产业重命名所涉 mapping 行 | 20 |

当前 `entity_external_identifiers=0`，所以 1,169 条候选在**未来独立、fresh snapshot 的 R2 package**中预期为 `created`。`ready_for_write=true` 仅表示用户核验的候选语义完整、没有三元 identity/ID/绑定/orphan 阻断；它不是 R2 authorization，不得推定任何 mapping Write 权限。

低置信度清单为空，但这不是语义背书：输入工作簿与批准 manifest 没有“置信度”字段，本包不从名称、来源平台或 taxonomy 推断低置信度。79 条宽边界节点的完整清单在 validation JSON；其存在只提升人工审阅优先级，不改变 mapping taxonomy。

## 13 条用户核验的双 taxonomy 展开

| canonical | 来源 | code | 工作簿来源分类 | 产生的 `source_taxonomy_type` |
| --- | --- | --- | --- | --- |
| 家用电器、白酒、汽车整车、跨境电商、燃料电池、物业管理、跨境电商 | eastmoney | BK0456、BK0896、BK1029、BK1115、BK1305、BK1343、BK1547 | 行业板块、概念板块 或 概念板块、行业板块 | 每个 code 各产生 `industry_sector`、`concept_sector` |
| 燃料电池、家用电器、跨境电商、物业管理、汽车整车、白酒 | ths | 300316、300814、301564、308717、881125、881273 | 行业板块、概念板块 或 概念板块、行业板块 | 每个 code 各产生 `industry_sector`、`concept_sector` |

13 个 source/code 均保留原始 `external_name`，并依据既有 `source_system|source_taxonomy_type|external_code` 生成各自确定性 ID。网页 taxonomy 研究 checkpoint `1c63479` 只保留为历史记录，绝不用于本候选输入或处置。

## 停止条件与下一步

- 任何输入 hash、1,169/818/351/241 counts、目标绑定、三元唯一性、确定性 ID 或 taxonomy 展开状态漂移，均应重新生成 R0 package 并停止，而不是修补候选数据。
- 本候选 package 通过人工 Review 后，才可另行准备 `phase-a-external-identifier-mapping` R2 package；该 package 必须重新做 fresh snapshot、recovery evidence、preflight、Write 和 Query/assert。
- 本包不授权 mapping Write、主题、产业关系、physical constraint、migration 17、Neo4j 或后续生命周期。
