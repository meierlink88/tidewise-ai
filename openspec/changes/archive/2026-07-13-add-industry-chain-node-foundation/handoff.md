# `add-industry-chain-node-foundation` 续作 Handoff

> **状态：SUPERSEDED / 已无同步归档。** 用户已取消sector逻辑实体、`industry_chain`容器和独立membership，后续改为粗细粒度统一`chain_node` + 单一typed edge，且当前不建立Neo4j。本change已通过`openspec archive --skip-specs`移动到`openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation`，`specsUpdated=false`；本文件只保存已发生状态与forward migration输入，不再安排旧模型续作。

## 1. Change 与执行环境识别

| 项目 | 当前记录 |
|---|---|
| Change | `add-industry-chain-node-foundation` |
| Branch | `codex/add-industry-chain-node-foundation` |
| Desktop-managed worktree | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai` |
| Repo root | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai` |
| Execution thread | `019f5684-b83f-7371-bd35-d315ae5bb87f` |
| Project manager / source thread | `019f5477-445d-75d3-acf2-61a4fdd5b1d4` |
| 原始 Handoff 编写前 HEAD | `cc60dff3a81fd1e73d7e751f648df30169d976b9` (`cc60dff`) |
| Layer 2 执行基线 | `5fc8a90c98c809259c2ddba6152dd721502cc83f` (`5fc8a90`)；执行前 local/remote 一致 |
| Layer 6 执行基线 | `0391999806aab54c7136bd8a479b037c29fb1bcb` (`0391999`)；执行前 local/remote 一致 |
| Neo4j Rebuild 执行基线 | `edc56950b4f9302cd7ef8d778e12068779aee702` (`edc5695`)；执行前 local/remote 一致 |

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
- Layer 2 已通过独立授权执行：只运行一次 `entity-seed -apply-scope industry-chain-master`，没有运行无 scope seed或幂等重跑。Report operation为46 created/15 updated，最终表级为 `entity_nodes 23 created +5 updated`、`industry_chain_profiles 2 created`、`chain_node_profiles 21 created +5 updated`。
- Layer 2 写后：`entity_nodes=634`、chain node=54/54 active、2 chain active+approved、26 pilot node aliases/profile完整；5个复用节点 UUID/status/中文名未变化。
- Layer 2写后当时membership、topology、physical constraint均为0，planned 27 membership ID与24 topology ID冲突命中为0；该历史基线随后分别用于Layer 3/4门禁。
- Pre-Layer2备份位于 `/private/tmp/tidewise_local_pre_layer2_20260713T015540Z.dump`，979,104 bytes，SHA-256 `2a547922d53a10b9d1bbf610e3aa3d14fd3b274f81b72270695e28ddf4467268`，242个TOC entries可读。
- Layer 3只读preflight确认27个membership ID/tuple唯一且数据库冲突为0，AI=12、半导体=15、advanced_packaging共享两条。已按TDD实现 `industry-chain-membership` scope并验证只生成27 memberships batch；尚未执行DML。
- Layer 3已通过独立授权执行：只运行一次`entity-seed -apply-scope industry-chain-membership`，report为created27/updated0/unchanged0。写后27/27 active且ID/tuple唯一，AI=12、半导体=15、advanced_packaging属于两链；topology/constraint仍0，无关表计数不变。
- Pre-Layer3备份：`/private/tmp/tidewise_local_pre_layer3_20260713T021205Z.dump`，983,891 bytes，SHA-256 `e33834390fadbcbc1273a3a1818ce835d12f61295e62efd23f21e0f8dd900b80`，242个TOC entries可读。
- Layer 4已通过独立授权执行：写前备份后只运行一次`entity-seed -apply-scope industry-chain-topology`，report为created24/updated0/unchanged0。写后24/24 active，AI10/半导体14，ID/tuple唯一，无self/substitutes/reverse duplicate/无效端点；membership仍27/27 active，constraint与无关表不变。Pre-Layer4备份为`/private/tmp/tidewise_local_pre_layer4_20260713T024610Z.dump`，986,070 bytes，SHA-256 `abe475308950b05b6a275ca4388b0bd0d28b39c3d8c751350c49fef2f198fe95`，253个TOC entries可读。

截至 2026-07-13，本 change **已完成Layer 2 master、Layer 3 membership、Layer 4 topology、首批4条physical constraint及首批6条`mapped_to_sector`写入与只读验收；取消指令到达前还完成了一次标准Neo4j rebuild与Query**。这些结果现均为superseded迁移输入，不再代表目标架构。

## 4. 当前数据范围

| 数据族 | 当前范围 | 状态 |
|---|---:|---|
| Industry chains | 2 | AI 算力基础设施、半导体制造；已写入 PG并验收 active+approved |
| Unique chain nodes | 26 | 21新增 +5复用；已写入/更新 PG并完成只读验收 |
| Memberships | 27 | 已写入PG并验收27/27 active；AI 12 + 半导体15 |
| Canonical topology | 24 | 已写入PG并验收24/24 active；AI 10 + 半导体14，无`substitutes_for`推测 |
| Physical constraints | 4已写入 +11 review-only | 首批4条已写入PG并验收active+approved、AI provenance与P2/P6正确；其余9条需补证、2条删除或改写 |
| `mapped_to_sector` | 6已写入 +6 review-only | 首批6条已写入PG并验收6/6 active、composite curation provenance正确；其余2条需补证、4条删除或改写 |
| Economy relationships | 0 | 不得虚构 |
| Commodity relationships | 0 | 不得虚构 |
| Benchmark relationships | 0 | 不得虚构 |

Layer 5首批4条已在独立授权与备份后仅执行一次`industry-chain-physical-constraint` scope：report created4/updated0/unchanged0，写后4/4 active approved且AI provenance、P2/P6和subject有效；review fixture剩余11条。

Layer 6只读Review已完成：实时基线仍为2 chains、26 unique nodes、27 active memberships、24 active topology、4 active approved constraints，`entity_edges=383`、`sector_source_mappings=89`；12条`mapped_to_sector`候选的严格口径为直接证据闭合0、语义认可但provenance须校正6、需补证2、删除或改写4。全部仍在review-only fixture，本轮未晋级、未写PG/Neo4j；详见`mapped-to-sector-review.md`。

用户随后批准首批6条进入正式seed及PG Write：在独立备份和实时preflight后仅执行一次`industry-chain-sector-mapping` scope，report为created6/updated0/unchanged0且FinalTableImpact仅`entity_edges`。写后`entity_edges=389`、`mapped_to_sector=6/6 active`，端点、identity和composite curation provenance逐项匹配；其他关系类型及产业链表计数不变。未访问或重建Neo4j。

Pre-mapping备份：`/private/tmp/tidewise_local_pre_mapping_20260713T050005Z.dump`，990,107 bytes，SHA-256 `26e62a90ff6fbafe6e3de44f5aba43947446e41daee3c62e2314c821dbea7bcb`；`pg_restore --list`成功读取253行清单。

最终Neo4j层已获独立授权并仅执行一次标准`graph-projector rebuild-entities`。Run `362f4bbe-90be-5e54-baf6-20f9d0d1ae6d`为succeeded，source/projected=`1014/1014`、skipped/failed=`0/0`；投影从陈旧基线551节点/383关系重建为`tidewise` namespace下574个统一`Entity`和440条关系。2 chains、26 pilot nodes、27 membership（12/15）、24 topology（10/14）及6条approved mapping均通过只读Query；constraint、candidate mapping、dangling、duplicate和双标签投影均为0。PG事实计数保持`574/389/6/27/24/4`。

## 5. 后续严格执行顺序

1. Layer 2 已完成，不得未经独立授权做幂等重跑。
2. Layer 3已完成，不得未经独立授权幂等重跑。
3. Layer 4已完成，不得未经独立授权幂等重跑。
4. 首批4条physical constraints已完成Write与Query验收，不得幂等重跑；其余11条继续留在review fixture并逐项补证/改写。
5. 首批6条`mapped_to_sector`已完成PG Write与Query验收，不得未经独立授权幂等重跑；其余6条仍review-only。不得用海外 market `COVERS_SECTOR` 中国 sector。
6. 最终Neo4j Rebuild与Query已验收，不得未经独立授权重跑；physical constraints不投影。
7. 用户已批准superseded closeout：跳过spec sync归档并创建Review PR；不得继续旧模型、创建新change、自行merge或清理branch/worktree。

每层都必须独立执行 `Review → Write → Query`；涉及图状态时才增加 `Rebuild → Query`。上一层授权、写入或只读验收不得推定下一层授权。

## 6. 已完成的 Layer 2 影响与验收

2026-07-13 Layer 2 最终表级实际影响为：

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

- 不得幂等重跑Layer 2、Layer 3或Layer 4。
- 不得幂等重跑首批4条physical constraint或首批6条sector mapping scope；不得写其余11条constraint或6条review-only sector mapping。
- 不得创造 economy、commodity 或 benchmark 关系补齐空清单。
- 不得写入未批准candidate或重跑已验收的Neo4j rebuild。
- 不得把 physical constraints 投影到 Neo4j。
- 不得启动 observation、event reasoning或新统一节点change来扩大当前任务。
- 本次只允许`--skip-specs`归档、验证、scoped commit/push和ready-for-review PR；不得Sync、merge、数据清理或worktree/branch cleanup。

## 8. 关键文件索引

Repo root：`/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai`

| 用途 | 仓库相对路径 | 绝对路径 |
|---|---|---|
| 候选 Review | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/candidate-review.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/candidate-review.md` |
| Physical constraint逐条证据审查 | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/physical-constraint-review.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/physical-constraint-review.md` |
| `mapped_to_sector`逐条证据审查 | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/mapped-to-sector-review.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/mapped-to-sector-review.md` |
| 正式产业链 seed | `backend/data/entity_foundation/industry_chains_v1.json` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/backend/data/entity_foundation/industry_chains_v1.json` |
| Review-only fixture | `backend/data/entity_foundation/review/industry_chain_candidates_v1.json` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/backend/data/entity_foundation/review/industry_chain_candidates_v1.json` |
| Stateful 计划与证据 | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/stateful-execution-plan.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/stateful-execution-plan.md` |
| Tasks | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/tasks.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/tasks.md` |
| Migration | `backend/migrations/000014_add_industry_chain_foundation.sql` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/backend/migrations/000014_add_industry_chain_foundation.sql` |
| Design | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/design.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/design.md` |
| Proposal | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/proposal.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/proposal.md` |
| Delta specs | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/specs/` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/specs/` |
| 本 handoff | `openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/handoff.md` | `/Users/meierlink/.codex/worktrees/cb4e/tidewise-ai/openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation/handoff.md` |

继续前还必须读取 repo root `AGENTS.md`、`.agents/skill-routing.md`、`.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/backend-boundaries.md`、`.agents/testing-tdd.md` 和 `openspec/config.yaml`，以及当前 change 全部 context files。

## 9. 已取消旧目标与后续输入

- 其余11条physical constraints和6条sector mapping候选全部随旧模型取消，不再补证或晋级。
- 已写PG事实与已完成Neo4j旧投影保持原状，只作为下一change的forward migration输入。
- 后续架构必须重新设计统一chain_node粒度、单一typed edge、旧表/旧entity/旧edge的前向迁移与停用顺序；当前任务不创建该change。

## 10. 明日恢复提示词

```text
复核已归档的superseded change add-industry-chain-node-foundation，使用当前 Codex Desktop-managed worktree 与 branch codex/add-industry-chain-node-foundation。先读取AGENTS及相关规则，并读取openspec/changes/archive/2026-07-13-add-industry-chain-node-foundation下的归档artifacts。

不要信任handoff中可能陈旧的HEAD、migration version或数据计数。先实时核对Git、PG与Neo4j，再只读确认PG为574 active projection entities、389 entity edges、27 memberships、24 topology、4 approved constraints、6 mapped_to_sector，Neo4j为574个统一Entity和440条关系。不得执行dbmigrate apply、entity-seed、INSERT/UPDATE/DELETE或Neo4j写操作。

该change已被统一产业链节点架构替代。旧PG facts和旧Neo4j projection保持原状且不得重跑或清理；其余候选全部取消。确认active change已通过`openspec archive --skip-specs`移动到archive且主规格未同步后，只等待PR Review/merge，不得自行merge或创建新change。
```
