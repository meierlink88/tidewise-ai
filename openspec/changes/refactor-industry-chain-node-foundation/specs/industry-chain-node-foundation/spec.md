## ADDED Requirements

### Requirement: 统一产业链节点实体
系统 SHALL 使用 `entity_type=chain_node` 表达所有粗粒度与细粒度产业概念，并复用 `entity_nodes` 的身份、名称、aliases、状态与时间字段，不得建立 sector 逻辑实体、独立 industry_chain 容器或固定全局层级。

#### Scenario: 保存产业概念
- **WHEN** 经 Review 的粗粒度或细粒度产业概念进入主数据
- **THEN** 系统必须保存一个 `entity_type=chain_node` 的 `entity_nodes` row
- **AND** 不得同时创建同义 sector、industry_chain 容器或 membership

#### Scenario: 粗节点作为视角入口
- **WHEN** 某个粗粒度节点在特定分析视角中充当产业链入口
- **THEN** 系统必须仍将其保存为普通 chain_node
- **AND** 不得在 profile 中保存全局 L1/L2/L3、parent、产业链名称或容器身份

### Requirement: 最小 chain node profile
系统 SHALL 使 `chain_node_profiles` 仅保存 `entity_id UUID` 主外键、非空 `definition TEXT` 和可空 `boundary_note TEXT`，并复用主实体层的中文名、英文及其他 aliases。

#### Scenario: 保存明确节点
- **WHEN** 系统保存边界无歧义的 chain_node
- **THEN** `definition` 必须为非空文本
- **AND** `boundary_note` 可以为 NULL

#### Scenario: 保存歧义节点
- **WHEN** chain_node 可能与粗细相邻节点范围重叠
- **THEN** 系统必须用非空 `boundary_note` 说明包含与排除范围

#### Scenario: 拒绝冗余字段
- **WHEN** profile 输入包含 position、category、unit_of_analysis、level、parent、market、source 或 observation 字段
- **THEN** validator 必须拒绝把这些字段保存到 `chain_node_profiles`

### Requirement: 产业候选来源边界
系统 SHALL 只把同花顺、东方财富外部分类作为 chain_node 初始化候选参考，不得建立生产 source mapping 表或把外部分类字段写入节点 profile。

#### Scenario: 整理外部候选
- **WHEN** 系统从同花顺或东方财富分类整理 chain_node 候选
- **THEN** 来源链接、快照时间和筛选理由必须留在候选 Review、OpenSpec 或 seed 评审材料
- **AND** 不得创建 `chain_node_source_mappings`

#### Scenario: 过滤非产业标签
- **WHEN** 外部候选表达涨停、融资融券、高股息或其他交易状态、机制、风格标签
- **THEN** 候选必须被排除，不得创建 chain_node

### Requirement: 唯一产业节点关系表
系统 SHALL 使用独立且唯一的 `chain_node_relations` 保存 chain_node 静态关系，不得复用 `entity_edges`，也不得恢复 industry_chain membership/topology 双表或 `industry_chain_entity_id`。

#### Scenario: 保存节点关系
- **WHEN** 经 Review 的产业节点关系进入 PostgreSQL
- **THEN** 系统必须保存关系 ID、有向端点、关系类型、mechanism、可选 condition、evidence/provenance、状态和时间
- **AND** 两个端点都必须引用有效 chain_node profile

#### Scenario: 拒绝非法结构
- **WHEN** 关系为自环、端点不是 chain_node、方向被自动对调或 from/type/to tuple 重复
- **THEN** 数据库或 validator 必须拒绝写入

### Requirement: 四类 MVP 关系语义
系统 SHALL 仅允许 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on` 四类有向关系，并按已批准的可判定语义验证方向和机制。

#### Scenario: 判定分类关系
- **WHEN** A 的全部实例属于 B
- **THEN** 系统可以保存 `A -> B is_subcategory_of`
- **AND** 不得用该类型表达物理组成或投入依赖

#### Scenario: 判定组成关系
- **WHEN** A 是 B 的可识别物理或系统组成
- **THEN** 系统可以保存 `A -> B is_component_of`
- **AND** 不得用该类型表达分类范围

#### Scenario: 判定直接投入
- **WHEN** A 的输出被 B 作为可识别输入消耗
- **THEN** 系统可以保存 `A -> B input_to`

#### Scenario: 判定功能依赖
- **WHEN** A 的目标功能或产出在 B 缺失或受限时会受约束，且不是分类、组成或直接投入
- **THEN** 系统可以保存 `A -> B depends_on`

#### Scenario: 拒绝已删除关系类型
- **WHEN** 输入使用 `contains`、`supplies_to`、`substitutes_for` 或 `transmits_to`
- **THEN** validator 必须拒绝写入

#### Scenario: 拒绝同一机制双重登记
- **WHEN** 相同有向端点与同一客观机制已登记为 `input_to` 或 `depends_on`
- **THEN** 系统必须拒绝再以另一类型登记

### Requirement: 动态事件传导隔离
系统 SHALL 将事件传导方向、强度、时滞和结论留给事件推理流程动态计算，不得将其固化为 chain_node 主数据或静态关系类型。

#### Scenario: 事件沿产业关系推导
- **WHEN** 后续事件推理沿 `input_to` 或 `depends_on` 路径分析影响
- **THEN** 推导结论必须保存到后续事件推理契约定义的结构
- **AND** 不得写入 `chain_node_relations` 作为 `transmits_to`

### Requirement: physical constraint subject 迁移
系统 SHALL 继续独立保存 physical constraints，并使每条 constraint 恰好引用一个 chain_node 或一个 `chain_node_relations` row。

#### Scenario: 迁移节点约束
- **WHEN** 旧 constraint 引用语义保持的 chain_node
- **THEN** 系统必须保留其 constraint ID、证据与节点 subject

#### Scenario: 迁移旧 topology edge 约束
- **WHEN** 旧 constraint 引用 topology edge 且该 edge 已通过 Review 唯一映射到新 relation
- **THEN** 系统必须把 subject 前向迁移为该 `chain_node_relations.id`

#### Scenario: 阻断歧义约束
- **WHEN** 旧 topology edge 被删除、拆分或无法唯一映射
- **THEN** 系统不得猜测 constraint subject
- **AND** 必须报告阻断项并停止该层验收

### Requirement: 分阶段有状态操作门禁
系统 SHALL 先完整验收 Phase A 节点模型与初始化，再进入 Phase B 关系建立；schema、业务 data 与旧结构 cleanup 必须各自执行独立的 `Review -> Write -> Query`，每个 PostgreSQL Write 前取得人工授权且每次 Write 后完成 Query 验收。

#### Scenario: Phase A schema 门禁
- **WHEN** schema/migration 实现、preflight、影响、备份和回滚边界已准备完成
- **THEN** 系统必须先提交 schema diff 供 Review 并单独取得 schema Write 授权
- **AND** Write 后必须立即 Query schema、约束、版本、引用与幂等结果并等待验收

#### Scenario: Phase A 节点 data 门禁
- **WHEN** Phase A schema Query 已验收且 chain_node 候选清单已准备完成
- **THEN** 系统必须提交候选映射与 data 影响供 Review 并单独取得 data Write 授权
- **AND** Write 后必须立即 Query counts、profile 完整性、重复、孤儿、stable references 与幂等结果并等待验收

#### Scenario: Phase A 完整验收
- **WHEN** Phase A schema 与节点 data 的 Query 尚未全部验收
- **THEN** 系统不得进入 Phase B

#### Scenario: Phase B relation schema 门禁
- **WHEN** relation schema 实现、preflight、影响、备份和回滚边界准备完成
- **THEN** 系统必须先 Review 并单独取得 schema Write 授权
- **AND** Write 后必须立即 Query 表、约束、索引、FK、版本与幂等结果并等待验收

#### Scenario: Phase B relation data 门禁
- **WHEN** relation schema Query 已验收，且四类候选边及 constraint 映射准备完成
- **THEN** 系统必须再次 Review 并单独取得 data Write 授权
- **AND** Write 后必须立即 Query 方向、端点、唯一性、机制冲突、孤儿、constraint subject 与幂等结果并等待验收

#### Scenario: 旧结构 cleanup 门禁
- **WHEN** relation/constraint data Query 已验收且需要停用或移除旧生产结构
- **THEN** 系统必须提交引用扫描、影响、备份和回滚边界供 Review，并单独取得 cleanup Write 授权
- **AND** Write 后必须立即 Query 旧结构状态、引用完整性、孤儿、counts 与重复执行幂等结果并等待验收

#### Scenario: 不重建 Neo4j
- **WHEN** Phase A 或 Phase B PostgreSQL 验收完成
- **THEN** 本 change 不得写入、清理或重建 Neo4j

### Requirement: 旧事实前向迁移
系统 SHALL 将既有 sector、industry_chain、membership、topology、physical constraint facts 与旧 Neo4j projection 视为迁移输入，使用版本化、幂等的 forward migration 处理，不得回滚历史或手工清库。

#### Scenario: 保留稳定身份
- **WHEN** 旧实体与目标 chain_node 语义一致
- **THEN** 系统必须优先复用旧 UUID 并迁移已有引用

#### Scenario: 合并旧身份
- **WHEN** 多个旧实体收敛到一个 chain_node
- **THEN** 系统必须保存显式 legacy-to-target 审计并迁移已注册引用，不得静默删除旧 ID

#### Scenario: 条件性 entity key 唯一约束
- **WHEN** 全库 preflight 尚未证明所有 `entity_key` 非空、无重复且写路径兼容
- **THEN** 系统不得增加全局唯一约束
- **AND** 只有 preflight 零冲突并单独获批后才可实施

#### Scenario: 重复执行迁移
- **WHEN** 已成功完成的 migration 或 seed 被再次执行
- **THEN** 系统必须报告 unchanged 或 already-applied，不得新增重复实体、关系或审计记录
