# Miniapp Context

## Purpose

Miniapp 是用户产品系统，由 Miniapp Frontend 和 Miniapp Application Backend Service 组成。

## Dependency Rule

Miniapp Frontend 只能调用 Miniapp Application Backend Service。Miniapp Application Backend
Service 需要 Data、User 或 Payment 能力时，只能调用对应 Domain Service 的 REST API。

## Application Backend Service Owns

- Miniapp 对外 API、认证入口和前端专用 DTO。
- 多个 Domain Service 的产品编排。
- Data API 错误、分页、时间和字段到 Miniapp contract 的转换。
- “上海当日最新一份优先、当日没有则历史最新一份”的首页 Report 选择语义。
- Miniapp 专用缓存和降级策略，但不拥有 Data 事实。

## Backend Implementation

Miniapp Application Backend Service 是完整 Kratos v3 Application：

- `api/miniapp/v1` 保存 OpenAPI 3.0.4、wire DTO 和 HTTP 绑定；
- `cmd/server` 显式构造依赖并运行 `kratos.App`；
- `internal/service` 实现 Miniapp API；
- `internal/biz` 承载 Report Use Case、产品时间窗口和 `ReportRepo` Port；
- `internal/data` 使用 Kratos HTTP Client 调用 Data Service；
- `internal/server` 拥有 Kratos HTTP Server、Request ID、Recovery、错误 envelope、
  health/readiness 与 Swagger UI；
- `internal/conf` 和 `configs` 承载本地/UAT 启动配置。

Miniapp 保持 HTTP-only 和固定 Data Service URL，不使用 gRPC、服务发现、Wire 或远程配置
中心。Miniapp binary dependency closure 不得包含 Gin。

## Does Not Own

- Data PostgreSQL、migration、repository、Neo4j 或 Data domain model。
- Entity、Raw Evidence、Atomic Evidence、Event 或 Report 正式事实。
- AgentOS 推理、报告转换、报告发布或运行状态。
- Admin Portal contract。

## Product Language

**首页 Report**:
首页本次会话展示的零或一份不可变 Data Report。Miniapp Backend 在上海自然日当日有发布时，
只选择 `published_at DESC, id ASC` 排序的第一份；当日没有发布时，回退到全部历史的最新一份。
Data 仍保留所有已发布 Report，首页选择不删除、合并或修改历史报告。
_Avoid_: 今日 Theme、当日多 Report Tab、前端自行排序或选择、跨 Report 聚合

**今日观潮**:
首页 Report 的产品入口标题。它不承诺 Report 一定在当日生成；发生历史回退时继续展示
该份 Report 的实际发布时间。
_Avoid_: 今日主题、今日推理、隐藏回退来源

**Report 分析投影**:
每份 Report 的实际分析 Section 及产业链分页摘要。产业链必有；地缘政治与宏观经济按本次报告
是否存在投影，不生成空卡片；公司能力上线前不生成占位层。卡片是 Miniapp Backend 从同一份
不可变、扁平 AgentOS Report 发布快照构造的产品 DTO，不是 Data 持久化事实，并始终保留所属
`report_id`。
_Avoid_: 固定四层、空层占位、Data 持久化首页卡片、跨 Report 聚合

**Report 卡片详情目标**:
每张首页 Report 卡片由对应 Section 或产业链的 Data 读取投影提供结构化详情目标。Miniapp 只把 `report_id`、目标类型
和 Report-local Key 传入 Taro 非 Tab 详情页；Evidence 入口只使用 Data 签发、绑定 Report 的 opaque scope token。
_Avoid_: 从标题解析路由、前端检索完整 Report JSON、Reason Tree ID

**产业链推理详情**:
一条 Report-owned 产业链快照的独立详情页。图节点和边只来自该 Report；相同名称不证明
存在正式 Data IndustryChain 或 ChainNode 关系。
_Avoid_: Reason Tree、正式产业链动态查询、把无边节点串联

**相关 Evidence**:
某一 Report 卡片、层、锚点、产业链或节点直接关联的 Atomic Evidence 产品投影。列表只展示发布时间、
摘要和有序关键词，列表项保持 Report Evidence Link 的显式 `position`；Evidence ID 只在
Data 内部用于持久化关联与诊断，Miniapp BFF 不向 Frontend 透出。
_Avoid_: 相关 Event、Event Evidence Link、按时间自行重排、Evidence 正文、来源技术元数据

## Home Report Selection

- Miniapp Backend 使用 `Asia/Shanghai` 计算当日 `[00:00, next 00:00)`，转为 UTC 后向 Data
  Report 列表以 `limit=1` 读取该范围内的最新结果。
- 当日查询非空时只返回 Data `published_at DESC, id ASC` 权威顺序的第一份 Report；当日查询
  为空时，Backend 立即以 `limit=1` 查询全部历史的最新一份。
- 首页只投影选中 Report 的分析摘要；详情和 Evidence 导航必须携带所属 `report_id`。
- 历史回退查询按 Data 的
  `published_at DESC, id ASC` 权威顺序。
- 全部没有 Report 是正常产品空态，不生成占位 Report、不回退 mock、不读取数据库。
- Report 已发布后不可变；Backend 不缓存或拼接研究语义，只把 Data 投影映射为 Miniapp DTO。
- 首页刷新重新执行完整选择流程。刷新失败保留本会话最近一次成功内容，并显示可重试错误；
  旧请求晚到不得覆盖更新后的 Report。

## Report API

- `GET /api/miniapp/v1/reports/home` 返回当日最新 Report 的元数据和可选上层摘要；当日为空时返回历史最新一份，
  全部为空时返回明确空集合。
- `GET /api/miniapp/v1/reports/home` 为选中 Report 返回可选上层卡片和首个产业链 page，同时返回
  `next_cursor`；BFF 不完整消费产业链集合，也不解码完整 Report JSON。
- `GET /api/miniapp/v1/reports/{report_id}/industry-chains?limit=&cursor=` 原样推进 Data 主导的
  report-bound cursor，并返回下一批产业链卡片。
- `GET /api/miniapp/v1/reports/{report_id}/layers/{layer_key}` 读取
  `geopolitics | macroeconomics` 上层详情，并通过 Data cursor 完整消费该 Report 的产业链摘要，
  供详情页末尾展示全部产业链入口；不读取单链 detail。
- `GET /api/miniapp/v1/reports/{report_id}/industry-chains/{chain_key}` 一对一读取单条产业链详情。
- `GET /api/miniapp/v1/reports/{report_id}/evidences?scope_token=` 一对一读取相关 Evidence。
- BFF 成功响应只返回 Miniapp DTO，不透传 Data `result/request_id` envelope、URL、
  SQL 或内部错误。
- Report、层、产业链或 Evidence scope 不存在时返回稳定 Miniapp 错误分类；网络/下游错误
  保持显式可重试，不伪造空集合。
- BFF 不扇出读取 Event、IndustryChain、ChainNode 或 Company，不补写或推断报告内容。

## Frontend Routes And State

- 首页保持 `pages/index/index` 和既有应用/底部 Tab 框架。
- 推理详情注册为 `pages/report/detail/index`，query 为
  `reportId + targetType=layer|industry_chain + targetKey`；`layer` 的 `targetKey` 只允许
  `geopolitics | macroeconomics`。
- 详情页是非 Tab 页面，使用官方 `Taro.navigateTo`/`navigateBack`，不引入自定义 Router；
  query 输入不可信，缺失、重复或非法参数必须在请求前进入明确参数错误状态。
- 首页与详情页的 Evidence 入口打开当前页面管理的底部抽屉，不切换路由；抽屉使用
  `ReportPort.getEvidences` 按当前 opaque scope token 延迟加载。
- 微信端的 Evidence 抽屉由页面会话内常驻的 `RootPortal` Host 承载；打开和关闭只更新
  Portal 子树，不改变首页或详情页的原生根结构，不使用手工滚动位置恢复。
- 首页、详情和 Evidence 抽屉分别拥有 `loading | ready | empty/not-found | error` 状态与重试；
  route 参数变化或重新进入时，较早请求不得覆盖新状态。
- 已成功读取的不可变详情可以在当前页面会话内按 Report/scope 缓存；重新进入页面重新读取。

## Homepage Presentation

- 首页标题固定为 `今日观潮`，只展示选中 Report 的实际发布时间，不提供当日多 Report Tab。
- 选中 Report 的地缘政治和宏观经济仅在对应 Section 存在时各展示一张摘要卡片。
- 上层摘要卡片直接使用 AgentOS 发布的 Section 根结论、结果、置信度和时间窗口；BFF 不从锚点
  聚合或从传导步骤反推。
- 选中 Report 的产业链卡片按 JSON 数组顺序全量可达；Frontend 在固定内部 ScrollView 中每次
  追加一个 bounded page，按 card local key 去重，并隔离刷新后晚到的旧响应。
- 产业链标题的总数使用 Report 发布快照中的 `industry_chain_count`，首页展示数使用本组
  已持久化的产业链卡片数；两者不要相互反推或硬编码。
- 公司分析拥有定稿发布基线前不显示空边界，也不从 Company 正式事实生成卡片。
- 状态同时显示中文文字与颜色；已知 code 使用 `升温 / 降温 / 分化 / 待验证`
  的规范样式，未知 code 保留发布方 label 并使用中性样式。锚点或节点每行名称靠左，结果、置信度
  和时间窗口统一靠右并自然换行。
- 卡片 Evidence 入口显示为 `依据`，带 `查看证据` 可访问名称，不显示 ID 或数量。

## Report Detail Presentation

- 地缘政治和宏观详情从各自一句话结论、影响锚点与“为什么”开始，再展示独立反转条件和
  向下传导；不展示已废弃的“推理步骤”或合并式“不确定性与反转条件”区块，不引入报告之外的研究判断。
- 传导目标保留 AgentOS 发布的宏观锚点、产业链和产业链节点结构化引用；v1 只允许产业链
  目标进入独立详情页，锚点与链节点目标仅展示，不生成无法加载的跳转。
- 上层详情末尾列出同一 Report 中全部产业链名称与结果，数量必须与 Report 摘要一致；
  上层页不嵌入链节点或产业链推理图，选择产业链进入独立详情页。
- 产业链详情先展示名称、一句话结论、结果、时间窗口和置信度；AgentOS 发布的可空
  `path_summary/accepted_hypothesis_summary` 仅在产品确认需要的位置展示，不由 BFF 生成。
- 图在一个横向 `ScrollView` 画布中布局全部 topology nodes，只绘制 Report 显式有向边；
  节点 `local_key` 同时是链图端点；未评估的结构节点仍保留但不可打开详情。
  长边使用独立正交通道和端口，不把没有边的相邻节点表现成关系。
- 图节点和选中节点卡都将 `直接证据 / 推理假设` 结论依据与 `待验证` 验证状态分开展示。
  选中节点额外展示本次影响、传导逻辑，以及链级反证与 Gap、停止条件。

## Evidence Presentation

- Evidence 底部抽屉直接从 Report Evidence Link 的显式顺序列表开始，不显示内部 scope
  标题、来源类型、关系立场、Evidence ID 或技术边界说明。
- 每项只展示 `published_at`、`summary` 和有序 `keywords`；空发布时间显示明确的时间待确认。
- Keywords 使用有边框的蓝色轻量 chip；发布时间与摘要优先级更高。
- 页面不按 `published_at` 重排；发布时间只是列表项属性，不添加装饰时间线点或连线。

## Frontend Mock Policy

真实 API 与开发 mock 必须实现同一 `ReportPort`。mock 只保留页面实际使用的固定 Report
fixture，并收敛到 `mocks/reports/`；API 失败不得静默回退 mock。构建期来源开关使用
`TARO_APP_REPORT_SOURCE=api|mock`，不保留旧领域命名的兼容变量。

## Runtime

Miniapp Application Backend 只通过其 Docker image 和 Compose 运行。Miniapp Frontend 不是
常驻 Service，也不进入 Docker runtime；Taro H5/weapp/tt watch/build 使用仓库锁定的
Node/Taro 依赖直接运行，并把 `dist/<platform>` 写入宿主机供微信或抖音开发者工具读取。
该运行方式不改变页面、Adapter、平台或 Backend 边界。
