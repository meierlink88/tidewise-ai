## MODIFIED Requirements

### Requirement: 联盟组织实体
系统 SHALL 支持把 `OPEC+`、`G7`、`WTO` 等跨经济体联盟组织、国际组织或多边协调机制保存为独立实体类型，并与国家政策机构、经济体和市场实体区分。

#### Scenario: 保存联盟组织
- **WHEN** 系统初始化或保存已批准的联盟组织实体
- **THEN** 必须在 `entity_nodes` 中保存 `entity_type=alliance_org`、`layer_code=alliance`、名称、规范名称、由缩写派生并按 NFKC + casefold 去重的 aliases 和状态，并在 `alliance_org_profiles` 中只保存最长 32 字符且不全局唯一的 abbreviation，以及无 default、`btrim` 后非空的 leadership summary 与 influence scope summary

#### Scenario: 区分联盟组织和政策机构
- **WHEN** 实体表示跨多个经济体的国际组织、联盟、论坛或规则协调机制
- **THEN** 系统必须使用联盟组织实体表达，而不是把该实体保存为单一经济体下的政策机构

#### Scenario: 不从大类或子类生成实体标签
- **WHEN** 联盟 Excel 包含大类或子类
- **THEN** 系统不得把这些列写入 profile、创建实体标签或复用事件标签；Neo4j 仍使用单一 `Entity` label

## ADDED Requirements

### Requirement: Economy 聚合身份 schema 边界
系统 SHALL 在 economy profile 中保存可校验的 identity kind，使国家、地区经济体、超国家聚合和全球聚合使用同一实体基础但保持语义隔离。

#### Scenario: 校验 economy profile
- **WHEN** 系统写入或更新 economy profile
- **THEN** 必须校验 identity kind 与 country code/ISO、currency 和兼容 region 的组合，限制 `MULTI` 为批准场景，拒绝把 `GLOBAL` 或 `EU` 聚合 code 声明为普通主权国家 ISO identity，并保证同一 code 只有一个 approved active economy

#### Scenario: 约束稳定 Entity Key
- **WHEN** 系统执行 entity identity preflight
- **THEN** 必须先报告并收敛空值或重复 `entity_key`，再建立全局唯一约束；merged source 必须保留自身不同 stable key，不得复用 target key
