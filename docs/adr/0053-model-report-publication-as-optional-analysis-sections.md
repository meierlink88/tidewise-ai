---
status: accepted
date: 2026-09-03
issue: 369
supersedes_in_part: 0052-replace-research-theme-with-report-publications.md
---

# 将 Report 发布建模为可选上层分析与必选产业链集合

## 背景

旧合同复制了过时原型：强制四层、持久化首页卡片、携带 AgentOS 工作流审计元数据，并把
结论依据和验证状态合并。它也要求读取端把完整 JSON 解码后切片，不适合一份报告包含 50 条以上
产业链的首页。

定稿领域边界是不可变中文分析快照。除 Atomic Evidence ID 外，层、锚点、产业链、节点和图边
全部是 Report-local 内容，不能关联 Event、Signal、Graphiti 或正式图主数据后动态重建。

## 决策

- AgentOS 定稿发布 fixture 是字段形状的最高验收基线；Data OpenAPI、领域类型、存储投影
  与 Miniapp 消费者必须保持一致。
- 发布请求严格为 `{publisher_report_id, report}`。`publisher_report_id` 是 AgentOS 对一次报告
  Artifact 的全局唯一、重试稳定身份；同身份同 canonical report 安全重放，不同 report 冲突。
- Report 根严格包含 `report_type/generated_at/timezone/industry_chains`，并可选包含
  `geopolitics/macroeconomics`。公司层在出现真实定稿基线前不进入合同。
- 上层 Section 按 AgentOS 契约扁平发布结论、结果、置信度、窗口、锚点、推理步骤、
  uncertainty、有序 Evidence refs 和分组向下传导；不在发布 JSON 中人为嵌套 summary/detail。
- 每条产业链也按 AgentOS 契约扁平发布结论、结果、窗口、置信度、可空路径摘要、
  可空已接受假设摘要、节点、边、uncertainty 和有序 Evidence refs。
- 结果、置信度、时间窗口、结论依据、验证状态、Evidence role 与传导类型由 AgentOS 同时发布稳定 code
  与中文 label。Data 校验已知结构映射但不生成中文；Miniapp Backend 原样透传；Frontend 按 code
  选择样式、按 label 展示，未知 code 使用中性样式。
- `conclusion_basis` 与 `validation_status` 是独立维度。只有
  `conclusion_basis.code=direct_evidence` 的锚点或节点允许并必须带 Evidence；推理假设不能获得
  “依据”入口。
- 传导按 AgentOS 的 `to_macroeconomics/to_industry_chains` 分组，保留 source conclusion、
  闭包 targets、logic、kind、confidence 与 status；不携带 Evidence。
- `reports.report` 保存完整不可变 JSONB，`content_hash` 只用于服务端幂等且不暴露 API。
  `report_evidence_links` 只保存 `id/report_id/evidence_id/scope_type/scope_path/position`；scope path
  由 Data 从 JSON 位置生成。
- Data 读取端只暴露 report-bound opaque `RPE` scope token，不暴露 Evidence ID、JSON path 或
  scope type/key。层、推理步骤与链结论 Evidence 是显式作用域，不能从子节点聚合。
- Data 在 PostgreSQL 用 JSON array ordinality 做 report-bound cursor 分页；BFF 转发分页并生成
  产品 DTO，Frontend 在固定内部 ScrollView 中每次追加 20 张卡片并去重、隔离过期响应。
- Migration 000081 只允许空 Report store 前向切换：移除 `contract_version`，重命名
  `content` 为 `report`，并收敛 Evidence link 字段。发现已有 Report 或 link 时 fail closed。

## 影响

AgentOS 只在仓库外增加确定性的 Publish Report step，不使用 LLM 再推理。部署顺序为 migration/
Data Service、Miniapp Backend、Miniapp Frontend，最后再启用 AgentOS publisher。不存在旧合同双读、
双写或非空历史数据转换；版本不匹配应显式失败。
