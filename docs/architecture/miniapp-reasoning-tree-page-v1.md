# Miniapp 推理树页面 V1

## 状态

- 任务：TW-07
- 当前状态：Spec、Implement 与本地验证已完成，等待 PR Review
- 前置：TW-06 已验收；Anchor 支持/反证汇总合同已通过 PR #71 合并验收
- 正式原型：`prototype/reasoning-tree-prototype.html?variant=A`

## 用途

用户从首页 Theme 卡片点击“查看影响路径”，进入 Theme 的研究依据页。页面展示 Theme 顶部判断，并用 Anchor Tab 切换同一 Theme 下多棵、以不同 Chain Node 为中心的 Research Anchor 推理树。

页面只渲染 PostgreSQL 已发布快照经 Data 和 Miniapp BFF 返回的 DTO，不访问 Neo4j，不解析 Markdown，也不在前端生成研究语义。

## Taro 参考结论

- 继续使用项目现有 Taro 4、React 18 和 TypeScript。
- 使用 Taro `ScrollView` 的横向滚动能力承载 Anchor Tab 和产业链路径，兼容微信、抖音小程序。
- 页面复用 TW-06 的 typed port、API/Mock adapter、按 Tab 首次加载及会话缓存，不引入新状态框架。
- 本任务不实现或验收 H5。

## 页面结构

页面按原型 A 固定为：

1. Theme 顶部信息：影响等级、名称、发布时间、一句话结论和 Theme 整体传导路径。
2. Anchor Tab：使用中心 Chain Node 主数据名称；多 Anchor 横向滚动。
3. 原子事件汇总：Anchor `fact_summary`、Event 数量及全部 Event 清单。
4. Anchor 结论：`one_line_conclusion` 和 `net_direction_summary`。
5. 当前支持与当前反证：分别读取 `support_summary` 与 `counter_summary`。
6. 产业链节点传导：有序节点、变化、影响和相邻节点传导机制。
7. 交易指向：`trading_direction` 独立卡片。
8. 下一检查点：`next_checkpoint` 独立卡片。

## 原子事件

- 展示该 Anchor 关联的全部 Event，不折叠、不限制前三条。
- 保持 BFF 返回的稳定时间顺序，前端不得重排。
- 每条展示标题、摘要、可用的事件时间和证据角色标签。
- 角色固定映射为：`driver` 驱动、`supporting` 支持、`contradicting` 反证、`context` 背景。
- `event_time` 缺失时不显示时间，不显示伪造占位时间。
- 不展示 Raw Document 正文或内部证据长文。

## 支持与反证

- “当前支持”和“当前反证”是 Anchor 级推导结论，不是 Event 列表。
- 页面原样展示 `support_summary` 和 `counter_summary`，不按 Event 角色拼接或推断。
- `counter_summary = null` 时仍保留“当前反证”卡片，显示“当前暂无明确反证”。
- Event 角色只在原子事件清单中作为可追溯标签展示。

## 产业链传导

- 路径保持单条从左到右的有序链路。
- 路径区域单独横向滚动，页面主体不得出现横向溢出。
- 每个 Path Node 使用固定宽度卡片；中心 Chain Node 明确高亮。
- 相邻节点之间显示带方向箭头的连接段，并完整展示后一节点的 `incoming_transmission_mechanism`。
- 机制文字允许多行换行，不截断、不省略。
- 页面不展示“正式关系”或“推断关系”标签，也不验证正式产业链关系。

节点变化方向固定映射：

| 值 | 页面文案 |
| --- | --- |
| `increase` | `↑ 增强` |
| `decrease` | `↓ 减弱` |
| `mixed` | `↕ 分化` |
| `unchanged` | `→ 持平` |
| `uncertain` | `待验证`，不显示箭头或涨跌方向色 |

## Tab 与滚动

- Anchor Tab 栏吸顶；Theme 顶部信息正常随页面滚走。
- 页面首次进入自动选择并加载第一棵树；其余行为继承 TW-06。
- 切换 Tab 时滚动到新推理树内容顶部。
- 已成功加载的数据继续使用当前页面会话缓存，但不保存各 Tab 的历史阅读位置。
- 单 Tab loading、error 和 retry 只影响当前内容区。

## 文本与窄屏

- 结论、事实汇总、支持/反证、节点变化与影响、传导机制、交易指向和下一检查点全部完整展示。
- 所有文本允许自然换行，不使用省略号截断。
- 长文本只增加垂直高度；除 Anchor Tab 和路径 ScrollView 外，不产生横向滚动。
- 保留安全区间距，按钮和卡片不得与系统底部区域重叠。

## 状态

参数错误、页面 loading、Theme 不可用、推理树未发布、列表错误、单 Tab loading/error/retry 沿用 TW-06 已验收状态合同。TW-07 只完成正式视觉，不改变错误码映射或请求策略。

## 范围外

- Data、Miniapp BFF、数据库和 Import 合同；由 TW-07 前置修正完成。
- AI 分析师 Prompt、Publisher 或 Agent 项目。
- Theme 首页结构、用户跟踪、指数、Neo4j 和运行时推理。
- H5 适配、原型 B/C 方案切换器。

## 验收

1. 页面信息结构与原型 A 一致。
2. 多 Anchor Tab、首次按需加载、缓存、切换回顶和局部重试正确。
3. Event 全量、稳定顺序、角色标签和缺失时间行为正确。
4. 支持/反证使用 Anchor 汇总字段；无反证显示明确空态。
5. 路径只在自身区域横向滚动，完整机制文本可读，页面无横向溢出。
6. 五种 `change_direction` 映射正确，`uncertain` 不制造方向含义。
7. 长文本与窄屏不截断、不重叠。
8. 微信与抖音小程序构建和相关前端测试通过。
