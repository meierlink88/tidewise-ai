## ADDED Requirements

### Requirement: 业务 Review 定义有限产业链数据批次
系统 SHALL 只在用户批准一个有限、可关闭的首批产业链范围后，才从已有 chain_node 向下探索细分节点；不得把本 change 扩张为 842 节点全量治理或通用数据平台。

#### Scenario: 范围先于候选
- **WHEN** Package 2 准备首批产业链数据完善
- **THEN** 必须先展示选择理由、起始方向、包含/排除边界和停止条件供用户 Review
- **AND** 范围未批准前不得生成或写入节点/关系候选

#### Scenario: 范围无法关闭
- **WHEN** 细分探索需要超出已批准边界、引入新关系类型或遍历全部节点
- **THEN** 当前批次必须停止扩张，并把超界内容留给后续数据批次

### Requirement: 有限批次只使用四类静态关系
系统 SHALL 只为已批准范围内 chain_node 提出 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`，并保持有向端点、证据和来源可审阅。

#### Scenario: 四类关系进入 final 候选
- **WHEN** 某条静态关系候选进入用户 Review
- **THEN** 必须展示有向端点、relation type、证据/来源、反例与置信度
- **AND** physical constraints 或其他关系类型不得进入本批次
