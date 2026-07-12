## Purpose

定义观潮家一阶段实体基础库 seed 的当前系统事实，覆盖基础实体、类型 profile、客观关系、校验边界、幂等写入和可审阅 report。

## Requirements

### Requirement: 实体基础 seed 数据
系统 SHALL 提供一阶段实体基础 seed 数据，用于初始化六层传导和事件知识图谱所需的基础实体、profile 和经过分批审阅的客观关系；实体主数据 seed 与关系 seed 必须解耦，空关系基线不得自动恢复历史样例关系，并将 benchmark 作为独立于 index、metric、commodity 和 instrument 的实体类型初始化。

#### Scenario: 初始化联盟组织
- **WHEN** 实体 seed 执行
- **THEN** 系统必须初始化 10 个核心联盟组织实体，至少覆盖 `OPEC+`、`OPEC`、`G7`、`G20`、`WTO`、`IMF`、`World Bank`、`OECD`、`EU` 和 `BRICS`

#### Scenario: 初始化所有实体类型
- **WHEN** 实体 seed 执行
- **THEN** 系统必须按 `seed-scope.md` 初始化联盟组织、经济体、政策机构、市场、指数、benchmark、板块、产业链节点、公司、证券、交易工具、指标、商品和人物的一阶段基础数据，并在 report 中输出各类型数量

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
- **THEN** 同一 `source_system`、`source_taxonomy_type` 和非空 `source_sector_code` 只能指向一个 canonical sector；无代码来源必须通过确定性规范化来源名称和非空字符串 `source_market_scope` 形成稳定唯一身份，无范围时 scope 固定为空串，唯一键不得包含 `snapshot_date`

#### Scenario: 更新无代码来源快照
- **WHEN** 新快照再次出现同一无代码 source mapping identity
- **THEN** 系统必须幂等更新该 mapping 的最新 `rank_snapshot`、`snapshot_date` 和 `source_url`，不得为每个快照创建新 mapping；第一版历史快照只保留在 Git 版本化的 `openspec/changes/add-market-sector-foundation/candidate-review.md`，不新增历史 snapshot 表

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

### Requirement: 旧板块 canonical convergence

系统 SHALL 通过版本化结构化 manifest 和显式执行模式，把既有 source-bound sector 收敛到 reviewed canonical sector，禁止普通 seed 隐式制造重复 active 主数据。

#### Scenario: 普通 seed 遇到 active legacy sector
- **WHEN** 数据库存在 manifest 覆盖的 active legacy sector，且 canonical sector 尚未完成 convergence
- **THEN** 普通 `entity-seed` 必须在任何写入前失败，并提示使用经过 Review 的显式 convergence 模式

#### Scenario: 原子执行 convergence
- **WHEN** 操作者显式批准并执行 sector convergence
- **THEN** 系统必须在单一事务内校验 60 项 manifest、写入 52 个 canonical sector、迁移已注册引用、停用 60 个 legacy sector、写入审计记录、source mappings 和 reviewed relationships；任一步失败必须整体回滚

#### Scenario: 保留旧身份审计
- **WHEN** replace、merge、replace_with_existing_index 或无 target retirement 完成
- **THEN** 系统不得删除或复用旧 UUID，必须保留旧 entity key/profile 并将旧实体标记 inactive，同时在 `entity_convergences` 的通用 `target_entity_id/type` 保存结构化处置结果

#### Scenario: 迁移已知和未来引用
- **WHEN** legacy sector 被现有或未来 entity edge/FK 引用
- **THEN** sector target 必须按显式 reference registry 重定向引用；index target 只能重定向类型兼容引用并重新校验 relationship policy；无 target、sector 专属引用指向 index或未注册引用必须阻断 convergence，不能依赖当前数据库没有关系的偶然状态

#### Scenario: 保留 legacy 来源
- **WHEN** concept/industry legacy sector 收敛到等价 canonical sector target
- **THEN** 系统必须把旧中文名追加为 canonical alias，并从旧 profile 生成指向 canonical 的 legacy source mapping；无 target 或 index target 不得伪造 sector mapping 或 benchmark 实体

#### Scenario: 拒绝仅凭事件相关进行合并
- **WHEN** legacy sector 与候选 target 只存在事件、上下游或跨链相关，而名称和覆盖范围不等价
- **THEN** manifest 必须使用无 target retirement，不得创建 source mapping 或重定向到该相关实体

#### Scenario: 收敛到已有 index
- **WHEN** 误建为 sector 的 legacy index 与 `indices.json` 已有 index 名称和范围等价
- **THEN** convergence audit 必须指向 existing index entity，旧 sector 必须 inactive，且不得改变 index/benchmark 语义或创建 sector source mapping

#### Scenario: convergence 幂等
- **WHEN** 同一版本 convergence 或普通 seed 在成功后重复执行
- **THEN** 系统不得新增重复 entity、mapping、edge 或 audit row，且必须报告 already-converged/unchanged 结果

#### Scenario: 普通 seed 保留当前 convergence alias 所有权
- **WHEN** 普通 entity seed 更新已收敛 canonical entity 的 aliases
- **THEN** 系统必须以正式 seed aliases 加当前最大合法 manifest 的 alias mutation audit 作为最终集合，保留当前 audit-owned aliases，并删除不再属于正式 seed且没有当前 convergence audit 所有权的普通 alias

#### Scenario: alias 顺序不制造无意义更新
- **WHEN** entity 没有 current convergence-owned alias，或正式 alias 与 current audit alias 需要合并
- **THEN** 正式 aliases 必须按 seed 首次出现顺序保留，current audit aliases 必须按 mutation provenance 顺序去重追加；NULL/空数组、重复 alias 和两层同名 alias 不得导致重复值、全量排序或无意义 UPDATE

#### Scenario: 前向恢复缺失 convergence alias
- **WHEN** 已应用 convergence 的环境因旧版普通 seed 覆盖而缺失当前 alias mutation audit 记录的 alias
- **THEN** 前向 repair migration 必须只从当前最大 manifest 的 append-only alias audit 恢复缺失值，重复执行不得再次更新；尚无 convergence manifest 的 fresh 环境必须 no-op

#### Scenario: Memory 与 PostgreSQL 原子语义一致
- **WHEN** 相同 convergence manifest 分别由 MemoryRepository 和 PostgresRepository 执行
- **THEN** 两者必须对 preflight、引用冲突、rollback、状态变化和 report 产生等价结果

#### Scenario: 同版本重复执行幂等
- **WHEN** 已应用 manifest version 使用完全相同 checksum 和逐项 payload 再次显式执行
- **THEN** 系统必须返回 already-converged/unchanged，不得新增 manifest、convergence、mutation audit、entity、mapping 或 edge

#### Scenario: 新版本 append-only
- **WHEN** 人工 Review 批准完整新版本并通过显式前向纠错模式执行
- **THEN** 系统必须以独立 ID 追加 manifest 和每个 legacy 的新审计行，并保持所有旧版本行不变；`UNIQUE(legacy_entity_id, manifest_version)` 必须防止同版本重复审计

#### Scenario: 禁止修改历史审计
- **WHEN** repository 调用方尝试 UPDATE 或 DELETE 已应用的 convergence manifest、逐项结论或 mutation audit
- **THEN** repository 必须拒绝该操作，数据库约束、权限或 trigger 必须提供防御性保护

#### Scenario: 确定当前结论
- **WHEN** 同一 legacy entity 存在多个已应用 manifest version
- **THEN** 当前结论必须由最大合法 manifest version 确定，且每个版本只能存在一条该 legacy 的结论，不得依赖回写 `is_current`

#### Scenario: 阻断非法版本
- **WHEN** 新 manifest version 小于或等于当前版本但 payload 不同、跳过 previous version、缺少 Review 元数据、checksum 不匹配或不是完整 60 项
- **THEN** 系统必须在任何业务写入和审计 INSERT 前失败

#### Scenario: 前向纠错迁移当前状态
- **WHEN** 新版本修改某 legacy 的 target 或 action
- **THEN** 系统必须锁定并验证上一 current 结论、上一版本记录的 reference/alias mutations 和当前 legacy mapping，只迁移有 provenance 且未漂移的引用，并在同一事务追加新 audit；任何漂移或类型冲突必须整体回滚

#### Scenario: 纠错 mapping 与 alias
- **WHEN** target 从 sector 改为另一 sector、index 或无 target
- **THEN** 系统必须按新结论更新或拒绝 operational legacy mapping，只撤销由上一 convergence 添加且无其他 current 来源依赖的 alias，不得覆盖旧 audit 或盲目迁移 target 的其他引用
