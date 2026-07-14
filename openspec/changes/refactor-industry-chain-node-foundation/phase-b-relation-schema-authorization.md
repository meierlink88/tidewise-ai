# Phase B relation schema R2 authorization Review package

## 状态、命名操作与授权边界

- 命名操作：`phase-b-relation-schema`。
- 风险等级：**R2**；未来只允许在当前 local development PostgreSQL 执行 migration 17 的 schema Write。
- 当前状态：**主对话已明确授权并完成一次 migration 17 Write 与立即 Query/assert；当前等待独立写后验收。**实际脱敏结果见 [execution evidence](phase-b-relation-schema-execution-evidence.md)。
- 前置事实：主对话已批准 checkpoint `f8b3406` 的分层 evidence contract；该批准只允许准备本包，不授权 schema/data Write。
- 本包只覆盖 relation schema 层；不授权 relation/physical-constraint data、task 2.7 runner、node/profile/mapping 变更、migration 18、Neo4j、UAT/prod/shared、Sync、Archive 或 Deliver。

## 固定环境、版本与执行目标

| 项目 | 固定值 / 执行前要求 |
| --- | --- |
| environment | local development only |
| PostgreSQL container / database | `tidewise-local-postgres` / `tidewise_local` |
| PostgreSQL | 16.14 |
| execution-window Goose before | `16`；`000017_add_chain_node_relations.sql` 必须是唯一 pending migration |
| allowed executor | `backend/cmd/dbmigrate` |
| allowed target | `-target-version 17`；实际 applied 只能是 `000017` |
| migration source / SHA-256 | `backend/migrations/000017_add_chain_node_relations.sql` / `a7f4406084c04e56f1f85bf24dfa10d58f112d3e5732b2e52a2849cbb514a3df` |
| expected data baseline | 842 active `chain_node`、842 `chain_node_profiles`、1,169 external identifiers、`entity_edges=331` |
| expected new-table state | 两张目标表及同名 constraint/index 均不存在 |

上述值来自已验收 Phase A evidence，仅作为执行包的冻结预期，不冒充未来 maintenance window 的 live 读回。实际 R2 获批后必须在同一维护窗口刷新 read-only preflight；任一身份、版本、对象或 count 漂移即 fail-closed。

## Migration 17 精确 schema diff

`backend/migrations/000017_add_chain_node_relations.sql` 的 Up 只创建以下对象，不包含 `INSERT`、`UPDATE`、`DELETE`、旧 topology/constraint 迁移或 Neo4j 操作。

### `chain_node_relations`

| 项目 | 精确契约 |
| --- | --- |
| columns | 12 列：`id`、`from_chain_node_entity_id`、`to_chain_node_entity_id`、`relation_type`、`mechanism`、`condition_note`、`evidence_note`、`provenance`、`verified_at`、`status`、`created_at`、`updated_at` |
| primary key | `id UUID PRIMARY KEY` |
| endpoint FK | from/to 均引用 `chain_node_profiles(entity_id) ON DELETE RESTRICT`；因此端点必须具有 chain-node profile，后续 repository 仍须验证 entity 当前 active/type |
| relation type | 仅 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on` |
| text/time/status | `mechanism`、`evidence_note`、`provenance` 非空白；`condition_note` 可空但非 NULL 时非空白；`verified_at NOT NULL`；status 默认 `active` 且仅 active/inactive；时间默认 `now()` |
| structural checks | 禁止 from=to 自环；`UNIQUE(from_chain_node_entity_id, to_chain_node_entity_id, relation_type)` |
| query index | `(to_chain_node_entity_id, relation_type)` |
| input/dependency mutual exclusion | partial unique expression index `(from, to, lower(btrim(mechanism))) WHERE relation_type IN ('input_to','depends_on')` |

不创建 `contains`、`supplies_to`、`substitutes_for`、`transmits_to`，不引用或复用 `entity_edges`，不恢复 membership/topology 双表或 `industry_chain_entity_id`。

### `chain_node_physical_constraints`

| 项目 | 精确契约 |
| --- | --- |
| columns | 12 列：`id`、`chain_node_entity_id`、`chain_node_relation_id`、`constraint_type`、`description`、`condition_note`、`evidence_note`、`provenance`、`verified_at`、`status`、`created_at`、`updated_at` |
| primary key | `id UUID PRIMARY KEY` |
| subject FK | node subject 引用 `chain_node_profiles(entity_id) ON DELETE RESTRICT`；relation subject 引用 `chain_node_relations(id) ON DELETE RESTRICT` |
| exact subject | 两个 nullable subject 恰有一个非 NULL |
| constraint type | 仅 migration 17 中固定的 10 个硬物理/产能/资源/工艺类型 |
| text/time/status | description/evidence/provenance 非空白；condition 可空但非 NULL 时非空白；verified_at 非空；status 默认 active 且仅 active/inactive；时间默认 `now()` |
| subject indexes | node subject 与 relation subject 各一个 partial 普通索引 |
| 明确延后 | 不创建 constraint semantic unique；identity 待出现可写 constraint 候选后独立 Review |

Down 仅按依赖顺序先 drop `chain_node_physical_constraints`、再 drop `chain_node_relations`。本包不授权 Down；migration 17 提交后若 Query/assert 失败，必须停止并提交新的 forward-fix/recovery Review，不得现场手工改表。

## Write 前 fresh read-only preflight

实际 R2 maintenance window 必须以独立 guard 和标准 migration check 刷新并保存脱敏证据：

1. 环境身份精确为 local `tidewise-local-postgres` / `tidewise_local` / PostgreSQL 16.14；显式运行时配置失败只能停止，不得回退其他数据库。
2. Goose current=16，仓库 migration 序列存在 17 且只有 17 pending；migration 17 源文件 SHA-256 与获批 checkpoint 一致。
3. `chain_node_relations`、`chain_node_physical_constraints` 及本 migration 的 index/constraint 名均不存在；catalog 中无同名 table/view/materialized view/sequence/function/trigger/rule 冲突或未知逻辑引用。
4. `entity_nodes=1,308`、active chain_node=842、`chain_node_profiles=842`、两者精确绑定=842、external identifiers=1,169（eastmoney=818、ths=351、双来源节点=241、双 taxonomy source code=13）、`entity_edges=331`。
5. Phase A 冻结的 12 类非目标 entity count/checksum、profile/mapping orphan、external triple/ID duplicate、blank/duplicate entity key 与已验收 evidence 完全一致。
6. 目标两表写前均不存在，因此预计 relation rows=0、physical constraint rows=0；96 条 candidate draft、51 blocked、7 rejected 与 4 blocked physical constraints 均不得在 schema 层写入。
7. 无长事务、无其他 writer、无等待锁或冲突 DDL。创建新表及 FK/index 会短暂取得 catalog 与被引用 `chain_node_profiles` 的 metadata locks；若无法在维护窗口立即取得锁则停止，不等待扩大影响。

任一版本、hash、count/checksum、对象、引用、锁或环境身份不一致，授权自动失效，必须保存差异并回到 Review；不得现场放宽断言、跳过 backup 或改用 apply-all。

## Fresh recovery evidence

当前 curated local 不视为 disposable。本层在 Write 前必须创建新的 PostgreSQL 16 custom-format backup，基线必须是 Goose=16、842 node/profile、1,169 mappings、两张 relation/constraint 表尚不存在。旧 Phase A backup 早于当前节点/mapping 状态，不能作为本层直接恢复基线。

未来执行证据必须记录稳定保存路径（不得依赖 Desktop worktree 生命周期）、创建时间、文件 size、SHA-256、PostgreSQL/`pg_dump`/`pg_restore` 版本、源环境 identity，并在 Write 前完成：

- `pg_restore --list` 成功且 TOC 非空；
- schema-only 与 data-only 全量解码成功，输出写入受控临时位置并按策略销毁；
- backup metadata 与 preflight 的 Goose/count/checksum 一致；
- 文件及 evidence 不包含 password、secret 或完整数据库连接串。

备份、hash、TOC 或任一全量解码失败即 0 writes。该 local backup 未经隔离 restore，不得声称 restore verified，也不得复用于 UAT/prod/shared。事务内失败依赖 rollback；提交后只允许经 Review 的 forward-fix，或另行明确授权的恢复操作。

## 唯一 Write 路径

只有主对话对命名操作 `phase-b-relation-schema` 明确授予 R2，且同一维护窗口的 fresh preflight 与 recovery evidence 全部通过后，才允许以运行时配置执行一次：

```text
TIDEWISE_DATABASE_URL='<runtime-only reviewed local DSN>' \
  go run ./cmd/dbmigrate -apply -target-version 17
```

标准 executor 必须报告实际 applied 只含 `000017`。禁止 apply-all、手工 SQL、Down、第二条执行路径、故意依赖后续 migration 失败或把 schema 与 relation/constraint data 合并。本次 migration 只允许 transactional DDL；任何 data row 变化均视为越界。

## Write 后立即只读 Query/assert

Write 原子结束后，无论命令成功或失败，都必须停止并运行只读 Query；全部断言通过后才可请求本层验收：

1. Goose current=17；实际 applied 仅 `000017`；无未知更高 migration 被应用。
2. 两张表存在，列、类型、nullability、defaults、PK、四个 FK `ON DELETE RESTRICT`、四类 relation CHECK、自环 CHECK、tuple unique、input/dependency partial unique、exact-one-subject CHECK、10 类 constraint CHECK 及三个普通/partial query index 与本包逐项一致。
3. 不存在额外 constraint semantic unique、JSONB、trigger、function、view、rule、旧 relation type、`entity_edges` FK 或 membership/topology 容器字段。
4. `chain_node_relations=0`、`chain_node_physical_constraints=0`；没有 relation/constraint candidate、node/profile、mapping、theme 或其他 data 被写入。
5. `entity_nodes=1,308`、842 active chain_node、842 profiles、1,169 mappings（818/351/241/13）、`entity_edges=331`，以及 12 类非目标 counts/checksums、orphan/duplicate/blank-key 断言与写前完全一致。
6. migration 状态显示 target 17 已到达且 pending 不再包含 17；相同 target 的计划应为空。除非未来明确授权把 no-op runner 复核纳入该 R2，否则不得为“幂等”再次执行 Write 命令。
7. 全部日志与 evidence 脱敏，且未连接、查询、写入、清理或 rebuild Neo4j。

任一 assertion 失败、出现部分 schema/data、保护基线变化或未知 catalog 对象，立即停止并报告；不得自动 Down、重试或进入 task 2.7。

## 顺序与下一层门禁

本包严格保持 `R2 package Review -> explicit R2 authorization -> migration 17 Write -> immediate Query/assert -> independent acceptance`。本文件提交只完成 package preparation；task 2.6 schema Write/Query 仍未完成。

schema Query 经主对话独立验收前，不得实现或执行 task 2.7 data runner，不得冻结数据库可执行 relation manifest，不得写入 96 条分类/组成候选。即使 schema 层未来通过，relation data 仍须先完成 R1 atomic runner 技术验收，再提交独立 `phase-b-relation-data` R2 package。`input_to`、`depends_on` 与 physical constraint 当前 write-ready 均为 0；Neo4j 始终不在本 change 的执行范围。

## R2 execution record

主对话明确批准命名操作 `phase-b-relation-schema` 后，maintenance window 先刷新环境、Goose=16/17-only pending、migration SHA、目标对象不存在、842/842/1,169/331、818/351/241/13、12 类保护 checksum、完整性与 activity 门禁；随后创建并验证新的 PostgreSQL 16.14 custom archive。所有断言通过后，唯一标准 target-version=17 Write 实际 applied 仅 `000017`。

立即只读 Query/assert 确认 Goose=17、两张目标表 schema/constraint/index 精确一致且 rows 均为 0；Phase A 数据、12 类 checksum、mapping aggregates 与完整性断言全部不变，额外 trigger/function/view/rule/JSONB 均为 0。未执行第二次 Write、relation/constraint data、migration 18 或 Neo4j 操作；task 2.7 仍等待本层独立验收。
