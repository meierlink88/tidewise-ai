## Why

现有联盟基础包含 10 个 `alliance_org`、50 个 economy 和 223 条 active `member_of`，联盟 profile 仍使用 `org_code/org_type/primary_domain/scope_region/official_url`。用户现已提供新的联盟候选真值源 `联盟组织列表1.0.xlsx`，明确当前联盟业务输入只包含名称、缩写、核心主导方和核心影响范围说明。旧 `表格_20260713.csv` 的 68 条候选、推荐结论与网页核验结果已被整体取代，只能作为 Git 历史记录，不能继续参与当前 manifest。

## What Changes

- **BREAKING**：将 `alliance_org_profiles` 最小目标收敛为 `entity_id`、`abbreviation`、`leadership_summary`、`influence_scope_summary`。名称与规范名称进入 `entity_nodes.name/canonical_name`；非空缩写保存在 profile，并按既有识别惯例派生为 alias，但 aliases 不是新的业务输入字段。
- `categories TEXT[]` 与 22-code allowlist 全部移出本 change。工作簿中的“大类”“子类”以及成员数、全球占比、约束力级别、影响力评级和其他 sheet 均不入库、不参与候选或 manifest。
- 当前候选唯一输入为 `联盟组织列表1.0.xlsx` SHA-256 `ac0d953c0cd93596fe6bf8a70541bbe658620e75d38a9b3178980071b2cdc102` 的首个 sheet `联盟组织`、范围 `A1:K51`：45 条数据行进入逐项 Review，5 条分组标题行不是实体。
- 45 条候选四个目标字段均非空，名称和规范化缩写均无重复。`UJR` 与 `CCAS` 的源缩写末尾各有一个 U+200C；候选规范化只删除该不可见字符，不做其他名称、缩写、主导方或影响说明的语义纠正。
- `leadership_summary` 直接保存“核心主导方”源文本。本阶段不得从该文本自动生成 `led_by`；如未来需要关系化，仍须在 Package 2 具备独立证据并随关系候选 Review。
- Package 1 已于 2026-07-14 获人工批准：45 条全部 approve，9 keep + 36 create，现有 10 条为 9 keep + OECD future forward inactivate；批准输入冻结为 `approved-alliance-manifest.md` v1 checksum `4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a`。旧 68 条 provisional Review 明确 superseded，不允许两套候选并存。
- 版本化且带 checksum 的 approved manifests 定义本 change 授权范围：alliance manifest 覆盖最终 active keys 和现有 active 的 `keep/merge/inactivate`；economy exception manifest 只覆盖逐项批准的冲突/重复/错误；member manifest 对 10 个 resolved target alliance scope 精确收敛，并穷尽分类现有 active edge 为 keep、preserve 或 proposed inactivate。
- alliance merge 继续保留 approved target 稳定 identity，source 只做 forward inactivate；合法 economy 不因不属于当前联盟成员全集而停用；stale `member_of` 保留 edge identity/provenance 后再按批准原因转 inactive。
- `member_of` 仍固定 `economy -> alliance_org` 且只表达 formal active。Package 2 已批准 10 个 resolved formal sets、79 个 economy 与 133 条候选；旧 223 条分为 31 keep、160 preserve_unresolved、10 preserve_pending_retype、22 OECD proposed_inactivate。preserve tuples 不属于本批 target tuple，保持原样且不阻断局部 MVP。
- 21 个 participant/signatory/framework/no-formal-membership 不生成 `member_of`；`participates_in`、`signatory_to` 等替代语义移至后续独立 relation-semantics change，本 change 不扩展 policy/schema。
- 所有收敛只允许 versioned forward migration/convergence 与幂等事务，禁止 `TRUNCATE`、无谓词 DELETE、清空重灌、历史 rollback 或手工修表。
- PostgreSQL 是本 change 的唯一事实源和完成边界；Neo4j 仍不在本 change。与 `refactor-industry-chain-node-foundation` 共享的源码、migration tests 和 PostgreSQL 状态在其 Deliver 且进入最新 `origin/main` 前保持冻结。

## Capabilities

### New Capabilities

- `alliance-economy-foundation`: 定义联盟四业务字段映射、economy identity/ISO 边界、Excel 候选来源和“联盟确认 → 补齐经济体 → 建立成员关系”的完整性门禁。

### Modified Capabilities

- `event-knowledge-schema`: 收敛联盟 entity/profile 字段并补充 economy 聚合身份与 ISO 边界。
- `entity-foundation-seeds`: 定义 Excel 候选 Review、稳定 identity 复用、exact diff、统一 master-data R2 与幂等验证。
- `entity-relationship-curation`: 定义 formal-active `member_of`，并禁止从“核心主导方”文本自动产生 `led_by`。

## Impact

- 未来 Apply 预计影响 `backend/migrations/`、`backend/data/entity_foundation/`、`backend/internal/apps/entityfoundation/seed/`、`backend/internal/repositories/` 及对应 Go/migration tests；本 R0 checkpoint 只修改当前 change artifacts。
- 不修改源工作簿、旧 CSV、`prototype/`、项目外 `doc/` 或 graphprojection 源码；不做网页研究。
- 当前不执行 migration、seed、PostgreSQL/Neo4j 写入、投影重建、清理、完成态 PR 或 merge。
