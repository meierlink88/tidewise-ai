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

## Does Not Own

- Data PostgreSQL、migration、repository、Neo4j 或 Data domain model。
- Entity、Raw Document、Event、Research Theme 和 Research Anchor 的事实数据。
- Admin Portal contract。

## Product Language

- **研究主题（Research Theme）**：Data Context 拥有的研究结果事实，是 Miniapp 首页主线内容的数据来源。
- **推理主线**：研究主题在 Miniapp 面向用户展示时使用的产品名称，不是另一种数据实体。
- **主题卡片**：首页列表中呈现一条推理主线的界面单元，不拥有独立于研究主题的业务事实。
- **主题跟踪**：用户选择持续关注某个研究主题的产品行为；“跟踪中”数量是当前用户已跟踪的主题数，不是 Research Theme 的事实属性。
- **影响路径页**：从首页 Theme 卡片进入的一句话结论研究依据页。一个 Theme 页面可包含多条推理树，每条树对应一个以产业链节点为中心的 Research Anchor，页面通过 Tab 切换。
- **产品可见主题**：按现有 Theme 查询合同处于发布窗口内的 Research Theme。首页不依赖 Research Anchor 发布状态，也不增加 `has_reasoning_tree` 字段。

## Reasoning Trees API

- Miniapp Frontend 先调用 `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees` 获取 Theme 与全部 Anchor Tab 摘要。
- Miniapp Frontend 在某个 Tab 首次选中时调用 `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees/{anchor_id}` 获取单棵完整推理树。
- Miniapp BFF 将两个请求分别一对一代理到对应 Data API，并映射成页面可直接渲染的 DTO。
- BFF 成功响应直接使用 Data envelope 的 `result` 内容，不向小程序返回 Data `request_id/result` 外壳。
- BFF 保留每棵树的单一 `events` 数组；Miniapp Frontend 按 `evidence_role` 确定性过滤出原子事实、当前支持和当前反证，不复制数据或重排。
- BFF 不为一次请求扇出多个 Anchor 查询，不访问 PostgreSQL/Neo4j，不补写或推断研究内容。
- BFF 对 Theme 不存在、Theme 尚未发布推理树、Anchor 不属于该 Theme 三种 `404` 状态分别返回 `RESEARCH_THEME_NOT_FOUND`、`RESEARCH_REASONING_TREES_NOT_FOUND`、`RESEARCH_REASONING_TREE_NOT_FOUND`；它们是 Miniapp 的稳定错误语义，不透传 Data 的 request ID 或错误外壳。
- 现有 Theme 详情 API 保持不变。
- 删除尚未正式使用的旧 `/api/v1/miniapp/research/anchors` 列表和按 Anchor ID 读取接口；不保留兼容别名，Research Anchor 统一作为 Theme 下的推理树子资源读取。

## Reasoning Trees Frontend Route

- 影响路径页固定注册为 `pages/research-theme/reasoning-trees/index`。
- 首页 Theme 卡片仅由“查看影响路径”按钮使用 `Taro.navigateTo` 跳转到 `/pages/research-theme/reasoning-trees/index?theme_id=<uuid>`；整张卡片、产业链节点和事件数量不触发该导航。
- 页面是非 Tab 页面，不引入自定义路由器；推理树 V1 以微信和抖音小程序为目标平台，不实现或验收 H5 专属路由、刷新与深链行为。
- `theme_id` 缺失或不是标准小写 UUID 时，页面展示参数错误且不得请求 BFF。
- 页面数据访问必须经过独立 typed port 和 adapter，页面组件不得直接实现 HTTP 调用。
- 页面打开后加载 Tab 摘要和排序后的第一棵树；其他 Tab 首次选中时才加载详情。
- Tab 摘要可用后所有 Tab 立即允许切换；各 Tab 的详情请求与 loading、ready、error 状态相互独立，切换 Tab 不取消其他在途请求，也不允许较晚完成的请求覆盖当前选中项。
- 已成功加载的单树按 `anchor_id` 缓存在当前页面会话，再次切换不重复请求；重新进入或刷新页面时重新加载。
- 单个 Tab 详情加载失败时，仅该 Tab 内容区显示错误与重试操作；其他已加载缓存保持可用，页面不自动切换 Tab。
- 单 Tab 重试只请求当前 `anchor_id` 的详情，不连带刷新列表或其他推理树。
- Theme 不存在时，小程序展示“该研究主题暂不可用”；Theme 存在但推理树尚未发布时展示“影响路径暂未生成”。两种状态均提供返回操作，且不向用户暴露内部错误码。
- 列表网络或服务故障展示可重试错误；推理树列表不存在合法空集合，因此不设计正常空态。

## Frontend Mock Policy

真实 API 尚未接入时，可以在 Miniapp Frontend 内保留仍被页面使用的 mock。mock 必须收敛到明确的 `mocks/` 或 `devdata/` 目录，并通过可替换 adapter 注入。未被页面、测试或开发场景引用的 mock、model、service 和 component 应删除。

本次源码治理不把 Miniapp Frontend 接入真实 BFF；该行为变更单独实施。

推理树前端与首页 Theme 列表共用构建期变量 `TARO_APP_RESEARCH_SOURCE`。`mock` 模式下两者都使用匹配的 Mock Adapter，`api` 模式下两者都调用真实 Miniapp BFF；不增加推理树专属开关，也不允许 API 失败后静默回退到 mock。TW-06 使用共享 fixture 验收页面状态，并实现 API Adapter 及合同测试；真实 Data、BFF 与小程序全链路验收留给 TW-08。
