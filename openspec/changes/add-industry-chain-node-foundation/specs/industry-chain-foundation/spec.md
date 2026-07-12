## ADDED Requirements

### Requirement: 独立产业链主数据
系统 SHALL 将产业链作为独立 `industry_chain` 实体保存，并以稳定代码、客观定义、范围、版本和 Review 状态区别于产业链节点、板块和动态主题。

#### Scenario: 创建产业链定义
- **WHEN** 已批准产业链进入实体基础库
- **THEN** 系统必须创建 `entity_type=industry_chain` 的实体和唯一 profile，并拒绝缺少稳定代码、定义、范围或合法 Review 状态的数据

#### Scenario: 产业链不是板块
- **WHEN** 同一业务主题同时存在产业链和市场板块
- **THEN** 系统必须保留两者独立实体身份，并通过已审阅客观关系连接，不得复用同一 profile 或 stable key

### Requirement: 链内节点成员与粒度
系统 SHALL 复用全局 `chain_node` 身份，并以链内成员记录保存该节点在特定产业链中的阶段、角色、顺序和核心性。

#### Scenario: 同一节点参与多条产业链
- **WHEN** 同一 `chain_node` 在两条产业链具有不同阶段或角色
- **THEN** 系统必须复用同一节点实体并保存两条独立 membership，不得复制节点身份

#### Scenario: 拒绝不合格节点粒度
- **WHEN** 候选节点只是单家公司、单个证券、短期热点或无法定义的宽泛主题
- **THEN** Review 必须拒绝其作为稳定 `chain_node`

### Requirement: 稳定产业链拓扑
系统 SHALL 在产业链范围内保存有来源、可审阅且端点均为 active membership 的稳定拓扑，并支持 `upstream_of`、`input_to`、`output_to`、`depends_on`、`substitutes_for` 和 `bottleneck_candidate_for`。

#### Scenario: 写入稳定拓扑
- **WHEN** 一条拓扑关系已通过 Review 且来源、方向、端点和核验时间完整
- **THEN** 系统必须按 chain scope 幂等保存并拒绝自环、重复关系或非成员端点

#### Scenario: 区分结构性候选与当前瓶颈
- **WHEN** 拓扑包含 `bottleneck_candidate_for`
- **THEN** 该关系只能表达结构性稀缺约束候选，不得包含当前严重度、传导强度、受益承压或预测结论

### Requirement: 产业链 typed observation
系统 SHALL 使用通用 observation governance envelope 管理来源、时间、幂等、质量和修订，并使用产业链节点指标表与拓扑流量表保存领域值，不得使用单一万能 EAV 表。

#### Scenario: 保存节点指标观察
- **WHEN** ingestion 提交节点产能、产量、库存、利用率、交期或价格观察
- **THEN** 系统必须同时保存 governance envelope 与 typed node observation，并校验 chain、node、metric、单位、观察期和来源

#### Scenario: 保存拓扑流量观察
- **WHEN** ingestion 提交贸易量、投入量、产出量或依赖流量观察
- **THEN** 系统必须引用有效 topology edge 和 metric，并以 typed flow observation 保存数值和观察期

#### Scenario: 拒绝万能属性写入
- **WHEN** 写入请求只提供任意属性名和文本值而没有明确 typed contract
- **THEN** 系统必须拒绝把该数据作为 validated 产业链 observation

### Requirement: 瓶颈分析输入与输出边界
系统 SHALL 只把 active 主数据、稳定拓扑、validated observation、事件证据和客观跨实体关系作为瓶颈分析输入，并将动态结论留给独立 event-driven reasoning 结果。

#### Scenario: 生成动态分析输入
- **WHEN** 推理流程评估产业链瓶颈
- **THEN** 输入必须包含实体或拓扑身份、metric、值、单位、观察期、来源和质量状态，且不得将输出回写为主数据事实

#### Scenario: 保持决策辅助定位
- **WHEN** API 或小程序展示产业链传导或瓶颈分析
- **THEN** 输出必须携带证据和不确定性并定位为市场理解与决策辅助，不得表达为直接投资建议

### Requirement: Serenity 研究契约映射
系统 SHALL 将市场故事、系统变化、必需部件、产业链层级、稀缺约束、证据等级和证伪条件映射为可解释的分析契约，并保持稳定事实、动态 observation 和 reasoning result 分离。

#### Scenario: 形成稀缺层分析输入
- **WHEN** event-driven reasoning 评估某一产业链稀缺层
- **THEN** 输入必须关联 chain/node/topology、需求或系统变化事件、约束 metric observations 和分级证据，而不得只依赖市场热度或公司叙事

#### Scenario: scorecard 不成为主数据
- **WHEN** 分析使用需求拐点、架构耦合、供应商集中、扩产难度、估值差、催化剂或风险 penalty
- **THEN** 评分、权重和排序必须作为带时点的可重算 reasoning result，不得写入产业链 profile、membership、topology 或客观 `entity_edges`

#### Scenario: 输出证伪条件
- **WHEN** 系统输出稀缺层、公司位置或市场板块传导判断
- **THEN** reasoning result 必须包含证据等级、缺失证据、主要风险和可判定的 downgrade/kill-switch 条件

### Requirement: 首批 MVP 试点 Review
系统 SHALL 将 AI 算力基础设施、半导体制造、机器人作为首批试点候选，将新能源汽车/储能、创新药/生物制造保留为第二批，并依据事件价值、可观测性、中国板块映射、节点粒度、稀缺约束和来源完整性进行人工 Review。

#### Scenario: 形成候选清单
- **WHEN** Propose 或 Apply 准备首批产业链
- **THEN** 系统必须展示每条试点链建议 10–20 个节点及复用/新增依据，并以三链去重后约 30–50 个节点作为审阅目标而非自动写入配额

#### Scenario: 未批准候选不进入 seed
- **WHEN** 链范围、节点、拓扑或跨实体关系尚未逐项获得人工 Review
- **THEN** 候选不得进入正式 seed、PostgreSQL 或 Neo4j

### Requirement: 下游消费契约
系统 SHALL 为事件抽取、event-driven reasoning、API 和小程序区分主数据、observation 与 reasoning result，并禁止消费者绕过服务端事实和证据边界。

#### Scenario: 事件抽取关联产业链
- **WHEN** 事件抽取识别产业链、节点、商品、benchmark、metric 或 sector
- **THEN** 抽取流程必须生成带证据的实体链接候选，不得修改产业链主数据或稳定拓扑

#### Scenario: 小程序读取产业链分析
- **WHEN** 小程序展示产业链结构、动态观察或传导分析
- **THEN** 小程序必须通过服务端 DTO 读取并区分 `master_data`、`observation` 和 `reasoning_result`
