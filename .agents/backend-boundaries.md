# Backend Boundaries

本文件定义 Go 后端多子系统结构、目录归属和依赖方向。处理任何后端 OpenSpec change 前，agent 必须先阅读本文件，再阅读对应 change artifacts 和相关代码。

## Backend Structure Model

后端采用单 Go module、多可部署子系统结构。

```text
backend/
├── cmd/
│   ├── miniapp-api/          # 面向小程序的 API/BFF 进程
│   ├── admin-api/            # 未来管理后台 API 进程
│   ├── ingestion-scheduler/  # 采集调度器进程
│   ├── source-ingest/        # 手动采集命令
│   ├── source-seed/          # 数据源初始化命令
│   ├── entity-seed/          # 实体初始化命令
│   └── dbmigrate/            # 数据库迁移命令
├── internal/
│   ├── apps/
│   │   ├── miniappapi/       # 小程序 API 子系统应用逻辑
│   │   ├── adminapi/         # 管理后台 API 子系统应用逻辑
│   │   ├── entityfoundation/ # 实体基础库初始化和实体主数据边界
│   │   └── ingestion/        # 数据采集子系统应用逻辑
│   ├── platform/             # 跨子系统共享平台能力，例如数据库连接和迁移执行器
│   ├── domain/               # 共享领域模型和值对象
│   ├── repositories/         # PostgreSQL 数据访问
│   ├── integrations/         # 跨多个子系统复用的外部平台集成
│   ├── http/                 # HTTP 协议适配和路由 glue
│   └── config/               # 强类型配置
├── migrations/               # 后端统一数据库 migration
├── data/                     # repo 内版本化 seed 数据资产
└── config/                   # local/uat/prod 非敏感配置模板
```

`cmd/*` 表达独立运行和独立部署的进程边界。`internal/apps/*` 表达业务子系统边界。`domain`、`repositories`、`config`、`integrations`、`platform` 是后端共享基础层。

## Subsystem Placement Rules

### Miniapp API

面向小程序用户的 API/BFF 能力放在：

```text
backend/cmd/miniapp-api/
backend/internal/apps/miniappapi/
```

小程序 API 不得直接调用采集 connector，不得包含数据采集调度逻辑。

### Admin API

面向运营和管理后台的 API 能力放在：

```text
backend/cmd/admin-api/
backend/internal/apps/adminapi/
```

管理后台可以通过 application/repository 查询采集源、运行记录和用户数据，但不得直接执行具体采集 connector。

### Ingestion

数据采集子系统放在：

```text
backend/cmd/ingestion-scheduler/
backend/cmd/source-ingest/
backend/cmd/source-seed/
backend/internal/apps/ingestion/
```

推荐子包：

```text
backend/internal/apps/ingestion/
├── core/           # 采集核心接口、注册表、凭证、限流和 raw document 写入
├── scheduler/      # 调度循环、到期来源选择、失败退避、run report
├── runtime/        # 单次采集执行编排
├── sourcecatalog/  # source seed、校验、统计
├── connectors/     # RSS、Eastmoney、RSSHub、web_fetch、local_file 等采集连接器
├── parsers/        # rss_item、eastmoney_json、html_text 等解析器
└── health/         # 来源健康状态和失败策略
```

采集子系统可以独立部署为 `ingestion-scheduler` 进程。MVP 阶段默认只运行一个调度器实例；如果需要多实例，必须先通过独立 change 设计分布式锁、数据库 leasing、Redis 锁或任务队列。

### Entity Foundation

实体基础库初始化能力放在：

```text
backend/cmd/entity-seed/
backend/internal/apps/entityfoundation/seed/
backend/data/entity_foundation/
```

实体基础库负责经济体、联盟组织、政策机构、市场、指数、板块、产业链节点、公司、证券、指标、商品、KOL 和实体关系等主数据初始化。它不属于数据采集子系统；采集子系统可以引用实体基础库的结果，但不得拥有实体 seed 逻辑。

## Integrations Boundary

`backend/internal/integrations` 只放跨多个后端子系统复用的外部平台集成，例如：

```text
backend/internal/integrations/
├── agentplatform/   # Dify 或外部 Agent 平台
├── payment/         # 支付
├── messaging/       # 短信、邮件、订阅消息
├── storage/         # OSS/S3/对象存储
├── authprovider/    # 第三方登录、OAuth
└── push/            # 推送服务
```

判断规则：

- 只服务采集链路的数据源适配器，放 `internal/apps/ingestion/connectors` 或 `internal/apps/ingestion/parsers`。
- 多个后端子系统都会调用的外部平台能力，放 `internal/integrations`。

例子：

- Eastmoney、RSS、RSSHub、web_fetch、local_file 属于采集子系统 connector。
- Dify、支付、对象存储、短信、第三方登录属于共享 integrations。

## Dependency Direction

允许依赖方向：

```text
cmd/*
  -> internal/apps/<subsystem>
  -> domain / repositories / config / integrations / platform

internal/apps/ingestion
  -> internal/apps/ingestion/core
  -> domain / repositories / config / platform
  -> internal/apps/ingestion/connectors
  -> internal/apps/ingestion/parsers

internal/apps/entityfoundation
  -> domain / repositories / config / platform

internal/integrations
  -> domain / config

internal/platform
  -> domain / config
```

禁止依赖：

- `internal/integrations` 不得 import `internal/apps/*`。
- `internal/apps/miniappapi` 不得 import `internal/apps/ingestion/connectors`。
- `internal/apps/adminapi` 不得 import `internal/apps/ingestion/connectors`。
- `cmd/*` 不得承载业务逻辑，只负责解析配置、组装依赖和启动进程。
- connector 不得直接写 SQL 或绕过 repository。
- parser 不得访问数据库、环境变量或外部网络。
- 数据库 migration runner 应放在 `internal/platform/dbmigration`，不得放在 `internal/migrations`。
- 数据库连接基础设施应放在 `internal/platform/database`，不得放在 `internal/database`。
- 实体基础库 seed 应放在 `internal/apps/entityfoundation/seed`，不得放在 `internal/entityseed` 或 `internal/apps/ingestion`。

## Migration And Data Rules

后端当前使用统一 PostgreSQL migration：

```text
backend/migrations/
```

不要为每个子系统创建独立 migration 目录，除非后续已经通过 OpenSpec change 决定拆分数据库或拆分 Go module。

数据库 migration 执行器代码放在：

```text
backend/internal/platform/dbmigration/
```

`backend/migrations` 是 SQL 文件来源，`internal/platform/dbmigration` 是读取、检查和执行 migration 的 Go 运行时能力。不要再创建 `backend/internal/migrations`，避免和 SQL 目录产生语义冲突。

数据库连接基础设施放在：

```text
backend/internal/platform/database/
```

`internal/platform/database` 只负责数据库驱动注册、连接创建、连接池配置和连通性检查。领域数据访问仍然属于 `internal/repositories`，不要把业务 repository 放入 `platform/database`。

长期 seed 数据资产放在：

```text
backend/data/<data-domain>/
```

采集源清单属于：

```text
backend/data/source_catalogs/
```

实体基础库属于：

```text
backend/data/entity_foundation/
```

## Future Split Criteria

只有满足以下条件之一时，才考虑把某个后端子系统拆成独立 Go module、独立 repo 或独立数据库：

- 需要独立版本化和独立发布生命周期。
- 需要独立团队长期维护。
- 运行时依赖显著不同，例如大量 Python SDK worker。
- 资源消耗或扩缩容模型明显不同。
- 子系统之间通过队列、事件流或稳定 API contract 通信，而不是共享 repository 直接访问 PostgreSQL。
- 单一后端 module 已经造成明显构建、测试或依赖冲突。

拆分必须通过独立 OpenSpec change 设计，不得在普通功能实现中顺手拆分。
