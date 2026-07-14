# Phase B relation schema R2 execution evidence

## 命名操作与范围

- 命名操作：`phase-b-relation-schema`。
- 授权基线：checkpoint `b738ae8` 的 [R2 authorization package](phase-b-relation-schema-authorization.md)。
- 环境：local development `tidewise-local-postgres` / `tidewise_local` / PostgreSQL 16.14。
- 实际 Write：标准 `backend/cmd/dbmigrate -apply -target-version 17` 仅执行一次，实际 applied 仅 `000017_add_chain_node_relations.sql`。
- 凭据由运行时安全注入；命令、日志与本 evidence 不记录密码或完整数据库连接串。
- 未执行 relation/constraint data、task 2.7 runner、migration 18、Neo4j、UAT/prod/shared、Down、手工 SQL 或第二 Write 路径。

## Fresh Write 前 preflight

| 断言 | 读回结果 |
| --- | --- |
| container / database / PostgreSQL | `tidewise-local-postgres` / `tidewise_local` / 16.14，container healthy |
| Goose | current=16；唯一 pending/remaining=`000017`；17 尚未 applied |
| migration 17 SHA-256 | `a7f4406084c04e56f1f85bf24dfa10d58f112d3e5732b2e52a2849cbb514a3df`，与 package 一致 |
| target catalog | 两张目标表、同名前缀 relation/index/function 均不存在 |
| entity / active chain_node / profiles | 1,308 / 842 / 842 |
| external identifiers / entity_edges | 1,169 / 331 |
| eastmoney / ths / dual-source / dual-taxonomy code | 818 / 351 / 241 / 13 |
| identity/integrity | mapping triple/ID duplicate=0；inactive/wrong target=0；profile/mapping/edge orphan=0；blank/duplicate entity key=0 |
| activity | other active client=0；long transaction=0；waiting lock=0 |
| protected baseline | 12 类共 466，逐类型 count/checksum 与 Phase A 验收 evidence 完全一致 |

标准 Phase A preflight 在 `REPEATABLE READ READ ONLY` 中同时确认：`entity_nodes.total=1308`、`profile.chain_node=842`、`external_identifier.total=1169`、`entity_edge.total=331`、legacy catalog/retired schema/全部已列 orphan/duplicate/blank 指标为 0；manifest 仍为 842 node/profile，SHA-256=`9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e`。

## Fresh recovery evidence

| 项目 | 结果 |
| --- | --- |
| stable custom archive | `/Users/meierlink/.local/share/tidewise-ai/backups/postgresql/phase-b/20260714T062741Z/tidewise_phase_b_pre_relation_schema17.dump` |
| size | 1,066,961 bytes |
| SHA-256 | `1ef876bc5f162b3bd6fea58f189d3424d3354369522562713e85c19e8c0c8afc` |
| tools | `pg_dump` / `pg_restore` 16.14 |
| archive catalog | `pg_restore --list` 成功，TOC 188 行 |
| full decode | schema-only 与 data-only 全量解码均 exit 0 |
| baseline | Goose=16、842 node/profile、1,169 mappings、目标两表不存在；与 Write 前 preflight 一致 |
| restore status | 未执行隔离 restore；不得声称 restore verified |

backup 路径位于 Desktop worktree 生命周期之外，文件权限为 `0600`。任一 backup 校验失败原本均会阻断 Write；本次全部通过后才执行 migration 17。

## 唯一 Write report

- maintenance window：2026-07-14 14:28:54 Asia/Shanghai。
- 标准 executor 输出 `OK 000017_add_chain_node_relations.sql`，Goose 成功迁移至 version 17。
- report：current=`17`；actual applied 仅 `000017`；remaining=`null`。
- 未使用 apply-all、Down、手工 SQL、第二执行路径或后续 migration 失败切层。

## 写后立即只读 Query/assert

| 断言 | 结果 |
| --- | --- |
| Goose | current=17；17 applied 恰好一次；标准 post-check pending/remaining 均为空 |
| table rows | `chain_node_relations=0`；`chain_node_physical_constraints=0` |
| columns | 两表各 12 列；类型、nullability、status/timestamp defaults 与 migration 17 一致 |
| relation constraints | PK、2 endpoint FK `ON DELETE RESTRICT`、7 个 CHECK、from/to/type tuple unique |
| physical constraint constraints | PK、2 subject FK `ON DELETE RESTRICT`、7 个 CHECK（含 exact-one-subject） |
| indexes | relation PK/tuple unique/to-type/input-dependency mechanism；constraint PK/node subject/relation subject，定义均精确一致 |
| forbidden/extra objects | trigger/function/view/rule/JSONB/constraint semantic unique/旧 relation type 均为 0 |
| protected data | entity_nodes=1,308；active chain_node=842；profiles=842；mappings=1,169；entity_edges=331 |
| mapping aggregates | eastmoney=818；ths=351；dual-source=241；dual-taxonomy code=13 |
| integrity | profile/mapping/edge orphan、mapping triple/ID duplicate、blank/duplicate entity key 均为 0 |
| 12 类非目标 | count/checksum 与 Write 前逐项完全一致，共 466 |

幂等证据使用标准 post-check：current=17 且 pending/remaining 为空，相同 target 的执行计划为空；按 package 边界没有为幂等再次运行 Write。未连接、查询、写入、清理或 rebuild Neo4j。

正式写后 Query/assert 与标准 Goose post-check 全部通过后，evidence 定稿阶段另发起了一次冗余只读摘要查询；该命令因遗漏 `FROM goose_db_version` 在 SQL 解析阶段失败，未产生数据库状态变化。依照 fail-closed/no-retry 边界没有重试该数据库查询，也没有执行任何后续层；stable backup 随后仅以文件读取重新确认 size/SHA-256 不变。

## 停止点

Task 2.6b 的 schema Write 与 Query/assert 已执行完成，等待主对话独立验收。本结果不授权 task 2.7、96 条 relation data、physical constraint data、migration 18 或任何其他环境/系统操作。
