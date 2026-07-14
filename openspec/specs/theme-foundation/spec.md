## Purpose

定义 Tidewise 投研主题实体的最小主数据契约、领域边界和未来 theme-node 关联隔离要求。

## Requirements

### Requirement: 投研主题实体
系统 SHALL 使用 `entity_type=theme` 表达 Tidewise 自有投研视角，并复用 `entity_nodes` 的身份、名称、canonical name、aliases、状态与时间字段。

#### Scenario: 保存投研主题
- **WHEN** 后续经独立 Review 的 Tidewise 分析视角进入主数据
- **THEN** 系统必须使用 `entity_type=theme` 和 `theme_profiles`
- **AND** Go 类型必须命名为 `Theme` / `ThemeProfile`

#### Scenario: 禁止兼容命名
- **WHEN** schema、枚举、Go 类型或 seed 输入使用 `research_theme`
- **THEN** validator 或测试必须拒绝该兼容命名

### Requirement: 最小 theme profile
系统 SHALL 使 `theme_profiles` 仅保存 `entity_id UUID` 主外键、非空 `definition TEXT` 与非空 `boundary_note TEXT`，不得重复保存主实体名称或 aliases。

#### Scenario: 校验主题定义和边界
- **WHEN** 系统保存 theme profile
- **THEN** definition 和 boundary_note 都必须为去除首尾空白后非空的文本

#### Scenario: 拒绝分析结果字段
- **WHEN** theme profile 输入包含热度、涨跌、推荐结论、时间周期、市场归属、证券成分或外部分类代码
- **THEN** validator 必须拒绝把这些字段保存到 `theme_profiles`

### Requirement: theme 领域边界
系统 SHALL 将 theme 与 sector、index、产业链容器、粗粒度 chain_node 和证券集合区分，并禁止通过 theme 保存直接投资建议。

#### Scenario: 产业概念不是 theme
- **WHEN** 候选本身表示可观察的产业、技术、材料、设备、工艺、产品或服务类别
- **THEN** 系统必须将其判定为 chain_node 候选，而不是 theme

#### Scenario: 指数或证券篮子不是 theme
- **WHEN** 候选由编制规则、行情成分或证券名单定义
- **THEN** 系统不得把它保存为 theme

#### Scenario: 投研视角不表达推荐
- **WHEN** 后续 API 或分析使用 theme 组织研究内容
- **THEN** 输出必须定位为市场理解和决策辅助，不得表达直接股票推荐或交易建议

### Requirement: 本 change 不初始化 theme 数据
系统 SHALL 在本 change 中只建立 theme schema 与领域边界，不得自行确定具体 theme 实例。

#### Scenario: 执行本 change 的 seed
- **WHEN** Phase A 节点初始化执行
- **THEN** 系统不得写入未经后续独立 Review 的 theme 实例

### Requirement: theme-node 关联隔离
系统 SHALL 将未来 theme 与 chain_node 的 link/scope 关系视为独立契约，不得写入 `chain_node_relations`。

#### Scenario: 提出主题覆盖节点
- **WHEN** 后续 change 需要表达 theme 覆盖或关注哪些 chain_node
- **THEN** 必须单独定义可审阅的 theme-node link/scope 契约
- **AND** 不得把该关联伪装成 `is_subcategory_of`、`is_component_of`、`input_to` 或 `depends_on`
