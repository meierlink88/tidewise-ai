## ADDED Requirements

### Requirement: Formal-Active Member Of 重建范围
系统 SHALL 只把 approved formal-active economy membership 重建为 `economy -> alliance_org member_of`，并把其他关系语义排除出本 change。

#### Scenario: 建立正式成员关系
- **WHEN** frozen artifact 中的 economy 是官方来源列明的 active 正式成员
- **THEN** importer 必须使用 economy → alliance_org 的 `member_of`，并保留 approved source、核验时间和 endpoint stable keys

#### Scenario: 排除非正式身份
- **WHEN** 候选身份是 observer、partner、applicant、suspended、former、participant、signatory 或 framework participant
- **THEN** 系统不得将其写为 active `member_of`，也不得在本 change 新增 relation type

#### Scenario: 排除可选关系
- **WHEN** alliance profile 包含核心主导方文本或候选材料提及下属机制
- **THEN** 不得自动生成 `led_by` 或 `part_of`；`participates_in`、`signatory_to` 也不在本 change

### Requirement: 废止旧 Member Of Disposition 策略
系统 SHALL 在 local 探索重建中以 latest approved 133 tuples 作为最终基线，不维护旧 223 条边的 keep/preserve/proposed-inactivate 执行策略。

#### Scenario: 读取旧 Disposition
- **WHEN** Review 历史包含 31 keep、160 preserve_unresolved、10 preserve_pending_retype 或 22 proposed_inactivate
- **THEN** importer 必须忽略其执行语义，不实现 preserve policy、forward inactivate 或全库 convergence engine

#### Scenario: 重建后核对集合
- **WHEN** 4.2 relationship rebuild 完成
- **THEN** active formal `member_of` 目标集合必须精确等于 approved 133 tuples，且每条边方向正确、两端存在且 active、无孤儿或重复

### Requirement: Cleanup 前关系依赖审计
系统 SHALL 在 R3 cleanup 授权前只读枚举 alliance/economy 端点涉及的全部 relation types 和 counts，区分本批 `member_of` 与跨域事实。

#### Scenario: 区分 Member Of 与 Has Market
- **WHEN** dependency audit 检查仓库 fixture 或真实 local `entity_edges`
- **THEN** 必须分别报告 economy → alliance_org `member_of` 与 economy → market `has_market`；fixture 已知基线为 223 与 40，真实 local counts 必须在执行前刷新

#### Scenario: 审阅其他跨域关系
- **WHEN** 发现 economy/alliance 与 market、index、benchmark、industry chain、company、person 或其他实体的关系
- **THEN** 必须列出 relation type、方向、端点、count/hash，并提交“删除并丢弃”或“保留/重建”决定；未决时阻止 cleanup

#### Scenario: 保护未授权跨域事实
- **WHEN** 4.1 R3 cleanup 获得授权
- **THEN** 只能改变 Review package 明确批准的 relation scope；未授权跨域 tuple 的 pre/post identity、端点、type 与 status 必须不变

### Requirement: Cleanup 与 Relationship Rebuild 独立授权
系统 SHALL 在 4.1 R3 中先完成 approved relation scope cleanup/zero Query，再在 4.2 R2 中重建 133 条关系；两者不得合并推定授权。

#### Scenario: Cleanup Zero Gate
- **WHEN** 4.1 完成
- **THEN** Query 必须证明获批旧 alliance/economy membership scope 为零，并证明未授权跨域事实未变化；失败时不得进入 rebuild

#### Scenario: Rebuild Exact Gate
- **WHEN** 4.2 获得独立授权并执行
- **THEN** Query 必须证明 133 条 approved tuples 集合相等、来源字段有效、端点 active、无孤儿/重复、方向正确且幂等复跑不改变集合

#### Scenario: 禁止 Neo4j 写入
- **WHEN** PostgreSQL cleanup 或 rebuild 执行
- **THEN** 不得连接、写入或重建 Neo4j；图投影由后续独立 change 处理
