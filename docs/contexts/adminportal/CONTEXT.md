# Admin Portal Context

## Purpose

Admin Portal 是管理产品，由 Admin Portal Frontend 和 Admin Application Backend Service
组成。当前通过采集中心提供 Data-owned Event、Atomic Evidence 和 Source 的管理查询入口。

## Dependency Rule

Admin Portal Frontend 只能调用 Admin Application Backend Service；Admin Application
Backend Service 只通过 Data 的版本化 REST API 读取事实，不访问下游数据库。

## Language

**事件中心（Event Center）**：
面向管理员查询 Data 已接纳的正式 Event。列表提供查询、分页、加载、错误与空状态，
并通过列表行右侧详情展示 Event summary、lifecycle 与完整业务命题 semantic
（actors、action、objects、stage、modality、time、jurisdictions、reason、method、metrics）；
其中 time 统一承载 occurrence、announcement、effectiveness 时间及其 precision；
不承载 Event 写入、采集执行、调度或配置控制面。
_Avoid_: 浏览器直连 Data、在 Admin 保存 Data 事实、把列表扩展成外部 Agent 控制台

**证据中心（Evidence Center）**：
以 Data-owned Atomic Evidence 为分页单位，关联其唯一 Raw Evidence 的标题、信源快照、时间与完整
Content Category 集合，构成 Admin 专用只读列表。列表行右侧详情继续展示 Atomic Evidence
semantic，以及 Raw Evidence 的来源地址、原创/转载归因、条件引用信源、Keywords 与完整分类说明；
列表、详情与筛选可使用 Raw Evidence 发布时的 `source_id` 信源快照，且不将其解析为当前 Source；
同时保留“原始文章”公开来源链接，并为符合受控 MinIO 对象路径的 Raw Evidence 提供独立的
“采集文档”打开入口。历史正文不返回浏览器且显示为暂无采集文档；除上述 `source_id` 外，不展示
完整正文、Evidence/Raw Evidence 技术身份、哈希、内部对象路径或内部生成/入库时间。
_Avoid_: 独立 Raw Evidence Tab、父子树、在 Admin 重新定义 Evidence 或 Category 事实

**信源管理（Source Management）**：
对 Data-owned Source 的有界只读管理投影，只展示名称/编码、归属、渠道、启用状态、优先级、
默认信源等级和更新时间。
_Avoid_: Adapter、Endpoint、app_key、provider config、在当前 Admin 页面创建或修改 Source

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
