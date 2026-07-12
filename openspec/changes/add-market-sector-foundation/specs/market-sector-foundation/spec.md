## ADDED Requirements

### Requirement: 市场板块实体分类法
系统 SHALL 定义市场板块基础实体分类法，使事件驱动投研能够引用稳定、可审阅、非推荐性质的板块实体。

#### Scenario: 初始化板块领域分类
- **WHEN** 系统准备将候选板块纳入正式实体基础库
- **THEN** 每个候选必须被归入 `industry_sector`、`theme_sector`、`market_sector`、`style_sector`、`region_sector` 或 `index_proxy_sector` 之一，不得只沿用来源系统的概念、行业或指数分类作为领域分类

#### Scenario: 区分来源分类和领域类型
- **WHEN** 来源系统将候选标记为概念、行业或指数
- **THEN** 系统必须把该来源分类保存为候选来源元数据或 source snapshot，并单独保存观潮家的领域实体类型和分类代码

#### Scenario: 拒绝把指标或商品保存为板块
- **WHEN** 候选对象表示利率、收益率、波动率、汇率、商品价格、加密资产价格或通用宏观指标
- **THEN** 系统不得将其保存为 `sector` 实体，必须归入 benchmark、metric、commodity 或 instrument 的对应边界

### Requirement: 板块稳定标识和命名
系统 SHALL 为市场板块实体提供稳定 `entity_key`、中文主名称、英文 aliases 和可审阅来源字段。

#### Scenario: 生成稳定板块 key
- **WHEN** 板块候选通过 Review 并进入正式 seed
- **THEN** `entity_key` 必须采用稳定可读格式并包含来源系统、领域分类和 slug，不得使用来源排名、当天热度或快照序号作为长期 key

#### Scenario: 保存中文主名称
- **WHEN** 板块实体保存中文市场常用名称
- **THEN** `name` 和 `canonical_name` 必须使用中文主名称，英文名和常见别名必须进入 aliases

#### Scenario: 保存来源快照
- **WHEN** 板块候选来自同花顺 Top 排名或其他行情源列表
- **THEN** 系统可以保存 rank snapshot 和 snapshot date 作为来源快照，但不得将排名作为长期入选依据或稳定标识

### Requirement: MVP 板块候选准入
系统 SHALL 使用候选准入规则从来源池筛选 MVP 板块，而不是机械复制来源 Top 排名。

#### Scenario: 评估候选池规模
- **WHEN** 用户以同花顺概念、行业、指数三个来源分类各取 Top 20 形成约 60 个候选
- **THEN** 系统必须将其视为 MVP 候选池，并要求人工 Review 后才能进入正式主数据

#### Scenario: 应用准入评分维度
- **WHEN** 开发者整理候选 Review 清单
- **THEN** 清单必须覆盖事件可映射性、传导差异、稳定性、市场覆盖、数据可获得性和与其他板块重叠度

#### Scenario: 覆盖主要传导簇
- **WHEN** MVP 板块候选被提交 Review
- **THEN** 候选集合必须覆盖金融地产、资源能源、工业制造、科技成长、消费医药、交通公用和国防安全等宏观事件主要传导簇

#### Scenario: 建议配额
- **WHEN** 候选数量控制在 40 到 60 个
- **THEN** 系统必须优先建议行业骨架 25 到 30 个、事件主题 15 到 20 个、指数代理或参考 benchmark 5 到 10 个，并允许用户 Review 后调整

### Requirement: 板块关系边界
系统 SHALL 只使用经过 Review 的客观关系连接板块与市场、benchmark 或产业链节点，不得把事件推理结论写入基础实体图。

#### Scenario: 市场覆盖板块
- **WHEN** 某个市场官方或行情源稳定覆盖某个板块
- **THEN** 系统可以使用 `market -> covers_sector -> sector` 表达客观覆盖关系，并保存来源名称、来源 URL、核验时间和状态

#### Scenario: 板块参考 benchmark
- **WHEN** 某个板块需要观察一个可核验 benchmark 作为行情或宏观参考
- **THEN** 系统必须通过已审阅关系连接现有 benchmark，不得复制一个同名 sector、index、metric 或 commodity

#### Scenario: 不写推理关系
- **WHEN** 事件抽取或 Agent 分析认为某事件可能影响某板块
- **THEN** 该结论不得写入实体基础 seed 或基础 `entity_edges`，必须由后续事件推理 change 定义事实、置信度、证据和审核边界

### Requirement: 投资建议安全边界
系统 SHALL 将市场板块基础能力限制为决策辅助所需的客观分类和关系，不得表达直接投资建议。

#### Scenario: 拒绝推荐字段
- **WHEN** 板块 seed、profile 或关系文件包含买入、卖出、持有、利好、利空、受益、承压、预测涨跌、目标价、仓位或投资建议字段
- **THEN** validator 必须拒绝该数据进入正式 seed

#### Scenario: 展示板块基础数据
- **WHEN** 后续 API 或前端展示板块基础数据
- **THEN** 展示内容必须表达为市场理解和事件映射辅助，不得表达为具体股票推荐或交易建议
