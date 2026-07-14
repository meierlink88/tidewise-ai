# Active Change Workflow Adoption 与 Task Design Efficiency Review

## 1. Adoption 历史与当前停点

本 change 已在 `origin/main@4b3df5ccb8ea837470f9bcaa8b2799d0762a742b` 上采用风险分级流程，并通过 merge commit `27e6e20` 保留既有 A/B checkpoints。workflow adoption 的结构与边界已获主对话认可，不再保留独立行政门禁；它只作为 Package 1 的历史证据，不追认或扩大业务授权。

- 当前唯一业务出口是 Package 1 的最终联盟 manifest Review：68 条候选与 10 个现有 active alliance 必须逐项确认。
- Package 1 未通过前不得启动正式成员来源、冻结 economy 范围或生成关系候选。
- 本轮 task packaging remediation 属于 R0，只修改 OpenSpec 流程表达；不改变 contract、candidate recommendation/final decision、源码或数据库。

## 2. 五 Package 风险与人工确认

| Package | 风险 | 人工确认与边界 |
|---|---|---|
| 1 联盟范围与 Spec Review | R0 | 保留一个最终联盟 manifest 人工 Review；历史 adoption、A contract 与 B draft 不再各设 gate。 |
| 2 Economy 与关系候选 Review | R0 | 联盟批准后连续生成 economy/member package，最终一次审阅业务语义；`led_by`/`part_of` 未决则排除，不阻塞。 |
| 3 R1 实现与自动技术验收 | R1 | 无独立人工 gate；满足产业链依赖后自动 overlap audit + TDD + scoped verification，禁止有状态写。 |
| 4A local `master-data` | R2 | 一次人工授权统一 schema + alliance + economy，执行单一 Review → Write → Query。 |
| 4B local `relationships` | R2 | R2A Query 通过后独立人工授权 relationship Write → Query；只含已批准关系。 |
| 5 Apply-final 与 Deliver | R0/R1 收口 | 一个 Apply-final 人工 Review；通过后才允许 Sync、Archive、PR、merge、cleanup。 |

生产写入、不可逆 cleanup 或其他未命名环境仍属于独立 R3，不在本 change 的 local R2 授权内。Neo4j rebuild 已移出本 change，未来必须由独立 graph projection change 重新 Propose/Review。

## 3. Task Design Efficiency

- checkbox 只对应可交付 package 或必须独立授权的 R2 层，不对应资料读取、单个测试文件、技术实现步骤或重复 Query。
- 同一风险边界内的生成、diff、测试、dry-run、preflight、self-review 和验证组成一个 package；失败时 package fail-closed，不制造新的行政 gate。
- 候选数据仍按规则/抽样/总体断言审阅，高风险、宽边界、冲突、异常 disposition 与 final manifest 保持逐项人工确认。
- R1 以自动技术验收结束，不新增中间人工 Apply gate；真正有状态操作仍由 R2A/R2B 分别授权。
- Apply-final 是 Sync/Archive/Deliver 前唯一实现验收出口。repo-wide full test 不作为默认 checklist，只在共享规则、跨模块契约、公共基础设施或边界不清等项目规则触发时运行。

## 4. Package 1 Candidate Review 证据

### 4.1 输入、生成规则与指纹

- CSV 输入：`表格_20260713.csv`；SHA-256 `584f990ddf3a0784d7586c0b0dc40aef7558620f8d8a0c27cb91a8b075002614`。
- 现有联盟基线：`f942d76:backend/data/entity_foundation/alliance_orgs.json`；SHA-256 `a797ed7b03a3f3acfc3e8fb885b3b19c16af8c4dc2f781efcb9d8ae2089ee37f`。
- adoption 前候选 artifact：SHA-256 `9536c4889a3f5fbb4676b8da7c5b1ba67d88fa7ffb1cad71825e7056b7cb83e8`；来源整改输入 checkpoint `ac21094`：SHA-256 `c13663d90c2f195d1ec4ebad8579cceaccd8991194795ddaeac09d1248e86210`；来源整改输出 checkpoint `276e131`：SHA-256 `a956fa1d20307181df2a80c9a609e3a4f408c6ea7870f9ddb15940c068f6fc80`。
- CSV 1—68 进入联盟候选，69—85 只记录排除；现有 10 条全部进入 disposition Review；正式来源只核验 identity、名称、职责和持续存在性，不读取成员集合。

### 4.2 Counts、抽样与异常全集

- 68 条候选：recommend approve 62、defer 4、merge 1、reject 1；所有 final decision 为空。
- 现有 active alliance 10 条全部覆盖；CSV 69—85 共 17 条已排除；67 条候选具备正式来源，Chip 4 是唯一正式来源 blocker。
- 确定性 QA sample 为 CSV 行 1、10、20、30、40、50、60、68，并追加全部非 approve、宽 identity、来源 blocker、alias/abbreviation 冲突项。抽样不替代全部候选与现有 disposition 的逐项人工决策。
- 必审异常：World Bank stable target merge；CSV 未列的 G7/G20/OECD；BRI、PGII、Chip 4、EU-US TTC defer；Silk Road Fund reject；ISO alias 冲突；无正式 abbreviation；Chip 4 source blocker；协议机制、倡议网络和联合国下属机构宽 identity。G7 轮值来源执行时刷新；中国—中亚机制与 IOMed 状态已由正式来源更新；TTC 官方页 archived 且最后部长级会议为 2024。

### 4.3 来源新鲜度整改（R0）

| 对象 | 正式来源结论 | Review 结论不变 |
|---|---|---|
| G7 | 法国 2026 主席国与 Évian Summit；执行时仍随轮值刷新 | `approve` / existing `keep` |
| 中国—中亚机制 | 秘书处 2024 成立、2025 全面运行 | `approve` / `create` |
| IOMed | 2025 签署、生效、理事会授权运营并具备总部/秘书处 | `approve` / `create` |
| EU-US TTC | 官方页面 archived / no longer updated，最后部长级会议为 2024-04 | `defer`，不进当前 active manifest |

### 4.4 Fail-closed

- 任一 final decision 留空，或 source、identity、alias、summary、category、existing disposition 冲突未解决，Package 1 均不得完成。
- 现有 active alliance 未穷尽，或 merge/inactivate 的 source/target、关系影响、预计 counts 未确认，Package 1 均不得完成。
- Package 1 通过只允许进入 Package 2 的 R0 候选工作，不授权 R1/R2 或任何 Neo4j 操作。

## 5. 两个 Local R2 执行包模板

每个 R2 必须独立记录目标 local 环境、精确入口、范围/排除项、approved manifest/version/checksum、recovery evidence、预计 counts、before/after assertions 和停止条件。curated local PostgreSQL 默认需要可恢复 backup；只有用户逐层明确批准的 disposable recovery 才可替代。

1. **R2A `master-data`**：必要 schema + alliance + economy 在一个授权和单事务边界内 forward converge；Query 同时验证 alliance active set、economy exceptions 与非目标保护快照。
2. **R2B `relationships`**：只在 R2A Query 验收后授权；写 approved `member_of`，以及 Package 2 同一 Review 明确批准的可选关系；Query 验证 active tuples、端点、方向、provenance、官方集合与幂等。

任一未授权范围、checksum/counts/断言漂移或 Query 失败立即停止，未执行授权失效。两个 R2 均不包含 Neo4j、生产写入或不可逆 cleanup。

## 6. 本轮人工门禁

请主对话只审阅本 task-design packaging：5-package 边界、7 个顶层 checkbox、历史状态映射、两个 R2 和 Neo4j 移出是否正确。通过不代表 Package 1 最终联盟 manifest 已批准；业务入口仍是 `alliance-candidate-review.md` 与 Package 1.2。
