## ADDED Requirements

### Requirement: 单一有限 AI 服务器算力基础设施批次
系统 SHALL 只在用户确认后处理一个以“AI服务器”为 entry node 的有限产业链数据批次，不得把本 change 扩张为 842 节点全量治理或通用数据平台。

#### Scenario: Review 唯一推荐范围
- **WHEN** AI 服务器算力基础设施批次进入 Proposal 业务 Review
- **THEN** 范围必须限定为最多两跳直接上游硬件与一跳直接部署节点，建议上限 18 个 chain_node 和 28 条静态关系
- **AND** 该范围在用户确认前不得被视为批准

#### Scenario: 包含直接硬件链
- **WHEN** 用户批准首批范围
- **THEN** 候选可以包含 AI 加速芯片/加速卡、HBM、高速互连/交换/光模块、服务器电源、液冷系统、AI服务器与 AI算力集群等直接硬件节点

#### Scenario: 排除超出边界内容
- **WHEN** 候选涉及完整半导体制造设备/材料链、矿产商品、通用云/IDC 运营、模型/软件/应用、公司/证券或 physical constraints
- **THEN** 系统必须排除该候选并留给后续独立批次

### Requirement: 首批仅使用四类静态关系
系统 SHALL 只为批准范围内 chain_node 提出 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`，并保持证据、方向和端点可审阅。

#### Scenario: 候选关系进入 manifest
- **WHEN** 双遍分析批准某条静态关系候选
- **THEN** change-specific manifest 必须保存有向端点、relation type、mechanism/condition、evidence/provenance 与 verified time

#### Scenario: 范围无法关闭
- **WHEN** 候选超过 hop/数量上限、需要新关系类型或依赖范围外产业
- **THEN** 当前批次必须停止扩张并把内容留给后续数据批次
