# Phase B relation data R2 authorization Review package

## 状态、命名操作与边界

- 命名操作：`phase-b-relation-data`。
- 风险等级：**R2，local-only**。
- 当前状态：R1 atomic runner、冻结 manifest 与本条件式执行包已获授权；失败恢复后唯一重试已完成，见 [R1/R2 execution evidence](phase-b-relation-data-execution-evidence.md)，等待主对话独立验收。
- 本包只覆盖 96 条已批准静态分类/组成关系。`input_to=0`、`depends_on=0`、physical constraint=0；51 条 blocked、7 条 rejected、4 条 blocked constraint 均排除。
- 不包含 schema/migration、node/profile、external identifier、`entity_edges`、theme、事件/观测、Neo4j、UAT/prod/shared、Sync/Archive/Deliver。

## 冻结输入与精确范围

| 项目 | 固定值 |
| --- | --- |
| manifest | [`relation-write-manifest.json`](relation-candidate-artifacts/relation-write-manifest.json) |
| manifest SHA-256 | `7651e0b591df1e03838df00ebc9acd6101ebcc76da18a6a314ff478c9f42990e` |
| source candidate | `relation-candidate-review.json` / `00bda835f16bd4df84eaa6f83d63c53146ca7edb09049def84146932e672340f` |
| approved node source | `final-seed-candidate-artifacts/node-profile-seed-manifest.json` / `9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e` |
| exact rows | 96：`is_subcategory_of=95`、`is_component_of=1`、`input_to=0`、`depends_on=0` |
| physical constraints | 0 |
| identity | deterministic UUID over `from_id|relation_type|to_id`；ID 与 tuple 均唯一 |
| evidence | 每条含内部 artifact path+SHA、derivation rule、`ffb243e` 首遍与 `main-serenity` 第二遍 Review、固定 `verified_at` |

生成器 [`generate_relation_write_manifest.py`](relation-candidate-artifacts/generate_relation_write_manifest.py) 稳定排序并生成 ID；[validation](relation-candidate-artifacts/relation-write-manifest-validation.json) 断言 96/95/1/0/0、自环=0、tuple duplicate=0、ID duplicate=0、blocked/rejected/constraint included=0。该 manifest 只表达静态集合从属或组成，不得外推为投入、依赖、供应瓶颈或事件传导。

## R1 runner 技术契约与证据

唯一入口是 `backend/cmd/entity-seed` 的 relation-only mode：

- dry-run 必须显式传 `-chain-node-relation-dry-run`；未来 Write 必须显式传 `-chain-node-relation-approved-data-write`，两者互斥且都要求冻结 manifest。
- relation mode 拒绝普通 entity seed、mapping、preflight/apply-scope、非默认 seed-dir 等参数；manifest 缺失、hash/count/type/evidence/ID 不匹配或 non-empty `physical_constraints` 在连接/写入前 fail-closed。
- dry-run 使用 `REPEATABLE READ READ ONLY` snapshot，读取真实 endpoint/ID/tuple 状态；不把合法输入固定报告为 created。
- Write runner 在一个 `SERIALIZABLE` PostgreSQL transaction 内完成：完整 baseline/schema preflight -> 全量 endpoint/ID/tuple plan -> 全量 write -> 每条精确读回 -> 全局 96/95/1/0/0、evidence、自环、tuple、orphan、constraint 与 842/842/1,169/331/Goose17 precommit assertions -> commit。
- 任一 plan 冲突、首条后中途 insert、单条读回或全局 precommit assertion 失败均 rollback；不循环调用逐行独立 transaction，不允许部分提交。

新鲜 targeted tests 覆盖第二项 plan 失败时 0 insert、第一条 insert 成功而第二条失败时整层 rollback、precommit 失败 rollback、read-only snapshot 0 writes、CLI flag 隔离、manifest hash/确定性契约。真实 local 只读 dry-run 已执行一次且 exit 0：`created=96, updated=0, unchanged=0`，按类型 `95/1`；未运行 backup 或 Write。

## R2 Write 前 fresh preflight

未来只有用户明确批准命名操作后，才在同一维护窗口刷新并保存脱敏证据：

1. 环境精确为 local `tidewise-local-postgres` / `tidewise_local` / PostgreSQL 16.14；显式运行时配置失败即停止，不得回退其他数据库。
2. Goose=17，无 migration 18；migration 17 已独立验收。`chain_node_relations=0`、`chain_node_physical_constraints=0`。
3. 两表各 12 列；relation constraint/index 为 c7/f2/p1/u1/4 indexes，physical constraint 为 c7/f2/p1/3 indexes，额外 trigger=0；任何 schema snapshot 漂移即停止。
4. active chain_node=842，active profiled endpoint=842，external identifiers=1,169，`entity_edges=331`；Phase A 12 类非目标 counts/checksums、orphan/duplicate/blank-key 与已验收 evidence 一致。
5. 冻结 manifest SHA 与 96/95/1/0/0、每条 deterministic ID、evidence/provenance/verified_at、端点绑定全部通过；真实 DB dry-run 必须再次得到 `96/0/0`。
6. 无长事务、其他 writer、等待锁或冲突操作。任一 identity/version/hash/count/schema/baseline/activity 漂移即 fail-closed，0 writes。

## Fresh recovery evidence

schema 层稳定 backup 是 Goose=16、migration17 前基线，不能冒充本 data 层的直接恢复证据。本层未来 maintenance window 必须在 Write 前创建新的稳定 PostgreSQL 16 custom-format backup，内容精确为 Goose17 + 842 node/profile + 1,169 mappings + 0 relation/constraint：

```text
docker exec tidewise-local-postgres pg_dump -U <runtime-only-user> -d tidewise_local -Fc --no-owner --no-acl > <stable-backup-path>
pg_restore --list <stable-backup-path>
pg_restore --schema-only --file=<temporary-schema-decode> <stable-backup-path>
pg_restore --data-only --file=<temporary-data-decode> <stable-backup-path>
```

执行证据必须记录稳定路径、UTC 时间、size、SHA-256、container/database identity、PostgreSQL/pg_dump/pg_restore 16.14 版本、TOC 数量与 schema/data full decode 结果；不得记录 secret 或完整连接串。任一 backup/hash/TOC/decode/baseline metadata 失败即 0 writes。该备份未经隔离 restore，不得声称 restore verified；事务内失败依赖 rollback，提交后只允许另行 Review 的 forward-fix 或恢复操作。

## 唯一 Write 路径

只有命名操作获得明确 R2 授权且同一窗口的 preflight 与 fresh backup 全部通过，才允许执行一次：

```text
APP_ENV=local DATABASE_PASSWORD='<runtime-only>' \
  go run ./cmd/entity-seed \
  -chain-node-relation-manifest ../openspec/changes/refactor-industry-chain-node-foundation/relation-candidate-artifacts/relation-write-manifest.json \
  -chain-node-relation-approved-data-write
```

禁止手工 SQL、普通 seed、第二执行入口、逐行 transaction、physical constraint write 或自动重试。唯一一次预期输出为 `created=96, updated=0, unchanged=0`；任一差异立即停止。

## Write 后立即只读 Query/assert

原子命令结束后无论成功或失败都立即只读验收：

1. Goose=17；relation 总数=96，类型=95/1/0/0，physical constraint=0。
2. manifest 96 个 ID、from/to/type/mechanism/condition/evidence/provenance/verified_at/status 逐条精确匹配；deterministic ID duplicate=0、tuple duplicate=0、自环=0、证据缺失=0。
3. 所有端点仍是 active profiled chain_node；endpoint/profile orphan=0，不存在 blocked/rejected/旧关系类型。
4. 842 active chain_node、842 profiles、1,169 mappings、`entity_edges=331`、12 类非目标 counts/checksums、schema snapshot 与写前完全不变。
5. 再运行相同 relation-only DB dry-run，只读预期 `created=0, updated=0, unchanged=96`。
6. 未写 physical constraint、其他表或 Neo4j，未运行 migration18、UAT/prod/shared 或第二次 Write。

任一 assertion 失败、部分写、hash/count/schema/baseline 漂移均立即停止并提交脱敏差异；不得自动重试、Down、forward-fix 或恢复。package Review 或未来 Write 成功均不授权 task 2.8/2.9、Neo4j 或生命周期后续。
