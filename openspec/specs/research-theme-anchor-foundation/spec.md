# research-theme-anchor-foundation Specification

## Purpose

定义研究主题与研究锚点的 PostgreSQL 事实结构、关联边界及观潮家 Miniapp 只读查询契约。

## Requirements

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

系统 SHALL 提供 `research_themes`，包含 `id UUID` 主键、非空 `analysis_batch_id`、`name`、`one_line_conclusion`、`impact_level`、`transmission_path`、自然语言 `trading_direction`、`transmission_stage`、`next_checkpoint`、`index_impact_summary`、可空 `window_start`/`window_end`、可空 `published_at` 以及平台审计字段。所有必填文本去除首尾空白后必须非空，且窗口两端必须同时为空或同时存在并满足 `window_end >= window_start`。

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

系统 SHALL 通过 PostgreSQL `CHECK` 或等价 domain 校验限制以下精确候选集合：`impact_level` 为 `high|focus|watch`；`transmission_stage` 为 `upstream|midstream|downstream|infrastructure|service`；`relation_role` 为 `driver|beneficiary|constraint|exposure`；`importance` 为 `primary|secondary|contextual`；`evidence_role` 为 `driver|supporting|contradicting|context`；`impact_direction` 为 `positive|negative|mixed|neutral`；`anchor_type` 为 `policy|supply|demand|technology|cost|geopolitics|market_structure`。`trading_direction` MUST 仅作非空自然语言文本校验。

#### Scenario: 拒绝未登记受控值
- **WHEN** 任一受控字段收到候选集合以外的值
- **THEN** 数据库或领域校验必须拒绝该结果

#### Scenario: 保留决策辅助边界
- **WHEN** 结果保存交易指向或影响方向
- **THEN** 系统必须将其作为市场理解和决策辅助文本/描述
- **AND** 不得把 schema 表达为直接投资建议或股票推荐

### Requirement: Miniapp 主题列表 API

系统 SHALL 提供 `GET /api/v1/miniapp/research/themes`。接口必须支持 `window_hours`（默认 24，合法范围 1..168）、`limit`（默认 20，合法范围 1..50）和 opaque `cursor`。接口只返回 `published_at IS NOT NULL` 且位于 `[window_start, as_of]` 的主题，并返回 `window_start`、`window_end`、`as_of`、`theme_count`、这些主题关联的去重 `event_count`、`items` 和 `next_cursor`。

#### Scenario: 返回首页主题卡片
- **WHEN** 客户端请求合法主题列表
- **THEN** 每个 item 至少必须包含 id、name、one_line_conclusion、impact_level、transmission_path、trading_direction、transmission_stage、next_checkpoint、published_at、affected_chain_nodes、related_indices、supporting_event_count、contradicting_event_count 和 has_more_detail
- **AND** `affected_chain_nodes` 必须返回节点 id、名称、relation_role、impact_summary
- **AND** `related_indices` 无映射时必须返回空数组

#### Scenario: 稳定排序和分页
- **WHEN** 客户端使用首次请求或返回的 cursor 获取主题列表
- **THEN** 系统必须按 `impact_level high > focus > watch`、同级 `published_at DESC`、最后 `id ASC` 排序，并以固定 `as_of` 的 keyset cursor 分页
- **AND** 不得依赖 `display_order`，不得重复或漏掉跨页记录

#### Scenario: 主题批次摘要和去重计数
- **WHEN** 查询窗口内主题关联同一 Event 多次或存在 driver/supporting/contradicting 关系
- **THEN** `event_count`、`supporting_event_count` 和 `contradicting_event_count` 必须按 `event_id` 去重，其中 supporting count 合并 `driver` 与 `supporting`，contradicting 独立计数

### Requirement: Miniapp 主题详情 API

系统 SHALL 提供 `GET /api/v1/miniapp/research/themes/{theme_id}`，支持同样的 `window_hours` 默认值和边界。接口必须返回窗口内已发布主题的完整字段、全部关联节点、全部指数和 Event 摘要；Event 摘要只包含 `event_id`、`title`、`summary`、`event_time`、`evidence_role`、`supported_claim`，不得包含 `raw_documents.content_text`。

#### Scenario: 查看主题影响路径
- **WHEN** 客户端请求窗口内存在的已发布 theme UUID
- **THEN** 系统必须返回主题自身、全部节点、全部指数和 Event 摘要，所有无关联集合使用 `[]`

#### Scenario: 隐藏未发布或过期主题
- **WHEN** 客户端请求不存在、未发布或不在窗口内的 theme UUID
- **THEN** 系统必须返回 not found，不得泄露其字段或关联内容

### Requirement: Miniapp 锚点列表与详情 API

系统 SHALL 提供 `GET /api/v1/miniapp/research/anchors` 和 `GET /api/v1/miniapp/research/anchors/{anchor_id}`，参数、发布窗口、cursor、空集合和错误语义与主题 API 一致。锚点列表 item 必须包含 id、anchor_type、name、one_line_conclusion、importance、transmission_path、trading_direction、published_at、related_chain_nodes、related_indices 和 related_event_count；详情必须返回锚点自身、上下游节点、指数和 Event 摘要。

#### Scenario: 查看锚点层
- **WHEN** 客户端请求合法锚点列表或窗口内已发布锚点详情
- **THEN** 系统必须按 importance 业务优先级、published_at DESC、id ASC 稳定排序并返回关联集合
- **AND** 不得增加或依赖 display_order

#### Scenario: 平行读取边界
- **WHEN** 客户端同时读取主题和锚点
- **THEN** 两类 API 必须各自查询对应主表和关系表
- **AND** 禁止 theme-anchor join 或通过关联表建立隐式关系

### Requirement: Miniapp 查询分层与 HTTP 契约

系统 SHALL 在 `backend/internal/apps/miniappapi` 提供 query application service、repository interface/implementation、DTO 和 handler 分层；handler 不得访问数据库，列表不得产生 N+1 查询，PostgreSQL 是唯一 API 事实源，不得查询 Neo4j。API 必须使用 RFC3339 UTC 时间；非法参数/UUID/cursor 返回 HTTP 400 和 `{"error":"..."}`，not found 返回 HTTP 404，repository failure 返回 HTTP 500。

#### Scenario: 校验参数和游标
- **WHEN** 客户端发送超出范围的 window_hours/limit、非法 UUID 或资源类型/窗口不匹配的 cursor
- **THEN** handler 必须返回 HTTP 400，且不得访问数据库

#### Scenario: 处理空数据和数据库错误
- **WHEN** 查询没有节点、指数或 Event，或 repository 返回失败
- **THEN** 空集合必须返回 `[]` 和 0；repository failure 必须返回 HTTP 500，不得将 null 或 raw SQL 错误结构暴露给客户端

#### Scenario: 读取路由边界
- **WHEN** Miniapp 请求四个 research endpoint
- **THEN** 路由必须挂在 `/api/v1/miniapp/research`，通过 service/repository 依赖注入执行
- **AND** 本 change 不提供写 API、Agent 导入 API、scheduler、前端页面或 Neo4j 查询

### Requirement: 删除、唯一性、迁移与事实源边界

系统 SHALL 为每个关联表使用所属主记录 ID 与目标 ID 的复合主键或等价唯一约束，不得包含 `display_order`。删除主题或锚点时必须级联删除其关联表记录；删除关系不得级联删除 `events`、`chain_node_profiles` 或 `index_profiles`。8 张表必须通过 `backend/migrations` 的 forward-only migration 创建；本 change 不要求 Neo4j 结构或投影变化，不执行 seed 或业务数据写入。

#### Scenario: 删除研究结果
- **WHEN** 删除一个主题或锚点
- **THEN** 对应的三类关系记录必须级联删除
- **AND** 被引用的事件、产业链节点和指数主数据必须保留

#### Scenario: 执行获批 migration
- **WHEN** 通过 Apply 和独立 local R2 授权执行 migration
- **THEN** PostgreSQL 必须创建 8 张表、约束、外键和索引
- **AND** 既有 events、chain_node_profiles、index_profiles 的主数据计数保持不变
