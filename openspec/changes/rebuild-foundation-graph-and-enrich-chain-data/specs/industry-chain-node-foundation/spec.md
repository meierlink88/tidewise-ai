## ADDED Requirements

### Requirement: 有限首批产业链范围门禁
系统 SHALL 在完善产业链数据前通过人工 Spec gate 冻结一个有限、代表性、可在本 change 关闭的首批范围，不得把全部 842 个既有 chain_node 的长期遍历或治理纳入本 change。

#### Scenario: 提交首批范围候选
- **WHEN** 系统准备 AI 算力基础设施、动力电池、光伏制造或其他候选范围供 Review
- **THEN** 每个候选必须列出 entry nodes、包含与排除边界、事件研究价值、关系语义覆盖、可用权威来源、预计节点/关系/constraint 上限和停止条件

#### Scenario: 人工冻结首批范围
- **WHEN** 用户选择一个或多个首批候选
- **THEN** 系统必须冻结 entry nodes、包含/排除、数量上限、来源时效和停止条件后才能生成 final candidate manifest
- **AND** 未进入范围的既有节点不得被机械遍历、修改或补全

#### Scenario: 首批范围无法关闭
- **WHEN** 研究发现范围需要扩张到未批准产业、超过数量上限或缺少可靠边界
- **THEN** 系统必须停止并将扩张内容留给后续批次，不得自行放宽 Spec gate

### Requirement: 首批节点关系与物理约束候选合同
系统 SHALL 对首批细分 chain_node、四类静态关系和 physical constraints 生成可审阅候选，并为每项记录 identity、定义/边界、来源、支持证据、反例、成立条件、置信度和 disposition。

#### Scenario: 生成细分节点候选
- **WHEN** 研究提出新的或需修订的细分 chain_node
- **THEN** 候选必须说明与 842 节点基线的复用、合并或新增关系，并提供 canonical name、aliases、definition、boundary、stable identity 提案和冲突检查

#### Scenario: 生成四类关系候选
- **WHEN** 研究提出 `is_subcategory_of`、`is_component_of`、`input_to` 或 `depends_on`
- **THEN** 候选必须记录有向端点、mechanism、condition、evidence/provenance、verified time、反例和置信度
- **AND** 不得生成 `contains`、`supplies_to`、`substitutes_for` 或 `transmits_to`

#### Scenario: 生成强证据物理约束候选
- **WHEN** 研究提出 physical constraint
- **THEN** 候选必须绑定明确 node 或 relation subject，提供可定位强外部来源、source URL、verified time、条件、反例和物理机制
- **AND** 只有名称/definition、价格、政策、情绪、市场表现或投资判断支持时必须保持 blocked 或 rejected

#### Scenario: 不输出投资建议
- **WHEN** 候选或证据涉及公司、证券或市场影响
- **THEN** 系统只能表达产业事实、约束与市场理解辅助，不得生成买卖建议、收益承诺或直接投资推荐
