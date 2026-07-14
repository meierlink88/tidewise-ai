## Why

修复 PostgreSQL 到 Neo4j 的当前实体模型投影闭环，并用一个审核通过的有限产业链数据批次验证 PG-first 端到端流程。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 基础投影 R1 连续执行；local Neo4j cleanup 与 rebuild 分层授权 | R3 | yes | R3_OPERATION | 等待前置 change 完整 Deliver 后审计最终基线；只允许最小 projector 适配及两个独立 local Neo4j R3 层 |
| 2 | 首批业务范围 Review；local PostgreSQL R2 Write；local Neo4j R3 sync | R3 | yes | SPEC_SEMANTICS | 仅处理审核通过的 AI 服务器算力基础设施批次，复用现有 schema/seed/repository/projector，先 PG 后 Neo4j |
| 3 | Apply-final Review 与 Git completion | R1 | yes | APPLY_FINAL | 汇总 scoped diff 与验证；获批后才允许 Sync、Archive、PR merge 与 Desktop cleanup |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 4 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

`human_gates=3` 是与 Gate Map 三个 `Human=yes` package 行一致的机器计数。Human Decision Register 另行记录 package 内 G1–G7 七个独立人工决策点；它不改写 Complexity Budget。普通 R1 测试、实现、dry read、修复、commit 与 push 在已批准 scope 内连续推进，不因 checkbox 人工停顿。

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-neo4j-foundation-cleanup | 1 | local | 1 | 仅清空 local Neo4j 的 Tidewise 投影节点与关系，保留 database、约束、索引和配置 | PostgreSQL、UAT、prod、shared、其他 namespace | backup | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 前置 change 已完整 Deliver；PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise 投影节点与关系为零；database、约束、索引、配置和 PostgreSQL 不变 | 环境不符、PG baseline 漂移、越过 namespace、cleanup 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | 从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及已批准关系 | 事件、观测、推理、未批准数据、UAT、prod、shared | backup | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分类型节点及分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | PG baseline 漂移、projector failed/skipped、counts/type 或完整性断言失败立即停止 |
| first-batch-postgres-write | 2 | local | 1 | 仅写入审核通过的 AI 服务器算力基础设施 change-specific manifest 中的 chain_node 与四类静态关系 | physical constraints、范围外 842 节点、通用导入、UAT、prod、shared | backup | new:first-batch-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 业务范围和 final manifest 已批准；local PG backup 可恢复；identity、scope、count、hash、schema 与范围外保护基线已冻结 | manifest 写入结果、端点、tuple、orphan、范围外保护与幂等 Query 全部通过 | manifest/基线漂移、identity 冲突、部分写入、范围外变化或断言失败立即停止 |
| first-batch-neo4j-sync | 2 | local | 2 | 仅从已验收 PG accepted baseline 同步首批 chain_node 与批准关系 | physical constraints、反写 PG、查询 API、派生关系、UAT、prod、shared | backup | new:first-batch-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG 写后 Query 已验收；post-write PG identity、scope、count、hash、schema 已冻结；取得单独 Neo4j R3 授权 | Neo4j 与 PG accepted baseline counts/type 一致；missing、duplicate、orphan、legacy 为零；typed Cypher 多跳验收通过 | PG accepted baseline 漂移、未单独授权、同步失败或 Query 断言失败立即停止 |

三个 Neo4j R3 layer 的 `Recovery Evidence=backup` 仅是机器 schema 要求的字面值，其 recovery evidence 始终是冻结且验收的 PostgreSQL projection/accepted baseline。不创建 Neo4j backup/rollback；Neo4j 仍是 disposable projection。

## Human Decision Register

| Gate | Package | Risk | Decision | Allows | Does Not Allow |
|---|---|---|---|---|---|
| G1 | 2 | R0 | Proposal 业务 Review 已批准唯一首批方向及边界 | 在前置依赖满足后按批准 scope 连续执行 R1 准备 | 任何 PG/Neo4j Write；前置 change Deliver 前启动 R1 |
| G2 | 1 | R3 | `local-neo4j-foundation-cleanup` 授权 | 仅清空批准的 local Tidewise projection | rebuild、PG Write、其他环境 |
| G3 | 1 | R3 | `local-neo4j-foundation-rebuild` 授权 | 仅从冻结 PG baseline 重建 local projection | PG Write、其他环境 |
| G4 | 2 | R2 | `first-batch-postgres-write` 授权 | 仅写批准 manifest 并 Query | Neo4j sync、范围扩张 |
| G5 | 2 | R3 | `first-batch-neo4j-sync` 授权 | 仅从 PG accepted baseline 同步并 Query | PG 反写、其他环境 |
| G6 | 3 | R1 | Apply-final Review | 通过后允许 Sync、Archive | PR merge、cleanup |
| G7 | 3 | R0 | Git completion | PR merge 与 Desktop-owned cleanup | 扩大 change scope |

## What Changes

- 等 `reinitialize-alliance-economy-foundation` 完整 Deliver、PR merge 与 cleanup 后，从最新 `origin/main` 对最终 PostgreSQL schema/data 和重叠文件做只读 baseline/overlap audit；若实际输出改变本 Proposal 假设，先回到 Review。
- 复用现有 graph projector，只做必要 R1 适配：节点读取 `entity_nodes`，通用关系读取 `entity_edges`，产业关系读取 `chain_node_relations`；移除已删除旧表和旧关系类型依赖。
- 只为当前 query、mapper、projector、CLI 契约增加 targeted tests；不建设新的 graph framework。
- local Neo4j 是 disposable projection。cleanup 与 rebuild 分成两个独立 R3 层，均以已验收 PG projection baseline 为恢复依据，不创建 Neo4j backup/rollback。
- rebuild Query 分别输出 `alliance_org`、`economy`、`chain_node` 节点 counts，以及每种已批准关系的 type/count，并证明 missing、duplicate、orphan、legacy 均为 0。
- 唯一推荐首批方向为“AI 服务器算力基础设施”，entry node 为“AI服务器”。范围限制为最多两跳直接上游硬件与一跳直接部署节点，建议上限 18 个 chain_node、28 条静态关系。
- 首批包含 AI 加速芯片/加速卡、HBM、高速互连与光模块、服务器电源、液冷系统、AI服务器、AI算力集群等直接硬件节点；排除完整半导体制造设备/材料链、通用云与 IDC 运营、模型/软件/应用、公司/证券、physical constraints。用户已在 Proposal Review 批准该方向与边界；仍必须等待前置 change 完整 Deliver，不得提前启动 Package 1 R1。
- AI 双遍 Review 仅作为生成和复核 change-specific manifest 的数据分析方法，不建设产品能力、policy engine 或通用候选平台。
- Package 2 默认复用现有 schema、entity-seed/repository 和 graph-projector；不预授权 migration、新 repository/service、runner framework、dry-run/report framework。Apply 只读 audit 如发现硬缺口，必须带证据回到 Review。
- 获批 manifest 只执行最小只读 preflight、原子 PG Write、写后 Query 与幂等复验；PG R2 验收后才可单独申请 Neo4j R3 sync。
- Neo4j 多跳只作为验收 Cypher：`input_to` 顺向、`depends_on` 反向，`is_subcategory_of`/`is_component_of` 不计入上下游；不开发查询 API、服务、推理引擎或派生关系。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `neo4j-graph-projection-foundation`: 当前实体模型投影、local 分层重建、PG-first 首批同步与 typed Cypher 验收。
- `industry-chain-node-foundation`: 单一有限 AI 服务器算力基础设施批次的节点/关系范围与关闭边界。
- `entity-relationship-curation`: 双遍候选分析方法、change-specific manifest 与 PG/Neo4j 独立授权。
- `entity-foundation-seeds`: 复用现有 seed/repository 完成有限 manifest 写入与范围保护。

## Impact

- Proposal Review 阶段只修改本 change 的 OpenSpec artifacts。
- Apply 若获后续批准，Package 1 仅影响现有 graph projector 相关 Go 文件与 targeted tests；Package 2 仅影响 change-specific 数据 manifest 和复用的既有 seed 写入入口。
- 不修改 `prototype/` 或项目外 `doc/`，不创建新 capability。
- Non-goals：physical constraints、通用导入平台、查询 API/推理引擎、842 节点全量治理、UAT/prod/shared、前端、事件提取/推理、观测数据、股票推荐。
