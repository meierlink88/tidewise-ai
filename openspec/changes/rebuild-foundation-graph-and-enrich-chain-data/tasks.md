## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 基础投影 R1 连续执行；local Neo4j cleanup 与 rebuild 分层授权 | R3 | yes | R3_OPERATION | 等待前置 change 完整 Deliver 后审计最终基线；只允许最小 projector 适配及两个独立 local Neo4j R3 层 |
| 2 | 842 节点全量四类关系候选 Review；local PostgreSQL R2 Write；local Neo4j R3 sync | R3 | yes | SPEC_SEMANTICS | 全量审计当前 842 个 chain_node，分批生成与 Review 候选，全部冻结后先 PG 后 Neo4j |
| 3 | Apply-final Review 与 Git completion | R1 | yes | APPLY_FINAL | 汇总 scoped diff 与验证；获批后才允许 Sync、Archive、PR merge 与 Desktop cleanup |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 4 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

`human_gates=3` 是与 Gate Map 三个 `Human=yes` package 行一致的机器计数。Human Decision Register 另行记录 package 内必要的业务 Review、stateful 授权、Apply-final 与 Git completion；普通 R1 测试、实现、dry read、修复、commit 与 push 不另增人工门禁。

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-neo4j-foundation-cleanup | 1 | local | 1 | 仅清空 local Neo4j 的 Tidewise 投影节点与关系，保留 database、约束、索引和配置 | PostgreSQL、UAT、prod、shared、其他 namespace | approved-disposable-recovery | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 前置 change 已完整 Deliver；PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise 投影节点与关系为零；database、约束、索引、配置和 PostgreSQL 不变 | workflow schema blocker 未解决、环境不符、PG baseline 漂移、越过 namespace、cleanup 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | 从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及符合端点边界的 entity_edges 与 chain_node_relations | market/index/benchmark、事件、观测、推理、未批准数据、UAT、prod、shared | approved-disposable-recovery | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分实体类型节点与分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | workflow schema blocker 未解决、PG baseline 漂移、projector failed/skipped、counts/type 或完整性断言失败立即停止 |
| all-chain-node-relations-postgres-write | 2 | local | 1 | 写入当前 842 个 chain_node 全量审计后经 Review 批准的细分节点与四类静态关系 | physical constraints、未审核候选、通用导入、UAT、prod、shared | backup | new:all-chain-node-relations-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 842/842 节点的四类关系审计状态与 final 候选已批准；local PG backup 可恢复；identity、scope、count、hash、schema 与范围外保护基线已冻结 | 批准数据写入结果、端点、tuple、orphan、范围外保护与幂等 Query 全部通过 | 覆盖率不足 842/842、基线漂移、identity 冲突、部分写入、范围外变化或断言失败立即停止 |
| all-chain-node-relations-neo4j-sync | 2 | local | 2 | 仅从已验收 PG accepted baseline 同步全量审计批准的 chain_node 与四类关系 | physical constraints、反写 PG、查询 API、派生关系、UAT、prod、shared | approved-disposable-recovery | new:all-chain-node-relations-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG 写后 Query 已验收；post-write PG identity、scope、count、hash、schema 已冻结；取得单独 Neo4j R3 授权 | Neo4j 与 PG accepted baseline counts/type 一致；missing、duplicate、orphan、legacy 为零 | workflow schema blocker 未解决、PG accepted baseline 漂移、未单独授权、同步失败或 Query 断言失败立即停止 |

### Workflow Schema Blocker

现行 workflow/lint 只允许 R3 layer 填写 `backup`，而 `approved-disposable-recovery` 仅允许 local R2。这无法诚实表达已批准的业务决策：三个 local Neo4j R3 layer 不创建 Neo4j backup/rollback，只以冻结且验收的 PostgreSQL projection/accepted baseline 重建 disposable projection。因此本 Proposal 保留真实的 `approved-disposable-recovery` 语义并接受 explicit task-design lint 失败；在项目 workflow schema 通过独立 change 合法表达此 R3 recovery 之前，本 change 不得进入 Apply。本 change 不修改 workflow/lint。

## Human Decision Register

| Gate | Package | Risk | Decision | Allows | Does Not Allow |
|---|---|---|---|---|---|
| G1 | 2 | R0 | 用户已批准当前 842 个 chain_node 全量四类关系完善范围 | 前置依赖满足后冻结 842 节点 identity 并分批生成候选 | 确认 final 候选、PG/Neo4j Write |
| G2 | 1 | R3 | `local-neo4j-foundation-cleanup` 授权 | 仅清空批准的 local Tidewise projection | rebuild、PG Write、其他环境 |
| G3 | 1 | R3 | `local-neo4j-foundation-rebuild` 授权 | 仅从冻结 PG baseline 重建 local projection | PG Write、其他环境 |
| G4 | 2 | R0 | 审核并冻结 final 节点/关系候选 | 准备审批数据的 PG preflight | PG/Neo4j Write |
| G5 | 2 | R2 | `all-chain-node-relations-postgres-write` 授权 | 仅写全量审计后批准数据并 Query | Neo4j sync、未审核数据 |
| G6 | 2 | R3 | `all-chain-node-relations-neo4j-sync` 授权 | 仅从 PG accepted baseline 同步并 Query | PG 反写、其他环境 |
| G7 | 3 | R1 | Apply-final Review | 通过后允许 Sync、Archive | PR merge、cleanup |
| G8 | 3 | R0 | Git completion | PR merge 与 Desktop-owned cleanup | 扩大 change scope |

## 1. 基础投影闭环 Package

- [ ] 1.1 等 `reinitialize-alliance-economy-foundation` 完整 Deliver、PR merge 与 cleanup 后，从最新 `origin/main` 只读核验最终 schema/data、重叠文件、active `entity_nodes`、`entity_edges`、`chain_node_relations` 与 projector gap；假设变化时回到 Review。
- [ ] 1.2 先写并运行仅覆盖当前 repository query、端点过滤、mapper、projector namespace/rebuild/run counts 和 CLI 输出的 targeted tests，再最小适配现有 projector；节点仅限 alliance_org/economy/chain_node，`entity_edges` 只投影两端均在该集合的关系，并投影 `chain_node_relations`。
- [ ] 1.3 只在 workflow schema blocker 已经独立解决后，准备并等待 `local-neo4j-foundation-cleanup` 独立 R3 授权；获批后只清 local Tidewise projection，不做 Neo4j backup/rollback，并立即 Query 为零。
- [ ] 1.4 cleanup 验收后准备并等待 `local-neo4j-foundation-rebuild` 独立 R3 授权；获批后从 PG baseline 重建，分别验收 alliance_org/economy/chain_node counts、各 relation type/count 及 missing/duplicate/orphan/legacy=0。

## 2. 842 节点全量产业链关系完善 Package

- [ ] 2.1 等前置 change 完整 Deliver 后，冻结当前 842 个 active chain_node 的 identity/count/hash；若最终基线不再是 842，必须带差异回到 Review，不得自行改写覆盖范围。
- [ ] 2.2 为 842 个基线节点建立 change-specific 全量覆盖清单，对四类关系分别记录待研究/有批准关系/不适用/证据不足；将全量工作分成可审阅批次，但不把任一批次视为 change 完成边界。
- [ ] 2.3 逐批从已有节点向下研究必要细分 chain_node；用一遍 AI 生成四类关系候选、来源、证据、反例、置信度和 disposition，再由主对话 double-check identity、端点、方向与证据；不为填满四类关系而伪造边。
- [ ] 2.4 逐批展示候选、不适用/证据不足结论与异常/冲突项，等待用户 Review；只有 842/842 节点的四类审计状态全部冻结且无未处置候选时，才能准备 PG Write。
- [ ] 2.5 默认作为纯数据任务复用现有 schema、写入能力与 graph-projector，不改源码、不新增测试、migration、repository/service、runner 或 dry-run/report framework；只读能力审计若发现硬缺口，必须带证据回到 Review。
- [ ] 2.6 准备并等待 `all-chain-node-relations-postgres-write` 独立 R2 授权；获批后在现有事务边界内原子写入全量审计的 final 节点/关系候选，立即 Query 覆盖率、端点、tuple、orphan、范围外保护与幂等。
- [ ] 2.7 只在 workflow schema blocker 已经独立解决且 PG Query 验收后，准备并等待 `all-chain-node-relations-neo4j-sync` 单独 R3 授权；获批后只读 PG 同步 Neo4j 并验收 counts/type、missing/duplicate/orphan/legacy=0。

## 3. Apply-final、Sync、Archive、Deliver Package

- [ ] 3.1 运行实际受影响 backend 边界的一次完整验证、OpenSpec 校验、diff/scope/secret 检查，提交 scoped Apply-final Review package 后停止等待人工 Review。
- [ ] 3.2 仅在 Apply-final Review 通过后 Sync、Archive、运行 `openspec validate --all` 并提交 archive checkpoint。
- [ ] 3.3 仅在 Git completion 授权后按规则完成 push、PR merge 与 Desktop-owned cleanup；全部条件满足前不得声明 Deliver。
