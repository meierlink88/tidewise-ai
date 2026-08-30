# Data Context

## Purpose

Data Domain Service 是当前唯一 Domain Service，负责稳定的数据事实、领域规则、持久化、受控导入和查询 API。

## Owns

- Entity 事实、独立 Object 事实、Object Schema、产业链节点及关系、
  Index 等正式事实。
- 完整 Raw Evidence、原子 Evidence、Evidence 阅读辅助 Keywords 及其确定性正式身份。
- 正式 Event、Event 与 Atomic Evidence 的证据关联，以及 Event-owned Actor/Asset 关系快照。
- 独立 Storyline 事实、三类蓝图锚点，以及 Storyline 与 Event 的当前关联事实。
- Research Theme、Theme Impact、Reason Tree 及其关联数据。
- PostgreSQL schema、migration 和 repository。
- 采集/清洗执行方使用的 Raw Evidence 与 Evidence Publication API、自然身份收敛、
  正式身份响应和事务规则。
- 面向 Miniapp/Admin Application Backend Service 的版本化 REST API。
- Data Service 自身的只读运行健康状态。
- Data Application 内数据库无关的 Domain Object ID 技术原语与格式合同。
- Source 管理、校验、固定清单初始化、持久化、动态 RSS Source 生命周期与
  完整 active Source 运行时快照。
- 面向外部 AgentOS 的版本化 Company 投影快照，包含 Data-owned Company 事实与
  formal CompanyIndustryLink。

## Does Not Own

- Miniapp 或 Admin Portal 的页面 DTO、交互状态和展示逻辑。
- User、Auth、Payment、Subscription 等未来独立领域。
- 数据采集 connector、parser、采集 prompt、采集调度执行或清洗/语义判断工作流。
- Agent 的模型推理和工作流运行。
- Entity seed、关系包构建/导入、历史主数据收敛批次，以及向 Neo4j 或 Qdrant
  投影 Entity 的执行能力；这些 authoring 能力转移给 Tidewise Reason。
- Neo4j、Qdrant、embedding Provider 的写入、同步、运行健康或生命周期管理。
- Event Semantic 候选、租约、审核、Submission、Resolution 或 Variable Signal 能力与持久化。
- Research formal publication/lineage 与 Research Analysis Context；Research 只保留
  `analyst_snapshot` 发布、读取与无状态图检索。

## Acquisition And Agent Boundary

Data 拥有 Source 配置与正式 Raw Evidence、Evidence 及发布合同；外部 Agent OS 只拥有
采集、Evidence 关键词生成、清洗和语义提取执行。AgentOS 在每次 Raw Collection Workflow
开始前通过 Data 版本化 API 读取一次完整 active Source 快照，之后为该 workflow
冻结。执行方也必须通过 Data API 发布结果，不直接访问 Data 数据库。Data 不反向调用、
不读取 AgentOS 数据库或本地 Artifact，也不执行 connector、parser、prompt、schedule
或 workflow。一次性迁移文件由操作员发布，不构成运行时 import 或数据库依赖。
外部 AgentOS 投影 Company 时同样只能读取 Data 版本化快照 API，不得直连
Data PostgreSQL；Data 不拥有由 AgentOS 产生的行业或产业链模型推断。

Event 直接通过 `event_evidence_links` 引用 Data-owned Atomic Evidence；轻量
`raw_documents`、`event_sources`、Event Tag 和旧 Event Publication 已退役。新 Event
写入必须在同一事务中包含至少一个 Atomic Evidence 关联。

## Language

**Source**:
Data-owned 的可执行采集入口配置，使用 `SRC + canonical lowercase UUID` 稳定身份
和全局唯一、不可变 `code`。fixed Source 不可删除，且 `code`/`ownership_type`/
`channel_type` 不可变；`adapter_key` 与其它运行字段可变。dynamic Source 只能为
`rss + generic_rss` 并可创建、启停、更新或删除。`app_key` 在当前 service-token
信任边界内以明文保存并返回。Source 不是历史 `source_catalogs`。
_Avoid_: Collection Channel 双写、credential_ref、Data 代理 Provider、adapter/channel 兼容性校验、source_catalogs

**Source Snapshot**:
每次 workflow 启动时的完整 active Source 集合，不分页，按 `channel_type`/`priority`/
`code`/`id` 稳定排序，整体不超过 500,000 bytes。空集合合法；读取、完整性或
容量校验失败时 fail closed，不返回部分项、不缓存回退。
_Avoid_: 分页、静默截断、丢弃单个非法 Source、workflow 中途刷新

**Domain Object ID**:
技术组件生成的与数据库无关身份，格式为“领域对象缩写前缀 + canonical lowercase
UUID”，中间不使用分隔符。Data Application 的受控注册表拥有全部对象前缀；普通写入由
owning Biz 生成，初始化发布调用同一生成器。请求不接收主键，Data 只持久化且数据库不生成。
可移植目录、关系和可重放事实基于受控自然键确定性生成 UUID，普通领域对象随机生成。
Data Service 管理的每张表都使用名为 `id` 的唯一主键；自然键或关系端点另作唯一约束。
_Avoid_: 裸 UUID、任意字符串前缀、`PREFIX_...`、`PREFIX-...`、调用方提交主键、数据库序列

**Event**:
经过外部提炼与同一现实动作判定后发布的标准化事件事实，包含 `EVT` 身份、
title、summary、严格十键业务语义 `semantic`（actors/action/objects/stage/modality/time/
jurisdictions/reason/method/metrics）和 ACTIVE/DEPRECATED/ARCHIVED lifecycle。time 包含可空
occurred/announced/effective UTC 时间与受控 precision，且至少一个时间锚点存在。Event 不复制
Evidence attribution；来源由 Evidence Link 保持。Data 不进行 Event 语义去重，也不接收 Event ID；
Reasoning 完成去重决定后才调用 Event 发布合同。
_Avoid_: 5W1H 作为 Event 身份、顶层 modality/occurred_at/announced_at 双写、Event attribution、Data 语义去重、调用方 Event ID、无证据 Event

**Event Evidence Link**:
Event 与 Atomic Evidence 的唯一当前证据关系，使用 `EEL` 身份、严格外键和 0.00–1.00 的独立
`contribution_weight`。同一 Event/Evidence 端点对只能出现一次，各权重不要求合计为 1。
_Avoid_: Artifact 引用、证据正文副本、Event Source 关系字段

**Event Publication Receipt**:
Reasoning 调用 Event 发布 API 时的最小传输幂等记录，使用 `EPR` 身份，按
`publisher_subject + publication_key` 唯一。同 key 且同 payload hash 返回原 Event，不新增
Event 或 Evidence Link；同 key 但不同 payload 冲突。Receipt 只解决 Data 成功但响应丢失的
安全重试，不是 Event 业务身份、语义去重键或异步任务。
_Avoid_: Event Dedupe Key、X-Request-ID 作为幂等键、发布重放创建新事实

**Event Actor Link / Event Asset Link**:
Event-owned 的预留关系快照。`EAC` 记录 Actor 的 opaque ID、可选类型/名称、关系类型、
强度与置信度；`EAS` 记录 Asset 的 opaque ID、可选类型/名称、影响方向与幅度。它们只外键到
Event，不证明 Actor/Asset 存在，也不定义其归属或生命周期。
_Avoid_: Actor/Asset entity、target 外键、lookup API、将快照当作主数据

**Storyline**:
持续演进的叙事主记录，以 `STL + canonical lowercase UUID` 为稳定身份，保存摘要、当前阶段、
生命周期、置信度与最新一次数据对账快照。三类蓝图 Storyline 严格决定唯一锚点：
`GEOPOLITICAL` 引用 GeopoliticRivalry，`MACRO` 引用 MacroEconomic，`INDUSTRY` 引用
IndustryChain；`CORPORATE` 当前不拥有蓝图锚点，也不引用 Company。Storyline 不属于
StorylineDomain，也不引用 StorylineDomainTactic；对账历史不由 Storyline 主记录保存。
_Avoid_: StorylineDomain 外键、Concept/Company 锚点、蓝图类型多个或缺失锚点、对账历史表

**Storyline Event Link**:
Storyline 与 Event 的唯一当前多对多关系，以 `SLE + canonical lowercase UUID` 为稳定身份。
同一 Storyline/Event 端点对只能出现一次，端点均使用 restrictive reference。Storyline 的首个
和最新 Event 时间不是独立事实，只能从已关联 Event 的非空 `occurred_at` 计算；不得用
`announced_at` 补值。
_Avoid_: `first_event_at`/`last_event_at` 副本、用 announced time 代替 occurred time、重复端点

**Organization**:
以 `ORG + canonical lowercase UUID` 为稳定身份的独立多边组织事实，覆盖联盟、协会、国际机制、贸易集团和
安全联盟。Organization 不使用通用 Entity、Profile 或旧 `alliance_org` UUID；Category、
Function 与 Domain Tag 通过 Data 目录连接稳定英文机器码和中文语义。
_Avoid_: Alliance Org、Organization Profile、通用 Entity 的组织别名、member_count

**Company**:
Entity 父领域下的独立公司事实，以 `COM + canonical lowercase UUID` 为稳定身份，以全局唯一、
不可变且非空的 `code` 为业务自然键。Company 直接保存名称、可空英文名和法定名称、别名、
经营区域、总部城市、成立/IPO 日期、法律形式、受控所有权类型、战略定位、说明、状态和时间戳；
未知新增事实使用 null，不以空字符串代替。Company 不使用 `entity_nodes`、Profile 或 shadow
Entity，只通过可空 `registration_country_id` restrictive 引用一个 Country。该关系优先表达已知注册国；
经业务明确批准的批量目录也可表达有来源、方法和置信度审计的推断企业所属国；常规 CRUD 不得自动猜测。Company 通过
Company–Industry Link 关联零个或多个正式 Industry。Company 不与 Storyline、Controller、
Security 或总部 Country 建立关系。
_Avoid_: Company Profile、Company shadow Entity、ticker 充当 code、自由文本 industry、
controller 字段、Storyline/Security link、headquarters_country_id、无审计的国家猜测、伪造未知属性

**Company–Industry Link**:
Company 的正式 Industry 分类关系，以 `CIL + canonical lowercase UUID` 为稳定身份；关系身份由
Company/Industry 端点确定性生成，同一端点对唯一，两个外键均 restrictive。完整 Industry
集合由 Company 聚合原子替换。
_Avoid_: `industry_name`、模糊行业匹配、重复端点、部分替换

**Company Projection Snapshot**:
外部 AgentOS 投影用的 Data-owned 只读快照合同，`schema_version` 固定为
`company-projection-snapshot.v1`，以 Company 与 formal CompanyIndustryLink 全集计算
lowercase SHA-256 `snapshot_id`。列表按 `(code, id)` 稳定分页；cursor 绑定快照，
事实漂移后必须以 409 fail closed 并从首页重启。`industry_links` 只是 formal Data
事实，不承载模型判断。
_Avoid_: AgentOS 直连 PostgreSQL、跨快照混合分页、部分结果、把推断写成 formal link

**Organization Category**:
Organization 唯一的可维护组织形态目录项，以 `OCA + canonical lowercase UUID`
为稳定身份，code 作唯一自然键。
_Avoid_: PostgreSQL enum、通用 Dictionary、带状态或展示顺序的分类

**Organization Function**:
以 `OFN + canonical lowercase UUID` 为稳定身份的 Organization 唯一可维护核心职能目录项，
code 作唯一自然键；Domain Tag 必须归属于一个 Function。
_Avoid_: 多选核心职能、自由文本职能、与 Domain Tag 混用

**Organization Domain Tag**:
以 `ODT + canonical lowercase UUID` 为稳定身份，在 Organization Function 下表达
细化投资主题语义的可维护目录项。Organization 可以选择
多个 Tag，但每个所选 Tag 的归属 Function 必须与 Organization 当前 Function 一致。
_Avoid_: 文本数组、跨 Function 标签、无中文语义的代码

**Organization Domain Tag Link**:
以 `ODL + canonical lowercase UUID` 为稳定身份的 Organization 与 Domain Tag 关系；
Organization 与 Domain Tag 端点组合另作唯一约束。

**Organization Membership**:
Country 与 Organization 之间带成员类型和生效历史的关系事实。有效期为闭区间，空起始
表示未知且向过去无界，空失效表示当前有效且向未来无界；同一 Country 与 Organization
的历史区间不得重叠，失效日期修正更新原行。
_Avoid_: member_count、重叠历史、用追加行表达 expiry_date 修正

**Raw Evidence**:
Data 正式保存的一份完整原始采集材料，以 `RAW + canonical lowercase UUID` 为 `id`，
包含来源与转载快照、完整正文、文章发布时间、
采集时间、正文哈希和受控内容分类。它可以在清洗完成前暂时没有 Evidence，但不能以
零 Evidence 作为正式清洗结果。
Data 还为新建 Raw Evidence 保存数据库生成的内部 `created_at`；发布方不提交，发布 API
不返回，历史行不回填。
_Avoid_: Event Evidence Record、外部执行 Artifact、Raw Document、只含摘录的证据链接

**Raw Evidence Content Category**:
描述一份完整 Raw Evidence 的内容形态或编辑目的的受控分类；同一材料可以拥有多个分类，
分类不表达 Atomic Evidence 的事实、预测、意图、推断或观点性质。
_Avoid_: Atomic Evidence Claim Type、自由文本标签、OpenSPG Object

**Evidence Category Catalog**:
Data PostgreSQL 拥有的唯一当前 Raw Evidence Content Category 目录，包含正式 Evidence
Category ID、稳定机器 code、中文名称和完整分类描述。AgentOS 在 Evidence Extraction 前通过
Data 只读合同取得全部目录，用模型输出的 code 确定性映射正式 ID，再通过 Raw Evidence
Publication 的 `category_ids` 发布；Data 继续根据 PostgreSQL 当前目录校验最终 ID 并创建
Raw Evidence Category Link。目录读取原样返回当前名称和描述，不提供 active、置信度、主分类、
展示顺序、版本、revision 或内容 hash，也不允许调用方复制或修改目录事实。
_Avoid_: AgentOS 自建或持久化 Catalog 副本、Prompt/YAML 中复制分类 ID、模糊名称映射、通过读取 API 修改或补写目录

**Raw Evidence Category Link**:
以 `RCL + canonical lowercase UUID` 为 `id` 的 Raw Evidence 与受控 Content Category 无顺序
事实关系；同一分类在一份材料中
只能出现一次。
_Avoid_: Primary Category、分类置信度、Atomic Evidence Category

**Atomic Evidence**:
清洗流程从一个 Raw Evidence 得到的、可直接消费的一条最小完整业务命题，以
`EVD + canonical lowercase UUID` 为 `id`。每条 Atomic Evidence 包含一条非空简洁事实摘要
`summary`、一至五个有序中文 `keywords`，以及由主体、动作、对象、阶段、情态、时间、辖区、
原因、执行方式、指标和归因组成的结构化 `semantic`。它按业务命题聚合而不是按句子、单个三元组
或单项指标机械拆分：媒体归因不是业务主体，同一披露中的同类经营指标可以同属一条 Evidence，
已经发生的结果与未来指引可以拆分。一个 Raw Evidence 正式清洗后必须拥有一至多条 Atomic
Evidence；`1:1` 的 `is_split` 为 false，`1:N` 的每条 `is_split` 均为 true。
Data 为新建 Atomic Evidence 保存数据库生成的内部 `created_at`；发布方不提交，发布 API
不返回，历史行不回填。Evidence 不可变，因此没有 `updated_at`。
_Avoid_: Event Evidence Link、完整 Raw Evidence、Evidence Group、Event

**Atomic Evidence Identity**:
Data 根据所属 Raw Evidence 身份与 Atomic Evidence 的 `summary + keywords + semantic` 规范内容确定性
派生正式 ID；调用方不提交 ID。该身份只保证同一 Raw Evidence 完整集合的安全重试，不执行
跨 Raw Evidence 的语义去重，不把相同语义的多来源 Evidence 合并为 Group。
_Avoid_: 调用方 Evidence ID、Expression Key、Evidence Group、Data 语义召回、embedding

**Raw Evidence Publication**:
采集完成后把一份完整 Raw Evidence 及可选 Content Category 集合原子接纳为正式 Data 事实的同步发布。
相同身份的全部内容及无序分类集合一致时允许安全重试，内容或分类漂移时冲突；成功响应只返回正式 Raw Evidence ID，
不创建发布回执或返回创建/复用分类；响应以 `id` 字段返回该身份。
_Avoid_: Evidence Publication、异步 Import Job、Idempotency-Key

**Evidence Publication**:
清洗完成后，为一个既有 Raw Evidence 一次提交完整 `1..N` Atomic Evidence 集合的同步
发布。整包只能首次创建或以完全一致内容安全重试，不能覆盖、追加、删除或发布零项；成功
响应返回外键 `raw_evidence_id`、按正式 ID 确定性排序的兼容字段 `ids`，以及按本次请求数组
位置返回的 `items[{input_index,id}]`。`ids` 排序没有业务语义；`items` 才是发布方把当前每条
输入 Evidence 可靠关联到正式 ID 的权威映射，完全一致或顺序变化的安全重试都按本次请求
重建该映射。发布不创建回执，也不返回创建/复用分类。
_Avoid_: Raw Evidence Publication、Group Publication、部分成功、可变清洗结果

**Data Runtime Health**:
Data Service 对自身既有运行状态的只读即时投影，不探测或代理 PostgreSQL 之外的
Entity projection storage。它不改变既有 `/healthz` 或 `/readyz` 合同。
_Avoid_: Neo4j/Qdrant 健康代理、读取业务事实、把健康检查结果持久化或用于自动修复

**产业链（Industry Chain）**:
围绕明确目标产出与终端用途，由多个独立 ChainNode 通过投入、组成、技术支撑或依赖形成的
有边界、有方向研究子图。IndustryChain 以 `ICH + canonical lowercase UUID` 为稳定身份，
直接拥有名称、别名、范围、目标产出、
终端用途、地域、截至日期、审核、技术路线、可观察变量和审计时间，不使用 `entity_nodes`
或 definition/profile 表。
_Avoid_: Industry、Concept、Chain Node 列表、IndustryChain shadow Entity、definition profile

**产业链–行业关系（Industry Chain–Industry Link）**:
IndustryChain 当前关联的正式 Industry 分类集合。关系沿用 `ERL + canonical lowercase UUID`
作为 Research Graph 稳定身份，只保存 IndustryChain 端点、Industry 端点和创建时间；同一端点
对只能存在一次，两个端点均为 restrictive reference。表的所有权直接决定
`mapped_to_industry` 语义，关系存在即表示当前映射。
_Avoid_: 通用 `entity_edges`、relation type 字段、status、mapping note、source、verified_at、updated_at

**产业链–概念关系（Industry Chain–Concept Link）**:
IndustryChain 当前关联的正式 Concept 集合。关系沿用 `ERL + canonical lowercase UUID`
作为 Research Graph 稳定身份，只保存 IndustryChain 端点、Concept 端点和创建时间；同一端点
对只能存在一次，两个端点均为 restrictive reference。表的所有权直接决定
`mapped_to_concept` 语义，关系存在即表示当前映射。
_Avoid_: 通用 `entity_edges`、relation type 字段、status、mapping note、source、verified_at、updated_at

**产业链节点（ChainNode）**:
产业链中可复用的独立环节事实，以 `CND + canonical lowercase UUID` 为稳定身份，直接拥有
名称、别名、定义、审核状态和审计时间。
ChainNode 不使用 `entity_nodes` 或 profile 表，也不保存边界备注；上中下游位置只属于
Industry Chain Node Membership。
_Avoid_: ChainNode Profile、ChainNode shadow Entity、`boundary_note`、节点全局上下游标签

**行业（Industry）**:
受控分类体系中的独立行业事实，以 `IND + canonical lowercase UUID` 为稳定身份，直接拥有
名称、别名、分类体系、行业代码、父级、
层级路径、定义和审核状态。Industry 不使用 `entity_nodes` 或 profile 表；根行业没有父级，
分类深度由层级路径推导，不保存分类版本、分类层级或边界备注。
_Avoid_: Industry Profile、Industry shadow Entity、持久化 classification level、从名称推测层级

**概念（Concept）**:
跨行业表达技术、政策、应用、需求、商业模式、企业/产品生态、事件叙事或市场主题的独立
事实，以 `CON + canonical lowercase UUID` 为稳定身份，直接拥有名称、别名、Concept Type、
定义和审核状态。Concept 不使用
`entity_nodes` 或 profile 表，也不保存边界备注。
_Avoid_: Concept Profile、Concept shadow Entity、把 Concept 当作 Industry 分类层级

**产业链节点归属（Industry Chain Node Membership）**:
一个 Chain Node 被纳入某一特定 Industry Chain 的当前正式上下文关系；行存在即表示当前归属。
只保存 IndustryChain/ChainNode 端点、上中下游阶段、位置和审计时间；阶段和位置不是节点的
全局属性。
_Avoid_: 节点全局上下游标签、节点之间的图谱边、review/status、inclusion reason、Evidence、Source、verified at

**产业链图谱边（Industry Chain Graph Edge）**:
同一 Industry Chain 的两个成员节点之间带明确方向和受控 relation type 的当前正式关系；行
存在即表示当前拓扑。这是 Data 当前唯一节点间拓扑事实，只保存身份、IndustryChain、两个
成员端点、relation type 和审计时间，并保持无环。
_Avoid_: `chain_node_relations`、机制/条件/压缩段解释、review/status、Evidence、Source、verified at、发现映射、关键词相关、单次 Research Anchor 的临时传导路径

**已退役：Chain Node Relation**:
历史上脱离 IndustryChain 上下文保存的全局 ChainNode 关系。它不能确定性转换为要求同链成员
端点的 Industry Chain Graph Edge，因此不再属于当前事实模型。
_Avoid_: 与 Industry Chain Graph Edge 双写、把全局关系猜测为某条产业链拓扑

**已退役：Chain Node Physical Constraint**:
历史上附着于 ChainNode 或 Chain Node Relation 的约束记录。当前 ChainNode、Membership 与
Industry Chain Graph Edge 均不拥有该事实。
_Avoid_: 无 owner 的约束表、通过旧关系恢复约束生命周期

**已退役：Industry Relationship Import Receipt**:
历史 Industry relationship package importer 的不可变执行回执。Importer 已不属于 Data runtime，
当前 typed IndustryChain Links 与 Graph Edge 不创建或依赖该回执。
_Avoid_: 把历史执行审计当作当前关系事实、为已退役 importer 保留 receipt schema

**实体关系类型（Entity Relation Type）**:
对实体关系语义、允许端点、固定方向和遍历含义的规范谓词；正式关系只能使用已批准的稳定 code。
_Avoid_: AI 自由关系字符串、无语义的 related_to

**已退役：Entity External Identifier**:
历史上把 Eastmoney、同花顺等外部分类代码映射到通用或独立 Data Object 的通用标识记录。
当前 Data 不拥有该映射的 API、导入流程或生命周期，也不把历史行转换为 Company 标识。
_Avoid_: 通用 `entity_external_identifiers`、从旧 Vendor code 猜测新领域映射

**已退役：Entity Redirect**:
历史上表达 Data Object merge 或 reclassification 的通用有向重定向。当前对象身份与正式关系
不依赖 Redirect，历史行不转换为 aliases 或 Entity Relation。
_Avoid_: 通用 `entity_redirects`、隐式 ID canonicalization、把 Redirect 当作正式业务关系

**产品实体（Product Entity）**:
企业生产、销售、采购或被市场需求的可识别产品对象。Product 不等同于表示经济环节或
活动的 Chain Node，也不等同于标准化可交易的 Commodity。
_Avoid_: Chain Node、Commodity、Technology、产品名称 Mention

**Legacy Publisher Artifact**:
既有 Event Publication 链路中由历史发布方采集执行生成的不可变原始文档对象。Data 不读取
该遗留执行方的数据库或 Artifact 存储位置；新的完整原始材料通过 Raw Evidence
Publication 进入 Data，并不复用该遗留对象。
_Avoid_: Data Raw Document、Event Evidence Record、Data 原始语料

> 以下“已退役”条目仅保留历史语义，不是当前代码、schema 或 API 合同。

**已退役：Event Evidence Record**:
Data 仅在正式 Event 引用了外部 Artifact 时接纳的轻量证据文档记录，保存 Artifact 身份、内容 SHA-256、稳定 `source_ref`、来源快照和必要时间元数据，不保存完整正文或 Artifact 存储位置。`source_ref` 只是无外键的外部来源引用，Data 不维护其 Source 主数据。内容 SHA-256 只用于检测同一 Artifact 身份是否发生内容漂移，不表示 Data 已读取原文或验证来源真实性。来源快照可保留公开 `source_url` 用于证据归因，允许没有公开地址的来源为空；该地址不是外部 Artifact 的内部位置，Data 不主动访问或校验。一个记录可以支持多个 Event，一个 Event 也可以引用多个记录。
V3 接纳字段只包含必填的 `artifact_id`、`content_sha256`、`source_ref`、`source_name`、`source_type`、`title`、`collected_at`，以及可选的 `source_url`、`published_at`、`language`、`mime_type`。`content_text`、Artifact URI、采集通道、采集状态、内容层级和独立来源外部 ID 不属于 V3 合同。
`source_type` 是由外部发布方治理的非空快照字符串，Data 只校验非空和长度，不维护对应枚举或主数据。
_Avoid_: 完整 Raw Document、采集缓存、未产生 Event 的文档

**已退役：Artifact-based Event Evidence Link**:
一个正式 Event 与 Event Evidence Record 之间的语义关联，必须包含 `artifact_id`、短而非空的 `evidence_statement`、`evidence_relation` 和 `source_level`。`evidence_statement` 是模型生成并由审核阶段确认的证据陈述，不承诺逐字摘录；程序仅通过 Artifact 身份和内容哈希建立正式血缘，不用原文字符串匹配裁决语义。`evidence_relation` 仅允许 `supports`、`contradicts`、`context`；前两类必须提交非空 `supports_fields`，`context` 可为空。`supports_fields` 仅允许 `title`、`factual_summary`、`occurred_at`、`fact_payload`。`source_level` 仅允许 `primary`、`secondary`，仅表示来源层级，不表示某条 Evidence 在 Event 内拥有主证据地位。Data 根据证据陈述计算 `evidence_hash`，不接收调用方计算的证据哈希。
同一 Artifact 在同一 Event 中只能出现一次，数据库按 `(event_id, raw_document_id)` 保证唯一；一个 Link 可通过 `supports_fields` 覆盖多个字段。再次提交已有 Link 时，关系、证据陈述、支持字段和来源层级必须全部一致，否则整批冲突。Event 不再保存或暴露主证据引用，所有 Evidence Link 具有平等的正式血缘地位。
_Avoid_: 完整正文副本、无语义 Artifact 引用、真实性认证结果

**已退役：Event Tag Assignment**:
正式 Event 的受控 Tag 映射。每个 Event 必须包含一至两个 active `news_category`，并可包含零至三个 active `index_category`；每项提交匹配的 Tag ID、kind、code，以及 `confidence`、非空 `assignment_reason` 和 `ai` 或 `rule` 来源。V3 不接收 Tag review status，Data 统一写为 `approved`。已有同 Tag 映射仅在内容一致时复用，新映射可以追加，冲突时整批失败。
_Avoid_: 待审核 Tag、未知或停用 Tag、静默覆盖已有分配依据

**已退役：Event Tag Catalog**:
Data PostgreSQL 拥有的唯一当前 Event 分类主数据集合，包含稳定 Tag ID、kind、code、名称和
启停状态。V1 不提供 Catalog 历史、版本、revision 或内容 hash；外部发布方每次分类通过 Data
只读合同取得当前 active Tag，校验 wire 字段、受控 kind、稳定排序和重复身份后使用。Event
Publication 仍由 Data 根据 PostgreSQL 当前 Tag 校验 ID、kind、code 和 active 状态。
_Avoid_: 外部发布方自建或持久化 Tag Catalog 副本、Catalog 版本身份、对 JSON 字节重复计算 hash、在 Prompt 或 YAML 中复制 Tag ID、模型创造 Tag

**已退役：Event Publication Batch**:
外部发布方将一至十个已完成提取与审核、状态固定为 `confirmed + verified` 的原子 Event，连同其共享 Event Evidence Record、证据关联、Tag、Review 和提取血缘，按照 Data 定义的严格同步合同整批原子提交为正式事实；候选、未验证或拒绝 Event 不进入 Data，任一成员失败时整批不可见。
每个 Event 独立提交必填的 `review_id`、`evidence_grade` 和非空 `reasons`；V3 不重复提交审核决定、Event/Fact 状态或组件版本，Data 统一写入 `confirmed + verified`。
V3 在批次顶层提交去重后的 `raw_documents`，各 Event 通过 `artifact_id` 引用共享证据。每个 Event 至少引用一个已声明 Artifact；每个顶层 Artifact 也必须至少被一个 Event 引用，未知或重复 Artifact 身份均使整批失败。
Data 在写事务前返回所有当前可确定的合同、枚举、Tag 和引用错误；自然身份内容冲突单独返回冲突错误。任一错误均阻止整个批次和 Receipt 落库，不允许部分成功。
_Avoid_: 独立 Raw Document 导入、Agent 直写数据库、先存全文后补 Event

**已退役：Event Import Receipt**:
Data 为每次成功 Event Publication Batch 生成的不可变审计凭证，记录调用主体、`package_id`、正式事实身份、`extractor_execution_id`、`extractor_agent_version`、每个 Artifact 对应的 `collector_execution_id` 和导入时间。以上执行血缘均为必填；Prompt、模型和 Profile 版本由外部发布方保存。Receipt 不承担请求幂等、重放判断或异步状态查询职责；失败事务不生成 Receipt。
`package_id` 只是外部发布方提供的审计关联编号，不唯一且不参与事实复用；相同 package 可以产生多个成功 Receipt，每次成功调用均由 Data 生成新的 `receipt_id`。
Event Publication 必须通过 Data 唯一的内部 Bearer service token 鉴权；Token 只存在于运行环境，不进入数据库。Data 将该凭据解析为稳定的 Data 内部 trust-domain `caller_subject` 写入 Receipt，不区分具体消费者。Event 的消费者级审计由必填 Collector/Extractor 执行血缘承担，与 Source、采集通道或 Artifact 来源无关。本期明确不提供 Data API 的逐消费者 token、scope 隔离或逐消费者 Receipt 主体。
V3 Receipt 存储在专用 `event_publication_receipts`。旧独立 Raw Document 导入和单 Event V1 导入退出后，其 `raw_document_import_receipts`、`event_import_receipts` 及专属数据库触发器/函数连同历史审计记录物理移除；该清理不得删除正式 Event、Event Evidence Record 或 Event Evidence Link。
每次成功调用均创建 Receipt 并返回 `201 Created`，响应包含 `receipt_id`、`package_id`、`imported_at`、Dedupe Key 到 Event ID 的 created/reused 映射、Artifact ID 到 Raw Document ID 的 created/reused 映射，以及 Event、Raw Document、Event Source、Event Tag 的 created/reused 分类计数；不返回 payload hash、replayed 或异步任务状态。
_Avoid_: Idempotency Record、Import Job、失败占位记录

**已退役：Event Dedupe Key**:
外部发布方为一个原子 Event 提交的稳定唯一业务身份，对应 Data 中唯一的 Event 事实；Data 的 Event UUID 是独立数据库身份。相同 Dedupe Key 不得对应不同核心事实，事实修订必须使用新的 Dedupe Key。
_Avoid_: Event UUID、Import Idempotency Key、可覆盖的事件名称

**已退役：Event 事实收敛（Event Fact Convergence）**:
相同 Event Dedupe Key 的 `title`、`factual_summary`、可空 `occurred_at` 和按 JSONB 语义比较的 `fact_payload` 必须完全一致，Data 复用已有 Event；任一核心字段修订必须使用新的 Dedupe Key。`first_seen_at` 与 `knowable_at` 不由调用方提交，由 Data 根据全部关联证据计算，并且后续只能随新增的更早证据向更早时间收敛。后续 Publication Batch 可以为该 Event 新增证据或 Tag 关联；已有且语义一致的关联直接复用，已有关系不得被静默改写或删除，冲突时整批失败。每次成功调用仍生成独立 Import Receipt。
`occurred_at = null` 表示证据不足以确定事件发生时间，不影响已确认、已验证且 Evidence 合同
有效的 Event 进入下游读取；Data 不用发布时间、采集时间或首次发现时间补写该字段。
_Avoid_: 覆盖 Event 核心事实、删除旧证据、用新 Receipt 表示新 Event

**研究主题（Research Theme）**:
一次完成分析侧校验并由授权发布主体提交的单 Theme Aggregate 内，对一组 Event 及其
分析对象影响形成的不可变、可发布研究判断快照，包含一句话结论、传导路径和结论演进
阶段。唯一当前发布模式 `analyst_snapshot` 的对象、变量和传导是分析师报告快照，
不是本体事实。同一现实议题在不同 Aggregate 中生成不同 Research
Theme；首页展示查询时间范围内全部成功发布的 Theme Aggregate，并按发布时间稳定分页。
已发布内容的纠错必须由分析侧使用新的 `analysis_batch_id` 发布完整修正 Aggregate，旧 Aggregate 保留审计。本期不提供更新、删除或撤回 Theme/Aggregate 的 API。
_Avoid_: 覆盖或删除历史 Theme、把 `theme_key` 当作跨 Aggregate 稳定身份、原地修订已发布 Aggregate

**聚合内主题键（Theme Key）**:
分析侧为单个 Research Theme 提供的 Aggregate 内稳定键，与 `analysis_batch_id` 共同用于确定性 Theme 身份、回执映射和错误定位。合法键长度为 1 至 128，只允许小写 ASCII 字母、数字及 `.`、`_`、`:`、`-`，不接受服务端规范化。`theme:` 前缀推荐但不强制。它不跨 Aggregate 合并主题，也不等同于未来 Research Thesis 身份；新 Aggregate 即使复用同一 Theme Key，也产生不同 Research Theme。
_Avoid_: 调用方提交 Theme UUID、把 Theme Key 当作长期主题身份

**长期研究命题（Research Thesis）**:
未来用于跨批次持续跟踪同一研究议题或产业瓶颈的独立对象。Research Theme 不承担该职责。

**研究主题发布聚合（Research Theme Publication Aggregate）**:
一次同步请求只发布一条 Theme 及其 1..N 棵完整 Reason Tree；二者是最小事务聚合，
要么全部可见，要么全部回滚。一次 Codex 分析产生多个 Theme 时分别提交，彼此不共享
事务。零 Theme 是 Codex 合法结果，不向 Data 提交占位对象。Data 按 `analyst_snapshot`
校验 DTO、结构、来源引用和事务，只校验 local key 闭包、路径、正式 Event 及调用方可选提交的
Evidence 归属。Data 不判断投研结论是否正确，也不把分析师快照晋升为正式本体事实。
_Avoid_: Theme 先可见再补 Tree、独立 Tree 写入口、部分入库、零 Theme 占位

**主题聚合发布时间（Theme Aggregate Published At）**:
一条完整 Research Theme Publication Aggregate 正式成为产品可见事实的服务端时间。Data 在该单 Theme 及其全部 Reason Tree 校验通过并提交事务时统一生成；失败发布不产生发布时间，幂等重放保留首次成功发布的时间。首页在调用方指定的查询时间范围内返回全部成功发布的 Aggregate，按 `published_at DESC, id ASC` 稳定排序并通过游标分页；范围内没有 Aggregate 时返回空集合。
_Avoid_: 调用方指定发布时间、只返回范围内最新 Aggregate、重放时刷新发布时间

**分析批次身份（Analysis Batch ID）**:
工程外部 Codex 为一条 Theme Publication Aggregate 提供的稳定、不可变发布身份，仅承担
幂等和审计关联，不表示 Data 拥有 Codex Run 或任务。同一 Analysis Batch ID 和相同
canonical 聚合属于幂等重放；同一身份对应不同内容返回冲突。校验失败不占用该身份；
新结论必须使用新身份创建新 Theme，不更新旧 Theme。
_Avoid_: 第二套幂等键、覆盖已发布批次、失败校验生成成功 receipt

**分析窗口（Analysis Window）**:
一条 Theme Aggregate 实际查询 Event 的知识可得时间范围，由
`discovery_window_start`（含）和 `discovery_window_end`（不含）表达，并固定
`analysis_as_of`。三者使用 UTC，结束时间必须晚于开始时间且不得晚于
`analysis_as_of`；它们与服务端发布时间相互独立。
_Avoid_: 零长度窗口、为 Tree 另设窗口、用发布时间替代分析窗口

**研究主题发布主体（Research Theme Publisher Subject）**:
Data 内唯一内部 service token 对应的稳定 trust-domain 身份，由 Data 从认证上下文解析，不由请求声明。首次成功发布时该主体取得 Aggregate 所有权；后续幂等重放必须来自同一 trust-domain 主体。主体身份独立于可轮换 token，审计只保存主体 ID，不保存凭据。本期不以调用方服务区分 Theme 所有权，所有持有 Data service token 的受信内部消费者处于同一发布信任域。
_Avoid_: 在请求体中声明发布者、以 token 字符串作为长期身份、把同一 trust-domain 内的消费者伪装成独立鉴权主体

**Theme 发布未知结果恢复（Theme Publication Unknown-outcome Recovery）**:
同步 Theme 发布请求超时后，发布器以完全相同的 Analysis Batch ID 和发布内容重试 POST。首次事务已成功时返回原结果并标记重放，未成功时正常执行，内容变化时返回冲突。本期不提供状态查询、轮询或异步任务接口。

**Theme 发布回执（Theme Publication Receipt）**:
一个成功 Theme Aggregate 的不可变技术回执，持久化 Analysis Batch ID、发布主体、
payload hash、唯一 Theme ID、publication contract/mode、Tree identity 到 Tree ID 的映射、
首次发布时间及整棵聚合的写入计数。`analyst_snapshot` 使用 aggregate-local `tree_key`
映射。Theme、全部 Trees、来源关联和
两类既有回执在同一事务内提交。
首次成功返回 `201 + replayed:false`，相同主体和载荷重放返回
`200 + replayed:true`；重放不得刷新结果。
_Avoid_: 在单条 Theme 上设置批次 ID 唯一约束、业务数据失败后保留回执、修改成功回执

**Theme 发布载荷哈希（Theme Publication Payload Hash）**:
对完整 Theme 发布请求体按 RFC 8785 规范化后计算的小写十六进制 SHA-256，用于批次幂等重放和内容冲突检测。哈希只覆盖调用方提交的批次身份、分析窗口和 Theme 内容，不包含认证信息、请求 ID、服务端发布时间或响应字段；由 Data 计算并返回，调用方不提交。
`analyst_snapshot` 计算前仅将无展示顺序的 `Theme.events` 按 `event_id` 规范化；
其余数组顺序仍属于载荷内容。

**Theme 发布规范数组顺序（Theme Publication Canonical Array Order）**:
Theme 发布请求只有一个 `theme`。`analyst_snapshot` 除 `Theme.events` 外的数组均使用合同
规定的唯一稳定顺序。`Theme.events` 是无展示顺序的唯一关联集合，调用方数组顺序不
构成发布门禁；Data 只在计算 hash 的副本中按 `event_id` 规范化，不改变请求、持久化或
Tree Event 展示顺序。UUID 必须使用标准小写字符串，同一作用域不得重复键或 ID。
_Avoid_: 大小写混合 UUID、重复关联、重排有展示语义的数组、将 V3 Theme Event 的 caller 顺序误作校验门禁

**Theme Aggregate Analyst Snapshot 发布 V3（Theme Aggregate Analyst Snapshot Publication V3）**:
`POST /api/data/v1/research-theme-imports` 顶层以
`publication_mode=analyst_snapshot` 明确判别。它发布分析师拥有的不可变报告快照：Theme
Impact、Tree、Node 和 Signal 使用 aggregate-local key 与 publication-time display snapshot，
不提交也不依赖 Entity、IndustryChain、Variable Definition、Variable Signal、Direct Impact、
Relation 或 GraphEdge formal ID。每个 Theme 与每棵 Tree 至少关联一个正式 Event；
Evidence ID 可选，提供时才校验存在且属于所声明 Event。JSON/schema 错误返回 `400`，
local-key closure、路径、Event/Evidence 等业务结构或引用错误返回 `422`。
_Avoid_: 根据 formal ID 是否存在猜测分支、将本体覆盖作为发布门禁、从 display 文案反推
formal identity、发布时晋升本体事实

**主题影响（Theme Impact）**:
Research Theme 的不可变关注对象快照，保存角色、方向、可空摘要和稳定展示顺序。所有
Impact 平等，不存在 subject、primary 或主要影响节点；一个 Theme 至少有一个 Impact。
`analyst_snapshot` Impact 使用 aggregate-local
`node_key + display_name`，只要求该 key 在至少一棵 Tree 中被覆盖。Theme 卡片短名称与
Tree Node 具体名称是独立 presentation snapshot，Data 不比较二者文案。
_Avoid_: `is_primary`、把展示顺序解释为影响优先级、要求 V3 分析对象先成为正式 Entity

**产品可见 Theme（Product-visible Research Theme）**:
属于一次成功 Theme Aggregate 事务且满足读取窗口的 Theme。新 `analyst_snapshot` Theme
必然同时拥有至少一棵完整 Tree。退役的 formal 历史行由 retirement migration 删除，
不转换为 snapshot。
_Avoid_: 新 Theme 无 Tree 可见、读取时动态补写 Tree

**推理树（Reason Tree）**:
一个 Theme 的不可变线性推导链路。`analyst_snapshot` Tree 使用 aggregate-local
`tree_key + display_name`，display name 可以是产业链、宏观、估值或商业模式传导路径，
不要求正式 Industry Chain。它可以解释一个或多个 Theme Impact，也可以包含
只承担传导上下文的节点。Tree 没有中心节点或主要影响节点。
_Avoid_: Research Anchor、中心节点、任意图、分叉、循环、用 Tree `display_order`
表达影响优先级

**Reason Tree 发布集合（Reason Tree Publication Set）**:
Reason Trees 只能通过 `POST /api/data/v1/research-theme-imports` 随所属 Theme
Aggregate 同步发布，不存在独立写入口。请求至少包含一棵 Tree，全部 Tree 与 Theme
共同使用 `analysis_batch_id` 幂等身份和同一事务。
_Avoid_: `research-reasoning-tree-imports` 写入口、先 Theme 后 Tree、第二套幂等键

**Reason Tree 身份与回执（Reason Tree Identity and Receipt）**:
`analyst_snapshot` Tree 身份由 `theme_id + NUL + tree_key` 确定性生成。每个 Theme
最多一条 Tree 集合回执，保存发布主体、payload hash、Tree identity 映射、
写入计数和首次发布时间；回执与全部 Tree 子记录同事务提交。
_Avoid_: 调用方提交 Tree ID、原地覆盖、每棵 Tree 单独回执

**Reason Tree 节点（Reason Tree Node）**:
Tree 的有序分析对象快照。`position` 从 1 连续排列并且是唯一路径顺序；一个节点在
Tree 内唯一。`analyst_snapshot` Node 使用 Tree-local `node_key + display_name`，不要求正式
Entity、ChainNode 或 membership。首节点全部 `incoming_*` 为空；后续节点必须保存传导机制，
允许标题和成立条件为空。
_Avoid_: 持久化结果节点标记、首节点伪造入边、从 Tree 回写正式图谱

**Variable Signal 展示快照（Variable Signal Display Snapshot）**:
每个 Tree 节点拥有 1..5 个按 `display_order` 排列的不可变显示快照，恰好一个
`primary`。`analyst_snapshot` Signal 使用 Node-local `signal_key` 和必填 `display_summary`，变量名与
方向可空，不提交或依赖正式 Variable/Signal/Lineage ID。读取时只返回发布时显示快照，
不用当前正式事实动态重写文案。
_Avoid_: 从显示文本反推正式事实、Inference 伪造 Signal ID、要求 V3 Signal 绑定本体

**Object Schema**:
Data Service 工程 `doctype/` 中每个 Object Type 独立维护的 OpenSPG Schema Mark
Language 文件，定义对象语义、属性和 OpenSPG 可表达的约束。它不是 PostgreSQL
行事实，也不由通用 `entity_type_definitions` 表持久化。SQL migration 仍独立表达
主键、唯一性、长度、默认值与 PostgreSQL 类型，并通过合同测试防止两侧漂移。
_Avoid_: JSON Schema、数据库通用类型定义表、未经项目规范批准的 OpenSPG 语法

**Region**:
一个独立区域事实，由稳定 `REG + canonical lowercase UUID` 标识、中英文名称、形成或使用类型、
可选说明和数据库生成创建时间构成。`region_type` 只取 `CONTINENT | GEOGRAPHIC |
MULTILATERAL | INVESTMENT`，由 PostgreSQL 原生 enum 保证。Region 不使用 `entity_nodes`
主表或 profile 表。
_Avoid_: `region_profiles`、`region_types` 字典表、把 Region 写回旧 Entity 聚合模式

**Country**:
一个拥有 ISO 3166-1 alpha-2 代码的国家或地区独立事实，由稳定
`COU + canonical lowercase UUID` 标识、中英文
简称、可选战略定位、可选关键资源和数据库生成时间构成。Country 通过显式多对多关系属于
零个或多个 Region；它不使用 `entity_nodes` 或 profile 表，语义消费者将其类型识别为
`country`。
_Avoid_: Economy、全球范围、超国家组织、Region、Country Profile、Country shadow Entity

**Country–Region Link**:
Country 被纳入一个 Region 的正式集合关系；其含义继承目标 Region 的形成或使用类型，
不复制 `region_type`。完整关系集合由 Country 聚合原子替换，重复 Country–Region 对只保留
一个事实。
_Avoid_: 自由文本区域、单值 Region 字段、把关系类型写在 Country 行中

**Subdivision（行政区域）**:
严格从属于一个 Country 的独立 ISO 3166-2 行政区域事实，以
`SUB + canonical lowercase UUID` 为稳定身份。`code` 只保存全大写字母或数字组成的
ISO 3166-2 本地部分，并在所属 Country 内唯一；完整展示码由 Country alpha-2 code 与
本地 code 组合。类型只允许 `PROVINCE | STATE | SAR | TERRITORY`。Subdivision 拥有中英文
名称、可选战略定位、可选关键资源和数据库生成时间，不使用通用 Entity/Profile。
香港、澳门可以同时作为 ISO 3166-1 Country 与中国下的 Subdivision 存在，两类对象语义
不同，不建立继承、alias 或自动 crosswalk。Subdivision 与 Region 也不建立直接关系。
Organization 的 `headquarters_subdivision_id` 继续只是预留文本，不因此获得外键、格式校验、
查询、关系语义或 API 合同。
_Avoid_: Region、Subdivision Profile、Subdivision shadow Entity、全局唯一 local code、`SUB_` 身份、Country/Subdivision alias、Subdivision–Region crosswalk

**Ministry（政府部门）**:
一个严格归属于 Country 或 Organization 之一的独立政府部门与监管机构事实，以
`MIN + canonical lowercase UUID` 为稳定身份。`code` 是在 Ministry 内唯一的人工业务编码，
不与 Country ISO code 耦合。Country 所属行不是超国家对象，Organization 所属行是超国家
对象；数据库以 XOR、限制删除外键与 `is_supranational` 一致性约束保证该归属。Ministry
保存中英文名称、受控机构层级、三个明确权力布尔值、可空管辖范围、人工领域标签、可选说明
和数据库生成时间。一个 Ministry 可以可选引用一个直接上级 Ministry；当前只保证外键存在，
不判断跨归属、环路、深度或层级业务规则。
_Avoid_: PublicAuthority、Ministry Profile、Ministry shadow Entity、Country ISO code 派生、调用方 ID、Subdivision 关系、层级推断

**Institution（金融机构）**:
一个严格归属于 Country 或 Organization 之一的独立金融机构事实，以
`INS + canonical lowercase UUID` 为稳定身份。`code` 是在 Institution 内唯一的人工业务
编码，不与 Country ISO code 耦合。归属与 `is_supranational` 使用和 Ministry 相同的数据库
XOR 与限制删除合同。Institution 保存中英文名称、受控机构类型、可空清算货币、SWIFT BIC、
LEI、受控系统重要性、可选说明和数据库生成时间；系统重要性空值表示未知或未评估，
`NON_SIB` 表示已经评估为非系统重要性。Institution 没有父子关系，也不与 Ministry、
Subdivision 或 Region 建立关系。
_Avoid_: FinancialInstitution、Institution Profile、Institution shadow Entity、regulatory_authority_id、region_id、Ministry 关系、调用方 ID

**GeopoliticRivalry（地缘政治对抗蓝图）**:
以 `GPR + canonical lowercase UUID` 为稳定身份的独立静态叙事蓝图，保存中英文名称、
受控对抗类型、自然语言描述、核心参与方文本、可空外围参与方文本、可空影响区域文本集合、
受控生命周期和数据库生成时间。参与方与影响区域只是人工整理内容，不证明正式 Actor 或
Region 存在，也不建立任何外键。GeopoliticRivalry 不拥有 Storyline；未来 Storyline 如需
使用该蓝图，由 Storyline 侧另行建立关系。
_Avoid_: Storyline 外键、Country/Region/Institution/Ministry 关系、Actor 解析、Tags、调用方 ID、名称去重

**MacroEconomic（宏观经济叙事蓝图）**:
以 `MEC + canonical lowercase UUID` 为稳定身份的独立静态叙事蓝图，保存中英文名称、
受控宏观类型、自然语言描述、受控生命周期和数据库生成时间。MacroEconomic 当前不表达
Country、Region、Institution 或其他外部归属，也不拥有 Storyline；未来 Storyline 如需使用
该蓝图，由 Storyline 侧另行建立关系。
_Avoid_: Economy Entity、Storyline 外键、Country/Region/Institution 关系、业务 code、调用方 ID、名称去重

**StorylineDomain（叙事线领域）**:
以 `SLD + canonical lowercase UUID` 为稳定身份的独立静态叙事领域目录项，以全局唯一、不可变且
只含大写 ASCII 字母、数字和下划线的 `code` 作为机器自然键，并保存中英文名称、自然语言描述、
内容边界、`GEOPOLITICAL | MACRO | INDUSTRY | CORPORATE` 受控分类、启用状态和数据库生成时间。
名称不是自然键，允许重复。当前 35 条受控目录由 Data-owned 版本化初始化包通过独立发布命令
按 code 确定性生成正式身份；初始化包不携带正式 ID、内容边界、subtype 或展示顺序，发布时
由描述生成内容边界并默认启用。StorylineDomain 当前不拥有 Storyline，也不与
StorylineDomainTactic、GeopoliticRivalry、MacroEconomic 或其他对象建立关系。
_Avoid_: Storyline 外键、Tactic 外键、名称去重、调用方 ID、subtype、展示顺序、从分类推断蓝图关系

**StorylineDomainTactic（叙事线领域手段）**:
以 `SDT + canonical lowercase UUID` 为稳定身份的独立静态手段目录项，以全局唯一、不可变且
只含大写 ASCII 字母、数字和下划线的 `key` 作为机器自然键，并保存中英文名称、自然语言描述
和数据库生成时间。虽然名称表达叙事领域手段，本期不保存 `domain_id`，也不证明或推断其
属于任何 StorylineDomain；未来关系必须另行定义方向、基数与迁移。
_Avoid_: Domain 外键、Storyline Thread Template、复合 Domain key、调用方 ID、隐式领域归属

Industry Chain 的可选主要国家范围使用 `primary_country_id` 引用独立 Country；不得把国家
写回 `geography` 自由文本或旧 Economy UUID。已退役的 Sector 持久化表不因 Country 切换而恢复。

**Economy Entity Retirement**:
Tidewise AI 1.0 用于混合表达国家、全球范围、区域和跨国对象的旧通用 Entity 类型已经退役。
活动代码、API、Schema 和最终数据库状态不得把 Country 表达为 `economy`，也不得为新
Country 创建兼容 UUID、双读或双写入口；历史 migration 和合法宏观经济词汇不受影响。
_Avoid_: Economy alias、Country/Economy fallback、从混合旧行猜测 Country

**Research Graph Search**:
Data Service 面向 Codex 分析师提供的同步、无状态、幂等只读图谱检索合同。Codex 显式
指定 seed Entity、每种 Relation 的方向、最大深度、可选 Industry Chain scope 以及
node/edge budget；Data 只校验引用和预算，并从 PostgreSQL 返回稳定排序、引用完整的
可达 EntityRelation 与 Industry Chain Graph 子图。`industry_chain_id` 只约束
Industry Chain Graph Edge，全局 EntityRelation 仍完全由显式 Relation filter 控制。
第一版不分页；预算超限整次返回结构化 `429`，不静默截断。Data Adapter 固定从
PostgreSQL 正式事实构造结果，不切换为 Data-owned Neo4j 投影。
_Avoid_: Data 自动选择 seed/主产业链/最佳路径、Theme readiness 或投资方向判断、
Codex 直连 PostgreSQL/Neo4j、把页级引用闭包当完整研究图谱、未声明的部分子图

**Reason Tree Event 关联（Reason Tree Event Association）**:
Tree 从父 Theme Event 集合选择正式 Event，并保存角色与稳定展示顺序；不复制 Event
正文或证据摘要。Theme Event 与 Tree Event 都只接受
`confirmed + verified` 的正式 Event 事实。`analyst_snapshot` 每棵 Tree 至少关联一个正式
Event，Evidence IDs 可选并仅在提供时
校验归属。V3 Theme Event 是无展示顺序的唯一集合，caller 顺序不构成发布门禁，canonical
hash 按 `event_id` 规范化；Tree Event 仍使用显式 `display_order`。仅在 Theme Event 集合中
出现不能替代 Tree 自身的来源关联。
_Avoid_: Tree 扩展父 Theme Event 边界、遗漏所用事实的来源 Event、要求特定角色组合、
复制 Event 文本

**Reason Tree 读取边界（Reason Tree Read Boundary）**:
列表使用 `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees`，详情使用
`GET /api/data/v1/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}`。
Theme 缺失、无 Tree 回执、Tree 缺失分别返回稳定 `404`；回执存在但投影无法完整
重建返回 `500 RESEARCH_REASONING_TREE_INVARIANT_VIOLATION`。读取按服务端顺序返回
Theme Impact identity；消费者通过 aggregate-local `node_key` 相交判断节点是否为
Theme Impact。
_Avoid_: `anchor_id`、独立 Anchor API、`is_theme_impact`、BFF 扇出查询

**Theme 与 Tree 摘要（Theme and Tree Summaries）**:
Theme 与 Tree 都可保存 `transmission_summary` 和检查点信息，但作用域不同：
Theme 概括整体投资结论，可综合多棵 Tree；Tree 只解释一条 Industry Chain 路径。
Data 原样承接分析师内容，不复制、拼接或执行研究质量校验。

**推理树合同验证（Reasoning Tree Contract Verification）**:
Data OpenAPI、稳定 wire DTO、错误码和 Data contract test 拥有 provider 合同；消费者使用
自己的 typed Adapter、OpenAPI drift test 和必要 HTTP smoke 验证消费。应用内 mock 只属于
该应用，不是 Data 合同或生产数据来源；真实 Theme/Tree 只能经发布 API 入库。

**传导阶段（Transmission Stage）**:
Research Theme 的分析阶段，仅允许 `identification`、`validation`、`diffusion`
和 `dampening`。它与可空 `conclusion_status` 是不同维度，Data 不根据其他字段推断。

## Source Ownership

Data Backend 使用 Kratos 分层，业务代码必须收敛到 `data-service/backend/`：

```text
api/data/v1/  versioned HTTP contract, strict binding and DTOs
cmd/          process and maintenance entrypoints
internal/biz/ Data-owned rules, use cases and outbound ports
internal/data/ persistence adapters and repository wrappers
internal/service/ API-to-Biz application service
internal/server/ Kratos transport, auth and lifecycle wiring
internal/conf/ Data-only runtime configuration
```

`data-service/backend/migrations/` 是 Data 的 PostgreSQL schema 资产，保留在 Backend
根目录；BFF 不得直接读取 migration 或 Data PostgreSQL。

## Runtime

Data Server 与 migration 只通过 Data Docker image 和 Compose 运行。
Data image 不再包含 Entity seed、Industry relationship import、Neo4j projector 或 Qdrant
projector，也不包含 Event Semantic audit/synthetic 命令。每个环境只保留一份
`configs/config.<environment>.yaml`；local 使用容器可访问的
外部 PostgreSQL endpoint，不维护宿主机直跑配置。Neo4j、Qdrant 不由 Data application
创建、持有、探测或写入。这不改变 Data 的 PostgreSQL 事实 ownership。
