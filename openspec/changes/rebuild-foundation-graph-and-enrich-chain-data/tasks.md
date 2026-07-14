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

## 1. 基础图谱重建 Package

- [ ] 1.1 等待 `reinitialize-alliance-economy-foundation` 完成 Apply-final、Sync、Archive、Deliver、PR merge 与 cleanup；fetch 最新 `origin/main`，证明 archive commit、branch/worktree cleanup、当前 branch 更新和 clean workspace，未满足时 fail-closed 停止。
- [ ] 1.2 对最终 PostgreSQL schema/data 与 projector 做只读 baseline/overlap audit，冻结 active alliance/economy/chain_node、`entity_edges`、`chain_node_relations`、constraint、projection run 的 identity/scope/count/hash/schema；若最终前置输出改变假设，先修订 artifacts 并重新 Review。
- [ ] 1.3 按 TDD 先新增 repository query contract、mapper table-driven、fake writer projector 和 CLI 测试，覆盖删除表无引用、两个关系 source、四类新关系、PG 原始边方向、旧类型排除、inactive/缺失端点、cleanup/rebuild 顺序、幂等与 failed/skipped fail-closed。
- [ ] 1.4 最小适配 `backend/internal/repositories/`、`backend/internal/apps/graphprojection/` 与 `backend/cmd/graph-projector/`，只从当前 PostgreSQL 事实生成 Tidewise projection；运行 targeted tests、受影响 backend 完整 suite、只读 dry-run、diff/scope/secret 检查并提交基础图谱 R1 Review checkpoint。
- [ ] 1.5 准备 `local-neo4j-foundation-cleanup` 独立 R3 Review package，列出 disposable local 环境身份、Tidewise namespace、PG recovery baseline、精确清理范围、排除范围、before/after assertions 与停止条件；取得明确授权前不得执行。
- [ ] 1.6 获得 1.5 独立授权后只执行 local Tidewise Neo4j cleanup，立即 Query 节点/关系为零且 database/约束/索引/配置与 PostgreSQL 不变；提交证据并等待该层验收，失败时不恢复旧图。
- [ ] 1.7 cleanup Query 验收后准备 `local-neo4j-foundation-rebuild` 下一层独立 R3 Review package，复验 PG baseline identity/scope/count/hash/schema 与 projector dry-run；取得明确授权前不得重建。
- [ ] 1.8 获得 1.7 独立授权后从 PostgreSQL 全量投影批准范围，Query 对比节点/关系 counts、类型、缺失、重复、悬空、旧关系残留及 run failed/skipped；提交基础图谱重建验收 checkpoint。

## 2. 首批产业链数据 Package

- [ ] 2.1 提交 AI 算力基础设施、动力电池、光伏制造等有限候选的 entry nodes、包含/排除、投研价值、证据可得性、预计节点/关系/constraint 上限与停止条件，等待用户人工冻结首批 Spec scope；不得遍历全部 842 节点。
- [ ] 2.2 在已批准 scope 内生成第一遍研究候选，记录输入指纹、节点 reuse/merge/update/create、四类关系方向/mechanism/condition、constraint subject/type、来源、URL、verified time、证据、反例、置信度、冲突与 disposition。
- [ ] 2.3 执行第二遍独立 AI Review，逐项输出 approve/reject/blocked/merge，并生成总体 counts、机器校验、异常/冲突、宽边界、低置信度和用户指定项清单；人工批准前不得冻结可写 manifest。
- [ ] 2.4 对获批候选按 TDD 先新增 fixture、loader/validator、identity/tuple conflict、证据合同、repository fake/sqlmock、migration 静态校验、事务原子性、dry-run/report、幂等与范围外 842 节点保护测试；增加 typed 关系多跳路径测试，证明 `input_to` 顺向、`depends_on` 反向且分类/组成边不计入下游。
- [ ] 2.5 最小实现批准 scope 所需的 seed/repository/migration/runner 与机器可审阅 artifacts，冻结 final manifest/hash/counts；运行 targeted tests、受影响 backend 完整 suite、真实 local 只读 dry-run、diff/scope/secret 检查并提交首批数据 R1 Review checkpoint。
- [ ] 2.6 准备 `first-batch-postgres-write` 独立 R2 package，列出 local identity、可恢复 backup、精确 manifest、identity/scope/count/hash/schema、排除范围、before/after assertions、唯一命令和停止条件；取得明确授权前不得写 PostgreSQL。
- [ ] 2.7 获得 2.6 独立授权后执行 preflight、原子 PostgreSQL Write 与立即 Query/assert，验收 created/updated/unchanged/conflict、证据、重复、孤儿、范围外 checksum 与幂等；失败 rollback/停止，提交证据并等待 PG 层验收。
- [ ] 2.8 PG Query 验收后冻结 post-write accepted baseline，准备并取得 `first-batch-neo4j-sync` 单独 R3 授权；只读 PG 同步 local Neo4j，除 counts 一致性外从指定 chain_node 验收含 depth/path/evidence 的 typed 关系多跳下游，分类边不得误算；提交端到端 checkpoint，不得反写 PG 或扩张范围。

## 3. Apply-final、Sync、Archive、Deliver Package

- [ ] 3.1 复核 proposal/design/delta specs/tasks 与实现及实际有状态证据一致，运行受影响 backend module 完整 `go test -count=1 ./...`、共享 architecture/contract tests、OpenSpec strict/all 适用校验、diff/scope/secret 检查，提交 scoped Apply-final Review package 后停止等待人工 Review。
- [ ] 3.2 仅在 Apply-final 人工 Review 明确通过后同步 delta specs、验证主规格与实现一致、Archive change，并运行 `openspec validate --all`、新鲜 Git 检查和 scoped archive checkpoint。
- [ ] 3.3 仅在 Archive 完成后按 Git 门禁 push、创建/更新非完成态到完成态 PR、等待 merge，并按 Desktop 所有权顺序完成远端 branch、任务/worktree 与本地 branch cleanup；所有 Deliver 条件满足前不得声明 change 关闭。
