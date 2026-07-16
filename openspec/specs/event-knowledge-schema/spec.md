## Purpose

定义观潮家 MVP 阶段事件知识 PostgreSQL schema 的当前系统事实，覆盖实体、关系、采集源、原始文档、事件事实、证据、标签、事件实体关联和 schema 迁移边界。

## Requirements

### Requirement: 事件知识 PostgreSQL schema
系统 SHALL 在 PostgreSQL 中保存 MVP 阶段的实体、可审计实体关系、采集源、原始文档、事件事实、事件证据、事件标签和事件实体关联，并将这些结构作为后续事件抽取、审核、图谱投影和 API 查询的事实基础。

#### Scenario: 创建事件知识 schema
- **WHEN** 本 change 的数据库迁移被执行
- **THEN** PostgreSQL 必须具备实体节点、实体关系、联盟组织 profile、各类既有实体 profile、采集源目录、原始文档、事件事实、事件来源证据、事件标签定义、事件标签关联和事件实体关联表

#### Scenario: 保存实体关系来源元数据
- **WHEN** 系统保存经过审阅的实体关系
- **THEN** `entity_edges` 必须保存来源名称、来源 URL 和核验时间，并继续保存证据说明、状态和更新时间

#### Scenario: 增量升级关系 schema
- **WHEN** 已有 PostgreSQL 应用实体关系来源字段 migration
- **THEN** migration 必须以增量方式增加字段，不得删除实体节点、实体 profile、采集数据、事件数据或其他现有业务数据

#### Scenario: 支持图谱投影
- **WHEN** 后续 change 需要构建实体图或事件图
- **THEN** 系统必须能够从 PostgreSQL 中的实体、事件、标签和关系表投影数据，而不是要求采集层直接写入图数据库

### Requirement: 真实数据库 schema 创建验证
系统 SHALL 能够把 repo 内事件知识 migration 应用到真实 PostgreSQL，并通过可重复验证确认关键表、索引和迁移版本存在。

#### Scenario: 创建关键表
- **WHEN** 开发者对空的 local PostgreSQL 执行事件知识 migration
- **THEN** 数据库必须创建 `source_catalogs`、`raw_documents`、`events`、`event_sources`、`entity_nodes` 和实体关系相关表

#### Scenario: 记录迁移版本
- **WHEN** migration 成功执行
- **THEN** 数据库必须记录已应用 migration 版本，使后续重复执行不会重复创建表或清空数据

#### Scenario: 保留已有数据增量更新
- **WHEN** 数据库已经存在采集源或原始文档数据并再次执行迁移检查
- **THEN** 系统不得清空、重建或丢弃已有业务数据

### Requirement: 并发迁移保护
系统 SHALL 在真实 PostgreSQL migration 执行时使用 advisory lock、迁移工具锁或等价机制，避免多个服务实例同时执行 DDL。

#### Scenario: 多实例同时启动
- **WHEN** 多个后端进程同时发现 pending migration
- **THEN** 只有一个进程可以执行 DDL，其余进程必须等待、跳过或失败并给出明确错误

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

### Requirement: 联盟组织实体
系统 SHALL 支持把 `OPEC+`、`G7`、`WTO` 等跨经济体联盟组织、国际组织或多边协调机制保存为独立实体类型，并与国家政策机构、经济体和市场实体区分。

#### Scenario: 保存联盟组织
- **WHEN** 系统初始化或保存 approved 联盟组织
- **THEN** 必须在 `entity_nodes` 中保存 `entity_type=alliance_org`、`layer_code=alliance`、name、canonical name、由缩写派生并按 NFKC + casefold 去重的 aliases 和 status，并在 `alliance_org_profiles` 中只保存 abbreviation、leadership summary 与 influence scope summary

#### Scenario: 区分联盟组织和政策机构
- **WHEN** 实体表示跨多个经济体的国际组织、联盟、论坛或规则协调机制
- **THEN** 系统必须使用联盟组织实体表达，而不是把该实体保存为单一经济体下的政策机构

#### Scenario: 不从大类或子类生成实体标签
- **WHEN** 联盟 Excel 包含大类或子类
- **THEN** 系统不得把这些列写入 profile、创建实体标签或复用事件标签；Neo4j 仍使用单一 `Entity` label 且不在本 change 执行任何投影操作

### Requirement: Economy Schema 保持现状
系统 SHALL 保持 `economy_profiles` 的现有 `country_code/currency_code/region` 结构，不因本次 local business-data rebuild 引入全局 identity/schema 改造。

#### Scenario: 不新增 Economy Identity 列
- **WHEN** migration 为 alliance profile 做最小演进
- **THEN** 不得新增 `economy_profiles.identity_kind`、区域/货币 policy 列或平行 economy profile 表

#### Scenario: 不新增 Entity Key 全局唯一
- **WHEN** importer 为 frozen 79 economy 实现幂等
- **THEN** 必须优先使用 approved stable keys 与现有 repository preflight，不得为本批新增全局 `entity_nodes.entity_key` 唯一索引；若无法安全实现，必须回到 R0 Review

### Requirement: 原始文档和事件事实分离
系统 SHALL 将外部采集得到的原始材料保存为 `RAW_DOCUMENT`，并通过 `EVENT_SOURCE` 将原始文档作为事件事实证据，不得让原始文档直接关联实体。

#### Scenario: 采集原始文档
- **WHEN** 采集层从 RSS、HTTP API、RSSHub、网页或本地文件获得外部材料
- **THEN** 系统必须保存原始文档记录，并记录来源、外部 ID、标题、正文、原始对象 URI、发布时间、采集时间、内容哈希和处理状态

#### Scenario: 关联事件证据
- **WHEN** 后续抽取流程从原始文档生成或更新事件事实
- **THEN** 系统必须通过事件来源证据表关联事件和原始文档，并保存来源等级、证据摘录和证据哈希

#### Scenario: 避免原文直连实体
- **WHEN** 原始文档中出现公司、板块、政策机构或指标
- **THEN** 原始文档不得直接写入实体关联表，实体关联必须由后续事件抽取、结构化校验或审核流程生成

### Requirement: 采集阶段不保存推理结论
系统 SHALL 在本阶段数据库 schema 中排除事件评分、主题贡献、传导强度、预测结论、复盘验证和用户预警任务等推理或产品化结论。

#### Scenario: 写入采集结果
- **WHEN** 采集层写入原始文档或采集状态
- **THEN** 数据库不得要求采集层填写利好利空、受益承压、影响强度、传导路径、预测结论或投资建议

#### Scenario: 后续需要分析结论
- **WHEN** 后续 change 需要保存 Agent 分析、主题贡献、传导强度或复盘验证
- **THEN** 必须通过独立 OpenSpec change 定义对应 schema 和安全边界

### Requirement: schema 迁移和回滚
系统 SHALL 为事件知识 schema 提供可审阅、版本化、长期保留的数据库迁移来源，并保证 local、uat、prod 使用同一套迁移定义。

#### Scenario: 执行迁移
- **WHEN** 开发者或部署流程初始化 PostgreSQL
- **THEN** 必须能够从 repo 内的迁移来源创建事件知识 schema

#### Scenario: 保留版本化 DDL 文件
- **WHEN** 本 change 或后续 change 修改数据库表、字段、索引、约束或枚举检查
- **THEN** 必须在 repo 内追加或更新对应阶段允许修改的版本化 SQL migration 文件，并让该文件成为 schema 变化的审阅依据

#### Scenario: 增量演进已有数据
- **WHEN** 数据库已经存在业务数据且需要执行 schema 更新
- **THEN** migration 必须以增量方式演进结构，不得通过清空表、删除全库、重建全库或丢弃既有业务数据来完成升级

#### Scenario: 遵循字段映射
- **WHEN** 本 change 实现 PostgreSQL migration
- **THEN** migration 必须覆盖 `design.md` 中 `Schema field mapping` 列出的 ER 核心字段、主键和外键关系

#### Scenario: 审阅回滚策略
- **WHEN** schema 变更需要回滚或降级
- **THEN** 本 change 必须提供 down migration、兼容迁移说明或明确的回滚策略，且回滚策略不得依赖清空业务数据

### Requirement: 事件知识图谱投影来源
系统 SHALL 将 PostgreSQL 中的实体、实体关系、事件、证据、标签和事件实体关联作为 Neo4j 图谱投影的事实来源。

#### Scenario: 投影实体图
- **WHEN** 系统构建 Neo4j 实体图
- **THEN** 必须从 `entity_nodes`、实体 profile 和 `entity_edges` 读取投影来源，而不是由采集层、前端或 Neo4j 手工数据直接生成

#### Scenario: 后续投影事件图
- **WHEN** 后续 change 构建事件图谱
- **THEN** 必须从 `events`、`event_sources`、事件标签和 `event_entity_links` 读取投影来源，并继续保留 PostgreSQL 事件事实和证据链

#### Scenario: 保留原始文档和事件事实分离
- **WHEN** 原始文档中的文本被用于后续事件图谱构建
- **THEN** 原始文档仍然不得直接连接实体图，必须先经过事件抽取、结构化校验和事件实体关联事实表

### Requirement: 实体关系来源元数据
系统 SHALL 在 `entity_edges` 中保存最小来源与核验元数据，使实体关系事实能够与来源信息一同在 local、uat 和 prod 环境中增量演进。

#### Scenario: 对已有关系数据库执行增量 migration
- **WHEN** 已存在 `entity_edges` 业务数据的数据库执行版本化 migration
- **THEN** 系统必须新增 `source_name`、`source_url` 和 `verified_at` 字段
- **AND** 不得清空、删除或重建 `entity_edges` 或其他业务表

#### Scenario: 重复执行 migration
- **WHEN** 已经应用实体关系来源元数据 migration 的数据库再次执行 migration 检查
- **THEN** 系统必须识别已应用版本并成功完成检查
- **AND** 不得重复创建字段或修改既有关系记录

#### Scenario: 迁移文件自动化验证
- **WHEN** 开发者验证后端 migration
- **THEN** 自动化测试必须确认 migration 包含三个来源字段、可重复执行的追加语义和非破坏性 SQL 约束

### Requirement: Benchmark PostgreSQL schema
系统 SHALL 通过增量 migration 增加 benchmark profile 和 observation schema，并保留全部既有实体、关系、采集和事件数据。

#### Scenario: 创建 benchmark profile
- **WHEN** benchmark schema migration 被执行
- **THEN** PostgreSQL 必须创建 benchmark profile 表并保存类型、官方 series code、provider、期限、标的 symbol、币种、单位、频率和来源 URL

#### Scenario: 创建 benchmark observations
- **WHEN** benchmark schema migration 被执行
- **THEN** PostgreSQL 必须创建 observation 表、benchmark 与时间索引、来源与时间幂等约束和质量状态约束

#### Scenario: 非破坏性升级
- **WHEN** 已有业务数据的数据库应用 benchmark migration
- **THEN** migration 不得删除、清空或重建既有实体、profile、关系、采集、事件或投影运行记录

#### Scenario: Observation 引用 benchmark
- **WHEN** 系统保存 benchmark observation
- **THEN** observation 必须引用 `entity_type=benchmark` 的有效实体，并拒绝引用 index、metric、commodity 或 instrument

### Requirement: Event 原子事实载荷
系统 SHALL 在既有 `events` 事实表上增加 `fact_payload JSONB NOT NULL DEFAULT '{}'`，用于保存可独立验证的原子 Event 事实；该载荷不得表达预测、评分或直接投资建议。

#### Scenario: 保存 JSON object 事实载荷
- **WHEN** 确定性执行器写入通过 evidence 校验的 Event package
- **THEN** `fact_payload` 必须接受 JSON object（包括空 object），并与既有 title、summary、时间、状态和 `dedupe_key` 字段共同保存

#### Scenario: 拒绝非 object 或越界载荷
- **WHEN** candidate payload 不是 JSON object，或包含买入卖出、涨跌预测、利好利空、传导强度、事件评分或直接投资建议
- **THEN** domain/repository contract 必须拒绝该 payload，不得写入事件事实

### Requirement: Event 证据镜像与最小归因
系统 SHALL 通过既有 `event_sources` 将 Event 与 `raw_documents` 关联，并支持已批准的最小证据归因字段；`raw_documents` 仍是证据镜像，不能被 Event 事实替代。

#### Scenario: 保存可追溯 evidence
- **WHEN** Event package 由一个或多个 raw document 支持
- **THEN** 系统必须保留现有 `source_level`、`evidence_excerpt`、`evidence_hash`，并可写入 `evidence_relation VARCHAR(32) NULL` 与 `supports_fields TEXT[] NOT NULL DEFAULT '{}'`

#### Scenario: 解释 evidence 字段
- **WHEN** 新执行器写入 `evidence_relation`
- **THEN** `supports`、`contradicts`、`context` 必须使用受控值，legacy NULL 必须解释为 unknown；新 `supports`/`contradicts` 写入必须有非空 `supports_fields`

#### Scenario: evidence 幂等
- **WHEN** 同一 Event/raw document/evidence hash 被重复提交
- **THEN** repository 必须先检查重复并跳过无意义写入，数据库唯一约束必须阻止重复三元组

### Requirement: 受控 Tag 分类和归因
系统 SHALL 以 Tidewise DB 的 `event_tag_defs` 作为 Tag 主数据唯一权威，保留 `event_tag_maps` 的既有唯一约束，并支持 Tag 置信度和分配理由；YAML 只能作为策略输入。

#### Scenario: 映射受控 Tag
- **WHEN** Event candidate 携带 Tag candidate
- **THEN** 系统必须按 DB 中已注册的 `event_tag_defs` 映射；YAML 不得创建或替代 Tag 主数据

#### Scenario: 保存 Tag 归因
- **WHEN** AI 或规则分配通过确定性校验
- **THEN** 可写入 `confidence NUMERIC(5,4) NULL CHECK (confidence >= 0 AND confidence <= 1)` 与 `assignment_reason TEXT NOT NULL DEFAULT ''`；新 AI/规则分配的 `assignment_reason` 必须非空，既有记录保持可读兼容

### Requirement: 事件字段扩展审慎门槛
系统 SHALL 对除 `fact_payload` 外的 `events` 新字段逐项评估，不得将未 Review 字段预批准。

#### Scenario: 延后非必要字段
- **WHEN** Review 讨论 `event_type_code`、`action_code`、`lifecycle_status`、`reference_period_start/end`、`last_seen_at` 或 `event_time_precision`
- **THEN** `event_type_code`/`action_code` 优先 defer 至受控 Tag 或 payload；`lifecycle_status` 因与 `event_status` 重叠 defer；`reference_period_start/end` 先放 payload；`last_seen_at` 可由 event_sources 聚合而 defer；`event_time_precision` 仅作为待用户 Review 的独立事实语义候选

### Requirement: Event schema 非破坏性增量 migration
系统 SHALL 通过编号 `000019` 的增量 migration 增加经 Review 批准的字段和约束，并保留既有数据。

#### Scenario: 保留 migration 前状态
- **WHEN** local PostgreSQL 执行 Event schema migration
- **THEN** migration 后受影响表行数必须与 fresh before 相同，且 migration 版本、目标列和约束必须可验证

#### Scenario: 失败安全
- **WHEN** schema、count、backup、migration 或 after assertion 失败
- **THEN** 系统必须立即停止剩余 scope，并采用 backup recovery 或 fail-closed/forward-fix 兼容策略，不得清空、回填或删除业务数据
