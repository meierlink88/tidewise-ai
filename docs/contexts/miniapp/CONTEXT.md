# Miniapp Context

## Purpose

Miniapp 是用户产品系统，由 Miniapp Frontend 和 Miniapp Application Backend Service 组成。

## Dependency Rule

Miniapp Frontend 只能调用 Miniapp Application Backend Service。Miniapp Application Backend Service 需要 Data、User 或 Payment 能力时，只能调用对应 Domain Service 的 REST API。

## Application Backend Service Owns

- Miniapp 对外 API、认证入口和前端专用 DTO。
- 多个 Domain Service 的产品编排。
- Data API 错误、分页、时间和字段到 Miniapp contract 的转换。
- Miniapp 专用缓存和降级策略，但不拥有 Data 事实。

## Does Not Own

- Data PostgreSQL、migration、repository、Neo4j 或 Data domain model。
- Entity、Raw Document、Event、Research Theme 和 Research Anchor 的事实数据。
- Admin Portal contract。

## Frontend Mock Policy

真实 API 尚未接入时，可以在 Miniapp Frontend 内保留仍被页面使用的 mock。mock 必须收敛到明确的 `mocks/` 或 `devdata/` 目录，并通过可替换 adapter 注入。未被页面、测试或开发场景引用的 mock、model、service 和 component 应删除。

本次源码治理不把 Miniapp Frontend 接入真实 BFF；该行为变更单独实施。
