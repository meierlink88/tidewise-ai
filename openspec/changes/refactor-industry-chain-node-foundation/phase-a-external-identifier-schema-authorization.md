# Phase A external identifier schema R2 authorization package

## 状态与授权边界

- 命名操作：`phase-a-external-identifier-schema`。
- 风险等级：**R2**；只允许在当前 local development PostgreSQL 执行 migration 16 的 schema Write。
- 当前状态：**R2 条件式 authorization package 已准备，尚未授权、尚未执行**。
- task 1.14 `phase-a-legacy-industry-cleanup` 已由主对话以 checkpoint `f2bc90a` 验收；该验收不等于本包的 Write 授权。
- 本包不授权 node/profile seed、external identifier mapping data、relation/constraint、migration 17、Neo4j、UAT/prod 或任何其他 PostgreSQL Write。

## 固定环境与当前只读状态

| 项目 | 固定值 / 证据 |
|---|---|
| environment | local development only |
| PostgreSQL container / database | `tidewise-local-postgres` / `tidewise_local` |
| PostgreSQL | 16.14 |
| live Goose | version `15` applied；`000016` pending |
| allowed execution path | `backend/cmd/dbmigrate` |
| allowed target | `-target-version 16`，实际 applied 只能是 `000016_add_entity_external_identifiers.sql` |
| current entity baseline | cleanup 后 `entity_nodes=466`；不得由 schema Write 改变 |
| current external identifier table | 不存在；migration 16 前置状态 |
| fresh read-only check | 2026-07-14：Goose JSON 返回 current=`15`、pending/remaining=`000016`；容器报告 PostgreSQL 16.14 |

写前必须在同一维护窗口再次以 `REPEATABLE READ READ ONLY` 完成环境、Goose、catalog、writer/长事务和受保护实体断言。任一漂移即 fail-closed，不得现场修改范围或改用 apply-all。

## Migration 16 精确 schema diff

`backend/migrations/000016_add_entity_external_identifiers.sql` 只执行以下 schema 变更：

| 对象 | 规格 |
|---|---|
| `entity_external_identifiers` | 新建通用实体外部标识表；Write 前必须不存在 |
| `id` | `UUID PRIMARY KEY`，由后续逐行 mapping 契约提供确定性 ID；本层不插入数据 |
| `entity_id` | `UUID NOT NULL REFERENCES entity_nodes(id) ON DELETE CASCADE` |
| `source_system` | `TEXT NOT NULL`；非空白检查；首批值由后续 mapping 层限制为 `eastmoney` / `ths` |
| `source_taxonomy_type` | `TEXT NOT NULL`；非空白检查；不可用 PostgreSQL enum 锁死扩展性 |
| `external_code` | `TEXT NOT NULL`；非空白检查；保留来源原始代码格式 |
| `external_name` | `TEXT NOT NULL`；非空白检查；保留来源平台原始名称 |
| `status` | `VARCHAR(32) NOT NULL DEFAULT 'active'`；检查值为 `active` 或 `inactive` |
| `created_at` / `updated_at` | `TIMESTAMPTZ NOT NULL DEFAULT now()`；由后续 data layer 负责更新语义 |
| 外部 identity 唯一约束 | `UNIQUE(source_system, source_taxonomy_type, external_code)`；禁止同一外部标识换绑实体 |
| entity 查询索引 | 普通索引 `(entity_id, source_system, source_taxonomy_type)` |
| 明确不创建 | 四列冗余唯一索引、JSONB、source mapping 表、profile 字段、seed/data rows、trigger、function、view、Neo4j projection |

当前 migration 的 session 门禁为 `tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified`。该值只防止误执行，不替代人工授权、recovery evidence 或 Query/assert。

## Recovery evidence

本包选择 `backup`，不声明当前 curated local 为 disposable，也不把已撤销 restore rehearsal 宣称为 verified。当前只读备份证据已生成并通过 archive 检查：

| 证据 | 值 |
|---|---|
| stable backup | `/Users/meierlink/.local/share/tidewise-ai/backups/postgresql/phase-a/20260714T015901Z/tidewise_phase_a_post_cleanup_pre_schema16.dump` |
| size / SHA-256 | `904964` bytes / `02b12193985b0fc88df8c04b3a8efe5c3312e2184513be6036676243f616a0e4` |
| archive integrity | PostgreSQL custom format；TOC `182` 行；`pg_restore --list`、schema-only 全量解码、data-only 全量解码均通过 |
| restore status | 未执行隔离 restore；`backup_verified=false` 保持不变 |
| source boundary | 只来自 `tidewise_local`，不包含凭据、连接串或环境 dump |

该 backup 只作为本 R2 schema 层的 recovery evidence；若文件不可访问、hash/size/TOC 漂移或来源环境不一致，立即停止并重新 Review。UAT/prod、共享环境或不可替代数据不得复用该 local package。

## Write 前 preflight 与预计影响

必须重新验证：

- container/database/PG 身份固定；Goose=15 且只有 migration 16 pending；无长事务、无其他 writer、无锁冲突。
- `entity_nodes=466`，12 类非目标 counts/checksums 与 task 1.14 写后 evidence 完全一致；`chain_node_profiles` / `theme_profiles` schema 与 rows 不变。
- `to_regclass('entity_external_identifiers') IS NULL`；不存在已有 table、index、constraint 或 function 同名冲突。
- catalog 仅预计新增 1 table、1 entity FK、1 主键、1 三列唯一约束、5 个 CHECK（4 个文本非空白 + 1 个 status 枚举）、1 三列普通 index；未知引用或触发器差异均阻断。
- `entity_external_identifiers` 预计写前 rows=0；本层不读取或转换任何工作簿，不产生 842 node、1,156 mapping 或 theme rows。

预计影响只有一次 transactional DDL：创建上述表、约束与普通索引。不存在 data delete/update/insert；不会触及 `entity_nodes`、profiles、`entity_edges`、event links 或旧结构。

## 唯一 Write 形状

获得主对话对本包的明确 R2 authorization 后，且仅在写前 preflight 与 recovery evidence 全部通过时，才可在运行时组装 DSN 和 session setting：

```text
TIDEWISE_DATABASE_URL='<runtime-only reviewed local schema DSN with dedicated session setting>' \
  go run ./cmd/dbmigrate -apply -target-version 16
```

禁止无 target apply-all、手工 SQL、Down、第二执行路径、故意触发 migration 17 失败或把 schema/data 两层合并。migration 15 已完成，target 16 必须精确命中；标准 executor 的 `applied` 必须来自锁内 before/after state，不能把 selected 计划列表冒充实际结果。

## Write 后立即 Query/assert

Write 原子结束后无论成功或失败，都必须先停止并执行只读 Query；只有全部通过才可提交该层验收：

1. Goose=current `16`，实际 applied 只有 `000016`，无更高 migration；migration 16 的 session token 未写入数据。
2. `entity_external_identifiers` 存在且仅包含批准列、PK、entity FK `ON DELETE CASCADE`、三列唯一约束、5 个 CHECK（`source_system`、`source_taxonomy_type`、`external_code`、`external_name` 非空白，以及 `status` 枚举）与三列普通索引；不存在四列冗余唯一索引、JSONB、trigger/function/view。
3. `entity_external_identifiers` rows=`0`；没有 node/profile、mapping、theme、relation 或其他 data rows 被写入。
4. `entity_nodes=466`、12 类非目标 counts/checksums、`entity_edges=331`、event links、profile rows/schema、orphans/duplicates/blank key 与 task 1.14 post evidence 完全一致。
5. schema-only 重复 Query 报告 already-present/unchanged；不得再次执行 migration，也不得扩大为 apply-all。
6. 查询和日志不得出现连接串、密码或 secret；未连接、写入、清理或 rebuild Neo4j。

任一列/约束/index/version/count/checksum/orphan/duplicate 不符，或出现任何 data row、未知 catalog 对象、环境漂移、锁/事务异常，立即停止。事务内失败依赖 rollback；提交后不得执行 Down，须另行提交 forward-fix package。

## 顺序与下一层边界

本包顺序为 `schema Review -> schema Write -> schema Query/assert`。schema Query 经主对话验收前：

- 不得进入 1.16 final seed candidate Review 以外的 Write；不得执行 842 node/profile seed。
- 不得执行 1,156 external identifier mapping、theme 数据、relation/constraint 或 Neo4j 操作。
- 不得把本包 Review/Write/Query 授权推定为下一层授权；每一层必须有独立命名 R2 package。

## Authorization request

当前只提交本 R2 package 供主对话 Review，**不请求立即执行**。主对话需逐项确认 local 环境、migration 16 only、recovery evidence、schema diff、预计零 data rows、before/after assertions、停止条件与排除范围；确认前不得运行 migration 16。

## R0/R1 package verification

- migration 16 SQL 已审计：与本包列、FK、唯一约束、索引、session 门禁和 no-data 范围一致；未发现需要生产代码修复的偏差。
- 现有 `backend/migrations/entity_external_identifiers_schema_test.go` 覆盖 session gate、主要列、级联 FK、三列唯一约束、普通索引、非空 checks、status checks、禁止 JSONB/source mapping/四列唯一/insert；本轮补充的测试仅加强 defaults/timestamps/不可逆 Down 契约。
- 本 checkpoint 不连接 PostgreSQL 执行 migration 16，不写 seed/mapping/relation，不连接或写入 Neo4j。
