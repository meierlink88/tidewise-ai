# Research Theme 推理树读取 V1 实施 Spec

## 状态

- 任务：TW-04
- 状态：实现、全量验证与 Code Review 已完成，等待 PR Review
- 前置：TW-01 合同、TW-02 数据结构、TW-03 Anchor Import V1 均已验收
- 权威语义：`docs/architecture/reasoning-tree-contract-v1.md`
- 共享样例：`src/testdata/reasoning-tree-v1/`

本文只把已冻结合同收敛为 TW-04 的可实施边界，不重新定义 Research Anchor，不修改 Theme Import V1 或 Anchor Import V1。

## 用途

Data Service 为一个已发布 Research Theme 提供两级只读能力：

1. 一次返回 Theme 摘要和全部 Research Anchor Tab 摘要。
2. 按 `theme_id + anchor_id` 返回一棵完整推理树。

页面先加载稳定 Tab 清单，再按当前 Tab 加载单树详情。推理树完全读取 PostgreSQL 已发布快照，不查询 Neo4j，不解析 Markdown，也不在 Data、BFF 或前端重新推理。

## 已发现的现状冲突

当前旧独立 Anchor Data API 和 Miniapp API 仍引用已经由 TW-02 删除的 `anchor_type`、`importance`、Anchor `published_at`、Anchor 指数等旧结构。它们不是可继续兼容的产品合同，而是尚未投入使用的失效路径。

TW-04 必须在新增 Data 推理树查询的同一改动中删除两组旧端点及其专属实现，避免主分支保留已知不可用的 Miniapp 代理。TW-05 只负责新增推理树 BFF，不再承担旧合同清理。

## HTTP 合同

两个端点均要求 Bearer service token 和现有 `data.research.read` scope，复用 Data Service 的 `request_id/result` envelope、认证错误和请求追踪方式。

### 推理树列表

```http
GET /api/data/v1/research/themes/{theme_id}/reasoning-trees
Authorization: Bearer <service-token>
```

- 不接收 `window_hours`、`limit`、`cursor` 或其他查询参数。
- 不分页；一次返回该 Theme 已发布的全部 Anchor 摘要。
- 即使 Theme 已离开首页时间窗口，只要 `theme_id` 明确且发布快照存在，仍可读取。

成功响应固定为：

```json
{
  "request_id": "request-uuid",
  "result": {
    "theme": {
      "...": "完整复用现有 ResearchThemeSummary"
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

`theme` 的字段和数组行为与现有 Theme 列表项完全相同；本接口不增加 `has_reasoning_tree`、Receipt 时间或 Anchor 数量冗余字段。

### 单树详情

```http
GET /api/data/v1/research/themes/{theme_id}/reasoning-trees/{anchor_id}
Authorization: Bearer <service-token>
```

- 不接收时间窗口或其他查询参数。
- `anchor_id` 必须属于 URL 中的 `theme_id`；只凭 Anchor ID 不能跨 Theme 读取。

成功响应固定为：

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
      "support_summary": "当前支持的整体结论",
      "counter_summary": "当前反证的整体结论或 null",
      "trading_direction": "交易研究指向",
      "next_checkpoint": "下一验证项",
      "event_count": 2,
      "events": [
        {
          "event_id": "uuid",
          "title": "事件标题",
          "summary": "事件摘要",
          "event_time": "2026-07-20T01:00:00Z",
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

- `event_time` 沿用 Event 当前可空语义；无时间时 JSON 字段按现有 Event DTO 约定省略。
- `support_summary` 原样返回且非空；`counter_summary` 在无 `contradicting` Event 时显式返回 `null`。
- `event_count` 等于响应中去重后的 `events` 数量。
- 中心节点与路径节点名称实时读取当前 Chain Node 主数据，不复制 Anchor 快照名称。
- 响应不返回 `position`；`path_nodes` 数组顺序本身就是传导顺序。
- 响应不返回指数、Receipt、`anchor_type`、`importance` 或 Anchor 级 `transmission_path`。

精确成功载荷以以下共享 fixture 为准：

- `01-reasoning-tree-list-result.json`
- `02-reasoning-tree-with-contradiction-result.json`
- `03-reasoning-tree-without-contradiction-unquantified-result.json`

## 稳定排序

- `reasoning_trees`：中心 Chain Node 当前规范名称使用 PostgreSQL `COLLATE "C" ASC`，名称相同时按中心节点 UUID `ASC`。
- `events`：`event_time ASC NULLS LAST`，时间相同时按 `event_id ASC`。
- `path_nodes`：按持久化 `position ASC`，不由应用层重排或推断路径。
- 空集合一律编码为 `[]`，不返回 `null`。

## 首页与时间语义

- 现有 `GET /api/data/v1/research/themes` 完全不变，继续按 Theme 批次、查询窗口和游标分页返回首页内容。
- 首页 Theme 查询不得 join、filter 或探测 `research_anchor_import_receipts`。
- 首页不增加 `has_reasoning_tree`，并始终保留“查看影响路径”入口。
- Anchor 尚未发布或发布失败不删除、不隐藏、不重新排序 Theme。
- 现有 `GET /api/data/v1/research/themes/{theme_id}` 合同也保持不变，不嵌入推理树。
- Theme `published_at` 在推理树页面仅用于显示分析时间，不构成推理树过期条件。

## 错误合同

路径参数无法解析为 UUID 时返回：

- HTTP `400` + `INVALID_REQUEST`。

资源错误必须明确区分：

- Theme 不存在：HTTP `404` + `RESEARCH_THEME_NOT_FOUND`。
- Theme 存在但没有成功 Anchor Publication Receipt：HTTP `404` + `RESEARCH_REASONING_TREES_NOT_FOUND`。
- `anchor_id` 不存在或不属于 URL 中的 Theme：HTTP `404` + `RESEARCH_REASONING_TREE_NOT_FOUND`。

共享 `04-theme-without-reasoning-trees-error.json` 冻结缺树响应。

成功 Receipt 已存在，但映射、计数或持久化推理树无法满足冻结不变量时返回：

- HTTP `500` + `RESEARCH_REASONING_TREE_INVARIANT_VIOLATION`。

该错误不得降级为 `404`、`200` 空数组或部分结果。数据库连接、查询或解码等普通意外故障继续使用现有 Data Service `500` 仓储失败合同。

## 读取不变量

列表成功前至少验证：

1. Theme 存在。
2. Theme 存在唯一成功 Anchor Receipt。
3. Receipt 的 `anchor_ids_by_center_chain_node_id` 非空，且与持久化 Anchor 的中心节点到 Anchor ID 映射完全相等。
4. Receipt `write_counts` 与该 Receipt 下的 Anchor、Event Association、Path Node 实际计数相等。
5. 返回的 Anchor 集合非空，并完整覆盖 Receipt 映射。

单树详情成功前至少验证：

1. Anchor 属于 URL 中的 Theme 和该 Theme 的成功 Receipt。
2. Anchor 的中心节点存在，并在路径中恰好出现一次。
3. 路径至少两个节点，`position` 从 1 连续且无重复节点。
4. Anchor 至少包含一个 `driver` Event；Event 关联无重复。
5. 所有 Event 与 Chain Node 展示主数据均可读取。

上述不变量在正常写入路径已由 TW-02/TW-03 保证；读取层仍须 fail closed，防止损坏数据被包装成合法产品结果。

## 查询与模块边界

- PostgreSQL 是唯一读取事实源。
- handler 不直接访问数据库；由 research application service 调用 repository read interface。
- 列表查询必须一次聚合 Theme 摘要和全部 Anchor Tab，不按 Anchor 循环查询中心节点。
- 单树详情必须一次聚合 Anchor、Event 和 Path Node，不按 Event 或节点循环查询。
- 可以使用少量固定数量的 SQL 查询完成不变量核验，但查询数量不得随 Anchor、Event 或 Path Node 数量增长。
- 复用现有 Theme Summary 映射；不要复制一套会漂移的 Theme DTO。
- 不为了两条查询建立通用多态研究树框架。

## 旧合同删除

TW-04 同步删除：

- Data `GET /internal/data/v1/research/anchors`。
- Data `GET /internal/data/v1/research/anchors/{anchor_id}`。
- Miniapp `GET /api/miniapp/v1/research/anchors`。
- Miniapp `GET /api/miniapp/v1/research/anchors/{anchor_id}`。
- 仅服务上述旧端点的 route、handler、use case、repository query/interface、DTO、Miniapp Data client、OpenAPI schema/drift anchor 和专属测试。

不提供兼容别名、弃用窗口或新旧双写。删除后，旧路径应不再注册，OpenAPI 不再声明旧 Data 路径。现有 Theme API、Theme Import V1、Anchor Import V1、Miniapp Theme API 均不受影响。

## 实现范围

最小新增或调整组件：

- Data research repository read model 与 PostgreSQL adapter。
- Data research application service 的列表和单树详情用例。
- Internal Data API handler、route、错误映射和 OpenAPI 合同。
- 共享 fixture 驱动的 transport、application、repository 与 PostgreSQL 集成测试。
- 上述旧 Data/Miniapp Anchor 合同清理。

## 范围外

- 新增 Miniapp 推理树 BFF 端点或 DTO；属于 TW-05。
- 小程序 typed port、adapter、路由、状态或页面；属于 TW-06/TW-07。
- 修改 Theme 首页列表、Theme Detail、Theme Import V1 或 Anchor Import V1。
- AI 分析师 Prompt、结构化输出、Publisher 或联调实现。
- Anchor 指数、Neo4j、异步查询、分页、人工审核、跨批次 Thesis 和运行时因果推理。
- 数据库 migration；TW-04 只读取 TW-02 已落地结构。

## TDD 与验收

实现按以下可观察行为推进：

1. 列表端点精确匹配共享多 Anchor fixture，并验证稳定 Tab 顺序。
2. 两个单树 fixture 分别验证有反证、无反证与未量化文本的原样读取。
3. 历史 Theme 离开首页窗口后，仍可按明确 `theme_id` 读取推理树。
4. Theme 不存在、Theme 缺树、Anchor 不存在和 Anchor 属于其他 Theme 返回各自错误码。
5. Receipt 映射、数量、路径或证据被破坏时返回不变量错误，绝不返回部分结果。
6. Event `NULL` 时间排在末尾；同时间、同名时使用 UUID 稳定打破平局。
7. 无反证时 `counter_summary: null` 原样返回，Event 数组仍保持非空且不返回 `null`。
8. 查询数量不随 Anchor/Event/Path Node 数量增长。
9. 首页 Theme 列表在有树、无树两种数据状态下响应一致，不增加任何树状态字段。
10. 旧 Data/Miniapp Anchor 路径不再注册，旧 OpenAPI 路径和旧 client contract 已删除。
11. 现有 Theme API、Theme Import V1、Anchor Import V1 和全量相关测试保持通过。

TW-04 完成条件是：新 Data 两端点、严格错误与不变量、稳定排序、旧合同清理和回归测试全部通过；不以 TW-05 BFF 或小程序页面完成为条件。
