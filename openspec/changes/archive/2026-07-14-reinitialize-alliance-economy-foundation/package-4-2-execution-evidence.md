# Package 4.2 Latest Manifest Rebuild Evidence

执行时间：2026-07-14。本证据记录已经单独授权的 local `tidewise_local` Package 4.2 重建结果，以及其后用户明确批准的一次同一冻结 artifact 零写幂等验证；不包含连接串、密码、实体 UUID 或其他敏感信息。

## 已有业务基线与本次 R1 修复

- migration `000018` 已完成，Goose=18；本次未执行 migration、cleanup、Neo4j 或其他 seed。
- 已存在的 Package 4.2 业务集合为 alliance/profile=45/45、economy/profile=94/94（79 target + 15 non-target）、formal-active `economy -> alliance_org member_of`=133、`has_market`=40。
- 发现的伪幂等根因是三个 `ON CONFLICT DO UPDATE` 无条件执行并刷新 `updated_at`；dependency audit 的全量 `protected_checksum` 也错误包含 target economy 行的时间戳。
- R1 最小修复将 entity/profile/member_of upsert 限制为批准业务字段 `IS DISTINCT FROM` 时才更新，并令 CTE 返回实际写入数；精确既有目标若返回非零写入数即 fail-closed 回滚。
- rebuild 内部保护仍是 15 non-target economy/profile 与所有非 `member_of` 跨域事实；dependency audit 的全量 `protected_checksum` 现在使用去除 `created_at`/`updated_at` 的稳定业务字段 fingerprint。该新语义与 Package 4.1 当时的历史 checksum 不作数值比较，仍兼容 Goose 17/18 alliance profile 的动态字段读取。

## Fresh Read-only Preflight

- frozen manifest SHA-256：`118573006c341830f3c977a994d12a537fe2d1ac8174a818d8007fe98e87a55d`。
- audit：`blocked=false`；alliance/profile=45/45，economy/profile=94/94，`member_of`=133，`has_market`=40。
- target/non-target 分层为 79/15；无 orphan、duplicate tuple 或 approved-field mismatch。
- dependency checksum：`3d7fb06f6bb7185ef8370aa6772c90ad5cc036f3cd6e99c537baa519c303c8e4`。
- 稳定全量 protected checksum：`388e2945842053f5eb0674e7901b2f8b351b743c21951b9f43cb314fea987c26`。

## Single Authorized Idempotency Validation

仅通过 change-specific `entity-seed` 入口对同一冻结 manifest 执行一次；结果：

```text
alliance/profile=45/45
target economy/profile=79/79
non-target economy/profile=15/15
member_of=133
orphans/duplicate_tuples/mismatches=0/0/0
entity_writes/profile_writes/member_writes=0/0/0
```

立即只读 post-audit 保持 `blocked=false`，`has_market`=40，dependency checksum 与稳定 protected checksum 均与 preflight 完全一致。零写入计数证明没有因本次同 artifact 验证刷新业务行的 `updated_at`。

## 停止边界

Package 4.2 已完成；本 checkpoint 不授权 Neo4j/graph rebuild、其他 migration、其他环境或数据、PR、Sync、Archive、Deliver 或 Package 5。
