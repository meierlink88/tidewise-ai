## 0. Active Change Workflow Adoption（R0）

- [x] 0.1 **已准备供 Review，尚未批准**：已安全吸收 `origin/main@4b3df5c` 的风险分级流程，并在 `workflow-adoption-review.md` 将本 change 的未来阶段映射到 R0—R3、阶段 Review package、候选数据审阅与 R2/R3 条件包；不追溯改写历史授权、业务候选或既有 checkbox。
- [ ] 0.2 **Workflow Adoption Review 门禁（R0）**：主对话审阅风险理由、证据、允许的下一步与明确不授权项；通过仅表示未来门禁采用新流程，不代表 2.3 通过，也不授权 C、R1 Apply、R2 PostgreSQL Write 或 R3 Neo4j rebuild。

## 1. A：Schema / Data Contract Review（历史 R0；状态不追认或重写）

- [x] 1.1 **已批准**：联盟最小 profile contract 为 `entity_id`、`abbreviation`、`categories TEXT[]`、`leadership_summary`、`influence_scope_summary`；名称/aliases/status 复用 `entity_nodes`，简称、长度、原子 22-code allowlist 与非空 summary 规则以 `contract-review.md` 为准。
- [x] 1.2 **已批准**：economy identity/ISO contract 区分 `sovereign_state`、`territory_economy`、`supranational_aggregate`、`global_aggregate`，并固定 `country_code`、ISO 3166、currency、region、EU/GLOBAL、active code 唯一与 stable `entity_key` 规则。
- [x] 1.3 **已批准**：不入库子类、CSV 成员数、全球占比、约束力级别、影响力评级；不新增实体标签、不复用事件标签，Neo4j 保持单一 `Entity` label。
- [x] 1.4 **Contract Review 门禁已通过**：主对话已批准 1.1—1.3；该批准只允许进入 B 的只读 provisional candidate draft，不构成 C、Apply 或任何 PostgreSQL/Neo4j 授权。

## 2. B：Alliance Candidate Review（R0 Candidate Review package）

- [x] 2.1 **已准备供 Review，尚未批准**：已在 `alliance-candidate-review.md` 将 CSV 第 1—68 条整理为 provisional 只读联盟候选草案，逐项留空 final decision，并展示 recommendation、目标 identity、名称、aliases、abbreviation、22-code categories、非空摘要、正式来源、现有 10 个联盟 exact diff 与 disposition 建议；未生成可执行 seed。
- [x] 2.2 **已准备供 Review，尚未批准**：已将 CSV 第 69—85 条战略矿产和农产品从 `alliance_org` 排除，只记录未来 `chain_node`、`commodity` 或 observation 候选方向，未创建实体、关系或后续 change。
- [ ] 2.3 **Alliance Candidate Review 门禁（R0）**：主对话依据 `workflow-adoption-review.md` 的输入指纹、counts、确定性 QA sample、冲突/异常全集和 fail-closed 断言，逐项确认 68 条候选与 10 条现有 active alliance 的最终 manifest，并审阅 reject/defer/unlisted/merge source 的 inactivate 或 merge 提议；manifest 未穷尽现有 active alliance、任一 stale/merge 未确认前，不得读取并冻结成员全集、形成 economy/关系候选或清理现有数据。通过只允许进入 C 的 R0 candidate package，不授权 R1/R2/R3。

## 3. C：Economy Candidate Review（R0 Candidate Review package；受 2.3 阻断）

- [ ] 3.1 仅对 2.3 已批准联盟逐一读取可审计正式成员来源，形成 formal active 成员全集，并单列 observer、partner、applicant、suspended、former 冲突；CSV 成员数仅作非权威对照。
- [ ] 3.2 将 formal active 成员全集与现有约 50 个 economy 做集合、canonical identity、ISO 3166、aliases、currency 和 region 差异审计；一致项复用稳定 entity key/UUID。
- [ ] 3.3 为缺失或冲突成员形成待补充 economy 候选清单，逐项包含规范中文名、英文名/aliases、identity kind、ISO code 或不适用、currency、region、正式来源、拟用 entity key 和冲突结论；不得生成可执行 seed。
- [ ] 3.4 生成 economy exception manifest：只列逐项确认的 identity 冲突、重复或明确错误 merge/inactivate；把所有其他合法 economy 纳入 key/UUID/status 保护快照，不得因不属于当前联盟成员全集而停用。
- [ ] 3.5 **Economy Candidate Review 门禁（R0）**：主对话依据 economy package 的来源/生成指纹、counts、确定性 sample、歧义/冲突全集、exception manifest 和非目标保护范围单独确认 economy 清单；未确认前不得生成最终 `member_of` 候选、修改正式 seed 或写数据库。通过只允许进入 D 的 R0 candidate package。

## 4. D：Relationship Contract / Candidate Review（R0 Candidate Review package；受 3.5 阻断）

- [ ] 4.1 在 3.5 通过后生成穷尽式 `member_of` manifest，固定 `economy -> alliance_org`，逐条包含 formal active 成员身份、两端 key、官方来源名称/URL、核验时间、现有 edge exact diff 和冲突报告，并覆盖每条现有 active `member_of`。
- [ ] 4.2 对不在最新 formal-active tuple set 的现有 active edge 逐条标记 `former/withdrawn/suspended/source_conflict/alliance_identity_convergence`，展示旧/新 identity、provenance、关系影响和预计 inactive counts；未决 source conflict 阻断 Write。
- [ ] 4.3 提交 `member_of` 数据完整性断言：两端存在且 active、无重复/悬空/错误方向，最终 active tuple set 与 approved manifest 集合相等，并按批准联盟计算成员集合/数量与同一官方来源逐项核对。
- [ ] 4.4 独立提交 `led_by` contract/candidates：`alliance_org -> economy/alliance_org`，只接受可解析且有证据的核心主导方；“多边”“轮值”等只保留文本，不造虚假实体。
- [ ] 4.5 独立提交 `part_of` contract/candidates：下属 `alliance_org -> alliance_org` 上级组织，只接受正式隶属证据；不得用合作或主题相关替代。
- [ ] 4.6 **Relationship Candidate Review 门禁（R0）**：主对话依据各关系 package 的来源/生成指纹、counts、确定性 sample、宽边界/冲突全集和 expected action classification，分别确认 `member_of`、`led_by`、`part_of`；后两层不得阻塞核心 alliance/economy/`member_of` MVP，任一候选 Review 都不代表 R2 Write 授权。

## 5. E：依赖与 Overlap Audit（R0 package）

- [ ] 5.1 等待 `refactor-industry-chain-node-foundation` 完成 Archive、Deliver 与 worktree/branch 隔离清理；在其完成前不得修改共享 entityfoundation seed/repository/migration tests，不得执行 migration、seed 或 PostgreSQL/Neo4j Write。
- [ ] 5.2 Deliver 后执行 `git fetch origin`，从最新 `origin/main` 重新建立本 branch 基线，读取产业链最终 artifacts/代码并输出共享文件、migration 序号、测试与 PostgreSQL 状态 overlap audit；若设计冲突先修订本 change artifacts 并重新 Review。
- [ ] 5.3 **R1 Apply 门禁**：只有 proposal artifacts 人工 Review、A—D 候选/契约与权威 manifest Review 和 5.1—5.2 全部通过后，才提交风险理由、允许修改的源码/测试边界、验证计划与明确不包含 R2/R3 写入的 Review package，由主对话明确授权进入 R1 Apply；不得由任务 checkbox、workflow adoption 或旧批准推定授权。

## 6. TDD 实现与只读 Dry Run（R1 Apply Review package；禁止有状态写）

- [ ] 6.1 先写 migration 静态测试与可重复 integration boundary，覆盖联盟 profile 目标字段、economy `identity_kind`、约束/索引、非破坏性 forward migration 和回滚/forward-fix 边界；先验证测试失败，再实现 migration 代码，但不 apply。
- [ ] 6.2 先写 loader/validator table-driven tests，覆盖 abbreviation→aliases、category allowlist/去重/禁用拼接、economy identity/ISO 组合、CSV 第 69—85 条排除、manifest 穷尽性/disposition enum 和未审阅候选 fail-closed；再实现生产校验。
- [ ] 6.3 先写 repository/service fake、sqlmock 或明确隔离的 integration tests，覆盖稳定 target identity、alliance/member exact diff、economy exception/protection snapshot、幂等 forward merge/inactivate/upsert、两个 active 重复阻断、created/updated/inactivated/merged/unchanged/failed report；再实现 repository/service/seed/convergence 逻辑。
- [ ] 6.4 先写 migration/command 防破坏测试，拒绝 `TRUNCATE`、无谓词 DELETE、清空重灌、历史 rollback 和 manifest checksum 漂移；再实现版本化 forward convergence。
- [ ] 6.5 先写关系 policy 与 graph mapper/writer fake tests，覆盖 `member_of` stale reason、inactive provenance 保留、`led_by`、`part_of` 方向/端点、formal active 边界、`MEMBER_OF/LED_BY/PART_OF` 映射和单一 `Entity` label；再实现生产代码。
- [ ] 6.6 运行 targeted tests、migration 静态验证、受影响交付边界 full suite 与共享 architecture/contract tests、OpenSpec strict validation、diff/scope/secret 检查；仅当变更触发共享规则、跨模块契约、公共基础设施或受影响边界不清时运行 repo-wide `go test ./...`。提交包含风险理由、变更范围、测试证据、允许的下一步和非授权项的 R1 scoped Apply Review package，不执行数据库或图谱写入。

## 7. F：PostgreSQL 分层 R0 Review → 命名 R2 Write → Query

- [ ] 7.1 **Alliance Write Review（R0）**：为命名 R2 层 `alliance-schema-and-data` 展示目标环境、精确入口、联盟 migration/schema diff、approved manifest/version/checksum、范围/排除项、现有 active 的穷尽 exact diff、每个 keep/create/merge/inactivate 的原因、旧/新 identity、profile/alias/关系影响、预计 counts、恢复选择及证据、事务、forward-fix、Write 前后断言与停止条件；单独请求该层 Write 授权。
- [ ] 7.2 **`alliance-schema-and-data`（R2）**：仅在 7.1 明确获批后执行 alliance schema/data forward convergence；立即 Query profile、identity、merge source/target、无双 active 重复、幂等，并证明 active alliance key set 与 approved manifest 集合相等，等待主对话验收。任一断言失败会使条件包内剩余未执行授权失效。
- [ ] 7.3 **Economy Write Review（R0）**：仅在 alliance Query 验收后，为命名 R2 层 `economy-data` 展示目标环境、精确入口、最终 economy seed、exception manifest、identity/ISO exact diff、范围/排除项、无关合法 economy key/UUID/status 保护快照、预计 counts、恢复证据、事务、forward-fix、Write 前后断言与停止条件；单独请求该层 Write 授权。
- [ ] 7.4 **`economy-data`（R2）**：仅在 7.3 明确获批后执行 economy forward convergence；立即 Query approved exceptions、identity/ISO/profile、稳定 target ID、重复/孤儿/关系影响/幂等，并证明未列入 exception manifest 的合法 economy 与保护快照逐项一致，等待主对话验收。任一断言失败会使剩余未执行授权失效。
- [ ] 7.5 **Member Of 最终 Review（R0）**：仅在 economy Query 验收后，为命名 R2 层 `member-of` 以真实 PostgreSQL IDs 刷新穷尽 manifest，展示目标环境、精确入口、formal-active target tuples、所有现有 active edge disposition、stale reason、旧/新 identity、官方来源、核验时间、范围/排除项、预计 create/update/inactivate、checksum、恢复证据、Write 前后断言与停止条件；单独请求该层 Write 授权。
- [ ] 7.6 **`member-of`（R2）**：仅在 7.5 明确获批后执行 `member_of` forward convergence；立即 Query 两端 active、方向、重复、悬空、inactive edge identity/provenance 保留、幂等，并证明 active tuple set 与 approved manifest 及官方正式成员集合逐项相等，等待主对话验收。
- [ ] 7.7 `led_by` 与 `part_of` 如继续推进，各自作为独立命名 R2 层执行 final candidate Review → Write → Query；不得被 `member-of` 授权覆盖，也不得阻塞核心 MVP 的 PostgreSQL 验收。
- [ ] 7.8 **PostgreSQL 验收门禁**：目标 PG 层的 Query 均经主对话验收后才可考虑 Neo4j；任一 manifest 不穷尽、checksum 漂移、active 集合不相等、identity/source/counts 不一致或无关 economy 保护断言变化立即停止，不得 truncate、delete/reload、手工修表或改走其他写路径。

## 8. G：Neo4j 独立 R0 Review → R3 Rebuild → Query

- [ ] 8.1 **Neo4j Review（R0）**：在 7.8 通过后提交独立 R3 package，明确只从 PostgreSQL active facts 重建、目标环境/namespace、精确入口、预计 `Entity` 节点与 `MEMBER_OF/LED_BY/PART_OF` counts、清理范围、恢复/灾备证据、前后断言与停止条件；单独请求 Rebuild 授权，不继承任何 R2 授权。
- [ ] 8.2 **Neo4j Rebuild（R3）**：仅在 8.1 独立明确获批后运行标准 graph projector rebuild，不直接手工写 Neo4j；立即 Query 单一 `Entity` label、`projection_namespace=tidewise`、关系方向/端点/类型/counts 与 PostgreSQL 一致，且无历史孤儿关系。
- [ ] 8.3 提交 PostgreSQL 与 Neo4j 最终 scoped evidence、targeted tests、`go test ./...`、OpenSpec strict validation 和 diff/secret 检查，停止等待 Apply 后人工 Review；批准前不得 Sync、Archive、Deliver 或创建完成态 PR。
