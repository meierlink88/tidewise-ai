# Event Publication V3

## 状态与权威

- 状态：Accepted
- 所属上下文：Data
- 上位决议：GitHub Issue #168
- 唯一写入口：`POST /api/data/v1/reviewed-event-imports`
- 被替代合同：Event Publication V2

V3 保留 V2 的同步原子发布、正式 Artifact 血缘、Tag、Review、Receipt、自然身份、
事务和鉴权边界，只替换 Evidence 的主次与文本语义合同。

## Evidence 合同

每个 Event 至少引用一个当前请求中声明或 Data 已正式接纳的 Artifact。每个 Link 包含：

- `artifact_id`
- `evidence_relation`: `supports | contradicts | context`
- `evidence_statement`: 非空自然语言证据陈述
- `supports_fields`: `title | factual_summary | occurred_at | fact_payload`
- `source_level`: `primary | secondary`

`evidence_statement` 由 Event Fact 模型生成并由独立 Review 审核，表达 Artifact 对该
Event 的支持内容，不承诺是原文连续逐字摘录。Data 只校验结构和 Artifact 正式引用，
不读取 Artifact 正文，也不通过字符串包含、日期文本或 Mention 匹配裁决语义。Data 对
该陈述计算 `evidence_hash`，而 Artifact 的 `content_sha256` 继续保护原始文档身份。

所有 Event Evidence Link 具有平等的正式血缘地位。V3 删除 `is_primary`，Event 删除
`primary_source_id`；同一 Event 可在不同发布批次持续追加多个来源的 Evidence，不因
主证据选择不同发生冲突。`source_level` 只表达来源性质，不能解释为 Event 内优先级。

同一 `(event_id, raw_document_id)` 只能有一条 Link。重复提交时，关系、证据陈述、
支持字段和来源层级必须一致，否则返回自然身份内容冲突。

## Breaking rollout 与迁移

Provider/Consumer、OpenAPI、fixture 和数据库在同一发布窗口切换至 V3，不提供长期双写
或旧请求兼容。forward migration：

1. 保留全部 Event、Raw Document 和 Event Evidence 行；
2. 将 V2 `evidence_excerpt` 原位重命名为 `evidence_statement`；
3. 将现有 V2 Event Evidence 行升级为 contract version 3；
4. 删除主 Evidence 唯一索引、`event_sources.is_primary`、
   `events.primary_source_id` 及相应外键；
5. 继续以 `(event_id, raw_document_id)` 保证 Link 唯一。

回滚不得恢复一个无法从业务事实确定的主 Evidence，因此本迁移为 forward-only。

## 保留的 V2 安全边界

以下合同不变：严格 JSON/未知字段拒绝、UUID/哈希/枚举/长度检查、Artifact 成员关系、
Tag Catalog、Review、confirmed+verified 正式状态、批次原子性、Receipt、权限、幂等与
自然身份冲突处理。模型不能直接写 PostgreSQL 或调用 Data 发布副作用。
