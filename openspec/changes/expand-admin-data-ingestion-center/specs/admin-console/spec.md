## ADDED Requirements

### Requirement: 数据采集中心菜单
系统 SHALL 在管理后台提供 `数据采集中心` 一级菜单，用于承载采集链路的只读查询展示和调度器配置能力。

#### Scenario: 登录后访问数据采集中心
- **WHEN** 管理员已通过 Admin Token 登录
- **THEN** sidebar 必须展示 `数据采集中心` 菜单
- **AND** 内容区必须展示 `原始数据`、`全球事件`、`搜索通道`、`调度器` 四个 tab

#### Scenario: 不把管理后台改成单一采集工作台
- **WHEN** 后续管理后台新增其他运营菜单
- **THEN** `数据采集中心` 必须作为一个独立一级菜单存在，不得占用整个 admin portal 的产品定位

### Requirement: 原始数据列表
系统 SHALL 在 `数据采集中心` 的 `原始数据` tab 展示 `raw_documents` 只读列表，并支持分页和标题模糊搜索。

#### Scenario: 分页展示原始数据
- **WHEN** 管理员打开 `原始数据` tab
- **THEN** 页面必须按采集时间倒序展示原始数据列表
- **AND** 每页必须展示 50 条
- **AND** 页面必须展示总数、当前页和翻页控制

#### Scenario: 按标题搜索原始数据
- **WHEN** 管理员输入标题关键词并执行搜索
- **THEN** 页面必须只展示标题匹配关键词的原始数据
- **AND** 搜索结果仍必须按每页 50 条分页

#### Scenario: 原始数据为空
- **WHEN** 当前查询条件下没有原始数据
- **THEN** 页面必须展示空状态，而不是展示错误页面

### Requirement: 全球事件列表
系统 SHALL 在 `数据采集中心` 的 `全球事件` tab 展示 `events` 只读列表，并支持分页、标题模糊搜索和关键筛选项。

#### Scenario: 分页展示全球事件
- **WHEN** 管理员打开 `全球事件` tab
- **THEN** 页面必须按 `first_seen_at` 倒序优先展示事件列表
- **AND** 每页必须展示 50 条
- **AND** 页面必须展示总数、当前页和翻页控制

#### Scenario: 按标题搜索全球事件
- **WHEN** 管理员输入事件标题关键词并执行搜索
- **THEN** 页面必须只展示标题匹配关键词的事件
- **AND** 搜索结果仍必须按每页 50 条分页

#### Scenario: 按事件状态筛选
- **WHEN** 管理员选择 `event_status`
- **THEN** 页面必须只展示匹配该 `event_status` 的事件

#### Scenario: 按事实状态筛选
- **WHEN** 管理员选择 `fact_status`
- **THEN** 页面必须只展示匹配该 `fact_status` 的事件

#### Scenario: 按事件发生时间筛选
- **WHEN** 管理员设置 `event_time` 起止范围
- **THEN** 页面必须只展示事件发生时间落在该范围内的事件

#### Scenario: 按首次发现时间筛选
- **WHEN** 管理员设置 `first_seen_at` 起止范围
- **THEN** 页面必须只展示首次发现时间落在该范围内的事件

#### Scenario: 全球事件为空
- **WHEN** 当前查询条件下没有事件
- **THEN** 页面必须展示空状态，而不是展示错误页面

### Requirement: 搜索通道列表
系统 SHALL 在 `数据采集中心` 的 `搜索通道` tab 展示 `source_catalogs` 只读列表，并支持按状态筛选。

#### Scenario: 展示搜索通道
- **WHEN** 管理员打开 `搜索通道` tab
- **THEN** 页面必须展示 source catalog 的来源名称、provider、channel、source type、URL 和状态
- **AND** 页面不得展示 parser 字段

#### Scenario: 按状态筛选搜索通道
- **WHEN** 管理员选择 `active`、`inactive` 或 `disabled` 状态
- **THEN** 页面必须只展示匹配该状态的搜索通道

#### Scenario: 搜索通道不分页
- **WHEN** 管理员查看搜索通道列表
- **THEN** 页面必须一次性展示当前筛选结果
- **AND** 页面不得展示分页控件

### Requirement: 调度器配置与执行记录
系统 SHALL 在 `数据采集中心` 的 `调度器` tab 保留调度器配置能力，并展示最近 50 条调度执行记录。

#### Scenario: 左右结构展示调度器
- **WHEN** 管理员打开 `调度器` tab
- **THEN** 页面左侧必须展示调度器配置表单
- **AND** 页面右侧必须展示最近调度执行记录列表

#### Scenario: 保存调度器配置
- **WHEN** 管理员修改调度器配置并保存
- **THEN** 页面必须继续通过 Admin Token 调用后端保存配置
- **AND** 刷新页面后必须展示已保存配置

#### Scenario: 展示最近 50 条执行记录
- **WHEN** 管理员查看调度器执行记录
- **THEN** 页面必须展示最近 50 条调度 run
- **AND** 记录必须按开始时间倒序排列
- **AND** 列表不得分页

#### Scenario: 执行记录统计表示轮次结果
- **WHEN** 页面展示调度器执行记录
- **THEN** 成功、失败、跳过和总数必须表示该轮调度内 source 执行结果
- **AND** 不得把最近运行摘要误表达为累计执行轮次数

#### Scenario: 执行记录为空
- **WHEN** 当前还没有调度执行记录
- **THEN** 页面必须展示空状态，而不是展示错误页面

### Requirement: 数据采集中心 Admin API
系统 SHALL 提供 Admin Token 保护的只读管理 API，供 `数据采集中心` 查询原始数据、全球事件、搜索通道和调度执行记录。

#### Scenario: 查询接口需要 Admin Token
- **WHEN** 管理后台调用数据采集中心查询 API
- **THEN** 请求必须携带 `Authorization: Bearer <token>`
- **AND** token 缺失或无效时后端必须拒绝访问

#### Scenario: 分页接口返回统一分页结构
- **WHEN** 管理后台查询原始数据或全球事件分页接口
- **THEN** 后端必须返回 `items`、`total`、`page` 和 `page_size`
- **AND** `page_size` 默认和管理后台展示口径必须为 50

#### Scenario: 查询接口不触发采集
- **WHEN** 管理后台调用原始数据、全球事件、搜索通道或调度记录查询接口
- **THEN** 后端不得启动采集器、调度器、connector、AI 模型调用或外部搜索 API
