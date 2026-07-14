## Why

`refactor-industry-chain-node-foundation` 已将 PostgreSQL 产业事实收敛为 842 个 `chain_node`、95 条 `is_subcategory_of`、1 条 `is_component_of` 与新的 physical constraint 契约；当前 `input_to`、`depends_on` 和 constraint 数据均为 0，因此只能形成分类骨架，不能生成完整产业全景或下游穿透。与此同时，graph projector 仍读取已删除的旧 sector/industry-chain 表，也尚未投影新的产业节点关系。需要先恢复“PostgreSQL 唯一事实源、Neo4j 可重建投影”的闭环，再以有限首批范围补充高价值产业节点、关系与强证据物理约束，避免把 842 节点长期治理塞入一个无法关闭的 change。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 基础图谱重建 Review | R3 | yes | R3_OPERATION | 等待前置 change 完整 Deliver 后重做只读 baseline/overlap audit；允许必要的最小 projector R1 适配，并在逐层独立授权后仅清理和重建 local Neo4j |
| 2 | 首批产业链数据 Review | R3 | yes | SPEC_SEMANTICS | 人工确定有限首批范围并审阅双遍 AI 候选；逐层独立授权后先写 local PostgreSQL 并 Query，再同步 local Neo4j 并 Query |
| 3 | Apply-final Review | R1 | yes | APPLY_FINAL | 汇总 scoped diff 与新鲜验证，获批后才允许 Sync、Archive，并按 Git 门禁继续 Deliver |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 4 |
| checkpoints | 3 |
| full_test_runs | 2 |
| continuous_automation_scope | packages:none |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-neo4j-foundation-cleanup | 1 | local | 1 | 仅删除 local Neo4j 的 Tidewise 投影节点与关系，保留 database、约束、索引和连接配置 | PostgreSQL、UAT、prod、shared、其他 namespace | backup | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | reinitialize change 已完整 Deliver；最新 origin/main baseline/overlap audit 通过；PG 投影输入 identity、scope、count、hash、schema 已冻结；环境为用户批准的 disposable local Neo4j | Tidewise 投影节点和关系为零；database、约束、索引、配置不变；PostgreSQL 不变 | 环境身份不符、前置未 Deliver、PG baseline 漂移、非 Tidewise namespace 进入范围、清理失败或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | 从已验收 PostgreSQL 全量投影 active alliance、economy、chain_node 及 PG 已批准关系 | observation、事件推理、未批准候选、UAT、prod、shared | backup | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 cleanup 后 PG baseline identity、scope、count、hash、schema 未漂移；projector targeted tests 与只读 dry-run 通过 | Neo4j 节点、关系、类型与 PG 投影输入逐项一致；缺失、重复、悬空和旧关系类型为零；Query 验收通过 | PG baseline 漂移、projector failed/skipped、counts/hash/schema 不符、超时或 Query 失败立即停止 |
| first-batch-postgres-write | 2 | local | 1 | 仅写入人工批准的有限首批 chain_node、四类静态关系与强证据 physical constraints | 其余 842 节点、事件/推理/观测、股票推荐、UAT、prod、shared | backup | new:first-batch-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | 首批 Spec gate 与双遍候选 Review 已批准；backup 可恢复；identity、scope、count、hash、schema、before assertions 与写入 manifest 已冻结 | PostgreSQL created/updated/unchanged/conflict、完整性、证据、反例、置信度、孤儿、重复与幂等 Query 全部通过 | 候选未批准、范围或 manifest 漂移、backup 失效、冲突、部分写入、断言或幂等失败立即停止 |
| first-batch-neo4j-sync | 2 | local | 2 | 仅从已验收 PostgreSQL 同步首批新增或更新的 chain_node、批准关系与明确纳入投影的 constraint 表达 | Neo4j 反写 PostgreSQL、未批准候选、UAT、prod、shared | backup | new:first-batch-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PostgreSQL 写后 Query 已独立验收；冻结 post-write PG identity、scope、count、hash、schema；取得单独 Neo4j R3 授权 | Neo4j 首批节点和关系与 PG accepted baseline 一致；无额外事实、重复、悬空或旧投影残留；Query 通过 | PG accepted baseline 漂移、环境不符、未单独授权、projector failed/skipped、counts/hash/schema 或 Query 不符立即停止 |

## What Changes

- 在 `reinitialize-alliance-economy-foundation` 完整完成 Apply-final、Sync、Archive、Deliver、PR merge 与 cleanup 后，从最新 `origin/main` 重新执行 PostgreSQL schema/data baseline、projector gap 与跨 change overlap audit；其最终 alliance/economy schema 和数据范围优先于本 Proposal 的当前快照。
- 测试先行修复现有 graph projector 对已删除旧表和旧关系类型的依赖；只做投影闭环所需的最小 R1 适配，使 active `alliance_org`、`economy`、`chain_node`、通用 `entity_edges` 和 `chain_node_relations` 都从 PostgreSQL 读取并映射。
- local 产品 1.0 探索期 Neo4j 被明确视为 disposable projection。取得独立 R3 授权后可不做 Neo4j backup/rollback，直接清理 Tidewise local 投影并以已冻结 PostgreSQL baseline 重建；PostgreSQL baseline 是确定性恢复来源。该豁免不适用于 UAT、prod、shared 或其他 namespace。
- 以 Proposal 人工 Spec gate 限定一个有限、代表性、可关闭的首批产业链范围；本 Proposal 只给出选择标准和推荐候选方向，不擅自定稿，也不遍历或逐项治理全部 842 节点。
- canonical graph 仍只使用 chain_node 与四类关系；Neo4j 保留 PostgreSQL 原始 typed 边，并支持基于 `input_to` 与 `depends_on` 的多跳下游查询，不把分类/组成关系误算为上下游事实。
- 对首批范围使用 AI 研究和双遍审核，提出细分 `chain_node`、`is_subcategory_of`、`is_component_of`、`input_to`、`depends_on` 及强证据 physical constraints；候选记录来源、证据、反例、条件、置信度、冲突和 disposition。
- 首批候选获批后，取得独立 R2 授权先写 PostgreSQL 并 Query 验收；再以 PostgreSQL accepted baseline 为唯一输入，取得单独 R3 授权同步 Neo4j 并 Query。任一授权不得推定另一层、环境或范围。
- Apply-final 只在前两 package 完成且验证通过后汇总；获人工 Review 后才允许 Sync、Archive、Deliver。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `neo4j-graph-projection-foundation`: 增加新 chain-node 模型的完整 PostgreSQL 投影来源、local disposable projection 的受控清理/重建边界、typed 关系多跳查询，以及 PG/Neo4j 一致性 Query 验收。
- `industry-chain-node-foundation`: 增加有限首批范围的人工 Spec gate、细分节点/四类关系/强证据 physical constraint 的候选要求，以及批次不得扩张为 842 节点全量治理的边界。
- `entity-relationship-curation`: 增加首批产业关系的双遍研究审阅、先 PostgreSQL 后 Neo4j 的分层授权顺序及各层 fail-closed 验收。
- `entity-foundation-seeds`: 增加有限首批新增或修订 chain_node 的候选、identity、definition/boundary、证据与幂等写入边界，不改变 842 节点既有基线的全量治理责任。

## Impact

- 影响 `tidewise-ai/backend/internal/apps/graphprojection/`、`backend/internal/repositories/`、`backend/cmd/graph-projector/` 及相关 Go 测试；首批数据若获批，可能影响 `backend/internal/apps/entityfoundation/seed/`、`backend/data/entity_foundation/`、必要的增量 migration 与测试。
- 影响 local PostgreSQL 的已批准首批主数据与 local Neo4j 的 Tidewise 投影；PostgreSQL 始终是唯一事实源，Neo4j 不得反写 PostgreSQL。
- Proposal/Review 阶段只修改本 change 的 OpenSpec artifacts，不修改源码、不访问或写入数据库/Neo4j。
- 不修改 `prototype/` 或项目外 `doc/`；不包含事件提取/推理、观测数据、股票推荐、UAT/prod/shared、前端或完成态 PR。
