# 一句话结论推理树 Tidewise 交付计划

## 状态

- 计划日期：2026-07-20
- 当前阶段：TW-01～TW-06 已验收
- 当前解锁任务：TW-07 Spec 确认
- 下一个门禁：TW-07 Spec 确认后进入 Implement
- 责任范围：Tidewise Data Service、Miniapp BFF 和小程序
- 外部依赖：AI 投研分析师侧负责生成与发布推理树结构化数据
- 合同优先级：若早期总体路线文档中的候选细节与本文件或 `reasoning-tree-contract-v1.md` 冲突，以用户逐项确认后的两份 Tidewise 文档为准

## 一、整体用途

本项目为首页 Theme（一句话结论）建设可展开的研究依据页。用户从首页点击“查看影响路径”后，可以按 Theme 涉及的产业链节点切换 Research Anchor，查看：

1. 一句话结论依赖的原子事件事实。
2. 以某个产业链节点为中心的 Anchor 结论。
3. 当前支持证据和当前反证。
4. 受影响的产业链节点、变化和传导路径。
5. 交易指向与下一检查点。

最终目标是让用户看清结论如何形成，并判断是否值得继续跟踪或采取投研行动。

## 二、核心边界

- Theme 是某个分析批次发布的一句话研究结论，是不可变批次快照。
- Chain Node 是长期维护的产业链主数据。
- Research Anchor 属于一个 Theme，并以一个 Chain Node 为中心，是不可变的推理树研究快照。
- Theme Import V1 保持不变，不建设 Theme Import V2。
- Research Anchor 通过独立 Anchor Import V1 发布。
- Theme 首页列表与推理树使用独立查询合同；Anchor 尚未发布或发布失败时，不影响已发布 Theme 按现有首页规则展示。
- PostgreSQL 保存已发布快照并作为页面查询事实源。
- Data Service 负责校验、幂等导入、事务存储和聚合查询。
- Miniapp BFF 只负责页面 DTO 映射，不进行因果推理。
- 小程序只负责页面状态和展示。
- 页面打开时不查询 Neo4j，不从 Markdown 反向解析，不在 BFF 或前端动态生成推理内容。
- 正式页面只实现推理树原型 A，不实现设计实验切换器。

## 三、执行门禁

整体用途先于任何单项 Spec 确认。整体用途未确认前，不开始 TW-01。

每个 TW 任务单独执行：

1. 阅读该任务相关代码、上下文和前置产物。
2. 使用 Grill with Docs 逐个确认高影响问题，一次只确认一个。
3. 形成并确认该任务的 Spec：用途、输入、输出、不变量、错误模式、范围外内容和验收样例。
4. 用户确认 Spec 后才允许 Implement。
5. 完成该任务的测试、Code Review 和独立提交。
6. 用户验收后才解锁下一任务。

本项目不使用 OpenSpec、to-spec 或 Superpowers 工作流。

## 四、任务清单

### TW-01 冻结推理树数据模型与接口合同

**用途**：为 AI 分析师、Data Service、Miniapp BFF 和小程序建立唯一、无歧义的推理树交换合同。

**主要产物**：

- Research Anchor 新语义和领域词汇。
- ER 图、字段字典、约束和旧字段处理方案。
- Research Anchor Import V1 请求、响应、严格校验、幂等及错误合同。
- Theme 推理树详情 Data API 与页面 DTO 合同。
- AI V3 到数据库再到页面 DTO 的字段血缘。
- 至少四组共享 fixture：多 Anchor、有反证、无反证且未量化、旧 Theme 无推理树。
- 旧独立 Anchor API 的废弃或兼容决定。

**范围外**：migration、Go 实现、前端实现、AI Prompt 与 Publisher 实现。

**验收**：各系统能够只依赖合同和 fixture 独立开发，不存在未定义字段或生命周期冲突。

### TW-02 演进 Research Anchor 数据库结构

**用途**：将现有 Anchor 表体系改造成 Theme 下、以 Chain Node 为中心的不可变推理树快照。

**主要产物**：

- `research_anchors` 结构演进。
- `research_anchor_events` 证据语义扩展。
- `research_anchor_chain_nodes` 改为唯一有序传导路径节点。
- 新增不可变 `research_anchor_import_receipts`。
- Theme、中心 Chain Node、唯一性、证据角色和路径不变量约束。
- 空表断言后原名重建旧 Anchor 表，删除且不重建 `research_anchor_indices`。
- 旧 Anchor 结构不做兼容、转换或恢复，migration 采用 forward-only。
- Anchor 发布回执限制删除所属 Theme；普通删除失败，仅本地 reset 可受控整体清理。
- migration 及数据库合同测试。
- 升级本地 `research-theme-dev-reset`：重置 Theme 发布快照时，同一事务内一并清空 Anchor、路径节点、Anchor Event 关联和 Anchor 导入回执；安全处理并恢复 Theme/Anchor 两类不可变回执触发器，继续保护 Event、Chain Node、Index、Tag、Raw Document 等主数据。

**依赖**：TW-01 已验收。

**验收**：migration 可在空库和当前本地库执行；Theme Import V1 不受影响；无效关联无法写入；本地 reset 的 dry-run 和 execute 能准确覆盖 Theme 与推理树发布数据，并证明受保护主数据计数不变。

### TW-03 实现 Research Anchor Import V1

**用途**：提供按单个 Theme 原子导入全部 Anchor、Event 证据和传导节点的内部接口。

**主要产物**：

- `POST /internal/data/v1/research-anchor-imports`。
- Theme、中心节点、Event、Path Node 和关系引用校验。
- Theme 范围单事务原子写入。
- 独立 receipt、canonical payload hash、幂等重放、冲突和并发控制。
- 权限和发布服务主体隔离。

**依赖**：TW-02 已验收。

**验收**：相同 Theme 和 payload 可重放；不同 payload 冲突；任一 Anchor 无效时该 Theme 的 Anchor 整体回滚；不修改 Theme 或主数据。

### TW-04 实现 Theme 推理树详情查询

**用途**：使 Data Service 提供 Theme 下稳定 Tab 摘要和按需加载的单树完整快照。

**主要产物**：

- 推理树列表端点返回 Theme 基础信息和全部 Anchor Tab 摘要。
- 单树详情端点返回中心节点、正反证据和有序传导节点。
- 两个端点均稳定排序且单次请求不产生 Anchor N+1。
- 首页 Theme 查询保持不变，不依赖 Anchor 回执，也不增加 `has_reasoning_tree`。
- 未发布推理树的 Theme 返回明确 `404`，不返回合法空集合。
- 删除未投入使用的旧独立 Anchor Data API。
- 同步删除只代理上述失效 Data 合同的旧独立 Anchor Miniapp API，避免主分支保留已知不可用路径。

**依赖**：TW-02 已验收。

**验收**：列表与单树详情可以分别驱动 Tab 和当前树内容，排序稳定；现有 Theme Detail 合同保持不变。

### TW-05 扩展 Miniapp BFF

**用途**：将 Data Service 的研究快照映射成小程序可直接渲染的页面 DTO。

**主要产物**：

- BFF Typed Client、Tab 列表 DTO、单树详情 DTO 和两个 HTTP 接口。
- 无反证、无量化值、主题不可用、未知枚举和上游错误映射。
- 每个 BFF 请求只调用一次对应 Data API，不扇出 N+1。
- 不再承担旧 Anchor API 清理；该清理已在 TW-04 与 Data 失效合同一起完成。

**依赖**：TW-04 已验收。

**验收**：共享 fixture 和真实 Data 响应均能稳定映射，小程序不需要理解数据库表。

### TW-06 建设小程序数据层与页面骨架

**用途**：建立可独立验证的 Theme 推理树页路由、数据访问和页面状态。

**主要产物**：

- 非 tabBar 详情页路由。
- 首页携带 `theme_id` 跳转。
- TypeScript contract、Port、API Adapter 和 Mock Adapter。
- 参数错误、loading、ready、主题不可用、列表错误、单 Tab 错误和 retry 状态。
- 每个 Tab 首次选中时加载详情并在当前页面会话缓存。

**依赖**：TW-01 已验收，可基于共享 fixture 实现。

**验收**：可从首页进入 Mock 详情页，路由和所有页面状态可独立测试。

### TW-07 实现小程序推理树页面

**用途**：按正式原型 A 把研究依据变成可读、可切换的 Theme 展开页。

**主要产物**：

- Theme 顶部信息。
- Anchor 横向 Tabs、首次按需加载和页面内缓存状态。
- 原子事件、Anchor 结论、支持证据和反证。
- 传导节点、变化、传导路径、交易指向和下一检查点。
- 长文本、窄屏、多 Anchor、无反证和未量化状态。
- 微信与抖音小程序构建兼容。

**依赖**：TW-06 已验收。

**验收**：页面与原型 A 信息结构一致，多 Anchor 正常切换，无横向溢出，微信和抖音构建通过。

### TW-08 Tidewise 独立链路验收

**用途**：证明同一共享 fixture 在导入、存储、查询、BFF 和页面中保持一致。

**主要产物**：

- Research Anchor Import V1 导入验收。
- PostgreSQL 表、关联和约束验收。
- Data API 和 Miniapp BFF 响应验收。
- 小程序页面内容、排序和状态验收。
- 多 Anchor、有反证、无反证、未发布树 404、无效引用、幂等重放和事务回滚覆盖。

**依赖**：TW-03、TW-05 和 TW-07 已验收。

**验收**：共享 fixture 中的 Theme、Anchor、Event、节点顺序和页面展示一一对应，全部失败与兼容场景通过。

## 五、依赖与顺序

```text
整体用途确认
  → TW-01 Spec 确认 → Implement → 验收
  → TW-02 Spec 确认 → Implement → 验收
  → TW-03 Spec 确认 → Implement → 验收
  → TW-04 Spec 确认 → Implement → 验收
  → TW-05 Spec 确认 → Implement → 验收
  → TW-06 Spec 确认 → Implement → 验收
  → TW-07 Spec 确认 → Implement → 验收
  → TW-08 Spec 确认 → 链路验收
```

TW-04 与 TW-03 在 TW-02 完成后具备技术并行条件，TW-06 在 TW-01 共享 fixture 冻结后也具备技术并行条件。但本计划默认依然按任务编号逐个确认和验收，除非用户明确批准并行。

## 六、整体范围外

- AI 分析师 Prompt、推理方法、Adapter、Outbox 和 Publisher 实现。
- 修改 `agent-raw-ingestion-mvp`。
- Theme Import V2 或对 Theme Import V1 的破坏性变更。
- 页面运行时查询 Neo4j 或其他图计算服务。
- 从 Markdown 反向解析推理树。
- 跨批次的长期 Research Thesis 或瓶颈跟踪对象。
- Anchor 管理后台和人工编辑写入能力。
- 推理树原型 A/B/C 设计实验切换器。

## 七、任务状态记录

| 任务  | Spec   | Implement | 验证   | 状态        |
| ----- | ------ | --------- | ------ | ----------- |
| TW-01 | 已确认 | 已完成    | 已通过 | 已验收      |
| TW-02 | 已确认 | 已完成    | 已通过 | 已验收      |
| TW-03 | 已确认 | 已完成    | 已通过 | 已验收      |
| TW-04 | 已确认 | 已完成    | 已通过 | 已验收      |
| TW-05 | 已确认 | 已完成    | 已通过 | 已验收      |
| TW-06 | 已确认 | 已完成    | 已通过 | 已验收      |
| TW-07 | 进行中 | 未开始    | 未开始 | Spec 确认   |
| TW-08 | 未开始 | 未开始    | 未开始 | 未解锁      |

## 八、参考资料

- 总体任务记录：`/Users/meierlink/Documents/david/创业项目/观潮家/research/methodology/2026-07-20-reasoning-tree-delivery-roadmap.md`
- 推理树原型：`prototype/reasoning-tree-prototype.html`
- 推理树截图：`/Users/meierlink/Documents/david/创业项目/观潮家/research/methodology/2026-07-20-reasoning-tree-prototype.png`
- 原首页原型：`prototype/miniprogram.html`
- Theme Import V1：`docs/architecture/research-theme-import-v1.md`
- 领域上下文导航：`CONTEXT-MAP.md`
- Data Context：`docs/contexts/data/CONTEXT.md`
- Miniapp Context：`docs/contexts/miniapp/CONTEXT.md`
