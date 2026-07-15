## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 基础投影兼容修复；local Neo4j cleanup 与 rebuild 分层授权 | R3 | yes | R3_OPERATION | 仅允许现有 projector 兼容当前 PG 模型，以及两个独立 local Neo4j R3 层 |
| 2 | 842 节点 usable-map additive 关系分析；local PostgreSQL R2 Write；local Neo4j R3 sync | R3 | yes | SPEC_SEMANTICS | 保留已验收 100 条关系，完成 842/842 候选发现与双遍检查；冻结新增集合后先 PG 后 Neo4j |
| 3 | Apply-final Review 与 Git completion | R1 | yes | APPLY_FINAL | 汇总 scoped diff 与验证；获批后才允许 Sync、Archive、PR merge 与 Desktop cleanup |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 5 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

`human_gates=3` 是 Gate Map 的 package 级机器计数。普通 R1 实现、targeted tests、只读审计、commit 与 push 连续推进；人工只处理 Proposal 业务边界、R2/R3 有状态授权、Apply-final 与 Git completion。

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-neo4j-foundation-cleanup | 1 | local | 1 | Neo4j cleanup：仅清空 local Tidewise namespace 节点与关系，保留 database、约束、索引和配置 | PG 业务数据、UAT、prod、shared、其他 namespace | approved-disposable-recovery | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise namespace 节点与关系为零；database、约束、索引、配置和 PG 业务数据不变 | 环境不符、PG baseline 漂移、越过 namespace 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | Neo4j rebuild：从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及符合端点边界的 entity_edges 与 chain_node_relations | market/index/benchmark、事件、观测、推理、未批准数据、UAT、prod、shared | approved-disposable-recovery | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分实体类型节点与分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | PG baseline 漂移、projector failed/skipped 或完整性断言失败立即停止 |
| all-chain-node-relations-postgres-write | 2 | local | 1 | 仅向 chain_node_relations 写入第一批已冻结 100 条关系 manifest | entity_nodes、profiles、external identifiers、physical constraints、未审核候选、UAT、prod、shared | backup | new:all-chain-node-relations-pg-backup | counts=100;hash=review-gated;schema=review-gated | 第一批 100 条 manifest、端点、identity、hash、schema 与保护基线已冻结；backup 可恢复 | 100 条关系及 95/1/3/1 分布、端点、tuple、保护范围与幂等 Query 通过 | 节点主数据变化、基线漂移、identity 冲突、部分写入、范围外变化或断言失败立即停止 |
| usable-map-additive-relations-postgres-write | 2 | local | 2 | 仅向 chain_node_relations additive 写入经独立 Review 冻结的 usable-map 新增关系，不删除或改写既有 100 条 | entity_nodes、profiles、external identifiers、既有 100 条内容、physical constraints、blocked/rejected、UAT、prod、shared | backup | new:usable-map-additive-relations-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 842/842 候选发现与双遍检查完成；新增 manifest、既有 100 条保护 hash、端点、scope、schema 与 backup 已冻结 | 既有 100 条逐行不变；新增关系集合、类型、端点、tuple、orphan、保护范围与幂等 Query 全部通过 | 842 基线、既有 100 内容、manifest、schema 或环境漂移，部分写入或断言失败立即停止 |
| usable-map-final-relations-neo4j-sync | 2 | local | 3 | Neo4j sync：从最终验收的 PG accepted baseline 同步全部批准 chain_node_relations；执行 Scope 与入口须在独立 R3 包中精确冻结 | PG 反写、未审核候选、physical constraints、查询 API、派生关系、其他 namespace、UAT、prod、shared | approved-disposable-recovery | new:usable-map-final-relations-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PostgreSQL additive 写后 accepted baseline 已验收冻结；取得独立 Neo4j R3 授权；旧 4-edge sync 包已 superseded | Neo4j 与最终 PG relation counts/type/hash 一致；missing、duplicate、orphan、legacy 为零 | PG baseline 漂移、旧包被调用、Scope/入口未冻结、同步或 Query 失败立即停止 |

## Human Decision Register

| Gate | Package | Risk | Decision | Allows | Does Not Allow |
|---|---|---|---|---|---|
| G1 | 2 | R0 | 已确认 usable-map additive 语义与组合证据门槛 | 对 842 节点做候选发现与双遍检查，无事实节点可无边 | 强制 node×type 三态或任何写入 |
| G2 | 1 | R3 | `local-neo4j-foundation-cleanup` 授权 | 仅清空批准的 local Tidewise namespace | rebuild、PG 业务写、其他环境 |
| G3 | 1 | R3 | `local-neo4j-foundation-rebuild` 授权 | 仅从冻结 PG baseline 重建 local projection | PG 业务写、其他环境 |
| G4 | 2 | R0 | 已冻结第一批 100 条 Tier 1 accepted baseline | 准备第一批 PG preflight | Neo4j sync、后续 additive 集合 |
| G5 | 2 | R2 | 已授权 `all-chain-node-relations-postgres-write` | 仅写第一批 100 条并 Query | Neo4j sync、未审核数据 |
| G6 | 2 | R0 | 独立审核 usable-map additive Review 包 | 冻结新增候选与保护基线 | R1/R2/R3 或任何写入 |
| G7 | 2 | R2 | `usable-map-additive-relations-postgres-write` 授权 | 仅 additive 写入最终新增关系并 Query | 删除/改写既有 100、Neo4j sync |
| G8 | 2 | R3 | `usable-map-final-relations-neo4j-sync` 授权 | 仅从最终 PG accepted baseline 同步并 Query | PG 反写、旧 4-edge 包、其他环境 |
| G9 | 3 | R1 | Apply-final Review | 通过后允许 Sync、Archive | PR merge、cleanup |
| G10 | 3 | R0 | Git completion | PR merge 与 Desktop-owned cleanup | 扩大 change scope |

## 1. 基础投影闭环 Package

- [x] 1.1 Apply 前从最新 `origin/main` 复验 schema、active 三类节点、`entity_edges`、`chain_node_relations` 与重叠文件；若相对本 Proposal 审计结论漂移则回到 Review。
- [x] 1.2 先写 targeted tests，再仅修改现有 repository query 与 relation mapper：节点限 active alliance_org/economy/chain_node，entity_edges 两端过滤，读取 chain_node_relations，补齐四类 typed mapping；projector/writer/CLI 默认复用。
- [x] 1.3 冻结 PG projection baseline，准备并等待 `local-neo4j-foundation-cleanup` R3 授权；获批后仅清 local Tidewise namespace，不做 Neo4j backup/rollback，并立即 Query 为零。授权与脱敏 execution evidence 见 [local Neo4j 基础投影 cleanup R3 授权包](reviews/local-neo4j-foundation-cleanup-r3.md)；该 cleanup checkpoint 只验收 Tidewise nodes/relationships 为 0/0，不包含 rebuild，后续状态见 task 1.4。
- [x] 1.4 cleanup 验收后等待 `local-neo4j-foundation-rebuild` R3 授权；获批后从 PG baseline 重建并分别验收 alliance_org/economy/chain_node counts、各 relation type/count 及 missing/duplicate/orphan/legacy=0。独立授权与脱敏 execution evidence 见 [local Neo4j 基础投影 rebuild R3 授权包](reviews/local-neo4j-foundation-rebuild-r3.md)；单次 `project-entities` 已验收为 981 nodes / 229 relationships，未执行 cleanup、retry、sync、PG 业务写或 Package 2。

## 2. 842 节点产业链关系完善 Package

- [x] 2.1 冻结 842 个 active chain_node 的 identity/count/hash，并完成第一轮严格证据分析；当时的 842×4 三态账本现仅保留为历史分析证据，不再是 Package 2 的最终覆盖定义。
- [x] 2.2 在既有 842 节点硬边界内完成第一轮分批 AI 生成与 double-check；未新增、细化或向下穿透节点，批次不构成关闭边界。
- [x] 2.3 冻结第一批 100 条高置信关系及证据。历史 Review 包见 [第一批 100 条关系 R0 包](reviews/chain-node-relations-r0/README.md)，semantic SHA `b578e957df6e6249f745f2661f11a2d03c73434dab85fe8e2fb35f33bf14f2d9`；该集合继续作为 Tier 1 accepted baseline，但不再是 Package 2 最终关系集合。
- [x] 2.4 用 targeted tests 最小解除 `cmd/entity-seed` 对历史 96 条关系 manifest 的绑定，直接复用现有通用 transaction batch；不得新增节点写入、migration、repository/service、runner 或通用导入框架。入口已 fail-closed 绑定冻结 path/file SHA/semantic SHA、842 baseline 与 100/95/1/3/1，并以 live read-only dry-run 证明 4 created、96 unchanged、0 updated；后续独立授权边界见 [全量 chain_node_relations PostgreSQL Write R2 Review 包](reviews/all-chain-node-relations-postgres-write-r2.md)。
- [x] 2.5 获得 `all-chain-node-relations-postgres-write` R2 授权后，仅向 chain_node_relations 写入第一批 100 条关系，并验收端点、类型、tuple、orphan、节点主数据保护和幂等。单次 Write 与 accepted PG baseline 见 [R2 执行证据](reviews/all-chain-node-relations-postgres-write-r2-execution.md)；未执行 Neo4j sync。
- 旧 `all-chain-node-relations-neo4j-sync` 4-edge R3 Review 包已 superseded 且不可执行；其 3 INPUT_TO + 1 DEPENDS_ON 将与后续 additive accepted baseline 一并同步。
- [x] 2.6 按严格 usable-map 语义对 842 个节点完成 additive 候选发现与两遍实质检查；逐行保护既有 100 条，重审被拒 checkpoint 的 156 条候选，只有类型、方向、具体机制/path、条件、反例和来源蕴含完全一致的 112 条进入整改 manifest，44 条保持 blocked。每条获批 `input_to` 都记录实际产出及消费/转化/嵌入/明确服务机制；不以数量或连通率补边。本 checkpoint 不授权任何写入，见 [842-node usable-map additive R0 整改 Review 包](reviews/chain-node-relations-usable-map-r0/README.md)。
- [x] 2.7 R0 additive manifest 独立 Review 冻结后，以 targeted RED/GREEN tests 最小更新现有固定 contract：入口精确绑定 additive path 与完整文件 SHA `9578cd18e3b629b1e8df11d517c94ad25597bb47826511217812e1e7794c2ed8`，内部逐行拼接并保护既有 100 条，冻结新增 semantic SHA `5a533399a77c430e9067bac5ff509362c8168965a198801d665c40723cee4487`、112/13/2/90/7 与最终 212/108/3/93/8。live read-only dry-run 为 112 created、0 updated、100 unchanged；dry-run 后只读 Query 仍为 100/95/1/3/1，未执行 PG/Neo4j Write，未新增框架。
- [x] 2.8 `usable-map-additive-relations-postgres-write` 已按独立 R2 Review 与 fresh backup 授权单次执行：新增 112、更新 0、既有 100 unchanged；写后 PostgreSQL 为 212=`108/3/93/8`，旧 100、combined hashes、端点、完整性、保护范围与 `0/0/212` dry-run 全部通过。见 [execution 证据](reviews/usable-map-additive-relations-postgres-write-r2-execution.md)；未执行 restore、Neo4j 或后续层。
- [ ] 2.9 PG additive accepted baseline 验收后，已准备 [usable-map final relations Neo4j sync R3 Review 包](reviews/usable-map-final-relations-neo4j-sync-r3.md)：唯一策略为现有 `rebuild-entities` 对 local Tidewise namespace cleanup+全量 rebuild，以确保 Neo4j 与 PG 最终集合完全一致；旧 checkpoint `e8b2658` 与旧 4-edge 包禁止调用。当前仅等待独立 Review 与明确 R3 授权，尚未执行 Neo4j。

## 3. Apply-final、Sync、Archive、Deliver Package

- [ ] 3.1 运行实际受影响 backend 边界的一次完整验证、OpenSpec 校验、diff/scope/secret 检查，提交 scoped Apply-final Review package 后停止等待人工 Review。
- [ ] 3.2 仅在 Apply-final Review 通过后 Sync、Archive、运行 `openspec validate --all` 并提交 archive checkpoint。
- [ ] 3.3 仅在 Git completion 授权后按规则完成 push、PR merge 与 Desktop-owned cleanup；全部条件满足前不得声明 Deliver。
