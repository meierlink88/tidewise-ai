# Phase B relation and physical constraint candidate Review（R0）

## 输入与确定性输出

- 输入：[842 node/profile manifest](final-seed-candidate-artifacts/node-profile-seed-manifest.json)，SHA-256=`9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e`。
- 生成器：[generate_relation_candidates.py](relation-candidate-artifacts/generate_relation_candidates.py)，规则版本=`phase-b-semantic-v2-second-review`。
- 全量候选：[relation-candidate-review.json](relation-candidate-artifacts/relation-candidate-review.json)，SHA-256=`00bda835f16bd4df84eaa6f83d63c53146ca7edb09049def84146932e672340f`。
- 算法只枚举名称自身后缀并查询 approved node 索引，复杂度为 `O(sum(name_length))`，不构造 842×842 全排列。
- `ready_for_write=false`：AI disposition 完成不等于证据契约批准、可写 manifest 冻结或 R2 授权。

| disposition | 数量 | 第二遍 Review 结论 |
| --- | ---: | --- |
| `ai_approved_candidate_draft` | 96 | `is_subcategory_of`=95、`is_component_of`=1；只表达集合从属或物理/系统组成 |
| `blocked_needs_evidence` | 51 | taxonomy boundary=47、`input_to`=3、`depends_on`=1；不得升级 |
| physical constraint blocked | 4 | 无 write-ready constraint |
| rejected | 7 | 原 5 条复合/并列误判，加 2 条产业 endpoint mismatch |

## 96 条静态语义候选

两遍 AI Review 已批准其业务 disposition：95 条 `A -> B is_subcategory_of` 仅表示 A 全部稳定语义实例属于 B；`汽车零部件 -> 汽车 is_component_of` 仅表示可识别物理/系统组成。这 96 条不得解释为投入、依赖、供应关系、瓶颈或动态事件传导。

每条草案保存 from/to 名称与 entity key、definition/boundary、方向、mechanism、condition、内部 artifact path+SHA、derivation rule、两遍 Review 记录、反例与不确定性。分层 evidence contract 已由主对话确认；当前仍不补造 source URL、不填写最终 `verified_at`，也不输出数据库可执行 manifest，因为 task 2.7 的 R1 runner、最终 manifest 冻结与 data R2 尚未开始。

## 51 条 blocked 与 7 条 rejected

- 47 条 `suffix-needs-boundary-proof` 保持 blocked，等待权威分类定义或技术标准证明“全部实例属于”。
- `锂 -> 锂电池`、`半导体材料 -> 半导体`、`光伏主材 -> 光伏电池组件` 保持 `input_to` blocked，等待 BOM、工艺或标准证明直接消耗。
- `半导体 -> 半导体设备` 保持 `depends_on` blocked，等待产线配置、不可替代性或资格认证证据。
- `稀土产业 -> 稀土永磁`、`铜产业 -> 铜缆高速连接` 降为 rejected：from endpoint 是产业范围，不是可被直接消耗的具体材料输出；未来只能换成具体材料节点并作为全新候选重新 Review。
- 原 5 条并列/复合名称继续 rejected。规则级仍拒绝自环、alias/synonym、旧关系类型、动态事件传导及同机制 `input_to`/`depends_on` 双记。

## Physical constraint Review

| subject | type | 第二遍处置 | 重新提议所需证据 |
| --- | --- | --- | --- |
| 半导体 | `process_yield` | blocked；subject 过宽 | 先收窄到具体工艺节点/关系，再补良率、敏感性和产能损失来源 |
| 半导体设备 | `equipment_capacity` | blocked research lead | 设备交付、安装、认证、装机与扩产数据 |
| 锂矿 | `resource_availability` | blocked research lead | 储量、品位、许可、建设周期与产量来源 |
| 电池 | `material_purity` | blocked；subject 过宽 | 先收窄到具体材料节点/关系，再补材料规格、失效机理与认证来源 |

价格、政策、情绪、市场表现和动态事件均未作为 physical constraint；4 条全部不进入可写 manifest。

## R1 remediation 与下一门禁

- relation-only CLI 在配置与数据库初始化前对任何非空 `physical_constraints` 明确 fail-closed；constraint repository/dry-run 尚未实现前不得静默忽略。
- JSON loader 只接受单个文档并验证 EOF，拒绝尾随第二个 JSON 值。
- migration 017 为两个 nullable constraint subject FK 增加 partial 查询索引，但未增加未经确认的 semantic unique。
- [证据契约 amendment](phase-b-evidence-contract-amendment-review.md) 已由主对话确认；当前只允许准备 task 2.6 的 schema R2 Review package。
- 本 candidate checkpoint 未 apply migration 17，未连接或写入 PostgreSQL/Neo4j；后续只准备了独立 schema R2 Review package，data R2 仍未开始。
