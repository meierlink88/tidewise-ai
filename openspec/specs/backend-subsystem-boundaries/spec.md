## Purpose

定义 Go 后端多子系统目录边界、共享基础层、实体基础库、采集器运行边界和依赖方向约束，作为后续后端 change 判断代码归属的当前系统事实。

## Requirements

### Requirement: 后端多子系统边界
系统 SHALL 在单一 Go module 内支持多个可独立运行的后端子系统，并通过 `cmd/*` 和 `internal/apps/*` 表达进程入口与业务子系统边界。

#### Scenario: 表达可运行进程
- **WHEN** 后端新增小程序 API、管理后台 API、采集调度器或运维命令
- **THEN** 进程入口必须位于 `backend/cmd/*`，并且入口只负责配置加载、依赖组装和启动流程

#### Scenario: 表达业务子系统
- **WHEN** 后端新增或修改业务子系统应用逻辑
- **THEN** 业务逻辑必须位于 `backend/internal/apps/<subsystem>`，不得直接堆叠在 `cmd/*`

#### Scenario: 共享基础层
- **WHEN** 多个子系统需要共享领域模型、repository、config、数据库连接、平台运行能力或跨子系统外部平台集成
- **THEN** 共享能力必须位于 `backend/internal/domain`、`backend/internal/repositories`、`backend/internal/config`、`backend/internal/platform` 或 `backend/internal/integrations`

#### Scenario: 区分数据库连接和业务数据访问
- **WHEN** 后端需要创建 PostgreSQL 连接、配置连接池或检查数据库连通性
- **THEN** 数据库连接基础设施必须位于 `backend/internal/platform/database`，业务 repository 必须继续位于 `backend/internal/repositories`

### Requirement: 实体基础库子系统边界
系统 SHALL 将实体基础库初始化定义为 backend 内的独立业务子系统能力，不得归入数据采集子系统。

#### Scenario: 初始化实体基础库
- **WHEN** 后端需要初始化经济体、联盟组织、政策机构、市场、指数、板块、产业链节点、公司、证券、指标、商品、KOL 或实体关系
- **THEN** 实体 seed 逻辑必须位于 `backend/internal/apps/entityfoundation/seed`，并通过 `backend/cmd/entity-seed` 运行

#### Scenario: 采集子系统引用实体
- **WHEN** 采集子系统需要把原始数据关联到实体基础库
- **THEN** 采集子系统可以读取或引用实体结果，但不得拥有实体 seed 逻辑或把实体初始化代码放入 `internal/apps/ingestion`

### Requirement: 采集器可运行子系统
系统 SHALL 将采集器定义为 backend 内独立可运行子系统，允许独立进程或容器 entrypoint 部署，但不得在 MVP 阶段拆成独立 Go module、repo 或数据库。

#### Scenario: 独立运行采集器
- **WHEN** 部署或本地开发需要启动采集器
- **THEN** 系统必须能够通过 `backend/cmd/ingestion-scheduler` 或采集相关命令运行采集子系统，而不要求启动小程序 API 进程

#### Scenario: 共享 backend 基础设施
- **WHEN** 采集器读取配置、访问 PostgreSQL、使用领域模型或执行 migration
- **THEN** 采集器必须复用 backend 内统一 config、domain、repository、platform 和 migration 来源

#### Scenario: 禁止提前拆分模块
- **WHEN** 普通功能 change 修改采集器
- **THEN** 不得顺手创建独立 `go.mod`、独立 repo、独立数据库 migration 根或独立版本发布边界

### Requirement: 后端依赖方向约束
系统 SHALL 通过文档和自动化检查约束后端子系统依赖方向，防止共享层反向依赖业务子系统或 API 子系统绕过采集边界。

#### Scenario: integrations 不依赖 apps
- **WHEN** 运行架构边界测试
- **THEN** `backend/internal/integrations` 不得 import `backend/internal/apps/*`

#### Scenario: API 子系统不直接调用采集 connector
- **WHEN** 运行架构边界测试
- **THEN** `miniappapi` 和 `adminapi` 不得直接 import `internal/apps/ingestion/connectors`

#### Scenario: cmd 不承载业务逻辑
- **WHEN** 新增或修改 `backend/cmd/*`
- **THEN** 入口代码必须只负责解析配置、组装依赖和启动进程，复杂业务流程必须下沉到对应 `internal/apps/<subsystem>`

#### Scenario: 平台能力不依赖业务子系统
- **WHEN** 运行架构边界测试
- **THEN** `backend/internal/platform` 不得 import `backend/internal/apps/*`

### Requirement: 图谱投影子系统边界
系统 SHALL 在 backend 内提供独立图谱投影子系统，用于把 PostgreSQL 事实数据投影到 Neo4j，并与采集、实体 seed、管理后台 API 和小程序 API 保持边界隔离。

#### Scenario: 放置图谱投影业务逻辑
- **WHEN** 后端实现 PG 到 Neo4j 的投影编排、运行报告、重建和错误处理
- **THEN** 业务逻辑必须位于 `backend/internal/apps/graphprojection` 或等价图谱投影子系统，而不是放入 `cmd/*`、`internal/apps/ingestion`、`internal/apps/entityfoundation/seed` 或 `internal/apps/adminapi`

#### Scenario: 放置 Neo4j 连接基础设施
- **WHEN** 后端实现 Neo4j driver 创建、连接池、连通性检查和关闭流程
- **THEN** 基础设施能力必须位于 `backend/internal/platform/graphdb` 或等价 platform 包，不得反向依赖业务子系统

#### Scenario: 提供图谱投影命令入口
- **WHEN** 开发者需要显式运行图谱投影、重建或连通性检查
- **THEN** 进程入口必须位于 `backend/cmd/graph-projector` 或等价 `cmd/*` 目录，并且入口只负责配置加载、依赖组装和启动流程

#### Scenario: 禁止采集器直接写 Neo4j
- **WHEN** 采集 connector、parser、runtime 或 scheduler 处理外部原始材料
- **THEN** 采集子系统不得直接写入 Neo4j，必须先通过 PostgreSQL 原始文档和后续事实表形成投影来源
