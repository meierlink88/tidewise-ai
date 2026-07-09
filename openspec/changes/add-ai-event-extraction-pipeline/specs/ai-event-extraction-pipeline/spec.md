## ADDED Requirements

### Requirement: AI 事件提取流水线
系统 SHALL 提供从 `raw_documents` 异步提取结构化事件、标签、实体关联和证据的 AI Event Extraction Pipeline，并将通过校验的结果写入事件知识边界。

#### Scenario: 处理待提取原始文档
- **WHEN** 事件提取 worker 读取到 pending 状态的提取任务
- **THEN** 系统必须加载对应 raw document、提取配置、prompt 版本、模型配置和实体基础库上下文，并生成结构化事件候选结果

#### Scenario: 不阻塞采集流程
- **WHEN** raw document 由采集流程写入成功
- **THEN** 系统必须允许后续事件提取异步执行，不得要求采集 connector 等待 AI 提取完成

### Requirement: 事件提取 job 状态
系统 SHALL 使用持久化 job 边界管理事件提取任务，并支持领取、重试、失败、跳过、成功和重跑。

#### Scenario: 创建提取任务
- **WHEN** 新 raw document 成功写入且满足事件提取条件
- **THEN** 系统必须创建 pending 状态的事件提取 job 或标记等价待处理状态，并保证重复触发不会创建无意义重复任务

#### Scenario: 领取任务
- **WHEN** 多个 worker 并发读取 pending 任务
- **THEN** 系统必须保证同一任务不会被多个 worker 同时成功处理，并记录任务开始时间和处理者信息

#### Scenario: 处理失败
- **WHEN** 模型调用、输出解析、实体匹配或数据库写入失败
- **THEN** 系统必须更新 job 状态、错误码、错误消息和重试次数，并按最大重试策略决定继续重试或进入 failed 状态

### Requirement: LLM 提取输出校验
系统 SHALL 对 AI 输出执行结构化 schema 校验、证据校验、来源校验和安全边界校验，只有通过校验的候选对象才能写入事件事实。

#### Scenario: 接收有效提取结果
- **WHEN** AI extractor 返回包含事件、标签、实体关联和证据的结构化 JSON
- **THEN** 系统必须校验事件标题、发生时间、事件类型、摘要、证据摘录、raw document 引用、标签和实体关联字段后再写入

#### Scenario: 拒绝无证据事件
- **WHEN** AI extractor 返回事件候选但缺少 raw document 引用、证据摘录或来源归因
- **THEN** 系统必须拒绝该事件进入事实表，并在 extraction run 或 job report 中记录跳过原因

#### Scenario: 拒绝投资建议字段
- **WHEN** AI extractor 返回买入卖出、涨跌预测、利好利空结论、传导强度、事件评分或直接投资建议
- **THEN** 系统不得把这些字段写成系统事实，必须丢弃、标记越界或将该候选结果判为校验失败

### Requirement: 事件标签标注
系统 SHALL 为事件候选结果生成可校验标签，并将标签关联保存为事件事实的一部分。

#### Scenario: 写入事件标签
- **WHEN** 事件候选通过结构化校验且包含标签
- **THEN** 系统必须把标签映射到受控 tag taxonomy 或 pending_review 状态，并保存标签来源、置信度和证据

#### Scenario: 拒绝未知高置信事实标签
- **WHEN** 模型返回未注册标签且未被配置允许进入 pending_review
- **THEN** 系统必须拒绝把该标签作为正式标签事实写入

### Requirement: 事件实体关联
系统 SHALL 将事件与实体基础库中的相关实体建立结构化关联，并保存关联类型、证据摘录、置信度和匹配方式。

#### Scenario: 关联已知实体
- **WHEN** 提取结果引用经济体、联盟组织、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品或人物
- **THEN** 系统必须尝试匹配实体基础库，并通过事件实体关联保存 entity ID、关联类型、证据摘录、置信度和匹配方式

#### Scenario: 处理无法匹配实体
- **WHEN** 模型返回的实体无法匹配到实体基础库
- **THEN** 系统不得直接创建未知正式实体，必须跳过、记录为 unmatched candidate 或进入后续审核边界

### Requirement: 事件证据链
系统 SHALL 将每个事件事实与一个或多个 raw document 证据关联，并保存证据摘录、证据哈希和来源质量信息。

#### Scenario: 保存事件证据
- **WHEN** 事件候选通过校验并写入事件事实
- **THEN** 系统必须通过事件来源证据边界关联事件和 raw document，并保存证据摘录、证据哈希、来源等级和内容来源类型

#### Scenario: 追加证据
- **WHEN** 新 raw document 被识别为已有事件的补充证据
- **THEN** 系统必须追加事件证据，而不是创建无关联重复事件

### Requirement: 事件去重与合并
系统 SHALL 在写入事件事实前执行可测试的去重或合并策略，避免同一事件因多篇文档重复生成。

#### Scenario: 匹配已有事件
- **WHEN** 事件候选与已有事件在事件类型、发生时间、地点、参与方、标题语义或去重 key 上匹配
- **THEN** 系统必须复用已有事件并追加 evidence、tag 或 entity link

#### Scenario: 创建新事件
- **WHEN** 事件候选无法匹配已有事件且通过校验
- **THEN** 系统必须创建新的事件事实，并保存提取来源和处理状态

### Requirement: 提取运行记录
系统 SHALL 记录每次事件提取运行的输入、prompt、模型、schema、输出摘要、状态、耗时和错误，支持审计和回放。

#### Scenario: 记录成功运行
- **WHEN** 事件提取任务成功完成
- **THEN** 系统必须保存 raw document ID、prompt 版本、模型 provider、模型名、schema 版本、生成事件数量、标签数量、实体关联数量、证据数量和耗时

#### Scenario: 记录失败运行
- **WHEN** 事件提取任务失败
- **THEN** 系统必须保存失败阶段、错误码、错误消息、是否可重试和必要的脱敏上下文，不得记录真实 API key、bearer token、cookie 或私有凭证
