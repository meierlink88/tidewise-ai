# Admin Portal Context

## Purpose

Admin Portal 是跨系统管理产品，由 Admin Portal Frontend 和 Admin Application Backend Service 组成。

## Dependency Rule

Admin Portal Frontend 只能调用 Admin Application Backend Service。Admin Application Backend Service 通过 REST API 调用 Data 以及未来 User、Payment 等 Domain Service。

## Application Backend Service Owns

- Admin 对外 API、管理员认证和前端专用 DTO。
- 跨 Domain Service 的管理编排、错误转换和审计入口。
- Admin 专用权限表达和页面查询 contract。

## Does Not Own

- Data、Miniapp、User 或 Payment 的数据库与 repository。
- 被管理领域的事实数据和领域规则。
- 已迁移到外部 agent-run 的采集调度能力。

Admin 当前可以没有独立业务数据库。未来确需 Admin-owned 审计或管理数据时，必须明确其数据 owner 和 API 边界。
