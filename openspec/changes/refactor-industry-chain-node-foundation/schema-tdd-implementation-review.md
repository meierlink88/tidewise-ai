# Task 1.12 Schema/TDD implementation Review

## 状态与禁止事项

- first-batch data contract checkpoint `cd4b072` 已由主对话验收通过，task 1.11 完成。
- checkpoint `775afda` 与首轮 remediation checkpoint `12820f9` 的 task 1.12 Review 均未通过；最终 remediation checkpoint 修复 migration session lock/实际执行审计与 aliases 稳定排序，并保留此前 domain/repository/validator、workbook parser 与 dry-run/report 代码及测试。
- 主对话已批准最终 remediation checkpoint `a0547e6`，task 1.12 完成；该批准只允许进入 task 1.13 Cleanup Review，不构成任何 PostgreSQL/Neo4j Write 授权。
- task 1.12 实现期间未执行 migration、cleanup、seed，未连接或写入 PostgreSQL/Neo4j，也未进入 Phase B。

## Schema diff

新增 `000016_add_entity_external_identifiers.sql`，只创建空结构：

- `id UUID PRIMARY KEY`
- `entity_id UUID NOT NULL REFERENCES entity_nodes(id) ON DELETE CASCADE`
- `source_system TEXT NOT NULL`
- `source_taxonomy_type TEXT NOT NULL`
- `external_code TEXT NOT NULL`
- `external_name TEXT NOT NULL`
- `status VARCHAR(32) NOT NULL DEFAULT 'active'`
- `created_at/updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `UNIQUE(source_system, source_taxonomy_type, external_code)`
- 普通索引 `(entity_id, source_system, source_taxonomy_type)`
- 所有 identity 文本非空 CHECK 与 `active/inactive` status CHECK
- 在任何 DDL 前要求当前 PostgreSQL session 显式设置 `tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified`，普通 apply 不得隐式越过独立 schema Write 授权

migration 不创建 `sector_source_mappings`、`chain_node_source_mappings`、JSONB、冗余四列唯一索引，也不插入任何数据。Down 明确阻断破坏性回滚，提交后只允许 reviewed forward-fix。

## Migration target 操作路径

标准 `dbmigrate` 提供 `-target-version`，CLI 通过 `ServiceOptions.TargetVersion` 传到 `GooseExecutor.Apply`，有 target 时使用 `goose.UpToContext`，无 target 时保留原 `goose.UpContext` apply-all 行为。AutoApply 在读取执行前状态之前先取得 PostgreSQL advisory lock；locker 从 `*sql.DB` 取得并持有同一个 pinned `*sql.Conn`，acquire、release 与异常清理都由该 session 完成。重复 Lock、未持锁 Unlock、unlock=false、query/context 错误与 connection Close/Discard 错误均显式返回，不能静默泄漏锁。

Service 在持锁后读取 before `current/pending`，执行后再次读取 after `current/pending`。`GooseExecutor.Apply` 返回的 selected 只是执行计划，不作为审计事实；report 的 `pending` 来自 before snapshot，`remaining` 来自 after snapshot，`applied` 仅由 before 中已从 after pending 消失且版本不高于 after current 的 migration 推导。target 必须精确到达，target 以上 migration 必须保持 pending；无 target 必须确认 after pending 为空。任一状态矛盾或 unlock 失败都使命令失败。target 必须是当前版本或严格向前且真实存在的 migration；非数字、低于当前版本、越过不存在版本都在 Write 前失败。

- cleanup Write 获得独立授权后：`TIDEWISE_DATABASE_URL='<reviewed URL including options=-c%20tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified>' go run ./cmd/dbmigrate -apply -target-version 15`
- 命令成功时 `applied` 只包含 `000015`，`remaining` 明确包含 `000016`；随后立即执行 cleanup Query 并等待验收。
- cleanup Query 验收且 schema Write 另行获批后：`TIDEWISE_DATABASE_URL='<reviewed URL including options=-c%20tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified>' go run ./cmd/dbmigrate -apply -target-version 16`

真实连接 URL 只能由受控环境或 secret 注入，不写入仓库或日志。当前 pgx v5 配置不依赖 `PGOPTIONS`；session setting 通过 PostgreSQL URL 的 `options` 参数进入每个 migration connection。

不得使用无 target 的 apply 后依赖 `000016` 授权异常来切分层级；session setting 与 target version 必须同时满足，但都不能代替 Review、备份和 Write 授权。本 checkpoint 没有运行上述命令。

## Domain / repository 契约

- 通用 `domain.EntityExternalIdentifier` 只校验必填 identity 与 active/inactive 状态，不把 PostgreSQL schema 锁成来源 enum。
- first-batch seed validator 额外限制 source 为 `eastmoney/ths`，taxonomy 为 `industry_sector/concept_sector/index_sector`。
- identifier ID 由 `source_system|source_taxonomy_type|external_code` 确定性生成；entity ID 由全新 `chain_node:<stable_suffix>` key 使用现有确定性 helper 生成，不接受旧 UUID 覆盖。
- memory/PostgreSQL repository 都要求目标是 active chain_node；相同 external identity 不得换绑 entity_id，只允许同实体 unchanged 或更新 external_name/status。
- PostgreSQL repository 在单事务内先对完整 external identity 取得 `pg_advisory_xact_lock`，再以 `FOR SHARE` 锁定 active chain_node target、以 `FOR UPDATE` 读取已有 tuple。首次不存在时使用 `INSERT ... ON CONFLICT DO NOTHING RETURNING`；若并发 winner 抢先写入，当前事务立即重读并将不同 entity 或确定性 ID 判为 identity conflict，而不是 unchanged。
- sqlmock transaction 测试覆盖 created、同 entity update，以及“首次查询为空、并发 winner 插入不同 entity、重读后 rollback/conflict”的竞态控制流，不再由 mock 直接预设 action。

repository 代码当前没有被默认 seed service 调用；在 task 1.15 schema Query 与 task 1.17 node/profile Query 验收前，不存在可执行 mapping data 写入路径。

## Workbook parser 证据

parser 只读取 Sheet「标准化保留」与「原名保留明细」，支持 XLSX shared string 与 inline string，并执行：

1. 解析 canonical、原始名称与宽边界标记；原始名称 trim 后去重并按确定性字符串顺序排序，canonical 仍从 aliases 排除；
2. 将来源代码拆成逐行 mapping draft；
3. 从 provider 专属名称列恢复 external_name；
4. 只将单一“行业板块/概念板块/指数板块”规范化为 taxonomy；
5. 组合分类保持 `taxonomy_resolved=false`，不根据代码前缀猜测；
6. 拒绝未知 canonical、缺失外部名称和跨行 code 冲突。

使用临时只读测试实际读取已批准工作簿，得到：842 nodes、950 original names、79 wide-boundary nodes、1,156 mappings、eastmoney 811、ths 345、241 dual-source nodes、13 unresolved mappings。临时测试中的外部绝对路径未提交仓库。

## Dry-run/report 格式

`BuildFirstBatchDryRun` 不接收 repository 或数据库连接，只生成审阅 report。以下是字段形状示意，不是最终 seed：

```json
{
  "ready": false,
  "node_count": 1,
  "original_name_count": 1,
  "wide_boundary_node_count": 1,
  "mapping_count": 0,
  "provider_counts": {"eastmoney": 0, "ths": 0},
  "dual_source_node_count": 0,
  "nodes": [
    {
      "entity_id": "deterministic-new-uuid",
      "entity_key": "chain_node:approved_stable_suffix",
      "entity_type": "chain_node",
      "canonical_name": "已批准名称",
      "aliases": ["已批准原名"],
      "definition": "经 Review 的定义",
      "boundary_note": "必要时填写包含与排除边界",
      "status": "active",
      "action": "created"
    }
  ],
  "mappings": [],
  "blockers": ["mapping eastmoney:example taxonomy is unresolved"],
  "conflicts": []
}
```

这是单节点字段形状示例，不代表首批实际 report；它同时展示未消歧 mapping 不进入可写列表，且 approved counts 校验继续阻断 `ready`。首批完整 report 必须校验 842 nodes、950 original names、79 wide-boundary nodes、1,156 mappings、811/345 provider counts 与 241 dual-source nodes；13 个代码逐项消歧、842 个 identity 与 definition/boundary 全部 Review 前不得推定为可执行 seed。

node snapshot 同时按 entity ID、key、canonical 建索引并携带 entity_type、status、aliases、definition、boundary_note：发现既有记录时三索引必须齐全，三个 identity 与完整内容完全一致才是 unchanged；aliases 在进入 identity 与 checksum 前统一 trim、去重和稳定排序，因此同一 alias 集合仅输入顺序变化仍为 unchanged；aliases 集合或 profile 漂移为 updated；非 chain_node、inactive/merged、索引缺失或互相矛盾为 conflict。mapping snapshot 同时按外部 tuple 与确定性 ID 建索引，发现既有记录时双索引必须齐全：全新为 created，同 entity 的 name/status 漂移为 updated，完整一致为 unchanged，换绑、ID 漂移、索引缺失或矛盾为 conflict。任一 conflict/blocker 都令 `ready=false`。

## 验证覆盖

- migration 字段、FK、唯一约束、普通索引、非空/status CHECK 与 forbidden structure 静态测试；
- domain 必填与状态 table-driven validation；
- memory repository create/unchanged/update/rebind conflict；
- PostgreSQL transaction lock、target/type/status、首次空读/并发 winner 重读、tuple/ID conflict 与 action contract；
- workbook/draft aliases 的 trim、去重、确定性排序与仅顺序变化 unchanged；逐行 mapping、external name 与组合 taxonomy 阻断；
- dry-run node type/status/profile drift、mapping tuple/ID snapshot、重复执行幂等、wide-boundary=79、稳定 identity、aliases、definition/boundary、snapshot 交叉冲突与 approved expectations；
- migration locker 的 pinned connection ownership、重复/非法 lifecycle、acquire/unlock 失败清理与 Close；AutoApply 锁内 before/after 重读、target 15 只实际应用 `000015` 且保留 `000016` pending、selected 预测不进入 report、target 未到达或状态矛盾阻断、unlock 错误可见、无 target 兼容、非法/回退/跳跃拒绝，以及 CLI/Service/Executor 参数传递。

真实双连接并发集成测试代码使用专用 `TIDEWISE_EXTERNAL_IDENTIFIER_CONCURRENCY_TEST_DATABASE_URL`，会验证同 tuple 不同 entity 并发时恰好一个成功、一个 conflict、最终仅一行。该测试必须等待 `000016` schema Query 验收和单独数据库测试授权，本 remediation checkpoint 明确不设置该环境变量、不运行它、不连接数据库。

## 后续门禁

task 1.12 已由主对话验收；当前只允许完成 task 1.13 Cleanup Review。cleanup Write、external identifier schema Write、node/profile seed Write、mapping data Write 都需要各自单独授权与写后 Query。
