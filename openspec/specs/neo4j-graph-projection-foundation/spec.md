## Purpose

定义 Neo4j 图谱投影基础设施的当前系统事实，覆盖 Neo4j 配置连接、PG 到 Neo4j 的实体图投影、幂等重建、投影运行记录和验证边界。

## Requirements

### Requirement: Neo4j 图谱投影基础设施
系统 SHALL 提供 Neo4j 图谱投影基础设施，使后端可以通过统一配置连接 Neo4j，并将 PostgreSQL 中的实体和实体关系投影为可重建的图谱查询视图。

#### Scenario: 加载 Neo4j 配置
- **WHEN** 后端图谱投影命令或 worker 启动
- **THEN** 系统必须通过统一 Go 强类型配置加载 Neo4j 启用状态、URI、database、连接超时和连接池参数

#### Scenario: 隔离 Neo4j 敏感凭证
- **WHEN** 系统需要连接 Neo4j
- **THEN** 真实用户名、密码、token 或私有连接密钥必须通过环境变量或部署平台 secret 注入，不得写入 repo 配置文件、OpenSpec artifact、日志或数据库

#### Scenario: 检查 Neo4j 连通性
- **WHEN** 开发者运行图谱投影连通性检查
- **THEN** 系统必须返回明确的成功或失败结果，并包含非敏感错误原因

### Requirement: 实体节点图谱投影
系统 SHALL 能够从 PostgreSQL `entity_nodes` 和实体 profile 中投影 Neo4j 实体节点，并保持节点使用稳定业务标识。

#### Scenario: 投影实体节点
- **WHEN** 图谱投影流程读取到 active、inactive 或 merged 状态的实体节点
- **THEN** Neo4j 中必须使用稳定 `entity_id` 和 `entity_key` upsert 对应 `Entity` 节点，并保存实体类型、层级、名称、规范名称、状态和投影命名空间

#### Scenario: 重复投影实体节点
- **WHEN** 同一批实体节点被重复投影
- **THEN** 系统必须更新或复用已有 Neo4j 节点，而不是创建重复节点

#### Scenario: 只清理本系统命名空间
- **WHEN** 开发者执行实体图全量重建
- **THEN** 系统只能清理本系统投影命名空间下的实体节点和关系，不得清空整个 Neo4j database 或删除其他命名空间数据

### Requirement: 实体关系图谱投影
系统 SHALL 能够从 PostgreSQL `entity_edges` 投影 Neo4j 实体关系，并保持关系类型安全、可审计和可重建。

#### Scenario: 投影实体关系
- **WHEN** 图谱投影流程读取到有效 `entity_edges`
- **THEN** Neo4j 中必须在对应实体节点之间 upsert 关系，并保存 edge ID、原始 relation type、来源、置信度、状态、更新时间和投影命名空间

#### Scenario: 拒绝不安全关系类型
- **WHEN** PostgreSQL 中的关系类型无法安全映射为 Neo4j relationship type
- **THEN** 系统必须拒绝该关系投影或映射为明确的安全 fallback 类型，并在运行报告中记录处理结果

#### Scenario: 跳过缺失端点的关系
- **WHEN** 某条 `entity_edges` 的起点或终点实体无法在投影快照中找到
- **THEN** 系统必须跳过该关系并记录错误原因，不得创建悬空图关系

### Requirement: 图谱投影运行记录
系统 SHALL 保存图谱投影运行记录，使开发者能够审计每次投影的输入范围、输出数量、错误和耗时。

#### Scenario: 保存投影运行摘要
- **WHEN** 一次图谱投影开始和结束
- **THEN** 系统必须记录运行 ID、投影类型、运行模式、状态、开始时间、结束时间、输入数量、成功数量、跳过数量、失败数量和错误摘要

#### Scenario: 查询最近投影结果
- **WHEN** 开发者需要验证图谱投影是否完成
- **THEN** 系统必须能够通过命令输出、repository 或 SQL 查询最近投影运行记录，而不是只依赖进程日志

### Requirement: 图谱投影可验证性
系统 SHALL 通过自动化测试和显式 smoke 边界验证 Neo4j 投影能力，普通测试不得依赖真实 Neo4j 或真实外部网络。

#### Scenario: 使用 fake writer 测试投影逻辑
- **WHEN** 运行普通 Go 单元测试
- **THEN** 系统必须使用 fake graph writer 或等价测试替身验证实体映射、关系映射、幂等、失败处理和运行报告

#### Scenario: 显式运行真实 Neo4j smoke
- **WHEN** 开发者显式启用 Neo4j smoke 环境变量并提供本地 Neo4j 凭证
- **THEN** 系统可以连接真实 Neo4j 执行少量投影验证，并且该 smoke 默认不得在普通 `go test ./...` 中自动运行

### Requirement: 空投影和已审阅关系重建
系统 SHALL 支持清空 local Neo4j 投影数据，并在关系分批审阅后仅从 PostgreSQL 当前事实重建实体图谱。

#### Scenario: 建立空 Neo4j 投影
- **WHEN** 开发者在确认 local 环境后执行关系清洗基线重置
- **THEN** 系统必须允许删除 Neo4j 中全部节点和关系数据，同时保留 database、约束、索引和连接配置

#### Scenario: PG 无关系时重建实体图
- **WHEN** PostgreSQL `entity_edges` 为空并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须只包含可投影实体节点且不包含任何实体关系

#### Scenario: 按已审阅 PG 关系重建
- **WHEN** 某一关系批次已写入 PostgreSQL 并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须移除历史投影关系并只投影当前 active `entity_edges`，不得保留 PostgreSQL 中不存在的关系

#### Scenario: 使用单一实体标签和命名空间
- **WHEN** PostgreSQL 实体被投影到 Neo4j
- **THEN** 节点必须使用 `Entity` 标签并通过 `projection_namespace=tidewise` 标识归属，不得叠加与同一数据集合重复的 `TidewiseEntity` 标签
