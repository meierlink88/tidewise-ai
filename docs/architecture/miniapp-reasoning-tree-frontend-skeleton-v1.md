# Miniapp 推理树前端骨架合同 V1

## 状态

- 所属任务：TW-06
- 当前状态：Spec、Implement、验证、Code Review 与 PR Review 已完成，PR #69 已合并并验收
- 前置条件：TW-05 Miniapp BFF 已合并至 `main`

## 用途

本任务为“一句话结论推理树”建立可运行、可独立测试的小程序前端骨架。用户从首页 Theme 卡片的“查看影响路径”按钮进入详情页，页面读取 Theme 下的 Research Anchor 摘要，并按 Tab 首次选中时加载对应推理树详情。

本任务只固定路由、数据访问边界和页面状态，不实现原型 A 的最终视觉与完整内容排版。正式推理树页面由 TW-07 完成，真实 Data → BFF → Miniapp 全链路验收由 TW-08 完成。

## Taro 参考结论

- 使用项目现有 Taro 4、React 18 和 TypeScript 技术栈，不引入第二套路由或状态框架。
- 非 tabBar 详情页使用 [`Taro.navigateTo`](https://docs.taro.zone/docs/apis/route/navigateTo)；页面通过 Taro 路由参数读取 `theme_id`。
- API Adapter 使用 [`Taro.request`](https://docs.taro.zone/docs/apis/network/request/)；页面组件不能直接发送 HTTP 请求。
- V1 目标平台为微信小程序和抖音小程序；H5 专属路由、刷新、深链与兼容性不在本任务范围。

## 页面路由

- 注册非 tabBar 页面 `pages/research-theme/reasoning-trees/index`。
- 首页只有 Theme 卡片的“查看影响路径”按钮触发导航：

```text
/pages/research-theme/reasoning-trees/index?theme_id=<uuid>
```

- 整张 Theme 卡片、产业链节点和事件数量不触发推理树导航。
- `theme_id` 必须是标准小写 UUID。缺失或非法时展示参数错误，不调用 BFF。

## 数据源

首页 Theme 列表与推理树页面共用现有构建期变量 `TARO_APP_RESEARCH_SOURCE`，不新增推理树专属开关：

| 值     | 首页               | 推理树页            |
| ------ | ------------------ | ------------------- |
| `mock` | Theme Mock Adapter | 推理树 Mock Adapter |
| `api`  | Theme API Adapter  | 推理树 API Adapter  |

- `api` 模式失败时不得静默回退到 mock。
- `mock` 模式使用 `src/testdata/reasoning-tree-v1/` 中已冻结的共享 fixture，不维护第二套业务样例。
- API Adapter 在 TW-06 内完整实现并通过 HTTP 合同测试；本任务不要求连接正在运行的真实后端进行验收。

## Typed Port

页面只依赖稳定的前端 Port，不感知 `Taro.request`、BFF URL 或 snake_case 传输格式：

```ts
interface ResearchReasoningTreePort {
  list(themeId: string): Promise<ReasoningTreeIndex>;
  get(themeId: string, anchorId: string): Promise<ReasoningTreeDetail>;
}
```

- `ReasoningTreeIndex` 覆盖 BFF 列表响应中的 Theme 和全部 Anchor Tab 摘要。
- `ReasoningTreeDetail` 覆盖单棵推理树、事件证据和有序传导节点。
- API wire DTO 使用 BFF 的 snake_case；转换后的页面模型使用项目现有 TypeScript 命名惯例。
- Mock Adapter 与 API Adapter 必须实现同一 Port，并返回相同页面模型。

## 请求与加载

页面首次进入：

1. 校验 `theme_id`。
2. 调用列表接口读取 Theme 与全部 Tab 摘要。
3. 保留服务端返回的 Tab 稳定顺序。
4. 自动选中第一个 Tab，并请求该 Anchor 详情。
5. 其他 Tab 在首次选中时请求详情。

Tab 摘要加载成功后，所有 Tab 立即可切换。各 Anchor 的详情请求相互独立：

- 切换 Tab 不取消其他在途请求。
- 较晚完成的请求只能更新自身 `anchor_id` 的缓存，不能覆盖当前选中项。
- 已成功加载的详情按 `anchor_id` 缓存在当前页面会话，再次切换不重复请求。
- 重新进入页面会创建新会话并重新加载，不做跨页面持久缓存。
- 单 Tab 重试只重试该 Anchor，不刷新列表或其他 Anchor。

## 页面状态

页面至少显式表达以下状态：

| 层级   | 状态                | 行为                                |
| ------ | ------------------- | ----------------------------------- |
| 路由   | `invalid`           | 展示参数错误，不请求 BFF            |
| 列表   | `loading`           | 展示页面级加载状态                  |
| 列表   | `ready`             | 展示 Theme、Tab 骨架和当前 Tab 内容 |
| 列表   | `themeUnavailable`  | 展示“该研究主题暂不可用”和返回操作  |
| 列表   | `treesNotPublished` | 展示“影响路径暂未生成”和返回操作    |
| 列表   | `error`             | 展示页面级错误和列表重试操作        |
| 单 Tab | `idle`              | 尚未选择，不发请求                  |
| 单 Tab | `loading`           | 只在当前 Tab 内容区展示加载状态     |
| 单 Tab | `ready`             | 使用该 Anchor 的缓存详情            |
| 单 Tab | `error`             | 只影响该 Tab，并提供单树重试        |

推理树列表不存在合法空集合。空列表或不符合合同的响应按服务不可用处理，不能显示为正常空态。

## 错误映射

API Adapter 将 BFF 稳定错误转换为前端可判别错误，不向页面暴露内部请求信息：

| BFF code                             | 页面语义                       |
| ------------------------------------ | ------------------------------ |
| `INVALID_REQUEST`                    | 参数错误                       |
| `RESEARCH_THEME_NOT_FOUND`           | 该研究主题暂不可用             |
| `RESEARCH_REASONING_TREES_NOT_FOUND` | 影响路径暂未生成               |
| `RESEARCH_REASONING_TREE_NOT_FOUND`  | 当前 Tab 不可用，可单独重试    |
| `RESEARCH_DATA_UNAVAILABLE`          | 服务暂不可用，可按当前层级重试 |

网络失败、超时、非预期状态码和无效响应均映射为受控服务错误，不展示原始异常文本。

## 页面骨架边界

TW-06 页面只需具备可验证的结构：

- 返回操作和 Theme 基本标题区。
- Anchor Tab 列表与当前选中态。
- 当前 Tab 的 loading、ready、error 占位内容。
- 页面级参数错误、列表加载、业务不可用和重试状态。
- 首页按钮能够导航并携带正确 `theme_id`。

以下内容由 TW-07 实现，不作为 TW-06 验收条件：

- 原型 A 的 1:1 视觉样式。
- 事实、Anchor 结论、支持证据与反证的正式卡片排版。
- 传导节点、变化、传导路径、交易指向和下一检查点的最终视觉。
- 长文本、窄屏和多 Anchor 的视觉精修。

## 测试与验收

TW-06 至少覆盖：

1. 首页只有“查看影响路径”按钮导航，且携带正确 `theme_id`。
2. 缺失或非法 `theme_id` 不调用 Port。
3. 列表成功后自动选择并加载第一棵树。
4. 其他 Tab 首次选择才加载，成功后在当前会话复用缓存。
5. 多个在途请求不会互相覆盖，当前选中 Tab 保持正确。
6. 单 Tab 失败和重试不影响其他 Tab 或列表。
7. Theme 不存在、推理树未发布、列表服务失败分别进入正确页面状态。
8. Mock Adapter 能读取共享 fixture 并形成页面模型。
9. API Adapter 的 URL、请求方法、响应映射和稳定错误映射符合 BFF V1 合同。
10. `mock` 与 `api` 使用同一 Port；`api` 失败不回退到 mock。
11. 微信小程序和抖音小程序构建通过；不要求 H5 构建或浏览器验收。

## 范围外

- 修改 Data Service、Miniapp BFF、数据库或导入合同。
- 修改 AI 投研分析师项目。
- 真实后端联调和端到端链路验收。
- 页面运行时访问 Neo4j 或自行推理研究语义。
- H5 页面适配。
- 推理树最终视觉实现。
