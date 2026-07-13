# Active Change Workflow Adoption Review

## 1. Adoption 范围与当前停点

本文件把 `origin/main@4b3df5ccb8ea837470f9bcaa8b2799d0762a742b` 交付的风险分级开发流程应用到本 change 的**未来门禁**。最新主分支已通过无冲突 merge commit `27e6e20` 安全吸收；既有 A：Schema / Data Contract Review 和 B：Alliance Candidate Review checkpoints 保持原样。

- adoption 本身属于 R0：只修改 OpenSpec tasks/review 元数据，不修改业务候选结论、源码、migration、seed 或数据库。
- 不追溯改写历史授权、历史证据或既有 checkbox；普通 checkbox 仍不等于人工授权。
- 当前业务停点仍为 task 2.3。B provisional 候选未获逐项确认前，不得进入 C，不得读取或冻结成员全集。
- 本轮 adoption Review 通过，只代表后续阶段按本文风险包执行；不代表 2.3 通过，也不授权 R1 Apply、R2 PostgreSQL 写入或 R3 Neo4j rebuild。

## 2. 剩余阶段风险映射

| 阶段 / task | 风险 | Review package 与边界 |
|---|---|---|
| 已完成 A、当前 B | R0 | 历史状态不追认或重写；B 继续使用逐项候选数据 Review package。 |
| C：economy 候选 | R0 | 仅在 2.3 通过后读取批准联盟的正式成员来源并形成 economy candidate package；不生成 seed、不写库。 |
| D：relationship 候选 | R0 | 仅在 3.5 通过后形成 `member_of`，以及独立的 `led_by` / `part_of` candidate packages；不写库。 |
| E：依赖与 overlap audit | R0 | 只读检查最新基线、共享文件、migration 序号、测试和 PostgreSQL 状态；发现冲突即回到 artifacts Review。 |
| 6.1—6.6：源码、测试、dry-run | R1 | 条件为新的 Apply 人工授权；只允许无有状态写入的源码和测试，不 apply migration/seed。 |
| 7.1、7.3、7.5、7.8 | R0 | 分层准备 Write Review、preflight、Query evidence 和验收包，本身不构成 Write。 |
| 7.2 | R2 | 命名执行层 `alliance-schema-and-data`；必须显式授权环境、操作、范围、恢复证据与断言。 |
| 7.4 | R2 | 命名执行层 `economy-data`；必须在 alliance Query 验收后才可执行。 |
| 7.6 | R2 | 命名执行层 `member-of`；必须在 economy Query 验收后才可执行。 |
| 7.7 | R2 | `led-by` 与 `part-of` 若推进，各自是独立命名执行层，不被 `member-of` 授权覆盖。 |
| 8.1 | R0 | 准备 Neo4j 独立 Review package，不连接或重建图。 |
| 8.2 | R3 | Neo4j rebuild 为独立显式授权，禁止并入 R2 条件包或跨层批量执行。 |
| 生产、不可逆 cleanup | R3 | 无论未来落在哪个 task，均须独立授权与恢复/灾备证据；当前未授权。 |

## 3. 阶段 Review package 顺序

1. **B / R0 alliance candidate package（当前）**：逐项确认 68 条 CSV 候选与 10 个现有 active alliance 的最终处置，产出穷尽 approved manifest。
2. **C / R0 economy candidate package**：只在 2.3 通过后，按批准联盟读取正式成员来源、形成成员全集、差异审计和 exception/protection manifest。
3. **D / R0 relationship candidate package**：只在 3.5 通过后形成 `member_of`；`led_by`、`part_of` 各自独立 Review。
4. **R1 Apply Review package**：A—D 与 overlap audit 完成后，展示允许修改的源码/测试范围、targeted tests、受影响交付边界 full suite 和不含有状态写入的证据。
5. **R2 条件式执行包**：只在 R1 Apply diff 获人工验收后，按第 5 节逐层请求授权并执行 PostgreSQL Write → Query。
6. **R3 Neo4j package**：只在 PostgreSQL 全部验收后独立 Review → Rebuild → Query；不继承任何 R2 授权。

依赖链保持为：**联盟确认 → 补齐经济体 → 建立成员关系**。上一阶段 Review/Query 未验收，不得进入下一阶段。

## 4. B：R0 Candidate Review package

### 4.1 输入、生成规则与指纹

- CSV 输入：`表格_20260713.csv`；SHA-256 `584f990ddf3a0784d7586c0b0dc40aef7558620f8d8a0c27cb91a8b075002614`。
- 现有联盟基线：`f942d76:backend/data/entity_foundation/alliance_orgs.json`；SHA-256 `a797ed7b03a3f3acfc3e8fb885b3b19c16af8c4dc2f781efcb9d8ae2089ee37f`。
- adoption 前候选 artifact：`alliance-candidate-review.md`；SHA-256 `9536c4889a3f5fbb4676b8da7c5b1ba67d88fa7ffb1cad71825e7056b7cb83e8`。该值只锁定 adoption 前的业务候选内容；本轮新增流程元数据后文件哈希会变化。
- 确定性生成边界：CSV 1—68 进入联盟候选；69—85 只记录排除；现有 10 条必须全部进入 disposition Review；正式来源只核验 identity、名称、职责和持续存在性，不读取成员集合。

### 4.2 Counts、抽样与异常全集

- CSV 联盟候选 68 条：recommend approve 62、defer 4、merge 1、reject 1；所有 final decision 均为空。
- 现有 active alliance 10 条已全部覆盖；CSV 69—85 共 17 条已排除。
- 正式来源链接 67 条；Chip 4 有 1 个正式来源 blocker。
- 确定性 QA sample 为 CSV 行 1、10、20、30、40、50、60、68，并追加**全部**非 approve、宽 identity 边界、来源 blocker、alias/abbreviation 冲突项。抽样只用于检查草案一致性，绝不替代 68 条候选和 10 条现有数据的逐项人工决策。
- 必审冲突/异常全集：World Bank stable target merge；CSV 未列但现有的 G7/G20/OECD；BRI、PGII、Chip 4、EU-US TTC defer；Silk Road Fund reject；ISO alias 冲突；无正式 abbreviation 项；Chip 4 来源 blocker；协议机制、倡议网络与联合国下属机构的宽 identity 边界。

### 4.3 Action classification 与 fail-closed

- `create/keep/merge/defer/reject/inactivate` 都只是预期动作分类，不是 Write 指令。
- 任一 final decision 留空，或任一 source、identity、alias、summary、category、existing disposition 冲突未解决，task 2.3 均不得通过。
- 现有 active alliance 未穷尽，或 merge/inactivate 的 source/target、关系影响和预计 counts 未逐项确认，task 2.3 均不得通过。
- 通过 2.3 只允许进入 C 的 R0 economy candidate package，不授权 seed、migration、PostgreSQL 或 Neo4j 操作。

## 5. 未来 R2 条件式执行包模板

每个命名 R2 层在执行前必须提交：目标环境、精确命令/入口、范围与排除项、approved manifest 版本与 checksum、恢复选择及证据、预计 counts、Write 前断言、Write 后 Query 断言、停止条件。包含策展数据的 local PostgreSQL 不自动视为可丢弃环境；除非用户逐项批准其满足 disposable local/test 条件，否则必须提供可恢复备份证据。

未来可在**一次明确批准且逐层命名**的条件包内列出以下顺序，但不得省略任何层的 Write → Query：

1. `alliance-schema-and-data` Write → Query；
2. alliance Query 验收后，`economy-data` Write → Query；
3. economy Query 验收后，`member-of` Write → Query。

`led-by` 与 `part-of` 如推进，必须各自命名、各自 Review/Write/Query。任一层失败、checksum 漂移、counts/断言不符或发生未批准范围变化，会立即使剩余未执行授权失效。Neo4j rebuild、生产写入和不可逆 cleanup 不得进入该 R2 包，必须走独立 R3 Review。

## 6. 本轮人工门禁

请主对话只审阅本 workflow adoption diff：风险映射是否正确、Review package 是否完整、未来 R2/R3 是否保持 fail-closed。通过后仅确认未来阶段采用新流程；B 的业务入口仍是 `alliance-candidate-review.md` 与 task 2.3 的逐项 Review。
