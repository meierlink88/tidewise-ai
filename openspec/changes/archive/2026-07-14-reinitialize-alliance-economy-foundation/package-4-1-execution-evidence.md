# Package 4.1 R3 Scoped Local Cleanup 执行证据

执行时间：2026-07-14。环境：`APP_ENV=local`、数据库身份 `tidewise_local`。本记录不包含连接串、账号、实体 UUID 或其他敏感信息。

## 授权与输入

- 唯一执行入口：`cmd/entity-seed` 的 change-specific `-alliance-economy-cleanup-approved-local`。
- frozen manifest：45 alliance / 79 economy / 133 formal-active `member_of`，SHA-256 `118573006c341830f3c977a994d12a537fe2d1ac8174a818d8007fe98e87a55d`。
- 写前 Goose：17；`000018_reinitialize_alliance_economy_foundation.sql` 保持 pending。
- 写前 dependency checksum：`c312731a72705ccf293ee3a24a58ed0aa07fe099375e12345402695600de0b9b`。
- 写前 protected checksum：`32dc4eaf1132a6a18d5850b0cfd19ab0536931652557b384c0a6edaf67992cdf`。
- 写前关键 count：alliance=10、alliance profile=10、economy=50、economy profile=50、economy → alliance `member_of`=223、economy → market `has_market`=40；foreign key inventory=26，未发现其他 alliance incident edge。

## 唯一 R3 写入结果

- deleted `economy -> alliance_org member_of`：223。
- deleted `alliance_org_profiles`：10。
- deleted `alliance_org entity_nodes`：10。
- 未执行 migration、rebuild、Neo4j、手工 SQL、backup/restore 或自动重试。

## 写后 Query / 保护断言

- alliance=0、alliance profile=0、economy → alliance `member_of`=0。
- economy=50、economy profile=50、economy → market `has_market`=40。
- `tracks_index`=43、`observes_benchmark`=10、indirect index profile=43；foreign key inventory 仍为 26，无其他 alliance incident edge。
- protected checksum 仍为 `32dc4eaf1132a6a18d5850b0cfd19ab0536931652557b384c0a6edaf67992cdf`；post-cleanup dependency checksum 为 `049625b97bd70f1a87693ea7dc6621ec9d1daca2d569de7db3777095bdfd2557`，其变化仅反映已批准的 alliance/profile/member_of cleanup。
- Goose 仍为 17，000018 仍 pending。

4.1 到此停止。4.2 latest manifest rebuild、migration 000018、Neo4j 与任何后续生命周期动作均未获本包授权。
