## ADDED Requirements

### Requirement: 联盟与 Economy 分层 seed 准入
系统 SHALL 在正式 seed 前分别审阅联盟与 economy 候选，并复用一致的现有稳定 identity；候选材料、Review 清单和 dry-run 不等于写入授权。

#### Scenario: 审阅联盟候选清单
- **WHEN** 系统准备联盟 seed
- **THEN** 必须先提交 schema/data contract 和逐项候选清单，展示 `approve/reject/merge/defer`、entity key、canonical name、aliases、profile、来源、现有数据差异与冲突，未确认项不得进入正式 seed

#### Scenario: 联盟确认后审计 Economy 差异
- **WHEN** 已批准联盟的官方 formal active 成员全集形成
- **THEN** 系统必须与现有 economy 实体做集合与 identity 差异审计，复用一致的 entity key/UUID，并为缺失成员形成包含中文名、英文名/aliases、identity kind、ISO 3166 code、currency、region 和来源的候选清单

#### Scenario: Economy 候选独立 Review
- **WHEN** economy 差异审计发现缺失或 identity 冲突
- **THEN** 必须提交独立候选 Review，未获确认不得生成可执行 seed、写 PostgreSQL 或生成最终关系候选

#### Scenario: 分层幂等写入与查询
- **WHEN** 后续 Apply 获得有状态授权
- **THEN** 必须按 alliance Write/Query 再 economy Write/Query 的顺序幂等执行，上一层 Query 未验收不得进入下一层，并输出 created、updated、unchanged、failed 和 identity 冲突统计

#### Scenario: 不从 CSV 自动生成 seed
- **WHEN** 联盟研究 CSV 包含组织候选、成员数、评级或第 69—85 条资源商品
- **THEN** loader、转换工具或 seed 流程不得将其直接写入正式 JSON、PostgreSQL 或 Neo4j

### Requirement: 联盟与 Economy seed 自动化验证
系统 SHALL 对联盟/economy migration、loader、validator、repository、dry-run 和 report 提供自动化验证。

#### Scenario: 运行验证
- **WHEN** 开发者验证本 change 实现
- **THEN** 相关包测试、migration 静态或可重复 integration tests、`go test ./...` 与 OpenSpec strict validation 必须通过，并证明未写入未审阅候选
