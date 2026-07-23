# Research Anchor Import V1 shared fixtures

这些文件是 Tidewise Data、Miniapp BFF 和小程序共用的确定性合同测试数据，不是 AI 分析师运行产物、生产 seed 或导入 outbox。

真实推理树只能由 AI 分析师发布器调用 `POST /api/data/v1/research-anchor-imports` 入库。

## Files

- `prerequisites.json`：已发布 Theme、Theme Event、Theme Chain Node 和路径辅助节点前置事实。
- `01-multi-anchor-import-request.json`：一个 Theme 下两棵完整 Anchor 推理树的合法导入请求。
- `01-multi-anchor-import-result.json`：首次成功导入结果；`replayed` 在重放响应中变为 `true`，其他字段不变。
- `01-reasoning-tree-list-result.json`：页面 Tab 列表读取结果。
- `02-reasoning-tree-with-contradiction-result.json`：包含反证的单树详情。
- `03-reasoning-tree-without-contradiction-unquantified-result.json`：没有反证且明确未量化的单树详情。
- `04-theme-without-reasoning-trees-error.json`：Theme 已存在但尚无成功 Anchor 回执时的读取错误。

## Stable identities

- Anchor namespace: `UUIDv5(DNS, "tidewise.ai/research-anchor-publication/v1")`
- Frozen namespace UUID: `f219ded4-fc65-5948-9e28-c1cdb6a8288e`
- Anchor name bytes: lowercase `theme_id + NUL + center_chain_node_id`

数组顺序必须保持合同顺序。测试不得为了通过校验而在读取后静默重排 fixture。

Data API 测试读取完整响应文件；Miniapp BFF 与小程序测试复用同一文件中的 `.result`，因为 BFF 成功响应不包含 Data 的 `request_id/result` 外壳。
