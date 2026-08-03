# Data Context

## Purpose

Data Domain Service 是当前唯一 Domain Service，负责稳定的数据事实、领域规则、持久化、受控导入和查询 API。

## Owns

- Entity、产业链节点及关系、Benchmark、Index 等主数据。
- 正式 Event、被 Event 引用的轻量 Evidence Record 及其证据关联。
- Research Theme、Theme Impact、Reason Tree 及其关联数据。
- PostgreSQL schema、migration、repository 和 Neo4j 可重建投影。
- AgentRun 使用的 Event Publication API、自然身份收敛、receipt 和事务规则。
- 面向 Miniapp/Admin Application Backend Service 的版本化 REST API。

## Does Not Own

- Miniapp 或 Admin Portal 的页面 DTO、交互状态和展示逻辑。
- User、Auth、Payment、Subscription 等未来独立领域。
- Source 主数据、完整原始 Artifact、数据采集 connector、parser、采集 prompt 或采集调度执行。
- Agent 的模型推理和工作流运行。

## AgentRun Boundary

位于本仓库 `agent-run/backend/` 的 AgentRun 拥有 Source 主数据、采集调度、采集
执行、完整原始 Artifact 和 Event 提取工作流。物理共仓不改变 ownership：AgentRun
只能按照 Data 定义的版本化 Event Publication 合同提交已提取 Event 及其证据引用，
不直接访问 Data 数据库；Data 不维护 Source Catalog，也不接纳未产生正式 Event 的
采集 Artifact。

Tidewise 中遗留的 Source Catalog、采集调度与采集运行控制面通过保留 `raw_documents` 来源快照的 forward migration 物理移除；该收敛不得删除历史 Event、Evidence Record 或既有证据关联。
Data 的 AgentRun Source Metadata、Admin Source Catalog 查询，以及 Admin Portal 对应代理接口、Client、Repository、Seed 和专属测试一并移除，不保留静态兼容路由；AgentRun 仅使用自身 Source Catalog。

## Language

**产业链（Industry Chain）**:
围绕明确目标产出与终端用途，由多个独立经济节点通过投入、组成、技术支撑或依赖形成的有边界、有方向研究子图。
_Avoid_: Industry、Concept、Chain Node 列表

**产业链节点归属（Industry Chain Node Membership）**:
一个 Chain Node 被纳入某一特定 Industry Chain 的上下文关系；上中下游阶段和位置属于该关系，不是节点的全局属性。
_Avoid_: 节点全局上下游标签、节点之间的图谱边

**产业链图谱边（Industry Chain Graph Edge）**:
同一 Industry Chain 的两个成员节点之间，带有明确方向和结构机制的关系。
_Avoid_: 发现映射、关键词相关、单次 Research Anchor 的临时传导路径

**实体关系类型（Entity Relation Type）**:
对实体关系语义、允许端点、固定方向和遍历含义的规范谓词；正式关系只能使用已批准的稳定 code。
_Avoid_: AI 自由关系字符串、无语义的 related_to

**产品实体（Product Entity）**:
企业生产、销售、采购或被市场需求的可识别产品对象。Product 不等同于表示经济环节或
活动的 Chain Node，也不等同于标准化可交易的 Commodity。
_Avoid_: Chain Node、Commodity、Technology、产品名称 Mention

**AgentRun Artifact**:
AgentRun 在采集执行中生成并长期保存的不可变原始文档对象，包含完整 Markdown 正文和全局唯一 Artifact 身份。它只属于 AgentRun；Data 不保存其存储位置，不读取或校验原文。
_Avoid_: Data Raw Document、Event Evidence Record、Data 原始语料

**Event Evidence Record**:
Data 仅在正式 Event 引用了 AgentRun Artifact 时接纳的轻量证据文档记录，保存 Artifact 身份、内容 SHA-256、AgentRun 稳定 `source_ref`、来源快照和必要时间元数据，不保存完整正文或 Artifact 存储位置。`source_ref` 只是无外键的外部来源引用，Data 不维护其 Source 主数据。内容 SHA-256 只用于检测同一 Artifact 身份是否发生内容漂移，不表示 Data 已读取原文或验证来源真实性。来源快照可保留公开 `source_url` 用于证据归因，允许没有公开地址的来源为空；该地址不是 AgentRun Artifact 的内部位置，Data 不主动访问或校验。一个记录可以支持多个 Event，一个 Event 也可以引用多个记录。
V3 接纳字段只包含必填的 `artifact_id`、`content_sha256`、`source_ref`、`source_name`、`source_type`、`title`、`collected_at`，以及可选的 `source_url`、`published_at`、`language`、`mime_type`。`content_text`、Artifact URI、采集通道、采集状态、内容层级和独立来源外部 ID 不属于 V3 合同。
`source_type` 是由 AgentRun Source Catalog 治理的非空快照字符串，Data 只校验非空和长度，不维护对应枚举或主数据。
_Avoid_: 完整 Raw Document、采集缓存、未产生 Event 的文档

**Event Evidence Link**:
一个正式 Event 与 Event Evidence Record 之间的语义关联，必须包含 `artifact_id`、短而非空的 `evidence_statement`、`evidence_relation` 和 `source_level`。`evidence_statement` 是模型生成并由审核阶段确认的证据陈述，不承诺逐字摘录；程序仅通过 Artifact 身份和内容哈希建立正式血缘，不用原文字符串匹配裁决语义。`evidence_relation` 仅允许 `supports`、`contradicts`、`context`；前两类必须提交非空 `supports_fields`，`context` 可为空。`supports_fields` 仅允许 `title`、`factual_summary`、`occurred_at`、`fact_payload`。`source_level` 仅允许 `primary`、`secondary`，仅表示来源层级，不表示某条 Evidence 在 Event 内拥有主证据地位。Data 根据证据陈述计算 `evidence_hash`，不接收调用方计算的证据哈希。
同一 Artifact 在同一 Event 中只能出现一次，数据库按 `(event_id, raw_document_id)` 保证唯一；一个 Link 可通过 `supports_fields` 覆盖多个字段。再次提交已有 Link 时，关系、证据陈述、支持字段和来源层级必须全部一致，否则整批冲突。Event 不再保存或暴露主证据引用，所有 Evidence Link 具有平等的正式血缘地位。
_Avoid_: 完整正文副本、无语义 Artifact 引用、真实性认证结果

**Event Tag Assignment**:
正式 Event 的受控 Tag 映射。每个 Event 必须包含一至两个 active `news_category`，并可包含零至三个 active `index_category`；每项提交匹配的 Tag ID、kind、code，以及 `confidence`、非空 `assignment_reason` 和 `ai` 或 `rule` 来源。V3 不接收 Tag review status，Data 统一写为 `approved`。已有同 Tag 映射仅在内容一致时复用，新映射可以追加，冲突时整批失败。
_Avoid_: 待审核 Tag、未知或停用 Tag、静默覆盖已有分配依据

**Event Tag Catalog**:
Data 拥有的 Event 分类主数据集合，包含稳定 Tag ID、kind、code、名称、启停状态和可校验版本身份；AgentRun 只能通过 Data 的版本化只读合同取得快照后进行分类。
_Avoid_: AgentRun 自建 Tag 主数据、在 Prompt 或 YAML 中复制 Tag ID、模型创造 Tag

**Event Publication Batch**:
AgentRun 将一至十个已完成提取与审核、状态固定为 `confirmed + verified` 的原子 Event，连同其共享 Event Evidence Record、证据关联、Tag、Review 和提取血缘，按照 Data 定义的严格同步合同整批原子提交为正式事实；候选、未验证或拒绝 Event 不进入 Data，任一成员失败时整批不可见。
每个 Event 独立提交必填的 `review_id`、`evidence_grade` 和非空 `reasons`；V3 不重复提交审核决定、Event/Fact 状态或组件版本，Data 统一写入 `confirmed + verified`。
V3 在批次顶层提交去重后的 `raw_documents`，各 Event 通过 `artifact_id` 引用共享证据。每个 Event 至少引用一个已声明 Artifact；每个顶层 Artifact 也必须至少被一个 Event 引用，未知或重复 Artifact 身份均使整批失败。
Data 在写事务前返回所有当前可确定的合同、枚举、Tag 和引用错误；自然身份内容冲突单独返回冲突错误。任一错误均阻止整个批次和 Receipt 落库，不允许部分成功。
_Avoid_: 独立 Raw Document 导入、Agent 直写数据库、先存全文后补 Event

**Event Import Receipt**:
Data 为每次成功 Event Publication Batch 生成的不可变审计凭证，记录调用主体、`package_id`、正式事实身份、`extractor_execution_id`、`extractor_agent_version`、每个 Artifact 对应的 `collector_execution_id` 和导入时间。以上执行血缘均为必填；Prompt、模型和 Profile 版本仍由 AgentRun 保存。Receipt 不承担请求幂等、重放判断或异步状态查询职责；失败事务不生成 Receipt。
`package_id` 只是 AgentRun 提供的审计关联编号，不唯一且不参与事实复用；相同 package 可以产生多个成功 Receipt，每次成功调用均由 Data 生成新的 `receipt_id`。
Event Publication 必须通过 Data 唯一的内部 Bearer service token 鉴权；Token 只存在于运行环境，不进入数据库。Data 将该凭据解析为稳定的 Data 内部 trust-domain `caller_subject` 写入 Receipt，不区分 AgentRun、Miniapp 或 Admin 等消费者。Event 的消费者级审计由必填 Collector/Extractor 执行血缘承担，与 Source、采集通道或 Artifact 来源无关。本期明确不提供 Data API 的逐消费者 token、scope 隔离或逐消费者 Receipt 主体。
V3 Receipt 存储在专用 `event_publication_receipts`。旧独立 Raw Document 导入和单 Event V1 导入退出后，其 `raw_document_import_receipts`、`event_import_receipts` 及专属数据库触发器/函数连同历史审计记录物理移除；该清理不得删除正式 Event、Event Evidence Record 或 Event Evidence Link。
每次成功调用均创建 Receipt 并返回 `201 Created`，响应包含 `receipt_id`、`package_id`、`imported_at`、Dedupe Key 到 Event ID 的 created/reused 映射、Artifact ID 到 Raw Document ID 的 created/reused 映射，以及 Event、Raw Document、Event Source、Event Tag 的 created/reused 分类计数；不返回 payload hash、replayed 或异步任务状态。
_Avoid_: Idempotency Record、Import Job、失败占位记录

**Event Dedupe Key**:
AgentRun 为一个原子 Event 提交的稳定唯一业务身份，对应 Data 中唯一的 Event 事实；Data 的 Event UUID 是独立数据库身份。相同 Dedupe Key 不得对应不同核心事实，事实修订必须使用新的 Dedupe Key。
_Avoid_: Event UUID、Import Idempotency Key、可覆盖的事件名称

**Event 事实收敛（Event Fact Convergence）**:
相同 Event Dedupe Key 的 `title`、`factual_summary`、可空 `occurred_at` 和按 JSONB 语义比较的 `fact_payload` 必须完全一致，Data 复用已有 Event；任一核心字段修订必须使用新的 Dedupe Key。`first_seen_at` 与 `knowable_at` 不由调用方提交，由 Data 根据全部关联证据计算，并且后续只能随新增的更早证据向更早时间收敛。后续 Publication Batch 可以为该 Event 新增证据或 Tag 关联；已有且语义一致的关联直接复用，已有关系不得被静默改写或删除，冲突时整批失败。每次成功调用仍生成独立 Import Receipt。
`occurred_at = null` 表示证据不足以确定事件发生时间，不影响已确认、已验证且 Evidence 合同
有效的 Event 进入 Event Semantic；Data 不用发布时间、采集时间或首次发现时间补写该字段。
_Avoid_: 覆盖 Event 核心事实、删除旧证据、用新 Receipt 表示新 Event

**研究主题（Research Theme）**:
一次完成分析侧校验并由授权发布主体提交的单 Theme Aggregate 内，对一组 Event 及其
分析对象影响形成的不可变、可发布研究判断快照，包含一句话结论、传导路径和结论演进
阶段。V2 formal 可以绑定正式产业链事实；V3 `analyst_snapshot` 的对象、变量和传导是
分析师报告快照，不是本体事实。同一现实议题在不同 Aggregate 中生成不同 Research
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
事务。零 Theme 是 Codex 合法结果，不向 Data 提交占位对象。Data 按 publication variant
校验 DTO、结构、来源引用和事务：V2 formal 分支继续校验正式本体与 Signal/Impact 血缘；
V3 `analyst_snapshot` 分支只校验 local key 闭包、路径、正式 Event 及调用方可选提交的
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
首次发布时间及整棵聚合的写入计数。V2 formal 使用 Industry Chain ID 映射；V3
`analyst_snapshot` 使用 aggregate-local `tree_key` 映射。Theme、全部 Trees、来源关联和
两类既有回执在同一事务内提交。
首次成功返回 `201 + replayed:false`，相同主体和载荷重放返回
`200 + replayed:true`；重放不得刷新结果。
_Avoid_: 在单条 Theme 上设置批次 ID 唯一约束、业务数据失败后保留回执、修改成功回执

**Theme 发布载荷哈希（Theme Publication Payload Hash）**:
对完整 Theme 发布请求体按 RFC 8785 规范化后计算的小写十六进制 SHA-256，用于批次幂等重放和内容冲突检测。哈希只覆盖调用方提交的批次身份、分析窗口和 Theme 内容，不包含认证信息、请求 ID、服务端发布时间或响应字段；由 Data 计算并返回，调用方不提交。
V3 `analyst_snapshot` 计算前仅将无展示顺序的 `Theme.events` 按 `event_id` 规范化；
其余数组顺序仍属于载荷内容。

**Theme 发布规范数组顺序（Theme Publication Canonical Array Order）**:
Theme 发布请求只有一个 `theme`。V2 formal 的 Impact/Event、Reason Trees、Tree Event、
Node 和 Signal 数组，以及 V3 `analyst_snapshot` 除 `Theme.events` 外的数组，均使用合同
规定的唯一稳定顺序。V3 `Theme.events` 是无展示顺序的唯一关联集合，调用方数组顺序不
构成发布门禁；Data 只在计算 hash 的副本中按 `event_id` 规范化，不改变请求、持久化或
Tree Event 展示顺序。UUID 必须使用标准小写字符串，同一作用域不得重复键或 ID。
_Avoid_: 大小写混合 UUID、重复关联、重排有展示语义的数组、将 V3 Theme Event 的 caller 顺序误作校验门禁

**Theme Aggregate 发布 V2（Theme Aggregate Publication V2）**:
顶层字段为 `analysis_batch_id`、`analysis_as_of`、`discovery_window_start`、
`discovery_window_end`、单个 `theme` 和 `reasoning_trees[1..N]`。Reason Tree Signal
必须声明 `formal_signal` 或 `analyst_inference` 血缘；非根节点 incoming transmission
必须声明 `formal_direct_impact` 或 `analyst_inference` 血缘。合同严格拒绝未知字段。
_Avoid_: `subject_entity_id`、Theme `name`、`impact_level`、`trading_direction`、
`transmission_path`、`next_checkpoint`、`market_confirmation_summary`

**Theme Aggregate Analyst Snapshot 发布 V3（Theme Aggregate Analyst Snapshot Publication V3）**:
沿用 `POST /api/data/v1/research-theme-imports` 的隔离 request variant，顶层以
`publication_mode=analyst_snapshot` 明确判别。它发布分析师拥有的不可变报告快照：Theme
Impact、Tree、Node 和 Signal 使用 aggregate-local key 与 publication-time display snapshot，
不提交也不依赖 Entity、IndustryChain、VariableDefinition、VariableSignal、DirectImpact、
Relation 或 GraphEdge formal ID。每个 Theme 与每棵 Tree 至少关联一个正式 Event；
Evidence ID 可选，提供时才校验存在且属于所声明 Event。JSON/schema 错误返回 `400`，
local-key closure、路径、Event/Evidence 等业务结构或引用错误返回 `422`。
_Avoid_: 根据 formal ID 是否存在猜测分支、将本体覆盖作为发布门禁、从 display 文案反推
formal identity、发布时晋升本体事实

**主题影响（Theme Impact）**:
Research Theme 的不可变关注对象快照，保存角色、方向、可空摘要和稳定展示顺序。所有
Impact 平等，不存在 subject、primary 或主要影响节点；一个 Theme 至少有一个 Impact。
V2 formal Impact 引用有效 Chain Node；V3 `analyst_snapshot` Impact 使用 aggregate-local
`node_key + display_name`，只要求该 key 在至少一棵 Tree 中被覆盖。Theme 卡片短名称与
Tree Node 具体名称是独立 presentation snapshot，Data 不比较二者文案。
_Avoid_: `is_primary`、把展示顺序解释为影响优先级、要求 V3 分析对象先成为正式 Entity

**产品可见 Theme（Product-visible Research Theme）**:
属于一次成功 Theme Aggregate 事务且满足读取窗口的 Theme。新 V2 Theme 必然同时拥有
至少一棵完整 Tree；新 V3 `analyst_snapshot` 同样必然包含至少一棵完整 Tree。既有 V1
历史数据保持可读，但不构成新发布兼容约束。
_Avoid_: 新 Theme 无 Tree 可见、读取时动态补写 Tree

**推理树（Reason Tree）**:
一个 Theme 的不可变线性推导链路。V2 formal Tree 属于一条 Industry Chain，同一 Theme
与同一 Industry Chain 最多一棵；V3 `analyst_snapshot` Tree 使用 aggregate-local
`tree_key + display_name`，display name 可以是产业链、宏观、估值或商业模式传导路径，
不要求正式 Industry Chain。两种分支都可以解释一个或多个 Theme Impact，也可以包含
只承担传导上下文的节点。Tree 没有中心节点或主要影响节点。
_Avoid_: Research Anchor、中心节点、任意图、分叉、循环、用 Tree `display_order`
表达影响优先级

**Reason Tree 发布集合（Reason Tree Publication Set）**:
Reason Trees 只能通过 `POST /api/data/v1/research-theme-imports` 随所属 Theme
Aggregate 同步发布，不存在独立写入口。请求至少包含一棵 Tree，全部 Tree 与 Theme
共同使用 `analysis_batch_id` 幂等身份和同一事务。
_Avoid_: `research-reasoning-tree-imports` 写入口、先 Theme 后 Tree、第二套幂等键

**Reason Tree 身份与回执（Reason Tree Identity and Receipt）**:
V2 formal Tree 身份由 `theme_id + NUL + industry_chain_entity_id` 确定性生成；V3
`analyst_snapshot` Tree 身份由 `theme_id + NUL + tree_key` 确定性生成。每个 Theme
最多一条 Tree 集合回执，保存发布主体、payload hash、对应分支的 Tree identity 映射、
写入计数和首次发布时间；回执与全部 Tree 子记录同事务提交。
_Avoid_: 调用方提交 Tree ID、原地覆盖、每棵 Tree 单独回执

**Reason Tree 节点（Reason Tree Node）**:
Tree 的有序分析对象快照。`position` 从 1 连续排列并且是唯一路径顺序；一个节点在
Tree 内唯一。V2 formal Node 必须是该 Tree Industry Chain 的 active/approved 成员；V3
`analyst_snapshot` Node 使用 Tree-local `node_key + display_name`，不要求正式 Entity、
ChainNode 或 membership。首节点全部 `incoming_*` 为空；后续节点必须保存传导机制，V3
允许标题和成立条件为空。
V2 可空正式 Graph Edge 必须 active/approved、属于同一链且端点匹配；为空表示分析推断。
V2 `formal_direct_impact` 的 source VariableSignal subject 必须等于前一节点、target
必须等于当前节点；`analyst_inference` 必须引用上游正式事实作为整棵 Tree 的 Event
血缘锚点，并引用实际使用的 Relation/Graph Edge。Data 忽略该关系的存储方向，只要求
它连接前一节点与当前节点；不要求根事实实体直接连接所有下游节点，允许多个连续
`analyst_inference` step 复用同一 accepted 根事实。
_Avoid_: 持久化结果节点标记、首节点伪造入边、从 Tree 回写正式图谱

**Variable Signal 展示快照（Variable Signal Display Snapshot）**:
每个 Tree 节点拥有 1..5 个按 `display_order` 排列的不可变显示快照，恰好一个
`primary`。V2 `formal_signal` 同时固定 VariableSignal UUID、Semantic Submission、
Evidence ID/hash，并验证节点 Entity、变量和方向一致；`analyst_inference` 不得冒充
正式 Signal，必须引用一个正式上游 Signal/Impact 作为事实/Event 血缘锚点，以及连接
相邻 Tree Node 的实际 Relation/Graph Edge；关系端点方向可与 Tree 传播顺序相反。
V3 `analyst_snapshot` Signal 使用 Node-local `signal_key` 和必填 `display_summary`，变量名与
方向可空，不提交或依赖正式 Variable/Signal/Lineage ID。读取时只返回发布时显示快照，
不用当前正式事实动态重写文案。
_Avoid_: 从显示文本反推正式事实、Inference 伪造 Signal ID、要求 V3 Signal 绑定本体

**Event 原生 Variable Signal（Event-native Variable Signal）**:
由正式 Event 的明确陈述与 Evidence 直接支持、描述规范 Entity 上受控变量状态或变化的
语义事实；它保留来源原有模态，不包含 Agent 自行预测或下游影响推导。
_Avoid_: Variable Signal Display Snapshot、模型预测、产业链传导结论

**Signal 断言模态（Signal Assertion Modality）**:
Event 原生 Variable Signal 对来源陈述性质的区分：`actual` 表示实际结果或当前可观测
状态，`stated_intent` 表示尚未兑现的明确计划或承诺，`source_forecast` 表示可归因于
明确外部主体的预测或指引。
_Avoid_: 实现状态、审核状态、Agent 预测置信度

**Variable Definition**:
对一个可用于 Event 语义的变量所作的受控 TBox 定义，明确其业务含义、值语义、允许
方向和适用 Entity Type。它不是一次观测、一个 Metric Entity 或自由 Prompt 词汇。
_Avoid_: Observation、Event-native Variable Signal、Metric Entity、模型自由变量名

**Entity Type Definition**:
Data PostgreSQL 中对规范 Entity Type 的受控 TBox 定义。除稳定 `type_key/version`、Signal
Subject 能力与允许 Event 角色外，V3 还持久化中英文名称、业务定义、纳入标准、排除标准和
`event_link_allowed`。这些字段由既有领域与 Data layer 读取、Context API 和 Research
Analysis Context 暴露，并约束 Event Semantic 投影与提交预检。首批既有 active 类型的内容
由本次 forward migration 一次性人工编写回填；这不表示存在运行时生成器、Curator、管理 UI
或持续同步功能。
_Avoid_: Prompt 内写死类型清单、模型预测类型作为事实、单独 TBox 生成系统、只改数据库而不改领域读取合同

**Measurement Value**:
Variable Signal 或未来 Observation 复用的结构化数值对象。它以受控角色区分绝对水平、
绝对变化、相对变化和百分点变化，保留 exact/range/单边界、原始与规范值、单位、币种、
尺度、比较口径、业务期间、近似标记和来源精度。Phase One 内嵌于 Variable Signal，
不单独成为 Observation。
_Avoid_: 松散 value/unit 字段、只保存格式化文本、浮点误差、丢弃原始边界或精度

**Observation**:
未来用于表达某个变量在一个时点或期间的独立实际测量对象，适用于连续采集、时间序列、
数据修订、多来源融合和预测兑现验证。Phase One 不建立 Observation，待持续指标需求
出现时复用 Measurement Value 并从 Variable Signal 无损迁移。
_Avoid_: Phase One Event-native Measurement、Variable Signal 方向断言、展示快照

**Direct Impact Assertion**:
一个 Event 原生 Variable Signal 通过明确的一跳业务关系和单一机制，对另一个规范
Entity 的受控变量形成的分析断言。它必须声明受影响 Variable Definition 与方向；
即使被接受也不是 Event 原生事实。Signal Subject 是变量原始落点，Target 必须是不同
Entity；没有合法跨实体 Target 时 Signal 可以独立存在。
_Avoid_: Signal 自影响、无受控变量的利好/利空、产业链多跳传导、Security 投资方向、
Sector/Concept 聚合结论

**Direct Transmission Rule**:
Data Service 拥有、版本化并审核的一跳 TBox 因果映射。它以 Source Entity Type、
Source Variable、Source Direction、Entity Relation Type 和 Target Entity Type 为
匹配条件，输出 Target 上的受影响 Variable 与方向；规则不包含具体 Event 或 Entity
实例 ID。阶段一只为 Golden Scenario 人工定义和批准少量规则，不建设递归、自动学习、
复杂条件或通用 DSL 引擎。
_Avoid_: Entity Relation 实例、Direct Impact Assertion、把 Golden Entity ID 写入规则、
AgentRun 私有规则、完整多跳 Transmission Rule 引擎

**语义候选审核状态（Semantic Candidate Review Status）**:
Event Entity Link 和 Variable Signal 各自拥有的领域审核
状态：`accepted` 可供下游使用；`pending_review` 正在等待自动 AI 复核或可恢复处理；
`needs_reanalysis` 需要补充 Evidence、重新提取或解析；`quarantined` 表示自动重试
预算耗尽后长期隔离；`rejected` 保留稳定拒绝原因；`superseded` 表示已有新候选替代但
旧记录继续审计。上游未 accepted 时，下游不得 accepted。
_Avoid_: 用 Submission 单一状态覆盖全部候选、把 Pending 当正式 ABox、等待人工 UI、
重试耗尽后自动接受、删除拒绝记录、为 V2 新增 DirectImpact

**Acceptance Policy**:
Data Service 用于把已通过基本合同校验的语义候选路由到审核状态的版本化策略。它组合
确定性门禁、逐字 Evidence、独立语义审核、冲突/歧义与经 Golden Fixture 校准的分对象
Confidence 阈值；Evidence Grade 或模型 Confidence 都不能单独决定接受或拒绝。
_Avoid_: 全局永久阈值、未经校准的置信常量、Grade A 自动接受、Grade C 自动拒绝

**独立语义复核（Independent Semantic Review）**:
Candidate Generator 之后的独立 AI 调用，使用独立 Prompt/版本，仅接收候选、Event
Evidence、Ontology Context 和校验清单，结构化输出 `pass | fail | indeterminate`。
它可以与 Generator 使用同一基础 LLM，但不读取 Generator 的自由推理过程，也不能直接
写 accepted；Submission 冻结 Reviewer 与 Adjudicator 的 Prompt/模型身份，Data 持久化
每轮 Review Snapshot。首次 `indeterminate` 进入 `needs_reanalysis`，冻结的第二轮再次
`indeterminate` 时进入 `quarantined`；未知结果恢复必须沿用已持久化的下一轮身份。最终状态始终由 Data Service 的确定性门禁和 Acceptance Policy 决定。
_Avoid_: Generator 自我确认、Reviewer 直接改领域状态、开放式多 Agent 辩论

**Event Semantic Submission**:
Data Service 对一个正式 Event 的一次语义提交、确定性校验、独立 AI Review Result、
Acceptance Policy 裁决和产物血缘记录，与 AgentRun 的一个 Agent Execution 一对一。
新 V3 Submission 仅包含 EventEntityLink 和 VariableSignal（其可选包含自然语言
Measurement），不接受、生成或要求 DirectImpact。Data 在写事务内以 PostgreSQL 正式
Entity/TBox/Evidence 重新校验 Qdrant 候选中被选中的 ID，并要求 Submission 的
`projected_entity_type` 与 PG Entity Type 完全一致；Signal 还必须满足该正式类型的
`signal_subject_allowed`。
它保存外部执行身份及 Agent/Ontology/Rule/Prompt/Model 版本快照，但不复制 AgentRun 的
runtime 状态、调度重试或执行错误；重新分析创建新 Submission 并 supersede 旧
Submission。
_Avoid_: Agent Execution 副本、Theme Analysis Batch、原地覆盖重新分析、跨 Event 批次

**Research Analysis Context**:
Data Service 面向工程外部 Codex 分析师提供的同步、无状态、实时批量读取合同。调用方
显式提交 discovery window、`analysis_as_of` 与 page size；Data 只返回
`confirmed + verified` Event 及 accepted/latest/non-superseded Event Semantics、
Evidence 和版本化 TBox。分页单位是完整 Event Bundle；Data 先选本页 Event，再返回
这些正式事实所引用 Entity、Relation、Variable、Rule、Policy 及端点的最小引用闭包。
cursor 只绑定标准化查询、稳定排序与合同版本，不绑定全库或页级字典 Payload；
`event_page_fingerprint` 与 `reference_closure_fingerprint` 分别记录本页事实和闭包
血缘。MVP 不保存 Snapshot，也不宣称严格历史回放；响应固定标记
`retrospective_reconstruction`，未来创建/更新的字典事实被排除，已知状态历史缺口
返回 `422`，实时引用不一致返回 `409` 并要求从第一页重查。PostgreSQL 在 JSON 聚合前
执行行数与原始字节预算，最终编码仍执行 Bundle、闭包和整页预算。超限返回包含组件、
上限和技术性重试建议的结构化 `429`；只有完成计数或测量时返回实际值，bounded
traversal 在 `budget+1` 提前停止时省略未知的实际总数。
_Avoid_: Data 聚合 Event 成 Theme、Snapshot/Task 表、Codex 直连 PG/Neo4j、当前 TBox
冒充严格历史 TBox、把全库字典重复装入每个 Event 页、先物化无界 JSON 再做资源限制

**Research Graph Search**:
Data Service 面向 Codex 分析师提供的同步、无状态、幂等只读图谱检索合同。Codex 显式
指定 seed Entity、每种 Relation 的方向、最大深度、可选 Industry Chain scope 以及
node/edge budget；Data 只校验引用和预算，并从 PostgreSQL 返回稳定排序、引用完整的
可达 EntityRelation 与 Industry Chain Graph 子图。`industry_chain_entity_id` 只约束
Industry Chain Graph Edge，全局 EntityRelation 仍完全由显式 Relation filter 控制。
第一版不分页；预算超限整次返回结构化 `429`，不静默截断。未来可在不改变 API 语义的
前提下把 Data Adapter 切换为 Neo4j 投影。
_Avoid_: Data 自动选择 seed/主产业链/最佳路径、Theme readiness 或投资方向判断、
Codex 直连 PostgreSQL/Neo4j、把页级引用闭包当完整研究图谱、未声明的部分子图

**Event Semantic Context Lease**:
Data Service 为 AgentRun 已领取的一个 Event Semantic Work Item 提供的短时数据快照授权。
它通过轻量 `context_manifest` 固定 Event/Evidence 身份与指纹、Ontology/Policy、Entity Type 与
Variable Definition 版本引用和可选 superseded Submission 边界；Evidence 摘录和
完整 TBox 对象由 Context API 按这些 pinned identities 读取，不复制进 manifest，也不复制全量
Entity / EntityRelation ABox。Lease 以 Agent Execution ID 为唯一恢复身份；同一执行可
精确续期并复用原 manifest；旧 snapshot-only Lease 重放时仅为该 Lease 生成 compact
manifest，并保留历史 snapshot。历史 `context_snapshot` 只保留审计兼容，新 Lease 不再写入
或对外返回它。Eligible Event cursor 查询与首次 Lease 创建使用同一 Semantic Input
Eligibility：正式 Event 必须 confirmed、verified、有 Event time，且返回的全部 Evidence
满足当前 Context 身份、哈希、摘要、来源和时间字段合同。对应 Submission 终结后 Lease
被消费。它不是任务、队列或 Agent 执行租约；调度、失败恢复和重试始终属于 AgentRun。
_Avoid_: Reanalysis Task、Agent Work Item、在 Data 中调度模型调用、无限续租

**Event Semantic Measurement**:
VariableSignal 的可选一对多、Evidence-grounded 自然语言量化附注。V3 wire 只包含
`measurement_text + evidence_ids`；Data 校验非空、长度、数量、Evidence 归属和引用
完整性，不解析或校验数值、单位、范围、百分比/百分点或时间归一化。
它仅供下游 Theme Analyst 阅读和推理，不用于数据库计算。
_Avoid_: Observation、伪造数值、任意无 Evidence 自由文本、恢复数值归一化强校验

**Event Semantic Qdrant Projection**:
Data Service 从 active/current 正式 PostgreSQL Entity 和 Variable Definition 构建的两个可重建
语义召回 collection。Data 拥有一次性全量 projector、普通批量 embedding Port、
OpenAI-compatible HTTP adapter 和 Qdrant 写入；它不拥有 accepted 状态，也不替代
PostgreSQL 事实校验。Data 代码不使用 Eino/eino-ext，AgentRun 也不通过 Data API
获取 embedding 或语义候选。
Entity projection 只读取 active 且其 active Entity Type Definition 明确
`event_link_allowed=true` 的正式 Entity；Qdrant payload 携带正式 Entity Type，AgentRun
exact/vector 均跨类型召回，再由候选级 Selector 消歧。
Entity 外层 point ID 必须等于 payload `entity_id`；payload 不重复保存 `point_id`。payload 还必须
携带用于投影来源审计的 `source_identity`、固定
`projection_version`、`embedding_model` 和基于冻结 projection document 的 SHA-256
`content_fingerprint`。AgentRun 对缺失、异源、旧版本或错误模型的 point fail closed；Data 仍以
PostgreSQL 正式 Entity ID/type/status 完成 Submission 最终复核。
Data semantic projection 的 Embedding/Qdrant HTTP Adapter 原样保留 `context.Canceled` 与
`context.DeadlineExceeded`，不把调用方取消或 deadline 包装为普通 endpoint unavailable。
_Avoid_: Qdrant 作为事实源、Data 执行 Agent Workflow、实时/CDC 同步、Data 代理 AgentRun 搜索

**Event Semantic Resolution Route**:
历史 V1 术语；已被 Event Semantic V2 Qdrant Projection 取代，不再对新 AgentRun 流程提供。
Data Service 暴露的版本化、受控 Entity Resolution 路径。ChainNode MVP 只允许从正式且
已批准的 Industry 或 Concept 锚点，经 `mapped_to_industry | mapped_to_concept` 到
IndustryChain，再经已批准 Membership 到 ChainNode；路由、锚点和候选均稳定排序并分页。
Industry 一级分区以正式 UUID 加显示名称提供，Route 同时声明方向、用途和下一操作。
Industry anchor 页只返回分区内存在正式映射且可到达 approved ChainNode 的后代叶级锚点，
因此 L3 mapping 无需模型递归即可到达。Anchor/Candidate 均在 PostgreSQL 使用
`canonical_name + entity_id` keyset 和 `LIMIT page_size + 1`，不得先物化全量结果再切页。
_Avoid_: 开放式图遍历、AgentRun 直连数据库/Neo4j、全库 Entity Catalog、模型发明路径

**Event Semantic Resolution Binding**:
历史 V1 术语；V2 不再生成或校验该 binding。
一个 Submission 实际选中的正式 Anchor→IndustryChain→ChainNode 路径回执。Data 在候选
响应中生成 receipt，Submission 事务中重新计算路径指纹；漂移返回可重试冲突，只有通过
核验的选中 binding 才持久化。未选候选与空候选不写表。
_Avoid_: 保存全部候选、模型生成 receipt、用日志或监控替代提交前校验

**Reason Tree Event 关联（Reason Tree Event Association）**:
Tree 从父 Theme Event 集合选择正式 Event，并保存角色与稳定展示顺序；不复制 Event
正文或证据摘要。Theme Event 与 Tree Event 都只接受
`confirmed + verified` 的正式 Event 事实。V2 formal 结构上数组可为空，但发布引用校验
要求每棵 Tree 覆盖自身引用的全部正式 Signal/DirectImpact 及上游正式事实的来源 Event；
V3 `analyst_snapshot` 每棵 Tree 至少关联一个正式 Event，Evidence IDs 可选并仅在提供时
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
Theme Impact identity；V2 消费者通过 formal Chain Node ID 相交，V3 消费者通过
aggregate-local `node_key` 相交判断节点是否为 Theme Impact。
_Avoid_: `anchor_id`、独立 Anchor API、`is_theme_impact`、BFF 扇出查询

**Theme 与 Tree 摘要（Theme and Tree Summaries）**:
Theme 与 Tree 都可保存 `transmission_summary` 和检查点信息，但作用域不同：
Theme 概括整体投资结论，可综合多棵 Tree；Tree 只解释一条 Industry Chain 路径。
Data 原样承接分析师内容，不复制、拼接或执行研究质量校验。

**推理树共享测试 Fixture（Reasoning Tree Shared Test Fixture）**:
`testdata/reasoning-tree-v1/` 保存 provider/consumer 共用的确定性合同样例，不是
生产数据来源；真实 Theme/Tree 只能经发布 API 入库。

**传导阶段（Transmission Stage）**:
Research Theme 的分析阶段，仅允许 `identification`、`validation`、`diffusion`
和 `dampening`。它与可空 `conclusion_status` 是不同维度，Data 不根据其他字段推断。

## Source Ownership

Data Backend 使用 Kratos 分层，业务代码必须收敛到 `analyse-data-service/backend/`：

```text
api/data/v1/  versioned HTTP contract, strict binding and DTOs
cmd/          process and maintenance entrypoints
internal/biz/ Data-owned rules, use cases and outbound ports
internal/data/ persistence adapters and repository wrappers
internal/service/ API-to-Biz application service
internal/server/ Kratos transport, auth and lifecycle wiring
internal/conf/ Data-only runtime configuration
```

`analyse-data-service/backend/migrations/` 与 `analyse-data-service/backend/data/` 是 Data 的统一事实资产，可以保留为 Backend 根资产，但不得被 BFF 直接读取。
