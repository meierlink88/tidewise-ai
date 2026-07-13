## ADDED Requirements

### Requirement: 联盟最小身份与 profile
系统 SHALL 使用 `entity_nodes` 保存联盟组织的 name、canonical name、aliases 和 status，并仅在 `alliance_org_profiles` 保存 abbreviation、受控多值 categories、leadership summary 与 influence scope summary。

#### Scenario: 保存联盟简称与 aliases
- **WHEN** 已批准联盟具有非空正式简称
- **THEN** 系统必须把简称保存到 profile 的 `abbreviation` 并同时加入实体 aliases；无正式简称时使用空值语义，不得把 `—` 保存为简称或 alias

#### Scenario: 保存受控多值分类
- **WHEN** 联盟候选同时属于政治、军事或经济等多个领域
- **THEN** 系统必须保存去重后的受控 category code 数组，不得保存“政治 / 军事”拼接字符串或 CSV 子类

#### Scenario: 排除分析性字段
- **WHEN** 候选材料包含成员数、全球占比、约束力级别或影响力评级
- **THEN** 系统不得把这些字段写入联盟 entity 或 profile，成员数必须由 active `member_of` 关系计算，其他字段留给后续 observation 或分析能力

### Requirement: Economy identity 与 ISO 边界
系统 SHALL 为 economy 保存明确的 identity kind，并区分主权国家、地区经济体、超国家聚合和全球聚合；内部稳定 identity、ISO 3166 code、currency 与 region 必须可审计且不得互相混淆。

#### Scenario: 保存国家或地区经济体
- **WHEN** economy 表示具有适用 ISO 3166-1 alpha-2 的主权国家或地区统计经济体
- **THEN** 系统必须保存对应 identity kind、规范中文名、英文名或 aliases、ISO code、currency、region 和来源，并复用已存在且一致的稳定 entity identity

#### Scenario: 区分欧盟聚合与成员国
- **WHEN** 候选范围同时包含 `economy:eu` 与欧盟成员国
- **THEN** 系统必须将前者标识为 supranational aggregate，保留各成员国独立 identity，且不得用聚合 economy 替代成员国 `member_of` 事实

#### Scenario: 区分全球聚合与国家
- **WHEN** 系统审计 `economy:global`
- **THEN** 必须将其标识为 global aggregate，不宣称其具有国家 ISO identity，也不得为其创建 `member_of`

### Requirement: 联盟确认到成员关系的依赖闭环
系统 SHALL 按“联盟确认 → 官方成员全集 → economy 差异审计与补齐 → 成员关系”的顺序准备数据，任一前置 Review 未通过时必须阻止下游候选冻结或写入。

#### Scenario: 联盟未确认时阻止 economy 冻结
- **WHEN** 联盟候选清单尚未逐项获得主对话确认
- **THEN** 系统不得冻结需要补充的 economy 范围，也不得生成关系候选

#### Scenario: 从正式来源形成成员全集
- **WHEN** 联盟清单已经确认
- **THEN** 系统必须为每个批准联盟读取可审计的正式成员来源，区分 active 正式成员与 observer、partner、applicant、suspended、former，并只用 formal active 形成 MVP economy 全集

#### Scenario: Economy 未确认时阻止关系候选
- **WHEN** economy 差异清单尚未单独获得主对话确认
- **THEN** 系统不得生成可执行 seed、写数据库或冻结 `member_of` 候选

#### Scenario: 验证成员数据完整性
- **WHEN** `member_of` PostgreSQL 写入完成
- **THEN** 每条关系的两端必须存在且 active，每个批准联盟的 active 正式成员集合和计算数量必须与同一官方来源逐项核对，CSV 成员数只能作为非权威 Review 对照

### Requirement: 候选参考范围隔离
系统 SHALL 将联盟研究 CSV 作为候选 Review 输入，而不是可执行 seed 或权威成员来源，并隔离不属于联盟组织的资源与商品条目。

#### Scenario: 审阅组织候选
- **WHEN** 系统使用 CSV 第 1—68 条准备联盟候选清单
- **THEN** 每条候选必须获得 `approve`、`reject`、`merge` 或 `defer` 结论，并展示 identity、名称、aliases、profile、来源与冲突，不得自动进入 seed

#### Scenario: 排除第 69—85 条
- **WHEN** 系统读取 CSV 第 69—85 条战略矿产和农产品
- **THEN** 必须排除出 `alliance_org` change，仅记录为未来 `chain_node`、`commodity` 或 observation 候选，不得自行创建实体、关系或后续 change

### Requirement: 自动化验证边界
系统 SHALL 通过自动化测试验证 alliance profile、economy identity、候选依赖链和排除边界，普通测试不得访问真实外部网络或真实数据库。

#### Scenario: 运行普通后端测试
- **WHEN** 开发者运行相关 Go tests 与 `go test ./...`
- **THEN** 测试必须使用 fixture、fake、table-driven tests、sqlmock 或明确隔离的 integration boundary 覆盖 profile、ISO、identity、candidate gating、dry-run、幂等与冲突行为
