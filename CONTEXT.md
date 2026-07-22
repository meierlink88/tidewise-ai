# Tidewise AI Agent Runner Context

本上下文定义观潮家 Agent 执行平台及其研究 Agent 使用的核心语言，不包含模型、框架或存储实现。

## Platform Language

**Agent Definition**:
平台支持的一类稳定 Agent 能力身份，例如 Collector Agent、Event Extractor Agent 或 Analyst Agent；它不等同于某次运行或某个模型。
_Avoid_: 用命令名、模型名或一次任务标识 Agent

**Agent Version**:
某个 Agent Definition 的不可变执行合同版本，冻结其输入、输出、工作流能力和执行协议。
_Avoid_: 原地修改已发布 Agent、把业务任务提示词当作 Agent Version

**Agent Execution**:
平台接受一个外部任务后创建的一次可追踪执行实例；它记录实际使用的 Agent Version 和任务快照，但不取代调用方拥有的业务 Run。
_Avoid_: Data Collection Run、分析批次、定时计划

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

**Provider Configuration**:
AgentRun PostgreSQL 中某个 LLM 或 Connector 当前生效的 Base URL、模型和必要 Key；V1 直接覆盖当前值，不保存配置版本或轮换历史。明文 Key 只允许 dev/UAT MVP，任何接口和 Artifact 都不得暴露它。
_Avoid_: Agent Version 固定护栏、Collection Prompt、Provider 环境变量 fallback

**Run Artifact Manifest**:
一次 Agent Execution 本地 Artifact 完成的权威标记；它记录执行身份、Prompt hash/长度、Connector 与 Candidate 统计、accepted 文件路径/hash 和时间戳，但不包含完整 Prompt、新闻正文或 Provider Key。
_Avoid_: PostgreSQL Execution 行、临时文件、Tidewise Raw Document Import receipt

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
