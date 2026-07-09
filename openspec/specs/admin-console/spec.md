## Purpose

定义观潮家 Web 管理后台的当前系统事实，覆盖独立前端工程、Minimal Dashboard 设计系统、自有 UI 基础层、Admin Token 登录、数据采集中心、调度器配置与执行记录，以及 Admin Token 保护的管理 API 边界。

## Requirements

### Requirement: 独立 Web 管理后台
系统 SHALL 在 `frontend/admin/` 提供独立 Web 管理后台，用于承载运营和系统管理能力，并与跨平台小程序工程保持边界隔离。

#### Scenario: 独立前端工程
- **WHEN** 开发者查看前端源码结构
- **THEN** 管理后台源码必须位于 `frontend/admin/`，不得混入 `frontend/miniapp/`

#### Scenario: 管理后台技术栈
- **WHEN** 开发者初始化或运行管理后台
- **THEN** 管理后台必须采用 Vite + React + TypeScript 技术栈，并以 Minimal Dashboard 作为标准设计系统

#### Scenario: 不影响小程序
- **WHEN** 管理后台新增页面、依赖或构建脚本
- **THEN** 不得破坏 `frontend/miniapp/` 的 Taro 小程序构建和运行边界

### Requirement: Minimal Dashboard 管理后台设计系统
系统 SHALL 将 Minimal Dashboard 作为 `frontend/admin/` 的标准设计系统，并通过 repo-local Codex skill 固化后续 agent 的使用入口。

#### Scenario: 使用 repo-local skill
- **WHEN** agent 处理 `frontend/admin` 的页面、组件、样式或布局开发
- **THEN** agent 必须读取 `.codex/skills/minimal-dashboard-design/SKILL.md`，并按该 skill 的 tokens、组件模式、preview 和 dashboard kit 转译生产实现

#### Scenario: 使用设计系统读取顺序
- **WHEN** agent 实现或修改 `frontend/admin`
- **THEN** 必须按 `.codex/skills/minimal-dashboard-design/library-consumption.json` 的推荐顺序读取设计系统资料
- **AND** 必须把相关 token、组件 reference、preview 和 dashboard kit 转译为 React/CSS 实现

#### Scenario: 区分设计资料和生产源码
- **WHEN** 开发者查看设计系统资产位置
- **THEN** 原始设计资料必须归档在 `../prototype/.design_library/minimal-dashboard/`，repo 内工作副本必须位于 `.codex/skills/minimal-dashboard-design/`

#### Scenario: 禁止直接复制 preview HTML
- **WHEN** agent 将 Minimal Dashboard 设计系统用于生产管理后台
- **THEN** 不得把 preview HTML、DOM 操作、内联脚本或示例页面直接复制到 `frontend/admin`

### Requirement: 管理后台自有 UI 基础层
系统 SHALL 在 `frontend/admin` 中提供自有的 Minimal Dashboard 风格 tokens、基础组件和后台布局，使后续页面不依赖 Ant Design 作为默认 UI 体系。

#### Scenario: 提供样式 tokens
- **WHEN** 管理后台页面渲染
- **THEN** 页面必须通过 `frontend/admin/src/styles/` 下的 Minimal Dashboard tokens 表达颜色、字体、间距、圆角、边框、状态色和基础阴影
- **AND** token 必须覆盖 Minimal Dashboard 的 color families、semantic aliases、字体族、4px spacing、主圆角和低阴影/零阴影策略

#### Scenario: 提供基础组件
- **WHEN** 管理后台页面需要按钮、卡片、输入、选择、开关、状态标记、tab、分页或列表展示
- **THEN** 页面必须优先使用 `frontend/admin/src/components/ui/` 下的自有组件，而不是直接使用 Ant Design 组件

#### Scenario: 提供后台布局
- **WHEN** 管理后台新增或修改页面
- **THEN** 页面必须复用 `frontend/admin/src/layouts/` 下的后台 shell、sidebar 和内容布局模式
- **AND** sidebar 必须使用品牌区、分组 label、图标菜单项和高对比 active 状态

#### Scenario: 截图验收
- **WHEN** 管理后台前端实现完成
- **THEN** 必须通过本地浏览器截图检查桌面视口下的 sidebar、按钮、卡片、表格和 tab 是否符合 Minimal Dashboard 风格
- **AND** 页面不得出现文本溢出、控件重叠或明显退回通用后台模板风格

### Requirement: Admin Token 登录页
系统 SHALL 提供独立登录页，使管理员输入 Admin Token 后进入管理后台，而不是在后台页面右上角临时输入 token。

#### Scenario: 未登录访问后台
- **WHEN** 管理员未提供或未保存 Admin Token
- **THEN** 系统必须展示符合 Minimal Dashboard 设计系统的登录页，而不是展示管理后台内容页

#### Scenario: 使用 Admin Token 登录
- **WHEN** 管理员在登录页输入 Admin Token 并提交
- **THEN** 系统必须保存 token 到本地会话状态，并在后续管理 API 请求中继续携带 `Authorization: Bearer <token>`

#### Scenario: 本地测试 token 提示
- **WHEN** 管理后台运行在本地测试阶段
- **THEN** 登录输入框下方可以展示当前测试 Admin Token 提示，便于开发者验证登录流程

#### Scenario: 退出登录
- **WHEN** 管理员点击退出登录
- **THEN** 系统必须清除本地保存的 Admin Token，并返回登录页

### Requirement: Admin Token 前端接入
系统 SHALL 允许管理后台通过 Admin Token 调用后端管理 API，并避免把 token 写入 repo 或前端源码。

#### Scenario: 输入 Admin Token
- **WHEN** 管理员首次访问管理后台或 token 失效
- **THEN** 页面必须允许管理员输入 Admin Token

#### Scenario: 请求携带 token
- **WHEN** 管理后台调用调度器或数据采集中心管理 API
- **THEN** 前端必须在请求头中携带 `Authorization: Bearer <token>`

#### Scenario: 不提交真实 token
- **WHEN** 开发者查看 repo 中的前端源码、配置和示例文件
- **THEN** 不得出现真实 Admin Token、模型 API key、搜索 API key 或数据库密码

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
