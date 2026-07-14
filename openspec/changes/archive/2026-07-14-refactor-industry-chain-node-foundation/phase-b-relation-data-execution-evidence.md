# Phase B relation data R1/R2 execution evidence

## 命名操作与边界

- 命名操作：`phase-b-relation-data`。
- 授权基线：checkpoint `7df39e3` 的 [R2 authorization package](phase-b-relation-data-authorization.md)，以及用户对失败恢复的两次明确授权。
- 环境：local development `tidewise-local-postgres` / `tidewise_local` / PostgreSQL 16.14，Goose=17。
- 冻结 manifest：[relation-write-manifest.json](relation-candidate-artifacts/relation-write-manifest.json)，SHA-256=`7651e0b591df1e03838df00ebc9acd6101ebcc76da18a6a314ff478c9f42990e`，96 条=`95 is_subcategory_of + 1 is_component_of`。
- 凭据由主对话认可的运行时方式安全注入；命令输出与本 evidence 均已脱敏，不记录密码或完整连接串。
- 未写 physical constraint、51 条 blocked、7 条 rejected、4 条 blocked constraint、Neo4j 或其他环境；未执行 migration 18、task 2.8/2.9 或生命周期后续。

## 首次执行失败、完整回滚与 R1 修复

首次获批 Write 在单事务 precommit 逐条读回阶段停止，错误定位到 relation `954fca34-56de-5894-a00a-72b5cfdbf348`；事务未 commit。独立只读读回确认 Goose17、relation=0、constraint=0、842/842/1,169/331 与保护基线均未变化。

根因是 manifest JSON 的 `time.Time` 与 PostgreSQL `timestamptz` 读回表示使用不同 `Location`，但代表同一时刻；旧 `reflect.DeepEqual` 将其误判为不相等。R1 checkpoint `ee0379e` 仅做以下修复：

- 所有 identity、endpoint、type、mechanism、condition、evidence、provenance、status 字段继续逐项精确比较；
- 仅 `VerifiedAt` 使用 `time.Time.Equal`，并明确零值只与零值相等；
- plan 的 unchanged 分类与 precommit 读回复用同一语义比较；
- RED/GREEN 覆盖同一时刻不同 Location 的 unchanged 分类、precommit 读回、其他标量漂移与零值边界。

该 R1 checkpoint 在任何重试前已运行 targeted tests、`go test -count=1 ./...`、relation generator tests、OpenSpec strict、diff/scope/secret 检查并 push。修复未改变 manifest、schema、SQL 或业务证据契约。

## Recovery evidence

| 项目 | 结果 |
| --- | --- |
| stable custom archive | `/Users/meierlink/.local/share/tidewise-ai/backups/postgresql/phase-b/20260714T070151Z/tidewise_phase_b_pre_relation_data.dump` |
| size | 1,076,733 bytes |
| SHA-256 | `9c36df2082240d6c4246b6c0d0a0f60a5ed99ea039c2a527a9eb15577038e1aa` |
| tools | 容器内 `pg_restore` PostgreSQL 16.14 |
| archive catalog | TOC 203 行，校验成功 |
| full decode | schema-only 56,193 bytes、data-only 3,801,080 bytes，均全量 decode 成功；临时 decode 文件已移除 |
| baseline | Goose17、842 node/profile、1,169 mappings、relation/constraint=0 |
| restore status | 未执行隔离 restore；不得声称 restore verified |

失败恢复前再次读取确认 archive size/SHA 未变，容器内 `pg_restore` 仍为 16.14。事务内失败依赖完整 rollback；提交后恢复或 forward-fix 均不在本次授权内。

## 重试前 fresh preflight

| 断言 | 读回结果 |
| --- | --- |
| database / PostgreSQL / Goose | `tidewise_local` / 16.14 / 17 |
| relation / physical constraint | 0 / 0 |
| active chain_node / profiles | 842 / 842 |
| external identifiers / entity_edges | 1,169 / 331 |
| relation schema | relation 12 columns、c7/f2/p1/u1/4 indexes；constraint 12 columns、c7/f2/p1/3 indexes |
| activity | other active=0；long transaction=0；ungranted lock=0；3 个等待均为 idle `ClientRead` |
| manifest | SHA 与 96/95/1/0/0 精确匹配 |
| real DB dry-run | `created=96, updated=0, unchanged=0` |
| Phase A protection | standard read-only preflight 确认 842/842/1,169/331、全部 orphan/duplicate/blank/legacy 指标和 12 类非目标 count/checksum 与冻结基线一致 |

任一漂移原本均会阻断重试；上述断言全部通过后才执行获批 runner。

## 唯一重试 Write report

- 执行窗口：2026-07-14 15:15 Asia/Shanghai。
- 唯一入口：`backend/cmd/entity-seed` relation-only mode，显式冻结 manifest 与 `chain-node-relation-approved-data-write`。
- 单事务输出：`created=96, updated=0, unchanged=0`；按类型 `is_subcategory_of=95`、`is_component_of=1`。
- 未使用手工 SQL、普通 seed、逐行事务、第二入口或第三次 Write。

## 写后立即只读 Query/assert

| 断言 | 结果 |
| --- | --- |
| Goose / relation / constraint | 17 / 96 / 0 |
| relation types | 95 / 1 / 0 / 0 |
| self-loop / incomplete evidence | 0 / 0 |
| ID duplicate / tuple duplicate | 0 / 0 |
| inactive、wrong-type、profile-missing endpoint | 0 |
| active chain_node / profiles / mappings / entity_edges | 842 / 842 / 1,169 / 331 |
| manifest semantic readback | DB snapshot dry-run 逐条比较，`created=0, updated=0, unchanged=96` |
| Phase A protection | 标准只读 preflight 的 orphan/duplicate/blank/legacy 指标均为 0；12 类非目标 count/checksum 与写前完全一致 |
| schema | Goose17 relation/constraint schema 未变化 |

写后 assertions 与幂等 dry-run 均 exit 0。未连接或操作 Neo4j。

## 停止点

Task 2.7b 的 local relation data Write 与 Query/assert 已完成，并由主对话独立验收 checkpoint `e2becc2`。本结果只允许进入 task 2.8/2.9 的普通 Apply-final 验证与 Review package，不授权 physical constraints、blocked/rejected relations、migration 18、Neo4j、其他环境或生命周期操作。
