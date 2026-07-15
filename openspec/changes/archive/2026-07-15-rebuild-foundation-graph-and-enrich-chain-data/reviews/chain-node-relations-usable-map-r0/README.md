# 842-node usable-map additive 关系 R0 整改 Review 包

本包整改被独立验收拒绝的 checkpoint `4b7b80cb639f9991cdb23c3a50007563b1aba4a3`。状态为 `ready_for_independent_r0_review`；只完成 R0 数据分析与 OpenSpec artifacts，不授权源码、PostgreSQL/Neo4j 写入、R1/R2/R3 或 Package 3。旧 156 条新增集合及旧 [4-edge Neo4j sync R3 包](../all-chain-node-relations-neo4j-sync-r3.md) 均已 superseded，不得执行。

## 冻结输入与保护边界

- 节点基线固定为 842 个 active chain_node；identity MD5 `d6b53dce56fb5ca72ec77eef816f0a4b`，profile MD5 `2876324fb6bffa41967812702c6bc038`。
- 已验收 100 条关系逐行保留：95 `is_subcategory_of`、1 `is_component_of`、3 `input_to`、1 `depends_on`；文件 SHA-256 仍为 `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268`，tuple/content diff 均为 0。
- 重审输入仍是被拒 checkpoint 的 156 条候选，candidate set 指纹由 `analysis-input.json` 冻结；排除既有 100 只排除重复 tuple，不排除相关端点继续参与发现。
- 不新增、细化或下穿节点，不修改 schema、profile、external identifier、directness 或节点 identity。

## 整改后的语义与双遍门槛

- `input_to` 只接受 A 的实际产出被消耗、转化、嵌入，或作为明确服务输出沿可解释路径进入 B。每条获批边都记录 `actual_output`、`explainable_path`、具体消费/转化/嵌入/服务机制、成立条件和真实替代路线。
- 设备、工具、软件能力或基础设施 A 仅因“使 B 能生产/运行”不能记为 `input_to`。达到限定语境硬前提的候选才可按 B → A 的 `depends_on` 重审；不得机械改型。
- `is_subcategory_of` 需源节点全部稳定实例属于目标；`is_component_of` 需目标定义边界内全部实例包含该组件。完全不含该组件的合理架构反例会阻断关系。
- 第一遍为独立语义审查；第二遍为不可见第一遍结论的 Serenity/evidence/boundary 审查。两遍分别保存理由、类型、方向、机制、路径、条件、反例、来源蕴含和 disposition；任一不一致即 blocked/rejected。
- Tier 1 来源必须实际蕴含当前 relation type、方向与机制；产业相关性不能替代 source-to-edge entailment。Tier 2 仅在两遍对稳定产业知识完全一致且无合理争议时进入 manifest。
- 不追求关系数量或连通率；842 coverage 仅表示每个节点已参加候选发现与两遍检查。

## R0 整改结果

| 指标 | 结果 |
|---|---:|
| baseline nodes / reviewed | 842 / 842 |
| original candidates re-reviewed | 156 |
| existing / new / total relations | 100 / 112 / 212 |
| new by type | subcategory 13 / component 2 / input 90 / depends 7 |
| total by type | subcategory 108 / component 3 / input 93 / depends 8 |
| new by evidence tier | Tier 1 16 / Tier 2 96 |
| original tuple kept / retyped+approved / blocked | 109 / 3 / 44 |
| connected / isolated nodes | 263 / 579 |
| initial discovery blocked + remediation blocked | 23 + 44 = 67 |
| pending / unreviewed | 0 / 0 |

三条一致改型为：`集成电路制造 depends_on 光刻机`、`集成电路制造 depends_on 半导体设备`、`果蔬加工 is_subcategory_of 食品加工`。EDA 两条因两遍分别判断为工具依赖与明确服务输出投入而不一致，保持 blocked；充电桩、光伏加工设备、农业机械、工程机械、锂电专用设备、传统制冷空调和泛电网设备等也未机械改成依赖关系。

新增关系 semantic SHA-256：`5a533399a77c430e9067bac5ff509362c8168965a198801d665c40723cee4487`。双遍 review SHA-256：`3236befde7ece0dd553a01644a70a8204f5c42e304598f6a525ff415733eecfe`。

## 文件

- `analysis-input.json`：842/100 冻结输入、被拒 156 候选指纹、规则与 reviewer 标识。
- `discovery-rules.json`：严格的四类关系、工具/设备排除、全称边界、tier 与 fail-closed 规则。
- `source-groups.json`：权威来源覆盖范围和 16 条 Tier 1 的逐边 source-to-edge entailment。
- `coverage-ledger.json`：842 个节点逐行 discovery / first pass / second pass 状态。
- `ai-double-review.json`：156 条候选的两遍独立、逐字段实质审查记录。
- `additive-final-candidate-manifest.json`：只含两遍完全一致的 112 条新增候选，并通过 hash 保护既有 100 条。
- `blocked-rejected-ledger.json`：初始发现阻断项和本轮 44 条语义、方向、全称或来源争议项。
- `validation-report.json`：端点、枚举、tuple、input path、双遍字段、Tier 1 蕴含、Tier 2 provenance、blocked 隔离、图分布和 hash 断言。

## Review 后续边界

独立 Review 只决定是否冻结本整改后的 additive manifest。即使通过，也必须先独立判断现有固定 manifest contract 是否需要最小 R1 更新，再分别准备 PostgreSQL additive R2 与最终 Neo4j sync R3 包；本 R0 checkpoint 不推定任何后续层。
