## ADDED Requirements

### Requirement: 全量完善当前 842 个 chain_node 的四类关系
系统 SHALL 以重新冻结的当前 842 个 active chain_node 为本 change 完成范围，只在这些既有节点之间全量审计 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`。

#### Scenario: 冻结 842 节点基线
- **WHEN** Package 2 准备开始关系审计
- **THEN** 必须冻结 active chain_node 的 identity、count 与 hash
- **AND** 若最终 count 不是 842 或 identity 集合发生漂移，必须回到 Review，不得自行改写范围

#### Scenario: 全量覆盖完成
- **WHEN** 842 全量业务目标申请完成验收
- **THEN** 842/842 节点的四类关系都必须有已审核状态：有批准关系、不适用或证据不足
- **AND** 不得存在待研究、未处置候选、未解决异常或冲突

#### Scenario: 完整性不强制伪造关系
- **WHEN** 某个节点在某类关系上没有语义上应存在的关系或缺少强证据
- **THEN** 必须记录不适用或证据不足及理由
- **AND** 不得为使节点拥有四类边而创建虚假关系

#### Scenario: 既有节点集合是硬边界
- **WHEN** 某个候选关系无法由当前 842 个既有 chain_node 表达
- **THEN** 必须记录不适用或证据不足及理由
- **AND** 不得创建细分 chain_node，不得修改节点主数据、profile、external identifier 或 identity

### Requirement: 全量研究可分批审阅但不降低关闭边界
系统 SHALL 允许将 842 节点的候选研究、double-check 与 Review 按节点群分批进行，并分别记录批次范围与全量覆盖进度；不得把局部批次视为 Package 或 change 完成。

#### Scenario: 分批候选进入 Review
- **WHEN** 某个审阅批次的既有节点关系候选完成
- **THEN** 必须展示有向端点、relation type、证据/来源、反例、置信度与不适用/证据不足结论
- **AND** physical constraints 或其他关系类型不得进入本 change

#### Scenario: 批次不构成关闭边界
- **WHEN** 某个研究批次完成候选审核
- **THEN** 未覆盖节点必须继续保留在 842×4 覆盖账本中
- **AND** 只有 842/842 全部形成已审核状态时才能申请 Package 2 完成
