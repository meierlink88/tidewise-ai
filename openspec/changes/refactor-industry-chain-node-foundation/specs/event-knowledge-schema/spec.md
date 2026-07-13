## MODIFIED Requirements

### Requirement: 实体和扩展 profile
系统 SHALL 使用统一实体节点表表达联盟组织、经济体、政策机构、市场、指数、产业链节点、投研主题、公司、证券、交易工具、指标、商品和人物，并使用对应 profile 表保存类型专属属性；系统不得继续创建 sector 逻辑实体或独立 industry_chain 容器实体。

#### Scenario: 保存统一实体
- **WHEN** 系统保存联盟组织、经济体、公司、证券、产业链节点、投研主题、指标或政策机构等对象
- **THEN** 必须先保存统一实体节点，并通过实体类型和层级字段表达基础归属

#### Scenario: 保存类型专属属性
- **WHEN** 某类实体具备组织代码、证券代码、上市市场、注册经济体、政策领域、计价货币、产业定义或投研主题边界等专属属性
- **THEN** 必须保存到对应 profile 表，而不是把所有字段塞进统一实体节点表

#### Scenario: 事件实体链接使用新类型
- **WHEN** 事件实体链接识别到产业概念或 Tidewise 投研视角
- **THEN** 必须分别链接 `chain_node` 或 `theme`
- **AND** 不得产生新的 `sector`、`industry_chain` 或 `research_theme` 身份
