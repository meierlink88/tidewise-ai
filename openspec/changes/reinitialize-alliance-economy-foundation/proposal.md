## Why

现有联盟基础仅覆盖 10 个组织、50 个 economy 和 223 条 `member_of`，联盟 profile 仍使用 `org_code/org_type/primary_domain/scope_region/official_url`，无法表达已批准的多值分类、简称识别、主导方式摘要与影响范围，也没有把联盟确认、经济体补齐和成员关系建立串成可审计闭环。联盟研究 CSV 同时混有组织、战略资源、成员数快照和分析评级，必须以严格 identity、字段、manifest 与候选 Review 契约隔离候选材料和可执行事实。

## What Changes

- **BREAKING**：将 `alliance_org_profiles` 收敛为 `entity_id`、`abbreviation`、`categories TEXT[]`、`leadership_summary`、`influence_scope_summary`；名称、规范名称、aliases 和状态继续只保存在 `entity_nodes`。非空简称同时进入 aliases，分类使用已批准 22-code 原子多值。
- 复核现有 50 个 economy，建立 `country_code`、四类 identity kind、ISO 3166、currency、region、`economy:eu` 和 `economy:global` 的明确边界；一致项复用稳定 identity。
- CSV 1—68 只作为联盟候选 Review 输入；69—85 战略矿产与农产品排除出 `alliance_org`，只记录未来 `chain_node`、`commodity` 或 observation 候选，不创建后续 change。子类、CSV 成员数、全球占比、约束力级别、影响力评级均不入库。
- 使用版本化、穷尽、带 checksum 的 approved manifests 定义最终 active 状态：alliance manifest 覆盖所有最终 active keys 及现有 active 的 `keep/merge/inactivate`；economy exception manifest 只覆盖逐项批准的冲突/重复/错误；member manifest 覆盖所有 formal-active tuples 及现有 active edge 的 keep/stale disposition。
- alliance merge 保留 approved target 的稳定 identity，source 只做 forward inactivate，禁止两个 active 重复实体；stale `member_of` 保留 edge identity/provenance 并按 `former/withdrawn/suspended/source_conflict/alliance_identity_convergence` 审阅后转 inactive。合法 economy 不因不属于当前联盟成员全集而停用。
- `member_of` 固定 `economy -> alliance_org` 且只表达 active 正式成员；observer、partner、applicant、suspended、former 不得混入。正式成员数仅由 active edges 计算并与同一官方来源集合核对。
- `led_by` 与 `part_of` 作为 Package 2 的非阻塞可选附录：只有证据充分且在同一候选 Review 获批才进入本次 MVP；否则明确排除，不产生额外 gate。
- 把约 40 个细粒度 tasks 重构为 5 个内聚 package：联盟范围与 Spec Review、economy/关系候选 Review、R1 实现与自动技术验收、两个 local PostgreSQL R2、Apply-final 与 Deliver。人工确认仅保留候选业务语义、R2A、R2B 和 Apply-final。
- local R2A `master-data` 在一个 Review → Write → Query 包中统一执行必要 schema、alliance、economy forward convergence；R2B `relationships` 独立写 approved `member_of` 及同一候选 Review 已批准的可选关系。所有写入必须具备 exact manifest、recovery evidence、before/after assertions、单事务边界和 fail-closed 停止条件。
- 所有收敛只允许 versioned forward migration/convergence 与幂等事务，禁止 `TRUNCATE`、无谓词 DELETE、清空重灌、历史 rollback 或手工修表。Query 必须证明 active sets 与 manifests 相等且无关合法 economy 未被误改。
- PostgreSQL 是本 change 的唯一事实源和完成边界。Neo4j Review/Rebuild/Query 全部移出，未来由独立 graph projection change 读取已验收 PostgreSQL facts，不阻塞本 change。
- 与 `refactor-industry-chain-node-foundation` 共享的 entityfoundation seed/repository、migration tests 和 PostgreSQL 状态在该 change 完成 Deliver 且结果进入 `origin/main` 前保持冻结；R0 候选设计可继续，Package 3 只能在更新到最新主分支并完成 overlap audit 后开始。

## Capabilities

### New Capabilities

- `alliance-economy-foundation`: 定义联盟最小 profile、economy identity/ISO 边界、候选来源和“联盟确认 → 补齐经济体 → 建立成员关系”的完整性门禁。

### Modified Capabilities

- `event-knowledge-schema`: 收敛联盟 profile 并补充 economy 聚合身份与 ISO 边界。
- `entity-foundation-seeds`: 定义联盟/economy 候选 Review、稳定 identity 复用、exact diff、统一 master-data R2 与幂等验证。
- `entity-relationship-curation`: 定义 formal-active `member_of`，以及可选 `led_by`、`part_of` 的方向、证据、候选准入和 relationship R2。

## Impact

- 未来 Apply 预计影响 `backend/migrations/`、`backend/data/entity_foundation/`、`backend/internal/apps/entityfoundation/seed/`、`backend/internal/repositories/` 及对应 Go/migration tests；本 R0 checkpoint 只修改当前 change artifacts。
- 不修改 `prototype/`、项目外 `doc/` 或 graphprojection 源码；不涉及产业链实现、市场、benchmark/index、事件抽取/推理、observation 实现、实体标签机制、Neo4j rebuild 或股票推荐。
- 当前不执行 migration、seed、PostgreSQL/Neo4j 写入、投影重建、清理、完成态 PR 或 merge。
