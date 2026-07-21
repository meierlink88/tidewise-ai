# 一句话结论推理树合同 V1

## 状态

- 所属任务：TW-01
- 当前状态：已由用户验收并冻结，作为 TW-02～TW-08 的唯一 V1 合同
- 实现状态：TW-01 文档与共享 fixture 已完成；TW-02 Anchor schema migration 与本地 reset 已实施并通过本地验证；TW-03～TW-08 未开始
- 规则：本文档中只有标记为“已确认”的内容属于冻结合同

## 用途

建立 AI 分析师、Data Service、Miniapp BFF 和小程序共用的 Research Anchor 发布与读取合同，使首页 Theme 能够展开为可追溯的产业链推理树页面。

## 已确认领域不变量

### 1. Theme 与 Anchor 中心节点完整覆盖

状态：已确认。

- 一个 Research Anchor 只属于一个 Theme，并且只对应一个中心 Chain Node。
- Anchor Import V1 中的中心 Chain Node 集合必须与该 Theme 已发布的 Theme Chain Node Association 集合完全一致。
- 每个 Theme Chain Node Association 恰好对应一个 Research Anchor。
- 中心节点缺失、多出或重复时，整个 Theme 的 Anchor Import 拒绝并回滚。
- Theme 与推理树使用独立读取合同；Theme 成功发布后按现有首页规则可见，不等待 Anchor 集合发布。
- 不存在部分 Anchor 发布或部分 Tab 可见状态。

### 2. 中心节点属于传导节点集合

状态：已确认。

- 每棵 Research Anchor 的传导节点集合必须包含其中心 Chain Node。
- 中心 Chain Node 在该 Anchor 中恰好出现一次。
- 导入合同必须能明确区分“当前锚点”与其他传导节点。
- 中心节点自身的变化、方向和说明由 AI 分析师作为快照内容发布。
- Data Service 校验并保存该事实；Miniapp BFF 和小程序不得临时补写中心节点。

### 3. V1 产业链传导部分是单条有序路径

状态：已确认。

- 每棵 Research Anchor 只包含一条有序的产业链传导路径。
- 传导路径由按明确位置排列的 `path_nodes` 组成。
- V1 不表达多条并行路径、节点分叉或通用图。
- “推理树”的分支来自当前支持证据与当前反证；产业链影响部分保持线性。
- Data Service 保存并按发布顺序返回路径，BFF 和小程序不得自行重排。

### 4. 路径可以包含非 Anchor 的辅助 Chain Node

状态：已确认。

- Research Anchor 的中心 Chain Node 必须来自该 Theme 已发布的 Theme Chain Node Association。
- 其他 `path_nodes` 可以引用任意已存在的 Chain Node，用于说明传导的上下文或影响去向。
- 辅助路径节点不会因出现在路径中而自动成为 Theme Chain Node Association 或获得 Anchor Tab。
- 每个辅助路径节点仍必须由 AI 分析师提交其当次变化和说明；Data Service 不根据主数据或图关系补写。

### 5. Anchor 路径不受正式产业链关系完整性限制

状态：已确认。

- Anchor 中相邻路径节点的连线是本批次的研究传导判断。
- Anchor Import V1 不强制提交 `chain_node_relations.id`。
- Data Service 不查询或校验相邻节点是否已存在正式产业链关系。
- 正式关系图谱缺失或不完整，不得单独成为 Anchor 导入失败的原因。
- Anchor Import 不会新增、修改或升格任何 `chain_node_relations` 正式关系。
- 连线的精确文本合同仍待后续确认。

### 6. V1 不区分“正式关系”和“推断关系”

状态：已确认。

- Anchor Import V1 不接收用于声称连线是“正式”或“推断”的关系来源枚举。
- Theme 推理树 Data API 和 Miniapp 页面 DTO 不返回这组来源标签。
- 小程序不展示“正式关系”或“推断关系”标签。
- 所有连线统一表达为本批次 Research Anchor 的研究传导判断。

### 7. 每条传导连线必须包含机制说明

状态：已确认。

- 每两个相邻 `path_nodes` 之间必须存在一条连线。
- 每条连线必须提交非空的 `transmission_mechanism`。
- `transmission_mechanism` 由 AI 分析师生成，用于说明影响为何从前一节点传到后一节点。
- Data Service 只校验字段存在且去除空白后非空，不生成、改写或判断机制内容的研究正确性。
- 任一连线缺少有效机制说明时，整个 Theme 的 Anchor Import 拒绝并回滚。

### 8. 路径节点同时保存受控变化方向和自然语言说明

状态：已确认。

每个 `path_nodes` 项必须由 AI 分析师提交：

- `change_direction`：必填，只允许 `increase`、`decrease`、`mixed`、`unchanged`、`uncertain`。
- `change_summary`：必填且非空，用于页面展示简短变化，例如“断供风险上升”或“价格尚未确认”。
- `impact_summary`：必填且非空，说明该变化在当前 Anchor 推理路径中的研究含义。

Data Service 只校验枚举值和非空字段，不从文本推断方向。`change_direction` 不表达投资利好或利空。

### 9. Anchor 净方向使用自然语言摘要

状态：已确认。

- 每个 Research Anchor 必须提交非空的 `net_direction_summary`。
- 它概括当前事实、支持证据和反证合并后的总体指向。
- 该文本由 AI 分析师生成，Data Service 只校验非空，不从路径节点或证据自动合成。
- V1 不将净方向收窄为 `bullish`、`bearish` 或 `neutral` 等枚举。
- `net_direction_summary` 表达研究事实的总体指向；`trading_direction` 表达用户应重点研究或关注的交易映射，二者不可互相替代。

### 10. Anchor 不保存重复的自然语言传导路径

状态：已确认。

- Research Theme 继续保留 Theme Import V1 中的 `transmission_path`，用于页面顶部的整体传导摘要。
- Research Anchor 不接收或保存 Anchor 级自然语言 `transmission_path`。
- Anchor 的传导语义唯一来自有序 `path_nodes` 及相邻节点之间的 `transmission_mechanism`。
- Data Service、Miniapp BFF 和小程序不得再生成、拼接或维护第二份 Anchor 路径文本。
- 现有 `research_anchors.transmission_path` 属于旧字段，具体迁移处置在 TW-02 按已确认旧数据策略执行。

### 11. Anchor Event 必须是所属 Theme Event 的子集

状态：已确认。

- Anchor Import V1 中的每个 `event_id` 必须已经存在于该 Anchor 所属 Theme 的 `research_theme_events` 关联中。
- Event 即使在 Data 中存在，但不属于当前 Theme，也不得被 Anchor 引用。
- 任一 Anchor Event 超出 Theme Event 边界时，整个 Theme 的 Anchor Import 拒绝并回滚。
- Anchor Import 不会新增或修改 Theme Event Association。
- 若新 Event 确实改变了 Theme 的证据边界，分析侧必须使用新分析批次重新发布 Theme。

### 12. 每棵 Anchor 至少包含一个 driver Event

状态：已确认。

- Anchor Event Evidence 角色仅允许 `driver`、`supporting`、`contradicting`、`context`。
- 每棵 Research Anchor 必须至少关联一个 `driver` Event。
- `supporting`、`contradicting` 和 `context` 可以为空。
- 没有 `contradicting` Event 是合法的已发布研究状态，不影响 Anchor 集合完整性。
- Theme 推理树响应与 Miniapp 页面 DTO 必须保留空反证状态，小程序显式展示“当前未观察到反向证据”，不隐藏反证区域。

### 13. 同一 Event 在同一 Anchor 中唯一

状态：已确认。

- 同一 `event_id` 在同一 Research Anchor 中最多出现一次。
- 该 Event 在该 Anchor 中只能承担一个 `evidence_role`。
- 数据库和导入合同必须防止同一 Anchor 内重复 Event 关联。
- 如果一个上游来源同时包含支持与反向事实，应在 Event 提取阶段形成不同的原子 Event；确实无法归类时，可在 Anchor 中将它标记为 `context`。

### 14. Anchor Event 关联使用中性的 evidence_summary

状态：已确认。

Anchor Import V1 的每个 Event 关联仅提交：

- `event_id`：已属于当前 Theme 的 Event UUID。
- `evidence_role`：`driver`、`supporting`、`contradicting` 或 `context`。
- `evidence_summary`：必填且非空，说明该 Event 在当前 Anchor 中具体支持、反驳或补充什么。

V1 不继续使用对反证语义不对称的 `supported_claim` 名称。Event 的 `title`、`summary` 和 `event_time` 从现有 Event 数据读取，导入请求不重复提交这些字段。Data Service 不生成或改写 `evidence_summary`。

### 15. 原子事件区使用 Anchor 汇总标题和可追溯 Event 明细

状态：已确认。

- 每个 Research Anchor 必须提交非空的 `fact_summary`，由 AI 分析师概括该 Anchor 所使用的原子 Event 事实组合。
- Data Service 只校验 `fact_summary` 非空，不生成或改写。
- 原子事件明细直接返回 Anchor Event 关联的 `event_id`、`title`、`summary` 和 `event_time`。
- Event 数量按当前 Anchor 的唯一 `event_id` 计数。
- Anchor Import V1 不接收重复 Event 事实的 `fact_statements` 或类似文本数组。

### 16. Anchor 名称来自中心 Chain Node 主数据

状态：已确认。

- Anchor Import V1 仅提交 `center_chain_node_id`，不提交独立 Anchor `name`。
- Research Anchor 不复制 Chain Node 名称。
- Data Service 在查询时通过中心 Chain Node 返回当前规范名称，Miniapp BFF 将它用作 Anchor Tab 和页面名称。
- Chain Node 后续规范更名时，Anchor 页面名称随主数据更新；Anchor 的结论、证据和路径快照仍保持不变。

### 17. 旧 anchor_type 和 importance 不进入新模型

状态：已确认。

- Research Anchor 的身份由所属 `theme_id` 和 `center_chain_node_id` 共同决定。
- Anchor Import V1 不接收 `anchor_type` 或 `importance`。
- 新 Theme 推理树 Data API 和 Miniapp 页面 DTO 不返回这两个旧字段。
- TW-02 删除现有 `research_anchors.anchor_type` 和 `research_anchors.importance` 及其枚举约束，不将其映射为新语义。

### 18. Anchor 批次归属来自 Theme，发布状态由 Theme 级回执承载

状态：已确认。

- `research_anchors` 通过 `theme_id` 归属 Research Theme。
- Anchor 行不重复保存 `analysis_batch_id` 或逐 Anchor `published_at`。
- 一个 Theme 最多有一条成功的 Research Anchor Publication Receipt。
- Receipt 记录 `theme_id`、发布服务主体、payload hash、Anchor ID 映射和数量、Anchor 集合正式可见时间及导入时间。
- Receipt、全部 Anchor、Anchor Event Evidence、Anchor Path Node 和 Anchor Transmission Edge 必须在同一数据库事务中提交或回滚。
- 独立 Theme 推理树读取只能在该 Theme 存在成功 Anchor Receipt 时返回完整树；现有 Theme Detail 合同保持不变。
- 页面的分析发布时间继续使用 Theme `published_at`；Anchor Receipt 时间用于技术审计与可见性判定。

### 19. theme_id 是 Anchor Import 唯一幂等身份

状态：已确认。

- Anchor Import V1 不增加 `idempotency_key` 或 `Idempotency-Key` 请求头。
- Data Service 对完整 Anchor Import 请求按冻结的 canonical JSON 规则计算 SHA-256。
- 同一 `theme_id` 和相同 payload hash 返回首次成功结果，并令当次响应 `replayed=true`。
- 同一 `theme_id` 已成功发布后提交不同 payload hash，返回冲突并禁止覆盖。
- 校验失败或事务回滚不生成 Receipt，也不占用 `theme_id`；依赖修复后可以重试。
- 已成功 Anchor 内容需要修订时，分析侧必须生成新分析批次并发布新 Theme。

### 20. Anchor 发布主体必须与 Theme 发布主体一致

状态：已确认。

- Anchor Import V1 复用现有 `data.research.import` scope。
- Data Service 从 Bearer service token 的认证上下文解析稳定 `publisher_subject`，请求体不得声明或覆盖该主体。
- Anchor Import 的 `publisher_subject` 必须与当前 Theme 所属 Theme Publication Receipt 中的发布主体一致。
- Anchor 的首次发布和幂等重放都必须来自该稳定主体。
- token 正常轮换后，只要解析出的稳定主体 ID 不变，仍允许发布或重放。
- 其他服务即使具备 `data.research.import` scope，也不能为该 Theme 发布或重放 Anchor。

### 21. Anchor 由 Theme 和中心 Chain Node 确定性标识

状态：已确认。

- Anchor Import V1 不接收 `anchor_id` 或 `anchor_key`。
- 单个 Theme 内的 Anchor 业务唯一约束为 `UNIQUE (theme_id, center_chain_node_id)`。
- Data Service 使用 `UUIDv5(DNS, "tidewise.ai/research-anchor-publication/v1")` 得到并冻结命名空间 `f219ded4-fc65-5948-9e28-c1cdb6a8288e`。
- `anchor_id` 使用该命名空间，对标准小写 `theme_id + NUL + center_chain_node_id` 计算 UUIDv5；NUL 分隔符用于避免字符串拼接歧义。
- 同一 Theme 幂等重放产生相同 Anchor UUID。
- 不同 Theme 即使具有相同中心 Chain Node，也产生不同 Anchor UUID。
- Receipt 使用 `anchor_ids_by_center_chain_node_id` 返回完整的中心 Chain Node UUID 到 Anchor UUID 映射。

### 22. Anchor Tabs 按中心 Chain Node 名称稳定排序

状态：已确认。

- V1 不增加 `display_order`、`anchor_order`、旧 `importance` 或 AI 排名字段。
- Data Theme 推理树读取先按中心 Chain Node 的当前规范名称以 PostgreSQL `COLLATE "C"` 升序返回 Anchor，名称相同时再按 `center_chain_node_id` 升序，避免不同环境 locale 产生不同顺序。
- Miniapp BFF 和小程序必须保持 Data 返回顺序，不得二次重排。
- Chain Node 规范名称后续变更时，Anchor Tab 名称和名称排序同步变更。

### 23. path_nodes 数组同时承载节点顺序和入边机制

状态：已确认。

- `path_nodes` 数组顺序就是产业链传导顺序，请求不另外提交 `position`。
- 每个 Path Node 包含 `chain_node_id`、`change_direction`、`change_summary`、`impact_summary` 和 `incoming_transmission_mechanism`。
- 第一个 Path Node 的 `incoming_transmission_mechanism` 必须显式为 `null`。
- 第二个及之后 Path Node 的 `incoming_transmission_mechanism` 必须是非空文本，表达影响如何从前一节点传入当前节点。
- V1 不接收独立 `path_edges` 数组，防止节点顺序与 Edge 列表不一致。
- Data Service 在持久化时根据数组下标保存路径位置，读取时按该位置恢复原顺序。

### 24. 同一 Anchor 路径禁止重复节点和循环

状态：已确认。

- 同一 Anchor 的 `path_nodes` 中，每个 `chain_node_id` 只能出现一次。
- Data Service 必须校验整条路径的节点唯一性，不能只校验中心节点。
- `A → B → A` 等任何形式的循环路径都不是 V1 的合法表示。
- 任一重复节点使该 Theme 的整批 Anchor Import 拒绝并回滚，错误必须定位到 Anchor 中心节点、`path_nodes` 字段路径和重复的 `chain_node_id`。
- 若真实研究中存在反馈回路，V1 只能在快照的说明文本中表达，不将其编码为循环路径。

### 25. 每条 Anchor 路径至少包含两个节点

状态：已确认。

- 每棵 Research Anchor 的 `path_nodes` 至少包含两个不同的 Chain Node。
- 因此每棵 Anchor 至少存在一条相邻节点间的传导边，并由后一节点的 `incoming_transmission_mechanism` 说明传导机制。
- 只包含中心节点的单节点结果不足以构成 V1 影响路径，Data Service 必须拒绝。
- 任一 Anchor 不满足最小路径长度时，该 Theme 的整批 Anchor Import 拒绝并回滚。

### 26. Anchor 必须拥有独立的一句话结论

状态：已确认。

- 每棵 Research Anchor 必须提交非空的 `one_line_conclusion`。
- 该字段表达围绕当前中心 Chain Node 得出的 Anchor 结论，不是 Theme 总体 `one_line_conclusion` 的别名。
- 内容由 AI 分析师生成并发布；Data Service 只校验必填和非空，不复制 Theme 结论，不从 Event、路径或中心节点推导。
- Miniapp BFF 和前端必须原样展示已发布的 Anchor 结论，不得用 Theme 结论补缺。

### 27. Anchor 必须拥有独立的交易指向

状态：已确认。

- 每棵 Research Anchor 必须提交非空的 `trading_direction`。
- 该字段表达围绕当前中心 Chain Node 的研究优先级、受益或承压方向以及交易映射，属于 Anchor 快照。
- Anchor `trading_direction` 不能由 Theme 级 `trading_direction` 代替，也不能与表达客观变化净指向的 `net_direction_summary` 互换。
- 内容由 AI 分析师生成；Data Service 只校验必填和非空并原样保存，BFF 和前端只负责展示。

### 28. Anchor 必须拥有独立的下一检查点

状态：已确认。

- 每棵 Research Anchor 必须提交非空的 `next_checkpoint`。
- 该字段表达围绕当前中心 Chain Node 下一步需要验证的指标、事实或条件，属于 Anchor 快照。
- Anchor `next_checkpoint` 不能由 Theme 级 `next_checkpoint` 代替。
- 内容由 AI 分析师生成；Data Service 只校验必填和非空并原样保存，不拼接、推断或设置默认值。

### 29. V1 不考虑 Anchor 指数关联

状态：已确认。

- Research Anchor Import V1 请求不接收 `indices` 或其他指数关联字段。
- V1 发布流程不写入 `research_anchor_indices`。
- Theme 推理树详情合同和 Miniapp DTO 不返回 Anchor 指数数据。
- 现有空的 `research_anchor_indices` 属于未投入使用的历史结构，由 TW-02 在空表断言通过后删除且不重建，不得因兼容它而扩大 V1 业务合同。

### 30. 当前支持与当前反证是 Anchor 级推导汇总

状态：已确认。

- `support_summary` 是 AI 分析师对该 Anchor 当前支持证据形成的整体结论，必填且非空白。
- `counter_summary` 是 AI 分析师对该 Anchor 当前反证形成的整体结论；存在 `contradicting` Event 时必填且非空白，没有 `contradicting` Event 时必须为 `null`。
- `events[]` 继续保存具体 Event、`evidence_role` 和 `evidence_summary`，用于事实追溯；它不是 Anchor 汇总结论的替代品。
- Data Service、Miniapp BFF 和小程序不得按 Event 角色拼接、推断或改写两个汇总字段。
- 没有反证时页面仍保留“当前反证”区域，并显示确定性空态文案，不伪造研究事实。

### 31. 原子事件汇总覆盖 Anchor 的全部 Event

状态：已确认。

- “原子事件汇总”包含该 Anchor 关联的全部去重 Event，不论其 `evidence_role` 是 `driver`、`supporting`、`contradicting` 还是 `context`。
- 事实清单回答“本次研究使用了哪些原子事件”；Event 角色回答单个 Event 在论证中的作用；Anchor 汇总回答证据整体目前支持或反驳什么。
- 页面在事实清单中展示每个 Event 的角色标签，但当前支持和当前反证只读取 Anchor 汇总字段。
- 原子事件数量按 `event_id` 去重计算；空数组不合法，因为每棵 Anchor 已被要求至少具有一个 `driver` Event。

### 32. Anchor Event 按事件时间正序展示

状态：已确认。

- Data Theme 推理树响应中，同一 Anchor 的 Event 默认按 `event_time` 从早到晚排列，便于用户按时间理解事实和因果推进。
- `event_time` 相同时按 `event_id` 升序排列；`event_time` 缺失的 Event 放在有时间的 Event 之后，并按 `event_id` 升序排列。
- 原子事件清单必须保持该稳定顺序；Anchor 支持与反证汇总不参与 Event 排序。
- Miniapp BFF 和前端不得按证据角色、标题或接收顺序重新排序。

### 33. Anchor Import 使用与 Theme Import 一致的确定性请求表示

状态：已确认。

- `anchors` 必须按规范化小写 `center_chain_node_id` 升序提交。
- 每个 Anchor 的 `events` 必须按规范化小写 `event_id` 升序提交。
- `path_nodes` 顺序是业务语义本身，必须保持 AI 分析师发布的真实传导顺序，不按 UUID 或名称排序。
- 请求中所有 UUID 必须使用标准小写字符串格式；数组内重复身份仍按已确认规则拒绝。
- Data Service 校验格式、唯一性和规定顺序，但不静默重排；只在结构和顺序全部通过后，才对原始合法表示计算 canonical SHA-256。
- 顺序不合法时，错误必须定位到相应的 Anchor 中心节点和字段路径，整批拒绝。

### 34. Anchor Import V1 使用独立主题级发布端点

状态：已确认。

- HTTP 端点固定为 `POST /internal/data/v1/research-anchor-imports`。
- 请求体顶层严格只包含 `theme_id` 和 `anchors`。
- `theme_id` 同时确定父 Research Theme、本次要原子发布的完整 Anchor 集合和唯一幂等身份。
- V1 不在 URL 中重复 `theme_id`，不接收 `analysis_batch_id`、额外幂等键、请求时间窗口或调用方指定的发布时间。
- 请求出现任何未声明顶层字段时，按严格 V1 合同拒绝整个请求。

### 35. Anchor Import V1 使用严格的 Anchor 字段合同

状态：已确认。

每个 `anchors` 项严格只包含：

```json
{
  "center_chain_node_id": "uuid",
  "one_line_conclusion": "中心节点结论",
  "fact_summary": "原子事实汇总",
  "net_direction_summary": "当前净方向",
  "trading_direction": "交易研究指向",
  "next_checkpoint": "下一验证项",
  "events": [
    {
      "event_id": "uuid",
      "evidence_role": "driver",
      "evidence_summary": "该事件承担的证据作用"
    }
  ],
  "path_nodes": [
    {
      "chain_node_id": "uuid",
      "change_direction": "increase",
      "change_summary": "当前变化",
      "impact_summary": "对 Theme 的影响",
      "incoming_transmission_mechanism": null
    }
  ]
}
```

- `one_line_conclusion`、`fact_summary`、`net_direction_summary`、`trading_direction` 和 `next_checkpoint` 都必填且非空。
- `events` 和 `path_nodes` 必须满足前述角色、子集、完整性、顺序、长度、中心节点和无循环约束。
- V1 不接收 `anchor_id`、`anchor_key`、`name`、`anchor_type`、`importance`、`transmission_path`、`indices`、`published_at` 或 `imported_at`。
- Anchor 项内出现任何其他字段时，按严格 V1 合同拒绝整个 Theme 的 Anchor Import。

### 36. Anchor Import V1 返回不可变发布回执

状态：已确认。

成功响应固定为：

```json
{
  "request_id": "request-uuid",
  "result": {
    "receipt_id": "uuid",
    "theme_id": "uuid",
    "payload_hash": "64位小写sha256",
    "anchor_ids_by_center_chain_node_id": {
      "chain-node-uuid": "anchor-uuid"
    },
    "counts": {
      "anchors": 3,
      "event_associations": 8,
      "path_nodes": 10,
      "receipts": 1
    },
    "published_at": "2026-07-20T10:00:00Z",
    "imported_at": "2026-07-20T10:00:00Z",
    "replayed": false
  }
}
```

- `anchor_ids_by_center_chain_node_id` 必须完整覆盖该 Theme 的中心 Chain Node 集合，key 和 value 都是标准小写 UUID。
- `counts` 字段名固定为 `anchors`、`event_associations`、`path_nodes` 和 `receipts`；首次成功的 `receipts` 恒为 `1`。
- `published_at` 是完整 Anchor 集合正式可见的服务端时间，不替代父 Theme 的分析发布时间；`imported_at` 是技术导入时间。
- 首次成功返回 HTTP `201`；同发布主体、同 Theme、同 payload 的幂等重放返回 HTTP `200`。
- 重放必须返回首次成功时的原 `receipt_id`、映射、计数和时间，仅将本次响应的 `replayed` 置为 `true`；`replayed` 不作为首次回执事实固化。

### 37. Anchor Import V1 复用 Theme Import V1 的错误分层

状态：已确认。

- HTTP `400` + `INVALID_REQUEST`：JSON 无法解析、未知字段、字段类型或基础格式不合法。
- HTTP `400` + `RESEARCH_ANCHOR_IMPORT_REJECTED`：枚举、必填文本、数组顺序、完整 Anchor 覆盖、最少路径长度、重复节点或事件等 V1 合同校验失败。
- HTTP `422` + `RESEARCH_ANCHOR_REFERENCE_NOT_FOUND`：被引用的 Theme、Event 或 Chain Node 不存在。
- HTTP `422` + `RESEARCH_ANCHOR_REFERENCE_INVALID`：引用对象存在，但 Event 不属于父 Theme 的已发布证据集，或 Anchor 中心节点不属于父 Theme 的已发布节点集。
- HTTP `409` + `RESEARCH_ANCHOR_PAYLOAD_CONFLICT`：同一 `theme_id` 已使用不同 payload 成功发布。
- HTTP `409` + `RESEARCH_ANCHOR_PUBLISHER_CONFLICT`：调用主体与父 Theme 的发布主体不一致，或与已有 Anchor Receipt 所有者不一致。
- 认证失败、缺少 `data.research.import` scope、请求体过大和意外服务端错误分别使用项目标准 HTTP `401`、`403`、`413` 和 `500`。
- 所有可定位错误的 `details` 统一包含 `center_chain_node_id`、`path` 和 `reference`；错误发生在顶层 `theme_id` 时，`center_chain_node_id` 为空字符串。
- 任一错误都在事务提交前使整个 Theme Anchor 集合失败，不返回部分成功结果。

### 38. Anchor Import V1 保持同步并通过幂等 POST 恢复未知结果

状态：已确认。

- V1 不增加 Research Anchor Import 状态查询端点。
- V1 不将发布改为异步任务，不引入 job、progress、polling 或 `unknown` 状态记录。
- 客户端在 POST 可能已成功但响应丢失时，必须使用相同 `theme_id` 和完全相同的 payload 重试原 POST。
- 首次事务已成功时返回原回执且 `replayed=true`；未成功时正常执行；payload 不同时返回冲突。
- 未成功的校验或事务不留下成功 Receipt 或占位状态。

### 39. 多条推理树使用列表与单树详情两个 Theme 子资源端点

状态：已确认。

- Data 推理树列表端点固定为 `GET /internal/data/v1/research/themes/{theme_id}/reasoning-trees`。
- Data 单树详情端点固定为 `GET /internal/data/v1/research/themes/{theme_id}/reasoning-trees/{anchor_id}`。
- 两个端点均不接收 `window_hours` 或其他时间窗口参数；已发布推理树完全按路径中的 `theme_id` 读取，即使 Theme 已离开首页时间窗口仍可通过明确 ID 访问。
- 现有 `GET /internal/data/v1/research/themes/{theme_id}` 合同保持不变，不在本期嵌入 Anchor 推理树。
- Miniapp 首页的“查看影响路径”进入推理树页面时，由 Miniapp BFF 先调用列表 API；选中某个 Anchor Tab 后，再调用对应单树详情 API。
- 一个 Theme 可以拥有多个 Research Anchor，每个 Anchor 就是一条独立推理树；同一页面通过 Tab 展示全部树。
- 列表 API 只返回构建稳定 Tab 清单所需的推理树摘要；单树详情 API 返回一棵完整推理树，禁止 BFF 为一次详情请求扇出查询多个 Anchor。
- 首页 Theme 列表不增加 `has_reasoning_tree` 或等价布尔字段。

### 40. 首页 Theme 列表不依赖推理树发布状态

状态：已确认。

- `GET /internal/data/v1/research/themes` 只负责按现有 Theme 批次、时间窗口和分页合同返回 Theme，不查询或依赖 Research Anchor Publication Receipt。
- Theme Import V1 成功后，即使 Anchor Import 尚未成功或失败，该 Theme 仍按现有首页规则可见。
- 首页响应不增加 `has_reasoning_tree`；首页不判断推理树是否存在。
- 推理树只在用户进入影响路径页面后，通过独立 `reasoning-trees` 子资源接口读取。
- Anchor Import 失败不删除、回滚或隐藏已成功的 Theme，也不允许任何部分 Anchor Tab 对推理树页面可见。

### 41. Anchor 发布以单个 Theme 为边界且不改变首页 Theme 批次

状态：已确认。

- 同一 Theme Publication Batch 中，每个 Theme 是独立的 Anchor 发布边界；但首页继续展示该批次全部已发布 Theme，不按 Anchor 状态筛选。
- 一个 Theme 的 Anchor 失败不阻塞其他 Theme 的 Anchor 发布，也不改变任何 Theme 的首页可见性。
- 首页 `theme_count` 与 `event_count` 继续完全按 Theme 查询合同计算，Event 按 `event_id` 去重。
- Theme Import V1 的数据库整批原子性与首页最新批次选择保持不变；Anchor 晚到不会改变 Theme `published_at` 或批次排序。

### 42. 推理树列表与详情端点不返回伪成功空结果

状态：已确认。

- 列表端点收到的 `theme_id` 在 `research_themes` 中不存在时，返回 HTTP `404` + `RESEARCH_THEME_NOT_FOUND`。
- Theme 存在但尚无成功 Research Anchor Publication Receipt 时，返回 HTTP `404` + `RESEARCH_REASONING_TREES_NOT_FOUND`。
- 首页始终保留“查看影响路径”入口，不因缺少推理树而隐藏或禁用；缺树只在进入页面并读取该子资源时处理。
- 列表成功返回 HTTP `200`，推理树摘要必须非空，并完整覆盖该 Theme 的 Theme Chain Node Association 集合。
- 详情端点的 `anchor_id` 不存在或不属于路径中的 `theme_id` 时，返回 HTTP `404` + `RESEARCH_REASONING_TREE_NOT_FOUND`，不得泄露其他 Theme 的 Anchor 数据。
- V1 永远不使用 HTTP `200` + 空 `reasoning_trees` 表达待发布、失败或不存在状态。
- 成功 Receipt 存在但摘要集合为空、不完整，或已列出的单树无法重建时，属于服务端数据不变式破坏，不能降级为合法空页。
- 上述不变式破坏统一返回 HTTP `500` + `RESEARCH_REASONING_TREE_INVARIANT_VIOLATION`，用于监控和运维定位；不得返回 `404`、空数组或部分结果。

### 43. Theme 推理树 Data API 分离 Tab 摘要与单树完整事实

状态：已确认。

列表成功响应固定为：

```json
{
  "request_id": "request-uuid",
  "result": {
    "theme": {
      "...": "复用现有 ResearchThemeSummary"
    },
    "reasoning_trees": [
      {
        "anchor_id": "uuid",
        "center_chain_node": {
          "id": "uuid",
          "name": "光模块"
        }
      }
    ]
  }
}
```

单树详情成功响应固定为：

```json
{
  "request_id": "request-uuid",
  "result": {
    "theme_id": "uuid",
    "reasoning_tree": {
      "anchor_id": "uuid",
      "center_chain_node": {
        "id": "uuid",
        "name": "光模块"
      },
      "one_line_conclusion": "中心节点结论",
      "fact_summary": "原子事实汇总",
      "net_direction_summary": "当前净方向",
      "trading_direction": "交易研究指向",
      "next_checkpoint": "下一验证项",
      "event_count": 3,
      "events": [
        {
          "event_id": "uuid",
          "title": "事件标题",
          "summary": "事件摘要",
          "event_time": "2026-07-20T08:00:00Z",
          "evidence_role": "driver",
          "evidence_summary": "该事件承担的证据作用"
        }
      ],
      "path_nodes": [
        {
          "chain_node_id": "uuid",
          "name": "AI芯片",
          "change_direction": "increase",
          "change_summary": "当前变化",
          "impact_summary": "对 Theme 的影响",
          "incoming_transmission_mechanism": null
        }
      ]
    }
  }
}
```

- 列表中的 `theme` 复用现有 `ResearchThemeSummary` 读取合同，页面分析时间仍来自 Theme `published_at`。
- `theme.published_at` 只用于展示分析发布时间，不作为推理树读取的过期条件。
- 列表中的 `reasoning_trees` 只包含 `anchor_id` 和 `center_chain_node`，并按已确认的中心节点规范名称和 UUID 稳定排序。
- 单树详情中的 `theme_id` 必须与路径参数一致，`reasoning_tree.anchor_id` 必须与路径参数一致并属于该 Theme。
- `center_chain_node.name` 与每个 `path_nodes[].name` 都来自当前 Chain Node 主数据，不是 Anchor 快照复制值。
- `events` 是单一去重关联列表，携带 Event 主数据展示字段与 Anchor 证据语义，并按已确认的事件时间正序返回。
- `event_count` 等于该树 `events` 中的去重 `event_id` 数量。
- `path_nodes` 数组顺序就是传导顺序，响应不另外返回 `position` 或 `path_edges`。
- V1 响应不返回 Anchor Receipt 时间、`anchor_type`、`importance`、Anchor 级 `transmission_path`、指数或重复的支持/反证 Event 分组数组；只原样返回 Anchor 级 `support_summary` 与可空的 `counter_summary`。

### 44. Miniapp BFF 镜像推理树列表与单树详情端点

状态：已确认。

- Miniapp BFF 新增 `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees`。
- Miniapp BFF 新增 `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees/{anchor_id}`。
- 小程序只调用这两个 Miniapp BFF API，不得直接访问 Data Service。
- BFF 的每个列表或详情请求只调用一次对应 Data API，不在服务端按 Anchor 扇出 N+1 请求。
- BFF 将 Data Read Model 确定性映射为小程序页面 DTO；单树详情保留单一 `events` 数组，不生成结论、证据、路径或排序语义。
- BFF 遵循现有 Miniapp API 规范，不返回 Data 的 `request_id/result` envelope；列表成功响应直接等于 Data envelope 的 `result`，即 `{ "theme": ..., "reasoning_trees": [...] }`。
- 单树详情成功响应也直接等于 Data envelope 的 `result`，即 `{ "theme_id": ..., "reasoning_tree": ... }`。
- Data `request_id` 只用于 BFF 内部链路追踪，不暴露为 Miniapp 页面 DTO 字段。
- 现有 Miniapp Theme 详情端点保持不变，不嵌入推理树数据。

### 45. BFF 不复制 Event 分组数组

状态：已确认（按最简实现）。

- Miniapp BFF 为每棵推理树返回单一 `events` 数组，字段与 Data Read Model 一致。
- BFF 不另外创建 `atomic_events`、`supporting_evidence` 或 `contradicting_evidence` 数组，避免同一 Event 在响应中重复载荷。
- 小程序在原子事件清单中使用 `evidence_role` 显示驱动、支持、反证或背景标签，并保持全部 Event 的稳定顺序。
- 当前支持和当前反证分别只读取 `support_summary`、`counter_summary`；前端不得从 Event 过滤结果生成 Anchor 汇总结论。

### 46. 小程序使用标准 Taro 页面路由进入影响路径页

状态：已确认。

- 页面路由固定为 `pages/research-theme/reasoning-trees/index`，并注册到 Taro `app.config`。
- 首页 Theme 卡片的“查看影响路径”使用 `Taro.navigateTo`，传递查询参数 `theme_id`：`pages/research-theme/reasoning-trees/index?theme_id=<uuid>`。
- 该页面是非 Tab 页面，不引入自定义路由器；实现遵循当前项目的 Taro 4、React 18 与 TypeScript 技术栈，并兼容微信小程序、抖音小程序和 H5。
- 页面必须先校验 `theme_id`；参数缺失或不是标准小写 UUID 时展示明确错误状态，不调用 Miniapp BFF。
- 页面通过独立的 typed port 和 adapter 调用推理树 BFF，不在页面组件中直接拼接或发送 HTTP 请求。

### 47. 每个 Anchor Tab 首次选中时加载并在页面会话内缓存

状态：已确认（按简单且稳定的实现）。

- 页面打开后先请求推理树列表，生成全部 Anchor Tab，并自动请求排序后第一棵树的详情。
- 用户首次选中尚未加载的 Tab 时，请求该 `anchor_id` 的单树详情。
- 单树成功加载后按 `anchor_id` 缓存在当前页面会话；再次切回同一 Tab 时直接展示缓存，不重复请求。
- Research Anchor 是不可变发布快照，因此 V1 不在每次切换时强制刷新，也不增加后台轮询或缓存失效协议。
- 用户重新进入或刷新页面时重新建立页面缓存，并重新请求列表及默认树详情。
- 某个 Tab 的详情首次加载失败时，只在该 Tab 内容区展示加载失败状态和重试操作，不清空其他已成功加载的 Tab 缓存。
- 页面保持当前选中的失败 Tab，不自动切回上一棵树或离开页面；用户可以重试当前 Tab，也可以切换到其他 Tab。
- 重试只重新请求当前 `anchor_id` 的单树详情，不重新请求 Tab 列表或其他单树。

### 48. 删除未投入使用的旧独立 Anchor 读取合同

状态：已确认。

- 旧 Data `GET /internal/data/v1/research/anchors` 和 `GET /internal/data/v1/research/anchors/{anchor_id}` 不再保留。
- 旧 Miniapp `GET /api/v1/miniapp/research/anchors` 和 `GET /api/v1/miniapp/research/anchors/{anchor_id}` 不再保留。
- 删除仅服务旧 `anchor_type`、`importance` 和独立 Anchor 列表语义的 DTO、application/repository 读取能力、OpenAPI 合同和专属测试。
- V1 不提供兼容别名、弃用窗口或新旧双写；所有 Research Anchor 产品读取统一收敛到 Theme 下的 `reasoning-trees` 子资源。
- 该删除只针对尚未正式投入使用的旧 Anchor 读取合同，不影响现有 Theme API、Theme Import V1 或新 Research Anchor Import V1。
- 上述 Data 与 Miniapp 两组旧端点统一由 TW-04 删除；TW-05 只新增新的推理树 BFF 合同，不让主分支在两个任务之间保留已知失效代理。

### 49. 旧 Anchor 空表断言通过后原名重建

状态：已确认。

- 当前本地 `research_anchors`、`research_anchor_chain_nodes`、`research_anchor_events`、`research_anchor_indices` 均为 `0` 行，且旧 Anchor 语义尚未正式投入使用。
- TW-02 必须使用新的 forward migration，禁止修改历史 migration。
- migration 在任何结构删除前先断言上述四张旧表全部为空；任一表存在数据时立即失败，不静默删除、不猜测转换，也不继续执行后续 DDL。
- 空表断言通过后删除旧 Anchor 子表和主表，再以原表名重建新语义的 `research_anchors`、`research_anchor_events`、`research_anchor_chain_nodes`，并新增 Theme 级 Research Anchor Publication Receipt 表。
- `research_anchor_indices` 在 V1 删除后不重建。
- 旧 Anchor 结构按未投入使用的废弃结构处理；不提供兼容层、数据转换或恢复旧结构的 Down 路径。该 migration 为 forward-only，回退只能通过经审阅的后续 forward migration 或迁移前备份完成。
- 重建不得修改或删除 `research_themes`、Theme 关联、`events`、`chain_node_profiles`、`entity_nodes` 或其他主数据。
- migration 失败必须由 PostgreSQL 事务整体回滚，不能留下新旧结构混合状态。
- TW-02 同步升级本地 `research-theme-dev-reset`。重置 Theme 发布快照时，必须在同一事务内清空所属 Anchor、Path Node、Anchor Event 关联和 Anchor 导入回执，安全禁用并恢复 Theme/Anchor 两类不可变回执触发器；Event、Chain Node、Index、Tag、Raw Document 等主数据继续受计数保护。

### 50. Anchor 发布回执复用 Theme Import V1 的不可变模式

状态：已确认。

- 新表名固定为 `research_anchor_import_receipts`。
- 最小字段固定为：`id`、`theme_id`、`publisher_subject`、`payload_hash`、`anchor_ids_by_center_chain_node_id`、`write_counts`、`published_at`、`imported_at`。
- `theme_id` 必须唯一并外键关联 `research_themes(id)`；一个 Theme 最多存在一条成功 Anchor 发布回执。
- `theme_id` 外键使用限制删除语义，不级联删除回执。Anchor 发布成功后，普通操作删除所属 Theme 必须因不可变回执而失败；只有本地 `research-theme-dev-reset` 可以在受控事务中临时解除回执保护并整体清理 Theme 发布快照。
- `publisher_subject` 保存稳定服务主体，不保存 Bearer token。
- `payload_hash` 是通过全部 V1 结构与排序校验后的合法请求 canonical SHA-256，固定为 64 位小写十六进制文本。
- `anchor_ids_by_center_chain_node_id` 和 `write_counts` 使用受约束 JSONB 保存首次成功结果快照；字段名与 Anchor Import V1 HTTP 响应保持一致。
- `published_at` 和 `imported_at` 均由 Data Service 在首次成功事务内生成；重放返回原值，页面不展示这两个回执时间。
- 回执使用与 `research_theme_import_receipts` 相同原则的 PostgreSQL trigger 禁止 `UPDATE`、`DELETE` 和 `TRUNCATE`。
- 回执对 `(id, theme_id)` 提供复合唯一键；每条 Anchor 使用 `(import_receipt_id, theme_id)` 复合外键关联同一 Theme 的回执，禁止错误绑定其他 Theme 的回执。
- `write_counts` 固定包含 `anchors`、`event_associations`、`path_nodes`、`receipts`；`anchors >= 1`、`event_associations >= anchors`、`path_nodes >= 2 * anchors`、`receipts = 1`，且映射键数量等于 `anchors`。
- 每条 `research_anchors.import_receipt_id` 必填；回执、全部 Anchor 与全部关联记录在同一事务提交。

### 51. research_anchors 使用不可变快照最小主表

状态：已确认。

- `research_anchors` 只包含：`id`、`theme_id`、`center_chain_node_entity_id`、`import_receipt_id`、`one_line_conclusion`、`fact_summary`、`net_direction_summary`、`support_summary`、`counter_summary`、`trading_direction`、`next_checkpoint`、`created_at`。
- `theme_id` 必填，外键关联 `research_themes(id)`，Theme 删除时级联删除其 Anchor。
- `center_chain_node_entity_id` 必填，外键关联 `chain_node_profiles(entity_id)`。
- `import_receipt_id` 必填，外键关联 `research_anchor_import_receipts(id)`。
- `(theme_id, center_chain_node_entity_id)` 建立唯一约束，保证一个 Theme 的一个中心节点恰好最多对应一棵 Anchor。
- `one_line_conclusion`、`fact_summary`、`net_direction_summary`、`support_summary`、`trading_direction`、`next_checkpoint` 必填并做非空白约束；`counter_summary` 可空但非空时必须为非空白文本。Data Import 不生成默认文本。
- `counter_summary` 与 `contradicting` Event 的存在性必须一致，由 Import application service 在写入前整批校验。
- Anchor UUID 继续按已确认的 `theme_id + center_chain_node_entity_id` 固定命名空间 UUIDv5 生成。
- 主表不保存 `analysis_batch_id`、节点 `name`、`anchor_type`、`importance`、Anchor 级 `transmission_path`、`published_at` 或 `updated_at`。
- 节点名称读取当前 Chain Node 主数据；批次和 Theme 发布时间读取所属 Theme；Anchor 发布技术时间读取回执但不进入产品 DTO。

### 52. research_anchor_chain_nodes 保存唯一有序路径节点

状态：已确认。

- `research_anchor_chain_nodes` 不再表示普通 Anchor 与 Chain Node 关联，而是保存一棵 Anchor 唯一线性传导路径中的有序 Path Node。
- 字段固定为：`anchor_id`、`position`、`chain_node_entity_id`、`change_direction`、`change_summary`、`impact_summary`、`incoming_transmission_mechanism`、`created_at`。
- `(anchor_id, position)` 为主键；`position` 从 `1` 开始且必须连续。它是传导路径语义的一部分，不是界面 `display_order`。
- `(anchor_id, chain_node_entity_id)` 建立唯一约束，禁止同一节点在一条路径中重复并防止循环表示。
- `anchor_id` 外键关联 `research_anchors(id)` 并在 Anchor 删除时级联删除；`chain_node_entity_id` 外键关联 `chain_node_profiles(entity_id)`。
- `change_direction` 只允许 `increase`、`decrease`、`mixed`、`unchanged`、`uncertain`；两个说明字段必填且非空白。
- `position = 1` 时 `incoming_transmission_mechanism` 必须为 `NULL`；`position > 1` 时必须为非空白文本。
- 每条路径至少两个节点、位置连续、中心节点恰好出现一次，由 Import application service 在写入前整批校验；任一失败拒绝整个 Theme Anchor 集合。

### 53. research_anchor_events 保存唯一 Event 证据关联

状态：已确认。

- `research_anchor_events` 字段固定为：`anchor_id`、`event_id`、`evidence_role`、`evidence_summary`、`created_at`。
- `(anchor_id, event_id)` 为主键，保证同一 Event 在一棵 Anchor 中最多出现一次并只承担一个角色。
- `anchor_id` 外键关联 `research_anchors(id)` 并在 Anchor 删除时级联删除；`event_id` 外键关联 `events(id)`，删除 Anchor 关联不得删除 Event 主数据。
- `evidence_role` 只允许 `driver`、`supporting`、`contradicting`、`context`；`evidence_summary` 必填且做非空白约束。
- 每棵 Anchor 至少包含一个 `driver` Event，并且全部 Event 必须属于父 Theme 已发布的 Theme Event Evidence Association 子集；这两项由 Import application service 在写入前整批校验。
- 任一角色、摘要、重复或父 Theme 证据边界校验失败时，整个 Theme Anchor 集合拒绝并回滚。

### 54. 共享 fixture 只用于确定性测试

状态：已确认。

- TW-01 的共享测试样例统一存放在 `src/testdata/reasoning-tree-v1/`。
- fixture 至少覆盖：多 Anchor、包含反证、无反证且明确未量化、Theme 已存在但推理树尚未发布四组场景。
- Data、Miniapp BFF 和小程序测试应复用该目录中的合同事实，禁止各自维护语义不一致的重复样例。
- 该目录不是 AI 分析师输出目录、生产 outbox、seed 数据或运行时导入入口。
- 真实推理树始终由 AI 分析师的发布器通过 Research Anchor Import V1 推送，由 Data Service 校验并写入 PostgreSQL。
- 测试 fixture 不得被生产启动流程自动导入，也不得作为没有真实分析结果时的产品回退数据。

### 55. 分析师 V3 与 Anchor Import V1 使用同名字段

状态：已确认。

- AI 分析师 V3 尚无需要兼容的已发布旧结构；V3 的 Anchor、Event Evidence 和 Path Node 字段直接采用 Anchor Import V1 名称与嵌套结构。
- 发布器只使用 Theme Import V1 返回的映射确定 `theme_id`，把该 Theme 的完整 `anchors` 集合装入请求并原样发送。
- 发布器不得重命名、拼接、推断、补写或根据 Markdown 重新生成任何 Anchor 字段。
- Data Service 只做严格结构、排序、引用和领域不变量校验；BFF 只做确定性 DTO 映射；小程序只做展示分组和交互状态管理。

字段血缘固定为：

| AI 分析 V3 / Import V1 | PostgreSQL | Data / BFF 读取 | 页面用途 |
| --- | --- | --- | --- |
| 顶层 `theme_id`（由 Theme Import 回执解析） | `research_anchors.theme_id`、回执 `theme_id` | 列表 `theme.id` / 详情 `theme_id` | 页面归属与路由校验 |
| `center_chain_node_id` | `research_anchors.center_chain_node_entity_id` | `center_chain_node.id`，并从主数据补 `name` | Anchor Tab 与中心节点名称 |
| `one_line_conclusion` | `research_anchors.one_line_conclusion` | 同名 | Anchor 结论 |
| `fact_summary` | `research_anchors.fact_summary` | 同名 | 原子事实汇总标题 |
| `net_direction_summary` | `research_anchors.net_direction_summary` | 同名 | 当前净方向判断 |
| `support_summary` | `research_anchors.support_summary` | 同名 | 当前支持的 Anchor 级推导结论 |
| `counter_summary` | `research_anchors.counter_summary` | 同名，可为 `null` | 当前反证的 Anchor 级推导结论 |
| `trading_direction` | `research_anchors.trading_direction` | 同名 | 交易研究指向 |
| `next_checkpoint` | `research_anchors.next_checkpoint` | 同名 | 下一检查点 |
| `events[].event_id` | `research_anchor_events.event_id` | 同名，并从 Event 主数据补标题、摘要和时间 | 原子事实与证据条目 |
| `events[].evidence_role` | `research_anchor_events.evidence_role` | 同名 | 单个 Event 的驱动、支持、反证或背景角色标签 |
| `events[].evidence_summary` | `research_anchor_events.evidence_summary` | 同名 | Event 对当前 Anchor 的证据说明 |
| `path_nodes[]` 数组位置 | `research_anchor_chain_nodes.position` | 数组顺序 | 单条传导路径顺序 |
| `path_nodes[].chain_node_id` | `research_anchor_chain_nodes.chain_node_entity_id` | `chain_node_id`，并从主数据补 `name` | 路径节点 |
| `path_nodes[].change_direction` | 同名 | 同名 | 变化方向视觉与文案 |
| `path_nodes[].change_summary` | 同名 | 同名 | 节点变化说明 |
| `path_nodes[].impact_summary` | 同名 | 同名 | 节点影响说明 |
| `path_nodes[].incoming_transmission_mechanism` | 同名 | 同名 | 与前一节点之间的传导机制 |

- Anchor UUID、回执 ID、payload hash、写入计数及 Anchor 发布时间均由 Tidewise 生成，不属于 AI 分析 V3 字段。
- Chain Node 名称和 Event 展示字段来自 Tidewise 主数据，不由 AI 发布器复制。

### 56. 页面区分不可用状态与可重试故障

状态：已确认。

- 路由 `theme_id` 缺失或格式非法时，页面显示参数错误，不调用 BFF。
- 推理树列表请求收到 Theme 不存在的 `404` 时，小程序展示“该研究主题暂不可用”；Theme 存在但推理树尚未发布时，展示“影响路径暂未生成”。两种状态均提供返回操作，BFF 保留内部错误语义。
- 列表请求发生网络错误、超时或可恢复服务错误时，页面展示“加载失败”与重试操作；重试重新请求列表，成功后再加载默认第一棵树。
- 列表端点按合同不返回合法空集合，因此页面不设计“0 棵推理树”的正常空态。
- 单个 Tab 详情加载失败继续遵循已确认的局部错误与局部重试规则，不清空列表或其他已加载树。
- 小程序不向用户直接展示 Data 或 BFF 内部错误码、堆栈或服务实现信息。

### 57. V1 数据库关系图

状态：已确认合同的结构化表达。

```mermaid
erDiagram
    RESEARCH_THEMES ||--o| RESEARCH_ANCHOR_IMPORT_RECEIPTS : "has successful publication"
    RESEARCH_THEMES ||--o{ RESEARCH_ANCHORS : owns
    RESEARCH_ANCHOR_IMPORT_RECEIPTS ||--|{ RESEARCH_ANCHORS : publishes
    CHAIN_NODE_PROFILES ||--o{ RESEARCH_ANCHORS : "is center node"
    RESEARCH_ANCHORS ||--|{ RESEARCH_ANCHOR_EVENTS : evidences
    EVENTS ||--o{ RESEARCH_ANCHOR_EVENTS : referenced_by
    RESEARCH_ANCHORS ||--|{ RESEARCH_ANCHOR_CHAIN_NODES : contains
    CHAIN_NODE_PROFILES ||--o{ RESEARCH_ANCHOR_CHAIN_NODES : appears_in

    RESEARCH_ANCHOR_IMPORT_RECEIPTS {
        uuid id PK
        uuid theme_id UK_FK
        text publisher_subject
        char_64 payload_hash
        jsonb anchor_ids_by_center_chain_node_id
        jsonb write_counts
        timestamptz published_at
        timestamptz imported_at
    }

    RESEARCH_ANCHORS {
        uuid id PK
        uuid theme_id FK
        uuid center_chain_node_entity_id FK
        uuid import_receipt_id FK
        text one_line_conclusion
        text fact_summary
        text net_direction_summary
        text trading_direction
        text next_checkpoint
        timestamptz created_at
    }

    RESEARCH_ANCHOR_EVENTS {
        uuid anchor_id PK_FK
        uuid event_id PK_FK
        text evidence_role
        text evidence_summary
        timestamptz created_at
    }

    RESEARCH_ANCHOR_CHAIN_NODES {
        uuid anchor_id PK_FK
        integer position PK
        uuid chain_node_entity_id FK
        text change_direction
        text change_summary
        text impact_summary
        text incoming_transmission_mechanism
        timestamptz created_at
    }
```

- 图中的 `UK_FK` 表示唯一外键；`PK_FK` 表示同时属于复合主键和外键。
- “至少一个 Event”“至少两个连续 Path Node”“中心节点恰好出现一次”“Anchor 集合完整覆盖 Theme Chain Node Association”由 Import application service 在同一事务写入前校验。

## 待确认内容

当前 TW-01 高影响业务与合同问题均已确认，文档一致性与 fixture 验证已通过，并已由用户验收。后续任务不得在未重新评审合同的情况下改变 V1 语义。

## 范围外

- Theme Import V1 变更或 Theme Import V2。
- migration、Go 代码和小程序实现。
- AI Prompt、Adapter、Outbox 和 Publisher 实现。
- 运行时 Neo4j 推理或 Markdown 反向解析。
- 跨批次 Research Thesis 跟踪。
