# Research Anchor Import V1 实施 Spec

## 状态

- 任务：TW-03
- 状态：实现与验证已完成，等待用户验收
- 前置：TW-01 合同已冻结，TW-02 数据结构已合并
- 权威语义：`docs/architecture/reasoning-tree-contract-v1.md`
- 共享样例：`src/testdata/reasoning-tree-v1/`

本文只把冻结合同收敛为 TW-03 的可实施边界，不修改 Theme Import V1，不重新定义 Research Anchor 领域语义。

## 用途

为 AI 投研分析师提供同步内部接口，按一个已正式发布的 Research Theme 原子发布其完整 Research Anchor 集合。一次成功发布同时写入：

- 一条不可变 Anchor Publication Receipt。
- 该 Theme 的全部 Research Anchor。
- 每棵 Anchor 的全部 Event Evidence Association。
- 每棵 Anchor 的完整有序 Path Node。

任一 Anchor 或关联事实无效时，整个 Theme 的 Anchor 集合回滚，不产生部分可见推理树。

## HTTP 合同

### 请求

```http
POST /internal/data/v1/research-anchor-imports
Authorization: Bearer <service-token>
Content-Type: application/json
```

调用主体必须具有 `data.research.import` scope。请求体不得声明发布主体。

顶层严格只允许：

```json
{
  "theme_id": "11111111-1111-4111-8111-111111111111",
  "anchors": []
}
```

每个 Anchor 严格只允许：

```json
{
  "center_chain_node_id": "22222222-2222-4222-8222-222222222222",
  "one_line_conclusion": "中心节点结论",
  "fact_summary": "原子事实汇总",
  "net_direction_summary": "当前净方向",
  "support_summary": "当前支持的整体结论",
  "counter_summary": "当前反证的整体结论",
  "trading_direction": "交易研究指向",
  "next_checkpoint": "下一验证项",
  "events": [
    {
      "event_id": "55555555-5555-4555-8555-555555555555",
      "evidence_role": "driver",
      "evidence_summary": "该事件承担的证据作用"
    }
  ],
  "path_nodes": [
    {
      "chain_node_id": "44444444-4444-4444-8444-444444444444",
      "change_direction": "increase",
      "change_summary": "当前变化",
      "impact_summary": "对 Theme 的影响",
      "incoming_transmission_mechanism": null
    }
  ]
}
```

未知字段、重复 JSON key、错误类型和非标准小写 UUID 均拒绝。

### 成功响应

首次成功返回 `201 Created`；同主体、同 Theme、同 payload 重放返回 `200 OK`：

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
      "anchors": 2,
      "event_associations": 4,
      "path_nodes": 4,
      "receipts": 1
    },
    "published_at": "2026-07-20T10:00:00Z",
    "imported_at": "2026-07-20T10:00:00Z",
    "replayed": false
  }
}
```

重放返回首次成功时的身份、映射、计数和时间，只动态改变 `replayed`。

## 前置条件与引用边界

1. `theme_id` 必须引用已通过 Theme Import V1 正式发布的 Theme。
2. Theme 必须具有可解析的 Theme Publication Receipt；没有回执的历史 Theme 一律拒绝，不提供环境绕过。
3. 当前发布主体必须与 Theme Publication Receipt 的稳定主体一致。
4. Theme 即使已不是最新批次，也允许延迟发布 Anchor；Anchor 发布时间不改变 Theme `published_at` 和首页批次排序。
5. `anchors[].center_chain_node_id` 集合必须与 Theme Chain Node Association 集合完全相等。
6. 每个中心节点只出现一次，并对应一棵 Anchor。
7. Anchor Event 必须已经属于父 Theme 的 Theme Event Evidence 集合。
8. Path Node 只要求引用现有 Chain Node；辅助 Path Node 不要求属于 Theme 中心节点集合，也不校验正式产业链关系。

## Anchor 不变量

- `one_line_conclusion`、`fact_summary`、`net_direction_summary`、`support_summary`、`trading_direction`、`next_checkpoint` 必填且非空白，Data 不生成或改写。
- 存在 `contradicting` Event 时 `counter_summary` 必填且非空白；不存在时必须显式为 `null`。
- 每棵 Anchor 至少一个 `driver` Event。
- Event 角色只允许 `driver`、`supporting`、`contradicting`、`context`。
- 同一 Event 在同一 Anchor 内唯一且只能承担一个角色。
- `evidence_summary` 必填且非空白。
- Path 至少包含两个不同 Chain Node。
- Path 必须包含中心 Chain Node 且恰好一次。
- Path 禁止重复节点和循环。
- `change_direction` 只允许 `increase`、`decrease`、`mixed`、`unchanged`、`uncertain`。
- `change_summary`、`impact_summary` 必填且非空白。
- 第一个 Path Node 的 `incoming_transmission_mechanism` 必须显式为 `null`。
- 第二个及之后 Path Node 的 `incoming_transmission_mechanism` 必须为非空白文本。

## 确定性、幂等与并发

- `theme_id` 是唯一幂等身份，不增加额外幂等键或状态查询接口。
- `anchors` 按 `center_chain_node_id` 升序；每棵 Anchor 的 `events` 按 `event_id` 升序。
- `path_nodes` 顺序是业务事实，不允许重排。
- Data 校验顺序，不静默规范化或排序。
- 合法请求使用与 Theme Import V1 相同的 canonical JSON SHA-256 规则计算 `payload_hash`。
- Anchor UUID 使用冻结 UUIDv5 命名空间，由 `theme_id + NUL + center_chain_node_id` 确定性生成。
- 同一 Theme 的并发导入在 PostgreSQL 事务内串行仲裁。
- 同主体、同 hash 返回首次回执；不同主体或不同 hash 返回冲突。
- 重放前验证回执映射、数量与持久化事实一致；不完整回执属于服务端不变量破坏。

## 事务顺序

单个数据库事务内依次：

1. 获取 Theme 级事务锁。
2. 查询已有 Anchor Receipt 并处理重放或冲突。
3. 读取并校验 Theme Publication Receipt、发布主体及 Theme 关联边界。
4. 批量核验所有 Event 和 Chain Node 引用。
5. 生成确定性 Anchor IDs、receipt ID、payload hash、计数和统一服务端时间。
6. 写入 receipt、全部 Anchor、Event Association 和有序 Path Node。
7. 查询核验映射与写入计数后提交。

任何步骤失败，整个事务回滚；不修改 Theme、Theme Association、Event、Chain Node 或正式产业链关系。

## 错误合同

- `400 INVALID_REQUEST`：JSON、未知字段、类型或基础格式错误。
- `400 RESEARCH_ANCHOR_IMPORT_REJECTED`：文本、枚举、顺序、重复项、完整覆盖、路径或 driver 等合同错误。
- `422 RESEARCH_ANCHOR_REFERENCE_NOT_FOUND`：Theme、Event 或 Chain Node 不存在。
- `422 RESEARCH_ANCHOR_REFERENCE_INVALID`：历史 Theme 无发布回执、Event 越过 Theme 证据边界、中心节点越过 Theme Association 边界。
- `409 RESEARCH_ANCHOR_PAYLOAD_CONFLICT`：同 Theme 已发布不同 payload。
- `409 RESEARCH_ANCHOR_PUBLISHER_CONFLICT`：调用主体与 Theme 或现有 Anchor Receipt 所有者不一致。
- 项目标准 `401`、`403`、`413`、`500` 保持不变。

可定位错误返回 `center_chain_node_id`、`path` 和 `reference`。一个请求存在多个错误时，只按确定性请求遍历顺序返回第一个错误，不聚合错误列表。

## 实现边界

新增最小组件：

- `domain/researchanchorimport`：严格解码、验证、canonical hash、确定性 Anchor ID。
- `usecase/researchanchorimport`：事务编排、引用与主体校验、幂等和结果。
- Anchor Import repository interface 与 PostgreSQL transaction adapter。
- Internal Data API handler、依赖注入和错误映射。

优先复用 Theme Import V1 已验证的严格 JSON、canonicalization、事务、回执核验和 HTTP envelope 模式，但 Anchor 保持独立领域包与 repository seam，不把两种发布合同混成通用导入框架。

## 范围外

- 修改 Theme Import V1。
- Anchor 查询 API、Miniapp BFF 和小程序页面；分别属于 TW-04、TW-05、TW-06/TW-07。
- AI 分析师 Prompt、Adapter、Outbox 或 Publisher。
- Anchor 指数、Neo4j、异步任务、状态查询、人工审核和跨批次 Research Thesis。
- 为历史 Theme 补造发布回执或 Anchor。

## 验收测试

- 共享多 Anchor fixture 首次导入成功，结果映射和 counts 精确匹配。
- 相同请求重放不增加 receipt、Anchor、Event 或 Path Node。
- payload 冲突、发布主体冲突及无 Theme Receipt 均拒绝。
- 缺失 Theme/Event/Chain Node 与越界 Theme Event/中心节点返回准确错误。
- 缺失支持汇总、反证汇总与 Event 角色不一致、缺失 driver、重复 Event、错误排序、路径过短、重复/循环节点、缺中心节点和非法入边机制均拒绝。
- 任一 Anchor 无效时数据库零部分写入。
- 并发相同请求只产生一次成功事实，另一次返回重放。
- 真实 PostgreSQL 集成测试证明 receipt、Anchor、Event、Path Node 同事务写入并可核验。
- 现有 Theme Import V1 与 Data Service 全量测试保持通过。
