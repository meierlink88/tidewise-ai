---
status: superseded
date: 2026-08-16
supersedes_in_part: 0011-data-owns-raw-evidence-and-evidence-publication.md
superseded_by: 0049-model-atomic-evidence-as-business-proposition.md
issue: 255
---

# 收敛 Atomic Evidence 为摘要与单层 5W1H 语义

## 背景

ADR-0011 建立 Data 对 Raw Evidence 与 Atomic Evidence 的 ownership，但首版持久化和发布
合同同时包含拆分顺序、单层/双层类型、两组 5W1H 列，以及 Expression Fingerprint、Key
和 Version。该结构让发布方和 Data 的 API、Service、Biz、Data 各层都承担同一组历史表达
细节，并把原子事实语义拆散到多列。

## 决策

- Atomic Evidence 只使用独立的非空 `summary` 和非空 `semantic JSONB` 表达内容。
- `semantic` 严格包含一层 `who`、`what`、`when`、`where`、`why`、`how`，不包含
  `summary` 或其他键。`what` 是非空字符串，其余五项必须存在并且只能是非空字符串或
  `null`。
- 删除 `split_order`、`layer_type`、Expression Key/Version，以及所有第一层 Source 与
  Source Core 字段。`is_split` 继续由 Data 根据一次完整发布的基数派生。
- Atomic Evidence 是无业务顺序的完整集合。Data 以正式 ID 排序保证读取和响应确定性，
  该顺序不表示原文位置或事实优先级。
- Data 根据 Raw Evidence ID 与规范的 `summary + semantic` 内容派生正式 Evidence ID，
  只用于同一 Raw Evidence 的安全重试；不提供跨来源语义去重或 Evidence Group。
- Issue #255 明确授权零兼容切换并删除全部历史 Atomic Evidence，不将旧双层字段猜测转换
  为单层 JSON。Raw Evidence 保留。

## 发布与回滚

发布前停止 Evidence 写入并确认 PostgreSQL 恢复点；使用候选 Data 镜像应用 forward-only
migration 后，发布匹配的新 Data Service 与发布方。旧应用和旧发布 payload 与新 schema
不兼容，不提供 mixed-version 窗口。回滚必须同时恢复迁移前数据库快照与上一版应用，不运行
down migration 假装恢复已删除事实。

## 影响

Data Context、OpenAPI、wire DTO、Biz 不变量、Service 映射、Data Adapter、fixture 和测试都
以 `summary + semantic` 为唯一当前合同。ADR-0011 的 Data ownership、两阶段 Publication、
完整 `1..N` 集合和 Raw Evidence/Event Evidence 隔离决定继续有效。
