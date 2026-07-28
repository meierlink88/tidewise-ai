# Research Theme + Reason Tree V1 shared fixtures

这些文件是 Data Service、Miniapp BFF 与小程序共用的确定性合同测试数据，不是生产
seed 或分析师运行产物。

- `00-theme-import-request.json`：分析侧先发布 Theme batch 的 V1 请求。
- `00-reasoning-tree-import-request.json`：随后为该确定性 Theme ID 发布完整 Reason Tree
  set 的 V1 请求。
- `01-reasoning-tree-list-result.json`：Theme 与完整 Tree Tab 列表读取结果。
- `02-reasoning-tree-with-contradiction-result.json`：包含三个有序产业链节点、正式 Graph
  Edge、Variable Signal 展示快照和反证 Event 的单树详情。
- `04-theme-without-reasoning-trees-error.json`：Theme 已发布但 Tree 尚未发布时的稳定错误。

数组顺序就是合同顺序，不允许消费者静默重排。Tree 节点是否属于 Theme Impact 由
`chain_node_entity_id` 与 `impact_node_ids` 的交集决定；fixture 不包含 subject、
primary impact 或结果节点标记。

`00`、`01`、`02` 是同一条发布/读取 lineage：Theme、Tree 和 Node ID 必须由 V1
identity 函数确定性生成；同一 Tree set 的 `published_at` 必须来自同一 receipt。
