# Admin Portal Context

## Purpose

Admin Portal 是跨系统管理产品，由 Admin Portal Frontend 和 Admin Application Backend Service 组成。

## Dependency Rule

Admin Portal Frontend 只能调用 Admin Application Backend Service。Admin Application Backend Service 通过 REST API 调用 Data、AgentRun 以及未来 User、Payment 等服务。

## Language

**AgentRun 管理代理（AgentRun Management Facade）**：
Admin Portal 面向管理员提供 AgentRun 管理入口，并把管理请求转交给 AgentRun。Schedule、Execution、Model Provider Configuration 和 Connector Configuration 的事实与规则仍由 AgentRun 拥有；浏览器不直接调用 AgentRun。
_Avoid_: Admin Portal 自建采集调度、复制 AgentRun 配置事实、浏览器直连 AgentRun

**采集 Agent 管理（Collector Agent Management）**：
Admin Portal 第一阶段面向 Collector Agent 提供的管理视图，覆盖其 Schedule、Execution 历史及运行所需配置。它是通用 Agent 管理能力的首个产品入口，不代表 AgentRun 只能管理 Collector。
_Avoid_: 把 Collector 专属字段固化为所有 Agent 的通用合同、把 Collector Schedule 配置扩展成所有 Agent 的通用控制面

**Agent 状态监控（Agent Status Monitor）**：
Admin Portal 对 AgentRun 全部已注册 Agent 的只读当前状态视图，展示名称、当前 Version、
是否工作中、当前 Execution Status 与更新时间；没有在途 Execution 时显示 `idle`。
AgentRun 是事实 Owner；Admin Portal 不允许从监控页接受/拒绝候选、重放任务或修改 Agent
状态，也不显示执行详情、Prompt、自由推理、Evidence/Artifact 正文、Connector 响应或凭据。
_Avoid_: 人工审核工作台、浏览器直连 AgentRun、Admin 自建监控状态、执行详情页

**监控中心（Execution Monitoring Center）**：
Admin Portal 对事件采集、Event 提取和事件语义三类执行对象的只读运行投影。
它把 AgentRun 已有原始状态确定性分组为成功、执行中和失败，同时保留原始枚举；
摘要只统计成功执行的安全业务产出。监控事实、时间窗口和状态分组由 AgentRun 拥有，
Admin Application Backend Service 只通过 AgentRun Admin API 代理，不读取任何下游数据库。
监控首页同时聚合 Data Service、AgentRun、Qdrant、Neo4j 四项运行健康状态与 Agent 当前状态；
Data 和 AgentRun 各自拥有依赖探测，Admin BFF 只做有界并发聚合。单个 Provider 失败时仍返回
安全的部分结果，不把下游错误文本或凭据暴露给浏览器。三类执行明细只在独立子页面按需加载。
_Avoid_: 浏览器直连 AgentRun、Admin 复制状态分组、展示 Prompt/正文/模型输出、从监控页发起重试或审批

**采集 Agent 定时配置（Collector Agent Schedule Configuration）**：
采集 Agent 唯一综合采集任务的周期触发配置。管理员可以维护其中的 Collection Prompt，并在“多个每日固定时间”或“标准五段式 Cron”两种策略中选择一种执行；Cron 支持分钟级和每小时等重复周期。Prompt 与调度策略的合法性仍由 AgentRun 负责。当前只有 `collector.v1` 生效，Admin Portal 不提供版本选择。配置保存与运行状态相互独立，管理员通过“开始”或“停止”即时改变现有 Schedule 的运行状态。停止只阻止后续触发，配置变化也只影响后续 Execution；两者都不取消在途 Execution。
_Avoid_: 同时启用多种调度策略、把多个 Prompt 建模成多条 Collector Schedule、为尚不存在的多版本能力增加选择流程、删除 Schedule、把编辑配置误当作开始运行、通过管理页立即执行一次、在 Admin Portal 配置调度时区、把停止 Schedule 表述为取消当前执行

**采集 Agent 配置就绪（Collector Configuration Readiness）**：
采集 Schedule 可以开始运行前，所需模型和全部已注册 Connector 均具有完整当前配置。Admin Portal 可以提前展示缺失项并阻止误操作，但 AgentRun 仍是最终校验者。
_Avoid_: 由 Admin Portal 复制 AgentRun 的完整配置规则、绕过 AgentRun 强制开始

**采集执行记录（Collector Execution Record）**：
采集 Agent 一次 Execution 的安全审计摘要，统一通过监控中心的“事件采集”明细子页面查看。
采集器配置不再提供独立执行记录 Tab，Admin BFF 也不保留仅服务该旧 Tab 的专用代理 API。
它不是可重放任务，也不包含 Prompt、采集正文、Artifact 或 Connector 调用内容。
_Avoid_: 在采集器配置复制执行历史、从记录列表读取业务载荷、把列表行当作可重放或可编辑任务

**模型配置视图（Model Provider Configuration View）**：
AgentRun 代码已注册模型供应商的可管理安全视图。它展示 Provider、Base URL、模型、配置状态和脱敏 Key；完整旧 Key 永不返回，管理员只能保留或用新 Key 覆盖。Provider 能力与代码紧密绑定，管理面永远不负责新增或删除。
_Avoid_: 回显完整 Key、把脱敏 Key 当作可再次提交的 Credential、允许清空必需的模型 Key、动态新增或删除 Provider

**连接器配置视图（Connector Configuration View）**：
AgentRun 代码已注册 Connector 的可管理安全视图。它展示 Connector、Base URL、配置状态和脱敏 Key；管理员可以保留、覆盖或明确清除可选 Key，完整旧 Key 永不返回。Connector 能力与代码紧密绑定，管理面永远不负责新增或删除。
_Avoid_: 回显完整 Key、用空输入意外清除 Key、动态新增或删除 Connector、把连接参数误当作能力注册

## Application Backend Service Owns

- Admin 对外 API、管理员认证和前端专用 DTO。
- 跨 Domain Service 的管理编排、错误转换和审计入口。
- AgentRun 管理面的前端适配、权限入口和安全响应转换。
- Admin 专用权限表达和页面查询 contract。

## Does Not Own

- Data、Miniapp、User 或 Payment 的数据库与 repository。
- 被管理领域的事实数据和领域规则。
- 已迁移到 AgentRun 的采集调度、执行历史、模型和 Connector 配置事实。
- AgentRun 拥有的执行监控事实、时间窗口语义和原始状态分组。

Admin 当前可以没有独立业务数据库。未来确需 Admin-owned 审计或管理数据时，必须明确其数据 owner 和 API 边界。

## Runtime

Admin Application Backend 与 Admin Portal Frontend 只通过各自 Docker image 和 Compose
运行。Frontend local/UAT 均使用 unprivileged nginx image；浏览器只使用相对
`/api/admin/*`，由 nginx 转发到 Compose 内部的 `adminportal:9013`。UAT 只公开 Admin Web
`9014`，不公开 Backend `9013`；浏览器和 Frontend container 不获得下游 Service Token
或数据库凭据。
