## Why

修复 PostgreSQL 到 Neo4j 的当前实体模型投影闭环，并完成当前 842 个 chain_node 的四类关系审计与数据完善。

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

## What Changes

- 等 `reinitialize-alliance-economy-foundation` 完整 Deliver、PR merge 与 cleanup 后，从最新 `origin/main` 对最终 PostgreSQL schema/data 和重叠文件做只读 baseline/overlap audit；假设变化时回到 Review。
- 复用现有 graph projector，只做必要 R1 适配：节点读取 `entity_nodes`，通用关系读取 `entity_edges`，产业关系读取 `chain_node_relations`；移除已删除旧表和旧关系类型依赖。
- 只投影 active `alliance_org`、`economy`、`chain_node`。`entity_edges` 仅在 from/to 两端都属于这三类已投影节点集合时才投影；不因 `has_market` 等边扩张到 market/index/benchmark，不生成孤儿边。`chain_node_relations` 按批准类型投影。
- local Neo4j 是 disposable projection。cleanup 与 rebuild 分成两个独立 R3 层，以 PostgreSQL 为唯一事实源和重建来源，不创建 Neo4j backup/rollback。
- rebuild Query 分别输出 `alliance_org`、`economy`、`chain_node` 节点 counts，以及每种已投影关系的 type/count，并证明 missing、duplicate、orphan、legacy 均为 0。
- Package 2 的完成边界是当前 842 个 chain_node 全部完成 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on` 四类关系审计，不再限定一个产业方向或首批范围。
- “完整”表示 842/842 节点在四类关系上均有已审核状态：有强证据时形成批准关系，没有语义上应存在的关系或证据不足时记录不适用/证据不足及理由；不为达到数量而伪造边。
- 以可管理的批次从已有节点向下探索细分 chain_node。AI 生成与主对话 double-check 只是本次候选分析方法，不是产品能力；分批只是审阅手段，不降低全量完成标准。
- 全部候选和无关系结论经用户 Review 冻结后，先单独授权 R2 写 PostgreSQL 并 Query，再单独授权 R3 同步 Neo4j 并 Query；禁止 Neo4j 反写 PG。
- Package 2 默认是纯数据任务，复用现有 schema、写入能力和 projector；不预设源码、migration、repository/service、runner、dry-run/report framework 或新测试。只读能力审计发现硬缺口时，必须带证据回到 Review。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `neo4j-graph-projection-foundation`: 当前实体模型投影源、端点边界、local 分层清空/重建与分类 Query 验收。
- `industry-chain-node-foundation`: 当前 842 个 chain_node 的全量四类关系审计、细分节点完善与覆盖率完成标准。
- `entity-relationship-curation`: 全量候选分批 Review、PG-first 写入与 Neo4j 独立授权边界。

## Impact

- Proposal remediation 只修改本 change 的 OpenSpec artifacts。
- Apply 当前被 workflow schema blocker 和前置 change Deliver 双重阻断。
- 未来若两个阻断都解除且用户再批准 Apply，Package 1 只影响现有 graph projector 相关 Go 文件与必要 targeted tests；Package 2 默认不改源码。
- Non-goals：physical constraints、通用导入/审核平台、查询 API/推理、market/index/benchmark 扩投影、UAT/prod/shared、前端、事件、观测、股票推荐。
