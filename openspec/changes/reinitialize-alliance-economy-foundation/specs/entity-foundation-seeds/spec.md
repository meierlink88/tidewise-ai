## ADDED Requirements

### Requirement: 联盟与 Economy 候选及 Master Data 准入
系统 SHALL 在正式 seed 前按 package 审阅联盟与 economy 候选，并复用一致的现有稳定 identity；候选材料、Review 清单和 dry-run 不等于写入授权。

#### Scenario: 审阅联盟候选清单
- **WHEN** 系统准备联盟 seed
- **THEN** 必须先提交四字段 data contract 和 45 条 Excel 逐项候选清单，展示源 sheet row、四字段源值、仅 U+200C 的 normalization、entity key、派生 abbreviation alias、空白最终 decision、现有 exact diff，并穷尽列出每个现有 active alliance disposition；未确认项不得进入正式 seed 或 convergence

#### Scenario: 执行已批准 Schema 校验
- **WHEN** 后续 loader/validator 验证联盟或 economy 候选
- **THEN** 必须执行 name/canonical 非空、abbreviation/aliases 长度与 NFKC + casefold 去重、非空 leadership/influence summary、四类 economy identity、受控 `MULTI`、active country code 唯一、全局 stable `entity_key` 唯一及兼容 region 规则；不得要求或写入 categories

#### Scenario: 联盟确认后审计 Economy 差异
- **WHEN** 已批准联盟的官方 formal active 成员全集形成
- **THEN** 系统必须与现有 economy 实体做集合与 identity 差异审计，复用一致的 entity key/UUID，并为缺失成员形成包含中文名、英文名/aliases、identity kind、ISO 3166 code、currency、region 和来源的候选清单

#### Scenario: Economy 与关系候选统一 Review
- **WHEN** economy 差异审计发现缺失或 identity 冲突
- **THEN** 必须先在 Package 2 内完成 economy diff/exception/protection，再连续生成穷尽的 `member_of` manifest，并将两者作为一个完整候选包提交业务 Review；未获确认不得生成可执行 seed 或写 PostgreSQL

#### Scenario: 统一 Master Data 幂等写入与查询
- **WHEN** 后续 Apply 获得有状态授权
- **THEN** R2A 必须把必要 schema、alliance 与 economy forward convergence 组成一个精确、单事务的 master-data 执行包，按 `Review → Write → Query` 输出 created、updated、inactivated、merged、unchanged、failed、identity 冲突和非目标保护统计；R2A Query 未验收不得进入独立 R2B relationship 执行包

#### Scenario: 生成 Alliance Exact Diff
- **WHEN** approved alliance manifest 与现有 PostgreSQL 比较
- **THEN** dry-run 必须输出最终 active set、每个 keep/create/merge/inactivate、原因、旧/新 identity、profile/alias/关系影响、预计 counts 和 manifest checksum；现有 active 未被穷尽覆盖时必须失败

#### Scenario: 生成 Economy Exception Diff
- **WHEN** approved economy candidates 与现有 economy 比较
- **THEN** dry-run 必须只把逐项批准的 identity 冲突、重复或明确错误列为 merge/inactivate，并把其他合法 economy 纳入保护快照，不得以联盟成员全集作为删除或停用范围

#### Scenario: 拒绝破坏性 Seed 重置
- **WHEN** seed 或 convergence 实现准备收敛现有数据
- **THEN** 系统必须拒绝 `TRUNCATE`、无谓词 DELETE、清空后重灌、未审阅 stale 清理或历史 migration rollback

#### Scenario: 不从 Excel 自动生成 seed
- **WHEN** 联盟 Excel 包含 45 条候选、大类、子类、成员数、评级或其他非目标列/sheet
- **THEN** loader、转换工具或 seed 流程不得将工作簿直接写入正式 JSON、PostgreSQL 或 Neo4j；只有人工批准的四字段 manifest 才能成为未来 mapping-only seed 输入

### Requirement: 联盟与 Economy seed 自动化验证
系统 SHALL 对联盟/economy migration、loader、validator、repository、dry-run 和 report 提供自动化验证。

#### Scenario: 运行验证
- **WHEN** 开发者验证本 change 实现
- **THEN** 相关包测试、migration 静态或可重复 integration tests、受影响交付边界完整 suite、共享 architecture/contract tests 与 OpenSpec strict validation 必须通过，并证明未写入未审阅候选；repo-wide full test 仅在项目规则触发时运行
