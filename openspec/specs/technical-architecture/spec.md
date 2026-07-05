## Purpose

定义观潮家源码工程的总体技术架构、前后端边界、Go API/BFF 边界、外部 Agent 平台集成边界、数据与异步基础设施边界、环境配置和部署边界，作为后续工程 change 的当前系统事实。

## Requirements

### Requirement: 分层技术架构
系统 SHALL 定义分层技术架构，将 Taro 跨平台小程序展示、Go API/BFF 聚合、服务端领域能力、外部 Agent 平台集成、数据采集、图谱/RAG 数据边界、存储、异步任务和部署职责分离。

#### Scenario: 审阅架构分层
- **WHEN** 开发者规划新的工程 change
- **THEN** 该 change 可以映射到已定义的 frontend、backend API/BFF、领域模块、Agent 平台集成、采集、图谱/RAG 数据边界、存储、异步任务或部署层

#### Scenario: 引入跨层能力
- **WHEN** 拟议能力跨越多个架构层
- **THEN** 对应 OpenSpec design 必须说明受影响的层以及层与层之间的边界

### Requirement: Monorepo 边界
系统 SHALL 使用 `tidewise-ai` 作为源码工程根目录和 OpenSpec 根目录，并在 repo 内明确 `frontend`、`backend`、`infra`、`openspec` 和 `.codex` 的边界。

#### Scenario: 引入源码区域
- **WHEN** 未来 change 创建前端、后端、API 契约、数据访问、外部集成、异步任务或基础设施文件
- **THEN** 文件必须放在对应的 `frontend`、`backend` 或 `infra` 区域，而不是创建平行工程结构

#### Scenario: 引用产品文档和原型
- **WHEN** 未来 change 需要产品文档或原型材料
- **THEN** 必须将 `doc` 和 `prototype` 作为参考输入，除非该 change 明确把文档或原型更新纳入范围

### Requirement: 小程序客户端边界
系统 SHALL 让 Taro 跨平台小程序只负责展示、交互、轻客户端状态，以及通过 service 边界发起 API 调用。

#### Scenario: 客户端需要市场智能能力
- **WHEN** Taro 小程序需要事件解释、AI 分析、报告生成、订阅状态、支付状态或图谱数据
- **THEN** 小程序必须调用 API/service 边界，而不是在客户端嵌入后端执行逻辑

#### Scenario: 请求敏感能力
- **WHEN** 需要模型访问、支付处理、数据库访问、RAG 检索、Agent 编排、图谱持久化或事件采集
- **THEN** 该能力必须在服务端实现，不能出现在小程序源码中

### Requirement: Go API/BFF 服务端边界
系统 SHALL 通过 Go + Gin API/BFF 边界暴露面向客户端的能力，隐藏内部模块、外部 Agent 平台和数据访问细节，并向客户端应用提供稳定契约。

#### Scenario: 小程序集成后端数据
- **WHEN** Taro 小程序页面用真实数据替换 mock 数据
- **THEN** 页面必须依赖已文档化的 API/BFF 契约，而不是直接调用 Agent 平台、采集、图谱、RAG、数据库、队列或第三方模型服务

#### Scenario: 内部服务拓扑变化
- **WHEN** 内部能力在模块化单体代码、worker 代码或独立服务之间迁移
- **THEN** 面向客户端的 API/BFF 契约仍然是稳定集成边界，除非另一个 OpenSpec change 明确修改该契约

### Requirement: 外部 Agent 平台集成边界
系统 SHALL 将 AI 推理、Agent 工作流、RAG 检索、Prompt 编排、工具调用和模型调用交由外部 Agent 平台承载，并在本工程后端保留调用、回调、校验、落库和展示接口边界。

#### Scenario: 生成 AI 或市场分析
- **WHEN** 系统生成 AI 辅助分析、市场解读、事件影响摘要或报告
- **THEN** 输出必须通过后端 Agent 平台集成边界获取或接收，并定位为决策辅助信息，而不是直接投资建议

#### Scenario: 需要 Agent 平台凭证或回写接口
- **WHEN** Agent 平台调用需要 API key、工作流参数、回调地址或结构化结果回写
- **THEN** 这些凭证、回调处理和结构化校验必须保留在 Go 后端，不得出现在前端源码中

### Requirement: 契约和共享类型边界
系统 SHALL 在用真实 Go 后端集成替换 mock-first 客户端服务前，定义跨语言 API 契约。

#### Scenario: 引入共享领域结构
- **WHEN** 未来 changes 定义事件、市场、板块、图谱节点、报告、订阅、AI 消息、错误、分页或 Agent 回写结构
- **THEN** 这些结构必须放在 API 契约边界中，并能服务于 Taro 前端、Go 后端和 Agent 平台回调

#### Scenario: 替换前端 mock 数据
- **WHEN** mock-first services 被真实 API 调用替换
- **THEN** 替换必须遵循共享 API 契约，而不是让页面代码依赖后端实现细节

### Requirement: 数据和异步基础设施边界
系统 SHALL 将结构化存储、图存储、向量存储、缓存/会话状态、队列、定时任务，以及长时间运行的报告或采集任务视为服务端基础设施边界。

#### Scenario: 设计持久化数据能力
- **WHEN** 未来 change 引入持久化事件、指标、实体、用户、订阅、报告、图关系、嵌入、缓存或任务状态
- **THEN** 该 change 必须识别相关存储或异步边界，并确保前端源码不包含直接访问逻辑

#### Scenario: 需要长时间处理
- **WHEN** 事件采集、嵌入生成、图谱更新、报告生成、通知投递或 AI 推理无法作为简单客户端请求完成
- **THEN** 该能力必须设计为服务端任务、worker、队列或定时流程

### Requirement: 后端环境配置边界
系统 SHALL 标准化支持 `local`、`uat`、`prod` 三类后端运行环境，并通过统一的 Go 强类型配置加载机制隔离环境差异。

#### Scenario: 加载环境配置
- **WHEN** Go 后端服务启动
- **THEN** 服务必须根据 `APP_ENV` 加载对应环境配置，并在启动阶段校验必填配置

#### Scenario: 管理非敏感配置
- **WHEN** 后端需要配置服务端口、日志级别、外部接口 base URL、数据库 host/port/name、Redis 地址、限流参数或回调路径
- **THEN** 这些非敏感配置可以放入 `backend/config` 下的环境配置文件或示例模板

#### Scenario: 注入敏感配置
- **WHEN** 后端需要数据库密码、Agent 平台 API key、支付密钥、JWT secret 或云厂商密钥
- **THEN** 这些敏感配置必须通过环境变量或部署平台 secret 注入，并且不得提交到 repo

#### Scenario: 使用配置
- **WHEN** 业务模块需要访问环境相关配置
- **THEN** 业务代码必须依赖统一 config 对象，而不是散落读取环境变量或硬编码 local/uat/prod 分支

### Requirement: 部署边界
系统 SHALL 将 Taro 小程序发布与 Go API/BFF、worker、Agent 平台集成、采集、图谱、RAG 数据边界和基础设施部署职责分离。

#### Scenario: 发布前端
- **WHEN** Taro 小程序前端准备发布
- **THEN** 它必须分别构建并发布到微信、抖音等小程序平台，并且不包含后端部署 artifacts 或密钥

#### Scenario: 部署服务端能力
- **WHEN** Go API/BFF、Agent 平台集成、采集、图谱、RAG 数据处理、支付回调、订阅推送或任务处理能力准备部署
- **THEN** 该能力必须作为服务端运行时基础设施部署，并独立于小程序发布
