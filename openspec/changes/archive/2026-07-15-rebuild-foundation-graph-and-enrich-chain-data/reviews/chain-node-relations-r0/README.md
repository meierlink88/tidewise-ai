# 842 个 chain_node 四类关系 R0 候选 Review 包

本包只完成候选分析与双遍 Review，不授权 PostgreSQL/Neo4j 写入，也不修改源码。当前状态是 `ready_for_human_freeze_review`，不是 `ready_for_write`。

## 冻结基线

- live PostgreSQL：Goose 18 / 19 applied rows；842 个 active chain_node。
- identity MD5：`d6b53dce56fb5ca72ec77eef816f0a4b`；profile MD5：`2876324fb6bffa41967812702c6bc038`。
- live PG 与已批准 842 节点 artifact 逐行差异为 0；未新增、细化、下穿或重定义节点。
- 当前 PG 关系：95 `is_subcategory_of`、1 `is_component_of`，端点、orphan、tuple duplicate 均为 0。

## 审查结果

- 842 个节点 × 4 种关系 = 3368 个状态，全部为 `approved_relation`、`not_applicable` 或 `insufficient_evidence`，无 pending/unreviewed。
- approved candidate manifest 共 100 条：95 分类、1 组成、3 投入、1 依赖。新增的 4 条投入/依赖候选均附官方来源 URL、时间、适用条件与反例。
- 既有 47 条分类线索因稳定细分类证据不足继续阻断；7 条反例候选维持不适用。
- 9 个批次只用于组织双遍审查；全部 842 节点完成才形成当前 Review 包，批次不是关闭边界。

## 关系方向

- `input_to`：投入方 → 使用该投入进行生产或运行的一方。
- `depends_on`：依赖方 → 被依赖的硬条件。
- 分类/组成不得计入上下游，多种关系不得由“名称相邻”“市场相关”或动态事件推断。

## 文件

- `chain-node-baseline.json`：842 身份冻结清单。
- `coverage-ledger.json`：3368 个 node×relation_type 最终状态及 9 批 hash。
- `approved-candidate-manifest.json`：等待人工冻结的 100 条候选，不可写。
- `blocked-rejected-ledger.json`：证据不足、反例与排除项。
- `ai-double-review.json`：第一遍确定性语义审查与第二遍 Serenity 卡点/替代/反例审查记录。
- `validation-report.json`：端点、唯一性、cycle、冲突、范围与 hash 断言。

## 人工 Review 边界

人工 Review 只决定是否冻结这份全量关系 manifest。批准前不得进入 task 2.4、不得写 `chain_node_relations`、不得同步 Neo4j。

## Checkpoint 验证

- artifact/schema：JSON 全部可解析；842×4 唯一覆盖、允许状态、理由、强证据字段、端点、tuple、ID、cycle、冲突与 self-hash 检查通过。
- `openspec validate rebuild-foundation-graph-and-enrich-chain-data --strict`：通过。
- explicit task-design lint：通过。
- `git diff --check`、scoped file list 与 secret scan：通过；diff 仅包含本 R0 Review 包与当前 change 的 `tasks.md`。
