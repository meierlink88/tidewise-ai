## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 基础投影 R1 连续执行；local Neo4j cleanup 与 rebuild 分层授权 | R3 | yes | R3_OPERATION | 等待前置 change 完整 Deliver 后审计最终基线；只允许最小 projector 适配及两个独立 local Neo4j R3 层 |
| 2 | 首批业务范围 Review；local PostgreSQL R2 Write；local Neo4j R3 sync | R3 | yes | SPEC_SEMANTICS | 仅处理审核通过的 AI 服务器算力基础设施批次，复用现有 schema/seed/repository/projector，先 PG 后 Neo4j |
| 3 | Apply-final Review 与 Git completion | R1 | yes | APPLY_FINAL | 汇总 scoped diff 与验证；获批后才允许 Sync、Archive、PR merge 与 Desktop cleanup |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 7 |
| stateful_layers | 4 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

`human_gates=7` 记录真实人工决策点，不把同一顶层 package 内的多次独立授权压缩为一次。普通 R1 测试、实现、dry read、修复、commit 与 push 在已批准 scope 内连续推进，不因 checkbox 人工停顿。

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-neo4j-foundation-cleanup | 1 | local | 1 | 仅清空 local Neo4j 的 Tidewise 投影节点与关系，保留 database、约束、索引和配置 | PostgreSQL、UAT、prod、shared、其他 namespace | approved-disposable-recovery | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 前置 change 已完整 Deliver；PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise 投影节点与关系为零；database、约束、索引、配置和 PostgreSQL 不变 | 环境不符、PG baseline 漂移、越过 namespace、cleanup 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | 从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及已批准关系 | 事件、观测、推理、未批准数据、UAT、prod、shared | approved-disposable-recovery | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分类型节点及分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | PG baseline 漂移、projector failed/skipped、counts/type 或完整性断言失败立即停止 |
| first-batch-postgres-write | 2 | local | 1 | 仅写入审核通过的 AI 服务器算力基础设施 change-specific manifest 中的 chain_node 与四类静态关系 | physical constraints、范围外 842 节点、通用导入、UAT、prod、shared | backup | new:first-batch-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 业务范围和 final manifest 已批准；local PG backup 可恢复；identity、scope、count、hash、schema 与范围外保护基线已冻结 | manifest 写入结果、端点、tuple、orphan、范围外保护与幂等 Query 全部通过 | manifest/基线漂移、identity 冲突、部分写入、范围外变化或断言失败立即停止 |
| first-batch-neo4j-sync | 2 | local | 2 | 仅从已验收 PG accepted baseline 同步首批 chain_node 与批准关系 | physical constraints、反写 PG、查询 API、派生关系、UAT、prod、shared | approved-disposable-recovery | new:first-batch-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG 写后 Query 已验收；post-write PG identity、scope、count、hash、schema 已冻结；取得单独 Neo4j R3 授权 | Neo4j 与 PG accepted baseline counts/type 一致；missing、duplicate、orphan、legacy 为零；typed Cypher 多跳验收通过 | PG accepted baseline 漂移、未单独授权、同步失败或 Query 断言失败立即停止 |

## Human Decision Register

| Gate | Package | Risk | Decision | Allows | Does Not Allow |
|---|---|---|---|---|---|
| G1 | 2 | R0 | Proposal 业务 Review：确认唯一推荐首批方向及边界 | 在前置依赖满足后按批准 scope 连续执行 R1 准备 | 任何 PG/Neo4j Write |
| G2 | 1 | R3 | `local-neo4j-foundation-cleanup` 授权 | 仅清空批准的 local Tidewise projection | rebuild、PG Write、其他环境 |
| G3 | 1 | R3 | `local-neo4j-foundation-rebuild` 授权 | 仅从冻结 PG baseline 重建 local projection | PG Write、其他环境 |
| G4 | 2 | R2 | `first-batch-postgres-write` 授权 | 仅写批准 manifest 并 Query | Neo4j sync、范围扩张 |
| G5 | 2 | R3 | `first-batch-neo4j-sync` 授权 | 仅从 PG accepted baseline 同步并 Query | PG 反写、其他环境 |
| G6 | 3 | R1 | Apply-final Review | 通过后允许 Sync、Archive | PR merge、cleanup |
| G7 | 3 | R0 | Git completion | PR merge 与 Desktop-owned cleanup | 扩大 change scope |

## 1. 基础投影闭环 Package

- [ ] 1.1 等 `reinitialize-alliance-economy-foundation` 完整 Deliver、PR merge 与 cleanup 后，从最新 `origin/main` 只读核验最终 schema/data、重叠文件、active `entity_nodes`、`entity_edges`、`chain_node_relations` 与 projector gap；假设变化时回到 Review。
- [ ] 1.2 先写并运行仅覆盖当前 repository query、四类 mapper、projector namespace/rebuild/run counts 和 CLI 输出的 targeted tests，再最小适配现有 projector；移除旧表/旧关系依赖，运行 targeted tests 与 dry read 后连续提交 R1 checkpoint。
- [ ] 1.3 准备并等待 `local-neo4j-foundation-cleanup` 独立 R3 授权；获批后只清 local Tidewise projection，立即 Query 为零，失败则停止。
- [ ] 1.4 cleanup 验收后准备并等待 `local-neo4j-foundation-rebuild` 独立 R3 授权；获批后从 PG baseline 重建，分别验收 alliance_org/economy/chain_node counts、各 relation type/count 及 missing/duplicate/orphan/legacy=0。

## 2. AI 服务器算力基础设施数据 Package

- [ ] 2.1 等待用户确认唯一推荐业务范围：entry node“AI服务器”，最多两跳直接上游硬件与一跳直接部署，建议上限 18 个 chain_node/28 条关系，包含直接硬件节点，排除完整半导体制造链、云/IDC 运营、软件/应用、公司/证券和 physical constraints。
- [ ] 2.2 范围批准且前置依赖满足后，用双遍 AI 分析生成并复核 change-specific manifest；只读确认现有 schema、entity-seed/repository、graph-projector 足够，发现硬缺口时携证据回到 Review，不新增 migration/repository/service/runner/report framework。
- [ ] 2.3 只为 manifest scope/identity、端点、现有事务写入、幂等、范围外保护和 typed Cypher 规则增加最小 tests；普通 R1 测试、修复、manifest/dry read 与 checkpoint 在批准 scope 内连续推进。
- [ ] 2.4 准备并等待 `first-batch-postgres-write` 独立 R2 授权；获批后执行最小 preflight、原子 PG Write、写后 Query 与幂等复验，任何漂移、冲突或范围外变化立即停止。
- [ ] 2.5 PG Query 验收后准备并等待 `first-batch-neo4j-sync` 单独 R3 授权；获批后只读 PG 同步 Neo4j，验收 counts/type、missing/duplicate/orphan/legacy=0，并用 Cypher 证明 `input_to` 顺向、`depends_on` 反向、分类/组成不计入上下游。

## 3. Apply-final、Sync、Archive、Deliver Package

- [ ] 3.1 运行实际受影响 backend 边界的一次完整验证、OpenSpec 校验、diff/scope/secret 检查，提交 scoped Apply-final Review package 后停止等待人工 Review。
- [ ] 3.2 仅在 Apply-final Review 通过后 Sync、Archive、运行 `openspec validate --all` 并提交 archive checkpoint。
- [ ] 3.3 仅在 Git completion 授权后按规则完成 push、PR merge 与 Desktop-owned cleanup；全部条件满足前不得声明 Deliver。
