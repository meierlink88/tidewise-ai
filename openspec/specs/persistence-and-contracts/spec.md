## Purpose

定义观潮家正式模块开发前的持久化、缓存、图谱/向量演进、数据采集、API 契约、Agent 回写和异步任务边界，作为后续事件、报告、订阅、采集和 AI 分析模块的当前系统事实。

## Requirements

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
- **THEN** 必须先确认database identity、grants/owner manifest、backup/recovery、before/after assertions、回切方式与停止条件，再取得独立明确授权；Package 10必须把raw receipt table/function owner转给`data_service_migrate`、收敛PUBLIC/function privilege且只向runtime授予必要SELECT/INSERT；两类receipt lookup必须由幂等key advisory transaction lock保护并使用plain `SELECT`，不得以row-locking clause迫使runtime获得`UPDATE` privilege

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

### Requirement: MVP 持久化基础设施
系统 SHALL 使用 PostgreSQL 作为 MVP 阶段结构化主存储，并使用 Redis 承载缓存、限流、幂等和短期任务状态。

#### Scenario: 选择结构化主存储
- **WHEN** 后续 change 引入用户、事件、市场指标、板块、资产、订阅、报告、Agent 分析结果或任务记录
- **THEN** 该 change 必须默认以 PostgreSQL 作为结构化事实数据的主存储

#### Scenario: 使用短期状态
- **WHEN** 后续 change 需要缓存、限流、幂等键、短期会话状态或任务进度
- **THEN** 该 change 必须通过 Redis 边界表达短期状态，而不是把短期状态散落在内存或页面中

### Requirement: 图谱和向量能力演进边界
系统 SHALL 在 MVP 阶段不直接引入独立图数据库或向量数据库，并通过 PostgreSQL 结构化关系和外部 Agent 平台结果承载初始图谱/RAG 数据边界。

#### Scenario: 存储初始关系数据
- **WHEN** 后续 change 需要保存事件、实体、板块、资产或报告之间的关系
- **THEN** 该 change 必须优先通过 PostgreSQL 结构化关系表达，并保留未来迁移到图数据库的边界

#### Scenario: 使用 RAG 或向量检索
- **WHEN** 后续 change 需要 RAG 检索、向量召回或 Prompt 编排
- **THEN** 该能力必须优先经由外部 Agent 平台或明确的服务端集成边界承载，而不是在前端或业务 handler 中直接实现

### Requirement: 数据采集输入边界
系统 SHALL 将热点事件和外部信号的受控接入定义为 Data Service ingestion/import 能力；来源访问、调度和采集执行由外部 `agent-run`/Agent Server承担，Tidewise只接受经认证的raw-document或reviewed-event API输入。

#### Scenario: 外部执行系统提交原始材料
- **WHEN** 外部执行系统采集新闻、公告、政策、市场异动、行业事件或热度信号
- **THEN** 结果必须通过 Data Service raw-document import完成身份、来源、幂等和whole-batch校验，不得由前端、BFF或业务handler直接使用

#### Scenario: 接入 Agent 采集结果
- **WHEN** 后续 change 通过外部 Agent API 获取已经采集或初步分析后的事件数据
- **THEN** Agent必须按材料成熟度调用raw-document或reviewed-event Data API，且不得直连Data DB

### Requirement: 采集数据标准化和存储分流
系统 SHALL 对采集后的原始数据执行来源追踪、去重、清洗、标准化和质量标记，并按数据形态进入关系型、向量或图谱存储边界。

#### Scenario: 标准化采集结果
- **WHEN** 采集层收到自研爬虫或外部 Agent API 的原始结果
- **THEN** 系统必须生成标准化事件、来源、时间、标签、实体、置信度和处理状态，而不是直接把原始结果作为系统事实

#### Scenario: 分流不同数据形态
- **WHEN** 标准化后的数据包含结构化事件、语义向量、实体关系或事件传导链
- **THEN** 系统必须根据数据形态进入 PostgreSQL、未来向量数据库或未来图谱数据库边界，并在独立 change 中定义非 PostgreSQL 存储的引入方式

### Requirement: API 契约边界
系统 SHALL 在真实业务 API 实现前定义 API 契约边界，覆盖请求响应 DTO、错误结构、分页、时间、ID、枚举和 Agent 回写 payload。

#### Scenario: 定义业务接口
- **WHEN** 后续 change 新增面向小程序的业务 API
- **THEN** 该 API 必须先定义契约，包含请求、响应、错误、分页和字段语义

#### Scenario: 定义 Agent 回写接口
- **WHEN** 后续 change 新增 Agent 平台结构化结果回写
- **THEN** 该回写必须先定义 payload 契约、幂等字段、状态字段和校验规则

### Requirement: Agent 平台回写治理
系统 SHALL 要求 Agent 平台回写必须经过后端鉴权、幂等、结构化校验、状态流转和落库边界。

#### Scenario: 接收 Agent 回写
- **WHEN** Agent 平台向本系统回写分析结果、报告内容、图谱关系或任务状态
- **THEN** 后端必须校验调用方身份、幂等键、payload 结构和状态流转后再保存结果

#### Scenario: 展示 Agent 输出
- **WHEN** 前端展示 Agent 平台生成的分析、报告或市场解读
- **THEN** 展示内容必须保持决策辅助定位，不得表达为直接投资建议

### Requirement: 异步任务边界
系统 SHALL 将 Agent 分析、报告生成、图谱更新、通知投递和支付/订阅后处理等长时间流程表达为服务端 job 边界；事件采集的schedule/execution job归外部`agent-run`，不得在Tidewise恢复采集runtime。

#### Scenario: 识别长时间任务
- **WHEN** 某个能力无法在一次普通 HTTP 请求中稳定完成
- **THEN** 该能力必须通过服务端 job、任务状态和可查询进度表达，而不是阻塞小程序请求

#### Scenario: 查询任务状态
- **WHEN** 小程序需要展示报告生成、AI 分析或订阅后处理进度
- **THEN** 小程序必须通过 API 契约查询任务状态，而不是直接读取内部队列、数据库或 Agent 平台状态

### Requirement: 采集原始数据持久化
系统 SHALL 将 Data Service受控import接收到的原始外部材料校验、标准化并保存到 PostgreSQL 的采集源目录和原始文档边界中。

#### Scenario: 保存采集源目录
- **WHEN** 系统注册外部来源
- **THEN** 必须保存来源通道、provider、connector、parser、来源类型、来源 URL、主题提示、授权策略、限流策略、凭证引用和状态

#### Scenario: 保存原始文档
- **WHEN** 已认证调用方提交通过whole-batch validation的原始文档候选
- **THEN** 必须保存对应原始文档，并保留来源、发布时间、采集时间、内容哈希、原始对象 URI、内容类型和入库状态

#### Scenario: 通过 migration 创建持久化结构
- **WHEN** 采集源目录、原始文档或事件证据相关结构需要创建或调整
- **THEN** 必须通过 repo 内版本化 SQL migration 创建或增量修改，不得只在代码模型中表达数据库结构

### Requirement: 本地基础设施配置边界
系统 SHALL 将本地 PostgreSQL 运行所需的非敏感配置和示例模板保存在 repo 内，并将真实 secret 留给环境变量或未提交文件。

#### Scenario: 查看本地配置模板
- **WHEN** 开发者查看本地Data Service数据库或import client运行模板
- **THEN** 模板必须说明需要的变量名和用途，但不得包含真实密码、真实 token 或生产连接串

#### Scenario: 切换 local、uat、prod
- **WHEN** 后续环境需要执行同一套 migration
- **THEN** 系统必须通过环境配置和 secret 注入切换连接目标，而不是修改 migration 文件或业务代码

### Requirement: 采集结果结构化校验
系统 SHALL 在原始文档进入数据库前执行结构化校验和质量标记，确保后续事件抽取不依赖未校验的外部响应。

#### Scenario: 校验必填字段
- **WHEN** 原始文档候选对象缺少标题、来源、内容哈希或可识别来源信息
- **THEN** 系统必须拒绝成功入库或标记为失败状态，并记录明确错误

#### Scenario: 标记处理状态
- **WHEN** 原始文档完成写入、解析失败、重复跳过或等待后续抽取
- **THEN** 系统必须保存可查询的入库状态，而不是只依赖进程日志

### Requirement: Agent 和采集结果边界
系统 SHALL 保持外部采集执行、Data Service受控import和后续 Agent 推理结果的边界清晰，避免原始响应绕过校验直接成为系统事实。

#### Scenario: 接收外部 Agent 采集结果
- **WHEN** 外部 Agent API 返回已经采集或初步整理的事件材料
- **THEN** Agent必须根据材料成熟度调用Data raw-document或reviewed-event API，由Data Service校验并写入对应事实边界

#### Scenario: 展示分析结果
- **WHEN** 后续前端或 API 展示基于采集数据生成的分析内容
- **THEN** 展示内容必须保持决策辅助定位，不得表达为直接投资建议

### Requirement: 版本化实体基础库初始化
系统 SHALL 通过 repo 内版本化实体 seed 初始化和更新实体基础库，而不是依赖手工数据库操作、临时 SQL 或散落脚本。

#### Scenario: 初始化实体基础库
- **WHEN** local、uat 或 prod 环境需要初始化一阶段实体基础库
- **THEN** 运维或开发者必须能够运行统一 seed 命令，将 repo 内实体清单写入 PostgreSQL，并得到实体总数、类型分布、profile 写入数量和关系写入数量的统计报告

#### Scenario: 覆盖所有实体 profile
- **WHEN** 一阶段实体 seed 完成
- **THEN** PostgreSQL 中必须至少完成联盟组织、经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物这些实体 profile 表的初始化验证

#### Scenario: 幂等更新实体基础库
- **WHEN** 同一套实体 seed 在同一数据库中重复执行
- **THEN** 系统必须按稳定实体 key 幂等 upsert，不得创建重复实体、重复 profile 或重复关系

#### Scenario: 审计实体基础库变更
- **WHEN** 后续新增、禁用或修改基础实体、profile 属性或基础关系
- **THEN** 该变更必须体现在 repo 内实体 seed、测试和必要 migration 中，而不是只修改数据库现状

#### Scenario: 保持决策辅助边界
- **WHEN** 实体 seed 初始化基础实体和关系
- **THEN** seed 数据不得包含投资建议、预测结论、利好利空判断、传导强度或事件评分

### Requirement: 采集源扩展配置持久化
系统 SHALL 在 PostgreSQL 采集源目录中持久化非敏感扩展配置，使来源参数可以随 `source_catalogs` 一起查询、审计和迁移。

#### Scenario: 创建扩展配置字段
- **WHEN** 数据库迁移执行到本 change 的版本
- **THEN** PostgreSQL 必须为 `source_catalogs` 提供 `source_config` JSONB 字段，默认值为空 JSON 对象，并且不得影响既有来源记录读取

#### Scenario: 读取扩展配置
- **WHEN** repository 查询 active source 或 seed 后读取来源记录
- **THEN** 系统必须返回 `source_config` 中的非敏感结构化参数，供受控source-metadata API、adapter contract tests或未来外部执行适配使用

### Requirement: 版本化来源初始化
系统 SHALL 通过 repo 内版本化来源清单初始化和更新统一采集源目录，而不是依赖手工数据库操作或散落脚本。

#### Scenario: 初始化来源目录
- **WHEN** local、uat 或 prod 环境需要初始化第一批采集源
- **THEN** 运维或开发者必须能够运行统一 seed 命令，将 repo 内来源清单写入目标 PostgreSQL，并得到接入来源总数和分类统计

#### Scenario: 审计来源变更
- **WHEN** 后续新增、禁用或修改采集来源
- **THEN** 该变更必须体现在 repo 内来源清单、测试和必要 migration 中，而不是只修改数据库现状

#### Scenario: 保护敏感配置
- **WHEN** 来源初始化涉及需要凭证的 provider
- **THEN** seed 数据必须只写入授权类型和凭证引用，不得写入真实 API key、cookie、token 或私有数据库连接信息

#### Scenario: 管理多类型来源
- **WHEN** 来源清单包含内容、行情、板块或本地回灌来源
- **THEN** PostgreSQL 中的 `source_catalogs` 必须能够保存这些来源的用途、类型、provider、connector、parser、扩展配置、授权策略、限流策略和状态，便于统一查询和治理

### Requirement: 统一 migration 和 seed 数据边界
系统 SHALL 在 MVP 阶段保持 backend 统一 PostgreSQL migration 和 repo 内 seed 数据资产边界，后端子系统不得在普通功能 change 中创建独立 migration 根或独立数据库。

#### Scenario: 使用统一 migration
- **WHEN** 小程序 API、管理后台 API、采集子系统或后端运维命令需要调整 PostgreSQL schema
- **THEN** 该变更必须使用 `backend/migrations` 作为统一 migration 来源，除非已有独立 OpenSpec change 决定拆分数据库

#### Scenario: 区分 migration 文件和执行器
- **WHEN** 后端需要读取、检查或执行 PostgreSQL migration
- **THEN** Go 执行器代码必须位于 `backend/internal/platform/dbmigration`，并且 SQL migration 文件必须继续位于 `backend/migrations`

#### Scenario: 管理 seed 数据资产
- **WHEN** 后端需要维护采集源清单、实体基础库或其他 repo 内长期 seed 数据
- **THEN** 数据资产必须放在 `backend/data/<data-domain>` 并通过对应 seed 命令和 repository 边界写入数据库

#### Scenario: 禁止隐式拆库
- **WHEN** 普通功能 change 修改某个后端子系统
- **THEN** 不得顺手创建子系统私有数据库、私有 migration 目录或私有 schema 管理机制

### Requirement: AI connector 非敏感配置持久化
系统 SHALL 允许 `source_catalogs.source_config` 保存 AI Web Research connector 的非敏感运行参数，并保持真实凭证隔离。

#### Scenario: 保存 AI connector 配置
- **WHEN** AI Web Research source 被 seed 到 `source_catalogs`
- **THEN** PostgreSQL 必须保存该 source 的 `collection_mode`、`search_plan_mode`、固定查询计划、`web_search_plan`、搜索选项、来源偏好、可信域名、LLM planner provider、API base URL、API 协议、模型名、`prompt_ref`、`prompt_version`、`prompt_variables`、时间窗口、结果上限、语言和输出 schema 等非敏感配置

#### Scenario: 引用真实凭证
- **WHEN** AI Web Research source 需要调用真实 Web Search API 或真实模型 API
- **THEN** `source_catalogs` 只能保存 `credential_ref` 或 `source_config.credential_refs` 这类凭证引用名，真实 API key 必须来自环境变量或部署平台 secret

### Requirement: AI 采集元数据追踪
系统 SHALL 在原始文档或其 raw metadata 中保留 AI Web Research 的采集上下文，使后续审计能够追踪材料来源、模型配置和提示词版本。

#### Scenario: 保存采集上下文
- **WHEN** AI Web Research item 写入原始文档边界
- **THEN** 系统必须保留 search_plan_mode、查询计划来源、web_search_plan 摘要、参与召回的 search tool、llm_provider、model、api_protocol、prompt_ref、prompt_version、prompt_purpose、search_options、source_preferences、trusted_domain_match、content_origin、retrieval_method、source_attribution_type、来源说明、provider 搜索元数据和原始返回片段等非敏感元数据

#### Scenario: 排除敏感元数据
- **WHEN** 保存 AI Web Research 原始返回或请求元数据
- **THEN** 系统不得保存真实 API key、Authorization header、cookie、私有 token 或其他敏感凭证

### Requirement: Neo4j 图谱投影持久化边界
系统 SHALL 在引入 Neo4j 时保持 PostgreSQL 作为结构化事实主存储，并将 Neo4j 中的数据限定为从 PostgreSQL 派生的可重建图谱投影。

#### Scenario: 写入图谱数据
- **WHEN** 系统需要把实体、实体关系、事件或事件实体关联写入 Neo4j
- **THEN** 对应事实必须先存在于 PostgreSQL，并通过投影流程写入 Neo4j

#### Scenario: 不把 Neo4j 作为事实源
- **WHEN** Neo4j 中存在某个实体节点或关系
- **THEN** 系统不得仅凭 Neo4j 数据把它视为权威事实，必须能追溯到 PostgreSQL 中的事实来源

#### Scenario: 记录投影运行状态
- **WHEN** 图谱投影流程执行
- **THEN** 系统必须在 PostgreSQL 或等价事实边界中保存投影运行状态、统计和错误摘要，便于审计和重试

#### Scenario: 禁止保存敏感连接信息
- **WHEN** 系统保存 Neo4j 或图谱投影配置
- **THEN** 配置中不得保存真实用户名、密码、token、私有连接串密钥或其他敏感凭证

### Requirement: migration SQL 与测试边界分离
系统 SHALL 使 `backend/migrations` 只保存版本化 SQL migration 与 `README.md`，并在现有 `backend/internal/platform/dbmigration` 测试边界统一验证 migration source、静态安全契约、Goose 执行边界和明确标记的可选 PostgreSQL integration 行为。

#### Scenario: 检查 migration 目录纯度
- **WHEN** 开发者运行 dbmigration contract tests
- **THEN** `backend/migrations` 中除版本化 `.sql` 和 `README.md` 外不得存在 Go 测试、执行器或其他运行时代码

#### Scenario: 迁移有效安全契约
- **WHEN** migration test 从 SQL 目录迁入 `internal/platform/dbmigration`
- **THEN** 当前 schema、授权开关、非破坏性、幂等、事务、rollback、Goose statement 和完整 migration chain 的仍有效契约必须继续由自动化测试保护

#### Scenario: 删除已废止 schema 测试
- **WHEN** 某项测试只要求已被后续 migration 和主规格明确废止的中间 schema 继续作为最终结构存在
- **THEN** 系统必须允许删除该断言
- **AND** 删除前必须证明 migration 版本/格式、完整链执行和仍有效安全边界由统一测试位置继续覆盖

#### Scenario: 不执行有状态验证
- **WHEN** 本行为保持 change 未取得任何数据库写入授权
- **THEN** 默认验证不得执行 migration、seed 或 PostgreSQL/Neo4j write
- **AND** 真实 PostgreSQL integration tests 必须继续保持显式 opt-in
