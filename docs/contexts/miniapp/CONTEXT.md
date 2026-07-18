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

## Product Language

- **研究主题（Research Theme）**：Data Context 拥有的研究结果事实，是 Miniapp 首页主线内容的数据来源。
- **推理主线**：研究主题在 Miniapp 面向用户展示时使用的产品名称，不是另一种数据实体。
- **主题卡片**：首页列表中呈现一条推理主线的界面单元，不拥有独立于研究主题的业务事实。
- **主题跟踪**：用户选择持续关注某个研究主题的产品行为；“跟踪中”数量是当前用户已跟踪的主题数，不是 Research Theme 的事实属性。

## Frontend Mock Policy

真实 API 尚未接入时，可以在 Miniapp Frontend 内保留仍被页面使用的 mock。mock 必须收敛到明确的 `mocks/` 或 `devdata/` 目录，并通过可替换 adapter 注入。未被页面、测试或开发场景引用的 mock、model、service 和 component 应删除。

本次源码治理不把 Miniapp Frontend 接入真实 BFF；该行为变更单独实施。
