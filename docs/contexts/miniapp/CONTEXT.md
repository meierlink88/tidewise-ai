# Miniapp Context

## Purpose

Miniapp 是用户产品系统，由 Miniapp Frontend 和 Miniapp Application Backend Service 组成。

## Dependency Rule

Miniapp Frontend 只能调用 Miniapp Application Backend Service。Miniapp Application Backend Service 需要 Data、User 或 Payment 能力时，只能调用对应 Domain Service 的 REST API。

## Application Backend Service Owns

- Miniapp 对外 API、认证入口和前端专用 DTO。
- 多个 Domain Service 的产品编排。
- Data API 错误、分页、时间和字段到 Miniapp contract 的转换。
- Miniapp 专用缓存和降级策略，但不拥有 Data 事实。

## Backend Implementation

Miniapp Application Backend Service 是仓库内首个完整 Kratos v3 Application：

- `api/miniapp/v1` 保存 OpenAPI 3.0.4、wire DTO 和 HTTP 绑定；
- `cmd/server` 显式构造依赖并运行 `kratos.App`；
- `internal/service` 实现 Miniapp API；
- `internal/biz` 承载 Research Use Case、业务校验和 `ResearchRepo` Port；
- `internal/data` 使用 Kratos HTTP Client 调用 Data Service；
- `internal/server` 拥有 Kratos HTTP Server、Request ID、Recovery、错误 envelope、
  health/readiness 与 Swagger UI；
- `internal/conf` 和 `configs` 承载本地/UAT 启动配置。

Miniapp 保持 HTTP-only 和固定 Data Service URL，不使用 gRPC、服务发现、Wire 或
远程配置中心。Miniapp binary dependency closure 不得包含 Gin。

## Does Not Own

- Data PostgreSQL、migration、repository、Neo4j 或 Data domain model。
- Entity、Raw Document、Event、Research Theme、Theme Impact 和 Reason Tree 的事实数据。
- Admin Portal contract。

## Product Language

- **研究主题（Research Theme）**：Data Context 拥有的研究结果事实，是 Miniapp 首页主线内容的数据来源。
- **推理主线**：研究主题在 Miniapp 面向用户展示时使用的产品名称，不是另一种数据实体。
- **主题卡片**：首页列表中呈现一条推理主线的界面单元，不拥有独立于研究主题的业务事实。
- **主题跟踪**：用户选择持续关注某个研究主题的产品行为；“跟踪中”数量是当前用户已跟踪的主题数，不是 Research Theme 的事实属性。
- **主题影响（Theme Impact）**：Theme 关注的 Chain Node 集合，节点之间没有主次；首页按
  Data 稳定顺序展示名称及由 `impact_direction` 机械映射的机会/风险/不确定判断，不展示
  `relation_role`、`impact_summary` 或变量状态。
- **影响路径页**：从首页 Theme 卡片进入的研究依据页。一个 Theme 页面可包含多棵 Reason Tree，每棵 Tree 对应一条 Industry Chain 推导链路，页面通过 Tab 切换。
- **产品可见主题**：按 Theme 查询合同处于发布窗口内的 Research Theme。首页不依赖 Reason Tree 发布状态；零 Tree Theme 仍保留入口，由影响路径页展示“影响路径暂未生成”。

## Theme Homepage

- 首页不展示没有真实用户数据合同的分类栏或“跟踪中”数量；Theme 搜索继续只在当前
  feed 内生效。
- 首页使用 Taro 页面级原生下拉刷新重新读取 Theme feed；刷新失败时保留最近一次成功
  数据，并在所有结果路径结束原生刷新状态。
- Theme 卡片的非零“政经事件”数量只打开当前页面内的关联事件底部面板；它不触发
  Reason Tree 导航，零事件数量保持只读。
- 关联事件面板按 Theme 详情 API 的稳定顺序纵向展示完整事件列表，每条只展示
  `event_time`、`title` 和 `summary`；空时间显示“时间待确认”，不展示
  `evidence_role`、`supported_claim`、来源、事件分类或额外详情入口。
- Theme 详情按 `theme_id` 缓存在当前首页会话，feed 刷新成功后失效并重新读取当前已
  打开的 Theme；旧请求晚到不得覆盖刷新后或新选中的 Theme。
- 事件面板使用页面局部状态和基础 Taro 组件覆盖微信、抖音小程序；面板滚动与触控不得
  传递到底层页面或触发页面下拉刷新。

## Reasoning Trees API

- Miniapp Frontend 先调用 `GET /api/miniapp/v1/research/themes/{theme_id}/reasoning-trees` 获取 Theme 与全部 Reason Tree Tab 摘要。
- Miniapp Frontend 在某个 Tab 首次选中时调用 `GET /api/miniapp/v1/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}` 获取单棵完整推理树。
- Miniapp BFF 将两个请求分别一对一代理到对应 Data API，并映射成页面可直接渲染的 DTO。
- BFF 成功响应直接使用 Data envelope 的 `result` 内容，不向小程序返回 Data `request_id/result` 外壳。
- BFF 保留每棵树的单一 `events` 数组、Tree 摘要、节点和 Variable Signal 展示快照，不拼接、推断或重排研究语义。
- BFF 不为一次请求扇出多个 Tree 查询，不访问 PostgreSQL/Neo4j，不补写或推断研究内容。
- BFF 对 Theme 不存在、Theme 尚未发布推理树、Tree 不属于该 Theme 三种 `404` 状态分别返回 `RESEARCH_THEME_NOT_FOUND`、`RESEARCH_REASONING_TREES_NOT_FOUND`、`RESEARCH_REASONING_TREE_NOT_FOUND`；它们是 Miniapp 的稳定错误语义，不透传 Data 的 request ID 或错误外壳。
- 现有 Theme 详情 API 保持不变。
- 不提供旧 Research Anchor API、字段或兼容别名；Reason Tree 只作为 Theme 子资源读取。

## Reasoning Trees Frontend Route

- 影响路径页固定注册为 `pages/research-theme/reasoning-trees/index`。
- 首页 Theme 卡片仅由“推导详情”按钮使用 `Taro.navigateTo` 跳转到
  `/pages/research-theme/reasoning-trees/index?theme_id=<uuid>`；整张卡片、关注节点和
  事件数量不触发该导航。
- 页面是非 Tab 页面，不引入自定义路由器；推理树 V1 以微信和抖音小程序为目标平台，不实现或验收 H5 专属路由、刷新与深链行为。
- `theme_id` 缺失或不是标准小写 UUID 时，页面展示参数错误且不得请求 BFF。
- 页面数据访问必须经过独立 typed port 和 adapter，页面组件不得直接实现 HTTP 调用。
- 页面打开后加载 Tab 摘要和排序后的第一棵树；其他 Tab 首次选中时才加载详情。
- Tab 摘要可用后所有 Tab 立即允许切换；各 Tab 的详情请求与 loading、ready、error 状态相互独立，切换 Tab 不取消其他在途请求，也不允许较晚完成的请求覆盖当前选中项。
- 已成功加载的单树按 `reasoning_tree_id` 缓存在当前页面会话，再次切换不重复请求；重新进入或刷新页面时重新加载。
- 单个 Tab 详情加载失败时，仅该 Tab 内容区显示错误与重试操作；其他已加载缓存保持可用，页面不自动切换 Tab。
- 单 Tab 重试只请求当前 `reasoning_tree_id` 的详情，不连带刷新列表或其他推理树。
- Theme 不存在时，小程序展示“该研究主题暂不可用”；Theme 存在但推理树尚未发布时展示“影响路径暂未生成”。两种状态均提供返回操作，且不向用户暴露内部错误码。
- 列表网络或服务故障展示可重试错误；推理树列表不存在合法空集合，因此不设计正常空态。

## Reasoning Tree Page Presentation

- 页面视觉与交互唯一基准为 `prototype/theme-direct-impact-investment-outlook-prototype.html`；原型业务文本仅为样例，正式页面全部由 API 数据生成。
- Reason Tree Tab 使用产业链名称；切换 Tab 后滚动到新树内容顶部，只复用数据缓存，不保存每个 Tab 的历史阅读位置。
- 原子事件按 BFF 稳定顺序全部展示，不折叠；每条显示标题、摘要和可用时间，不展示内部证据角色。
- 当前支持与当前反证是 Tree 级结论性描述；无反证时保留卡片并显示“当前暂无明确反证”。
- 产业链路径使用横向 ScrollView 展示全部紧凑节点与箭头，默认选择最大 `position` 节点；选择节点只更新下方单个详情面板。
- 紧凑节点只展示节点序号/名称、primary Signal `display_summary`、机会/风险/不确定判断和影响强度，不展示第二个 Signal、数据缺口或选择状态文案。
- 节点详情展示节点序号/名称、机会/风险/不确定判断、影响强度、完整有序 `signals[]` 变量状态和节点 `impact_summary` 投资含义；不展示 Signal 内部角色。
- 位置大于 1 时继续展示所选节点的真实 `incoming_*` 传导标题、机制和成立条件；首节点不展示或伪造传入关系。
- 页面不展示“直接影响节点”“后续推导节点”“直接/间接”“信号入口”“路径节点”“结果节点”“变量信号”“推导依据”“数据缺口”或派生的节点路由标签。
- “判断边界”只展示 `conclusion_boundary_summary`；“后续验证”按分析师顺序展示完整 `checkpoints[].summary`，不与 `invalidation_conditions` 按索引组合。
- 所有已展示的研究文本自然换行并完整展示，不使用省略号截断；未展示的 Evidence、依据、缺口、失效条件与强血缘继续保留在现有合同中。

## Frontend Mock Policy

真实 API 尚未接入时，可以在 Miniapp Frontend 内保留仍被页面使用的 mock。mock 必须收敛到明确的 `mocks/` 或 `devdata/` 目录，并通过可替换 adapter 注入。未被页面、测试或开发场景引用的 mock、model、service 和 component 应删除。

本次源码治理不把 Miniapp Frontend 接入真实 BFF；该行为变更单独实施。

推理树前端与首页 Theme 列表共用构建期变量 `TARO_APP_RESEARCH_SOURCE`。`mock` 模式下两者都使用匹配的 Mock Adapter，`api` 模式下两者都调用真实 Miniapp BFF；不增加推理树专属开关，也不允许 API 失败后静默回退到 mock。TW-06 使用共享 fixture 验收页面状态，并实现 API Adapter 及合同测试；真实 Data、BFF 与小程序全链路验收留给 TW-08。
