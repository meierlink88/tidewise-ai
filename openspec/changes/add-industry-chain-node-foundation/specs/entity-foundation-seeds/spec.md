## ADDED Requirements

### Requirement: 静态产业链 seed
系统 SHALL 通过现有 entity foundation 边界初始化已批准的 industry chain、改进后的 chain node profile、membership、topology 和 physical constraint，并保持 stable key 与幂等 upsert。

#### Scenario: 加载两条试点候选
- **WHEN** AI 算力基础设施和半导体制造候选进入 Review
- **THEN** loader 必须展示每链约 10–15 节点及复用/新增依据，候选未批准前不得进入正式 seed

#### Scenario: 复用共享节点
- **WHEN** 两条链包含语义等价的 GPU、EDA、半导体设备、先进封装或其他已有节点
- **THEN** seed 必须复用 stable key，并通过独立 membership 表达不同链内角色

### Requirement: 物理约束 seed 门禁
系统 SHALL 只允许权威技术证据完整且人工 Review approved 的物理约束进入正式 seed。

#### Scenario: 拒绝 AI candidate 直接入库
- **WHEN** physical constraint 仅由 AI 生成或缺少权威技术来源
- **THEN** seed validator 必须阻止其进入 approved seed 或 PostgreSQL 事实

#### Scenario: 拒绝非物理约束
- **WHEN** seed 包含市场结构、供应商集中、认证、监管、融资、替代难度或当前投研结论
- **THEN** validator 必须返回明确错误

#### Scenario: 分层写入首批人工批准约束
- **WHEN** 用户逐项批准4条AI生成的physical constraint并为每条提供显式人工approval gate
- **THEN** 正式seed必须只包含这4条approved记录、保留`generated_by_ai=true`并使用已审阅direct provenance；其余11条仍只能留在review-only fixture

#### Scenario: physical constraint scope隔离
- **WHEN** 使用`industry-chain-physical-constraint` scope准备执行
- **THEN** batch必须仅含4条physical constraint，不得携带profile、membership、topology、entity edge、sector mapping或其他数据族；repository必须从已持久化事实锁定并校验同链active subject

### Requirement: 静态 seed report
系统 SHALL 在未来执行后分别报告 industry chain、chain node profile、membership、topology、physical constraint 和跨实体关系的 created、updated、unchanged、failed 数量。

#### Scenario: 输出可审阅统计
- **WHEN** 经批准的静态 seed 执行
- **THEN** report 必须按数据族输出统计，使每层可独立验收
