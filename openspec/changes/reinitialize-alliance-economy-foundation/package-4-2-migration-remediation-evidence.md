# Package 4.2 Migration Remediation Evidence

执行时间：2026-07-14。此证据仅记录 migration 000018 的恢复结果与 audit 兼容修复；不包含连接串、密码、实体 UUID 或其他敏感信息。

## Migration 000018 状态

- 第一次 migration 调用未继承授权 session setting，fail-closed；未应用 SQL。
- 认证阶段失败的调用未建立数据库连接，未应用 SQL。
- 经用户批准的受控子 shell 连接级 PostgreSQL `options` 重试后，正式 `dbmigrate/Goose` 入口将 `000018_reinitialize_alliance_economy_foundation.sql` 从 Goose 17 成功 apply 到 18。
- 之后的只读 migration status 报告：current version=18，pending=nil，remaining=nil。

## Audit Compatibility Remediation

000018 删除了旧 profile 字段，而 dependency fingerprint audit 曾静态读取这些字段，导致 migration 后 audit fail-closed。R1 remediation 改为将 `alliance_org_profiles` 行转换为 `to_jsonb`，按键存在性生成：

- Goose 17 旧字段的原有 fingerprint 拼接语义；
- Goose 18 `abbreviation`、`leadership_summary`、`influence_scope_summary` 的稳定 fingerprint。

纯 JSON table-driven 测试覆盖两个 schema 形态；真实 Goose 18 local 只读 audit 已成功，报告 alliance/profile/member_of=0/0/0、economy/profile=50/50、`has_market`=40、foreign key inventory=26、无其他 alliance incident edge，protected checksum 仍为 `32dc4eaf1132a6a18d5850b0cfd19ab0536931652557b384c0a6edaf67992cdf`。

## 停止边界

Package 4.2 rebuild 尚未执行，`tasks.md` 的 4.2 checkbox 保持未完成。此次 R1 checkpoint 不授权任何 PG 写入、Neo4j、其他 migration、Sync、Archive、Deliver 或 PR。
