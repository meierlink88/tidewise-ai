# Admin Portal Context

## Purpose

Admin Portal 是管理产品，由 Admin Portal Frontend 和 Admin Application Backend Service
组成。当前只提供 Data-owned Event 的管理查询入口。

## Dependency Rule

Admin Portal Frontend 只能调用 Admin Application Backend Service；Admin Application
Backend Service 只通过 Data 的版本化 REST API 读取事实，不访问下游数据库。

## Language

**事件中心（Event Center）**：
面向管理员查询 Data 已接纳的正式 Event。列表提供查询、分页、加载、错误与空状态，
不承载 Event 写入、采集执行、调度或配置控制面。
_Avoid_: 浏览器直连 Data、在 Admin 保存 Data 事实、把列表扩展成外部 Agent 控制台

**运行时健康（Runtime Health）**：
Admin Backend 对 Data Service 的有界健康探测。浏览器只看到安全的状态、检查时间、延迟
和受控原因码，不暴露下游地址、令牌或错误正文。
_Avoid_: 把共享基础设施健康混入应用所有权、泄漏连接信息

## Application Backend Service Owns

- Admin 对外 API、管理员认证和前端专用 DTO。
- Data API 的前端适配、错误转换与安全响应。
- Admin 专用页面查询 contract。

## Does Not Own

- Data、Miniapp 或未来领域服务的数据库与 repository。
- 采集调度、执行历史、模型、Connector 或外部 Agent 配置。
- Data 的事实数据和领域规则。

Admin 当前没有独立业务数据库。未来确需 Admin-owned 审计或管理数据时，必须先明确数据
owner 和 API 边界。

## Runtime

Admin Backend 与 Admin Frontend 只通过各自 Docker image 和 Compose 运行。浏览器只使用
相对 `/api/admin/*`；Frontend container 不获得下游 Service Token 或数据库凭据。
