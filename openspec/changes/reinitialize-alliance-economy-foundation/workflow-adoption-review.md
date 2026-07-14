# Active Change Workflow Adoption 与 Task Design Efficiency Review

## 1. Adoption 历史与当前停点

本 change 已在 `origin/main@4b3df5ccb8ea837470f9bcaa8b2799d0762a742b` 上采用风险分级流程，并通过 merge commit `27e6e20` 保留既有 A/B checkpoints。workflow adoption 的结构与边界已获主对话认可，不再保留独立行政门禁；它只作为 Package 1 的历史证据，不追认或扩大业务授权。

- Package 1 已于 2026-07-14 通过：45 条全部 approve、9 keep + 36 create、现有 9 keep + OECD forward inactivate；canonical checksum `4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a`。
- 当前唯一业务出口是 Package 2.1：审阅 membership model、79 条 economy target、133 条 resolved formal-active tuples、32 条 proposed inactivate 与 160 条 blocked source-conflict。
- 当前 checkpoint 仍属于 R0，只修改 OpenSpec Review artifacts；不修改源码或数据库。

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

- 当前输入：`联盟组织列表1.0.xlsx`；SHA-256 `ac0d953c0cd93596fe6bf8a70541bbe658620e75d38a9b3178980071b2cdc102`。
- 唯一读取范围：首个 sheet `联盟组织` 的 `A1:K51`；不读取其他 sheet。
- 现有联盟基线：`backend/data/entity_foundation/alliance_orgs.json`；SHA-256 `a797ed7b03a3f3acfc3e8fb885b3b19c16af8c4dc2f781efcb9d8ae2089ee37f`。
- 旧 CSV、68 条 recommendation 与网页核验仅保留为历史，不参与当前生成、抽样、异常或 manifest。
- 生成规则只映射名称、缩写、核心主导方和核心影响范围说明；分组标题、大类、子类、成员数、占比、评级及其他 sheet 均排除。

### 4.2 Counts、抽样与异常全集

- 45 条数据行、5 条分组标题；四个目标字段均 45/45 非空；名称重复 0，规范化缩写重复 0。
- 唯一 normalization：sheet row 45 `UJR<U+200C>` → `UJR`、sheet row 50 `CCAS<U+200C>` → `CCAS`；不做其他语义纠正。
- 与现有 10 条文件基线比较：9 个 `keep` identity、36 个 `create` 已批准；现有 `alliance_org:oecd` 已批准未来 forward inactivate，但 R2A 授权前保持现状。
- 确定性 QA sample 为 sheet rows 3、17、26、32、40、51，并追加 rows 45、50、全部 keep 映射和 `alliance_org:oecd`。抽样不替代 45 条候选与 10 条现有 disposition 的逐项决策。
- “核心主导方”只映射为 `leadership_summary`，不得自动生成 `led_by`；疑似业务语义问题只形成单行 Review note，不由 agent 改写。

### 4.3 Fail-closed

- Package 1 决策已穷尽并冻结在 `approved-alliance-manifest.md`；输入或 checksum 漂移时必须重新 Review。
- Package 1 批准只允许 Package 2 的 R0 候选工作，不授权 R1/R2 或任何 Neo4j 操作。

## 5. 两个 Local R2 执行包模板

每个 R2 必须独立记录目标 local 环境、精确入口、范围/排除项、approved manifest/version/checksum、recovery evidence、预计 counts、before/after assertions 和停止条件。curated local PostgreSQL 默认需要可恢复 backup；只有用户逐层明确批准的 disposable recovery 才可替代。

1. **R2A `master-data`**：必要 schema + alliance + economy 在一个授权和单事务边界内 forward converge；Query 同时验证 alliance active set、economy exceptions 与非目标保护快照。
2. **R2B `relationships`**：只在 R2A Query 验收后授权；写 approved `member_of`，以及 Package 2 同一 Review 明确批准的可选关系；Query 验证 active tuples、端点、方向、provenance、官方集合与幂等。

任一未授权范围、checksum/counts/断言漂移或 Query 失败立即停止，未执行授权失效。两个 R2 均不包含 Neo4j、生产写入或不可逆 cleanup。

## 6. 当前人工门禁

五 package、7 个顶层 checkbox、两个 local R2 和 Neo4j 移出保持不变。当前只等待 `package-2-candidate-review.md` R0 v1 的唯一业务 Review；Package 3 及以后保持未开始。
