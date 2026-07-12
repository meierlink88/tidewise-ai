## ADDED Requirements

### Requirement: 市场板块 seed 审阅准入
系统 SHALL 将市场板块 seed 从来源快照候选转换为经过 Review 的实体基础数据。

#### Scenario: 同花顺 Top 候选进入 Review
- **WHEN** 同花顺概念板块、行业板块、指数板块三个来源分类各 Top 20 被整理为候选池
- **THEN** 系统必须将其保存或呈现为候选 Review 清单，而不得直接全部写入正式主数据

#### Scenario: 候选评分不固化为实体身份
- **WHEN** 候选 Review 清单包含评分分项、总分、核心、扩展或观察分层
- **THEN** 这些字段不得成为 stable key、领域分类或不可变实体身份；如需要持久化，必须进入候选 Review、source snapshot 或推理调度边界

#### Scenario: 行业作为稳定骨架
- **WHEN** 候选中包含来源行业板块
- **THEN** 系统必须优先评估其作为 `industry_sector` 稳定骨架的适配性，并覆盖主要宏观事件传导簇

#### Scenario: 概念作为主题映射层
- **WHEN** 候选中包含来源概念板块
- **THEN** 系统必须只接受可解释、非短期炒作且有稳定定义的主题进入 `theme_sector`

#### Scenario: 指数板块允许作为 sector
- **WHEN** 候选中包含来源指数板块分类
- **THEN** 系统必须允许其以 `source_taxonomy_type=index_sector` 进入 sector Review，并按 semantic sector 分类归入行业、主题、市场、风格或区域板块，同时单独判别是否需要关联 benchmark、正式 index 或来源代码

### Requirement: 市场板块 profile 校验
系统 SHALL 在写入数据库前校验市场板块 profile 的领域分类、市场范围和 Review 状态，并通过结构化 source mapping 校验来源系统、来源分类和来源代码。

#### Scenario: 校验领域分类
- **WHEN** seed loader 读取 `sector` profile
- **THEN** profile 必须提供可校验的 semantic classification 字段，并且该字段必须属于已批准的市场板块分类法，且不得使用 `index_sector`

#### Scenario: 校验来源分类
- **WHEN** seed loader 读取来自同花顺或其他来源系统的 sector source mapping
- **THEN** mapping 必须保存可审阅的 `source_taxonomy_type` 或等价字段，区分 `concept`、`industry` 和 `index_sector`

#### Scenario: 校验主要市场
- **WHEN** 板块 profile 声明主要市场范围
- **THEN** 引用的市场实体必须存在且 `entity_type=market`

#### Scenario: 校验主要经济体
- **WHEN** 板块 profile 声明主要经济体范围
- **THEN** 引用的经济体实体必须存在且 `entity_type=economy`

#### Scenario: 保留旧快照字段
- **WHEN** 现有 `rank_snapshot` 和 `snapshot_date` 字段仍用于来源审阅
- **THEN** 系统必须将其作为来源快照字段保留，不得将其作为稳定排序、推荐依据或唯一入选依据

#### Scenario: 支持多个来源映射
- **WHEN** 完全同义且范围一致的候选被合并为一个 canonical sector
- **THEN** seed 必须通过 `sector_source_mappings` 或等价结构化 manifest 保留多个来源分类、来源代码、来源名称、来源 URL、快照排名和映射状态，使审计可以追溯原始候选

#### Scenario: source mapping 唯一性
- **WHEN** 多个来源映射被写入 PostgreSQL
- **THEN** 同一 `source_system`、`source_taxonomy_type` 和非空 `source_sector_code` 只能指向一个 canonical sector；无代码来源必须通过来源名称和快照日期防止同一候选重复导入

### Requirement: 市场板块关系 seed 策略
系统 SHALL 只把已经 Review 的板块客观关系写入正式关系 seed。

#### Scenario: 写入市场覆盖板块关系
- **WHEN** `covers_sector` 关系获得人工 Review
- **THEN** seed 必须只允许 `market -> sector` 方向，并保存来源名称、来源 URL、核验时间和状态

#### Scenario: 拒绝错误方向
- **WHEN** 关系文件包含 `sector -> market` 的 `covers_sector` 关系
- **THEN** validator 必须拒绝该关系并返回明确错误

#### Scenario: 不写未审阅 benchmark 关系
- **WHEN** 板块和 benchmark 的关联尚未逐项 Review
- **THEN** 系统不得把候选关联写入正式 seed、PostgreSQL 或 Neo4j

#### Scenario: 写入 benchmark 跟踪关系
- **WHEN** `tracked_by_benchmark` 关系获得人工 Review
- **THEN** seed 必须只允许 `sector -> benchmark` 方向，并保存来源名称、来源 URL、核验时间和状态，且不得使用 `observes_benchmark` 表达该方向
