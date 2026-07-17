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

### Requirement: Raw document import receipt 持久化合同
Data Service SHALL以forward-only `backend/migrations/000022_add_raw_document_import_receipts.sql`新增独立append-only `raw_document_import_receipts`表；现有21个migration文件MUST保持byte-for-byte不变，其按路径排序SHA-256 manifest的聚合hash MUST保持`2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`。raw receipt MUST NOT与`event_import_receipts`、events、sources、tags或review状态复用table或generic polymorphic schema。

#### Scenario: 创建 raw receipt schema artifact
- **WHEN** Package 4实现raw-document import persistence contract
- **THEN** repo migration文件数必须从21变为22且只能新增`000022_add_raw_document_import_receipts.sql`；必须记录新文件exact hash，Goose Up必须保持default single transaction且不得含`NO TRANSACTION`、`IF NOT EXISTS`或`OR REPLACE`，Down必须以`RAISE EXCEPTION`失败并禁止destructive/successful no-op，Package 4禁止执行SQL或修改前21个文件

#### Scenario: 验证最小表合同
- **WHEN** 检查`raw_document_import_receipts`
- **THEN** 表必须精确包含`id UUID`、`caller_identity TEXT`、`idempotency_key TEXT`、`payload_hash CHAR(64)`、`raw_document_ids UUID[]`、`result_payload JSONB`和`imported_at TIMESTAMPTZ`七个NOT NULL列，其中`imported_at`默认`now()`

#### Scenario: 验证 receipt constraints
- **WHEN** 检查raw receipt schema对象
- **THEN** 必须存在primary key、`UNIQUE(caller_identity,idempotency_key)`、trim后非空且最多200字符caller/key、64位lowercase hex hash、一维/无NULL/至少一个raw document ID、required/cross-field-consistent JSON result checks、imported-time index、固定`prevent_raw_document_import_receipt_mutation()`及拒绝`UPDATE/DELETE/TRUNCATE`的statement trigger

#### Scenario: 原子提交 raw documents 与 receipt
- **WHEN** 一个valid raw batch首次导入
- **THEN** 所有raw document写入/复用和immutable receipt insert必须位于一个PostgreSQL transaction并一起commit或rollback；whole-batch validation必须拒绝重复raw identity/ID，receipt raw IDs按canonical candidate order保存且stored result的receipt ID/hash/raw IDs/items/imported time必须与columns一致并足以精确重放首次structured business result

#### Scenario: receipt 查询语义
- **WHEN** 已认证caller按自身identity与idempotency key读取receipt status
- **THEN** 已存在row必须表示completed immutable result并返回stored snapshot，缺少row必须表示unknown/尚未commit；status不得按当前source/raw状态合成不同结果

#### Scenario: receipt retry conflict
- **WHEN** 已认证caller向import endpoint以既有idempotency key提交不同`raw-document-import-v1`canonical hash
- **THEN** Data Service必须返回409且不得插入、更新、删除或truncate任何raw document/receipt

#### Scenario: raw identity overlap race
- **WHEN** 不同caller或不同idempotency key的并发batch包含重叠source external ID/content hash
- **THEN** Data Service必须以sorted raw-identity transaction locks和conflict-safe insert/winner re-read串行化；external-ID与hash命中不同existing rows时必须409整批回滚，禁止无序选择任一row

#### Scenario: resolved membership 唯一
- **WHEN** 全部candidate完成raw identity resolution且尚未构造receipt
- **THEN** resolved ID数量必须等于candidate数量且全部唯一；不同candidate折叠到同一row必须返回`RAW_DOCUMENT_BATCH_COLLISION` 409并整批回滚

#### Scenario: event receipt 语义不变
- **WHEN** reviewed-event import执行或重放
- **THEN** 它必须继续使用既有`event_import_receipts`及event/source/tag/review transaction contract，不得读写`raw_document_import_receipts`来替代event receipt

### Requirement: Raw receipt local schema apply gate
系统SHALL把`000022`的local schema应用作为独立Package 5 R2两层操作，并MUST与Package 10 roles/credentials层分离；用户于2026-07-17批准在amendment获批且preflight evidence精确通过后自动执行一次该local migration。

#### Scenario: Package 5 order 1 preflight
- **WHEN** 准备应用`000022`
- **THEN** 必须以server-enforced`BEGIN READ ONLY`直接查询identity和既有Goose ledger，ledger缺失即停且不得运行可能EnsureDBVersion写入的check mode；确认applied=21、仅`000022`pending、显式排除new file后的21-file聚合hash、新文件exact hash/transactional Up/failing Down、全部固定对象名缺失、DDL/ledger privilege、business schema/data counts及完整可读backup archive+hash+documented recovery；不得声称restore-tested，任一漂移必须停止

#### Scenario: Package 5 order 2 apply and verify
- **WHEN** order 1全部证据通过且amendment已获批准
- **THEN** 系统必须以target-version 000022只执行一次apply，再只读断言migration=22、固定columns/constraints/index/function/trigger及初始receipt rows=0；除Goose ledger一条记录和列明new objects外pre-existing business schema/data必须不变，并运行全部synthetic DML rollback的atomicity/constraint/advisory-lock integration tests

#### Scenario: Package 5 不伪造真实 commit race
- **WHEN** rollback-only local integration验证两个connection的advisory lock
- **THEN** 两个transaction都必须rollback且最终receipt rows=0；winner commit后loser replay必须由Package 4 state-machine/SQL winner-re-read tests覆盖，不得在curated local持久化不可清理fixture或把rollback case报告为真实commit-race

#### Scenario: Package 5 fail closed
- **WHEN** apply、assertion或integration test失败、超时、出现partial state或任何identity/count/hash/schema漂移
- **THEN** 必须立即停止并保留backup/现场证据，不得restore、retry、forward-fix、seed、持久化业务数据、触碰Neo4j/UAT/prod/shared/deployment或执行role/credential变更

### Requirement: Data 数据库角色边界
系统 SHALL 定义 `data_service_rw`、`data_service_migrate`、`data_service_ro` 三类最小 PostgreSQL role，并 MUST 将真实 role/grant/credential 切换作为独立 R2 授权操作，与 R1 代码/服务边界调整分离。

#### Scenario: 完成代码边界调整
- **WHEN** Data/BFF 代码和 HTTP contract 已通过测试
- **THEN** 不得据此推定已授权创建role、变更grant、写secret或切换database credential；唯一migration授权是Package 5精确的local `000022`，且不得推定Package 10权限授权

#### Scenario: 请求 local role 切换
- **WHEN** 操作者准备切换 local Data PostgreSQL roles
- **THEN** 必须先确认database identity、grants/owner manifest、backup/recovery、before/after assertions、回切方式与停止条件，再取得独立明确授权；Package 10必须把raw receipt table/function owner转给`data_service_migrate`、收敛PUBLIC/function privilege且只向runtime授予必要SELECT/INSERT

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
