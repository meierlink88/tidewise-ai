## Why

修复 PostgreSQL 到 Neo4j 的当前实体模型投影闭环，并按用户确认的业务目标完善当前 842 个 chain_node 的四类关系；产业研究与入库仍是数据工作，只有已被最新代码事实证明的 projector/写入入口缺口才进入最小 R1 修复。

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
| local-neo4j-foundation-cleanup | 1 | local | 1 | 仅清空 local Neo4j 的 Tidewise namespace 节点与关系，保留 database、约束、索引和配置 | PG 业务数据、UAT、prod、shared、其他 namespace | approved-disposable-recovery | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise namespace 节点与关系为零；database、约束、索引、配置和 PG 业务数据不变 | workflow schema blocker 未解决、环境不符、PG baseline 漂移、越过 namespace 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | 从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及符合端点边界的 entity_edges 与 chain_node_relations | market/index/benchmark、事件、观测、推理、未批准数据、UAT、prod、shared | approved-disposable-recovery | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分实体类型节点与分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | workflow schema blocker 未解决、PG baseline 漂移、projector failed/skipped 或完整性断言失败立即停止 |
| all-chain-node-relations-postgres-write | 2 | local | 1 | 仅向 chain_node_relations 写入 842/842 全量审核完成并经 Review 冻结的四类关系 final manifest | entity_nodes、profiles、external identifiers、physical constraints、未审核候选、UAT、prod、shared | backup | new:all-chain-node-relations-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 842/842 覆盖账本与 relation final manifest 已批准；端点、identity、scope、count、hash、schema 与范围外保护基线已冻结；backup 可恢复 | 全量批准关系写入结果、端点、tuple、orphan、范围外保护与幂等 Query 全部通过；节点主数据不变 | 覆盖不足 842/842、节点主数据变化、基线漂移、identity 冲突、部分写入、范围外变化或断言失败立即停止 |
| all-chain-node-relations-neo4j-sync | 2 | local | 2 | 仅从已验收 PG accepted baseline 同步 842 全量审核批准的四类 chain_node_relations，端点复用既有 chain_node | 节点主数据变更、physical constraints、反写 PG、查询 API、派生关系、UAT、prod、shared | approved-disposable-recovery | new:all-chain-node-relations-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG 写后 Query 已验收；accepted baseline 已冻结；取得单独 Neo4j R3 授权 | Neo4j 与 PG accepted baseline relation counts/type 一致；missing、duplicate、orphan、legacy 为零 | workflow schema blocker 未解决、PG baseline 漂移、未单独授权、同步或 Query 失败立即停止 |

### 当前 Workflow Schema Blocker

最新 `origin/main` 的 `.agents/openspec-workflow.md` 与 task-design lint 仍把 `approved-disposable-recovery` 限定为 local R2，无法诚实表达三个 local Neo4j R3 layer 的 disposable recovery。不得填写虚假的 `backup`，也不得在本 change 修改 workflow/lint；因此 explicit lint 预期仅因这一现行 schema 限制失败。`allow-local-neo4j-disposable-r3-recovery` 完整 Deliver 前，本 change 不得进入 Apply。

## Human Decision Register

| Gate | Package | Risk | Decision | Allows | Does Not Allow |
|---|---|---|---|---|---|
| G1 | 2 | R0 | 已确认本 change 完成 842/842 全量四类关系审核状态 | workflow blocker 解除后在本 change 内分批研究与 Review | 将剩余节点拆到后续 change、final 候选或任何写入 |
| G2 | 1 | R3 | `local-neo4j-foundation-cleanup` 授权 | 仅清空批准的 local Tidewise namespace | rebuild、PG 业务写、其他环境 |
| G3 | 1 | R3 | `local-neo4j-foundation-rebuild` 授权 | 仅从冻结 PG baseline 重建 local projection | PG 业务写、其他环境 |
| G4 | 2 | R0 | 审核并冻结 842/842 全量 final 关系候选与无关系理由 | 准备 chain_node_relations PG preflight | PG/Neo4j Write |
| G5 | 2 | R2 | `all-chain-node-relations-postgres-write` 授权 | 仅写全量批准数据并 Query | Neo4j sync、未审核数据 |
| G6 | 2 | R3 | `all-chain-node-relations-neo4j-sync` 授权 | 仅从 PG accepted baseline 同步并 Query | PG 反写、其他环境 |
| G7 | 3 | R1 | Apply-final Review | 通过后允许 Sync、Archive | PR merge、cleanup |
| G8 | 3 | R0 | Git completion | PR merge 与 Desktop-owned cleanup | 扩大 change scope |

## What Changes

- 已从最新 `origin/main=1733b8ba5fe85c17ed06f50412273ca5711b0d02` 完成无冲突合并与 overlap audit；三个前置 change 均已 Deliver/merge/cleanup。Apply 前只需复验是否漂移，不再保留失效的前置等待。
- 最小修复 `backend/internal/repositories/graph_projection.go`：删除对已移除 `sector_profiles`、`industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges` 的查询，节点改读 current `entity_nodes`，关系改读 `entity_edges` 与 `chain_node_relations`。
- 最小修复 `backend/internal/apps/graphprojection/mapping.go`：补齐 `is_subcategory_of`、`is_component_of`、`input_to` 映射并保留已有 `depends_on`，正确标识关系来源；只增加对应 repository/mapping/projector targeted tests。
- 只投影 active `alliance_org`、`economy`、`chain_node`；`entity_edges` 仅在两端都属于该集合时投影，绝不因关系扩张到 market/index/benchmark 或生成孤儿边。
- 复用已有 `graph-projector project-entities/rebuild-entities`、namespace 删除和 Neo4j upsert。cleanup 使用已存在的 namespace 删除语义并单独验收，rebuild 再单独授权；Neo4j 不备份。
- 本 change 必须只在当前 842 个既有 chain_node 之间完成四类关系全量分析与关联。每个节点都要形成已审核状态：已批准关系、不适用或证据不足及理由；分批只用于研究、double-check 和 Review，不构成 change 关闭边界。
- 不向下穿透、不研究或创建更细化 chain_node；不新增或修改 entity_nodes、chain_node_profiles、external identifiers 或节点 identity。现有节点无法表达的候选必须记录为不适用/证据不足，不得通过新建节点解决。
- AI 研究、主对话 double-check、候选证据整理均是一次性数据分析，不是系统能力。
- 关系 repository 已有通用只读 dry-run、事务批量写、端点/tuple/幂等校验；但 `cmd/entity-seed` 把关系入口锁死在历史 96 条 manifest/hash。若批准新的关系批次，Package 2 需要一个经 Review 的最小 R1 CLI 解锁，直接调用现有通用 batch 方法，不建设新平台。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `neo4j-graph-projection-foundation`: 让已有 projector 兼容当前实体模型，并明确 local 分层清空/重建与分类 Query。
- `industry-chain-node-foundation`: 在本 change 内完成 842/842 四类关系审核状态；批次仅作为研究和 Review 手段。
- `entity-relationship-curation`: 维持候选 Review、PG-first 与 Neo4j 独立授权；不把 AI 分析提升为产品能力。

## Impact

- 本轮只修改本 change 的 OpenSpec artifacts。
- 后续 Apply 的确定性源码范围是 `backend/internal/repositories/graph_projection.go`、`backend/internal/apps/graphprojection/mapping.go` 及其 targeted tests；projector/app/writer/CLI 默认复用。
- Package 2 只需要最小修改 `backend/cmd/entity-seed/main.go` 及现有 relation contract/tests，以解除历史 96 条 manifest 绑定并复用既有关系 batch；无其他 coding 需求。
- Non-goals：physical constraints、通用导入/审核平台、查询 API/推理、market/index/benchmark 扩投影、UAT/prod/shared、前端、事件、观测、股票推荐。
