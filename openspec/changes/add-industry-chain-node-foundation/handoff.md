# `add-industry-chain-node-foundation` 续作 Handoff

## 1. Change 与执行环境识别

| 项目 | 当前记录 |
|---|---|
| Change | `add-industry-chain-node-foundation` |
| Branch | `codex/add-industry-chain-node-foundation` |
| Desktop-managed worktree | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai` |
| Repo root | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai` |
| Execution thread | `019f5684-b83f-7371-bd35-d315ae5bb87f` |
| Project manager / source thread | `019f5477-445d-75d3-acf2-61a4fdd5b1d4` |
| Handoff 编写前 HEAD | `cc60dff3a81fd1e73d7e751f648df30169d976b9` (`cc60dff`) |
| 编写前远端状态 | `origin/codex/add-industry-chain-node-foundation` 与 `cc60dff` 一致 |

本文件提交后必须以实际 `git rev-parse HEAD` 和远端 branch 为准；不得因为本文记录了 `cc60dff` 而跳过实时 Git 核对。该 worktree 由 Codex Desktop 管理，不得手工删除或创建替代 worktree。

## 2. 目标与已批准架构

本 change 为全球事件驱动投研分析系统建立静态产业链骨架，定位为市场理解与决策辅助，不表达直接投资建议。

- 新增 `industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints` 4 张表，并扩展既有 `chain_node_profiles`。
- PostgreSQL 是实体、产业链、拓扑、跨实体关系与物理约束事实源；Neo4j 只是可从 PostgreSQL active/approved facts 重建的统一 `Entity` 图投影。
- Physical constraints 保持 PostgreSQL-only：未来推理先查 Neo4j chain/node/topology 路径，再通过 repository 批量补读约束；当前不得把 constraint 投影为 Neo4j 节点、关系或关系属性。
- 通用 observation governance、产业链 typed observations、metric definitions/bindings、revision/quality/idempotency 和 ingestion writer 不属于本 change，后续由独立 `add-industry-chain-observation-foundation` 设计。
- 静态主数据、未来 observation、future reasoning result 三层分离；推理结论不得回写静态主数据。

## 3. 截至 2026-07-13 的完成状态

- Explore、Propose、人工 Review、候选冻结已完成；proposal/design/delta specs/tasks 已建立并通过多轮 Review 修订。
- Domain validator、manifest/loader、Memory/Postgres repository、relationship policy、physical constraint query、Neo4j graph source/mapping/projector 已按 TDD 实现并有自动化测试。
- 正式 seed 与 review-only fixture 已版本化；candidate 数据没有混入可执行 approved manifest。
- Layer 1 migration 已完成：`000014_add_industry_chain_foundation.sql` 将 local `tidewise_local` 从 version 13 升至 14。
- Migration 前备份：`/private/tmp/tidewise_local_pre_000014_20260712T165928Z.dump`，959,309 bytes，SHA-256 `8c4a25c7c001aae6cbb35e6f4e7915454b4db1df3635ebeb9cc6931218709371`；`pg_restore --list` 成功读取 212 个 TOC entries。该文件位于本机临时目录，不应视为长期或生产备份。
- Layer 1 只读验收：4 张新表存在；`chain_node_profiles` 4 个新增列和 `generated_by_ai` 存在；32 个 PK/UNIQUE/FK/CHECK constraints 与 12 个 indexes 已核对。
- 4 张新表业务行均为 0。既有 `chain_node` 仍为 33/33 active；5 个复用节点 UUID/status 未变化。
- 新表为空，planned 27 个 membership ID 与 24 个 topology ID 冲突命中均为 0。
- OpenSpec 当前 tasks 状态为 19/24；`5.4a` Layer 1 已完成，`5.4b` Layer 2+ 仍未完成。

截至 2026-07-13，本 change **没有任何 Layer 2 或后续数据写入，也没有执行 Neo4j rebuild**。

## 4. 当前数据范围

| 数据族 | 当前范围 | 状态 |
|---|---:|---|
| Industry chains | 2 | AI 算力基础设施、半导体制造；正式 seed 已准备，尚未写入 PG |
| Unique chain nodes | 26 | 21 新增 + 5 复用；正式 seed/profile 已准备，尚未写入 PG |
| Memberships | 27 | AI 12 + 半导体 15；已批准并准备，尚未写入 PG |
| Canonical topology | 24 | AI 10 + 半导体 14；无 `substitutes_for` 推测，尚未写入 PG |
| Physical constraints | 15 | 全部为 review-only candidate，`generated_by_ai=true`，不得整体晋升或写入 |
| `mapped_to_sector` | 12 | 全部为 review-only candidate，尚未逐项批准，不得写入 |
| Economy relationships | 0 | 不得虚构 |
| Commodity relationships | 0 | 不得虚构 |
| Benchmark relationships | 0 | 不得虚构 |

## 5. 明日严格执行顺序

1. 对 Layer 2 chain/node master 做新的只读 preflight，核对实时 Git、migration version、4 张新表行数、33 个既有节点、5 个复用身份、23 个 planned 新实体 stable key/UUID 冲突以及 seed scope。
2. 向用户报告 preflight、精确预计影响、验证查询和停用边界，取得 **Layer 2 单独明确授权**。
3. 只写 chain/node master，然后立即只读验收；不得在同一授权中顺带写 membership。
4. Layer 2 验收通过后，对 Layer 3 membership 再做 preflight、Review、单独授权、Write、Query。
5. Layer 3 验收通过后，对 Layer 4 canonical topology 再做 preflight、Review、单独授权、Write、Query。
6. 15 条 physical constraints 按证据强弱逐项 Review；只有权威技术证据闭合且获得显式人工 approval gate 的条目才可进入 approved seed/write，未批准条目继续留在 review fixture。
7. 12 条 `mapped_to_sector` 按来源、端点和“分析映射而非身份/法定覆盖/影响方向”逐项 Review；不得用海外 market `COVERS_SECTOR` 中国 sector。
8. PostgreSQL 各层事实全部验收后，才可另行申请 Neo4j rebuild 授权；physical constraints 不投影。
9. Rebuild 后再单独进行只读 Query 验收，验证 2 chains、26 nodes、27 memberships、24 topology 和已批准跨实体路径。

每层都必须独立执行 `Review → Write → Query`；涉及图状态时才增加 `Rebuild → Query`。上一层授权、写入或只读验收不得推定下一层授权。

## 6. 明日第一步：Layer 2 预计影响与验收

基于 2026-07-13 的只读 preflight，Layer 2 最终表级预计影响为：

| 表 | Created | Updated | Unchanged / 保持不变 |
|---|---:|---:|---|
| `entity_nodes` | 23 | 5 | 其余既有 28 行保持；5 个复用实体只更新英文 aliases，身份不变 |
| `industry_chain_profiles` | 2 | 0 | 幂等重跑应 unchanged |
| `chain_node_profiles` | 21 | 5 | 幂等重跑应 unchanged |

23 个新 `entity_nodes` = 2 个 industry chains + 21 个新 chain nodes。5 个复用节点必须保持以下 UUID、stable key、状态和中文规范名不变，只允许按 approved seed 更新英文 aliases 与 profile 字段：

| Stable key | UUID | 中文规范名 |
|---|---|---|
| `chain_node:power_grid` | `3bd72a86-f31f-5f88-b0fc-22d4a6a4f784` | 电网 |
| `chain_node:data_center` | `255fc68e-9254-5016-b3a2-3c728a047ba9` | 数据中心 |
| `chain_node:gpu` | `8a21c5bf-454a-5ebc-9a60-e8b85b2a3696` | GPU |
| `chain_node:eda` | `4e9c51af-2487-5b98-b8bb-44b3fa086c9b` | EDA软件 |
| `chain_node:lithography_machine` | `dea1dd40-973d-5431-8612-a01a45395a0b` | 光刻机 |

Layer 2 写入后的只读验收至少覆盖：

- 2 个 `industry_chain` active + approved，21 个新增 node active，5 个复用 UUID/stable key/status/中文名不变。
- 26 个试点节点均有期望 aliases、definition、node category、unit of analysis 和 granularity。
- `industry_chain_memberships`、`industry_chain_topology_edges`、`industry_chain_physical_constraints` 仍为 0；不得出现 `mapped_to_sector` 新 edge。
- Seed report 与最终表级 created/updated/unchanged 对账；第二次幂等执行必须单独获得授权，不能仅为证明 unchanged 擅自重跑。

## 7. 禁止与未授权项

- 不得看到本 handoff 后直接开始 Layer 2 写入；必须先做只读 preflight、报告实时结果并取得用户明确授权。
- 不得将 15 条 physical constraint candidates 或 12 条 `mapped_to_sector` candidates 写入正式 seed/PG，也不得修改其审批状态。
- 不得创造 economy、commodity 或 benchmark 关系补齐空清单。
- 不得提前执行 membership、topology 或 Neo4j rebuild。
- 不得把 physical constraints 投影到 Neo4j。
- 不得启动 observation 或 event reasoning 实现来扩大当前 change scope。
- Apply 后第二次人工 Review 通过前，不得 Sync、Archive、创建 PR、merge 或 Deliver。

## 8. 关键文件索引

Repo root：`/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai`

| 用途 | 仓库相对路径 | 绝对路径 |
|---|---|---|
| 候选 Review | `openspec/changes/add-industry-chain-node-foundation/candidate-review.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/add-industry-chain-node-foundation/candidate-review.md` |
| 正式产业链 seed | `backend/data/entity_foundation/industry_chains_v1.json` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/backend/data/entity_foundation/industry_chains_v1.json` |
| Review-only fixture | `backend/data/entity_foundation/review/industry_chain_candidates_v1.json` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/backend/data/entity_foundation/review/industry_chain_candidates_v1.json` |
| Stateful 计划与证据 | `openspec/changes/add-industry-chain-node-foundation/stateful-execution-plan.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/add-industry-chain-node-foundation/stateful-execution-plan.md` |
| Tasks | `openspec/changes/add-industry-chain-node-foundation/tasks.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/add-industry-chain-node-foundation/tasks.md` |
| Migration | `backend/migrations/000014_add_industry_chain_foundation.sql` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/backend/migrations/000014_add_industry_chain_foundation.sql` |
| Design | `openspec/changes/add-industry-chain-node-foundation/design.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/add-industry-chain-node-foundation/design.md` |
| Proposal | `openspec/changes/add-industry-chain-node-foundation/proposal.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/add-industry-chain-node-foundation/proposal.md` |
| Delta specs | `openspec/changes/add-industry-chain-node-foundation/specs/` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/add-industry-chain-node-foundation/specs/` |
| 本 handoff | `openspec/changes/add-industry-chain-node-foundation/handoff.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/add-industry-chain-node-foundation/handoff.md` |

继续前还必须读取 repo root `AGENTS.md`、`.agents/skill-routing.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/backend-boundaries.md`、`.agents/testing-tdd.md` 和 `openspec/config.yaml`，以及当前 change 全部 context files。

## 9. 未解决与待 Review

- 15 条 physical constraints 的权威证据强弱分级、证据缺口关闭和逐项人工批准；不得整体晋升。
- 12 条 `mapped_to_sector` 的来源、端点和语义逐项批准。
- 后续独立 `add-industry-chain-observation-foundation`：observation governance、typed observation、产业链 domain metrics 与采集契约。
- 后续 event reasoning change：事件到 chain/node/sector 的证据化传导、动态观察验证、不确定性和证伪条件；不得在当前静态 foundation 中提前实现。

## 10. 明日恢复提示词

```text
继续 OpenSpec change add-industry-chain-node-foundation，使用当前 Codex Desktop-managed worktree 与 branch codex/add-industry-chain-node-foundation。先读取 AGENTS.md、.agents/skill-routing.md、.agents/openspec-workflow.md、.agents/git-workflow.md、.agents/backend-boundaries.md、.agents/testing-tdd.md、openspec/config.yaml、openspec-apply-change skill，以及当前 change 的 proposal/design/tasks/delta specs/candidate-review/stateful-execution-plan/handoff。

不要信任 handoff 中可能陈旧的 HEAD、migration version 或数据计数。先实时核对 git branch/HEAD/origin 同步与 scoped status，再仅以只读方式核对 local PostgreSQL：migration version、4 张产业链表、现有行数、33 个 chain_node、5 个复用 UUID/status、23 个 planned 新实体 stable key/UUID 冲突。不得执行 dbmigrate apply、entity-seed、INSERT/UPDATE/DELETE 或 Neo4j 操作。

把 Layer 2 chain/node master 的实时 preflight、精确 created/updated/unchanged、影响表、验收查询和停用方案报告给用户，等待 Layer 2 单独明确授权后才可写入。不得在 Layer 2 授权中顺带写 membership/topology；15 条 physical constraints 与12条 mapped_to_sector 仍是 candidate；economy/commodity/benchmark 为空；不得提前 Neo4j rebuild、Sync、Archive 或 PR。每层独立执行 Review → Write → Query，涉及图时再单独 Rebuild → Query。
```
