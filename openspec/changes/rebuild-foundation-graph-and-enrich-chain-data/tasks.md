## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 基础投影兼容修复；local Neo4j cleanup 与 rebuild 分层授权 | R3 | yes | R3_OPERATION | 仅允许现有 projector 兼容当前 PG 模型，以及两个独立 local Neo4j R3 层 |
| 2 | 842 节点全量关系分析/候选 Review；local PostgreSQL R2 Write；local Neo4j R3 sync | R3 | yes | SPEC_SEMANTICS | 在本 change 内完成 842/842 四类关系审核状态；分批仅用于研究与 Review，全部冻结后先 PG 后 Neo4j |
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
| local-neo4j-foundation-cleanup | 1 | local | 1 | Neo4j cleanup：仅清空 local Tidewise namespace 节点与关系，保留 database、约束、索引和配置 | PG 业务数据、UAT、prod、shared、其他 namespace | approved-disposable-recovery | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise namespace 节点与关系为零；database、约束、索引、配置和 PG 业务数据不变 | 环境不符、PG baseline 漂移、越过 namespace 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | Neo4j rebuild：从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及符合端点边界的 entity_edges 与 chain_node_relations | market/index/benchmark、事件、观测、推理、未批准数据、UAT、prod、shared | approved-disposable-recovery | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分实体类型节点与分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | PG baseline 漂移、projector failed/skipped 或完整性断言失败立即停止 |
| all-chain-node-relations-postgres-write | 2 | local | 1 | 仅向 chain_node_relations 写入 842/842 全量审核完成并经 Review 冻结的四类关系 final manifest | entity_nodes、profiles、external identifiers、physical constraints、未审核候选、UAT、prod、shared | backup | new:all-chain-node-relations-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 842/842 覆盖账本与 relation final manifest 已批准；端点、identity、scope、count、hash、schema 与范围外保护基线已冻结；backup 可恢复 | 全量批准关系写入结果、端点、tuple、orphan、范围外保护与幂等 Query 全部通过；节点主数据不变 | 覆盖不足 842/842、节点主数据变化、基线漂移、identity 冲突、部分写入、范围外变化或断言失败立即停止 |
| all-chain-node-relations-neo4j-sync | 2 | local | 2 | Neo4j sync：仅从已验收 PG accepted baseline 同步 842 全量审核批准的四类 chain_node_relations，端点复用既有 chain_node | 节点主数据变更、physical constraints、反写 PG、查询 API、派生关系、UAT、prod、shared | approved-disposable-recovery | new:all-chain-node-relations-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PostgreSQL accepted baseline 写后 Query 已验收并冻结；取得单独 Neo4j R3 授权 | Neo4j 与 PG accepted baseline relation counts/type 一致；missing、duplicate、orphan、legacy 为零 | PG baseline 漂移、未单独授权、同步或 Query 失败立即停止 |

## Human Decision Register

| Gate | Package | Risk | Decision | Allows | Does Not Allow |
|---|---|---|---|---|---|
| G1 | 2 | R0 | 已确认本 change 完成 842/842 全量四类关系审核状态 | 在本 change 内分批研究与 Review | 将剩余节点拆到后续 change、final 候选或任何写入 |
| G2 | 1 | R3 | `local-neo4j-foundation-cleanup` 授权 | 仅清空批准的 local Tidewise namespace | rebuild、PG 业务写、其他环境 |
| G3 | 1 | R3 | `local-neo4j-foundation-rebuild` 授权 | 仅从冻结 PG baseline 重建 local projection | PG 业务写、其他环境 |
| G4 | 2 | R0 | 审核并冻结 842/842 全量 final 关系候选与无关系理由 | 准备 chain_node_relations PG preflight | PG/Neo4j Write |
| G5 | 2 | R2 | `all-chain-node-relations-postgres-write` 授权 | 仅写全量批准数据并 Query | Neo4j sync、未审核数据 |
| G6 | 2 | R3 | `all-chain-node-relations-neo4j-sync` 授权 | 仅从 PG accepted baseline 同步并 Query | PG 反写、其他环境 |
| G7 | 3 | R1 | Apply-final Review | 通过后允许 Sync、Archive | PR merge、cleanup |
| G8 | 3 | R0 | Git completion | PR merge 与 Desktop-owned cleanup | 扩大 change scope |

## 1. 基础投影闭环 Package

- [x] 1.1 Apply 前从最新 `origin/main` 复验 schema、active 三类节点、`entity_edges`、`chain_node_relations` 与重叠文件；若相对本 Proposal 审计结论漂移则回到 Review。
- [x] 1.2 先写 targeted tests，再仅修改现有 repository query 与 relation mapper：节点限 active alliance_org/economy/chain_node，entity_edges 两端过滤，读取 chain_node_relations，补齐四类 typed mapping；projector/writer/CLI 默认复用。
- [x] 1.3 冻结 PG projection baseline，准备并等待 `local-neo4j-foundation-cleanup` R3 授权；获批后仅清 local Tidewise namespace，不做 Neo4j backup/rollback，并立即 Query 为零。授权与脱敏 execution evidence 见 [local Neo4j 基础投影 cleanup R3 授权包](reviews/local-neo4j-foundation-cleanup-r3.md)；Tidewise nodes/relationships 已验收为 0/0，task 1.4 rebuild 仍未授权。
- [ ] 1.4 cleanup 验收后等待 `local-neo4j-foundation-rebuild` R3 授权；获批后从 PG baseline 重建并分别验收 alliance_org/economy/chain_node counts、各 relation type/count 及 missing/duplicate/orphan/legacy=0。独立 Review/授权包见 [local Neo4j 基础投影 rebuild R3 授权包](reviews/local-neo4j-foundation-rebuild-r3.md)；当前仅完成准备，尚未授权或执行。

## 2. 842 节点产业链关系完善 Package

- [ ] 2.1 冻结 842 个 active chain_node 的 identity/count/hash 与 842×4 覆盖账本；每个节点/关系类型最终只能是已批准关系、不适用或证据不足，并保留理由。
- [ ] 2.2 将 842 个既有节点分成可审阅研究批次，只分析节点之间的四类关系；不向下穿透、不研究或创建细分节点，不修改节点主数据。AI 生成与主对话 double-check 只作为本次数据分析方法，批次完成不得视为 Package 或 change 完成。
- [ ] 2.3 逐批展示关系候选、来源、证据、反例、置信度、方向、异常及不适用/证据不足理由；只有 842/842 全部审核状态完成、无待研究或未处置项时，才等待用户冻结全量 relation final manifest。
- [ ] 2.4 用 targeted tests 最小解除 `cmd/entity-seed` 对历史 96 条关系 manifest 的绑定，直接复用现有通用 transaction batch；不得新增节点写入、migration、repository/service、runner 或通用导入框架。
- [ ] 2.5 等待 `all-chain-node-relations-postgres-write` R2 授权；获批后只向 chain_node_relations 写入 842 全量 final manifest，并立即 Query 覆盖率、端点、类型、tuple、orphan、节点主数据保护和幂等。
- [ ] 2.6 PG 验收后，等待 `all-chain-node-relations-neo4j-sync` R3 授权；获批后只读 PG accepted baseline 同步，并验收 counts/type、missing/duplicate/orphan/legacy=0 及 typed 多跳规则。

## 3. Apply-final、Sync、Archive、Deliver Package

- [ ] 3.1 运行实际受影响 backend 边界的一次完整验证、OpenSpec 校验、diff/scope/secret 检查，提交 scoped Apply-final Review package 后停止等待人工 Review。
- [ ] 3.2 仅在 Apply-final Review 通过后 Sync、Archive、运行 `openspec validate --all` 并提交 archive checkpoint。
- [ ] 3.3 仅在 Git completion 授权后按规则完成 push、PR merge 与 Desktop-owned cleanup；全部条件满足前不得声明 Deliver。
