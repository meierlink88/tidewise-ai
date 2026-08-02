# Tidewise AI Agent Runner Context

本上下文定义观潮家 Agent 执行平台及其研究 Agent 使用的核心语言，不包含模型、框架或存储实现。

## Platform Language

**Agent Definition**:
平台支持的一类稳定 Agent 能力身份，例如 Collector Agent、Event Extractor Agent 或 Analyst Agent；它不等同于某次运行或某个模型。
_Avoid_: 用命令名、模型名或一次任务标识 Agent

**Agent Version**:
某个 Agent Definition 的不可变执行合同版本，冻结其输入、输出、工作流能力和执行协议。
已批准的特定例外只有 Collector Issue #121：其 Web Search freshness 修正按
`collector.v1` conformance fix 交付，并由 Collector 可靠性 Spec 记录部署前后的
可观察差异；该例外不授权其他 Agent 或后续变更原地修改已发布版本。
_Avoid_: 原地修改已发布 Agent、把业务任务提示词当作 Agent Version

**Agent Execution**:
平台接受一个外部任务后创建的一次可追踪执行实例；它记录实际使用的 Agent Version 和任务快照，但不取代调用方拥有的业务 Run。
_Avoid_: Data Collection Run、分析批次、定时计划

**Agent Status Monitor**:
AgentRun 为全部已注册 Agent Definition 提供的只读当前状态快照：Agent Key、显示名称、
当前 Version、是否存在在途 Execution、当前 Execution Status 与更新时间；没有在途
Execution 时状态为 `idle`。AgentRun 拥有该状态事实，Admin Portal 只做代理和展示。
_Avoid_: 单次执行详情、阶段/耗时/重试/候选统计、人工审核 UI、Admin Portal 自建运行状态、
泄露业务载荷

**Agent Input**:
提交给某个明确 Agent Version 的业务输入；其结构由该版本定义，Schedule 中保存的输入会在每次触发时复制为新的 Agent Execution 快照。
_Avoid_: 把所有 Agent Input 都称为 Prompt、模型或 Connector 配置

**Agent Schedule**:
平台为某个 Agent Definition 保存的唯一周期性触发计划；它固定绑定一个明确 Agent Version，每次到期触发都会创建新的 Agent Execution。
_Avoid_: Agent Execution、Collection Run、仅属于 Collector 的定时器

**Admin Portal Service**:
代表运维管理员调用 AgentRun 管理面的受信任后端服务；浏览器不直接访问 AgentRun Admin API。
_Avoid_: Tidewise Data Service、浏览器客户端、Agent Execution 调用方

**Collection Run**:
由 Tidewise Data Center 拥有的一次业务采集运行身份；AgentRun 不拥有其调度、最终状态或 Watermark。Collector V1 HTTP 不传输该身份，Data Service 在自身边界内保存它与 Agent Execution ID 的映射。
_Avoid_: Agent Execution、Connector Invocation

**Collection Attempt**:
Data Center 对同一 Collection Run 发起的一次完整执行尝试。V1 中每次新的远程尝试使用新的 Idempotency-Key 创建新的 Agent Execution；AgentRun 不持久化 Collection Attempt 实体。
_Avoid_: Agent Execution、进程重启、同一 Idempotency-Key 的 HTTP 重放

**Collector Run Spec**:
调用方提交给 Collector Agent 的不可变业务任务快照；V1 的业务内容是一段自然语言 Collection Prompt，执行身份和采集时间由 AgentRun 生成，Provider 与 Connector 细节不属于调用方输入。
_Avoid_: Connector 配置、模型参数、Source 配置、第二套 Run Schema

**Collector Agent**:
读取一个不可变采集任务，调用固定 Connector，并把 Connector 直接返回的原始信息写成本地 Raw Document Artifact 的 Agent；它不提炼 Event、不产生投资分析，也不在 V1 中主动发布给 Data Center。
_Avoid_: Event Extractor Agent、Analyst Agent、固定来源调度器

**Connector Invocation**:
某个 Collector Agent Execution 对一个固定 Connector 的一次调用记录，用于说明该通道是否完成、失败或因执行级故障而未启动，以及直接结果数量；V1 不取消、重试或跨进程恢复 Invocation。
_Avoid_: 自动 HTTP retry、Data Collection Attempt、持久化任务 lease

**Collection Candidate**:
Connector 直接结果按 canonical URL 合并后、写成本地 Raw Document Artifact 前的临时质量门禁对象；它经过时间门禁和去重并进入一个可审计终态。Candidate 不是 PostgreSQL 正式事实，只有 accepted Candidate 才生成本地 Raw Document Artifact。
_Avoid_: Candidate 数据库实体、Tidewise 正式 Raw Document、LLM 生成的事实

**Raw Document Artifact**:
由 accepted Candidate 生成、保存在 AgentRun 持久化 Artifact Volume 中的原始采集文件；它是未来 Data Import 的输入，但尚不是 Tidewise Data 的正式 Raw Document。
_Avoid_: Collection Candidate、Tidewise 正式 Raw Document、临时文件

**Collection Prompt**:
调用方提交、定义一次采集“采什么”的完整自然语言任务文本；Collector 只对它做查询语义规划，不在本地维护业务采集内容。
_Avoid_: Connector 配置、agent-run 本地业务提示词、机器输出协议

**Model Provider Configuration**:
定义 Agent 访问某个已由 AgentRun 实现并注册的模型供应商及其选定模型所需的当前运行信息；它属于模型调用边界，不属于采集通道，也不能通过新增任意配置 key 创造模型能力。
_Avoid_: Connector Configuration、Agent Version、Collection Prompt、未实现 Provider 的动态注册

**Connector Configuration**:
定义一个已由 AgentRun 实现并注册的 Connector 访问其外部采集通道所需的当前运行信息；Connector 本身就是平台中的采集适配器，不再额外称为 Connector Provider，也不能通过新增任意配置 key 创造采集能力。
_Avoid_: Model Provider Configuration、Connector Invocation、Collection Prompt、未实现 Connector 的动态注册

**Run Artifact Manifest**:
一次 Agent Execution 本地 Artifact 完成的权威标记；它记录执行身份、Prompt hash/长度、Connector 与 Candidate 统计、accepted 文件路径/hash 和时间戳，但不包含完整 Prompt、新闻正文、Model Provider Key 或 Connector Key。
_Avoid_: PostgreSQL Execution 行、临时文件、Tidewise Raw Document Import receipt

**Artifact Publication**:
一次已完成 Collector 物化结果从持久化 pending 计划进入 accepted Markdown、共享 dedup index、Run Artifact Manifest 和 PostgreSQL 终态的可恢复提交过程。它只恢复文件发布与终态对账，不重新运行 Planner 或 Connector。
_Avoid_: Agent Execution 重试、任务队列、Eino checkpoint、Data Raw Document Import

**Artifact Ready Signal**:
Collector Artifact Publication 完成且至少产生一个 accepted Raw Document Artifact 后形成的持久化依赖事实；它使下游 Event Fact Extraction 可立即开始，并可在漏触发时被重新发现。
_Avoid_: Agent Schedule、轮询文件目录、Collector Workflow 内的 Event 提取节点

**Event Fact Extractor Agent**:
读取一个或多个已完成 Collector Execution 的 accepted Raw Document Artifact，提取、校验并发布原子 Event 核心事实的 Agent Definition；它不建立 Event 到 Entity、Chain Node 或影响信号的正式语义关联。
当前 `event-fact-extractor.v2` 保留确定性 Eino Workflow，但四个模型阶段分别通过
`submit_event_candidates`、`submit_duplicate_judgments`、`submit_tag_assignments` 和
`submit_event_reviews` 强制 Function Call 提交固定 DTO。每阶段只绑定一个无副作用结果
Function，错误调用最多修正一次；Data Publication 仍由 Application/Journal 执行。
Evidence 使用模型生成、独立审核的 `evidence_statement`，程序不再通过 Artifact 正文
字符串包含或日期文本搜索裁决 Event 语义。
_Avoid_: Collector Agent、Event Semantic Enricher、把采集与提取合成一个 Agent

**Event Extraction Work Item**:
AgentRun 对一组明确 Collector Execution 和一个 Event Fact Extractor Agent Version 承担的持久化处理义务；它跨 Agent Execution 尝试、审核等待和发布重试保持稳定。
_Avoid_: Agent Execution、Agent Schedule、临时内存任务

**Event Fact Candidate**:
Event Fact Extractor 从 Artifact 中形成并通过候选级校验的原子事实解释；在 Event Publication 成功前它只属于 AgentRun，不是 Data 的正式 Event。
_Avoid_: Data Event、未经校验的模型输出、Event Semantic Link

**Event Review State**:
AgentRun 对 Event Fact Candidate 是否可发布的审核结论；平台保留自动通过、待人工审核和拒绝等状态，即使当前运行策略只启用自动通过路径。
_Avoid_: Data Event Status、模型自行选择数据库状态

**Event Publication Journal**:
AgentRun 为一次待发布 Event Publication Batch 持久化的不可变请求正文、内容哈希和投递结果；未知调用结果只能重发同一正文，不重新运行 Event 提取。
Data 单个 Batch 最多十个 Event，但一个 Artifact Unit 可以拥有多个稳定 Journal；只有该 Unit
全部 Journal 获得 Data 回执后 Unit 才是 `published`，不得因候选超过十个截断或拒绝整个 Unit。
_Avoid_: Data Import Receipt、Eino checkpoint、可原地修改的 Outbox 草稿

**Event Semantic Enricher**:
在正式 Event 已存在后，从 Data 的精简、pinned Context 动态取得 Entity Type/角色、
Variable Definition、适用 Entity Type、方向、modality 和 Measurement 合同。
`event-semantic-enricher.v3` 的 Stage A 只提取 Event 原文 raw mention 与 Evidence 血缘；
不预测 Entity Type、不预先分配角色，也不生成 Signal。AgentRun 先对整个 Event 批量执行
跨 Entity Type 正规名/别名精确匹配，再对未唯一命中 mention 执行一次跨类型 batch vector
recall。Selector 只能从当前 mention 的 Qdrant 候选选择正式 ID 或 `no_match`，并按候选携带
的正式 Entity Type Definition 分配允许角色；Entity Type 不由模型输出。Data 在 Submission
中比对 `projected_entity_type` 与 PostgreSQL Entity ID/type/status/TBox。Vector Top-K 由
`semantic_retrieval.entity_top_k` 配置，范围 1..20，当前校准值为 10。
Selector 与独立 Review 只使用候选 canonical name、name 和正式 aliases 确认 identity；不删除
“系统/服务/设备/产品”等后缀，不使用字符串包含或其他手写简称规则。vector Top-1 不自动接受；
只有唯一 exact identity 被主 Selector 拒绝时，由独立 Reviewer 做一次有界选择复核。
`mention_not_entity` 只用于日期、数值、
状态、行为、报告、会议等真正非实体，真实公司、产品、技术、指数不得用它掩盖 ABox/TBox gap。
Entity Resolution 完成后，AgentRun 才按正式 Entity Type 从 pinned complete Variable
Definition directory 确定性筛选适用目录，并以独立 Signal Stage 生成 Event-native
VariableSignal 与可选自然语言 Measurement。EventEntityLink 可以没有 Signal；单个 Mention、
Selection、Signal 或 Review item 非法时只隔离该候选，不撤销同 Event 的其他合法事实。
对象同一性不使用 AgentRun 手写简称、职衔、国别前缀或证券后缀规则。唯一 canonical/alias exact
identity 来自正式投影；其他候选由 Selector 提议并由独立 AI Reviewer 判断是否同一业务对象。
合法简称缺口归正式 alias 数据治理，不在 Workflow 中建设第二套字符串 TBox。
AgentRun 的 embedding 调用必须经 Eino `embedding.Embedder`与 eino-ext 官方
OpenAI-compatible adapter；自定义 Qdrant adapter 只弥补官方单 query Retriever 无法提供的
Event-batch、跨类型召回和候选白名单能力。每批未命中 mention 只发生一次
`EmbedStrings` 和一次 Qdrant query batch。
模型只有在严格 JSON envelope 一次修复后仍不可解析时才终止整个 Event；Candidate 只需
引用属于当前 Event 的 Evidence ID，AgentRun 不再要求 Mention 字符串逐字出现在 Event 或
Evidence 文本中。候选内容错误进入
per-stage audit/isolation，并为 `no_match`、TBox 排除、retrieval/transport failure 记录 owner
classification。Reviewer 的缺失或非法 item 会被隔离并使用该候选自身 Evidence 形成 fail，不扩大为
Execution 失败。Submission 同时冻结 Reviewer 与 Adjudicator 的 Prompt/模型身份；首次
`indeterminate` 后只允许以冻结的 Adjudicator 身份执行第二轮，未知结果按 Data 已持久化
的 Review Snapshot 恢复，第二次 `indeterminate` 进入 quarantine。Measurement 只携带 `measurement_text + evidence_ids`，完整语义由 AI
审核对照 Evidence，不进行数值解析或归一化。
严格 envelope 要求 mention/selection/signal/review 分别显式携带 `mentions`、`selections`、
`variable_signals`、`items` 数组；`null`、`{}`、缺字段、`null` 数组或错误类型不能解释为空结果。
顶层 envelope 严格解析后，数组 item 按本 V3 固定 DTO 独立解析；单个 item 字段/类型非法只隔离
该 item，不触发整个 envelope repair。Semantic retrieval 的取消与 deadline 原样向 Execution 传播，
不包装为可重试远端错误。
AgentRun 对 Qdrant 外层 point ID 与 payload Entity ID、source identity、projection version、
embedding model 和 content fingerprint fail closed；payload 不要求重复 `point_id`。
`EMBEDDING_API_KEY` 在进程启动配置阶段校验，不允许领取 Work Item 后才暴露缺失。
Worker 每次先领取本地 ready work；只有本地为空时才向 Data 发现新 Event。Execution 已
持久化进入 succeeded、terminal failed 或 retry scheduled 后立即合并一次进程内通知，
继续领取下一项。60 秒 ticker 仅用于启动发现、漏通知、重启、retry maturity 和未知结果
对账，不再作为逐任务节拍；仍保持单 processing permit，不并行执行模型。
_Avoid_: Event Fact Extractor Agent、DirectImpact、Direct Target/Rule、产业链传导、Theme/机会/风险结论、AgentRun 自研 embedding HTTP 协议

**Event Semantic Work Item**:
AgentRun 对一个 Data Event 的一次初始语义分析或显式重新分析承担的持久化处理义务。
它拥有队列状态、幂等键、执行租约、有限尝试次数和当前 Agent Execution；初始分析按
Event ID 幂等建立，重新分析必须明确携带被替代的 Data Submission ID。
`skipped` 是一次性历史审计操作专用终态：它只表示历史 Event 的持久化输入不满足当前
Event Semantic 合同。正常发现、无候选、模型失败和新数据合同异常都不得写入
`skipped`；历史操作结束后，新 Work Item 仍只按正常状态机运行。
正式 Event 的 `occurred_at` 可以为空；这不影响 Work Item 发现、领取或 Context 水合，且不得
用发布、采集或首次发现时间替代未知发生时间。
_Avoid_: Data Context Lease、Data Submission、临时内存任务、无限重试

**Event Semantic Reanalysis Request**:
受信任调用方通过 AgentRun 内部 API 明确请求重新分析既有 Data Submission 的命令。
AgentRun 将其转为本地 Work Item 并自行闭环执行；Data Service 不保存或调度此任务。
_Avoid_: Data 任务队列、原地覆盖 Submission、无 supersedes identity 的模糊重跑

**Dedup Index Cache**:
从 accepted Raw Document Artifact 确定性派生的 TSV 缓存，用于 canonical URL、正文 SHA-256 与 SimHash64 去重。索引缺失可从 Markdown 重建；索引不是事实载体，也不得在缺失时被静默当作空历史。
_Avoid_: Raw Document Artifact、Candidate ledger、PostgreSQL Candidate 表

**Skipped Agent Execution**:
因同一 Agent Definition 已有活跃 Execution 而未启动目标 Agent Workflow 的终态 Agent Execution；它保留触发身份和阻塞来源以供审计，但不是排队任务。
_Avoid_: Active Agent Execution、Collection Attempt、等待队列、不同 Agent 之间的全局互斥

**Agent Lifecycle Log**:
AgentRun 在 Runtime、Application/Use Case、Worker Cycle 和已提交状态迁移边界输出的
结构化 JSON 业务事件。它只记录 Agent/Execution/Work Item 身份、安全计数、阶段、耗时
和稳定错误码，不记录 Prompt、正文、Evidence、模型原始输入输出、Connector body 或
凭据。日志用于核验，不参与调度、幂等、重试、对账或发布正确性。
_Avoid_: 数据库审计事实、状态机输入、Eino node callback、日志驱动业务恢复

## Language

**瓶颈假设（Bottleneck Thesis）**:
关于某个系统变化正在使产业链特定层级形成稀缺约束，并可能产生可跟踪投资价值的持续研究判断。
_Avoid_: 每日结论、热点、概念题材

**系统变化（System Change）**:
由 Event 反映的技术、需求、供给、政策或经济结构变化，它使原有系统设计或供给能力承受新的压力。
_Avoid_: 新闻影响、市场故事

**约束变量（Constraint Variable）**:
决定系统能否继续扩张或正常运行的可解释因素，例如产能、功率、带宽、良率、纯度、认证、许可或交期。
_Avoid_: 模糊瓶颈、直接影响

**瓶颈层（Bottleneck Layer）**:
约束变量实际落在产业链中的层级或节点集合，其扩张、替代或绕行能力不足以在研究时间范围内消除约束。
_Avoid_: 热门赛道、普通受益环节

**价值捕获（Value Capture）**:
企业因控制、供应或接近瓶颈层而把稀缺性转化为订单、价格、收入、利润或竞争地位的能力。
_Avoid_: 概念相关、业务沾边

**瓶颈更新（Thesis Update）**:
新 Event 或观察结果对既有瓶颈假设产生的增量影响，只能是支持、加强、削弱、改变时间或推翻之一。
_Avoid_: 重新生成报告

**失效条件（Invalidation Condition）**:
一旦被观察到，就足以明显降低或否定瓶颈假设的事先定义条件。
_Avoid_: 泛化风险提示

**决策准备度（Decision Readiness）**:
瓶颈假设在证据、企业价值捕获、市场关注差、时机、流动性和风险方面达到可用于投研决策参考的程度；它不是自动买卖指令。
_Avoid_: 买入评级、自动交易信号

**后验验证（Outcome Validation）**:
在预设时间窗口后，用产业兑现指标和市场结果检验瓶颈假设、企业映射及时间判断是否成立。
_Avoid_: 仅按股价复盘
