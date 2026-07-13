# Task 1.12 Schema/TDD implementation Review

## 状态与禁止事项

- first-batch data contract checkpoint `cd4b072` 已由主对话验收通过，task 1.11 完成。
- 本 checkpoint 只提交 migration/domain/repository/validator、workbook parser 与 dry-run/report 代码及测试。
- 未执行 migration、cleanup、seed，未连接或写入 PostgreSQL/Neo4j，未进入 task 1.13 或 Phase B。

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

## Domain / repository 契约

- 通用 `domain.EntityExternalIdentifier` 只校验必填 identity 与 active/inactive 状态，不把 PostgreSQL schema 锁成来源 enum。
- first-batch seed validator 额外限制 source 为 `eastmoney/ths`，taxonomy 为 `industry_sector/concept_sector/index_sector`。
- identifier ID 由 `source_system|source_taxonomy_type|external_code` 确定性生成；entity ID 由全新 `chain_node:<stable_suffix>` key 使用现有确定性 helper 生成，不接受旧 UUID 覆盖。
- memory/PostgreSQL repository 都要求目标是 active chain_node；相同 external identity 不得换绑 entity_id，只允许同实体 unchanged 或更新 external_name/status。
- PostgreSQL upsert SQL 返回 `created/updated/unchanged/invalid_target/identity_conflict`，不会把冲突静默当作成功。

repository 代码当前没有被默认 seed service 调用；在 task 1.15 schema Query 与 task 1.17 node/profile Query 验收前，不存在可执行 mapping data 写入路径。

## Workbook parser 证据

parser 只读取 Sheet「标准化保留」与「原名保留明细」，支持 XLSX shared string 与 inline string，并执行：

1. 解析 canonical、原始名称与宽边界标记；
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
  "mapping_count": 0,
  "provider_counts": {"eastmoney": 0, "ths": 0},
  "dual_source_node_count": 0,
  "nodes": [
    {
      "entity_id": "deterministic-new-uuid",
      "entity_key": "chain_node:approved_stable_suffix",
      "canonical_name": "已批准名称",
      "aliases": ["已批准原名"],
      "definition": "经 Review 的定义",
      "boundary_note": "必要时填写包含与排除边界",
      "action": "created"
    }
  ],
  "mappings": [],
  "blockers": ["mapping eastmoney:example taxonomy is unresolved"],
  "conflicts": []
}
```

这是单节点字段形状示例，不代表首批实际 report；它同时展示未消歧 mapping 不进入可写列表，且 approved counts 校验继续阻断 `ready`。首批完整 report 必须在 13 个代码逐项消歧、842 个 identity 与 definition/boundary 全部 Review 后重新生成，不得由当前解析证据推定为可执行 seed。

## 验证覆盖

- migration 字段、FK、唯一约束、普通索引、非空/status CHECK 与 forbidden structure 静态测试；
- domain 必填与状态 table-driven validation；
- memory repository create/unchanged/update/rebind conflict；
- PostgreSQL SQL target/type/status、tuple conflict 与 action contract；
- workbook aliases、逐行 mapping、external name 与组合 taxonomy 阻断；
- dry-run count、稳定 identity、aliases、definition/boundary、snapshot 冲突与 approved expectations。

## 后续门禁

主对话验收本 checkpoint 后仍只允许进入 task 1.13 cleanup Review。cleanup Write、external identifier schema Write、node/profile seed Write、mapping data Write 都需要各自单独授权与写后 Query。
