## MODIFIED Requirements

### Requirement: 联盟组织实体
系统 SHALL 支持把 `OPEC+`、`G7`、`WTO` 等跨经济体联盟组织、国际组织或多边协调机制保存为独立实体类型，并与国家政策机构、经济体和市场实体区分。

#### Scenario: 保存联盟组织
- **WHEN** 系统初始化或保存已批准的联盟组织实体
- **THEN** 必须在 `entity_nodes` 中保存 `entity_type=alliance_org`、`layer_code=alliance`、名称、规范名称、aliases 和状态，并在 `alliance_org_profiles` 中只保存 abbreviation、受控多值 categories、leadership summary 与 influence scope summary

#### Scenario: 区分联盟组织和政策机构
- **WHEN** 实体表示跨多个经济体的国际组织、联盟、论坛或规则协调机制
- **THEN** 系统必须使用联盟组织实体表达，而不是把该实体保存为单一经济体下的政策机构

#### Scenario: 不使用实体标签扩展联盟分类
- **WHEN** 联盟同时属于多个业务领域
- **THEN** 系统必须使用 profile 的受控 categories，不得新增实体标签机制或复用事件标签，Neo4j 仍使用单一 `Entity` label

## ADDED Requirements

### Requirement: Economy 聚合身份 schema 边界
系统 SHALL 在 economy profile 中保存可校验的 identity kind，使国家、地区经济体、超国家聚合和全球聚合使用同一实体基础但保持语义隔离。

#### Scenario: 校验 economy profile
- **WHEN** 系统写入或更新 economy profile
- **THEN** 必须校验 identity kind 与 country code/ISO、currency 和 region 的组合，拒绝把 `GLOBAL` 或 `EU` 聚合 code 声明为普通主权国家 ISO identity
