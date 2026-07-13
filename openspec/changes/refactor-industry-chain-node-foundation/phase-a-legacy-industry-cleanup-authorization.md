# Phase A legacy industry cleanup R3 authorization package

## 状态与授权对象

- 命名操作：`phase-a-legacy-industry-cleanup`。
- 风险等级：**R3**；migration 15 会不可逆删除 local PostgreSQL 中的旧产业实体、关系、约束与审计结构。
- 当前状态：**仅 R0 authorization package 草案，未授权、未执行**。
- 本文件只请求人工 Review；普通 Apply、task 1.13 验收、local 风险接受或本文件提交均不构成 Write 授权。
- 若获授权，只允许在下述当前 local 开发环境执行一次 migration 15 Write，并立即完成同层 Query/assert。任何环境、范围、基线或命令变化都使授权失效。

## 固定环境与执行范围

| 项目 | 固定值 |
|---|---|
| environment | local development only |
| PostgreSQL container / database | `tidewise-local-postgres` / `tidewise_local` |
| PostgreSQL | 16.14 |
| execution path | 标准 `backend/cmd/dbmigrate` |
| before Goose | version 14；15、16 pending |
| allowed target | `-target-version 15`，实际 applied 只能是 `000015_refactor_industry_chain_node_phase_a.sql` |
| cleanup targets | sector 112、industry_chain 2、旧 chain_node 54，共 168 |
| protected non-target | 12 类共 466 |
| maintenance window | 无 seed、ingestion、admin 或其他 PostgreSQL writer；执行前确认无长事务 |

明确排除：migration 16、external identifier schema、node/profile seed、mapping、relation、任何手工 SQL/清库、Down、UAT/prod、Neo4j 查询/写入/cleanup/rebuild，以及联盟、经济体、国家、市场、benchmark/index 等非目标数据调整。

## Local-only recovery evidence 与风险接受

主对话明确接受以下证据组合，作为**本次 local 开发库 cleanup** 的 recovery evidence：

| 证据 | 固定值 / 结论 |
|---|---|
| stable backup | `/Users/meierlink/.local/share/tidewise-ai/backups/postgresql/phase-a/20260713T100759Z/tidewise_phase_a_pre_cleanup.dump` |
| size / SHA-256 | `991015` bytes / `75c791a67d98d1b93ff73575a7e91d80eeb5c1262282c8d518e682ea2eee24d3` |
| archive | PostgreSQL custom format、compression 6、`--no-owner --no-acl` |
| archive integrity | TOC 253 行；`pg_restore --list`、schema-only 全量解码、data-only 全量解码均已通过 |
| frozen target evidence | [168 行 identity CSV](cleanup-target-identities.csv)，文件 SHA-256 `869aa9b0fde89b8274460b7255bb59ff84025431bddef90f860b1eb5344a8dee`；与 archive 共同规范行 SHA-256 `03d058855573bb7fc8d0b38a602bb77dda414088c9246f6251d25a33f77dc220` |
| protected baseline | 12 类 466 行的 counts/checksums，见下表 |
| recovery boundary | 提交前依赖单事务 rollback；提交后停止后续层，只允许经 Review 的 forward-fix；若必须整库恢复，另行提交恢复操作 package |

命名 R2 restore rehearsal 已撤销且未执行，临时资源零残留，`backup_verified=false`。以上风险接受不表示 backup 已实际恢复，也不得把 `backup_verified` 改为 true。它只适用于当前 local 开发库；UAT、prod、共享环境或不可替代数据执行 destructive cleanup 前仍须提供更强恢复验证，默认包括隔离 restore rehearsal 或等价的已验证灾难恢复能力。

Migration 15 当前要求 session setting `tidewise.phase_a_cleanup_write_authorized` 的值为 `reviewed_backup_verified`。该字符串是现有 migration 的机械激活 token；在本 local-only package 中，它只表示主对话已 Review 并接受上述 recovery evidence，不得作为实际 restore 或 `backup_verified=true` 的证据。真实连接配置只能在获授权后的运行时注入，不得写入 artifact、evidence 或日志。

## Write 前必须新鲜通过的 assertions

正式请求执行授权时必须附上同一维护窗口内的新鲜只读 preflight；本 R0 package 不运行该查询。结果必须与以下冻结证据完全一致：

- 当前 database/container/PostgreSQL 身份与本文件一致；Goose version=14，migration 15/16 均 pending。
- stable backup 存在、权限可读、size/SHA-256 完全一致；archive TOC 与 schema/data 全量解码复核通过。
- `entity_nodes=634`；目标恰为 168 个 active rows：sector=112、industry_chain=2、chain_node=54；目标规范行 SHA-256=`03d058855573bb7fc8d0b38a602bb77dda414088c9246f6251d25a33f77dc220`。
- 目标相关 rows 与预计删除数一致：chain node profile=54、sector profile=112、sector source mapping=89、industry chain profile=2、membership=27、topology=24、physical constraint=4、convergence manifest=1、convergence=60、reference move=29、alias move=29、entity edge=58、event link=0。
- Catalog 恰为 foreign key=49、function=1、trigger=4、procedure/view/rule=0；37 个直接引用 `entity_nodes(id)` 的 FK 扫描不得出现未知引用。
- `entity_key.blank=0`、duplicate groups=0、`status.merged=0`；membership/topology duplicate groups=0；所有已定义 orphan 指标=0。
- 下列 12 类非目标 counts/checksums 均一致，合计 466：

| entity_type | rows | checksum |
|---|---:|---|
| alliance_org | 10 | `c3a3fe9972c0eb41826c8c1db3b5856a` |
| benchmark | 10 | `d439c91f71bd797145f0805cb7b4a147` |
| commodity | 45 | `0142c571cb1798a574eaa15806f26347` |
| company | 77 | `4cc1696c9f001a7a2c99cea1dca0c279` |
| economy | 50 | `c08598819824043e0835ff0e97434b2e` |
| index | 43 | `f6b0fcece79785b47430dd77dcc8c066` |
| instrument | 4 | `cde558f4067d5f40e9061ca1756ad816` |
| market | 47 | `c0eab0716909e24ff0cf84f38c41f236` |
| metric | 43 | `31755470c6487327074a2b7d6f156d1e` |
| person | 30 | `2733a863cc2d5041d4c5e03662cdc6fa` |
| policy_body | 30 | `192cf2aa723e9442592eea2fa24176db` |
| security | 77 | `f21ae68fc251b0ed4d2ca1dbed3e1ddd` |

任一值漂移、stable backup/archive 证据失效、出现未知引用、存在长事务或 writer、维护窗口不成立时，必须停止并重新提交 R3 package，不得现场调整范围。

## 唯一允许的 Write 形状

获主对话明确授权后，执行者只能在运行时组装经过审阅的 local DSN，并注入上述专属 session setting；artifact、命令证据和日志只保留占位符：

```text
TIDEWISE_DATABASE_URL='<runtime-only reviewed local cleanup DSN with dedicated session setting>' \
  go run ./cmd/dbmigrate -apply -target-version 15
```

Executor 必须使用 pinned PostgreSQL connection 取得/release session advisory lock，并在持锁后读取 before state、执行后读取 after state。成功 report 的 `applied` 只能包含 migration 15，`remaining` 必须包含 migration 16；target 未精确到达、unlock/release 失败或报告状态矛盾均视为失败。不得用无 target apply-all、故意触发 migration 16 失败、手工 SQL 或第二条执行路径替代。

## Write 后立即 Query/assert

Write 命令原子结束后，不论 report 成功或失败，都必须先停止其他层并执行只读核验；只有以下全部通过才可提交 Query evidence：

- Goose version=15；migration 16 仍 pending；实际 applied 列表只含 15。
- `entity_nodes=466`；sector/industry_chain/旧 chain_node rows 均为 0。
- 12 类非目标 counts/checksums 与 Write 前完全一致，总计仍为 466。
- `entity_edges=331`，且不存在指向已删除目标的 edge；目标 `event_entity_links=0`。
- `sector_profiles`、`sector_source_mappings`、`industry_chain_profiles`、memberships、topology、physical constraints、convergence/audit tables、相关 function/trigger 均不存在。
- `chain_node_profiles` 只含 `entity_id`、`definition`、可空 `boundary_note` 的目标最小 schema，rows=0；`theme_profiles` 为空且 definition/boundary 均为非空约束。
- 全部 profile、edge、event link 与其他 FK orphan 指标=0；blank/duplicate 指标=0；不得新增全局 `entity_key` 唯一约束。
- 查询确认 migration 16、seed、mapping、relation 均未执行；Neo4j 未被连接或修改。PG cleanup 后旧 Neo4j projection 被明确记录为暂时陈旧。

Query/assert 未经主对话验收前，不得请求或执行 migration 16，也不得进入 seed 或 Phase B。

## 停止、失败与恢复边界

- Write 前断言失败：不执行 migration，冻结差异并返回主对话 Review。
- Migration 事务失败：依赖 rollback；确认 Goose 仍为 14、15/16 pending，并重跑只读 preflight。不得运行 Down、手工补删或自动重试。
- Report 包含 16、target 未到 15、catalog/count/checksum/orphan/非目标保护任一写后断言失败：立即停止所有后续层，保留脱敏 evidence；不得把部分成功解释为验收通过。
- 提交后发现问题：只允许另行 Review 的 forward-fix migration/命令。若需要使用 stable backup 做整库恢复，必须新建独立恢复操作 package；本次未完成 rehearsal 不能被宣称为恢复验证。
- 日志或 evidence 不得包含连接串、密码、secret、环境 dump 或个人隐私信息。

## Authorization request

请求主对话 Review 本 R3 package。**当前不请求立即执行，且尚无 `phase-a-legacy-industry-cleanup` Write 授权。** 若主对话未来明确授权，必须逐项确认环境、migration 15 only、168/466 范围、local-only recovery risk acceptance、before/after assertions、停止条件与维护窗口；任何省略或范围变化均视为未授权。

## R0 checkpoint 验证

- `openspec validate refactor-industry-chain-node-foundation --strict`：通过。
- `git diff --check`：通过。
- scope：仅 tasks、cleanup review、restore rehearsal 历史状态、design 与本 R3 authorization package 五个 OpenSpec artifacts；无源码、migration 或数据库 evidence 修改。
- 完整 PostgreSQL URL scan 与 secret material scan：均为 0。
- 本 checkpoint 未连接 PostgreSQL/Neo4j，未创建 Docker 资源，未执行 migration、cleanup、seed、restore 或 assertions。
