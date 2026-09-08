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
Report v4/v5 按地缘政治故事线、宏观经济故事线、产业链所属 Concept 分为三个分页分组；
一条故事线或一个 Concept 对应一张结论卡片，空分组不生成占位卡片。BFF 只读取 Data 的
摘要、因果链目录和单链详情投影，不解码完整发布快照。旧无版本报告与旧层级/产业链独立读取接口已退役。
所有卡片、详情和证据始终绑定所属 `report_id`。
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

## Report v4/v5 integration

- v8 报告基线使用 `report-publication/v5`；Miniapp 同时支持 v4/v5，URL 仍为 v1。
- v5 显式透传 `judgment_origin`、`reasoning_sources`、`variable_signals`、`graph.scope` 和详情中的 `companies`；变量信号保留 Data 签发的 evidence scope token/count，不暴露 Evidence ID。
- 当前首页仍为三个既有分组；独立 `industry_chain_analyses`、`company_analyses` 的新入口另行设计，本次不合并到已有概念、不伪造故事线。
- v5 的直接/推理标签来自 `judgment_origin`，不从未来结论方向或 `conclusion_basis` 反推直接事实；变量信号在 typed 数据层保留，展示另行设计。

- API 保持 `/api/miniapp/v1` 与 `/api/data/v1`。`schema_version` 表示既有报告内容格式，不新增 URL 版本。
- 首页选中 v4/v5 时返回 `analysis_groups`，按 `geopolitical_stories`、`macroeconomic_stories`、
  `concept_analyses` 顺序各取首批 20 项，每组独立保留 Data cursor；不拉取图谱或 Evidence 清单。
- `GET /reports/{report_id}/analyses/{kind}` 分页读取结论卡片；`/{analysis_key}` 读取目录及宏观锚点；
  `/{analysis_key}/industry-chains/{chain_key}` 读取该单元的产业链图谱和节点推理。
- v4 详情统一按宏观经济/产业链类型展示因果链 Tab。内容顺序为结论、关键机制、支持/反证，
  产业链再展示完整横向图谱、选中节点的支持/反证/后续验证，最后是整链后续验证。
- 图谱只用报告显式拓扑边；结构节点缺少当期评估时不继承整链结论；仅观察不展示虚构置信度。
- 所有 `evidence_count` 原样使用 Data 发布时计算的 scope 内去重 Evidence 数量，
  与 opaque token 打开的清单一致；不以节点或因果链数量替代证据数量。
- 最新报告读取失败或格式不支持时显式报错，不跳过该报告改选旧报告。旧无版本快照已退役，明确报错。

## Report API

Miniapp Report 仅保留以下五个 GET 接口（前缀 `/api/miniapp/v1`）：

- `/reports/home`：上海当日最新优先，无当日报告则历史最新；只接受 v4/v5，旧无版本报告明确返回 503，不跳过最新报告或静默回退 mock。
- `/reports/{report_id}/analyses/{kind}`：三类卡片独立分页。
- `/reports/{report_id}/analyses/{kind}/{analysis_key}`：卡片详情、宏观影响与产业链目录。
- `/reports/{report_id}/analyses/{kind}/{analysis_key}/industry-chains/{chain_key}`：所属单元下的产业链图谱及节点判断。
- `/reports/{report_id}/evidences?scope_token=`：按 opaque token 读取事件展示清单。

旧 `/reports/{report_id}/industry-chains`、`/reports/{report_id}/layers/{layer_key}`、
`/reports/{report_id}/industry-chains/{chain_key}` 已删除，返回 404；不保留旧格式首页展示分支。
本次只清理 Miniapp 消费与公开接口，Data Service API 和持久化报告不变。旧客户端/旧链接不再兼容，回滚应用可恢复旧行为。

BFF 不读取完整报告或直接查询领域数据库；失败保持显式可重试，响应不透出内部错误。

## Frontend Routes And State

- 首页保持 `pages/index/index` 和既有应用/底部 Tab 框架。
- 推理详情注册为 `pages/report/detail/index`，query 为
  `reportId + targetType=geopolitical_stories|macroeconomic_stories|concept_analyses + targetKey`。
  旧 `layer|industry_chain` 路由在请求前进入参数错误状态。
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

- 标题固定为 `今日观潮`，每张卡片展示实际发布时间；三类分组、搜索、分页与事件清单保留。
- v4/v5 使用已定稿 normalized 首页：故事线/概念一张卡片，摘要、传导逻辑、受影响锚点与结果。
- `evidence_count` 显示为 `X 条事件`，不从节点或产业链数反推。
- 不支持旧无版本报告，不渲染旧层级卡片，不调用已退役 API。

## Report Detail Presentation

- 只接受三类分析单元详情目标；使用卡片详情中的 macro_impacts 和 industry_chains 目录。
- 产业链 Tab 按所属 report/kind/analysis/chain 读取，保持显式图谱与完整节点推理，不动态补推。
- 保留已定稿 UI：顶部结论、因果链 Tab、本链结论、关键机制、支持/反证、图谱、节点详情及后续验证。

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
