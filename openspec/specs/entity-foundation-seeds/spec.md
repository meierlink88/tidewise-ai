## Purpose

定义观潮家一阶段实体基础库 seed 的当前系统事实，覆盖基础实体、类型 profile、客观关系、校验边界、幂等写入和可审阅 report。

## Requirements

### Requirement: 实体基础 seed 数据
系统 SHALL 提供一阶段实体基础 seed 数据，用于初始化六层传导和事件知识图谱所需的基础实体、profile 和经过分批审阅的客观关系；实体主数据 seed 与关系 seed 必须解耦，空关系基线不得自动恢复历史样例关系，并将 benchmark 作为独立于 index、metric、commodity 和 instrument 的实体类型初始化。产业概念必须统一初始化为 chain_node，theme 只建立类型能力而不得在本 change 自行初始化实例。

#### Scenario: 第一批 chain node 名称范围
- **WHEN** 第一批 chain_node data contract 进入 Review
- **THEN** 系统必须只以已批准工作簿 Sheet「标准化保留」的 842 个互异标准化节点名作为 canonical 范围
- **AND** 950 个原始名称必须按契约进入 name/canonical/aliases，108 个同义合并不得重新产生重复实体
- **AND** aliases 必须 trim、去重并按确定性字符串顺序稳定排序；同一 alias 集合仅输入顺序变化不得改变 identity action 或 checksum
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

### Requirement: 实体 seed 校验
系统 SHALL 在写入数据库前校验实体 seed 的结构、引用关系和安全边界。

#### Scenario: 拒绝重复实体 key
- **WHEN** seed 文件中出现重复实体 key
- **THEN** loader 或 validator 必须返回明确错误，并阻止继续写入

#### Scenario: 拒绝悬空 profile 或关系
- **WHEN** profile 或实体关系引用不存在的实体 key
- **THEN** validator 必须返回明确错误，并阻止继续写入

#### Scenario: 禁止推理结论字段
- **WHEN** seed 文件包含利好利空、预测结论、传导强度、事件评分或投资建议字段
- **THEN** validator 必须返回明确错误

### Requirement: 实体 seed 报告
系统 SHALL 在实体 seed 执行后输出可审阅 report，使开发者能够确认初始化范围和结果。

#### Scenario: 输出初始化统计
- **WHEN** 实体 seed 命令执行完成
- **THEN** report 必须包含实体总数、按实体类型统计、按层级统计、各 profile 表写入数量、关系类型统计、created、updated、unchanged 和 failed 数量

### Requirement: Change-Specific Alliance Economy Importer
系统 SHALL 为本 change 提供一个只接受 frozen approved artifact 的最小 importer，优先复用现有 entity-seed/repository，并拒绝演变为通用数据导入框架。

#### Scenario: 加载 Frozen Artifact
- **WHEN** importer 启动 preflight 或获授权的 rebuild
- **THEN** 必须验证固定版本、checksum、45 alliance、79 economy、133 `member_of`、四个联盟字段、端点和方向；不接受 Excel、旧 CSV、旧 223 disposition 或任意外部 manifest 作为执行输入

#### Scenario: 限制实现范围
- **WHEN** 开发者实现 importer
- **THEN** 只能增加本批所需的 loader/validator、最小 repository 适配和固定入口，不得增加通用 service、policy engine、任意实体 mapping framework、计划语言或复杂 dry-run/report 子系统

#### Scenario: 保持 Economy 现有结构
- **WHEN** importer 映射 approved economy
- **THEN** 必须写入现有 `country_code/currency_code/region` profile，不得要求 `identity_kind`、新区域/货币规则或全局 `entity_key` 唯一索引

### Requirement: Read-Only Dependency Preflight
系统 SHALL 在 R3 cleanup Review 前输出可审计的只读 dependency package，且不得在 R1 执行 migration、seed 或 database write。

#### Scenario: 报告目标与引用
- **WHEN** preflight 审计 local PostgreSQL
- **THEN** 必须按表、FK、relation type、方向和 endpoint type 报告目标 counts/hash，并覆盖 alliance/economy profiles、entity edges、external identifiers 以及 market/sector/industry-chain/company/person 等对 economy 的引用

#### Scenario: 报告跨域事实
- **WHEN** 发现不由 45/79/133 重建的 economy/alliance 跨域关系或引用
- **THEN** preflight 必须报告其 count/hash；已确认的 economy 与跨域事实必须保留，其他 alliance incident edge 或审计漂移即 fail-closed，不得由 importer 静默级联删除

#### Scenario: 防止错误环境
- **WHEN** 环境不能被明确证明为获批 local 探索数据库
- **THEN** cleanup/rebuild 入口必须拒绝执行，不得把 local 豁免推广到 UAT、prod 或 shared

### Requirement: Scoped Cleanup 与 Latest Rebuild
系统 SHALL 把 cleanup 与 rebuild 实现为两个独立、可审阅、fail-closed 的执行包；前者为 R3，后者为 R2。

#### Scenario: 精确清理并断言 Zero
- **WHEN** 4.1 R3 获得明确授权且 preflight 未漂移
- **THEN** importer 必须只删除 `alliance_org`、`alliance_org_profiles` 与 economy → alliance_org `member_of`，并在提交/进入下一包前证明 alliance/profile/member scope 为零、economy/profile 仍为 50，且保护 hash 不变

#### Scenario: 原位重建目标 Economy
- **WHEN** 4.2 rebuild 获得独立授权
- **THEN** 35 个现有 target economy 必须保留 stable ID/entity_key 后原位 upsert，44 个缺失 target 才创建，15 个 non-target economy/profile 不得被 manifest convergence 删除、停用或改写

#### Scenario: 精确重建并查询
- **WHEN** 4.1 zero Query 已验收且 4.2 R2 获得独立授权
- **THEN** importer 必须以单事务或明确 fail-closed 边界重建 45/79/133，并输出 exact counts、端点、方向、孤儿、重复与 checksum

#### Scenario: 幂等复跑
- **WHEN** 对已经符合 frozen manifest 的 local 数据再次运行 4.2
- **THEN** 不得创建重复 entity/profile/edge 或改变集合，Query 必须报告 45/79/133 仍精确成立

#### Scenario: 漂移时停止
- **WHEN** 环境、manifest checksum、preflight count/hash、FK/关系类型、跨域决策或 Query assertion 与授权包不一致
- **THEN** importer 必须停止且不得把旧批准解释为扩大 scope 的权限

### Requirement: 联盟与 Economy Rebuild 自动化验证
系统 SHALL 对最小 migration、manifest validator、repository/importer、精确 scope、原子性、zero/post assertions 与幂等提供 targeted tests。

#### Scenario: 运行验证
- **WHEN** 开发者验证本 change 实现
- **THEN** 相关包测试、migration 静态或隔离 integration tests、受影响 backend suite、共享 architecture/contract tests 与 OpenSpec strict 必须通过；普通测试不得访问真实外部网络或写真实 PostgreSQL/Neo4j
