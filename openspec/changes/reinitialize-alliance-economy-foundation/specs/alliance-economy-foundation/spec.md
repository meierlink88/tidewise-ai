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

### Requirement: Alliance 权威 Manifest 收敛
系统 SHALL 使用版本化且穷尽的 approved alliance manifest 定义重新初始化后的最终 active alliance 集合，并通过 forward convergence 处理现有 identity，不得把 upsert 或新增数量视为完成。

#### Scenario: 穷尽处置现有 Active Alliance
- **WHEN** 系统准备 alliance Write Review
- **THEN** manifest 必须列出最终 active keys，并为每个现有 active `alliance_org` 给出 `keep`、`merge` 或 `inactivate`；任何未列出项必须阻断 Write，不得静默清理

#### Scenario: 处理 Reject 或 Defer
- **WHEN** 现有 alliance 候选最终为 reject、defer 或未进入批准 active set
- **THEN** 系统必须把它列为待 Review 的 inactivate diff，展示原因、identity、profile、关系影响和预计 counts；未经批准必须保持原状态并阻断完整收敛

#### Scenario: 合并重复 Alliance Identity
- **WHEN** 两个现有实体被批准 merge 为同一 alliance
- **THEN** 系统必须保留批准 target 的稳定 key/UUID，保留 source identity 但将其 forward inactivate，并逐项审阅 aliases、profile 和关系影响；最终不得存在两个 active 重复实体

#### Scenario: 验证最终 Active 集合
- **WHEN** alliance convergence Write 完成
- **THEN** Query 必须证明 PostgreSQL active alliance keys 与 approved manifest active keys 集合相等，并证明没有未经批准的 delete、merge 或 inactivate

#### Scenario: 禁止破坏性重置
- **WHEN** 系统实现或执行 alliance convergence
- **THEN** 只能使用版本化 forward migration/convergence 和幂等事务，不得使用 `TRUNCATE`、无谓词 DELETE、清空重灌或历史 rollback

### Requirement: Economy 保留优先与异常收敛
系统 SHALL 保留不属于当前批准联盟成员全集的合法 economy；只有 approved economy exception manifest 中逐项确认的 identity 冲突、重复或明确错误项可以 merge/inactivate。

#### Scenario: 保留非成员 Economy
- **WHEN** 某合法 economy 不在当前批准联盟的 formal-active 成员全集中
- **THEN** 系统必须保持其稳定 key、UUID、profile 和 active 状态，不得因非成员身份删除、停用或合并

#### Scenario: 审阅 Economy 异常处置
- **WHEN** economy identity 审计发现重复、冲突或明确错误
- **THEN** 候选 diff 必须展示原因、旧/新 identity、关系影响和预计 counts，并在独立 Review 批准后才可 forward merge/inactivate

#### Scenario: 验证未误伤合法 Economy
- **WHEN** economy convergence Query 执行
- **THEN** 除 approved exception manifest 列出的实体外，所有受保护合法 economy 的 key、UUID 和 status 必须与 Write 前快照逐项一致

### Requirement: 自动化验证边界
系统 SHALL 通过自动化测试验证 alliance profile、economy identity、候选依赖链和排除边界，普通测试不得访问真实外部网络或真实数据库。

#### Scenario: 运行普通后端测试
- **WHEN** 开发者运行相关 Go tests 与 `go test ./...`
- **THEN** 测试必须使用 fixture、fake、table-driven tests、sqlmock 或明确隔离的 integration boundary 覆盖 profile、ISO、identity、candidate gating、dry-run、幂等与冲突行为
