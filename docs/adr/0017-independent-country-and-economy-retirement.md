---
status: accepted
date: 2026-08-14
supersedes_in_part: 0010-event-semantic-entity-first-cross-type-resolution.md, 0013-data-entity-domain-and-projection-retirement.md
extends: 0016-tidewise-ai-2-object-schema-and-independent-region.md
---

# 独立 Country 事实与 Economy 退役

## 背景

Tidewise AI 1.0 将国家、全球范围、区域和跨国对象混合保存为 `economy` Entity，身份位于
`entity_nodes`，属性位于 `economy_profiles`。该模型无法与 2.0 已采用的独立 Region 事实和
逐对象 OpenSPG Schema 保持同一语义边界。

## 决策

- Data 的 Entity 领域建立独立 Country Object。`countries` 保存稳定 `COU_ + ISO3` 身份、
  中英文名称和可选战略事实；`country_region_links` 保存 Country 到 Region 的多对多关系。
- Country 不引用 `entity_nodes`，不建立 `country_profiles`，也不保留 Economy 兼容身份、
  双读、双写或回退路径。
- Country 从 PostgreSQL、Data、Biz、Service、OpenAPI、HTTP、认证和组合根完整贯通。
  读写调用方只使用版本化 Data API；Region 集合通过独立幂等 PUT 在一个事务中整体替换。
- `doctype/country.schema` 使用仓库批准的 OpenSPG MarkLang 描述 Country 属性和多值 Region
  关系。PostgreSQL 继续拥有长度、唯一性、外键和时间戳约束。
- Event Semantic 将独立 Country 识别为 `country` 并允许正式 Country ID 成为 Entity Link；
  Research Analysis 与 Research Graph 从 Country 权威表解析，不创建 shadow Entity。
- Market、Company、Person 和 Industry Chain 的活动国家引用改为 Country ID；
  Industry Chain 通过可空 `primary_country_id` 表达主要国家范围。旧 Sector 持久化表已在
  migration 000015 退役，本次不恢复历史表；仅保留已执行 migration 记录。
- `000046` 删除 `economy_profiles`、指向 Economy 的旧 Entity/Event 关系与 Economy
  Entity 行。包含 Economy candidate 的完整 Event Semantic submission（包含 lease、candidate/review
  snapshot、Signal 与 Impact）及依赖它的完整 Research publication aggregate 一并退役，
  不保留快照与可查询事实不一致的残缺聚合。旧混合 Economy 事实不自动转换，Country
  主数据、语义事实与 Research publication 由另行审核的 2.0 发布过程重建。

## 与既有权威的关系

- 取代 ADR-0013 中 Economy 继续作为通用 Entity/Profile 事实的部分；其他尚未重构的
  Object 可以暂时继续使用 `entity_nodes`。
- 取代 ADR-0010 中国家形态的 Event Semantic 候选依赖 Economy UUID 的部分；PostgreSQL
  最终校验、候选隔离和 Evidence 门禁继续有效。
- 延续 ADR-0016 的逐 Object Schema 与独立持久化方向，并建立 Country–Region 正式关系。

## 发布与回滚

这是不支持混合版本的协调式破坏性切换。发布前停止旧写入并取得 PostgreSQL 快照，然后
应用完整 migration ledger；`000046` 在同一事务内清除旧 Economy 身份、无法继续表达的
Entity/Event 引用及依赖它们的整个 Research publication aggregate。该特例只存在于
快照保护的 2.0 cutover migration，不放开常规 Research 不可变事实的修改权限。随后部署新
Data Service，并通过 2.0 路径发布审核后的 Country
事实。若失败，必须同时恢复切换前 PostgreSQL 快照与上一版应用；不得只重建 Economy 表、
只回滚应用或运行 `000046` down。
