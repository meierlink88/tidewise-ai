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
首页 Report 的分享入口文案；页面栏目按 tidetell1.0 视觉基线显示“今日推理主线”。
它不承诺 Report 一定在当日生成；发生历史回退时继续展示该份 Report 的实际发布时间。
_Avoid_: 今日主题、隐藏回退来源

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
_Avoid_: Reason Tree、正式产业链动态查询、把展示示意连线解释为真实拓扑关系

**相关 Evidence**:
某一 Report 卡片、层、锚点、产业链或节点直接关联的 Atomic Evidence 产品投影。列表只展示发布时间、
摘要和有序关键词，接口保持 Report Evidence Link 的显式 `position`，前端按发布时间倒序展示；Evidence ID 只在
Data 内部用于持久化关联与诊断，Miniapp BFF 不向 Frontend 透出。
_Avoid_: 相关 Event、Event Evidence Link、改写服务端 position、Evidence 正文、来源技术元数据

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
- 首页展示筛选为全部、地缘政治、宏观经济、产业链。全部按地缘政治、宏观经济、产业链、历史概念的既有来源顺序展示；其中先展示 `industry_chain_analyses`，再展示历史 `concept_analyses`。两种来源保留真实类型与独立 cursor，相同 local_key 不跨类型去重；详情导航使用卡片来源类型。`company_analyses` 暂不接入。
- v5 的直接依据节点高亮来自 `judgment_origin=direct`，不从未来结论方向或 `conclusion_basis` 反推直接事实；变量信号在选中节点的核心分析中展示。

- API 保持 `/api/miniapp/v1` 与 `/api/data/v1`。`schema_version` 表示既有报告内容格式，不新增 URL 版本。
- 首页选中 v4/v5 时返回 `analysis_groups`，按 `geopolitical_stories`、`macroeconomic_stories`、
  `concept_analyses` 顺序各取首批 20 项；v5 额外读取 `industry_chain_analyses` 首批 20 项。每组独立保留 Data cursor；不拉取图谱或 Evidence 清单。
- `GET /reports/{report_id}/analyses/{kind}` 分页读取结论卡片；`/{analysis_key}` 读取目录及宏观锚点；
  `/{analysis_key}/industry-chains/{chain_key}` 读取该单元的产业链图谱和节点推理。
- v4 详情统一按宏观经济/产业链类型展示因果链 Tab。内容顺序为结论、关键机制、支持/反证，
  产业链再展示横向节点图谱和选中节点的核心分析，不展示后续验证。
- 图谱按报告节点顺序展示，节点之间和底部使用固定示意连线，不将其视为真实拓扑关系；API graph.edges 保留。结构节点缺少当期评估时不继承整链结论。
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

## Frontend Visual Baseline

- 后续 Miniapp 视觉对齐以用户选定的 `prototype/tidetell1.0/miniprogram.html` 为设计参考；
  该原型的静态示例不定义数据、API 或业务事实。本轮票据 #462 只对齐首页。
- 顶部展示“全球政经事件”和“读懂全球政经变化”；日期、星期和
  “截至 HH:mm”来自所选 Report 的 `publishedAt`，按上海时区展示。没有 Report 时不伪造日期。
  当前合同没有统计窗口，不展示原型的“过去24小时”，也不固定示例日期或时间。
- 首页默认“全部”；地缘政治、宏观经济和产业链筛选只改变已有卡片的展示范围。
  产业链与历史概念卡片同时可从全部和产业链视图访问，每个来源保持原有 cursor、路由身份和 Evidence scope。
- 栏目标题“今日推理主线”；数量使用当前筛选已加载卡片数，有后续页时明确标识
  “已加载 N 条主线”，不硬编码 3、不把已加载数量当作服务端总数。
- 首页使用原型墨蓝纹理、象牙白底、浅金标签与圆角卡片；样式 token 限于首页，
  保留报告全文和锚点方向。不添加统计、跟踪、传导阶段或“尚未显现”等数据字段。

## Frontend Routes And State

### WeChat Report Sharing

- 首页和推理详情通过微信右上角菜单支持发送给朋友/群及分享到朋友圈，不新增页面分享按钮。
- 首页分享标题为“观潮家 · 今日观潮”，打开时沿用最新 Report 选择规则，不固定分享时报告。
- 详情分享标题使用当前分析标题；加载期间使用“观潮家 · 深度分析”。朋友 path 与朋友圈
  query 只包含已验证的 `reportId + targetType + targetKey`，始终定位同一份 Report 分析。
  非法路由继续展示参数错误，不伪造或回退到其他报告。
- 微信场景 1154 为朋友圈单页模式：使用系统导航栏且挤压页面，首页不重复绘制自定义导航；
  跨页“查看影响路径”禁用，报告正文、搜索、分组、分页和页内证据弹层继续可用。
- 分享回调不主动调用分享菜单 API，不调用受限导航栏 API；详情标题由页面配置提供。
- 微信分享页面配置只进入 weapp 产物，tt 保持原有交互，不承诺朋友圈能力。
- 首页与详情的朋友/朋友圈分享均使用包内固定封面 `assets/share-cover.jpg`，通过根路径
  `imageUrl` 引用，不依赖网络图片。封面为用户定稿的手绘科技、大数据与 AI 主题，包含
  海浪 Logo 和故事线示例。朋友圈由微信按 1:1 展示，可能裁切 5:4 图片边缘。
  分享接入票据：#452；固定封面票据：#454。

### Routes

- 首页保持 `pages/index/index` 和既有应用/底部 Tab 框架。
- 推理详情注册为 `pages/report/detail/index`，query 为
  `reportId + targetType=geopolitical_stories|macroeconomic_stories|concept_analyses|industry_chain_analyses + targetKey`。
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
- 首页卡片的 `evidence_count` 显示为 `X 条政经事件`，不从节点或产业链数反推。
- 不支持旧无版本报告，不渲染旧层级卡片，不调用已退役 API。

## Report Detail Presentation

- 按用户后续确认，小程序详情不展示“后续验证”区块，涵盖宏观、产业链、节点详情及仅观察状态；follow_up 原数据字段、接口与解析保留。

- 支持/反证沿用原文字与数据，采用两张并排分支卡片，标题仅“支持”“反证”；蓝灰/浅金背景、顶部色边及圆点，通过分支线连接关键机制。正文完整换行，不生成原型示例标题、不截断，也不添加原型对话动作。

- 关键机制采用原型深蓝圆角问题卡片的视觉，不生成问句：标题“关键机制”，完整展示宏观 assessment.transmission_logic 或产业链 reasoning_summary.logic。原字段为字符串，不按箭头拆成节点；换行间以分号分隔成连续正文，保留条件和箭头。本链结论下 assessment.scope 仍直接使用原始字符串。

- 票据 #470 后续：Tab 下方用白色圆角底板承载推理内容；“本链结论”采用原型事件根节点的浅金渐变、描边及左侧金色内线。移除本链结论的方向、置信度、周期和判断来源标签，保留 assessment.conclusion；原型来源说明位置显示 assessment.scope，右下角保留原 Evidence 数量和点击路径；不复制原型的事件标题、来源、更新时间或关键词。
- 产业链图谱节点仅显示名称和方向标签：升温红色、降温绿色、分化黄色，直接依据节点白底高亮，选中状态独立于判断来源。核心分析采用浅色外框、白色内卡，显示选中节点名称、方向、affected_nodes[].assessment.conclusion，下方逐项 bullet 展示同一节点 variable_signals 的 variable_name、source_direction（上升/下降/稳定/分化/未知）、signal；可选信号缺失时不补造、不借用整链信号。无涨幅、周期、置信度、公司卡片或验证时间线。内容核对基线为 uat-trial-005 报告，tidetell1.0 只提供视觉样式。
- 白色推理底板到支持/反证结束；下方产业链图谱与核心分析是独立的淡黄色区域，核心分析内部的结论卡保持白色。

- 票据 #470：推理树 Tab 上方显示完整宏观或产业链名称，下方灰色小字显示“宏观经济”或“产业链”，不再使用分类小标签。选中项采用左侧金色竖线、深蓝名称；内容区为象牙白圆角顶部。保留原有横向滚动、选中状态、加载和各推理线正文，不增加原型的对话提示或能力。

- 票据 #468：导航标题为“深度分析”。顶部依次为故事线名称标签、所属报告发布时间、summary.conclusion 与 summary.transmission_logic；移除影响度标签及逻辑卡片边框。不展示顶部事件统计、跟踪状态，下方每条推理线的 Evidence scope/count 保持原有归属。
- Miniapp Service 拥有 UI 组合：读取详情后，使用现有 Data ListReports 每页 100 条及原 cursor，按精确 reportId 查找所属报告 published_at，作为详情根级可选 published_at 返回。Data API 与数据库不变，不读取完整领域报告或直接查库，不用首页最新时间替代历史报告时间。
- 查询沿用请求 context/超时；上游失败、循环 cursor 或列表耗尽未找到报告显式返回可重试错误。旧 BFF 未返回时间时前端保持正文可用、隐藏时间。展示为“X 分钟/小时/天前发布”，超过 7 天或未来时间使用上海绝对日期；不把发布时间表述成更新时间。

- 只接受三类分析单元详情目标；使用卡片详情中的 macro_impacts 和 industry_chains 目录。
- 产业链 Tab 按所属 report/kind/analysis/chain 读取，保留报告节点和评估归属，不动态补推；图谱连线为展示示意。
- 保留已定稿 UI：顶部结论、因果链 Tab、本链结论、关键机制、支持/反证、图谱和核心分析。

## Evidence Presentation

- 按票据 #464 与用户确认的 tidetell1.0 设计，顶部展示所属卡片的分析类型与故事线名称。
- Evidence 使用既有 `published_at`、`keywords`、`summary` 与可选的 `semantic_tags` 展示投影，按时间倒序展示；同时间保留接口顺序，空或无效时间置末。排序不修改接口数组或服务端 position。
- 每项依次展示时间、语义标签、完整 summary 浅灰正文块、原有 keywords。语义标签为后端预先整理的 `kind + text` 列表，依序来自 actors/action/objects/metrics；action 固定红色。不解析完整 semantic JSON，不生成标题、等级或推断指标。
- `/evidences` 原路由、scope_token 和原三字段保持不变，只追加可选 `semantic_tags`，不新增 include 参数。旧 Data 未返回标签时，BFF 与新版前端仍正常展示旧内容。先发布可忽略额外字段的兼容 Miniapp，再升级可接收可选字段的 BFF，最后升级 Data；不假设发布新版会立即替换所有旧会话。
- 采用暖灰底、圆点与竖线时间轴；空发布时间仍显示时间待确认。保留 scope 请求、加载、空态、重试和关闭行为。

## Response Compatibility

- Frontend 只读取并校验实际消费的字段，忽略响应 envelope 和嵌套 DTO 中新增的未知字段；新增字段不会令旧页面整体失败。
- 必需字段、类型、报告版本、report/scope 身份和图谱引用不变量继续校验；该兼容约定不放宽导航请求输入校验。
- 该策略随新版 Miniapp 发布生效，已安装的旧版严格客户端仍需 Backend 保持原有响应结构。语义扩展发布前不得假设所有旧客户端已升级。

## Frontend Mock Policy

真实 API 与开发 mock 必须实现同一 `ReportPort`。mock 只保留页面实际使用的固定 Report
fixture，并收敛到 `mocks/reports/`；API 失败不得静默回退 mock。构建期来源开关使用
`TARO_APP_REPORT_SOURCE=api|mock`，不保留旧领域命名的兼容变量。

## Runtime

Miniapp Application Backend 只通过其 Docker image 和 Compose 运行。Miniapp Frontend 不是
常驻 Service，也不进入 Docker runtime；Taro H5/weapp/tt watch/build 使用仓库锁定的
Node/Taro 依赖直接运行，并把 `dist/<platform>` 写入宿主机供微信或抖音开发者工具读取。
该运行方式不改变页面、Adapter、平台或 Backend 边界。
