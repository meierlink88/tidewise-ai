# Phase B evidence contract amendment Review（R0）

## 已确认的证据契约

现有 delta spec 对四类 relation 一律要求 source URL，但第二遍 Review 确认 96 条分类/组成候选的事实基础是已批准 842-node manifest、确定性规则和两遍独立 AI Review。为它们补造网页 URL 会制造重复且不真实的证据链。

推荐批准以下分层契约：

| 候选类型 | 充分证据 | 最终可写要求 | 不允许的外推 |
| --- | --- | --- | --- |
| `is_subcategory_of` / `is_component_of` | approved internal artifact path + SHA-256、derivation rule、两遍独立 AI Review 记录、反例/边界 | `verified_at` 必填；完整 Review disposition；冻结 manifest | 投入、依赖、供应、瓶颈、动态事件传导 |
| `input_to` / `depends_on` | 强外部 BOM、工艺、标准、认证、产线或不可替代性证据 | source URL、`verified_at`、condition、反例必填 | 仅凭名称、definition/boundary 或概念邻近升级 |
| physical constraint | 强外部物理、产能、资源或工艺证据；subject 足够具体 | source URL、`verified_at`、condition、反例必填 | 价格、政策、情绪、行情，或过宽 subject |

主对话已确认 checkpoint `f8b3406` 的上述 amendment。它只改变证据充分性判断，不改变四类 relation 语义、数据库字段、方向、tuple unique 或 R2 人工授权门禁；该确认只允许准备 `phase-b-relation-schema` 的 R2 Review package，不授权 migration 17 或 relation/constraint data Write。

## 已完成的双遍 disposition

- AI-approved candidate draft：96（`is_subcategory_of`=95、`is_component_of`=1）。
- blocked：51（taxonomy boundary=47、`input_to`=3、`depends_on`=1）。
- rejected：7，其中 2 条是 `稀土产业` / `铜产业` 作为直接投入端点不成立。
- physical constraint blocked：4；write-ready=0。

完整机器证据见 [relation-candidate-review.json](relation-candidate-artifacts/relation-candidate-review.json)，其 `ready_for_write=false`。

## Constraint semantic identity 延后待决项

migration 017 当前只实现 PK、exact-one-subject CHECK、subject FK 与查询索引，没有擅自加入 semantic unique。

推荐的后续 Review identity 是：`subject_kind + subject_id + constraint_type + normalized condition_note`。若未来 condition 被确认属于 identity，则再评估两个 subject-specific partial unique expression indexes（node subject / relation subject 分开，NULL condition 规范化）；若 condition 只是可变说明，则应从 identity 排除并改由更稳定的 mechanism key 表达。本 change 在该语义确认前不实施任何 constraint semantic unique，也不写 constraint data。该问题延后到出现可写 constraint 候选之前处理，不是本 checkpoint 要求用户确认的第二个事项。

## 后续顺序

1. evidence contract amendment 已由主对话确认。
2. task 2.6 只先准备 [relation schema R2 Review package](phase-b-relation-schema-authorization.md)；schema Write 与 Query 仍需明确授权。
3. schema Query 验收后，在 task 2.7 内先完成 relation-only atomic runner、snapshot/conflict validator、单事务 rollback、precommit assertions 与 dry-run/report 的 R1 技术验收。
4. 冻结最终可写 manifest 后，才可准备 relation data R2 package并请求人工授权。

本 artifact 不是 R2 package，不授权 migration、relation/constraint Write 或 Neo4j 操作。
