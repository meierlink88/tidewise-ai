## MODIFIED Requirements

### Requirement: 实体基础 seed 数据
系统 SHALL 提供一阶段实体基础 seed 数据，用于初始化六层传导和事件知识图谱所需的基础实体、profile 和经过分批审阅的客观关系；实体主数据 seed 与关系 seed 必须解耦，空关系基线不得自动恢复历史样例关系，并将 benchmark 作为独立于 index、metric、commodity 和 instrument 的实体类型初始化。产业概念必须统一初始化为 chain_node，theme 只建立类型能力而不得在本 change 自行初始化实例。

#### Scenario: 第一批 chain node 名称范围
- **WHEN** 第一批 chain_node data contract 进入 Review
- **THEN** 系统必须只以已批准工作簿 Sheet「标准化保留」的 842 个互异标准化节点名作为 canonical 范围
- **AND** 950 个原始名称必须按契约进入 name/canonical/aliases，108 个同义合并不得重新产生重复实体
- **AND** 「宽边界保留」只能视为已保留节点的审阅子集，不得当作排除清单
- **AND** 工作簿不得直接作为可执行 seed，具体 UUID/entity_key、definition/boundary 与 dry-run/report 仍须 Review

#### Scenario: 第一批 definition 与 boundary 审阅
- **WHEN** 系统为 842 个 chain_node 准备 final seed dry-run
- **THEN** 每个节点必须包含说明“节点是什么”的非空 definition
- **AND** definition 不得是 canonical/alias 原样复制或“与该名称相关”等循环模板
- **AND** 对同名消歧、合并范围、粗细重叠或宽边界节点必须提供明确包含/排除范围的 boundary_note
- **AND** 边界清晰节点的 boundary_note 可以为 NULL

#### Scenario: 第一批全新身份与幂等
- **WHEN** 系统为第一批节点生成身份与 dry-run
- **THEN** 每个节点必须使用经 Review 的全新 UUID/entity_key，不得复用历史 sector/industry_chain/chain_node 身份
- **AND** report 必须列出 created/updated/unchanged/conflict、UUID/key/canonical 冲突、重复 aliases 与重复执行预期
- **AND** 现存 node snapshot 必须包含 entity_type、status、aliases、definition、boundary_note，且发现既有记录时 ID/key/canonical 三索引必须齐全；只有所有字段完全一致才可 unchanged，aliases/profile 漂移必须 updated，非 chain_node、非 active、snapshot 索引缺失或交叉不一致必须 conflict
- **AND** report 必须校验宽边界节点数恰为 79，且每个宽边界节点具有非空 boundary_note
- **AND** 任一 identity 冲突必须阻断 Write

#### Scenario: 初始化联盟组织
- **WHEN** 实体 seed 执行
- **THEN** 系统必须初始化 10 个核心联盟组织实体，至少覆盖 `OPEC+`、`OPEC`、`G7`、`G20`、`WTO`、`IMF`、`World Bank`、`OECD`、`EU` 和 `BRICS`

#### Scenario: 初始化所有实体类型
- **WHEN** 实体 seed 执行
- **THEN** 系统必须按 `seed-scope.md` 初始化联盟组织、经济体、政策机构、市场、指数、benchmark、产业链节点、公司、证券、交易工具、指标、商品和人物的一阶段基础数据，并在 report 中输出各类型数量
- **AND** 不得初始化 sector、industry_chain 容器或未经独立 Review 的 theme 实例

#### Scenario: 补充事件投研所需的最小市场实体
- **WHEN** 第二批经济体与市场关系进入正式 seed 前
- **THEN** 系统必须先通过人工 review 补充核心主权债券市场、关键商品交易场所和高事件敏感区域市场，并为每个新增市场保存稳定 entity key、规范名称、所属经济体、币种、市场类别和权威来源

#### Scenario: 区分抽象市场和交易场所
- **WHEN** 市场主数据同时包含抽象市场与具体交易场所
- **THEN** 市场 profile 必须保存可校验的市场类别，使事件推导能够以抽象市场作为主要影响落点并避免与下属交易场所重复计权

#### Scenario: 修正指数市场归属并限定正式指数
- **WHEN** 第三批市场与指数关系进入正式 seed 前
- **THEN** 系统必须通过人工 review 修正错误的指数市场归属，补充全球股票分析市场和三个高事件敏感区域股票指数，并且只为具有明确编制方法的正式指数生成 `market -> index` 关系

#### Scenario: 可观测基准不作为指数初始化
- **WHEN** seed 清单包含政府债券收益率、商品连续价格、现货价格或参考利率
- **THEN** 系统必须将其初始化为 benchmark 和对应 profile，不得将其作为 index 实体或 `tracks_index` 关系写入

#### Scenario: 全球分析市场不建立经济体归属
- **WHEN** 系统初始化全球股票或全球贵金属分析市场
- **THEN** 系统必须允许其作为指数或 benchmark 的分析市场，但不得使用 `economy:global -> market` 的 `has_market` 关系表达虚假的属地事实

#### Scenario: 空关系基线不恢复旧关系
- **WHEN** 实体主数据 seed 在尚无已审阅关系批次时执行
- **THEN** 系统必须保持实体和 profile 幂等初始化，但不得写入原有历史样例关系或任何其他未审阅关系

#### Scenario: 初始化已审阅客观基础关系
- **WHEN** 某一关系族已经通过人工 review 并加入正式关系 seed
- **THEN** 系统必须只写入成员关系、归属关系、市场指数关系和 benchmark 定义关系等可核验客观关系，并保存来源名称、来源 URL 和核验时间，不得写入推理结论或投资判断

#### Scenario: 按政治命名要求初始化中文主名称
- **WHEN** 实体 seed 初始化涉及中国香港或中国台湾的经济体、市场或机构
- **THEN** 系统必须使用包含“中国香港”或“中国台湾”的中文主名称和规范名称，并只将“香港”“台湾”等常见写法作为 aliases 保存

#### Scenario: 按产业链节点维护公司和证券快照
- **WHEN** 实体 seed 初始化公司和证券
- **THEN** 系统必须按每个具体产业链节点维护不少于 10 个代表性上市公司映射，去重后写入唯一公司主体，并为每家公司至少关联一个主证券

## REMOVED Requirements

### Requirement: 市场板块 seed 审阅准入
**Reason**: sector 不再是生产逻辑实体，外部板块只作为 chain_node 候选参考。
**Migration**: 将仍属产业概念的候选纳入 Phase A chain_node Review；非产业标签拒绝，来源证据仅保留在评审材料。

### Requirement: 市场板块 profile 校验
**Reason**: `sector_profiles` 与 `sector_source_mappings` 被统一节点模型取代。
**Migration**: 使用最小 `chain_node_profiles` 校验 definition/boundary；旧 sector mapping 受控删除且不迁移，获批外部代码只进入通用 `entity_external_identifiers`。

### Requirement: 市场板块关系 seed 策略
**Reason**: sector 关系不再进入 `entity_edges`，产业 topology 统一由四类 `chain_node_relations` 表达。
**Migration**: 旧关系随旧产业实体受控清理；Phase B 只基于全新节点与新证据重新提出关系候选。

### Requirement: 旧板块 canonical convergence
**Reason**: 固定 canonical sector 集合与 sector convergence 不再是目标模型。
**Migration**: 在可恢复备份后受控删除仅服务旧 sector convergence 的 manifest/audit/reference/alias 数据与结构；若扫描发现非 sector 生产用途则停止并提交保留理由供 Review。
