## ADDED Requirements

### Requirement: 联盟关系类型、方向与候选层
系统 SHALL 对 `member_of`、`led_by`、`part_of` 使用明确方向、端点类型与证据，并在同一候选 package Review 中以 `member_of` 作为不被其他两层阻塞的 MVP 核心关系。

#### Scenario: 建立正式成员关系
- **WHEN** economy 是联盟官方来源列明的 active 正式成员
- **THEN** 系统必须使用 `economy -> alliance_org` 的 `member_of`，在候选 Review 中展示 formal active 身份与冲突报告，并在正式 edge 保存官方来源名称、URL 和核验时间

#### Scenario: 排除非正式或非 active 成员身份
- **WHEN** 候选身份是 observer、partner、applicant、suspended 或 former
- **THEN** 系统不得将其写为 active `member_of`；如需结构化表达，必须先通过关系契约 Review，不能自行新增 relation type

#### Scenario: 建立有证据的主导关系
- **WHEN** 联盟存在可解析且有证据的明确核心主导 economy 或 alliance organization
- **THEN** 系统可以在独立候选层提出 `alliance_org -> economy/alliance_org` 的 `led_by`；Excel 的“核心主导方”只保存为 `leadership_summary`，不能单独充当关系证据，“多边”“轮值”或无法解析的文本不得生成虚假实体或关系

#### Scenario: 建立正式隶属关系
- **WHEN** 某已批准下属机构或机制与上级联盟组织存在可审计的正式隶属关系
- **THEN** 系统可以在独立候选层提出 `alliance_org -> alliance_org` 的 `part_of`，不得用合作、主题相关或共同成员替代隶属

#### Scenario: 可选附录不阻塞核心 MVP
- **WHEN** `led_by` 或 `part_of` 在 Package 2 截止时证据不足、存在未决冲突或未获同一次业务 Review 批准
- **THEN** 必须将其排除出本次 MVP，alliance、economy 与 `member_of` 核心可以继续排队，且不得顺带写入未审阅关系

### Requirement: Member Of 候选与写入闭环
系统 SHALL 在联盟 manifest 确认后，于 Package 2 内依次完成 economy 审计并生成版本化、穷尽的 formal-active `member_of` manifest，在 package 末统一 Review，并在 PostgreSQL 写入前后验证端点、成员身份、来源、stale disposition 与集合相等；禁止仅新增而保留过期 active edge。

#### Scenario: 生成成员关系候选
- **WHEN** 联盟 manifest 已确认，且 Package 2 已依次形成官方成员全集并完成 economy diff/exception/protection 审计
- **THEN** 候选清单必须逐条包含方向、两端 entity key、formal active 身份、官方来源、核验时间、现有 edge 差异与冲突，并穷尽覆盖每条现有 active `member_of`；不得依赖 Excel 成员数字段

#### Scenario: 分类过期 Active Member Of
- **WHEN** 现有 active `member_of` 不在最新 approved formal-active tuple set 中
- **THEN** manifest 必须将其分类为 `former`、`withdrawn`、`suspended`、`source_conflict` 或 `alliance_identity_convergence`，展示旧/新 identity、provenance、关系影响和预计 counts，并在 Package 2 业务 Review 与 R2B 精确执行包授权后才可转 inactive

#### Scenario: 阻止未决来源冲突
- **WHEN** 正式成员来源互相冲突且 Review 尚未决定 disposition
- **THEN** 系统必须阻止 member convergence Write，不得自动保留、停用或创建 relation type

#### Scenario: Master Data Query 未验收时阻止关系写入
- **WHEN** R2A master-data Write 后 Query 尚未获得人工验收
- **THEN** 系统不得请求或执行 `member_of` Write

#### Scenario: 写入后核对正式成员集合
- **WHEN** `member_of` Write 完成
- **THEN** Query 必须证明所有端点存在且 active、无重复/悬空/错误方向，PostgreSQL active `member_of` tuple set 与 approved manifest 集合相等，并按联盟将 active edge 集合和数量与同一官方正式成员来源逐项核对

#### Scenario: 保留 Inactive Edge 审计
- **WHEN** stale `member_of` 获批转 inactive
- **THEN** 系统必须保留原 edge identity 与 provenance，使用 forward convergence 更新状态，不得删除 edge、清空关系表或把历史身份改造成未经批准的新 relation type

#### Scenario: 在同一 R2B 包中执行已批准可选关系
- **WHEN** `led_by` 或 `part_of` 已具备完整证据且在 Package 2 同一次业务 Review 中获批
- **THEN** R2B 可以将其与 approved `member_of` 一并列入精确 manifest、影响范围和 Query 断言；否则必须排除，不得增加阻塞核心 MVP 的额外人工门禁
