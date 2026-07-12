## ADDED Requirements

### Requirement: 市场板块实体分类法
系统 SHALL 定义市场板块基础实体分类法，使事件驱动投研能够引用稳定、可审阅、非推荐性质的板块实体。

#### Scenario: 初始化板块领域分类
- **WHEN** 系统准备将候选板块纳入正式实体基础库
- **THEN** 每个候选必须被归入 `industry_sector`、`theme_sector`、`market_sector`、`style_sector`、`region_sector` 或 `index_sector` 之一，不得只沿用来源系统的概念、行业或指数板块分类作为领域分类

#### Scenario: 区分三层概念
- **WHEN** 来源系统将候选标记为概念板块、行业板块或指数板块
- **THEN** 系统必须分别保存 external/source taxonomy、semantic sector 分类和 market benchmark 关联判断，不得用任意一层覆盖另外两层

#### Scenario: 指数板块仍为 sector 候选
- **WHEN** 同花顺指数板块表达行业或主题暴露，例如半导体材料设备或卫星产业
- **THEN** 系统必须允许其作为 `sector` 候选进入 Review，不得因来源分类包含“指数”而自动排除为 benchmark-only

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

#### Scenario: 不固化 Top 排名
- **WHEN** 三类 Top 20 候选被转为正式 sector
- **THEN** Top 排名不得成为 `entity_key`、领域分类、实体身份或永久主数据属性

### Requirement: MVP 板块候选准入
系统 SHALL 使用候选准入规则从来源池筛选 MVP 板块，而不是机械复制来源 Top 排名。

#### Scenario: 评估候选池规模
- **WHEN** 用户以同花顺概念板块、行业板块、指数板块三个来源分类各取 Top 20 形成约 60 个候选
- **THEN** 系统必须将其视为 MVP 候选池，并要求人工 Review 后才能进入正式主数据

#### Scenario: 使用已确认评分权重
- **WHEN** 开发者评估 MVP sector 候选
- **THEN** 必须按事件可解释性 25、传导独立性 20、行情敏感度 15、数据完整性 15、长期稳定性 15、市场代表性 10 计算候选评估结果，并原则上只让 70 分以上候选进入 MVP

#### Scenario: 应用准入评分维度
- **WHEN** 开发者整理候选 Review 清单
- **THEN** 清单必须覆盖事件可映射性、传导差异、稳定性、市场覆盖、数据可获得性和与其他板块重叠度

#### Scenario: 覆盖主要传导簇
- **WHEN** MVP 板块候选被提交 Review
- **THEN** 候选集合必须覆盖金融地产、能源电力、有色化工材料、工业基建、半导体电子、AI 软件通信、汽车新能源、医药生科、消费农业、交通公用、国防航天卫星和政策主题等主要传导簇

#### Scenario: 控制正式 sector 规模
- **WHEN** 候选评估和去重完成
- **THEN** MVP 正式 sector 数量应约为 50 到 60 个，并保留未入选候选的 Review 原因

#### Scenario: 三类候选都保留 Review 权利
- **WHEN** 候选池按概念板块、行业板块、指数板块各 Top 20 组织
- **THEN** 系统不得预设削减指数板块配额，必须优先给出去重、交叉映射和 benchmark 关联规则，并允许用户 Review 后确定最终入选清单

#### Scenario: 运行分层不等于实体身份
- **WHEN** 系统将 MVP sector 分为核心约 30、扩展约 20 和观察约 10
- **THEN** 该分层必须被表达为推理调度、Review 或候选管理层信息，不得写入 stable key、语义分类或不可变实体身份

### Requirement: 板块关系边界
系统 SHALL 只使用经过 Review 的客观关系连接板块与市场、benchmark 或产业链节点，不得把事件推理结论写入基础实体图。

#### Scenario: 市场覆盖板块
- **WHEN** 某个市场官方或行情源稳定覆盖某个板块
- **THEN** 系统可以使用 `market -> covers_sector -> sector` 表达客观覆盖关系，并保存来源名称、来源 URL、核验时间和状态

#### Scenario: 板块参考 benchmark
- **WHEN** 某个板块需要观察一个可核验 benchmark 作为行情或宏观参考
- **THEN** 系统必须通过已审阅关系连接现有或新增 benchmark，使 sector 表达可被事件影响的产业/主题暴露，benchmark 表达可观测行情标尺

#### Scenario: 不写推理关系
- **WHEN** 事件抽取或 Agent 分析认为某事件可能影响某板块
- **THEN** 该结论不得写入实体基础 seed 或基础 `entity_edges`，必须由后续事件推理 change 定义事实、置信度、证据和审核边界

#### Scenario: 完全同义候选合并
- **WHEN** 多个候选完全同义且覆盖范围一致
- **THEN** 系统必须保留一个 canonical sector，并保存多个来源映射、别名或 cross-reference

#### Scenario: 部分重叠候选保留关系
- **WHEN** 多个候选粒度不同或只存在部分重叠
- **THEN** 系统必须分别保留对应 sector，并通过后续已审阅上下位或交叉关系表达，不得强行合并

### Requirement: 投资建议安全边界
系统 SHALL 将市场板块基础能力限制为决策辅助所需的客观分类和关系，不得表达直接投资建议。

#### Scenario: 拒绝推荐字段
- **WHEN** 板块 seed、profile 或关系文件包含买入、卖出、持有、利好、利空、受益、承压、预测涨跌、目标价、仓位或投资建议字段
- **THEN** validator 必须拒绝该数据进入正式 seed

#### Scenario: 展示板块基础数据
- **WHEN** 后续 API 或前端展示板块基础数据
- **THEN** 展示内容必须表达为市场理解和事件映射辅助，不得表达为具体股票推荐或交易建议
