## 1. A：Schema / Data Contract Review

- [ ] 1.1 提交联盟最小 profile contract Review：`entity_id`、`abbreviation`、`categories TEXT[]`、`leadership_summary`、`influence_scope_summary`，名称/aliases/status 复用 `entity_nodes`；确认 abbreviation→aliases、空简称和 category 原子值/allowlist 规则。
- [ ] 1.2 提交 economy identity/ISO contract Review：`sovereign_state`、`territory_economy`、`supranational_aggregate`、`global_aggregate` 与 `country_code`、ISO 3166、currency、region 组合规则，明确 `economy:eu`、`economy:global` 不与主权国家混淆。
- [ ] 1.3 确认不入库字段：子类、CSV 成员数、全球占比、约束力级别、影响力评级；确认不新增实体标签、不复用事件标签、Neo4j 保持单一 `Entity` label。
- [ ] 1.4 **Contract Review 门禁**：主对话明确批准 1.1—1.3 前，不得冻结联盟候选、生成 economy 范围、生成关系候选、修改源码/migration/seed 或执行任何 PostgreSQL/Neo4j Write。

## 2. B：Alliance Candidate Review

- [ ] 2.1 将 CSV 第 1—68 条整理为只读联盟候选清单，逐项展示 `approve/reject/merge/defer`、目标 entity key/UUID 复用建议、canonical name、aliases、abbreviation、categories、leadership/influence summaries、正式来源和现有 10 个联盟差异；不得生成可执行 seed。
- [ ] 2.2 将 CSV 第 69—85 条战略矿产和农产品从 `alliance_org` 排除，只记录为未来 `chain_node`、`commodity` 或 observation 候选，不自行创建实体、关系或后续 change。
- [ ] 2.3 **Alliance Candidate Review 门禁**：主对话逐项确认最终联盟清单；未确认前不得读取并冻结成员全集、不得形成 economy 候选或关系候选。

## 3. C：Economy Candidate Review

- [ ] 3.1 仅对 2.3 已批准联盟逐一读取可审计正式成员来源，形成 formal active 成员全集，并单列 observer、partner、applicant、suspended、former 冲突；CSV 成员数仅作非权威对照。
- [ ] 3.2 将 formal active 成员全集与现有约 50 个 economy 做集合、canonical identity、ISO 3166、aliases、currency 和 region 差异审计；一致项复用稳定 entity key/UUID。
- [ ] 3.3 为缺失或冲突成员形成待补充 economy 候选清单，逐项包含规范中文名、英文名/aliases、identity kind、ISO code 或不适用、currency、region、正式来源、拟用 entity key 和冲突结论；不得生成可执行 seed。
- [ ] 3.4 **Economy Candidate Review 门禁**：主对话单独确认 economy 清单；未确认前不得生成最终 `member_of` 候选、修改正式 seed 或写数据库。

## 4. D：Relationship Contract / Candidate Review

- [ ] 4.1 在 3.4 通过后生成 `member_of` 候选，固定 `economy -> alliance_org`，逐条包含 formal active 成员身份、两端 key、官方来源名称/URL、核验时间、现有 edge 差异和冲突报告。
- [ ] 4.2 提交 `member_of` 数据完整性断言：两端存在且 active、无重复/悬空/错误方向，并按批准联盟由 active edges 计算成员集合与数量，再与同一官方来源逐项核对。
- [ ] 4.3 独立提交 `led_by` contract/candidates：`alliance_org -> economy/alliance_org`，只接受可解析且有证据的核心主导方；“多边”“轮值”等只保留文本，不造虚假实体。
- [ ] 4.4 独立提交 `part_of` contract/candidates：下属 `alliance_org -> alliance_org` 上级组织，只接受正式隶属证据；不得用合作或主题相关替代。
- [ ] 4.5 **Relationship Candidate Review 门禁**：主对话分别确认 `member_of`、`led_by`、`part_of`；后两层不得阻塞核心 alliance/economy/`member_of` MVP，任一候选 Review 都不代表 Write 授权。

## 5. E：等待产业链 Change Deliver 后进入 Apply

- [ ] 5.1 等待 `refactor-industry-chain-node-foundation` 完成 Archive、Deliver 与 worktree/branch 隔离清理；在其完成前不得修改共享 entityfoundation seed/repository/migration tests，不得执行 migration、seed 或 PostgreSQL/Neo4j Write。
- [ ] 5.2 Deliver 后执行 `git fetch origin`，从最新 `origin/main` 重新建立本 branch 基线，读取产业链最终 artifacts/代码并输出共享文件、migration 序号、测试与 PostgreSQL 状态 overlap audit；若设计冲突先修订本 change artifacts 并重新 Review。
- [ ] 5.3 **Apply 门禁**：只有 proposal artifacts 人工 Review、A—D 候选/契约 Review 和 5.1—5.2 全部通过后，主对话才可明确授权进入 Apply；不得由任务 checkbox 或旧批准推定授权。

## 6. TDD 实现与只读 Dry Run

- [ ] 6.1 先写 migration 静态测试与可重复 integration boundary，覆盖联盟 profile 目标字段、economy `identity_kind`、约束/索引、非破坏性 forward migration 和回滚/forward-fix 边界；先验证测试失败，再实现 migration 代码，但不 apply。
- [ ] 6.2 先写 loader/validator table-driven tests，覆盖 abbreviation→aliases、category allowlist/去重/禁用拼接、economy identity/ISO 组合、CSV 第 69—85 条排除和未审阅候选 fail-closed；再实现生产校验。
- [ ] 6.3 先写 repository/service fake、sqlmock 或明确隔离的 integration tests，覆盖稳定 identity 复用、联盟/economy 分层 dry-run、幂等 upsert、identity conflict、created/updated/unchanged/failed report；再实现 repository/service/seed 逻辑。
- [ ] 6.4 先写关系 policy 与 graph mapper/writer fake tests，覆盖 `member_of`、`led_by`、`part_of` 方向/端点/provenance、formal active 边界、`MEMBER_OF/LED_BY/PART_OF` 映射和单一 `Entity` label；再实现生产代码。
- [ ] 6.5 运行 targeted tests、`go test ./...`、migration 静态验证、OpenSpec strict validation、diff/scope/secret 检查；提交只含代码与测试的 scoped Apply diff 供人工 Review，不执行数据库或图谱写入。

## 7. F：PostgreSQL 分层 Review → Write → Query

- [ ] 7.1 **Alliance Write Review**：展示联盟 migration/schema diff、最终逐项 seed、dry-run、现有 10 个联盟 identity 复用/冲突、预计 created/updated/inactive counts、可恢复备份证据、事务与 forward-fix；单独请求 alliance Write 授权。
- [ ] 7.2 仅在 7.1 获批后执行 alliance schema/data Write；立即 Query profile 字段、categories、abbreviation aliases、排除字段、identity、counts、冲突与幂等，等待主对话验收。
- [ ] 7.3 **Economy Write Review**：仅在 alliance Query 验收后展示最终 economy seed、identity/ISO diff、dry-run、复用/新增 counts、备份/事务/forward-fix；单独请求 economy Write 授权。
- [ ] 7.4 仅在 7.3 获批后执行 economy Write；立即 Query identity kind、ISO/code、currency、region、aliases、稳定 ID、重复/孤儿/冲突与幂等，等待主对话验收。
- [ ] 7.5 **Member Of 最终 Review**：仅在 economy Query 验收后以真实 PostgreSQL entity IDs 刷新候选，展示方向、formal active 身份、官方来源、核验时间、预计新增/更新/停用、集合差异与成员计数断言；单独请求 `member_of` Write 授权。
- [ ] 7.6 仅在 7.5 获批后执行 `member_of` Write；立即 Query 两端 active、方向、重复、悬空、provenance、按联盟 active edge 集合/数量与官方来源逐项一致、幂等，等待主对话验收。
- [ ] 7.7 `led_by` 与 `part_of` 如继续推进，各自独立执行 final candidate Review → Write → Query；不得被 `member_of` 授权覆盖，也不得阻塞核心 MVP 的 PostgreSQL 验收。
- [ ] 7.8 **PostgreSQL 验收门禁**：目标 PG 层的 Query 均经主对话验收后才可考虑 Neo4j；任一 counts、identity、来源或集合不一致立即停止，不得手工修表或改走其他写路径。

## 8. G：Neo4j 独立 Review → Rebuild → Query

- [ ] 8.1 在 7.8 通过后提交 Neo4j Review：明确只从 PostgreSQL active facts 重建、目标 namespace、预计 `Entity` 节点与 `MEMBER_OF/LED_BY/PART_OF` counts、清理范围、失败/恢复边界；单独请求 Rebuild 授权。
- [ ] 8.2 仅在 8.1 获批后运行标准 graph projector rebuild，不直接手工写 Neo4j；立即 Query 单一 `Entity` label、`projection_namespace=tidewise`、关系方向/端点/类型/counts 与 PostgreSQL 一致，且无历史孤儿关系。
- [ ] 8.3 提交 PostgreSQL 与 Neo4j 最终 scoped evidence、targeted tests、`go test ./...`、OpenSpec strict validation 和 diff/secret 检查，停止等待 Apply 后人工 Review；批准前不得 Sync、Archive、Deliver 或创建完成态 PR。
