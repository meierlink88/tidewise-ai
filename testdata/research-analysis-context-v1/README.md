# Corrected Research Analysis Context V1 shared fixtures

这些文件冻结 Data Service 与工程外 Codex 分析师之间的 provider-consumer 合同，不是
生产 seed，也不会写入数据库。

- `analysis-context-page.json` 与 `analysis-context-page-2.json`：两个连续 Event 页；
  第一页包含完整 Event Bundle 及其最小引用闭包，第二页冻结 cursor 终止语义。闭包只包含
  本页正式语义引用的 Entity、Relation、Variable、Rule、EntityType 和 Acceptance
  Policy。
- `research-graph-search-request.json`：Codex 从 Event seed Entity 发起的显式、只读、
  有深度和 node/edge budget 的图谱检索。
- `research-graph-search-result.json`：引用完整的确定性一跳子图。
- `inconsistent-error.json`：实时页级引用漂移时必须从第一页重查的冲突合同。
- `resource-limit-error.json`：查询超限时的结构化、无内部实现信息的失败合同。

消费者必须逐页读取完整 Event Bundle，以稳定 ID/显式版本合并页级闭包。实时查询若
返回 `409 RESEARCH_ANALYSIS_CONTEXT_INCONSISTENT`，应丢弃本轮结果并从第一页重查。
`429` 表示整次查询失败，消费者应按 `retry_guidance` 缩小技术查询范围，不能把它当成
可发布的部分输入。Codex 需要更多背景关系时调用 Research Graph Search；不能直连
PostgreSQL/Neo4j，也不能把页级闭包误当成完整研究图谱。

Analysis Context 尚未正式发布，因此页级闭包语义直接修正
`research-analysis-context.v1`，不存在 v2 兼容层。资源错误只有在 Data 完成计数或
测量时返回 `actual_rows`/`actual_bytes`；bounded traversal 仅探测到超限时省略未知
实际总数。
