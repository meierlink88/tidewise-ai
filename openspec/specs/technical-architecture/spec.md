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

#### Scenario: 落地后端骨架
- **WHEN** Go API/BFF 后端骨架被创建
- **THEN** 骨架必须位于 `backend/`，并提供可编译、可测试、可本地启动的服务端入口

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

#### Scenario: 验证配置骨架
- **WHEN** 后端骨架提供 local、uat、prod 配置模板
- **THEN** 模板必须保留环境差异结构，同时不得包含真实 secret 或生产凭证

### Requirement: 部署边界
系统 SHALL 将 Taro 小程序发布与 Go API/BFF、worker、Agent 平台集成、采集、图谱、RAG 数据边界和基础设施部署职责分离。

#### Scenario: 发布前端
- **WHEN** Taro 小程序前端准备发布
- **THEN** 它必须分别构建并发布到微信、抖音等小程序平台，并且不包含后端部署 artifacts 或密钥

#### Scenario: 部署服务端能力
- **WHEN** Go API/BFF、Agent 平台集成、采集、图谱、RAG 数据处理、支付回调、订阅推送或任务处理能力准备部署
- **THEN** 该能力必须作为服务端运行时基础设施部署，并独立于小程序发布

### Requirement: MVP 数据基础设施选择
系统 SHALL 将 PostgreSQL 定义为 MVP 阶段结构化主存储，将 Redis 定义为缓存、限流、幂等和短期任务状态基础设施，并将独立图数据库和向量数据库延后到明确需求后的独立 change。

#### Scenario: 规划持久化模块
- **WHEN** 开发者规划需要保存业务事实数据的新 change
- **THEN** 该 change 必须优先基于 PostgreSQL 设计结构化数据边界，并说明是否需要 Redis 短期状态

#### Scenario: 规划图谱或向量能力
- **WHEN** 开发者规划图谱查询、RAG 检索、向量召回或复杂关系推理能力
- **THEN** 该 change 必须说明该能力由 PostgreSQL、外部 Agent 平台或未来独立图/向量基础设施承载

### Requirement: API 契约优先开发
系统 SHALL 在真实前后端集成或 Agent 回写实现前，先定义可审阅的 API 契约作为跨端和跨系统协作边界。

#### Scenario: 开始真实 API 开发
- **WHEN** 后续 change 准备把前端 mock-first service 替换为真实 Go API 调用
- **THEN** 该 change 必须先定义或引用 API 契约，而不是直接让页面或 handler 推断字段结构

#### Scenario: 变更已发布契约
- **WHEN** 后续 change 修改已被前端、后端或 Agent 平台使用的契约
- **THEN** 该 change 必须说明兼容性影响、迁移方式和版本处理

### Requirement: 模块化单体演进边界
系统 SHALL 在 MVP 阶段采用 Go 后端模块化单体，将 HTTP、应用服务、领域模型、数据访问、外部集成和异步任务分层组织，并在容量或团队边界明确后再拆分独立服务。

#### Scenario: 新增服务端模块
- **WHEN** 后续 change 新增事件、报告、订阅、Agent 分析或支付回调等服务端模块
- **THEN** 该模块必须映射到 HTTP、应用服务、领域、repository、integration 或 job 边界，而不是把业务逻辑直接堆叠在入口或 handler 中

#### Scenario: 拆分独立服务
- **WHEN** 后续 change 拟将某个能力从模块化单体拆分为独立服务
- **THEN** 该 change 必须说明部署、契约、数据所有权和回滚影响

### Requirement: 数据采集层架构边界
系统 SHALL 将数据采集层定义为服务端输入层，由 ingestion 编排清洗标准化流程，由 integrations 适配自研爬虫和外部 Agent API，由 jobs 调度采集任务，由 repositories 保存标准化结果。

#### Scenario: 规划采集能力
- **WHEN** 后续 change 规划热点事件、政策事件、市场异动、公告、产业动态或热度信号采集
- **THEN** 该 change 必须说明自研爬虫、外部 Agent API、清洗标准化、入库和后续分析触发之间的边界

#### Scenario: 使用采集数据
- **WHEN** 后续 change 需要把采集结果用于事件流、Agent 分析、报告生成、订阅推送或图谱关系
- **THEN** 该 change 必须使用经过清洗标准化和来源追踪的数据，而不是直接使用原始爬虫结果或外部 Agent 原始响应

### Requirement: 本地数据库和采集 smoke 架构边界
系统 SHALL 将真实数据库建表和真实采集 smoke 归入后端与基础设施边界，不得让前端、小程序或 prototype 参与数据库访问和采集执行。

#### Scenario: 运行真实建表
- **WHEN** 开发者需要创建或更新 PostgreSQL schema
- **THEN** 必须通过后端 migration 命令、API 启动检查或部署流程执行，而不是通过前端、小程序或手工散落 SQL 执行

#### Scenario: 运行真实采集
- **WHEN** 开发者需要验证公开来源采集和入库链路
- **THEN** 必须通过后端 ingestion job 和 repository 边界运行，并保持采集数据只作为原始事实材料

#### Scenario: 保持分析安全边界
- **WHEN** smoke 数据后续被用于事件抽取、Agent 分析或展示
- **THEN** 系统必须继续保持决策辅助定位，不得把采集原文直接表达为投资建议

### Requirement: 参考系统采集通道接入边界
系统 SHALL 将 Vibe-Research、Vibe-Trading 和 Stock 的数据采集实现作为参考输入，提炼通道、凭证、限流、解析和幂等经验，但不得直接复制其生产无关代码、模拟数据或业务推理逻辑。

#### Scenario: 使用 Vibe-Research 参考
- **WHEN** 后续 change 实现 RSS 或 Atom 采集
- **THEN** 可以参考 Vibe-Research 的配置型 RSS 源、并发抓取、时间过滤和缓存思路，但必须落到本工程 Go 后端采集层和 `RAW_DOCUMENT` 模型

#### Scenario: 使用 Vibe-Trading 参考
- **WHEN** 后续 change 实现连接器注册、fallback、限流、RSSHub、Eastmoney、Tushare 或 AKShare 边界
- **THEN** 可以参考 Vibe-Trading 的 loader registry、按 host 限流、RSSHub route 和 SDK 可用性判断，但不得把行情 K 线 loader 直接等同于事件原文采集

#### Scenario: 使用 Stock 参考
- **WHEN** 后续 change 实现 Eastmoney、RSS、网页抓取或本地文件回灌
- **THEN** 可以参考 Stock 的脚本入口、字段映射和文件输出经验，但不得把示例新闻、模拟数据或历史输出文件作为生产采集结果

### Requirement: SDK 通道运行时边界
系统 SHALL 将 Tushare 和 AKShare 视为 SDK 型外部通道，Go 主服务只保留配置、接口、凭证引用和任务边界，真实 SDK 执行由后续独立 worker、sidecar 或内部 HTTP wrapper 承载。

#### Scenario: 定义 SDK 采集源
- **WHEN** 采集源目录中出现 `sdk_tushare` 或 `sdk_akshare`
- **THEN** 系统必须能够表达 provider、connector、parser、授权、凭证引用和状态，但不得要求 Go 主服务直接加载 Python SDK

#### Scenario: 后续接入 SDK worker
- **WHEN** 后续 change 需要真实执行 Tushare 或 AKShare 采集
- **THEN** 必须定义 worker 或 wrapper 的部署、契约、错误处理、限流和凭证注入方式

### Requirement: 图谱和向量延后边界
系统 SHALL 在本阶段只通过 PostgreSQL 保存初始实体、事件、证据和关系数据，并保留未来图数据库或向量数据库投影边界。

#### Scenario: 保存关系数据
- **WHEN** 后续 change 创建实体关系或事件实体关联表
- **THEN** 这些表必须作为未来图谱投影来源，而不是直接要求图数据库存在

#### Scenario: 需要向量或 RAG
- **WHEN** 后续 change 需要向量召回、RAG 检索或 Prompt 编排
- **THEN** 必须通过外部 Agent 平台或独立 OpenSpec change 定义，不得混入采集层实现

### Requirement: Neo4j 图谱投影架构边界
系统 SHALL 将 Neo4j 定义为从 PostgreSQL 权威事实源派生的图谱查询库，用于多跳关系查询、路径分析、图谱推理和后续可视化，而不是替代 PostgreSQL 成为主事实库。

#### Scenario: 规划 Neo4j 图谱能力
- **WHEN** 后续 change 需要使用 Neo4j 保存实体图或事件图
- **THEN** 该 change 必须说明 PostgreSQL 中的事实来源、投影规则、重建方式和 Neo4j 只作为图谱查询视图的边界

#### Scenario: 恢复 Neo4j 数据
- **WHEN** Neo4j 数据损坏、清空或图模型需要调整
- **THEN** 系统必须能够从 PostgreSQL 事实表重新投影恢复图谱数据

#### Scenario: 保持服务端部署边界
- **WHEN** 部署 Neo4j 或图谱投影 worker
- **THEN** 该能力必须作为服务端基础设施和后端运行时能力部署，不得出现在小程序或管理后台前端源码中

### Requirement: UAT CI/CD 部署边界
系统 SHALL 将 GitHub-hosted runner、GitHub Container Registry、UAT self-hosted runner 和办公室私有网络 UAT 环境定义为相互分离的部署边界。

#### Scenario: 规划 UAT 发布链路
- **WHEN** 开发者规划 backend 或 admin portal 的 UAT 发布
- **THEN** 该发布链路必须区分云端 CI 构建、镜像仓库、内网 runner 部署和 UAT 运行环境
- **AND** 不得要求 GitHub-hosted runner 直接访问办公室内网资源

#### Scenario: 区分小程序发布和服务端部署
- **WHEN** backend 和 admin portal 通过 UAT GitHub Actions 发布
- **THEN** 该流程不得发布微信或抖音小程序
- **AND** 小程序发布必须继续作为独立平台发布流程处理

#### Scenario: 使用 repo 内基础设施边界
- **WHEN** UAT 部署需要 workflow、Dockerfile、compose 模板、env example 或部署说明
- **THEN** 文件必须位于 `.github/workflows/`、`backend/`、`frontend/admin/` 或 `infra/uat/` 对应边界内
- **AND** 不得创建平行工程结构
