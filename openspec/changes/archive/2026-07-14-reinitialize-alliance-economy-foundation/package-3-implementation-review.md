# Package 3.1 R1 Implementation Review

> 状态：R1 实现与只读审计已收敛，等待本 checkpoint 的主对话独立验收。本文件不是 4.1 R3 cleanup 或 4.2 R2 rebuild 的写入授权。

## 1. 收缩后的实现边界

本 change 是一次性但可审阅、可重放的 local PostgreSQL 业务数据批次，不是通用导入产品。

R1 必须保留：

- `alliance_org_profiles` 三个业务字段的最小原地 schema/domain/repository 适配；名称仍只在 `entity_nodes`；
- frozen approved data artifact：45 alliance、79 economy、133 formal-active `member_of`；
- 一个只服务本 change、优先复用现有 entity-seed/repository 的最小 importer；
- 精确 scoped cleanup/rebuild 的 preflight、zero/post Query 和 targeted tests。

R1 必须排除：

- `economy_profiles.identity_kind`、区域/货币 schema policy、全局 `entity_nodes.entity_key` unique、平行 v18 profile 表；
- 通用 manifest/service/policy/mapping framework、复杂 dry-run/report；
- 旧 223 disposition 的 keep/preserve/proposed-inactivate 执行策略、backup/recovery/restore rehearsal、Neo4j。

## 2. Frozen Target

| 集合 | Approved count | Importer canonical checksum |
|---|---:|---|
| Alliance | 45 | `4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a` |
| Economy | 79 | `66c1e13af00ca1c898132fec113812e971aa52adbf2da17400e331a4c5ae3db1` |
| Formal-active `member_of` | 133 | `c3d652571fa93307088633cbfe06dce18b1257d8ac2b3ee2ae88c5a27e69fcf7` |

Package 2 的 economy 历史审阅 checksum `95613a...` 包含当时仅用于候选审阅的 `identity_kind` 列。本 amendment 已明确 importer 不要求该字段，因此执行 artifact 删除该列并产生上表新的 importer checksum；79 条 identity/name/aliases/country/currency/region/action 未改。最终文件 SHA-256 为 `118573006c341830f3c977a994d12a537fe2d1ac8174a818d8007fe98e87a55d`。旧 existing-disposition checksum 只保留为历史证据，不是 importer 输入。

## 3. 真实 Local PostgreSQL 只读基线

本轮只读审计未执行 migration、seed、cleanup 或其他写入。

| 项目 | 只读结果 |
|---|---:|
| Goose schema version | 17 |
| Active alliance / alliance profiles | 10 / 10 |
| Active economy / economy profiles | 50 / 50 |
| Active `economy -> alliance_org member_of` | 223 |
| Active `economy -> market has_market` | 40 |
| `market_profiles.economy_entity_id` 非空引用 | 47 |
| `company_profiles.registration_economy_entity_id` 非空引用 | 77 |
| `person_profiles.economy_entity_id` 非空引用 | 30 |
| `entity_external_identifiers` 目标引用 | 0 |
| event links 目标引用 | 0 |
| `benchmark_profiles` 直接 economy FK | 0 |

间接市场依赖：43 条 `index_profiles` 通过 economy-linked market 被关联；相关 market edges 包含 43 条 `tracks_index` 与 10 条 `observes_benchmark`。这些不是 45/79/133 rebuild artifact 的恢复内容。

## 4. 目标表、FK 与引用类别

R3 cleanup Review 至少必须覆盖：

- 主表/profile：`entity_nodes`、`alliance_org_profiles`、`economy_profiles`；
- 关系：`entity_edges.from_entity_id/to_entity_id`，按 relation type、方向、endpoint type 和 tuple hash 分组；
- 直接 economy 引用：`market_profiles.economy_entity_id`、`sector_profiles.primary_economy_entity_id`、`industry_chain_profiles.primary_economy_entity_id`、`company_profiles.registration_economy_entity_id`、`person_profiles.economy_entity_id`；
- 通用引用：`entity_external_identifiers.entity_id ON DELETE CASCADE`，以及其他引用 `entity_nodes` 的 profile、convergence/audit 表；
- 间接事实：economy-linked market 关联的 index/benchmark 及 `tracks_index`、`observes_benchmark` edges。

任何未在执行前刷新、未给出 count/hash 或未决定处置的引用都阻断 4.1。

## 5. 15 个现有但不在 79 Target 的 Economy

只读审计确认以下 15 个现有 economy 均存在：

```text
economy:ar, economy:au, economy:bd, economy:ch, economy:cl,
economy:eu, economy:global, economy:hk, economy:il, economy:kr,
economy:ma, economy:mx, economy:nz, economy:tw, economy:ua
```

已知跨域引用分布：

- company registration：AU、CH、CL、KR、TW；
- market profile：AU、EU、GLOBAL、HK、KR、TW；
- person profile：EU、IL、TW、UA；
- `has_market`：AU、EU、HK、KR、TW。

这 15 个 economy 不在 latest 79 target，因此若 cleanup 删除它们，4.2 不会自动恢复其跨域事实。主对话必须按关系/FK 类别明确选择：

1. **删除并丢弃**：明确接受关联 profile/edge 的业务事实消失，并把精确 count/hash 写入 R3 授权；或
2. **保留/重建**：从 entity cleanup 排除对应 economy，或把其 entity/profile/edge 纳入另行批准的恢复清单。

当前没有选择，故 4.1 R3 **blocked for authorization**；这不阻止 R1 源码和测试完成，但禁止任何 cleanup。

## 6. Package 4 精确执行包

### 4.1 R3 Scoped Local Cleanup

未来 Review package 必须提供：

- local 环境身份与 goose version；
- 目标表、精确谓词、relation types、before counts 和 canonical hashes；
- 上述 15 个非目标 economy 与所有跨域 FK/edges 的逐类处置；
- migration 与 cleanup 的 SQL 顺序、事务边界和 fail-closed 条件；
- zero Query：获批 alliance/economy/member scope 为零；未授权跨域 tuple 不变。

用户已豁免本次 local 探索数据的 backup、rollback 和恢复演练。该豁免不适用于 UAT、prod 或 shared。Package 3.1、普通 Apply 或本 Review 草案都不构成 4.1 授权。

### 4.2 R2 Latest Manifest Rebuild

只在 4.1 zero Query 验收后另行授权：

- 只加载最终冻结 artifact；
- 重建 45 alliance、79 economy、133 formal-active `member_of`；
- Query 证明 count 与集合精确相等、端点存在且 active、方向为 economy → alliance、无孤儿、无重复；
- 同一 artifact 复跑不产生新增、删除或集合漂移。

4.1 成功不推定 4.2 授权；任一版本/count/hash/依赖漂移都必须重新 Review。

## 7. R1 实现与规模

复用点：现有 `cmd/entity-seed`、`NewPostgresRepository`、`buildProfileUpsert`、`entitySeedUUID`、`entity_nodes/economy_profiles/entity_edges` 与 provenance 列。没有新增 service、policy engine、通用 runner、通用 manifest framework 或通用 dry-run/report 子系统。

新增 change-specific 生产文件：

- `alliance_economy_manifest.go`：391 行，冻结文件 checksum、45/79/133 与端点/来源校验；
- `alliance_economy_rebuild.go`：498 行，其中主要为本批 preflight/fingerprint/cleanup/rebuild/exact Query SQL；
- migration 000018：48 行，只原地调整 alliance profile 三字段。

现有入口最小修改：`cmd/entity-seed/main.go` +77 行；`postgres_repository.go` 将 alliance profile 从旧五字段改为获批三字段；domain 仅增加 alliance profile 三字段验证。已撤回早期过宽的 `identity_kind` domain/schema、全局 `entity_key` unique、平行 v18 tables、通用 plan/dry-run 模型和旧 223 disposition validator。

## 8. R1 验证证据

- targeted/共享：`go test -count=1 ./internal/apps/entityfoundation/seed ./cmd/entity-seed ./internal/domain ./migrations ./internal/platform/dbmigration ./internal/architecture ./internal/repositories`；
- 受影响 backend：`go test -count=1 ./...`；
- explicit task-design lint：`OPENSPEC_TASK_LINT_CHANGE=reinitialize-alliance-economy-foundation go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1`；
- OpenSpec：`openspec validate reinitialize-alliance-economy-foundation --strict`；
- Git/安全：`git diff --check`、scoped status、敏感字段扫描。

本轮未运行 migration 000018、cleanup、rebuild、seed 或任何 PostgreSQL/Neo4j 写入。新 SQL 只经过静态与 sqlmock targeted tests；真实写入验证必须等待 4.1/4.2 各自授权。跨域处置仍未决，因此 4.1 当前保持 blocked for authorization。
