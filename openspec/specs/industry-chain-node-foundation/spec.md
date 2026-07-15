## Purpose

定义统一产业链节点、最小 profile、静态关系、物理约束边界及分阶段有状态操作门禁的当前系统事实。

## Requirements

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

#### Scenario: 约束 subject 与证据强度
- **WHEN** physical constraint 的 subject 是过宽节点，或仅有节点名称/definition 支持
- **THEN** 候选必须保持 blocked，并先收窄到具体节点或关系 subject
- **AND** 不得把价格、政策、情绪、市场表现或无 source URL 的弱推断升级为物理约束

### Requirement: 分阶段有状态操作门禁
系统 SHALL 先完整验收 Phase A cleanup、通用外部标识 schema、全新节点/profile 初始化与外部标识 mapping data，再进入 Phase B 关系建立。structure implementation checkpoint 的通过不构成任何数据库 Write 授权；cleanup、external identifier schema、node/profile seed 与 mapping data 必须分别执行独立的 `Review -> Write -> Query`。

#### Scenario: Phase A cleanup 门禁
- **WHEN** 完整引用审计、可恢复备份、精确删除范围、影响和回滚边界已准备完成
- **THEN** 系统必须先提交 cleanup diff/dry-run 供 Review 并单独取得 cleanup Write 授权
- **AND** Write 后必须立即 Query 旧类型、旧表、引用、孤儿、非目标保护与幂等结果并等待验收

#### Scenario: Migration target 分层执行
- **WHEN** cleanup 与 external identifier schema 同时处于 pending migration
- **THEN** 标准 dbmigrate 路径必须支持显式 target version，并在 cleanup Write 只执行到 `000015`
- **AND** 成功报告必须只列出实际 applied migration，`000016` 必须保持 pending 直到 schema Write 单独授权
- **AND** 无 target 的既有 apply-all 行为不得变化，非法、回退或不存在的跳跃 target 必须在 Write 前拒绝
- **AND** AutoApply 必须在同一个 pinned PostgreSQL connection 上 acquire/release session advisory lock，并在持锁后读取 before 状态、执行后读取 after 状态
- **AND** applied/remaining 必须由 after current/pending 与 before pending 推导，不得把 Executor 的预选 migration 列表宣称为实际结果；target 未到达、状态矛盾或 unlock 失败必须使操作失败

#### Scenario: Phase A data contract 与实现门禁
- **WHEN** structure implementation checkpoint 已通过且第一批名称范围已批准
- **THEN** 系统必须先单独 Review aliases、definition/boundary、全新 UUID/key、外部标识 schema、taxonomy、去重、幂等与 dry-run/report 契约
- **AND** 契约批准前不得实现 final seed、外部标识 schema 或请求任何 Write
- **AND** 契约批准后必须先提交 TDD implementation diff、测试、schema diff 与 dry-run 格式供 Review

#### Scenario: Phase A external identifier schema 门禁
- **WHEN** cleanup Query 已验收且外部标识 schema implementation 已通过 Review
- **THEN** 系统必须单独取得 schema Write 授权
- **AND** Write 后必须立即 Query 表、列、FK、唯一约束、索引、版本与幂等结果并等待验收
- **AND** schema Write 必须显式 target `000016`，不得通过让后续 migration 故意失败来切分状态

#### Scenario: Phase A node/profile seed 门禁
- **WHEN** external identifier schema Query 已验收
- **THEN** 系统必须提交 842 个 node/profile 的 final seed dry-run、definition/boundary、aliases、全新 key/ID 与影响并单独取得 seed Write 授权
- **AND** Write 后必须立即 Query counts、profile 完整性、新 key/ID、重复、孤儿与幂等结果并等待验收

#### Scenario: Phase A mapping data 门禁
- **WHEN** node/profile seed Query 已验收
- **THEN** 系统必须提交 1,169 条逐行 external identifier mapping report 并单独取得 mapping Write 授权
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
- **THEN** 系统必须先在同一 task package 内完成 relation-only atomic write runner、manifest/snapshot validator、提交前 assertions 与 dry-run/report 的 R1 技术验收
- **AND** 只有可写 manifest 精确冻结后，才可形成 data R2 package 并单独取得 Write 授权
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

### Requirement: 在当前 842 个 chain_node 间建立 usable-map 四类关系
系统 SHALL 以重新冻结的当前 842 个 active chain_node 为硬边界，只在这些既有节点之间发现和审核 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`，并原样保留既有 100 条 accepted baseline。

#### Scenario: 冻结 842 节点基线
- **WHEN** Package 2 准备开始关系审计
- **THEN** 必须冻结 active chain_node 的 identity、count 与 hash
- **AND** 若最终 count 不是 842 或 identity 集合发生漂移，必须回到 Review，不得自行改写范围

#### Scenario: 全量候选发现覆盖完成
- **WHEN** 842 全量业务目标申请完成验收
- **THEN** 842/842 节点都必须参加候选发现与两遍独立检查
- **AND** 不得存在未处置候选、未解决异常或冲突；无事实节点允许保持无边

#### Scenario: 覆盖不强制伪造关系
- **WHEN** 某个节点没有满足语义与证据门槛的关系
- **THEN** coverage ledger 必须证明该节点已完成发现与双遍检查
- **AND** 不得为使节点拥有某类边而创建虚假关系或强制 node×relation_type 三态

#### Scenario: usable-map input_to
- **WHEN** A 的实际产出被消耗、转化、嵌入，或作为明确服务输出沿可解释产业链进入 B 的生产、产品、交付或运行过程
- **THEN** 可以登记方向为 A 到 B 的 `input_to`，即使当前节点集合缺少中间环节，但必须列出已知或缺失中间路径、具体进入机制、成立条件和真实替代路线
- **AND** 设备、工具、软件能力或基础设施仅因使 B 能生产或运行不得登记为 `input_to`；也不得在未证明硬前提时机械改为 `depends_on`
- **AND** 名称邻近、市场相关、共同事件或宽泛行业相邻不得作为登记依据

#### Scenario: 分类和组成采用全称边界
- **WHEN** 候选类型为 `is_subcategory_of` 或 `is_component_of`
- **THEN** 分类必须证明源节点全部稳定实例属于目标，组成必须证明目标定义边界内全部实例包含源组件
- **AND** 供应商或规格可替代不能排除“部分目标实例完全不含该组件”的反例；存在该反例时必须阻断

#### Scenario: 既有节点集合是硬边界
- **WHEN** 某个候选关系无法由当前 842 个既有 chain_node 表达
- **THEN** 该候选必须阻断并记录无法表达的原因
- **AND** 不得创建细分 chain_node，不得修改节点主数据、profile、external identifier 或 identity

### Requirement: 全量研究可分批审阅但不降低关闭边界
系统 SHALL 允许将 842 节点的候选研究、double-check 与 Review 按节点群分批进行，并分别记录批次范围与逐节点发现覆盖进度；不得把局部批次视为 Package 或 change 完成。

#### Scenario: 分批候选进入 Review
- **WHEN** 某个审阅批次的既有节点关系候选完成
- **THEN** 必须展示有向端点、relation type、机制、条件、证据 tier/来源、反例、置信度与双遍 disposition
- **AND** physical constraints 或其他关系类型不得进入本 change

#### Scenario: 批次不构成关闭边界
- **WHEN** 某个研究批次完成候选审核
- **THEN** 未覆盖节点必须继续保留在 842 节点发现账本中
- **AND** 只有 842/842 全部完成候选发现与双遍检查时才能申请冻结 additive manifest
