## MODIFIED Requirements

### Requirement: 联盟组织实体
系统 SHALL 支持把 `OPEC+`、`G7`、`WTO` 等跨经济体联盟组织、国际组织或多边协调机制保存为独立实体类型，并与国家政策机构、经济体和市场实体区分。

#### Scenario: 保存联盟组织
- **WHEN** 系统初始化或保存 approved 联盟组织
- **THEN** 必须在 `entity_nodes` 中保存 `entity_type=alliance_org`、`layer_code=alliance`、name、canonical name、由缩写派生并按 NFKC + casefold 去重的 aliases 和 status，并在 `alliance_org_profiles` 中只保存 abbreviation、leadership summary 与 influence scope summary

#### Scenario: 区分联盟组织和政策机构
- **WHEN** 实体表示跨多个经济体的国际组织、联盟、论坛或规则协调机制
- **THEN** 系统必须使用联盟组织实体表达，而不是把该实体保存为单一经济体下的政策机构

#### Scenario: 不从大类或子类生成实体标签
- **WHEN** 联盟 Excel 包含大类或子类
- **THEN** 系统不得把这些列写入 profile、创建实体标签或复用事件标签；Neo4j 仍使用单一 `Entity` label 且不在本 change 执行任何投影操作

## ADDED Requirements

### Requirement: Economy Schema 保持现状
系统 SHALL 保持 `economy_profiles` 的现有 `country_code/currency_code/region` 结构，不因本次 local business-data rebuild 引入全局 identity/schema 改造。

#### Scenario: 不新增 Economy Identity 列
- **WHEN** migration 为 alliance profile 做最小演进
- **THEN** 不得新增 `economy_profiles.identity_kind`、区域/货币 policy 列或平行 economy profile 表

#### Scenario: 不新增 Entity Key 全局唯一
- **WHEN** importer 为 frozen 79 economy 实现幂等
- **THEN** 必须优先使用 approved stable keys 与现有 repository preflight，不得为本批新增全局 `entity_nodes.entity_key` 唯一索引；若无法安全实现，必须回到 R0 Review
