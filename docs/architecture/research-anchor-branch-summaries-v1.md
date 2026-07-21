# Research Anchor 支持与反证汇总合同修正 V1

## 状态

- 任务：TW-07 前置修正
- 当前状态：Implement 与验证已完成，等待验收
- 原因：已冻结的旧合同把 Event 分组直接展示为“当前支持/当前反证”，与用户确认的 Anchor 级推导结论语义冲突
- 后续：本任务验收后才能实施 TW-07 小程序最终页面

## 用途

“当前支持”和“当前反证”是 AI 分析师对一棵 Research Anchor 的证据整体进行推导后形成的结论性描述，不是具体 Event 清单，也不能由 Data、Miniapp BFF 或前端按 `evidence_role` 拼接生成。

Event 关联继续承担事实追溯：它保存具体 Event、证据角色和该 Event 对 Anchor 的 `evidence_summary`。Anchor 汇总字段回答整体证据目前支持什么、反驳什么；两者不是重复事实。

## 冻结字段

Research Anchor Import V1 的每个 Anchor 新增：

```json
{
  "support_summary": "军事行动与供应警告共同抬高断供尾部风险，但尚未形成实体断供证据",
  "counter_summary": "原油价格走弱且外交对话仍在继续，当前不能确认上涨行情"
}
```

- `support_summary`：必填、非空白，由 AI 分析师直接生成。
- `counter_summary`：当 Anchor 存在 `contradicting` Event 时必填、非空白；没有 `contradicting` Event 时必须为 `null`。
- 发布器原样传递，不拼接、不改写、不从 Event 列表推断。
- Tidewise 只校验字段结构、非空规则及其与 `contradicting` Event 是否存在的一致性。
- `events[]`、`evidence_role` 和 `evidence_summary` 保持不变，仍是具体证据追溯的唯一事实。

## 数据库

`research_anchors` 增加：

- `support_summary TEXT NOT NULL`，带非空白约束。
- `counter_summary TEXT NULL`，非空时带非空白约束。

跨表规则“是否存在 contradicting Event”由 Import application service 在事务写入前校验；数据库不使用跨表 CHECK 或触发器推断该语义。

该能力尚未正式发布，迁移不得根据旧 `evidence_summary` 猜测回填 Anchor 汇总。迁移执行前必须断言 `research_anchor_import_receipts`、`research_anchors` 及其关联表为空；非空时停止并要求使用受控本地清理或重新发布流程处理，不静默删除或改写数据。

## HTTP 与读取合同

以下链路一对一增加两个字段，不做语义转换：

```text
AI Anchor V1
  -> Research Anchor Import V1
  -> research_anchors
  -> Data reasoning-tree detail
  -> Miniapp BFF reasoning-tree detail
  -> Miniapp page model
```

单树详情固定返回：

```json
{
  "support_summary": "...",
  "counter_summary": null
}
```

- JSON 始终包含 `counter_summary`；无反证时显式返回 `null`。
- Data 与 BFF 保留单一 `events` 数组，不复制为支持/反证数组。
- 前端不得从 `events[]` 生成、补全或修订两个汇总字段。
- 无反证时页面仍保留“当前反证”卡片，并显示确定性空态文案“当前暂无明确反证”。该文案是界面状态，不写回研究事实。

## 幂等与兼容

- 端点路径与 Research Anchor Import V1 名称保持不变，不创建 V2。
- 新字段属于尚未正式投入使用合同的发布前修正；合法请求的 canonical payload 与 hash 将包含新字段。
- 迁移前必须无既有 Anchor receipt，避免同一 Theme 的旧 payload hash 与新合同并存。
- Theme Import V1、Theme 数据、Event、Chain Node 与 Theme 首页查询不变。

## 实现范围

- forward migration 与 migration tests。
- Anchor domain、strict JSON、canonical hash、Import validation 和 PostgreSQL repository。
- Data 单树读取 DTO、repository/application/transport。
- Miniapp BFF Data client、DTO、mapping 和 HTTP contract。
- 共享 fixture 的有反证、无反证、多 Anchor 样例。
- 相应 domain、repository、application、transport、drift 和回归测试。
- 向 AI 分析师侧同步冻结请求合同；不修改分析师仓库。

## 范围外

- TW-07 页面视觉实现。
- Theme Import V1、Anchor Event 角色或 Event 主数据。
- 从旧数据自动生成支持或反证汇总。
- H5、Neo4j、指数、人工审核和长期 Research Thesis。

## 验收

1. 缺失或空白 `support_summary` 时整批拒绝。
2. 存在 `contradicting` Event 但 `counter_summary` 缺失、为空或 `null` 时整批拒绝。
3. 不存在 `contradicting` Event 但提交非空 `counter_summary` 时整批拒绝。
4. 合法有反证和无反证 fixture 均可导入、重放并保持 payload hash 稳定。
5. 任一 Anchor 无效时 receipt、Anchor、Event association 和 path node 零部分写入。
6. Data 与 BFF 原样返回两个字段和单一 Event 数组。
7. Theme Import V1、现有 Theme API 和推理树错误合同回归通过。
