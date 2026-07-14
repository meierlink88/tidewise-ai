## Purpose

定义联盟与经济体基础在 local 探索环境中的最小身份/profile 契约、45/79/133 冻结重建基线、跨域保护与独立执行授权边界。

## Requirements

### Requirement: 联盟最小身份与 Profile
系统 SHALL 使用 `entity_nodes` 保存联盟的 name、canonical name、派生 aliases 和 status，并仅在 `alliance_org_profiles` 保存 abbreviation、leadership summary 与 influence scope summary。

#### Scenario: 映射四个联盟业务字段
- **WHEN** importer 读取 approved alliance artifact
- **THEN** 必须把名称映射到 `entity_nodes.name/canonical_name`、缩写映射到 `alliance_org_profiles.abbreviation`、核心主导方映射到 `leadership_summary`、核心影响范围说明映射到 `influence_scope_summary`，profile 不得重复 name

#### Scenario: 校验联盟字段
- **WHEN** 联盟准备进入 rebuild
- **THEN** abbreviation 必须 `btrim` 后最长 32、非空时进入 aliases 且不要求全局唯一；leadership/influence 必须 `NOT NULL`、无 default、`btrim` 后非空且分别不超过 500/1000 字符

#### Scenario: 排除非目标字段
- **WHEN** 输入包含 categories、大类、子类、成员数、全球占比、约束力级别、影响力评级或其他 sheet
- **THEN** 系统不得写入这些字段、生成实体标签或复用事件标签；成员数只由 active `member_of` 计算

### Requirement: Economy 保持现有 Profile Schema
系统 SHALL 使用现有 `economy_profiles(entity_id, country_code, currency_code, region)` 表达 approved 79 条 economy，不为本批新增 identity kind、区域/货币规则或全局 entity-key 约束。

#### Scenario: 保存 Approved Economy
- **WHEN** importer 验证 79 条 frozen economy target
- **THEN** 必须按 approved stable key、名称/aliases、country code、currency 和 region 写入现有结构，并保证所有 133 条关系端点可解析

#### Scenario: 保持聚合边界
- **WHEN** approved artifact 包含 EU 或 global 聚合 economy
- **THEN** 系统不得用聚合 economy 替代国家成员，不得为 global aggregate 生成 `member_of`，且不要求新增数据库列表达该业务边界

#### Scenario: 现有 Schema 无法表达时停止
- **WHEN** R1 证明某条 approved economy 无法以现有三字段或 stable-key repository 机制安全表达
- **THEN** 必须提交最小证据回到 R0 Review，不得自行新增 `identity_kind`、区域/货币 policy 或全局 `entity_key` 唯一索引

### Requirement: Approved 45/79/133 重建基线
系统 SHALL 只以 frozen approved artifact 定义 local 探索环境重建后的联盟、经济体与 formal-active 成员关系集合。

#### Scenario: 冻结重建目标
- **WHEN** 系统加载本 change artifact
- **THEN** 必须验证 45 alliance、79 economy、133 `economy -> alliance_org member_of` 及对应版本/checksum；任何漂移必须阻止执行并重新 Review

#### Scenario: 废止旧 Disposition 执行语义
- **WHEN** 系统发现旧 223 条 `member_of` 的 keep、preserve 或 proposed-inactivate 分类
- **THEN** 只能把它们视为历史候选证据，不得作为 cleanup/rebuild importer 的输入或保护策略

#### Scenario: 验证重建后精确集合
- **WHEN** latest manifest rebuild 完成
- **THEN** Query 必须证明 alliance=45、79 target economy/profile 与 formal-active `member_of`=133，15 个 non-target economy/profile 保留，全部端点存在且 active、方向正确、无孤儿或重复，并且同一 artifact 复跑不产生额外变化

### Requirement: Local Scoped Cleanup 前置审计
系统 SHALL 在任何破坏性 cleanup 授权前，以只读方式穷尽审计目标表、FK、关系类型/count 和跨域业务事实。

#### Scenario: 枚举依赖与跨域关系
- **WHEN** R1 准备 Package 4 Review package
- **THEN** 必须列出 `entity_nodes`、alliance/economy profiles、`entity_edges`、external identifiers 和所有引用 economy/alliance UUID 的 profile/FK，并分别报告 `member_of`、`has_market` 及其他关系的方向和 count

#### Scenario: 已确认的跨域事实保护
- **WHEN** economy 与 market、index、benchmark、industry chain、company、person 或其他实体存在不在 45/79/133 中恢复的跨域事实
- **THEN** 必须保留全部现有 economy/profile 与这些事实，并以 count/hash 验证不变；任何其他 alliance incident edge、未知 FK 或审计漂移必须阻止 R3 cleanup

#### Scenario: 限定 Local 豁免
- **WHEN** 主对话审阅 cleanup 包
- **THEN** 无 backup、rollback 或恢复演练的豁免只适用于明确识别的 local 探索环境，不适用于 UAT、prod 或 shared

### Requirement: 分离 Cleanup 与 Rebuild 授权
系统 SHALL 将破坏性的 scoped local cleanup 标为 R3，将 latest manifest rebuild 标为 R2，并要求两个执行包分别获得明确授权。

#### Scenario: 执行 Scoped Cleanup
- **WHEN** 主对话明确授权 4.1 R3 且环境、表、谓词、relation types、counts/hashes、跨域处置和停止条件与 Review package 一致
- **THEN** 系统可以精确清理获批 scope，并必须在进入下一包前 Query 证明该 scope 为零且未授权跨域事实未变化

#### Scenario: 执行 Latest Manifest Rebuild
- **WHEN** 4.1 zero Query 已验收且主对话独立授权 4.2 R2
- **THEN** 系统可以使用 change-specific importer 重建 frozen 45/79/133，并立即执行 exact/integrity/idempotency Query

#### Scenario: 禁止推定写入授权
- **WHEN** R1 完成、普通 Apply 获批、旧授权存在或上一执行包成功
- **THEN** 系统仍不得推定 4.1 或 4.2 的授权；任何环境、count/hash、dependency 或 assertion 漂移都必须停止

### Requirement: R1 自动化验证边界
系统 SHALL 通过 fixture、fake、table-driven tests、sqlmock 或隔离 integration boundary 验证 alliance schema、frozen artifact、change-specific importer 和 dependency audit，普通测试不得写真实数据库。

#### Scenario: 运行 R1 验证
- **WHEN** 开发者验证 Package 3.1
- **THEN** targeted tests、受影响 backend suite、共享 architecture/contract/migration tests、OpenSpec strict、task-design lint、diff/scope/secret 必须通过，并证明未引入通用 importer framework、economy identity schema、entity-key 全局唯一或 Neo4j 变更
