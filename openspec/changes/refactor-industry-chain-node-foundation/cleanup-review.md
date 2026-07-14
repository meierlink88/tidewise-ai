# Task 1.13 Cleanup Review

## 状态与授权边界

- task 1.12 checkpoint `a0547e6` 与 task 1.13 Cleanup Readiness Review 均已由主对话明确批准；本文件保留其只读 readiness 证据，并索引后续已经独立授权、执行完成的 task 1.14 R3 cleanup evidence。
- R3 `phase-a-legacy-industry-cleanup` 已在当前 local 维护窗口按独立 authorization package 执行：先重新运行标准 Phase A `REPEATABLE READ READ ONLY` preflight、补充冻结查询与 archive 复核，再仅执行 migration 15，并立即完成同层只读 Query/assert。没有执行 migration 16、seed、mapping、relation 或任何 Neo4j 查询/写入/rebuild。
- 主对话已批准 workflow adoption checkpoint `0b21f74` 与本 task 1.13 Cleanup Readiness Review；随后又明确接受本文件记录的稳定 custom-format backup、archive integrity/hash、冻结基线与 forward-fix 作为**仅限当前 local 开发库**的 cleanup recovery evidence。该风险接受只覆盖已完成的 R3 migration 15，不构成任何后续 Write 授权。
- `backup_verified=false` 保持不变：archive integrity/hash 已核验，但命名 R2 restore rehearsal 在创建目标 database 和执行 restore 前由主对话撤销，未完成实际恢复验证。固定 disposable container/volume/network/secret 已清理且无残留；本次 cleanup 使用的 session token 只表示主对话已接受 local recovery evidence，不得被解释为实际 restore 或 `backup_verified=true`。
- 本文件的 readiness 部分按新工作流标记为 **R0 Cleanup Readiness Review package**：scope 是只读 preflight、backup archive 证据、冻结目标与影响分析；其历史 non-goals 是 restore rehearsal、migration、cleanup、seed、relation 和任何 PostgreSQL/Neo4j Write/rebuild。后续 R3 execution record 的范围严格限定为已完成的 migration 15 与同层 Query/assert。
- 命名 R2 `phase-a-backup-restore-rehearsal` 已撤销、未完成且不再是本次 local cleanup 的硬门槛；`backup_verified` 仍保持 false。命名操作 `phase-a-legacy-industry-cleanup` 已按 [R3 独立 authorization package](phase-a-legacy-industry-cleanup-authorization.md) 另行 Review、授权并执行；不得把 local 风险接受、task checkbox、历史 Apply 批准或本 package 验收解释为任何后续 Write 授权。

## 当前环境与只读快照

标准入口命令形状如下，密码由本地 PostgreSQL 容器环境在进程内注入，不进入仓库、命令证据或日志：

```text
APP_ENV=local DATABASE_PASSWORD='<container-injected>' go run ./cmd/entity-seed -phase-a-preflight
```

该入口由 `PostgresRepository.RunPhaseAPreflight` 显式使用 `REPEATABLE READ`、`READ ONLY`。首次真实运行发现 catalog CTE 使用 PostgreSQL 保留字 `references`，以 TDD 将其仅重命名为 `catalog_references` 后成功运行。

| 项目 | 当前证据 |
|---|---|
| 环境 | local development PostgreSQL container `tidewise-local-postgres` |
| 数据库 | `tidewise_local` |
| PostgreSQL | 16.14 (`16.14-1.pgdg13+1`) |
| postmaster 启动时间 | `2026-07-12 04:38:51.459335+00` |
| 目标冻结事务时间 | `2026-07-13 10:09:05.458564+00` |
| transaction snapshot | `21657:21657:` |
| 当前 Goose version | `14` applied；`15`、`16` 尚未执行 |
| preflight JSON | `/Users/meierlink/.codex/visualizations/2026/07/13/019f5a19-7f5e-76f0-b173-5dbe1c029dba/outputs/phase-a-preflight-20260713T100759Z.json` |
| preflight JSON SHA-256 | `e2697e8529bb6588a90d69139f773831ed050a1f8fc89000f1cc91286918a34a` |
| ID/FK 原始证据 | `/Users/meierlink/.codex/visualizations/2026/07/13/019f5a19-7f5e-76f0-b173-5dbe1c029dba/outputs/phase-a-cleanup-targets-and-fk-evidence.txt` |
| ID/FK 证据 SHA-256 | `504482d94e01a8ffd6b923e00e2d42cb0b075f5a252564e32f9ddb9d7bebf082` |

## 逻辑备份与恢复边界

| 项目 | 证据 |
|---|---|
| 文件 | `/Users/meierlink/.codex/visualizations/2026/07/13/019f5a19-7f5e-76f0-b173-5dbe1c029dba/outputs/tidewise_phase_a_pre_cleanup_20260713T100759Z.dump` |
| 完成时间 | `2026-07-13 18:08:25 CST` |
| 大小 | `991015` bytes |
| SHA-256 | `75c791a67d98d1b93ff73575a7e91d80eeb5c1262282c8d518e682ea2eee24d3` |
| 工具版本 | `pg_dump` / `pg_restore` 16.14 |
| 格式 | PostgreSQL custom format，compression 6，`--no-owner --no-acl` |
| TOC | 253 行；`pg_restore --list` 成功 |
| archive 解码 | `pg_restore --schema-only --file=/dev/null` 与 `--data-only --file=/dev/null` 均成功 |
| 目标集合核对 | backup 与冻结 live snapshot 均为 168 行；规范排序后的共同 SHA-256 为 `03d058855573bb7fc8d0b38a602bb77dda414088c9246f6251d25a33f77dc220` |
| backup_verified | **false**；restore rehearsal 已撤销，未实际 restore 到隔离 PostgreSQL；仅 archive integrity/hash verified |

### Local-only recovery evidence 风险接受

主对话明确接受以下组合证据作为**本次 `tidewise_local` migration 15 cleanup 的 recovery evidence**：稳定 custom-format backup 的固定 size/SHA-256、`pg_restore --list` 目录完整性、schema-only/data-only 全量解码、168 个目标与 466 个非目标的冻结 counts/checksums、完整引用审计，以及提交后只允许 reviewed forward-fix 的恢复边界。该决定是对当前 local 开发库可重建性和业务风险的显式接受，不表示 backup 已实际恢复，`backup_verified` 不得改为 true。

该例外不适用于 UAT、prod、共享环境或包含不可替代数据的数据库；这些环境执行 destructive cleanup 前仍必须提供与风险相称的更强恢复验证，默认包括隔离 restore rehearsal 或等价的已验证灾难恢复能力。本 change 不把 local 风险接受推广为项目级默认规则。

备份命令形状：

```text
pg_dump --format=custom --compress=6 --no-owner --no-acl \
  --username='<container POSTGRES_USER>' --dbname='<container POSTGRES_DB>' \
  --file='<container backup path>'
docker cp '<postgres container>:<container backup path>' '<review output path>'
```

若未来为 UAT/prod、不可替代数据或独立恢复操作另行执行实际演练，必须使用独立、空白、同 major version 的 PostgreSQL 16 环境，不能指向当前开发库；该可选路径不是本次 local cleanup 的前置条件：

```text
pg_restore --exit-on-error --no-owner --no-acl \
  --dbname='<isolated_restore_database>' '<reviewed backup file>'
```

恢复后必须只读复跑本文件中的 entity counts、目标集合 SHA-256、非目标 checksums、Goose version、orphan/duplicate checks。该 rehearsal 会创建 schema/data，超出本轮只读授权；主对话单独批准并验收前，Cleanup Write 必须保持阻断。

## 冻结目标集合

完整 168 行 `entity_type,id,entity_key,status` 已提交到 [cleanup-target-identities.csv](cleanup-target-identities.csv)：

| 类型 | 目标行数 |
|---|---:|
| `sector` | 112 |
| `industry_chain` | 2 |
| `chain_node` | 54 |
| **合计** | **168** |

提交 CSV SHA-256 为 `869aa9b0fde89b8274460b7255bb59ff84025431bddef90f860b1eb5344a8dee`；按完整行规范排序后与 backup 中同一集合的共同 SHA-256 为 `03d058855573bb7fc8d0b38a602bb77dda414088c9246f6251d25a33f77dc220`。所有 168 行当前均为 `active`；`status.merged=0`。

## 预计删除、删除结构与保留 counts

| 表/对象 | 当前 rows | migration 15 预计删除 | 预计保留/结果 |
|---|---:|---:|---|
| `entity_nodes` | 634 | 168 | 466 |
| `chain_node_profiles` | 54 | 54 | 表保留并收敛为最小 schema，写后 rows=0 |
| `sector_profiles` | 112 | 112 | 表删除 |
| `sector_source_mappings` | 89 | 89 | 表删除 |
| `industry_chain_profiles` | 2 | 2 | 表删除 |
| `industry_chain_memberships` | 27 | 27 | 表删除 |
| `industry_chain_topology_edges` | 24 | 24 | 表删除 |
| `industry_chain_physical_constraints` | 4 | 4 | 表删除 |
| `entity_convergence_manifests` | 1 | 1 | 表删除 |
| `entity_convergences` | 60 | 60 | 表删除 |
| `entity_convergence_reference_moves` | 29 | 29 | 表删除 |
| `entity_convergence_alias_moves` | 29 | 29 | 表删除 |
| `entity_edges` | 389 | 58 | 331 |
| `event_entity_links` | 0 | 0 | 0 |
| `prevent_entity_convergence_audit_mutation()` | 1 function | 1 | function 删除 |
| convergence append-only triggers | 4 | 4 | 随表删除 |
| `theme_profiles` | 不存在 | 0 | migration 15 创建空最小表 |

`entity_edges.from_entity_id` 有 6 个目标引用、`to_entity_id` 有 58 个目标引用；存在同一 row 两端均命中的重叠，因此实际待删除 row 数是 58，不是 64。`event_entity_links` 当前无目标引用。

## Catalog 与逻辑引用审计

标准 preflight 返回：

| 引用类型 | 数量 | 结论 |
|---|---:|---|
| foreign key | 49 | 包括目标表内部 FK 与所有直接指向 `entity_nodes` 的生产 FK |
| function | 1 | 仅 `prevent_entity_convergence_audit_mutation`，migration 15 显式删除 |
| trigger | 4 | 仅 convergence append-only triggers，相关表与 function 显式删除 |
| procedure | 0 | 无 |
| view | 0 | 无 |
| rule | 0 | 无 |

额外动态扫描了全部 37 个直接引用 `entity_nodes(id)` 的单列 FK column。非零目标引用只有：

| FK column | 引用数 | migration 15 处理 |
|---|---:|---|
| `chain_node_profiles.entity_id` | 54 | 先删目标 profile，再 ALTER 最小 schema |
| `entity_convergence_alias_moves.from_entity_id` | 29 | 整表删除 |
| `entity_convergence_alias_moves.to_entity_id` | 29 | 整表删除 |
| `entity_convergence_reference_moves.from_entity_id` | 29 | 整表删除 |
| `entity_convergence_reference_moves.to_entity_id` | 29 | 整表删除 |
| `entity_convergences.legacy_entity_id` | 60 | 整表删除 |
| `entity_convergences.target_entity_id` | 29 | 整表删除 |
| `entity_edges.from_entity_id` | 6 | 按冻结 ID 删除相关 edge |
| `entity_edges.to_entity_id` | 58 | 按冻结 ID 删除相关 edge |
| `industry_chain_profiles.entity_id` | 2 | 整表删除 |
| `sector_profiles.entity_id` | 112 | 整表删除 |
| `sector_source_mappings.sector_entity_id` | 89 | 整表删除 |

其余 25 个 direct FK columns 对目标集合均为 0，包括 alliance/benchmark/commodity/company/economy/index/instrument/market/metric/person/policy_body/security profiles 与 `event_entity_links.entity_id`。当前未发现未知 FK 或逻辑引用；若 Write 前复跑出现任何新增 reference、非零未知 FK 或 catalog 差异，必须阻断 migration 15。

## 非目标保护基线

| entity_type | rows | checksum |
|---|---:|---|
| `alliance_org` | 10 | `c3a3fe9972c0eb41826c8c1db3b5856a` |
| `benchmark` | 10 | `d439c91f71bd797145f0805cb7b4a147` |
| `commodity` | 45 | `0142c571cb1798a574eaa15806f26347` |
| `company` | 77 | `4cc1696c9f001a7a2c99cea1dca0c279` |
| `economy` | 50 | `c08598819824043e0835ff0e97434b2e` |
| `index` | 43 | `f6b0fcece79785b47430dd77dcc8c066` |
| `instrument` | 4 | `cde558f4067d5f40e9061ca1756ad816` |
| `market` | 47 | `c0eab0716909e24ff0cf84f38c41f236` |
| `metric` | 43 | `31755470c6487327074a2b7d6f156d1e` |
| `person` | 30 | `2733a863cc2d5041d4c5e03662cdc6fa` |
| `policy_body` | 30 | `192cf2aa723e9442592eea2fa24176db` |
| `security` | 77 | `f21ae68fc251b0ed4d2ca1dbed3e1ddd` |

合计 466 行。Cleanup Query 必须逐类型同时匹配 row count 与 checksum，任何差异都视为误删或并发漂移并停止后续 schema/seed 阶段。

## 完整性与条件门禁

- `entity_key.blank=0`、`entity_key.duplicate_groups=0`，当前快照满足全局唯一性的只读 preflight 条件，但本 change 仍不得据此自动增加全局唯一约束。
- membership/topology duplicate groups 均为 0。
- sector/chain_node/industry_chain profile、membership、topology、physical constraint、entity edge、event link 的全部已定义 orphan 指标均为 0。
- 54 个旧 chain_node profile 中 `definition_blank=28`；这些 rows 全部属于删除集合，不会迁移或复用，不能用模板补写。
- `archive_mode=off`，不存在 WAL/PITR 保护。当前 local cleanup 的已接受恢复边界由稳定 custom-format backup、archive 完整性/全量解码、冻结基线和 reviewed forward-fix 共同构成；实际 restore 未验证，不得据此声称具备已演练恢复能力。

## migration 15 范围、顺序与锁影响

`000015_refactor_industry_chain_node_phase_a.sql` 必须先于 `000016_add_entity_external_identifiers.sql`。migration 15 的固定顺序是：

1. 校验专属 session setting；
2. 冻结三类旧实体 ID 到 transaction-local temp table；
3. 删除目标 `event_entity_links` 与 `entity_edges`；
4. 叶到根删除 convergence/audit 与 industry-chain 专属表/function；
5. 删除 sector mapping/profile；
6. 删除旧 chain_node profiles，并 ALTER 为最小 schema；
7. 动态扫描所有直接 FK，发现残留目标引用即抛错并回滚；
8. 删除 168 个旧 `entity_nodes`；
9. 创建空 `theme_profiles`。

目标表当前总 relation size 约 1.3 MiB；最大单表为 `entity_nodes` 320 kB、`entity_edges` 240 kB。数据量较小不等于无锁风险：DELETE 会取得相应表的 `ROW EXCLUSIVE` 与行锁，DROP/ALTER/CREATE 会取得 `ACCESS EXCLUSIVE`，可能阻塞实体读取/写入；整个 Goose migration 是单事务，必须在维护窗口、无 seed/ingestion/admin 写入时执行，并在执行前确认无长事务。任何 lock timeout、statement error、未知 FK 或 count mismatch 必须 rollback；不得临时禁用约束或手工补删。

## 已执行 R3 命令形状（历史记录）

本次唯一获准且已执行的命令形状：

```text
TIDEWISE_DATABASE_URL='<reviewed URL including options=-c%20tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified>' \
  go run ./cmd/dbmigrate -apply -target-version 15
```

该命令已在以下条件全部满足后由主对话通过 [独立 R3 package](phase-a-legacy-industry-cleanup-authorization.md) 单独授权：local-only recovery evidence 风险接受仍有效；稳定 backup size/hash 与 archive integrity/full decode 证据可复核；立即重跑 preflight 且冻结 CSV/count/checksum/catalog 无漂移；确认维护窗口与备份文件可访问。成功报告只包含 migration 15，migration 16 继续 pending。Cleanup Query 人工验收前不得请求或执行 migration 16。

## 回滚、forward-fix 与写后 Query 边界

- 提交前错误：依赖 migration 单事务 rollback，确认 Goose version 仍为 14，并复跑全部 preflight；不得执行 Down 或部分手工修复。
- 提交后 schema/data 错误：migration 15 的 Down 明确阻断。优先停止后续层并提交 reviewed forward-fix migration；若必须使用整库 backup 恢复，须重新提交独立恢复操作 package，不能把本次未完成 rehearsal 视为恢复验证。
- 写后 Query 必须验证：Goose version=15、migration 16 pending；旧三类 entity rows=0；旧专属表/function/trigger 不存在；`chain_node_profiles` 最小列/约束正确且 rows=0；`theme_profiles` 空表存在；目标 event links/edges=0；entity_edges=331；所有 orphan/duplicate=0；12 类非目标 count/checksum 完全不变；重复执行被门禁或报告 already-clean，不扩大删除范围。
- PostgreSQL cleanup 后既有 Neo4j projection 会暂时陈旧；本 change 不查询、清理、写入或 rebuild Neo4j。

## R3 执行结果与当前 blocker

1. 命名 R2 restore rehearsal 已由主对话撤销且未完成，`backup_verified=false`；固定 disposable 资源已清理且无残留。这些历史事实保持不变，且不被改写为 restore verified。
2. 主对话随后独立批准并已执行 [命名 R3 `phase-a-legacy-industry-cleanup` package](phase-a-legacy-industry-cleanup-authorization.md)。新鲜 evidence 位于 `/Users/meierlink/.local/share/tidewise-ai/cleanup-evidence/20260713T113234Z/`：preflight、archive、Goose、migration report 与写后 Query/assert 均通过；task 1.14 已完成、等待主对话验收。
3. task 1.14 execution checkpoint `f2bc90a` 已由主对话验收；task 1.15b 的 [external identifier schema R2 execution](phase-a-external-identifier-schema-authorization.md) 已完成 Write 与 Query/assert，等待主对话验收。验收前不得进入 1.16、seed、mapping、relation 或任何 Neo4j 操作，本次 local 风险接受不构成后续 R2/R3 Write 授权。

## Readiness checkpoint 验证

- `go test -count=1 ./internal/apps/entityfoundation/seed ./cmd/entity-seed`：通过。
- `go test -count=1 ./...`：沙箱内首次仅因 `httptest` 无权绑定本机临时端口失败；不设置数据库测试 URL，在沙箱外以相同命令重跑通过。
- `openspec validate refactor-industry-chain-node-foundation --strict`：通过。
- `git diff --check`：通过。
- scoped file review 与 secret pattern scan：范围仅为 Phase A preflight 修复、task/Review artifacts 和冻结 ID/key CSV；未发现 literal connection string、密码、token、private key 或 API key。命令形状中的连接信息均为尖括号占位符。

## R3 execution evidence

- 新鲜 evidence：`/Users/meierlink/.local/share/tidewise-ai/cleanup-evidence/20260713T113234Z/`；仅含脱敏 container/version、backup hash/archive、标准 preflight/Goose JSON、冻结 target diff、写后只读 assertions 与命令报告。
- Write 前：backup size/hash、TOC=253、schema/data decode、Goose=14、15/16 pending、634=168+466、catalog=`foreign_key:49,function:1,trigger:4`、长事务/writer=0 均通过。
- Write：只运行一次 target-version 15；report 的实际 applied 只有 `000015_refactor_industry_chain_node_phase_a.sql`。
- Write 后：Goose=15、16 pending；466 非目标 count/checksum 不变；旧实体/表/function/trigger=0；`entity_edges=331`；profile schema、orphan/duplicate/blank-key 与无全局 entity_key 唯一约束均通过。
- 该记录不声称 `backup_verified=true`，不包含或暴露连接串、密码或其他 secret；未执行 Neo4j 操作。
