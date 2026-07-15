## ADDED Requirements

### Requirement: 在当前 842 个 chain_node 间建立 usable-map 四类关系
系统 SHALL 以重新冻结的当前 842 个 active chain_node 为硬边界，只在这些既有节点之间发现和审核 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`，并原样保留既有 100 条 accepted baseline。

#### Scenario: 冻结 842 节点基线
- **WHEN** Package 2 准备开始关系审计
- **THEN** 必须冻结 active chain_node 的 identity、count 与 hash
- **AND** 若最终 count 不是 842 或 identity 集合发生漂移，必须回到 Review，不得自行改写范围

#### Scenario: 全量候选发现覆盖完成
- **WHEN** 842 全量业务目标申请完成验收
- **THEN** 842/842 节点都必须参加候选发现与两遍独立检查
- **AND** 不得存在未处置候选、未解决异常或冲突；无事实节点允许保持无边

#### Scenario: 覆盖不强制伪造关系
- **WHEN** 某个节点没有满足语义与证据门槛的关系
- **THEN** coverage ledger 必须证明该节点已完成发现与双遍检查
- **AND** 不得为使节点拥有某类边而创建虚假关系或强制 node×relation_type 三态

#### Scenario: usable-map input_to
- **WHEN** A 的产出沿可解释产业链进入 B 的生产、产品、交付或运行过程
- **THEN** 可以登记方向为 A 到 B 的 `input_to`，即使当前节点集合缺少中间环节
- **AND** 名称邻近、市场相关、共同事件或宽泛行业相邻不得作为登记依据

#### Scenario: 既有节点集合是硬边界
- **WHEN** 某个候选关系无法由当前 842 个既有 chain_node 表达
- **THEN** 该候选必须阻断并记录无法表达的原因
- **AND** 不得创建细分 chain_node，不得修改节点主数据、profile、external identifier 或 identity

### Requirement: 全量研究可分批审阅但不降低关闭边界
系统 SHALL 允许将 842 节点的候选研究、double-check 与 Review 按节点群分批进行，并分别记录批次范围与逐节点发现覆盖进度；不得把局部批次视为 Package 或 change 完成。

#### Scenario: 分批候选进入 Review
- **WHEN** 某个审阅批次的既有节点关系候选完成
- **THEN** 必须展示有向端点、relation type、机制、条件、证据 tier/来源、反例、置信度与双遍 disposition
- **AND** physical constraints 或其他关系类型不得进入本 change

#### Scenario: 批次不构成关闭边界
- **WHEN** 某个研究批次完成候选审核
- **THEN** 未覆盖节点必须继续保留在 842 节点发现账本中
- **AND** 只有 842/842 全部完成候选发现与双遍检查时才能申请冻结 additive manifest
