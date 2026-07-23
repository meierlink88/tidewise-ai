# Data Context

## Purpose

Data Domain Service 是当前唯一 Domain Service，负责稳定的数据事实、领域规则、持久化、受控导入和查询 API。

## Owns

- Entity、产业链节点及关系、Benchmark、Index 等主数据。
- 正式 Event、被 Event 引用的轻量 Evidence Record 及其证据关联。
- Research Theme、Research Anchor 及其关联数据。
- PostgreSQL schema、migration、repository 和 Neo4j 可重建投影。
- AgentRun 使用的 Event Publication API、自然身份收敛、receipt 和事务规则。
- 面向 Miniapp/Admin Application Backend Service 的版本化 REST API。

## Does Not Own

- Miniapp 或 Admin Portal 的页面 DTO、交互状态和展示逻辑。
- User、Auth、Payment、Subscription 等未来独立领域。
- Source 主数据、完整原始 Artifact、数据采集 connector、parser、采集 prompt 或采集调度执行。
- Agent 的模型推理和工作流运行。

## External Agent Boundary

外部 AgentRun 拥有 Source 主数据、采集调度、采集执行、完整原始 Artifact 和 Event 提取工作流。AgentRun 只能按照 Data 定义的版本化 Event Publication 合同提交已提取 Event 及其证据引用，不直接访问 Data 数据库；Data 不维护 Source Catalog，也不接纳未产生正式 Event 的采集 Artifact。

Tidewise 中遗留的 Source Catalog、采集调度与采集运行控制面通过保留 `raw_documents` 来源快照的 forward migration 物理移除；该收敛不得删除历史 Event、Evidence Record 或既有证据关联。
Data 的 AgentRun Source Metadata、Admin Source Catalog 查询，以及 Admin Portal 对应代理接口、Client、Repository、Seed 和专属测试一并移除，不保留静态兼容路由；AgentRun 仅使用自身 Source Catalog。

## Language

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

**Event Publication Batch**:
AgentRun 将一至十个已完成提取与审核、状态固定为 `confirmed + verified` 的原子 Event，连同其共享 Event Evidence Record、证据关联、Tag、Review 和提取血缘，按照 Data 定义的严格同步合同整批原子提交为正式事实；候选、未验证或拒绝 Event 不进入 Data，任一成员失败时整批不可见。
每个 Event 独立提交必填的 `review_id`、`evidence_grade` 和非空 `reasons`；V2 不重复提交审核决定、Event/Fact 状态或组件版本，Data 统一写入 `confirmed + verified`。
V2 在批次顶层提交去重后的 `raw_documents`，各 Event 通过 `artifact_id` 引用共享证据。每个 Event 至少引用一个已声明 Artifact；每个顶层 Artifact 也必须至少被一个 Event 引用，未知或重复 Artifact 身份均使整批失败。
Data 在写事务前返回所有当前可确定的合同、枚举、Tag 和引用错误；自然身份内容冲突单独返回冲突错误。任一错误均阻止整个批次和 Receipt 落库，不允许部分成功。
_Avoid_: 独立 Raw Document 导入、Agent 直写数据库、先存全文后补 Event

**Event Import Receipt**:
Data 为每次成功 Event Publication Batch 生成的不可变审计凭证，记录调用主体、`package_id`、正式事实身份、`extractor_execution_id`、`extractor_agent_version`、每个 Artifact 对应的 `collector_execution_id` 和导入时间。以上执行血缘均为必填；Prompt、模型和 Profile 版本仍由 AgentRun 保存。Receipt 不承担请求幂等、重放判断或异步状态查询职责；失败事务不生成 Receipt。
`package_id` 只是 AgentRun 提供的审计关联编号，不唯一且不参与事实复用；相同 package 可以产生多个成功 Receipt，每次成功调用均由 Data 生成新的 `receipt_id`。
Event Publication 必须通过内部 Bearer service token 鉴权；Token 只存在于运行环境，不进入数据库。Data 从凭据解析稳定 `caller_subject` 写入 Receipt，用于服务级审计，与 Source、采集通道或 Artifact 来源无关。
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
获准发布 Research Theme 的稳定内部服务身份，由 Data 从认证上下文解析，不由请求声明。首次成功发布时该主体取得批次所有权；后续幂等重放必须来自同一主体。主体身份独立于可轮换 token，审计只保存主体 ID，不保存凭据。
_Avoid_: 在请求体中声明发布者、以 token 字符串作为长期身份、其他服务接管已发布批次

**Theme 发布未知结果恢复（Theme Publication Unknown-outcome Recovery）**:
同步 Theme 发布请求超时后，发布器以完全相同的 Analysis Batch ID 和发布内容重试 POST。首次事务已成功时返回原结果并标记重放，未成功时正常执行，内容变化时返回冲突。本期不提供状态查询、轮询或异步任务接口。

**Theme 发布回执（Theme Publication Receipt）**:
一个成功 Research Theme Publication Batch 的不可变技术回执，也是 Analysis Batch ID 全局唯一性、发布主体所有权、payload hash、首次 Theme IDs、发布时间和写入数量的持久化依据。每批只有一条回执；回执、全部 Theme 及其关联事实必须在同一事务中提交或回滚。`replayed` 表示当前响应是否来自重放，不是回执中固化的首次结果。
V1 HTTP 成功结果使用 `theme_ids_by_key` 对象返回完整的 Theme Key 到 Theme UUID 映射；`counts` 仅包含 `themes`、`chain_node_associations`、`event_associations` 和 `receipts`。首次成功返回 `201 Created` 和 `replayed: false`，同主体同载荷重放返回 `200 OK` 和 `replayed: true`。两种结果都返回首次成功时的 `receipt_id`、`payload_hash`、`published_at`、`imported_at`、Theme 映射和数量；重放不得刷新或重算这些结果。
_Avoid_: 在单条 Theme 上设置批次 ID 唯一约束、业务数据失败后保留回执、修改成功回执

**Theme 发布载荷哈希（Theme Publication Payload Hash）**:
对完整 Theme 发布请求体按 RFC 8785 规范化后计算的小写十六进制 SHA-256，用于批次幂等重放和内容冲突检测。哈希只覆盖调用方提交的批次身份、分析窗口和 Theme 内容，不包含认证信息、请求 ID、服务端发布时间或响应字段；由 Data 计算并返回，调用方不提交。

**Theme 发布规范数组顺序（Theme Publication Canonical Array Order）**:
Theme 发布请求只有一种合法数组表示：`themes` 按批次内 Theme Key 升序，`chain_nodes` 按规范化小写节点 UUID 升序，`events` 按规范化小写 Event UUID 升序。UUID 必须使用标准小写字符串，同一数组不得重复键或 ID。Data 校验但不重排数组，通过结构与顺序校验后才按原顺序计算 canonical hash。
_Avoid_: 大小写混合 UUID、重复关联、服务端静默重排、同一语义存在多种合法数组顺序

**Theme 发布 V1（Theme Publication V1）**:
批次级字段仅包含 `analysis_batch_id`、`window_start`、`window_end` 和 `themes`。每个 Theme 仅包含 `theme_key`、`name`、`one_line_conclusion`、`impact_level`、`transmission_path`、`trading_direction`、`transmission_stage`、`next_checkpoint`、`market_confirmation_summary`、`chain_nodes` 和 `events`。V1 使用严格字段合同，出现未声明字段时整个批次拒绝。
_Avoid_: 将完整分析报告直接作为发布请求、在 V1 中携带分析侧内部字段或指数关联

**影响等级（Impact Level）**:
Research Theme 对产品用户呈现的影响程度，允许值为 `high`（高影响）、`focus`（重点关注）、`watch`（持续观察），当前是 Theme 唯一保存和展示的强弱判断。该值由分析侧明确提交；Data 不从结论可信度、Event 数量或其他字段推断。分析侧的结论可信度 `confidence` 不属于当前 Theme 发布合同，发布时忽略且不得映射为 Impact Level。
_Avoid_: 把结论可信度和影响程度混为同一字段

**主题产业链节点关联（Theme Chain Node Association）**:
分析 Agent 对单个产业链节点在 Research Theme 中所承担角色的明确判断，由节点、角色和关联依据共同组成。角色仅允许 `driver`、`beneficiary`、`constraint`、`exposure`；关联依据必须说明该节点为何与 Theme 相关。每个 Theme 至少包含一个产业链节点关联。Data 只校验关联事实，不从整体传导路径或既有产业链关系自动推断。
_Avoid_: 没有产业链节点的 Theme、仅提交节点 ID、同时提交 ID 数组和关联对象、由 Data 补写分析语义

**研究锚点（Research Anchor）**:
隶属于一个 Research Theme、并以该 Theme 已关联的一个 Chain Node 为中心的不可变推理树研究快照。它表达本批次中针对该节点的结论、正反证据、传导节点及后续检查点，不是 Chain Node 主数据或运行时动态推理。Anchor 不复制中心节点名称；读取时使用中心 Chain Node 的当前规范主数据名称。Anchor 不再按 `anchor_type` 分类，也不使用 `importance` 表达可见性或优先级。
_Avoid_: 独立于 Theme 的 Anchor、把 Anchor 当作 Chain Node 类型、使用旧类型或重要性枚举区分 Anchor、在 Anchor 中复制节点名称、页面打开时生成 Anchor

**Anchor 净方向摘要（Anchor Net Direction Summary）**:
一句自然语言文本，概括某棵 Research Anchor 当前事实、支持证据和反证合并后的总体指向，例如“风险上升，行情未确认”。它由 AI 分析师明确生成，不收窄为利多/利空枚举，也不等同于用户投研行动导向的 Trading Direction。
_Avoid_: 由 Data 或 BFF 合成净方向、用 Trading Direction 替代净方向、强制转换为利多/利空枚举

**Anchor 事实汇总（Anchor Fact Summary）**:
一句由 AI 分析师生成的非空文本，用于概括当前 Research Anchor 所引用的原子 Event 事实组合，但不替代 Event 本身。页面的事实明细和数量分别来自已关联 Event 的标题/摘要/时间与去重计数，不另建不可追溯的事实文本数组。
_Avoid_: 用 `fact_statements` 复制 Event 事实、从 Markdown 反向生成事实列表、由 BFF 合成事实汇总

**Research Anchor 完整覆盖（Complete Anchor Coverage）**:
一个 Theme 的推理树集合必须对该 Theme 的每个 Theme Chain Node Association 恰好提供一个以该节点为中心的 Research Anchor，中心节点集合不得缺失、新增或重复。任一 Anchor 缺失或无效时，该 Theme 的 Anchor 集合整体不发布，并保持无可见推理树；这不影响已发布 Theme 按独立首页合同继续可见。
_Avoid_: 部分 Anchor 可见、额外 Anchor Tab、一个中心节点对应多个 Anchor

**产品可见 Theme（Product-visible Research Theme）**:
属于成功 Research Theme Publication Batch，并满足首页时间窗口与最新批次查询条件的 Theme。首页 Theme 列表不读取 Research Anchor Publication Receipt，不判断推理树是否存在，也不增加 `has_reasoning_tree`；Theme/Event 计数完全由 Theme 查询合同决定。推理树属于进入影响路径页后读取的独立子资源。
_Avoid_: 用 Anchor 发布状态过滤首页 Theme、把待补树 Theme 隐藏为技术暂存、在首页增加推理树存在性字段、Anchor 失败时删除或隐藏已成功导入的 Theme

**Research Anchor 发布回执（Research Anchor Publication Receipt）**:
一个 Theme 的完整 Research Anchor 集合成功发布后产生的不可变技术回执，同时是该 Theme 的 Anchor 发布幂等性和可见性依据。它记录 `receipt_id`、`theme_id`、发布服务主体、payload hash、完整的中心节点到 Anchor ID 映射、Anchor/Event 关联/路径节点/回执计数、Anchor 集合正式可见时间和导入时间。回执、全部 Anchor 及其 Event/路径关联必须在同一事务中提交或回滚；只有存在成功回执的 Theme 才拥有可见推理树。同一 `theme_id` 成功发布后不得使用不同内容覆盖。幂等重放返回首次结果，`replayed` 只是本次 HTTP 响应状态。
_Avoid_: 每个 Anchor 独立发布、部分写入后生成回执、在 Anchor 行重复保存 Theme 分析批次和逐行发布时间

**Research Anchor 发布幂等身份（Research Anchor Publication Idempotency Identity）**:
所属 Research Theme 的 `theme_id`，也是 Anchor Import V1 唯一幂等身份，不再增加请求头或请求体幂等键。同一 Theme 和相同 canonical payload 是幂等重放；同一 Theme 成功发布后使用不同 payload 属于冲突。校验失败或事务回滚不会占用该身份，成功内容的修订必须通过新分析批次和新 Theme 发布。
_Avoid_: 第二套 Anchor 幂等键、原地覆盖已发布 Anchor、为校验失败请求生成成功回执

**Research Anchor 确定性请求（Deterministic Research Anchor Request）**:
用于保证 Anchor Publication canonical hash 跨运行时稳定的唯一合法数组表示。`anchors` 按小写中心节点 UUID 升序，每棵 Anchor 的 `events` 按小写 Event UUID 升序，而 `path_nodes` 保持不可重排的业务传导顺序。所有 UUID 使用标准小写格式。Data 校验而不重排，通过后才对原始合法表示计算 canonical hash。
_Avoid_: 服务端静默重排请求、将语义路径按 UUID 排序、混用大小写 UUID、同一语义存在多种合法数组表示

**Research Anchor Import V1**:
通过 `POST /api/data/v1/research-anchor-imports` 对一个已发布 Theme 的完整 Anchor 集合执行同步、原子、幂等发布的内部合同。顶层严格只有 `theme_id` 和 `anchors`；`theme_id` 同时表达父 Theme、整体发布边界和幂等身份。每棵 Anchor 严格只提交中心节点、独立结论、事实汇总、净方向、交易指向、下一检查点、Event 证据和有序路径节点。
_Avoid_: 将 Anchor 嵌入 Theme Import V1、按单棵 Anchor 分批发布、增加第二套批次或幂等字段、接收未声明字段、由调用方提交 Anchor ID/名称/时间或历史类型字段

**Research Anchor Import 错误定位（Research Anchor Import Error Location）**:
遵循 Theme Import V1 错误分层的可操作失败信息。结构与合同错误使用 `400`，缺失或越界引用使用 `422`，payload 或发布主体冲突使用 `409`。可定位错误统一携带 `center_chain_node_id`、`path` 和 `reference`，使发布器能直接定位具体 Anchor 和字段。
一个请求同时存在多个错误时，只按请求的确定性遍历顺序返回第一个错误，不聚合错误列表；发布器修正后使用完整请求重新提交，整批原子失败规则不变。
_Avoid_: 只返回黑盒失败文本、将不存在与越界引用混为内部错误、在失败响应中返回部分写入结果、返回顺序不稳定的错误集合

**Research Anchor 未知结果恢复（Research Anchor Unknown-outcome Recovery）**:
同步 Anchor Import POST 在客户端未收到响应时的唯一恢复方式：使用原 `theme_id` 和完全相同的 payload 重试 POST。已成功则返回首次 Receipt，未成功则执行新事务，内容改变则冲突。
正式 Theme 的 Anchor 可以在更新分析批次已经发布后延迟补发；只要父 Theme 及其发布回执仍存在且主体一致，补发没有人为截止时间。晚到的 Anchor 不改变父 Theme 的发布时间，也不把旧 Theme 提升为最新批次。
_Avoid_: 增加状态查询端点、将小批量发布改为异步 job、为失败请求持久化 unknown 占位、响应丢失后修改 payload 重试、因 Theme 不是最新批次而拒绝补树

**Research Anchor 身份（Research Anchor Identity）**:
一棵 Research Anchor 在单个 Theme 内的身份由 `theme_id` 与 `center_chain_node_id` 共同决定，不另设 `anchor_key`。调用方不提交 Anchor UUID；Data 使用冻结命名空间 `f219ded4-fc65-5948-9e28-c1cdb6a8288e`，对标准小写 `theme_id + NUL + center_chain_node_id` 生成确定性 UUIDv5 `anchor_id`。
_Avoid_: 调用方生成 Anchor UUID、维护第二套 Anchor Key、使用 Anchor ID 跨 Theme 合并研究快照

**Research Anchor 展示顺序（Research Anchor Presentation Order）**:
同一 Theme 下多个 Research Anchor 在读取端的稳定顺序，由 Data 按中心 Chain Node 的当前规范名称使用 PostgreSQL `COLLATE "C"` 升序排列，名称相同时再按中心节点 UUID 升序。V1 不保存 `display_order`、重要性或 AI 排名；BFF 和小程序保持 Data 顺序。
_Avoid_: 使用 Theme Import 的 canonical UUID 顺序当作产品顺序、由 BFF 或前端二次重排、复活旧 importance 字段

**Research Anchor 发布主体（Research Anchor Publisher Subject）**:
已成功发布 Research Theme 的同一稳定内部服务身份，也是该 Theme 的 Anchor 集合唯一允许的发布和重放主体。Data 从认证上下文解析主体并与 Theme Publication Receipt 比对；请求不得声明发布者，token 轮换不改变稳定主体归属。
没有 Theme Publication Receipt 的历史 Theme 不具备可核验发布主体，不能接收 Research Anchor 发布，也不存在本地环境或历史数据绕过。
_Avoid_: 其他服务接管 Theme 的 Anchor 发布、在请求体声明发布者、使用 token 字符串作为长期主体身份、为无发布回执的历史 Theme 补发 Anchor

**Anchor 一句话结论（Anchor One-line Conclusion）**:
围绕 Research Anchor 中心 Chain Node 得出的必填、非空研究结论，属于当次 Anchor 快照。它与 Theme 的总体一句话结论层级不同，由 AI 分析师生成，Data 只校验完整性并原样保存。
_Avoid_: 复制 Theme 结论作为默认值、由 Data/BFF/前端从证据或路径临时生成

**Anchor 交易指向（Anchor Trading Direction）**:
围绕 Research Anchor 中心 Chain Node 提出的必填、非空研究与交易映射，可表达研究优先级、受益或承压方向以及应重点关注的环节。它与 Theme 级交易指向和表达客观变化的 Anchor Net Direction Summary 层级、职责不同，由 AI 分析师生成，Data 原样保存。
_Avoid_: 复用 Theme 交易指向补缺、将客观变化方向当作交易建议、由 Data/BFF/前端推断

**Anchor 下一检查点（Anchor Next Checkpoint）**:
围绕 Research Anchor 中心 Chain Node 定义的必填、非空后续验证项，明确下一步需要跟踪的指标、事实或条件。它属于 Anchor 快照，与 Theme 级检查点层级不同，由 AI 分析师生成，Data 原样保存。
_Avoid_: 复用 Theme 检查点补缺、由 Data 拼接条件或设置默认文本

**Anchor 指数关联（Anchor Index Association）**:
未纳入 Research Anchor V1 的能力。V1 导入、存储写入、Theme 推理树详情和 Miniapp DTO 都不包含 Anchor 指数数据；历史空的 `research_anchor_indices` 在新 migration 的空表断言通过后删除且不重建。
_Avoid_: 为兼容空闲历史表而增加 `indices` 请求字段、强制指数主数据、在 BFF 临时补齐指数关联

**Theme 推理树集合读取（Theme Reasoning Trees Read）**:
通过 `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees` 返回某个 Theme 已发布的完整 Anchor Tab 摘要集合，并通过 `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees/{anchor_id}` 按需返回一棵完整推理树。一个 Theme 可以对应多个 Anchor，每个 Anchor 是一条独立推理树。列表不嵌入现有 Theme Detail，也不分页；详情必须校验 Anchor 属于路径中的 Theme。两个端点不接收查询时间窗口，完全按 `theme_id` 读取已发布不可变快照，即使 Theme 已离开首页窗口仍可访问。Theme 缺失、待补树或单树不存在分别使用明确 `404`，不返回合法空数组。首页始终保留影响路径入口；Theme 存在但待补树时由推理树页面解释为“影响路径暂未生成”。
_Avoid_: 为推理树扩张现有 Theme Detail、由 BFF 为一次请求扇出多个 Anchor 查询、使用旧独立 Anchor 详情代替 Theme 子资源、将整个 Theme 误称为一棵推理树、用 `200` + 空数组掩盖待发布或数据破坏

**推理树读取不变量破坏（Reasoning Tree Read Invariant Violation）**:
成功 Research Anchor Publication Receipt 存在，但 Anchor 映射、数量、Event 证据或路径节点无法完整重建时的数据损坏状态。Data 必须返回 `500 RESEARCH_REASONING_TREE_INVARIANT_VIOLATION`，不把它解释为尚未生成、资源不存在、合法空集合或部分成功。
_Avoid_: 用 `404` 隐藏已发布数据损坏、返回部分 Anchor Tab、忽略回执与事实不一致、让 BFF 猜测缺失数据

**Research Anchor 推理树读取模型（Research Anchor Reasoning Tree Read Model）**:
单树详情端点返回的一棵完整已发布视图，包含 Anchor ID、当前中心节点主数据身份与名称、Anchor 结论/事实汇总/净方向/交易指向/下一检查点、全部 Event 证据及其主数据展示字段，以及带当前节点名称的有序路径节点。它不暴露技术 Receipt 或历史 Anchor 字段，BFF 可用一次对应 Data 请求驱动当前 Tab。
_Avoid_: 在 BFF 拉取第二批 Data 补名称/事件、返回路径位置与数组顺序两套事实、暴露 Receipt 时间或重复证据汇总

**Anchor 传导节点（Anchor Path Node）**:
某棵 Research Anchor 的单条有序产业链传导路径中，被选中并附带当次影响判断的 Chain Node 快照项。每个节点的快照必须由受控的变化方向、简短变化描述和影响说明组成。一条合法路径至少包含两个不同节点。节点数组顺序就是传导顺序；第一个节点没有入边机制，其余节点都携带从前一节点传入的非空机制说明。每棵 Anchor 的传导节点必须包含其中心 Chain Node 且恰好一次；路径内任何 Chain Node 都只能出现一次，因此不允许任何重复节点或循环。中心节点的变化和说明也必须来自已发布快照，不由 BFF 或前端补写。除中心节点外，路径可以引用未被 Theme 选为 Anchor 的既有 Chain Node 作为传导上下文；该引用不会新增 Theme 关联或 Anchor。V1 不表达分叉、多路径、循环或通用图结构。
_Avoid_: 单节点路径、路径中缺少中心节点、任何重复节点、循环路径、由页面临时合成中心节点

**Anchor 节点变化方向（Anchor Node Change Direction）**:
Anchor Path Node 在当次研究快照中所观察或判断的变化方向，只允许 `increase`、`decrease`、`mixed`、`unchanged` 或 `uncertain`。它表达“上升、下降或未确认”，不等同于投资上的利好或利空；具体变化和研究含义分别由 `change_summary` 和 `impact_summary` 表达。
_Avoid_: 用 `positive`/`negative` 混合数值变化与投资判断、由 BFF 从文本推断变化方向

**Anchor 辅助路径节点（Anchor Supporting Path Node）**:
在 Anchor Chain Transmission Path 中用于说明上下游背景或影响去向，但未被当前 Theme 选为 Anchor 中心的既有 Chain Node。它拥有当次快照的变化和说明，但不会因此获得 Anchor Tab 或变成 Theme Chain Node Association。
_Avoid_: 为所有路径上下文节点强制创建 Anchor、由 Data 自动扩充 Theme 产业链节点

**Anchor 产业链传导路径（Anchor Chain Transmission Path）**:
一棵 Research Anchor 中由 Anchor Path Node 和 Anchor Transmission Edge 组成的唯一有序产业链路径，用于表达本批次研究判断中影响如何经过中心节点传导。该结构是 Anchor 传导语义的唯一事实表达，不另外保存自然语言 `transmission_path`。“推理树”的分支来自支持证据与反证；V1 的产业链路径本身不分叉。
_Avoid_: 在 V1 中建模任意图、多条并行路径、同时保存可能冲突的 Anchor 路径文本、由读取端自行排序节点

**Anchor 传导连线（Anchor Transmission Edge）**:
一棵 Research Anchor 中相邻两个 Anchor Path Node 之间的当次研究传导判断。V1 不使用独立 Edge 数组；连线机制由后一个 Path Node 的 `incoming_transmission_mechanism` 承载。它属于 Anchor 快照，不要求在 `chain_node_relations` 中已有对应关系；Data 不以正式图谱是否存在或完整作为推理树的发布条件，Anchor 发布也不会新增或修改正式关系主数据。每条连线必须包含非空的传导机制说明，由 AI 分析师发布，Data 只校验存在性和非空性。V1 不将连线标记为“正式关系”或“推断关系”，页面也不展示这组来源标签。
_Avoid_: 因正式关系缺失而拒绝研究路径、把 Anchor 连线当作正式产业链关系、未验证却标记“正式关系”、从 Anchor 导入回写关系主数据

**整体传导路径（Causal Chain）**:
分析侧对 Research Theme 端到端影响路径的命名。发布器将其一对一重命名为 `transmission_path`，不拼接、推断或改写；发布合同只保留 `transmission_path`。整体路径与单个节点的角色关联是两类互补事实，不能相互替代。

**传导路径（Transmission Path）**:
Research Theme 在 Tidewise 发布边界中的整体影响路径。Data 只校验其存在且非空，不解析或重写路径内容。

**主题 Event 证据关联（Theme Event Evidence Association）**:
一个 Event 对 Research Theme 中具体判断所承担的证据角色，由 Event、证据角色和被支持或反驳的判断共同组成。角色仅允许 `driver`、`supporting`、`contradicting`、`context`，且每个 Theme 至少有一个 `driver` Event。Data 只校验证据事实，不从市场确认、结论文本或其他字段推断。
_Avoid_: 仅提交 Event ID、同时提交 ID 数组和证据对象、没有 driver Event 的 Theme

**Anchor Event 证据（Anchor Event Evidence）**:
一个已属于 Research Theme 的 Event 在其中某棵 Research Anchor 中承担的具体证据角色和与该 Anchor 相关的证据摘要。一棵 Anchor 只能从所属 Theme 已发布的 Theme Event Evidence Association 中选择 Event，不得借 Anchor 发布引入 Theme 证据边界外的 Event。每棵 Anchor 必须至少有一个 `driver` Event；`supporting`、`contradicting` 和 `context` 可以为空，没有反证时也是需要明确呈现的合法研究状态。同一 Event 在同一 Anchor 中恰好最多出现一次并承担一个证据角色。`evidence_summary` 由 AI 分析师提交并必须非空；Event 标题、摘要和时间仍由 Event 主数据提供。
_Avoid_: Anchor 引用 Theme 之外的 Event、没有 driver Event 的 Anchor、同一 Event 在一棵 Anchor 中承担多个角色、把无反证视为导入错误、由 Anchor Import 扩展 Theme Event 集合、从 Markdown 临时生成证据

**Anchor 支持与反证汇总（Anchor Support and Counter Summaries）**:
AI 分析师基于一棵 Research Anchor 的完整证据形成的两项结论性快照。`support_summary` 必填并表达整体证据目前支持什么；`counter_summary` 只在存在 `contradicting` Event 时提交，没有反证时为 `null`。它们不等于具体 Event 清单，也不得由发布器、Data、BFF 或前端按 `evidence_role` 拼接生成。Event 角色和 `evidence_summary` 继续承担单条事实追溯，页面在原子事件清单中展示角色标签。
_Avoid_: 用 Event 文本拼接 Anchor 汇总、把汇总复制成 Event 数组、没有反证时伪造反证结论、在 BFF 或前端重新推理

**Anchor 原子事件汇总（Anchor Atomic Event Rollup）**:
该 Research Anchor 关联的全部去重 Event 事实视图，覆盖 `driver`、`supporting`、`contradicting` 和 `context` 四种角色。它复用同一份 Anchor Event Evidence Association 和 Event 主数据，与 Anchor 级支持/反证汇总提供不同阅读视角，不创建第二套事实或关联。读取时按 `event_time` 从早到晚排列，时间相同时以 `event_id` 升序稳定排序，缺失时间的 Event 放在最后。
_Avoid_: 只展示 driver Event、排除反证或上下文 Event、为页面汇总复制存储 Event、按行数而非去重 `event_id` 计数、由 BFF 或前端重新排序

**Research Anchor 读取边界（Research Anchor Read Boundary）**:
Research Anchor 只能作为所属 Research Theme 下的推理树子资源读取。旧的独立 Anchor 列表和详情 API 基于已经废弃的 `anchor_type`、`importance` 语义，尚未正式投入使用，因此直接删除而不提供兼容层。新的列表读取返回 Theme 下全部推理树摘要，单树详情读取必须同时使用 `theme_id` 和 `anchor_id` 并校验归属关系。
_Avoid_: 恢复全局 Anchor 列表、仅凭 `anchor_id` 跨 Theme 读取、同时维护旧 Anchor DTO 与新推理树 DTO、为未使用接口建立兼容层

**旧 Anchor 结构替换（Legacy Anchor Schema Replacement）**:
旧 Anchor 四张表尚未正式投入使用，当前本地均为空。新 forward migration 必须先断言四表为空，非空则停止；断言通过后才可删除旧结构并以原表名重建新 Research Anchor、Event Evidence 和 Path Node 结构，同时新增 Theme 级发布回执。V1 删除空的 `research_anchor_indices` 且不重建，不迁移或伪造旧语义数据。
_Avoid_: 修改历史 migration、静默删除旧数据、猜测转换旧 Anchor、同时保留新旧字段、为当前 V1 重建 Anchor 指数表

**Research Anchor 发布回执（Research Anchor Import Receipt）**:
一个 Research Theme 的完整 Anchor 集合首次成功发布后形成的不可变技术事实，存放于 `research_anchor_import_receipts`。每个 Theme 最多一条回执；回执保存发布主体、payload hash、中心节点到 Anchor ID 的映射、写入计数以及首次发布时间和导入时间。所有 Anchor 必须通过 `import_receipt_id` 关联同一回执，回执与业务数据在一个事务内提交。
_Avoid_: 每棵 Anchor 单独回执、修改或删除回执、保存 token、重放时生成新时间或新 ID、让没有回执的 Anchor 对产品可见

**Research Anchor 主表（Research Anchor Record）**:
Research Anchor 主表只保存 Theme 归属、中心 Chain Node、发布回执和不可变自然语言快照：一句话结论、原子事实汇总、净方向、当前支持、可空的当前反证、交易研究指向、下一检查点。它不复制 Theme 批次、节点名称、Theme 发布时间或旧 Anchor 分类字段；同一 Theme 与中心节点组合唯一。
_Avoid_: 复制节点名称、保存 analysis batch、恢复 anchor type/importance、保存第二份 transmission path、用 updated_at 暗示可变业务记录

**Anchor 路径节点记录（Research Anchor Chain Node Record）**:
`research_anchor_chain_nodes` 保存一棵 Anchor 唯一线性路径的节点快照，而不是普通多对多关联。`position` 从 1 开始并表达路径业务顺序；首节点没有传入机制，后续节点必须说明从前一节点传入的机制。同一节点在一条路径中唯一，整条路径至少两个节点且必须包含中心节点一次。
_Avoid_: 把 position 当界面 display order、跳号、重复节点、首节点伪造传入机制、后续节点缺失传导机制、将该表当正式产业链关系

**Research Anchor Event 记录（Research Anchor Event Record）**:
`research_anchor_events` 是 Anchor 中 Event 证据语义的唯一存储，每个 Anchor 与 Event 组合唯一。它只保存受控证据角色和非空证据摘要；Event 标题、摘要与时间始终来自 Event 主数据。每棵 Anchor 至少有一个 driver，且全部 Event 必须属于父 Theme 已发布的 Event 子集。
_Avoid_: 复制 Event 正文、同一 Event 多角色、没有 driver、越过父 Theme 证据边界、为页面分组创建重复关联表

**推理树共享测试 Fixture（Reasoning Tree Shared Test Fixture）**:
`src/testdata/reasoning-tree-v1/` 中用于验证 Data、BFF 和小程序合同一致性的确定性 JSON 样例。它不是 AI 分析师运行产物或产品数据来源；真实 Research Anchor 只能由分析师发布器通过 Anchor Import V1 入库。
_Avoid_: 生产启动时导入 fixture、把 fixture 当分析师 outbox、真实分析缺失时回退到测试数据、各服务复制并修改不同版本

**推理树字段血缘（Reasoning Tree Field Lineage）**:
AI 分析师 V3 与 Research Anchor Import V1 对 Anchor、支持/反证汇总、Event Evidence 和 Path Node 使用同名结构。发布器只把 Theme Import 回执解析出的 `theme_id` 与对应完整 `anchors` 集合装入请求，不做语义转换。Tidewise 生成 Anchor/Receipt 身份与时间，并用 Chain Node、Event 主数据补充读取名称和展示字段；BFF 不改写研究语义。
_Avoid_: 发布器字段重命名、从 Markdown 重新解析、Data 推断缺失语义、AI 复制节点名称或 Event 正文、BFF 改写证据和路径

**主题指数关联（Theme Index Association）**:
Research Theme 与指数之间的方向性影响判断。当前数据库保留该能力，但 Theme 发布 V1 不接收或写入指数关联；请求出现 `indices` 或 `index_entity_ids` 时按未知字段拒绝。没有指数关联时对外读取为空集合。分析侧本地保存的指数信息不因此成为已发布事实，未来通过新合同版本引入。

**传导阶段（Transmission Stage）**:
研究主题结论沿证据与影响路径发展的当前生命周期。`identification` 表示刚识别出传导假设，`validation` 表示已有事件或市场证据验证，`diffusion` 表示影响已向多个产业链节点或市场对象扩散，`dampening` 表示驱动减弱或出现反向证据。该判断由分析侧明确提交；Data 不设置默认值或从其他字段推断。
_Avoid_: 上游阶段、中游阶段、下游阶段、用市场确认状态替代传导阶段

**市场确认状态（Market Confirmation）**:
市场数据对 Research Theme 判断的确认、背离、混合或未观察状态。当前仅保留在分析侧结构化结果和报告中，用于证据一致性与报告质量校验，不进入 Theme 发布合同或 Data 持久化。它不能推导或替代 Transmission Stage 等数据库字段。

**市场确认摘要（Market Confirmation Summary）**:
对 Research Theme 整体市场验证情况的必填自然语言说明，可以涵盖指数、股票、ETF、商品等市场对象。内容由 AI 分析师生成，发布器将分析侧 `market_confirmation.reasoning` 一对一重命名后原样传递；Data 只校验存在且非空，不生成默认文本，也不解析、推断或改写。没有观察到市场验证时，摘要必须明确说明未获得可归属的市场证据，以区分没有证据和分析遗漏。
_Avoid_: 使用 `index_impact_summary` 表示整体市场确认、与单个指数关联的影响说明混用

**下一检查点（Next Checkpoint）**:
研究主题当前尚待显现或验证的凝练、可执行中文观察项，不是固定枚举。它由 AI 分析师直接生成，发布器原样传递；Data 只校验非空，不拼接、不改写也不设置默认值。分析侧更详细的确认条件不进入当前 Theme 发布合同。
_Avoid_: 发布器机械拼接确认条件、由 Data 生成观察项

**交易研究指向（Trading Direction）**:
基于当前结论给出的受益、承压、关注方向及交易映射，以自然语言保存，不是做多/做空枚举。内容由 AI 分析师生成，发布器将分析侧 `research_direction` 一对一重命名后原样传递；Data 只校验非空。泛化的“继续关注”等表述应在分析侧校验或审核阶段拦截。
_Avoid_: 用 Trading Direction 表达下一检查点、由发布器或 Data 补写交易语义

## Source Ownership

Data 业务代码必须收敛到 `src/backend/services/data/`：

```text
cmd/          process and maintenance entrypoints
usecase/      import, query, seed and projection orchestration
domain/       Data-owned rules and models
repositories/ persistence ports and implementations
adapters/     PostgreSQL, Neo4j, migration and inbound/outbound adapters
transport/    Data REST routes, handlers, middleware and DTOs
config/       Data-only runtime configuration
```

`src/backend/migrations/` 与 `src/backend/data/` 是 Data 的统一事实资产，可以保留为 Backend 根资产，但不得被 BFF 直接读取。
