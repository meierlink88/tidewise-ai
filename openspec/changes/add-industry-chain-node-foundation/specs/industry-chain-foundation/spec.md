## ADDED Requirements

### Requirement: 独立产业链主数据
系统 SHALL 将产业链作为独立 `industry_chain` 实体保存，并以稳定代码、定义、范围、版本、来源和 Review 状态区别于节点、板块和动态主题。

#### Scenario: 创建 approved 产业链
- **WHEN** 产业链定义具备权威来源并通过人工 Review
- **THEN** 系统必须创建 `entity_type=industry_chain` 的实体与唯一 profile，只有 active + approved 才可进入正式投影

### Requirement: 链内 membership
系统 SHALL 复用全局 `chain_node` 身份，并以 membership 保存节点在特定产业链中的阶段、角色、顺序和核心性。

#### Scenario: 共享节点参与两条链
- **WHEN** 同一节点参与 AI 算力基础设施和半导体制造且链内角色不同
- **THEN** 系统必须复用同一节点实体并保存两条 membership，不得复制节点身份

### Requirement: 最小稳定拓扑
系统 SHALL 只使用 `supplies_to`、`depends_on` 和 `substitutes_for` 表达链内稳定拓扑，并校验方向、共存规则、同链 active membership 和来源。

#### Scenario: 保存 supplies_to
- **WHEN** 一个节点向另一个节点提供部件、材料、设备能力或服务
- **THEN** 系统必须按供应方到接收方保存 `supplies_to`

#### Scenario: 保存 depends_on
- **WHEN** 一个节点的正常运行依赖另一节点且不是简单直接供给事实
- **THEN** 系统必须按依赖方到被依赖方保存 `depends_on`

#### Scenario: 保存 substitutes_for
- **WHEN** 权威技术证据表明一个节点可替代另一个节点的相同功能位置
- **THEN** 系统必须保存单向 `substitutes_for`，不得自动生成反向关系

#### Scenario: 保存 canonical topology edge
- **WHEN** 业务事实是 supplier 向 receiver 直接供应
- **THEN** 系统必须只保存 supplier → receiver 的 `supplies_to`，不得再以 receiver → supplier 的 `depends_on` 重复表达同一事实

#### Scenario: 独立机制允许共存
- **WHEN** 相同端点存在独立证据证明直接供应和无法自然表达为供应的功能/基础设施依赖是两种不同机制
- **THEN** `supplies_to` 与 `depends_on` 才可共存，并必须分别保存证据

#### Scenario: 拒绝冲突拓扑
- **WHEN** 同一方向端点同时包含 `substitutes_for` 与 `supplies_to` 或 `depends_on`
- **THEN** validator 必须拒绝该冲突

### Requirement: 物理约束定义
系统 SHALL 使用 `industry_chain_physical_constraints` 保存相对稳定的物理/工程机制，并将 AI candidate、权威技术证据和人工 Review 作为 approved 事实门禁。

#### Scenario: 保存物理约束
- **WHEN** 节点或 topology edge 存在 power capacity、thermal dissipation、bandwidth、latency、production capacity、process yield、material purity、reliability、process cycle time、packaging density、equipment capacity、infrastructure access 或 physical expansion cycle 约束
- **THEN** 系统必须保存恰一同链主体、机制、物理边界、缓解路径、来源、核验时间和 Review 状态

#### Scenario: AI 只生成 candidate
- **WHEN** AI 根据 Serenity 启发识别潜在物理约束
- **THEN** 系统只能创建 candidate；缺少权威技术证据和显式人工 approval gate 时，不得进入 approved、正式 seed 或推理事实输入

#### Scenario: 人工批准后保留 AI 来源
- **WHEN** AI candidate 已取得权威技术证据、人工 Review 将 `review_status` 批准为 approved，且 seed/write 执行上下文携带显式人工 approval gate
- **THEN** 系统必须允许 approved 写入并保持 `generated_by_ai=true`；`ApprovedByHuman` / `IndustryChainApprovalGate` 不得持久化为物理约束事实字段，也不得替代 `review_status`

#### Scenario: constraint-only写入校验持久化subject
- **WHEN** approved physical constraint不携带同批membership或topology执行分层写入
- **THEN** repository必须在同一事务内以稳定顺序锁定已持久化node membership或topology edge，确认同链active后才写入，并在subject缺失、inactive或identity冲突时原子rollback

#### Scenario: 物理约束不进入 Neo4j
- **WHEN** future reasoning 完成 Neo4j chain/node 路径查询
- **THEN** 系统必须通过 repository 按 chain/node/topology edge 从 PostgreSQL 补充读取 physical constraints，不得依赖当前图投影返回 constraint

#### Scenario: 拒绝非物理和推理字段
- **WHEN** 约束包含 supplier concentration、qualification、know-how、regulation、substitution difficulty、审批、融资、severity、score、受益承压、利好利空或预测
- **THEN** validator 必须拒绝该数据

### Requirement: 两条 MVP 试点 Review
系统 SHALL 只将 AI 算力基础设施和半导体制造作为首批试点，每链约 10–15 节点、去重后约 20–30 节点，并逐项 Review membership、topology、physical constraint 和跨实体关系。

#### Scenario: 第二批不进入本 change
- **WHEN** 候选包含机器人、新能源汽车/储能或创新药/生物制造
- **THEN** 候选必须保留为第二批，不得进入本 change 正式 seed

### Requirement: 三层消费边界
系统 SHALL 区分静态主数据、未来 observation 和 future reasoning result；当前 change 不得拥有通用 observation governance。

#### Scenario: 后续 observation 消费静态 ID
- **WHEN** 后续 `add-industry-chain-observation-foundation` 设计动态观测
- **THEN** 它可以引用稳定 chain、node、topology 和 physical constraint ID，但 metric、quality、revision、idempotency 和 writer 必须由后续 change 自行定义

#### Scenario: 保持决策辅助定位
- **WHEN** future reasoning 展示产业链传导或瓶颈分析
- **THEN** 输出必须携带证据、不确定性和证伪条件，并定位为市场理解与决策辅助，不得表达为直接投资建议
