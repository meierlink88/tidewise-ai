## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 基础投影兼容修复；local Neo4j cleanup 与 rebuild 分层授权 | R3 | yes | R3_OPERATION | 仅允许现有 projector 兼容当前 PG 模型，以及两个独立 local Neo4j R3 层 |
| 2 | 842 节点关系完善范围/候选 Review；local PostgreSQL R2 Write；local Neo4j R3 sync | R3 | yes | SPEC_SEMANTICS | 保持 842 全量业务目标；先确定可关闭批次方案，再按批准数据先 PG 后 Neo4j |
| 3 | Apply-final Review 与 Git completion | R1 | yes | APPLY_FINAL | 汇总 scoped diff 与验证；获批后才允许 Sync、Archive、PR merge 与 Desktop cleanup |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 4 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

`human_gates=3` 是 Gate Map 的 package 级机器计数。普通 R1 实现、targeted tests、只读审计、commit 与 push 连续推进；人工只处理 Proposal 业务边界、R2/R3 有状态授权、Apply-final 与 Git completion。

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-neo4j-foundation-cleanup | 1 | local | 1 | 仅清空 local Neo4j 的 Tidewise namespace 节点与关系，保留 database、约束、索引和配置 | PG 业务数据、UAT、prod、shared、其他 namespace | approved-disposable-recovery | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise namespace 节点与关系为零；database、约束、索引、配置和 PG 业务数据不变 | workflow schema blocker 未解决、环境不符、PG baseline 漂移、越过 namespace 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | 从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及符合端点边界的 entity_edges 与 chain_node_relations | market/index/benchmark、事件、观测、推理、未批准数据、UAT、prod、shared | approved-disposable-recovery | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分实体类型节点与分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | workflow schema blocker 未解决、PG baseline 漂移、projector failed/skipped 或完整性断言失败立即停止 |
| all-chain-node-relations-postgres-write | 2 | local | 1 | 仅写入当次已冻结、经 Review 批准的细分节点与四类关系数据批次 | physical constraints、未审核候选、范围外节点/关系、UAT、prod、shared | backup | new:all-chain-node-relations-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 批次 manifest、端点、identity、scope、count、hash、schema 与范围外保护基线已批准并冻结；backup 可恢复 | 批准数据写入结果、端点、tuple、orphan、范围外保护与幂等 Query 全部通过 | 基线漂移、identity 冲突、部分写入、范围外变化或断言失败立即停止 |
| all-chain-node-relations-neo4j-sync | 2 | local | 2 | 仅从已验收 PG accepted baseline 同步当次批准的 chain_node 与四类关系 | physical constraints、反写 PG、查询 API、派生关系、UAT、prod、shared | approved-disposable-recovery | new:all-chain-node-relations-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG 写后 Query 已验收；accepted baseline 已冻结；取得单独 Neo4j R3 授权 | Neo4j 与 PG accepted baseline counts/type 一致；missing、duplicate、orphan、legacy 为零 | workflow schema blocker 未解决、PG baseline 漂移、未单独授权、同步或 Query 失败立即停止 |

### 当前 Workflow Schema Blocker

最新 workflow/lint 仍不允许 R3 使用 `approved-disposable-recovery`。本 change 保留真实语义，不伪造 Neo4j backup，不修改 workflow；explicit task-design lint 因此预期失败，且独立规则变更前不得 Apply。

## Human Decision Register

| Gate | Package | Risk | Decision | Allows | Does Not Allow |
|---|---|---|---|---|---|
| G1 | 2 | R0 | 确认 842 全量目标的可关闭批次方案 | 冻结批次边界并开始候选分析 | final 候选或任何写入 |
| G2 | 1 | R3 | `local-neo4j-foundation-cleanup` 授权 | 仅清空批准的 local Tidewise namespace | rebuild、PG 业务写、其他环境 |
| G3 | 1 | R3 | `local-neo4j-foundation-rebuild` 授权 | 仅从冻结 PG baseline 重建 local projection | PG 业务写、其他环境 |
| G4 | 2 | R0 | 审核并冻结当次 final 节点/关系候选 | 准备对应 PG preflight | PG/Neo4j Write |
| G5 | 2 | R2 | `all-chain-node-relations-postgres-write` 授权 | 仅写批准批次并 Query | Neo4j sync、未审核数据 |
| G6 | 2 | R3 | `all-chain-node-relations-neo4j-sync` 授权 | 仅从 PG accepted baseline 同步并 Query | PG 反写、其他环境 |
| G7 | 3 | R1 | Apply-final Review | 通过后允许 Sync、Archive | PR merge、cleanup |
| G8 | 3 | R0 | Git completion | PR merge 与 Desktop-owned cleanup | 扩大 change scope |

## 1. 基础投影闭环 Package

- [ ] 1.1 Apply 前从最新 `origin/main` 复验 schema、active 三类节点、`entity_edges`、`chain_node_relations`、重叠文件与 workflow blocker；若相对本 Proposal 审计结论漂移则回到 Review。
- [ ] 1.2 先写 targeted tests，再仅修改现有 repository query 与 relation mapper：节点限 active alliance_org/economy/chain_node，entity_edges 两端过滤，读取 chain_node_relations，补齐四类 typed mapping；projector/writer/CLI 默认复用。
- [ ] 1.3 workflow blocker 独立解除后，冻结 PG projection baseline，准备并等待 `local-neo4j-foundation-cleanup` R3 授权；获批后仅清 local Tidewise namespace，不做 Neo4j backup/rollback，并立即 Query 为零。
- [ ] 1.4 cleanup 验收后等待 `local-neo4j-foundation-rebuild` R3 授权；获批后从 PG baseline 重建并分别验收 alliance_org/economy/chain_node counts、各 relation type/count 及 missing/duplicate/orphan/legacy=0。

## 2. 842 节点产业链关系完善 Package

- [ ] 2.1 在 Proposal Review 确认批次关闭方式：单 change 直至 842/842，或推荐的本 change 首批闭环、后续 data-only changes 持续完成 842 总目标；未确认不得开始候选 Apply。
- [ ] 2.2 冻结 active chain_node identity/count/hash 与覆盖账本；按批准边界为四类关系记录有候选/不适用/证据不足，并从已有节点向下研究有证据的细分节点。AI 生成与主对话 double-check 只作为本次数据分析方法。
- [ ] 2.3 展示当批节点/关系候选、来源、证据、反例、置信度、方向、异常及无关系理由，等待用户冻结 final manifest；不得把候选 Review 推定为写授权。
- [ ] 2.4 若 final manifest 仅含现有节点关系，先用 targeted tests 最小解除 `cmd/entity-seed` 对历史 96 条 manifest 的绑定并复用现有通用 transaction batch；若包含新增节点，先回到 Review 展示 entity+relation 原子事务缺口，不预授权实现。
- [ ] 2.5 等待 `all-chain-node-relations-postgres-write` R2 授权；获批后只写 final manifest，并立即 Query 端点、类型、tuple、orphan、范围外保护和幂等。
- [ ] 2.6 PG 验收且 workflow blocker 解除后，等待 `all-chain-node-relations-neo4j-sync` R3 授权；获批后只读 PG accepted baseline 同步，并验收 counts/type、missing/duplicate/orphan/legacy=0 及 typed 多跳规则。

## 3. Apply-final、Sync、Archive、Deliver Package

- [ ] 3.1 运行实际受影响 backend 边界的一次完整验证、OpenSpec 校验、diff/scope/secret 检查，提交 scoped Apply-final Review package 后停止等待人工 Review。
- [ ] 3.2 仅在 Apply-final Review 通过后 Sync、Archive、运行 `openspec validate --all` 并提交 archive checkpoint。
- [ ] 3.3 仅在 Git completion 授权后按规则完成 push、PR merge 与 Desktop-owned cleanup；全部条件满足前不得声明 Deliver。
