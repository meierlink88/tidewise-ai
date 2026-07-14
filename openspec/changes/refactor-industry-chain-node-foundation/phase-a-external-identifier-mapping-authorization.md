# Phase A external identifier mapping R2 authorization Review package

## 状态、风险与边界

- 命名操作：`phase-a-external-identifier-mapping`。
- 风险等级：**R2**；未来仅限 local development 的 `tidewise-local-postgres` / `tidewise_local`，不得扩展至 UAT、prod、shared 或其他数据库。
- 本文件是 **R0 authorization Review package**，不构成 Write 授权；主对话必须在下列执行阻断解除后，单独明确授权该命名操作。
- 不包含 schema/migration、node/profile seed、theme、relation/constraint、cleanup、Neo4j 查询/写入/rebuild、UAT/prod/shared、Sync、Archive 或 Deliver。

## 冻结输入与预计范围

| 项目 | 当前冻结值 |
| --- | --- |
| mapping manifest | `mapping-candidate-artifacts/external-identifier-mapping-candidates.json` |
| manifest SHA-256 | `05539cd9f940cfcc5ec67cde5c395563b672ffa52d56090da0a83bd0d5997658` |
| validation | `mapping-candidate-artifacts/external-identifier-mapping-validation.json` |
| validation SHA-256 | `2fdad50dc5091b30723953f22d3116b3926e08e6d2e72156872ae3ba554e9a11` |
| 预计 mapping rows | 1,169，全部初次 `created` |
| eastmoney / ths | 818 / 351 |
| 双来源 canonical 节点 | 241 |
| 单 taxonomy / 双 taxonomy 来源 code | 1,143 / 13 |
| 三元 identity / 确定性 ID 重复、预期 orphan、candidate blockers | 0 / 0 / 0 |

用户核验工作簿是分类事实源。13 个组合来源分类 code 各产生 `industry_sector` 和 `concept_sector` 两条记录；相同 `(source_system, external_code)` 的跨 taxonomy 重复是预期，唯一身份仍严格为 `(source_system, source_taxonomy_type, external_code)`。不得使用 `1c63479` 的网页研究、二选一、排除或组合字符串。

## 当前 fresh 只读 preflight（2026-07-14，Asia/Shanghai）

下列断言通过 local container 内显式 `REPEATABLE READ READ ONLY` 事务获得；未执行 DDL/DML。标准 Go `entity-seed -phase-a-preflight` 因运行时密码会出现在 shell 命令中而被安全策略拒绝，故未执行；本包记录了等价的无凭据容器内只读 guard，并要求实际 R2 窗口仍使用项目标准 preflight entry 和独立 guard。

| 断言 | 读回结果 |
| --- | --- |
| identity | `tidewise_local`；PostgreSQL `16.14`；container=`tidewise-local-postgres` |
| Goose | current `16`；仓库 migration 序列止于 `000016` |
| approved nodes / profiles / exact active chain_node-profile binding | 842 / 842 / 842 |
| `entity_external_identifiers` / `entity_edges` | 0 / 331 |
| chain-node profile orphan / external identifier orphan / blank key / duplicate key | 0 / 0 / 0 / 0 |
| external identifier catalog | 9 columns、8 constraints（PK/FK/unique/5 checks）、3 indexes |
| active non-idle sessions / waiting locks | 0 / 0 |

12 类非目标实体的 count/checksum 基线在本次只读事务内与 cleanup/migration 16 后保护范围一致：`alliance_org=10/c3a3fe9972c0eb41826c8c1db3b5856a`、`benchmark=10/d439c91f71bd797145f0805cb7b4a147`、`commodity=45/0142c571cb1798a574eaa15806f26347`、`company=77/4cc1696c9f001a7a2c99cea1dca0c279`、`economy=50/c08598819824043e0835ff0e97434b2e`、`index=43/f6b0fcece79785b47430dd77dcc8c066`、`instrument=4/cde558f4067d5f40e9061ca1756ad816`、`market=47/c0eab0716909e24ff0cf84f38c41f236`、`metric=43/31755470c6487327074a2b7d6f156d1e`、`person=30/2733a863cc2d5041d4c5e03662cdc6fa`、`policy_body=30/192cf2aa723e9442592eea2fa24176db`、`security=77/f21ae68fc251b0ed4d2ca1dbed3e1ddd`。

实际 R2 前必须在同一维护窗口重新执行标准 `-phase-a-preflight`、独立 `psql` environment guard 和 mapping-specific snapshot。任一数据库 identity、Goose、manifest hash/count、842 绑定、外部标识零行、保护 checksum、catalog、writer、长事务或锁值漂移，立即 fail-closed，不得 Write。

## Recovery evidence

本层选择 `backup`，不把 curated local 视为 disposable，也不把已撤销 restore rehearsal 声称为 verified。

| 证据 | 当前值 |
| --- | --- |
| stable custom archive | `/Users/meierlink/.local/share/tidewise-ai/backups/postgresql/phase-a/20260714T015901Z/tidewise_phase_a_post_cleanup_pre_schema16.dump` |
| size / SHA-256 | `904,964` bytes / `02b12193985b0fc88df8c04b3a8efe5c3312e2184513be6036676243f616a0e4` |
| 当前复核 | 文件存在、size/hash 匹配 |
| archive integrity 历史证据 | custom archive TOC=182、`pg_restore --list`、schema-only 与 data-only 全量解码已在 schema-16 package 证据完成 |
| restore status | 未执行 restore rehearsal；`backup_verified=false` |

本机当前无 `pg_restore` binary，因此本 R0 只能新鲜复核 size/hash；实际 R2 maintenance window 必须使用获批准工具重新验证 TOC 与 schema/data decode。无法重新验证、备份不可访问或 hash/size/环境不一致均为停止条件；提交后纠错只能使用新 Review 的 forward-fix，禁止 Down、手工 SQL 或清库。

## 未来唯一 Write 路径、事务和冲突语义

1. 先刷新本包全部 read-only preflight，并验证 candidate manifest 的 1,169/818/351/241/13 与 SHA-256 完全一致。
2. 只接受一个经 R1 Review 的 mapping runner：逐行读取冻结 manifest，验证 active `chain_node` target、三元 identity、确定性 ID、`external_name`/status 和完整 snapshot；不得调用通用 entity/profile/relationship seed。
3. 每条 `UpsertExternalIdentifier` 在独立 PostgreSQL transaction 中对三元 identity 取得 `pg_advisory_xact_lock`，锁定 active target，先读取、再 `INSERT ... ON CONFLICT DO NOTHING`，并在并发 winner 后重读。
4. 同一 ID/三元 identity 换绑 entity、确定性 ID 漂移或 snapshot 两索引不一致必须 conflict 并 rollback；仅同 entity 的 `external_name`/status 漂移可 update；完全一致才是 unchanged。
5. 本命名层的业务 Write 范围只能是 1,169 行 `entity_external_identifiers`。任何 entity_nodes、profile、edge、event、theme、relation、constraint、migration 或 Neo4j 写入都必须 fail-closed。

## R1 execution blocker

当前 repository 已实现并测试 `UpsertExternalIdentifier` 的三元 unique、advisory lock、target 验证、冲突重读和 idempotent update；但 `backend/cmd/entity-seed` 的 manifest/service 目前只执行 entity/profile/relationship，未读取本 mapping manifest，也未调用该 repository 方法。

因此此包**尚无可审计的唯一 mapping runner**，不能进入 R2 Write。下一步必须是独立、无数据库 Write 的 R1 implementation Review：定义只读 manifest loader、mapping-only runner、full dry-run/snapshot report 和测试，并在主对话验收后更新本包的精确命令形状。不得以手工 SQL、临时脚本、通用 entity seed 或第二执行路径绕过该阻断。

## Write 后立即 Query/assert（仅供未来单独授权）

若且仅若 R1 blocker 已解除、主对话对本命名操作明确授权且 Write 成功，必须立即只读断言：

1. Goose 仍为 `16`，没有 schema/migration 变化；实际数据影响仅 `entity_external_identifiers`。
2. rows=1,169；eastmoney=818、ths=351；241 个 canonical 节点同时两来源；13 个 source/code 各恰好两条（industry/concept）。
3. `(source_system, source_taxonomy_type, external_code)` 与 deterministic ID 各无重复；所有 1,169 行绑定 active chain_node；orphan=0。
4. `source_system`、taxonomy、external_code、external_name、status 非空且 status=active；无 JSONB、拼接 taxonomy 或旧 source mapping data。
5. 第二次只读 dry-run 必须预期 1,169 unchanged、0 created/updated/conflict；实际第二次 Write 仍需新的单独授权。
6. 842 chain_node/profile binding、entity_edges=331、12 类非目标 count/checksum、blank/duplicate key、profile/event/edge orphan 与写前基线完全不变。

任一断言失败、部分 Write、未知对象、锁/事务异常或输出含 secret 都立即停止，保存脱敏证据并等待主对话；不得自动重试或扩大范围。

## 明确不授权

- 当前文件不授权任何 PostgreSQL/Neo4j Write、mapping runner 实现、R2 执行、second-write 幂等验证、migration、schema 修改、cleanup、seed 其他实体、关系/约束、Neo4j rebuild、UAT/prod/shared、Sync、Archive、Deliver、PR merge 或 branch cleanup。
