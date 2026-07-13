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
系统 SHALL 只把同花顺、东方财富外部分类作为 chain_node 初始化候选和外部 identity 参考，不得恢复 sector mapping、建立 `chain_node_source_mappings` 或把外部分类字段写入节点 profile；经 Review 的外部代码只能进入通用 `entity_external_identifiers`。

#### Scenario: 整理外部候选
- **WHEN** 系统从同花顺或东方财富分类整理 chain_node 候选
- **THEN** 来源链接、快照时间和筛选理由必须留在候选 Review、OpenSpec 或 seed 评审材料
- **AND** 不得创建 `chain_node_source_mappings`
- **AND** 外部标识必须逐行保存，不得把 provider/code 拼接字符串写入 profile 或 identifier 字段

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

### Requirement: 历史 physical constraint 清理
系统 SHALL 在 cleanup 中删除全部历史 industry-chain physical constraints 及其旧 topology/node subject，不得迁移或复用旧 constraint ID；未来 constraint 只能基于新节点/关系重新 Review 后创建。

#### Scenario: 清理历史约束
- **WHEN** cleanup Write 获得授权
- **THEN** 系统必须先删除引用旧 topology 或旧 chain_node 的历史 constraints
- **AND** cleanup Query 必须证明历史 constraint 与 subject 引用均为零

#### Scenario: 提出新约束
- **WHEN** 后续需要为新 chain_node 或 `chain_node_relations` 建立 physical constraint
- **THEN** 必须作为全新候选保存新的 ID、证据和 subject
- **AND** 不得从旧 constraint 或 topology edge 机械迁移

### Requirement: 分阶段有状态操作门禁
系统 SHALL 先完整验收 Phase A cleanup、通用外部标识 schema、全新节点/profile 初始化与外部标识 mapping data，再进入 Phase B 关系建立。structure implementation checkpoint 的通过不构成任何数据库 Write 授权；cleanup、external identifier schema、node/profile seed 与 mapping data 必须分别执行独立的 `Review -> Write -> Query`。

#### Scenario: Phase A cleanup 门禁
- **WHEN** 完整引用审计、可恢复备份、精确删除范围、影响和回滚边界已准备完成
- **THEN** 系统必须先提交 cleanup diff/dry-run 供 Review 并单独取得 cleanup Write 授权
- **AND** Write 后必须立即 Query 旧类型、旧表、引用、孤儿、非目标保护与幂等结果并等待验收

#### Scenario: Phase A data contract 与实现门禁
- **WHEN** structure implementation checkpoint 已通过且第一批名称范围已批准
- **THEN** 系统必须先单独 Review aliases、definition/boundary、全新 UUID/key、外部标识 schema、taxonomy、去重、幂等与 dry-run/report 契约
- **AND** 契约批准前不得实现 final seed、外部标识 schema 或请求任何 Write
- **AND** 契约批准后必须先提交 TDD implementation diff、测试、schema diff 与 dry-run 格式供 Review

#### Scenario: Phase A external identifier schema 门禁
- **WHEN** cleanup Query 已验收且外部标识 schema implementation 已通过 Review
- **THEN** 系统必须单独取得 schema Write 授权
- **AND** Write 后必须立即 Query 表、列、FK、唯一约束、索引、版本与幂等结果并等待验收

#### Scenario: Phase A node/profile seed 门禁
- **WHEN** external identifier schema Query 已验收
- **THEN** 系统必须提交 842 个 node/profile 的 final seed dry-run、definition/boundary、aliases、全新 key/ID 与影响并单独取得 seed Write 授权
- **AND** Write 后必须立即 Query counts、profile 完整性、新 key/ID、重复、孤儿与幂等结果并等待验收

#### Scenario: Phase A mapping data 门禁
- **WHEN** node/profile seed Query 已验收
- **THEN** 系统必须提交 1,156 条逐行 external identifier mapping report 并单独取得 mapping Write 授权
- **AND** Write 后必须立即 Query provider counts、taxonomy、external name/code、唯一性、entity 绑定、孤儿与幂等结果并等待验收

#### Scenario: Phase A 完整验收
- **WHEN** Phase A cleanup、external identifier schema、node/profile seed 与 mapping data 的 Query 尚未全部验收
- **THEN** 系统不得进入 Phase B

#### Scenario: Phase B relation schema 门禁
- **WHEN** relation schema 实现、preflight、影响、备份和回滚边界准备完成
- **THEN** 系统必须先 Review 并单独取得 schema Write 授权
- **AND** Write 后必须立即 Query 表、约束、索引、FK、版本与幂等结果并等待验收

#### Scenario: Phase B relation data 门禁
- **WHEN** relation schema Query 已验收，且基于新节点的四类候选边及任何全新 constraint 候选准备完成
- **THEN** 系统必须再次 Review 并单独取得 data Write 授权
- **AND** Write 后必须立即 Query 方向、端点、唯一性、机制冲突、孤儿、constraint subject 与幂等结果并等待验收

#### Scenario: 不重建 Neo4j
- **WHEN** Phase A cleanup 使 PostgreSQL 不再包含旧产业事实
- **THEN** 本 change 不得写入、清理或重建 Neo4j
- **AND** 必须明确记录既有 Neo4j projection 已暂时陈旧并留给后续独立 change

### Requirement: 旧事实受控清理与延后导入
系统 SHALL 在可恢复备份和完整引用审计后，使用版本化、幂等 migration/受控命令清除旧 sector、industry_chain、chain_node、membership、topology、physical constraint 及相关关系/审计引用；全新 chain_node 导入必须等待后续数据范围和身份契约 Review，不得回滚历史、手工清库或复用旧 ID/key。

#### Scenario: 精确清理旧身份与引用
- **WHEN** cleanup Write 获得授权
- **THEN** 系统必须按 FK/逻辑引用顺序删除旧 profile、source mapping、membership、topology、constraint、entity edge、event link、convergence/audit 和旧 entity rows
- **AND** 不得删除 alliance、economy/country、market、benchmark/index 等非目标事实

#### Scenario: 全新生成身份
- **WHEN** 最终批准的 chain_node seed 进入写入
- **THEN** 系统必须遵循后续独立 Review 批准的 UUID/entity_key 契约
- **AND** 不得复用或收敛旧 sector、industry_chain、chain_node 的 UUID/entity_key

#### Scenario: 条件性 entity key 唯一约束
- **WHEN** 全库 preflight 尚未证明所有 `entity_key` 非空、无重复且写路径兼容
- **THEN** 系统不得增加全局唯一约束
- **AND** 只有 preflight 零冲突并单独获批后才可实施

#### Scenario: 重复执行 cleanup 或 seed
- **WHEN** 已成功完成的 cleanup 或 final seed 被再次执行
- **THEN** cleanup 必须报告 already-clean/unchanged 且不得扩大删除范围，seed 必须报告 unchanged 且不得新增重复实体
