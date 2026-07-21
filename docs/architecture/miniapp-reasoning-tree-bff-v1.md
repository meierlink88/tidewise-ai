# Miniapp 推理树 BFF 合同 V1

## 状态

- 所属任务：TW-05
- 当前状态：实现、全量验证与双轴 Code Review 已完成，进入 PR Review
- 前置条件：TW-04 Data 推理树读取接口已合并至 `main`

## 用途

Miniapp Application Backend Service 将 Data Service 的 Theme 推理树已发布快照映射为小程序可直接渲染的 DTO。它是 Miniapp Frontend 唯一可调用的后端边界；不暴露 Data Service 的 envelope、请求 ID、认证方式或内部错误信息。

## 范围

新增两个只读 HTTP 接口：

1. `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees`
2. `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees/{anchor_id}`

每个 BFF 请求只调用一次对应的 Data API：

| Miniapp BFF | Data Service |
| --- | --- |
| `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees` | `GET /internal/data/v1/research/themes/{theme_id}/reasoning-trees` |
| `GET /api/v1/miniapp/research/themes/{theme_id}/reasoning-trees/{anchor_id}` | `GET /internal/data/v1/research/themes/{theme_id}/reasoning-trees/{anchor_id}` |

调用 Data Service 时继续使用 Miniapp Service 身份令牌和 `X-Request-ID` 转发机制；这些实现细节不出现在 Miniapp 响应中。

## 请求合同

- `theme_id` 和 `anchor_id` 必须是标准小写 UUID；缺失、非 UUID 或包含大写字符时返回 `400 INVALID_REQUEST`。
- 两个端点不接受 query 参数；任一 query 参数存在时返回 `400 INVALID_REQUEST`。
- 列表端点只需要 `theme_id`；详情端点还需要属于该 Theme 的 `anchor_id`。

## 成功响应

成功时返回 HTTP `200`，直接返回对应 Data envelope 的 `result` 内容，不保留 Data 的 `{ request_id, result }` 外壳。

列表响应：

```json
{
  "theme": {
    "id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    "name": "AI 算力扩产",
    "one_line_conclusion": "算力资本开支推升，AI 供应链涨价周期开启",
    "impact_level": "high",
    "transmission_path": "...",
    "trading_direction": "...",
    "transmission_stage": "diffusion",
    "next_checkpoint": "...",
    "market_confirmation_summary": "...",
    "published_at": "2026-07-21T00:00:00Z",
    "affected_chain_nodes": [],
    "related_indices": [],
    "supporting_event_count": 2,
    "contradicting_event_count": 0
  },
  "reasoning_trees": [
    {
      "anchor_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
      "center_chain_node": {
        "id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
        "name": "光模块"
      }
    }
  ]
}
```

详情响应：

```json
{
  "theme_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "reasoning_tree": {
    "anchor_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
    "center_chain_node": { "id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "name": "光模块" },
    "one_line_conclusion": "...",
    "fact_summary": "...",
    "net_direction_summary": "...",
    "support_summary": "...",
    "counter_summary": null,
    "trading_direction": "...",
    "next_checkpoint": "...",
    "event_count": 2,
    "events": [
      {
        "event_id": "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
        "title": "...",
        "summary": "...",
        "event_time": "2026-07-20T00:00:00Z",
        "evidence_role": "driver",
        "evidence_summary": "..."
      }
    ],
    "path_nodes": [
      {
        "chain_node_id": "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
        "name": "光模块",
        "change_direction": "increase",
        "change_summary": "...",
        "impact_summary": "...",
        "incoming_transmission_mechanism": null
      },
      {
        "chain_node_id": "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
        "name": "算力基础设施",
        "change_direction": "increase",
        "change_summary": "...",
        "impact_summary": "...",
        "incoming_transmission_mechanism": "光模块交付改善提升算力集群扩容能力"
      }
    ]
  }
}
```

`events` 保持唯一、完整、稳定排序的单一数组。BFF 不复制成支持/反证数组，也不重排；小程序只在原子事件清单中显示 `evidence_role` 标签。`support_summary` 与可空的 `counter_summary` 由 BFF 原样映射，不能从 Event 推断。`event_time` 使用 UTC RFC3339 或 `null`，路径顺序和 Tab 顺序由 Data 决定并原样保留。

## 错误合同

所有错误均不返回 Data 的 `request_id`、服务身份、上游 URL 或原始错误文本。

```json
{
  "error": {
    "code": "RESEARCH_REASONING_TREES_NOT_FOUND",
    "message": "research Theme has no published reasoning trees"
  }
}
```

| HTTP 状态 | code | 语义 |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | 路径参数非法或请求带有不允许的 query 参数。 |
| 404 | `RESEARCH_THEME_NOT_FOUND` | Theme 不存在；前端展示“该研究主题暂不可用”。 |
| 404 | `RESEARCH_REASONING_TREES_NOT_FOUND` | Theme 存在但尚无成功发布的完整推理树；前端展示“影响路径暂未生成”。 |
| 404 | `RESEARCH_REASONING_TREE_NOT_FOUND` | Anchor 不属于该 Theme 或不存在；前端只影响当前 Tab。 |
| 502 | `RESEARCH_DATA_UNAVAILABLE` | Data 网络、超时、非上述业务错误、协议/载荷错误或未知受控枚举；前端提供重试，不展示内部细节。 |

## 不变量

- BFF 不访问 PostgreSQL、Neo4j、repository 或 Data Service 的 Go 实现。
- BFF 不生成、拼接、补写或推断 Theme、Anchor、证据、路径或交易语义。
- BFF 不按 Anchor 扇出请求：列表只调一次 Data 列表，详情只调一次 Data 详情。
- BFF 不接受或返回旧独立 Anchor 资源路径。
- BFF 仅把被 Data 合同允许的受控枚举映射给 Miniapp；未知值按上游不可用处理，不能静默透传给页面。
- 现有 Miniapp Theme 首页列表和 Theme 详情 API 维持不变。

## 实现边界

实现限定在 `src/backend/services/miniapp/`：

- 扩展 `dataclient` 的局部 DTO、typed client、fake 与 drift test。
- 扩展 `usecase` 的局部页面 DTO、输入校验、枚举校验和错误映射。
- 扩展 `transport` 的两个路由与稳定错误响应。
- 增加 fixture 映射、单次调用、错误语义、未知枚举和 HTTP 合同测试。

不修改 Data Service、数据库 schema/migration、Research Anchor Import、Miniapp Frontend、Taro 路由或 AI 分析师项目。

## 验收

1. 共享 fixture `01`、`02`、`03` 的 Data result 可被 BFF 一对一映射为上述 DTO。
2. fixture `04` 的未发布推理树状态映射为 `404 RESEARCH_REASONING_TREES_NOT_FOUND`。
3. Theme 缺失和 Anchor 缺失分别保留对应的稳定 `404` code。
4. 每个 BFF 接口调用一次且仅一次对应 Data client 方法。
5. `counter_summary: null`、未量化路径说明和 `event_time: null` 不会被改写、删除或重排。
6. 未知枚举、Data 超时、网络失败和无效响应安全地映射为 `502 RESEARCH_DATA_UNAVAILABLE`。
7. 现有 Theme 首页列表与详情的回归测试通过。
