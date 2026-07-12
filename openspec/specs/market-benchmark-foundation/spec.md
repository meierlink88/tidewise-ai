## Purpose

定义具体市场 benchmark、时序观测值和客观关系的当前系统事实，并明确其与 index、metric、commodity、instrument 及 Neo4j 投影的边界。

## Requirements

### Requirement: Benchmark 实体定义
系统 SHALL 使用独立 benchmark 实体表达具体可观测的收益率、价格或参考利率，并与正式指数、通用指标、商品和交易工具类别区分。

#### Scenario: 保存 benchmark 定义
- **WHEN** 系统初始化具体政府债券收益率、商品价格、现货价格或数字资产参考利率
- **THEN** 必须创建 `entity_type=benchmark` 的统一实体节点和 benchmark profile，不得将其保存为 index 或 metric

#### Scenario: 官方代码未知
- **WHEN** 权威来源没有提供可核验的 official series code
- **THEN** benchmark profile 必须将该字段保存为空值，不得生成内部占位符冒充官方代码

#### Scenario: 区分数字资产具体标的
- **WHEN** BTC 与 ETH benchmark 都引用数字资产交易工具类别
- **THEN** 系统必须通过 benchmark profile 的 underlying symbol 区分具体标的，不得把 BTC 和 ETH 创建为通用 instrument 类别实体

### Requirement: Benchmark 观测值
系统 SHALL 在 PostgreSQL 保存 benchmark 的带时间戳观测值、来源和质量状态，并保证同一来源重复写入幂等。

#### Scenario: 幂等写入观测值
- **WHEN** 相同 benchmark、observed time 和 source name 的观测值被重复写入
- **THEN** repository 必须更新或复用现有记录，不得创建重复观测

#### Scenario: 保存质量状态
- **WHEN** 系统保存 benchmark observation
- **THEN** quality status 必须属于 raw、validated、suspect 或 rejected，并保存值、单位和来源 URL

#### Scenario: 不投影逐时点观测
- **WHEN** 系统重建 Neo4j 实体图
- **THEN** benchmark observations 不得被创建为 Neo4j 节点或关系

### Requirement: Benchmark 客观关系
系统 SHALL 使用明确方向和端点类型的客观关系连接市场、benchmark、metric、commodity 和 instrument。

#### Scenario: 市场观测 benchmark
- **WHEN** 某个市场以具体 benchmark 作为分析观测对象
- **THEN** 系统必须使用 `market -> observes_benchmark -> benchmark` 并保存来源与核验时间

#### Scenario: Benchmark 测量指标
- **WHEN** benchmark 对应一个通用测量维度
- **THEN** 系统必须使用 `benchmark -> measures -> metric`，不得复制一个同名 metric

#### Scenario: Benchmark 引用标的
- **WHEN** benchmark 对应商品或交易工具类别
- **THEN** 系统必须使用 `benchmark -> references -> commodity/instrument`

#### Scenario: 拒绝推理字段
- **WHEN** benchmark 关系包含涨跌方向、影响强度、受益承压、预测或投资建议
- **THEN** validator 必须拒绝该关系

### Requirement: 首批 Benchmark 审阅迁移
系统 SHALL 在人工 review 后初始化首批 10 个 benchmark，并保持 PostgreSQL 与 Neo4j 定义关系一致。

#### Scenario: 用户确认前保持空 seed
- **WHEN** 首批 benchmark 定义和关系尚未获得用户明确确认
- **THEN** 系统不得将候选数据写入正式 seed、PostgreSQL 或 Neo4j

#### Scenario: 写入首批 benchmark
- **WHEN** 用户确认五个 10 年期政府债券收益率、Brent、WTI、黄金、BTC 和 ETH benchmark 清单
- **THEN** entity seed 必须幂等写入 10 个 benchmark profile 和已确认关系，并输出按实体与关系类型统计

#### Scenario: 图谱重建验收
- **WHEN** 首批数据完成 PG 验收并重建 Neo4j
- **THEN** Neo4j benchmark 节点和三类关系必须与 PG active 事实一致，且不得恢复此前误建的 index 节点

### Requirement: 板块与 benchmark 职责边界
系统 SHALL 在市场板块候选审阅中严格区分 semantic sector 和 market benchmark，同时允许同一来源对象在审阅后形成 sector 与 benchmark 的关联。

#### Scenario: 指数板块候选
- **WHEN** 来源指数板块表示行业或主题暴露，例如半导体材料设备、卫星产业或类似对象
- **THEN** 系统必须允许其作为 sector 候选进入 Review，并单独判断是否需要关联 benchmark 作为可观测行情标尺

#### Scenario: index sector 不降级
- **WHEN** 候选的 `source_taxonomy_type` 为 `index_sector`
- **THEN** 系统不得将整类候选改为 benchmark-only，必须先按 semantic sector 审阅其事件暴露职责

#### Scenario: 宽基指数对象
- **WHEN** 候选对象表示宽基市场表现，例如上证指数、沪深300、标普500、纳斯达克100或类似对象
- **THEN** 系统必须优先将其判别为正式 index 或 benchmark，不得仅因来自指数分类而复制为普通 sector

#### Scenario: 利率收益率候选
- **WHEN** 候选对象表示政府债券收益率、政策利率、参考利率或信用利差
- **THEN** 系统必须将其纳入 benchmark 或 metric 边界，不得保存为 sector

#### Scenario: 商品价格候选
- **WHEN** 候选对象表示原油、黄金、有色金属、农产品或其他商品的价格序列
- **THEN** 系统必须复用 commodity 和 benchmark 边界，不得复制为 sector

#### Scenario: 行业主题指数候选
- **WHEN** 某个指数确实代表一个行业或主题板块表现
- **THEN** 系统可以同时保留 sector 的事件暴露职责和 benchmark 的行情标尺职责，但必须保存来源说明、代码边界和人工 Review 结果

#### Scenario: 概念板块行情代码
- **WHEN** 同花顺概念板块具备 885、886 或类似板块指数代码和行情序列
- **THEN** 系统不得因此将该概念板块降级为 benchmark-only，必须优先保留其 semantic sector 身份，并按需关联 benchmark

### Requirement: 板块参考 benchmark 关系安全
系统 SHALL 只把板块与 benchmark 的客观参考关系用于市场理解，不得表达投资判断。

#### Scenario: 保存参考关系
- **WHEN** 用户确认某个板块需要观察某个 benchmark
- **THEN** 系统必须复用现有 benchmark 实体并通过 `sector -> tracked_by_benchmark -> benchmark` 保存已审阅关系，不得创建同名 benchmark 副本，也不得修改既有 `market -> observes_benchmark -> benchmark` 语义

#### Scenario: 一对一或多对一关联
- **WHEN** 一个或多个 sector 共享同一可观测行情标尺
- **THEN** 系统必须允许 sector 与 benchmark 形成一对一或多对一关联，并保持两者实体职责独立

#### Scenario: 拒绝预测语义
- **WHEN** 板块与 benchmark 关系包含利好利空、预测涨跌、传导强度或投资建议
- **THEN** validator 必须拒绝该关系
