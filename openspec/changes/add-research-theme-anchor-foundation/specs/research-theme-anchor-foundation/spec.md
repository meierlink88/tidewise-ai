## ADDED Requirements

### Requirement: 平行研究主题与研究锚点结果

系统 SHALL 将研究主题与研究锚点保存为两套彼此平行的 PostgreSQL 研究结果结构；研究结论不是独立实体，而是各自的 `one_line_conclusion` 属性。系统 MUST 使用 `analysis_batch_id` 保存外部 Agent 批次编号，且不得创建 `research_analysis_runs`、`research_conclusions` 或任何 `research_conclusion_*` 表。

#### Scenario: 保存研究结果
- **WHEN** 后续获批的 Agent 回写一个研究主题或研究锚点
- **THEN** 系统必须把结果保存到对应主表，并保存名称、一句话结论、传导路径和自然语言交易指向
- **AND** 主题与锚点之间不得存在外键或关联表

#### Scenario: 拒绝第三类研究结论实体
- **WHEN** schema 或回写模型尝试创建 research conclusion 主表、运行批次表或通用多态关系表
- **THEN** 迁移/契约验证必须拒绝该结构

### Requirement: 研究主题字段与边界

系统 SHALL 提供 `research_themes`，包含 `id UUID` 主键、非空 `analysis_batch_id`、`name`、`one_line_conclusion`、`impact_level`、`transmission_path`、自然语言 `trading_direction`、`transmission_stage`、`next_checkpoint`、`index_impact_summary`、可空 `window_start`/`window_end`、可空 `published_at` 以及平台审计字段。所有必填文本去除首尾空白后必须非空，且 `window_end` 非空时不得早于 `window_start`；只允许两者同时为空或同时存在。

#### Scenario: 校验主题结果
- **WHEN** 保存主题且必填文本为空白，或只提供一个窗口端点，或结束时间早于开始时间
- **THEN** 系统必须拒绝保存

#### Scenario: 保存主题附加信息
- **WHEN** 保存合法主题结果
- **THEN** 系统必须保存 impact level、transmission stage、next checkpoint、指数影响摘要和分析窗口
- **AND** `trading_direction` 必须保持可表达复杂语义的 TEXT，而不是正负枚举

### Requirement: 研究锚点字段与边界

系统 SHALL 提供 `research_anchors`，包含 `id UUID` 主键、非空 `analysis_batch_id`、`anchor_type`、`name`、`one_line_conclusion`、`importance`、`transmission_path`、自然语言 `trading_direction`、可空 `published_at` 以及平台审计字段。所有必填文本去除首尾空白后必须非空。

#### Scenario: 保存锚点结果
- **WHEN** 保存合法研究锚点
- **THEN** 系统必须保存 anchor type、importance、传导路径和自然语言交易指向
- **AND** 锚点不得继承主题的窗口或指数摘要字段

### Requirement: 主题与锚点的三类独立关联

系统 SHALL 为主题和锚点分别提供 chain node、index、event 三类关联表，共 6 张表。关联表 MUST 使用所属主表与目标主数据的明确 UUID 外键；不得使用通用多态关系。

#### Scenario: 关联既有主数据
- **WHEN** 研究结果关联产业链节点、指数或事件
- **THEN** 系统必须分别写入对应的 theme/anchor relation 表，并引用 `chain_node_profiles.entity_id`、`index_profiles.entity_id` 或 `events.id`
- **AND** 不得复制或初始化被引用的主数据

#### Scenario: 保存关系语义
- **WHEN** 写入 chain node 关系
- **THEN** 系统必须保存 relation role 与 impact/relation summary
- **WHEN** 写入 index 关系
- **THEN** 系统必须保存 impact direction 与 impact summary
- **WHEN** 写入 event 关系
- **THEN** 系统必须保存 evidence role 与 supported claim

### Requirement: 受控值与证据语义

系统 SHALL 通过 PostgreSQL `CHECK` 或等价 domain 校验限制以下精确候选集合：`impact_level` 为 `low|medium|high|critical`；`transmission_stage` 为 `upstream|midstream|downstream|infrastructure|service`；`relation_role` 为 `driver|beneficiary|constraint|exposure`；`importance` 为 `primary|secondary|contextual`；`evidence_role` 为 `supports|contradicts|context`；`impact_direction` 为 `positive|negative|mixed|neutral`；`anchor_type` 为 `policy|supply|demand|technology|cost|geopolitics|market_structure`。`trading_direction` MUST 仅作非空自然语言文本校验。

#### Scenario: 拒绝未登记受控值
- **WHEN** 任一受控字段收到候选集合以外的值
- **THEN** 数据库或领域校验必须拒绝该结果

#### Scenario: 保留决策辅助边界
- **WHEN** 结果保存交易指向或影响方向
- **THEN** 系统必须将其作为市场理解和决策辅助文本/描述
- **AND** 不得把 schema 表达为直接投资建议或股票推荐

### Requirement: 删除、唯一性与查询边界

系统 SHALL 为每个关联表使用所属主记录 ID 与目标 ID 的复合主键或等价唯一约束，不得包含 `display_order`。删除主题或锚点时必须级联删除其关联表记录；删除关系不得级联删除 `events`、`chain_node_profiles` 或 `index_profiles`。主表和关系表 MUST 提供按批次、发布时间和目标主数据查询所需的最小索引。

#### Scenario: 删除研究结果
- **WHEN** 删除一个主题或锚点
- **THEN** 对应的三类关系记录必须级联删除
- **AND** 被引用的事件、产业链节点和指数主数据必须保留

#### Scenario: 防止重复关系和排序字段
- **WHEN** 同一研究结果尝试重复关联同一事件、指数或产业链节点
- **THEN** 数据库必须通过复合主键或唯一约束拒绝重复关系
- **AND** 8 张表均不得出现 display_order

### Requirement: PostgreSQL 事实源与迁移边界

系统 SHALL 将 8 张表作为 PostgreSQL 事实源，通过 `backend/migrations` 的 forward-only migration 创建；本 change 不要求 Neo4j 结构或投影变化，不执行 seed，不写入业务数据。任何 schema 回滚 MUST 采用新的审阅 forward-fix 或经批准的恢复方案，不得 destructive drop。

#### Scenario: 执行获批 migration
- **WHEN** 通过 Apply 和独立 local R2 授权执行 migration
- **THEN** PostgreSQL 必须创建 8 张表、约束、外键和索引
- **AND** 既有 events、chain_node_profiles、index_profiles 的主数据计数保持不变

#### Scenario: 排除图谱与业务功能
- **WHEN** 本 change 完成 Proposal 或后续 Apply
- **THEN** 不得修改 Neo4j、Event 提取 Agent、研究报告 Agent、scheduler、API、小程序页面或查询接口
