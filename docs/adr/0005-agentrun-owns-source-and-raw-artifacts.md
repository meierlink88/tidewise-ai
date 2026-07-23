---
status: accepted
---

# AgentRun 拥有 Source 与原始 Artifact，Data 只接纳 Event 证据

## 背景

历史 Tidewise Data 同时维护 Source Catalog、采集调度表、完整 Raw Document
正文以及 Event 导入。这使采集控制面分散在 Tidewise 与 AgentRun 两处，也让
`raw_documents` 同时承担原始语料仓库和正式 Event 证据两种职责。

AgentRun 已成为完整采集与 Event 提取系统：它拥有 Source、调度、采集执行、
完整 Markdown Artifact、提取执行和失败恢复。继续由 Data 保存同一份原文或
维护并行 Source Catalog 会制造双重事实来源。

## 决策

- AgentRun 独占 Source 主数据、采集控制面、执行面和完整原始 Artifact。
- Tidewise Data 不维护 Source Catalog，不访问 AgentRun 数据库或 Artifact，
  也不验证原始文档真实性。
- `raw_documents` 表名继续保留，但新语义是“被正式 Event 引用并由 Data
  接纳的轻量证据文档记录”。
- Data 只在同步 Event Publication V2 中创建或复用 `raw_documents`；没有
  产生正式 Event 的 Artifact 永不进入 Data。
- 新证据记录保存 AgentRun Artifact 身份、内容 SHA-256、无外键
  `source_ref`、来源快照和必要时间元数据，不保存正文或 Artifact URI。
- AgentRun 通过 `POST /internal/data/v2/reviewed-event-imports` 原子发布一至
  十个 `confirmed + verified` Event。V2 不提供显式幂等键、payload hash、
  重放响应或状态查询。
- Event、Artifact 和关联通过自然身份收敛；相同身份内容不一致时整批冲突。
  每次成功调用仍创建不可变审计 Receipt。
- Source Catalog、旧采集调度/运行表、旧 Source 查询接口、独立 Raw Document
  导入及单 Event V1 导入退出 Data。

## 一致性边界

- 一个 Publication Batch 使用一个 PostgreSQL transaction。
- 任一 Event、Evidence、Tag、Review、引用或冲突校验失败时整批回滚。
- 相同 Event Dedupe Key 的核心事实不可变；后续只允许追加语义一致的新证据
  或 Tag。
- 相同 Artifact ID 的轻量证据字段不可变。
- AgentRun 负责调用失败后的修正与重试；Data 不保存失败任务或恢复状态。

## 数据兼容

- forward migration 保留所有历史 Event、`raw_documents` 正文和
  `event_sources` 关联。
- 历史 `content_text` 继续可读；V2 新记录不写正文。
- 旧 Source 与采集控制面表、旧 Import Receipt 表及其专属代码被物理移除。
- 新约束通过 V2 身份或合同版本区分历史行，不能要求历史记录补齐 AgentRun
  Artifact 身份。

## 取舍

选择该边界后，Data 无法独立读取原始文档或重新运行 Event 提取；这些能力属于
AgentRun。作为交换，Source、Artifact 和执行血缘只有一个权威来源，Data 的职责
收敛为正式事实接纳、事务一致性、审计与查询。

不采用以下方案：

- Data 与 AgentRun 各自维护 Source Catalog。
- Data 再保存一份完整 Markdown。
- Data 保存 AgentRun 本机路径或 Artifact URI 并主动读取原文。
- 用 `package_id`、请求哈希或 Receipt 实现第二套显式幂等协议。

## 与既有决策的关系

本 ADR 取代 ADR-0002 中“Tidewise 保留 Source Catalog”的局部决策，并明确
`unified-data-collection-center-v1.md` 的 Proposed Data-owned 采集中心设计已
被废弃。ADR-0002 的 Application Backend Service、Domain Service 和 REST
边界继续有效。
