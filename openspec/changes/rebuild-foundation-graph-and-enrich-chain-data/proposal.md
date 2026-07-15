## Why

修复 PostgreSQL 到 Neo4j 的当前实体模型投影闭环，并按用户确认的业务目标完善当前 842 个 chain_node 的四类关系；产业研究与入库仍是数据工作，只有已被最新代码事实证明的 projector/写入入口缺口才进入最小 R1 修复。

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

## What Changes

- 已从最新 `origin/main=007f6efdaf5bbe5b880a9adfc5c502e0e39849f2` 完成无冲突 Active Change Adoption；所有前置 change 均已 Deliver/merge/cleanup。Apply 前只需复验是否漂移，不再保留失效的前置等待。
- 最小修复 `backend/internal/repositories/graph_projection.go`：删除对已移除 `sector_profiles`、`industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges` 的查询，节点改读 current `entity_nodes`，关系改读 `entity_edges` 与 `chain_node_relations`。
- 最小修复 `backend/internal/apps/graphprojection/mapping.go`：补齐 `is_subcategory_of`、`is_component_of`、`input_to` 映射并保留已有 `depends_on`，正确标识关系来源；只增加对应 repository/mapping/projector targeted tests。
- 只投影 active `alliance_org`、`economy`、`chain_node`；`entity_edges` 仅在两端都属于该集合时投影，绝不因关系扩张到 market/index/benchmark 或生成孤儿边。
- 复用已有 `graph-projector project-entities/rebuild-entities`、namespace 删除和 Neo4j upsert。cleanup 使用已存在的 namespace 删除语义并单独验收，rebuild 再单独授权；Neo4j 不备份。
- 本 change 只在当前 842 个既有 chain_node 之间完善四类关系。覆盖表示每个节点都参加候选发现与双遍检查；无直接或间接事实时允许无边，不再要求每个 node×relation_type 强制形成三态记录。
- 不向下穿透、不研究或创建更细化 chain_node；不新增或修改 entity_nodes、chain_node_profiles、external identifiers 或节点 identity。现有节点无法表达的候选必须记录为不适用/证据不足，不得通过新建节点解决。
- 现有 100 条关系作为 Tier 1 accepted baseline 逐行保留；新增候选只排除重复 tuple，同一端点可继续产生其他机制或类型的边。`input_to` 只接受 A 的实际产出被消耗、转化、嵌入，或作为明确服务输出沿可解释路径进入 B；间接边必须列出中间路径。设备、工具、软件能力或基础设施仅因使 B 能生产/运行不得登记为 `input_to`，也不得机械改成 `depends_on`。
- 新增证据分 Tier 1 usable-map 与 Tier 2 AI knowledge。Tier 1 来源必须逐边蕴含 relation type、方向与具体机制；产业相关性不等于来源支持。两遍独立审查分别保存理由、类型、方向、机制、路径、条件、具体反例和 disposition，任一不一致即阻断。AI 研究与 double-check 是一次性数据分析，不是系统能力。
- `reviews/all-chain-node-relations-neo4j-sync-r3.md` 的 4-edge sync 意图已 superseded，明确不可执行；未来只基于 additive PG 写后最终 accepted baseline 重新准备独立 R3 包。
- 关系 repository 已有通用只读 dry-run、事务批量写、端点/tuple/幂等校验；但 `cmd/entity-seed` 把关系入口锁死在历史 96 条 manifest/hash。若批准新的关系批次，Package 2 需要一个经 Review 的最小 R1 CLI 解锁，直接调用现有通用 batch 方法，不建设新平台。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `neo4j-graph-projection-foundation`: 让已有 projector 兼容当前实体模型，并明确 local 分层清空/重建与分类 Query。
- `industry-chain-node-foundation`: 在 842 个既有节点间完成 usable-map 候选发现与双遍检查，允许无事实节点保持无边。
- `entity-relationship-curation`: 维持候选 Review、PG-first 与 Neo4j 独立授权；不把 AI 分析提升为产品能力。

## Impact

- 本轮只修改本 change 的 OpenSpec artifacts。
- 后续 Apply 的确定性源码范围是 `backend/internal/repositories/graph_projection.go`、`backend/internal/apps/graphprojection/mapping.go` 及其 targeted tests；projector/app/writer/CLI 默认复用。
- 第一批 100 条所需 manifest 解锁已完成。additive final manifest 若改变固定 path/hash/count，未来仅允许在独立 Review 后最小更新现有 contract/tests；无其他 coding 需求。
- Non-goals：physical constraints、通用导入/审核平台、查询 API/推理、market/index/benchmark 扩投影、UAT/prod/shared、前端、事件、观测、股票推荐。
