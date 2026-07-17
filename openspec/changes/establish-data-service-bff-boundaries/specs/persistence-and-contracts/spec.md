## ADDED Requirements

### Requirement: Data PostgreSQL 独占 ownership
现有 PostgreSQL 数据库及其 Entity、Chain Node、Raw Document、Event、Event Tag、Research Theme、Research Anchor、Index 和投影运行记录 SHALL 整体归 Data Service 独占；Miniapp、Admin 与 Agent MUST NOT 持有 Data DB 凭据或直接执行 SQL。

#### Scenario: BFF 访问持久化数据
- **WHEN** Miniapp/Admin 需要查询或修改 Data Domain 状态
- **THEN** 它必须调用 Data Service API，且其运行配置不得包含 Data PostgreSQL credential

#### Scenario: 保持现有表归属
- **WHEN** 服务边界迁移开始
- **THEN** Research Theme/Anchor 与其他现有 Data tables 必须继续归 Data Service，不得为了目录分层机械搬表、拆 schema 或重写 migration

#### Scenario: 保留历史 scheduler tables
- **WHEN** Tidewise scheduler/runtime与其repositories被删除
- **THEN** `ingestion_scheduler_configs`、`ingestion_runs`、`ingestion_run_sources`及migration `000005_add_ingestion_scheduler.sql`必须原样保留，且不得drop、truncate、重写历史SQL或删除既有rows

#### Scenario: 停止 scheduler 数据访问
- **WHEN** runtime退役完成
- **THEN** production应用不得继续创建、更新或通过Admin API读取scheduler config/run；未来历史审计需求必须另开只读Data API change

### Requirement: Data 数据库角色边界
系统 SHALL 定义 `data_service_rw`、`data_service_migrate`、`data_service_ro` 三类最小 PostgreSQL role，并 MUST 将真实 role/grant/credential 切换作为独立 R2 授权操作，与 R1 代码/服务边界调整分离。

#### Scenario: 完成代码边界调整
- **WHEN** Data/BFF 代码和 HTTP contract 已通过测试
- **THEN** 不得据此推定已授权创建 role、变更 grant、写 secret、执行 migration 或切换数据库 credential

#### Scenario: 请求 local role 切换
- **WHEN** 操作者准备切换 local Data PostgreSQL roles
- **THEN** 必须先确认数据库 identity、grants manifest、backup/recovery、before/after assertions、回切方式与停止条件，再取得独立明确授权

### Requirement: 未来领域数据库隔离
未来 Identity、Membership、Billing、Subscription 等独立领域服务 SHALL 使用独立数据库 ownership；跨领域引用 SHALL 保存 UUID 并通过 API contract 校验，MUST NOT 建立跨数据库 foreign key。

#### Scenario: 新增未来领域服务
- **WHEN** 后续 change 引入 Identity 或 Billing 数据
- **THEN** change 必须定义独立数据库与 API ownership，不得把其表默认加入 Data PostgreSQL

### Requirement: 跨服务事务边界
每个 Data command SHALL 在 Data Service 自身 PostgreSQL transaction 内保证原子性；BFF MUST NOT 持有跨服务数据库 transaction，系统 MUST NOT 在本 change 引入分布式事务。

#### Scenario: BFF 发起复合写操作
- **WHEN** 一个页面操作需要修改多条 Data records
- **THEN** BFF 必须调用一个有幂等 identity 的 Data aggregate command，由 Data Service 在单一 transaction 内完成或回滚
