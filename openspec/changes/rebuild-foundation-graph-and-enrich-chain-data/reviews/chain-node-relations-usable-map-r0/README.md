# 842-node usable-map additive 关系 R0 Review 包

本包只完成 R0 数据分析、双遍审查和 Proposal artifacts 修订。状态为 `ready_for_independent_r0_review`；不授权源码修改、PostgreSQL/Neo4j 写入、R1 contract、R2、R3 或 Package 3。

## 冻结输入与保护边界

- 节点基线固定为当前 842 个 active chain_node；identity MD5 `d6b53dce56fb5ca72ec77eef816f0a4b`，profile MD5 `2876324fb6bffa41967812702c6bc038`。
- 已验收 100 条关系逐行保留：95 `is_subcategory_of`、1 `is_component_of`、3 `input_to`、1 `depends_on`；既有 tuple/content diff 均为 0。
- 不新增、细化或下穿节点，不修改 schema、profile、external identifier 或 directness 维度。
- 旧 [4-edge Neo4j sync R3 包](../all-chain-node-relations-neo4j-sync-r3.md) 已 superseded，明确不可执行。

## 已批准分析语义

- 四类关系仍为 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`。
- `input_to` 表示 A 的产出沿可解释产业链进入 B 的生产、产品、交付或运行过程，允许跨越当前缺失或未建模的中间节点，方向为上游 A → 下游 B。
- 同一对节点可以有不同机制/类型的有向边；同一客观机制不得重复登记。
- Tier 1 usable-map 要求明确机制/方向、双遍一致和可信外部来源；同一权威产业链资料可以支持一组相关边。
- Tier 2 ai-knowledge 允许暂缺外部来源，但两遍独立审查必须对类型、方向、机制、条件和反例完全一致，且 provenance 明确标记 stable industrial knowledge。
- 任一遍不一致、机制不清、边界过宽或存在合理反例的候选均进入 blocked/rejected ledger；名称邻近、市场相关、共同事件或宽泛行业相邻不得补边。

## R0 结果

| 指标 | 结果 |
|---|---:|
| baseline nodes / reviewed | 842 / 842 |
| existing / new / total relations | 100 / 156 / 256 |
| new by type | subcategory 22 / component 17 / input 111 / depends 6 |
| total by type | subcategory 117 / component 18 / input 114 / depends 7 |
| new by evidence tier | Tier 1 23 / Tier 2 133 |
| connected / isolated nodes | 295 / 547 |
| blocked/rejected / pending | 23 / 0 |

547 个 isolated 节点均已参与候选发现和两遍检查；没有满足语义与证据门槛的关系时保持无边，不以覆盖率强迫补边。23 条宽边界、方向冲突或合理反例候选均未进入 manifest。

新增关系 semantic SHA-256：`a32b69eacf86407cb182a8fb62e581f7fde2df9c59dbffc20bc21ae8a34e0094`。

## 文件

- `analysis-input.json`：842/100 冻结输入、文件/content/rules 指纹。
- `discovery-rules.json`：四类关系、input_to 宽化边界、tier 与拒绝规则。
- `source-groups.json`：Tier 1 权威产业链来源组及映射。
- `coverage-ledger.json`：842 个节点逐行 candidate discovery / first pass / second pass 状态。
- `ai-double-review.json`：第一遍语义审查与第二遍 Serenity 机制/方向/替代/反例审查记录。
- `additive-final-candidate-manifest.json`：只含 156 条新增候选，并通过 hash 引用、保护既有 100 条。
- `blocked-rejected-ledger.json`：宽边界、冲突、合理反例和方向错误候选。
- `validation-report.json`：端点、枚举、tuple、同机制重复、分类 cycle、跨类型冲突、证据字段、tier、双遍 disposition、图分布和 hash 断言。

## Review 后续边界

独立 Review 只决定是否冻结本 additive candidate manifest。即使通过，也必须先独立判断现有固定 manifest contract 是否需要最小 R1 更新，再分别准备 PostgreSQL additive R2 与最终 Neo4j sync R3 包；任何后续层都不得由本 R0 checkpoint 推定。
