## MODIFIED Requirements

### Requirement: 事件知识 PostgreSQL schema
系统 SHALL 在 PostgreSQL 中保存 MVP 阶段的采集源、原始文档、事件事实、事件证据、事件 Tag 定义和事件 Tag 关联，并将这些结构作为后续事件抽取、审核和 API 查询的事实基础；事件事实 SHALL 增加 `fact_payload JSONB` 以承载可验证的原子事实载荷，且不得承载预测或投资建议。现有 `event_entity_links` 保持原样，不属于本 change。

#### Scenario: 保存原子事件事实载荷
- **WHEN** 确定性执行器写入通过证据校验的 Event package
- **THEN** `events` 必须保留既有标题、摘要、时间、状态和 `dedupe_key` 字段，并允许以 `fact_payload` 保存 JSON object 形式的原子事实；空 object 对既有记录合法，缺失字段不得破坏旧记录读取

#### Scenario: 拒绝推理或投资建议载荷
- **WHEN** candidate payload 包含买入卖出、涨跌预测、利好利空、传导强度、事件评分或直接投资建议
- **THEN** 系统不得把这些内容写入 `fact_payload` 或事件关系事实，并记录拒绝原因

#### Scenario: 增量增加事件 schema
- **WHEN** 编号为 `000019` 的未来 migration 在已有 raw document、事件、证据或 Tag 数据的 PostgreSQL 上执行
- **THEN** migration 必须只增量增加已 Review 批准的列、约束或必要索引，不得清空、删除或重建既有业务数据

### Requirement: 原始文档和事件事实分离
系统 SHALL 将外部采集得到的原始材料保存为 `RAW_DOCUMENT`，并通过 `EVENT_SOURCE` 将原始文档作为事件事实证据；本 change 不建立或修改实体关联。

#### Scenario: 关联可追溯证据
- **WHEN** Event package 由一篇或多篇 raw document 支持
- **THEN** 系统必须通过 `event_sources` 关联事件和 raw document，并保留证据摘录、证据哈希及经 Review 批准的证据关系/支持字段（如适用）

#### Scenario: 保持实体关联范围外
- **WHEN** raw document 或 Event candidate 包含实体名称
- **THEN** 本 change 不得执行实体匹配、生成实体候选、写入实体关联或修改 `event_entity_links`；相关能力必须由未来独立 change 定义

### Requirement: 受控 Tag 分类和审计
系统 SHALL 保留 `event_tag_maps`，并在 Review 批准后保存最小 Tag 归因信息；Tidewise DB 是 Tag 主数据唯一权威，YAML 只能作为策略输入。

#### Scenario: 使用数据库 Tag 主数据
- **WHEN** Event package 携带 Tag candidate
- **THEN** 系统必须按 Tidewise DB 中的 `event_tag_defs` 进行映射；未注册 Tag 不得作为正式 Tag 事实写入，YAML 不得创建或替代 Tag 主数据

#### Scenario: 审计 Tag 分配
- **WHEN** Tag 通过确定性校验写入
- **THEN** 系统应在经 Review 批准的最小字段中保存 Tag 的 confidence/assignment reason；旧数据和既有唯一约束必须保持可读兼容

#### Scenario: 延后非必要事件字段
- **WHEN** Review 讨论 `event_type_code`、`action_code`、`lifecycle_status`、`reference_period_start/end`、`last_seen_at` 或 `event_time_precision`
- **THEN** 除非形成独立字段语义、约束、兼容和回滚决策，否则前五项必须 defer；`event_time_precision` 只能作为待用户 Review 的候选，不得在本 Proposal 中预批准
