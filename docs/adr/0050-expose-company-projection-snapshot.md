---
status: accepted
date: 2026-08-30
issue: 357
amends: 0037-independent-company-persistence.md
extends: 0031-data-owns-source-management.md
---

# 向外部 AgentOS 提供 Company 投影快照

## 背景

ADR-0037 将 Company 和 CompanyIndustryLink 收敛为 Data 拥有的独立事实，但当时明确
不增加 HTTP/OpenAPI。外部 Tidewise AgentOS 需要把这些事实投影到其拥有的
Graphiti/Neo4j；如果直连 Data PostgreSQL，将越过 Context 边界，并无法稳定检测分页
期间的事实漂移。

## 决策

- Data 提供经 service bearer 认证的
  `GET /api/data/v1/entities/companies`，专用 scope 为 `data.companies.read`。
- response `schema_version` 固定为 `company-projection-snapshot.v1`，每页返回
  64 位 lowercase SHA-256 `snapshot_id`、`items` 和可空 `next_cursor`。
- Company 按 `(code, id)` 稳定升序 keyset 分页，默认 50，最大 100。cursor 是有界
  opaque 值，同时绑定 snapshot identity 和上一页边界。
- repository 在同一个 PostgreSQL `REPEATABLE READ + READ ONLY` 事务中，对全部
  Company 与 CompanyIndustryLink 行的 canonical JSONB 表示计算 SHA-256，再读取分页。
  事务显式固定 `UTC` session timezone，避免同一 `timestamptz` 事实在不同实例产生
  不同摘要。
  续页 cursor 的 snapshot 与当前事实不同时返回
  `409 COMPANY_PROJECTION_SNAPSHOT_CHANGED`，不返回部分结果；消费方从首页重启。
- item 原样表达 Company 所有投影字段；可空 scalar 显式为 `null`，数组始终
  为数组，日期为 ISO date，时间戳为 UTC RFC3339。`industry_links` 只包含
  Data 已持久化的 formal CompanyIndustryLink，不包含模型推断。
- 该合同只是 provider 快照读取。Data 不写 Neo4j，不生成公司与 Industry/
  ChainNode 的推断关系，也不接受 Company 写入。外部 AgentOS 必须使用其 typed
  client 严格验证本合同，不得读取 Data PostgreSQL。

当前摘要为每页重新校验全量事实，以不引入额外持久化 revision 的方式优先保证
正确性。初始 13,264 条 Company、100 条每页时，本地全量读取约增加 20–30 秒
摘要开销；数据量显著增长前，应以单调 projection revision 或可分页 manifest 替代每页全量哈希。

## 发布与回滚

先发布 Data provider 与 OpenAPI/provider fixture，使用真实授权请求验证非空首页、
UTC 字段、formal link 数组和 cursor。随后才发布 AgentOS consumer。回滚 consumer
时只停止投影；回滚 provider 时删除路由和 scope 即可，无 PostgreSQL schema 变更，
不删除 Company 或 CompanyIndustryLink 事实。
