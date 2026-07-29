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
V2 接纳字段只包含必填的 `artifact_id`、`content_sha256`、`source_ref`、`source_name`、`source_type`、`title`、`collected_at`，以及可选的 `source_url`、`published_at`、`language`、`mime_type`。`content_text`、Artifact URI、采集通道、采集状态、内容层级和独立来源外部 ID 不属于 V2 合同。
`source_type` 是由 AgentRun Source Catalog 治理的非空快照字符串，Data 只校验非空和长度，不维护对应枚举或主数据。
_Avoid_: 完整 Raw Document、采集缓存、未产生 Event 的文档

**Event Evidence Link**:
一个正式 Event 与 Event Evidence Record 之间的语义关联，必须包含 `artifact_id`、短而非空的 `evidence_excerpt`、`evidence_relation`、`source_level` 和 `is_primary`。`evidence_relation` 仅允许 `supports`、`contradicts`、`context`；前两类必须提交非空 `supports_fields`，`context` 可为空。`supports_fields` 仅允许 `title`、`factual_summary`、`occurred_at`、`fact_payload`。`source_level` 仅允许 `primary`、`secondary`，表示来源层级而不是 Event 主证据。每个 Event 必须且只能显式指定一条 `is_primary=true`，Data 用它设置 `events.primary_source_id`；后续不得静默更换既有主证据。Data 根据摘录计算 `evidence_hash`，V2 不接收重复 `source_url`、内容层级或调用方计算的证据哈希；Data 只校验合同，不读取 AgentRun 原文核对摘录。
同一 Artifact 在同一 Event 中只能出现一次，数据库按 `(event_id, raw_document_id)` 保证唯一；一个 Link 可通过 `supports_fields` 覆盖多个字段。再次提交已有 Link 时，关系、摘录、支持字段、来源层级和主证据标记必须全部一致，否则整批冲突。
_Avoid_: 完整正文副本、无语义 Artifact 引用、真实性认证结果

**Event Tag Assignment**:
正式 Event 的受控 Tag 映射。每个 Event 必须包含一至两个 active `news_category`，并可包含零至三个 active `index_category`；每项提交匹配的 Tag ID、kind、code，以及 `confidence`、非空 `assignment_reason` 和 `ai` 或 `rule` 来源。V2 不接收 Tag review status，Data 统一写为 `approved`。已有同 Tag 映射仅在内容一致时复用，新映射可以追加，冲突时整批失败。
_Avoid_: 待审核 Tag、未知或停用 Tag、静默覆盖已有分配依据

**Event Tag Catalog**:
Data 拥有的 Event 分类主数据集合，包含稳定 Tag ID、kind、code、名称、启停状态和可校验版本身份；AgentRun 只能通过 Data 的版本化只读合同取得快照后进行分类。
_Avoid_: AgentRun 自建 Tag 主数据、在 Prompt 或 YAML 中复制 Tag ID、模型创造 Tag

**Event Publication Batch**:
AgentRun 将一至十个已完成提取与审核、状态固定为 `confirmed + verified` 的原子 Event，连同其共享 Event Evidence Record、证据关联、Tag、Review 和提取血缘，按照 Data 定义的严格同步合同整批原子提交为正式事实；候选、未验证或拒绝 Event 不进入 Data，任一成员失败时整批不可见。
每个 Event 独立提交必填的 `review_id`、`evidence_grade` 和非空 `reasons`；V2 不重复提交审核决定、Event/Fact 状态或组件版本，Data 统一写入 `confirmed + verified`。
V2 在批次顶层提交去重后的 `raw_documents`，各 Event 通过 `artifact_id` 引用共享证据。每个 Event 至少引用一个已声明 Artifact；每个顶层 Artifact 也必须至少被一个 Event 引用，未知或重复 Artifact 身份均使整批失败。
Data 在写事务前返回所有当前可确定的合同、枚举、Tag 和引用错误；自然身份内容冲突单独返回冲突错误。任一错误均阻止整个批次和 Receipt 落库，不允许部分成功。
_Avoid_: 独立 Raw Document 导入、Agent 直写数据库、先存全文后补 Event

**Event Import Receipt**:
Data 为每次成功 Event Publication Batch 生成的不可变审计凭证，记录调用主体、`package_id`、正式事实身份、`extractor_execution_id`、`extractor_agent_version`、每个 Artifact 对应的 `collector_execution_id` 和导入时间。以上执行血缘均为必填；Prompt、模型和 Profile 版本仍由 AgentRun 保存。Receipt 不承担请求幂等、重放判断或异步状态查询职责；失败事务不生成 Receipt。
`package_id` 只是 AgentRun 提供的审计关联编号，不唯一且不参与事实复用；相同 package 可以产生多个成功 Receipt，每次成功调用均由 Data 生成新的 `receipt_id`。
Event Publication 必须通过 Data 唯一的内部 Bearer service token 鉴权；Token 只存在于运行环境，不进入数据库。Data 将该凭据解析为稳定的 Data 内部 trust-domain `caller_subject` 写入 Receipt，不区分 AgentRun、Miniapp 或 Admin 等消费者。Event 的消费者级审计由必填 Collector/Extractor 执行血缘承担，与 Source、采集通道或 Artifact 来源无关。本期明确不提供 Data API 的逐消费者 token、scope 隔离或逐消费者 Receipt 主体。
V2 Receipt 存储在专用 `event_publication_receipts`。旧独立 Raw Document 导入和单 Event V1 导入退出后，其 `raw_document_import_receipts`、`event_import_receipts` 及专属数据库触发器/函数连同历史审计记录物理移除；该清理不得删除正式 Event、Event Evidence Record 或 Event Evidence Link。
每次成功调用均创建 Receipt 并返回 `201 Created`，响应包含 `receipt_id`、`package_id`、`imported_at`、Dedupe Key 到 Event ID 的 created/reused 映射、Artifact ID 到 Raw Document ID 的 created/reused 映射，以及 Event、Raw Document、Event Source、Event Tag 的 created/reused 分类计数；不返回 payload hash、replayed 或异步任务状态。
_Avoid_: Idempotency Record、Import Job、失败占位记录

**Event Dedupe Key**:
AgentRun 为一个原子 Event 提交的稳定唯一业务身份，对应 Data 中唯一的 Event 事实；Data 的 Event UUID 是独立数据库身份。相同 Dedupe Key 不得对应不同核心事实，事实修订必须使用新的 Dedupe Key。
_Avoid_: Event UUID、Import Idempotency Key、可覆盖的事件名称

**Event 事实收敛（Event Fact Convergence）**:
相同 Event Dedupe Key 的 `title`、`factual_summary`、可空 `occurred_at` 和按 JSONB 语义比较的 `fact_payload` 必须完全一致，Data 复用已有 Event；任一核心字段修订必须使用新的 Dedupe Key。`first_seen_at` 与 `knowable_at` 不由调用方提交，由 Data 根据全部关联证据计算，并且后续只能随新增的更早证据向更早时间收敛。后续 Publication Batch 可以为该 Event 新增证据或 Tag 关联；已有且语义一致的关联直接复用，已有关系不得被静默改写或删除，冲突时整批失败。每次成功调用仍生成独立 Import Receipt。
_Avoid_: 覆盖 Event 核心事实、删除旧证据、用新 Receipt 表示新 Event

**研究主题（Research Theme）**:
一次完成分析侧校验并由授权发布主体提交的分析批次内，对一组 Event 及其产业链影响形成的不可变、可发布研究判断快照，包含一句话结论、传导路径和结论演进阶段。同一现实议题在不同分析批次中生成不同 Research Theme；首页只展示最新成功发布批次的 Theme。
已发布内容的纠错必须由分析侧生成新的运行身份并发布完整修正批次，旧批次保留审计。本期不提供更新、删除或撤回 Theme/批次的 API。
_Avoid_: 覆盖或删除历史 Theme、把 `theme_key` 当作跨批次稳定身份、原地修订已发布批次

**批次内主题键（Theme Key）**:
分析侧为单个 Research Theme 提供的批次内稳定键，在同一 Analysis Batch ID 内唯一，用于确定性 Theme 身份、回执映射和错误定位。合法键长度为 1 至 128，只允许小写 ASCII 字母、数字及 `.`、`_`、`:`、`-`，不接受服务端规范化；同批次按 ASCII 字节顺序排序。`theme:` 前缀推荐但不强制。它不跨批次合并主题，也不等同于未来 Research Thesis 身份；新批次即使复用同一 Theme Key，也产生不同 Research Theme。
_Avoid_: 调用方提交 Theme UUID、把 Theme Key 当作长期主题身份

**长期研究命题（Research Thesis）**:
未来用于跨批次持续跟踪同一研究议题或产业瓶颈的独立对象。Research Theme 不承担该职责。

**研究主题发布批次（Research Theme Publication Batch）**:
同一次分析产生、通过分析侧校验并由授权发布主体共同发布的一组 Research Theme，是 Theme 导入和发布的最小原子单元。发布批次至少包含一个 Theme；没有可发布 Theme 时分析侧不发起发布，也不生成回执。任一 Theme 引用了不存在的 Event、产业链节点或其他强关联主数据时，整个批次拒绝且不产生任何可见 Theme；首页只展示完整成功发布的最新批次。本期不建模人工审核状态或审核元数据。
_Avoid_: 部分入库、跳过无效 Theme 后继续发布、展示未完整发布的批次

**主题批次发布时间（Theme Batch Published At）**:
一个完整 Research Theme Publication Batch 正式成为产品可见事实的服务端时间。Data 在整批校验通过并提交事务时统一生成，同一批次全部 Theme 共享该时间；失败批次不产生发布时间，幂等重放保留首次成功发布的时间。首页在调用方指定的查询时间范围内，按批次发布时间选择最新成功发布批次；范围内没有批次时返回空集合。
_Avoid_: 调用方指定发布时间、按单条 Theme 选择最新批次、重放时刷新发布时间

**分析批次身份（Analysis Batch ID）**:
分析侧一次运行的全局唯一、不可变身份，由分析侧 `run_id` 一对一传入，同时承担 Theme 发布的幂等身份，不另设幂等键。同一 Analysis Batch ID 和相同 canonical 发布内容属于幂等重放，并保留首次发布结果；同一身份对应不同内容属于冲突，不能覆盖或修订已发布批次。校验失败不占用该身份，修复依赖后可以相同请求重试；内容修订必须使用新的分析运行身份。
_Avoid_: 第二套幂等键、覆盖已发布批次、失败校验生成成功 receipt

**分析窗口（Analysis Window）**:
一个研究主题发布批次所覆盖的事实时间范围，由批次级 `window_start` 和 `window_end` 表达，使用 UTC 时间且结束时间必须严格晚于开始时间。同一批次所有 Theme 共享该窗口；分析窗口与服务端发布时间相互独立。
_Avoid_: 零长度窗口、每个 Theme 重复声明窗口、用发布时间替代分析窗口

**研究主题发布主体（Research Theme Publisher Subject）**:
Data 内唯一内部 service token 对应的稳定 trust-domain 身份，由 Data 从认证上下文解析，不由请求声明。首次成功发布时该主体取得批次所有权；后续幂等重放必须来自同一 trust-domain 主体。主体身份独立于可轮换 token，审计只保存主体 ID，不保存凭据。本期不以调用方服务区分 Theme 所有权，所有持有 Data service token 的受信内部消费者处于同一发布信任域。
_Avoid_: 在请求体中声明发布者、以 token 字符串作为长期身份、把同一 trust-domain 内的消费者伪装成独立鉴权主体

**Theme 发布未知结果恢复（Theme Publication Unknown-outcome Recovery）**:
同步 Theme 发布请求超时后，发布器以完全相同的 Analysis Batch ID 和发布内容重试 POST。首次事务已成功时返回原结果并标记重放，未成功时正常执行，内容变化时返回冲突。本期不提供状态查询、轮询或异步任务接口。

**Theme 发布回执（Theme Publication Receipt）**:
一个成功 Research Theme Publication Batch 的不可变技术回执，也是 Analysis Batch ID 全局唯一性、发布主体所有权、payload hash、首次 Theme IDs、发布时间和写入数量的持久化依据。每批只有一条回执；回执、全部 Theme 及其关联事实必须在同一事务中提交或回滚。`replayed` 表示当前响应是否来自重放，不是回执中固化的首次结果。
V1 HTTP 成功结果使用 `theme_ids_by_key` 对象返回完整的 Theme Key 到 Theme UUID 映射；`counts` 仅包含 `themes`、`impacts`、`event_associations` 和 `receipts`。首次成功返回 `201 Created` 和 `replayed: false`，同主体同载荷重放返回 `200 OK` 和 `replayed: true`。两种结果都返回首次成功时的 `receipt_id`、`payload_hash`、`published_at`、`imported_at`、Theme 映射和数量；重放不得刷新或重算这些结果。
_Avoid_: 在单条 Theme 上设置批次 ID 唯一约束、业务数据失败后保留回执、修改成功回执

**Theme 发布载荷哈希（Theme Publication Payload Hash）**:
对完整 Theme 发布请求体按 RFC 8785 规范化后计算的小写十六进制 SHA-256，用于批次幂等重放和内容冲突检测。哈希只覆盖调用方提交的批次身份、分析窗口和 Theme 内容，不包含认证信息、请求 ID、服务端发布时间或响应字段；由 Data 计算并返回，调用方不提交。

**Theme 发布规范数组顺序（Theme Publication Canonical Array Order）**:
Theme 发布请求只有一种合法数组表示：`themes` 按批次内 Theme Key 升序，`chain_nodes` 按规范化小写节点 UUID 升序，`events` 按规范化小写 Event UUID 升序。UUID 必须使用标准小写字符串，同一数组不得重复键或 ID。Data 校验但不重排数组，通过结构与顺序校验后才按原顺序计算 canonical hash。
_Avoid_: 大小写混合 UUID、重复关联、服务端静默重排、同一语义存在多种合法数组顺序

**Theme 发布 V1（Theme Publication V1）**:
批次级字段为 `analysis_batch_id`、`analysis_as_of`、`window_start`、`window_end` 和
`themes`。每个 Theme 使用显式结论、影响强度、传导阶段、投资指引、时间范围、可空
摘要、Theme Impact 与 Event 关联字段。合同严格拒绝旧字段和未知字段。
_Avoid_: `subject_entity_id`、Theme `name`、`impact_level`、`trading_direction`、
`transmission_path`、`next_checkpoint`、`market_confirmation_summary`

**主题影响（Theme Impact）**:
Research Theme 与受影响 Chain Node 的不可变关系，保存角色、方向、可空摘要和稳定
展示顺序。所有 Impact 平等，不存在 subject、primary 或主要影响节点；当前目标类型
只允许有效 Chain Node。一个 Theme 至少有一个 Impact。
_Avoid_: Company/Concept 等其他目标类型、`is_primary`、把展示顺序解释为影响优先级

**产品可见 Theme（Product-visible Research Theme）**:
属于成功 Theme Publication Batch 且满足查询窗口的 Theme。Theme 发布与 Reason Tree
发布是两个独立同步事务；Theme 可以在没有 Tree 回执时继续可见。
_Avoid_: 用 Tree 发布状态过滤首页 Theme、要求 Theme 与 Tree 同时可见

**推理树（Reason Tree）**:
一个 Theme 在一条 Industry Chain 内的不可变线性推导链路。同一 Theme 与同一
Industry Chain 最多一棵 Tree；一棵 Tree 可以解释该链内一个或多个 Theme Impact，
也可以包含只承担传导上下文的 Chain Node。Tree 没有中心节点或主要影响节点。
_Avoid_: Research Anchor、中心节点、任意图、分叉、循环、用 Tree `display_order`
表达影响优先级

**Reason Tree 发布集合（Reason Tree Publication Set）**:
`POST /api/data/v1/research-reasoning-tree-imports` 对一个已发布 Theme 的完整 Tree
集合执行同步、原子、幂等发布。`theme_id` 是集合幂等身份；请求至少包含一棵 Tree，
每棵 Tree 至少命中一个 Theme Impact，全部 Tree 的命中并集覆盖父 Theme 的所有
Impact。发布主体必须与父 Theme 回执一致。
_Avoid_: 单树分批发布、空集合回执、部分覆盖、第二套幂等键

**Reason Tree 身份与回执（Reason Tree Identity and Receipt）**:
Tree 身份由 `theme_id + NUL + industry_chain_entity_id` 确定性生成。每个 Theme
最多一条 Tree 集合回执，保存发布主体、payload hash、Industry Chain 到 Tree ID
映射、写入计数和首次发布时间；回执与全部 Tree 子记录同事务提交。
_Avoid_: 调用方提交 Tree ID、原地覆盖、每棵 Tree 单独回执

**Reason Tree 节点（Reason Tree Node）**:
Tree 的有序 Chain Node 快照。`position` 从 1 连续排列并且是唯一路径顺序；一个
节点在 Tree 内唯一。每个节点必须是该 Tree Industry Chain 的 active/approved
成员。首节点全部 `incoming_*` 为空；后续节点必须保存传导标题、机制和成立条件。
可空正式 Graph Edge 必须 active/approved、属于同一链且端点匹配；为空表示分析推断。
_Avoid_: 持久化结果节点标记、首节点伪造入边、从 Tree 回写正式图谱

**Variable Signal 展示快照（Variable Signal Display Snapshot）**:
每个 Tree 节点拥有 1..5 个按 `display_order` 排列的弱外部引用快照，恰好一个
`primary` 且顺序为 1；其余角色为 `supporting` 或 `contradicting`。同一分析批次
复用同一 key 时，方向和展示摘要必须一致。该展示快照自身不建设或替代 Event-native
Signal 事实表，也不验证其 Entity、Evidence、Metric 或观测语义；Phase One 的正式
Event-native Variable Signal 由下述独立领域对象与 API 管理。
_Avoid_: Variable Signal 外键、事实读取 API、从展示文本推断研究语义

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
Event Entity Link、Variable Signal 和 Direct Impact Assertion 各自拥有的领域审核
状态：`accepted` 可供下游使用；`pending_review` 正在等待自动 AI 复核或可恢复处理；
`needs_reanalysis` 需要补充 Evidence、重新提取或解析；`quarantined` 表示自动重试
预算耗尽后长期隔离；`rejected` 保留稳定拒绝原因；`superseded` 表示已有新候选替代但
旧记录继续审计。上游未 accepted 时，下游不得 accepted。
_Avoid_: 用 Submission 单一状态覆盖全部候选、把 Pending 当正式 ABox、等待人工 UI、
重试耗尽后自动接受、删除拒绝记录

**Acceptance Policy**:
Data Service 用于把已通过基本合同校验的语义候选路由到审核状态的版本化策略。它组合
确定性门禁、逐字 Evidence、独立语义审核、冲突/歧义与经 Golden Fixture 校准的分对象
Confidence 阈值；Evidence Grade 或模型 Confidence 都不能单独决定接受或拒绝。
_Avoid_: 全局永久阈值、未经校准的置信常量、Grade A 自动接受、Grade C 自动拒绝

**独立语义复核（Independent Semantic Review）**:
Candidate Generator 之后的独立 AI 调用，使用独立 Prompt/版本，仅接收候选、Event
Evidence、Ontology Context 和校验清单，结构化输出 `pass | fail | indeterminate`。
它可以与 Generator 使用同一基础 LLM，但不读取 Generator 的自由推理过程，也不能直接
写 accepted；最终状态始终由 Data Service 的确定性门禁和 Acceptance Policy 决定。
_Avoid_: Generator 自我确认、Reviewer 直接改领域状态、开放式多 Agent 辩论

**Event Semantic Submission**:
Data Service 对一个正式 Event 的一次语义提交、确定性校验、独立 AI Review Result、
Acceptance Policy 裁决和产物血缘记录，与 AgentRun 的一个 Agent Execution 一对一。
它保存外部执行身份及 Agent/Ontology/Rule/Prompt/Model 版本快照，但不复制 AgentRun 的
runtime 状态、调度重试或执行错误；重新分析创建新 Submission 并 supersede 旧
Submission。
_Avoid_: Agent Execution 副本、Theme Analysis Batch、原地覆盖重新分析、跨 Event 批次

**Event Semantic Context Lease**:
Data Service 为 AgentRun 已领取的一个 Event Semantic Work Item 提供的短时数据快照授权。
它固定 Event、Evidence、Ontology、Variable、Rule 和可选 superseded Submission 边界，
并在创建事务中持久化 Entity / EntityRelation 在内的完整 Context snapshot。Lease 以
Agent Execution ID 为唯一恢复身份；同一执行可精确续期，但只能复用原 snapshot，不能
刷新实时数据。对应 Submission 终结后 Lease 被消费。它不是任务、队列或 Agent 执行租约；
调度、失败恢复和重试始终属于 AgentRun。
_Avoid_: Reanalysis Task、Agent Work Item、在 Data 中调度模型调用、无限续租

**Reason Tree Event 关联（Reason Tree Event Association）**:
Tree 可以从父 Theme Event 集合选择零个或多个正式 Event，并保存角色与稳定展示
顺序；不复制 Event 正文或证据摘要。Theme Event 与 Tree Event 都只接受
`confirmed + verified` 的正式 Event 事实。
_Avoid_: Tree 扩展父 Theme Event 边界、要求特定角色组合、复制 Event 文本

**Reason Tree 读取边界（Reason Tree Read Boundary）**:
列表使用 `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees`，详情使用
`GET /api/data/v1/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}`。
Theme 缺失、无 Tree 回执、Tree 缺失分别返回稳定 `404`；回执存在但投影无法完整
重建返回 `500 RESEARCH_REASONING_TREE_INVARIANT_VIOLATION`。读取按服务端顺序返回
Theme Impact IDs，消费者仅通过 ID 相交判断节点是否为 Theme Impact。
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

Data 业务代码必须收敛到 `analyse-data-service/backend/`：

```text
cmd/          process and maintenance entrypoints
usecase/      import, query, seed and projection orchestration
domain/       Data-owned rules and models
repositories/ persistence ports and implementations
adapters/     PostgreSQL, Neo4j, migration and inbound/outbound adapters
transport/    Data REST routes, handlers, middleware and DTOs
config/       Data-only runtime configuration
```

`analyse-data-service/backend/migrations/` 与 `analyse-data-service/backend/data/` 是 Data 的统一事实资产，可以保留为 Backend 根资产，但不得被 BFF 直接读取。
