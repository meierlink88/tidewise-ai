# Tidewise 统一数据采集中心 V1 架构 Spec

> **Superseded**
>
> 本 Proposed 设计已被 ADR-0005 与 `event-publication-v2.md` 取代，不得继续
> 实施。Source、采集控制面、完整 Artifact 与 Event 提取归 AgentRun；
> Tidewise Data 只接纳正式 Event 和被引用的轻量证据记录。以下正文仅保留为
> 历史设计背景。

## 状态

- 状态：Superseded
- 日期：2026-07-22
- 涉及仓库：`tidewise-ai`、`tidewise-ai-agentrun`
- 所属上下文：Data
- 决策前置：采用本 Spec 前，必须新增 ADR，修订 `docs/contexts/data/CONTEXT.md`，并取代 ADR-0002 中“connector、parser、prompt 和调度执行全部归外部 agent-run”的局部决策。

### 已冻结的第一阶段决策

- 第一阶段交付 `Collector Agent V1`，范围是完整复刻当前 Codex 原始信息采集逻辑；Event 提取、审核和产业链分析不进入 Collector Agent。
- 第一阶段同时建立可继续承载 Event Extractor Agent、Analyst Agent 的 agent-run 执行平台基础，不把 Collector 写成一次性专用进程。
- agent-run 使用独立 PostgreSQL 数据库 `tidewise_ai_server`。
- `tidewise_ai_server` 可以与 Tidewise Data 数据库位于同一 PostgreSQL instance，但使用独立 database、连接身份、migration、连接池、备份和恢复边界。
- 两个数据库之间禁止跨库 SQL、FDW、共享表、视图或数据库级直接读取；Data Center 与 agent-run 只通过版本化 HTTP Interface 交互。

### Grill Review 决议（2026-07-22）

实施前已对本 Spec 与 AgentRun Collector Spec 做严格交叉审查。以下歧义已经关闭，后文若有冲突以本节为准：

| 阻塞项 | 决议 | 验收方式 |
|---|---|---|
| 两仓 Run Schema 不一致 | Data 的不可变 Run Spec 在线格式唯一命名为 `collector_run_request.v1`；它也是本文概念上的 `collection_run_spec.v1`，不得维护第二套字段不同的 JSON | 两仓共享 golden fixture |
| Data Attempt 与 AgentRun Attempt 混淆 | `Collection Attempt` 是 Data 对同一 Run 的跨执行重试；每个 Attempt 创建一个独立 Agent Execution。AgentRun 内部 `Agent Execution Attempt` 只表示 Worker lease/接管 | 状态映射和幂等测试 |
| Data 下发具体模型与 Connector 清单破坏所有权 | Data 只下发逻辑 LLM Profile Key 和 Connector Profile Set Key；模型、URL、Credential、enabled Connector、并发、单 Connector 限制都由 AgentRun 的不可变 Profile Version 决定 | 请求 schema 拒绝越权字段 |
| Run Spec 推/拉未决 | V1 在创建请求中推送完整不可变快照；AgentRun 不回 Data 拉取 | Data fake/AgentRun consumer test |
| Queue 未决 | AgentRun V1 使用自身 PostgreSQL `FOR UPDATE SKIP LOCKED` + lease；Data 使用自己的 Run/Lease 机制；不引入跨库队列或外部 MQ | kill/restart 和双领取测试 |
| Credential 冻结与轮换未定义 | AgentRun 使用不可变 Credential Version；AES-256-GCM envelope encryption，随机 DEK 由数据库外版本化 KEK 包装 | 轮换、泄密扫描和恢复测试 |
| Codex Live Search 不能服务端运行 | V1 逻辑 `live_search` 映射 AgentRun `brave_web_search` Adapter，消费 Brave Web Search 直接 title/url/description/age，不调用 LLM Context、不二次打开 URL；仅对齐行为合同，不宣称索引/排序等价 | fake 合同测试 + 部署 smoke |
| “每 Connector 一次”在崩溃后无法 exactly-once | AgentRun 在调用前持久化唯一 Invocation；V1 不做 Connector HTTP 自动重试。崩溃后的未知调用标记 `interrupted_unknown`，同一 Collection Attempt 不重放；Data 可创建下一 Attempt | 故障注入测试 |
| Raw Import Outcome 误设逐项 rejected | Raw Import 是整批严格事务，只返回 `created/reused`；本地 rejected 指 AgentRun 在建批前形成的 Candidate 终态，Data 4xx 使整批失败 | OpenAPI consumer test |
| 去重算法实现漂移 | 以 Codex 合同为权威：URL canonicalization、正文 SHA-256、BLAKE2b-64 token SimHash64、Hamming `<=10`；不沿用当前半径 3/SHA token 漂移 | 跨语言 golden fixtures |

AgentRun 的完整审查证据和算法细节记录在 `docs/specs/collector-agent-v1-platform-foundation.md`。在 Data 侧 ADR/Context、Prompt Registry、Run/Attempt、Remote AI Adapter 和 Outcome Interface 尚未交付前，本 Spec 仍为 Proposed，不能宣称端到端上线。

## 背景

Tidewise 当前拥有 202 条 Source Catalog 记录，但它们代表受治理的来源和执行配置，不等于 202 套独立、可持续运行的采集实现。当前记录分布为：

| Connector Key | Source 数量 | 当前状态 |
|---|---:|---|
| `rss_feed` | 109 | active |
| `eastmoney` | 48 | inactive |
| `market_provider` | 26 | inactive |
| `local_backfill` | 9 | inactive |
| `web_fetch` | 7 | inactive |
| `llm_web_research` | 2 | inactive |
| `rsshub_feed` | 1 | inactive |

Data 当前拥有 Source Catalog、Raw Document/Event 受控导入、幂等、事务和 receipt；历史上的固定 Connector、Parser 和 Scheduler 已从当前运行入口移除。`tidewise-ai-agentrun` 当前通过 Eino、Planner 和多个搜索 Connector 完成 AI 搜索采集，但仍以本地文件维护业务采集提示词，也未成为由 Data 统一调度的远程执行器。

本 Spec 将 Tidewise Data Collection 定义为统一采集中心：统一维护 Source、执行策略、采集提示词、调度计划、Run 状态和正式数据入口；固定采集由 Tidewise 内的独立 Worker 执行，AI 采集由 `tidewise-ai-agentrun` 作为远程 AI Executor 执行。

## 目标

1. 以一个 Source Catalog 和一个 Scheduler 统一管理固定采集与 AI 采集。
2. Data 成为 Source、Prompt、Schedule、Run、Watermark 和 Receipt 的权威来源。
3. 固定 Connector 与 AI Executor 共享统一的任务和结果合同。
4. Data API、调度控制器、固定采集 Worker 和 AI Executor 能独立部署、扩缩容和故障隔离。
5. 允许逐步恢复并激活现有 Source，不要求一次性重写或启用全部来源。
6. 让一次采集 Run 可重放、可审计、可判定、可恢复，并准确关联其 Source 配置和 Prompt 版本。

## 非目标

- 不使用 Eino 重写 RSS、HTTP、Eastmoney 等确定性采集。
- 不让 `tidewise-ai-agentrun` 维护独立 Scheduler 或 Source Catalog。
- 不让 agent-run 直接访问 Tidewise 数据库。
- 不在 Data API 进程内运行 Connector、长时间 AI 推理或定时任务。
- 不把所有结构化行情强制转换为 Raw Document。
- V1 不要求立即激活全部 202 条 Source。
- V1 不引入人工提示词审核 UI；先冻结领域合同和内部管理能力。

## 领域语言

**Source**：
一个被 Tidewise 治理的数据来源身份。Source 描述数据从哪里来、属于什么业务类型，不表示具体由哪个进程执行。

**Source Category**：
Source 的业务分类，例如 news、market、research。AI 不是 Source Category。

**Connector Key**：
访问 Source 的方式，例如 `rss_feed`、`eastmoney`、`llm_web_research`。

**Executor Kind**：
执行采集任务的运行方式。V1 至少包含 `local_deterministic` 和 `remote_ai`。

**Source Execution Profile**：
某个 Source 当前期望执行配置的版本化定义，包含 Connector、Parser、Executor、Schedule、限制、增量策略、Credential 引用和 Prompt Version 引用。

**Collection Prompt**：
定义 AI 采集“采什么”的业务提示词，包括目标、范围、排除项、来源偏好和质量策略。由 Data Collection 拥有。

**Executor Protocol Prompt**：
定义 AI Executor “如何安全执行”的稳定约束，例如输出 Schema、工具协议、禁止伪造来源和日志脱敏。由 AI Executor 实现拥有，不包含具体业务采集目标。

**Collection Run**：
Data Scheduler 针对一个 Source Execution Profile 创建的一次不可变执行实例。

**Collection Attempt**：
Data 对同一个 Collection Run 发起的一次执行尝试，以从 1 开始的序号区分。每个 Remote AI Collection Attempt 对应一个新的 Agent Execution；它不等同于 AgentRun 内部的 Worker 接管记录。

**Run Spec**：
Collection Run 的不可变任务快照。它冻结本次执行使用的 Source 配置、Prompt、变量、期限、预算和输出合同。

**Run Outcome**：
Executor 返回的结构化执行结果，包含状态、计数、错误分类、Watermark 建议和 Receipt 引用。

**Candidate**：
Executor 发现但尚未被 Data 正式接纳的候选数据。Candidate 经受控导入 Interface 校验后才能成为正式数据事实。

**Watermark**：
某个 Source 已成功处理到的位置，例如发布时间、游标、ETag、Last-Modified 或行情时间点。

## 所有权

### Tidewise Data Collection 拥有

- Source Catalog 与 Source 状态。
- Source Execution Profile 及其版本。
- Schedule Policy 和全局优先级。
- Collection Prompt 与不可变 Prompt Version。
- Collection Run、Lease、Attempt、Heartbeat、Watermark 和最终状态。
- 固定 Connector、Parser 及其能力注册表。
- Candidate 的受控导入、校验、幂等、事务和 Receipt。
- Connector 和 AI Executor 的调度、并发限制和成功判定。
- 正式 Raw Document、Event 和结构化市场数据事实。

### `tidewise-ai-agentrun` 拥有

- Agent Definition、Agent Version、Agent Execution、Attempt 和执行状态。
- LLM Provider Profile、Connector Profile 和加密 Credential。
- Agent Execution 使用的不可变运行配置快照和 Delivery Outbox。
- 一次 AI Run 内部的 Eino Workflow。
- Query Planner、八 Connector 的有界并行和 Connector 级失败隔离。
- AI Run 内部的搜索预算、模型调用和 checkpoint。
- 搜索及专业财经 Connector 的调用实现。
- Executor Protocol Prompt 和输出 Schema 适配。
- 对 Data 下发 Run Spec 的校验和执行。
- Candidate Batch 与 Run Outcome 的回传。

### `tidewise-ai-agentrun` 不拥有

- 业务 Collection Prompt。
- Source 的启停状态和长期身份。
- 定时计划和下一次运行时间。
- 跨 Run 的全局优先级与最终重试策略。
- 正式 Watermark 和 Data Collection Run 账本；agent-run 只保存自己的 Agent Execution 账本。
- Candidate 是否成为 Tidewise 正式数据事实的最终决定。

## 部署形态

统一采集中心是逻辑归属，不是单进程单体。至少拆分为以下独立运行入口：

| 运行入口 | 主要职责 |
|---|---|
| `data-api` | Source/Prompt 管理 Interface、Run 查询、Candidate 导入、Receipt 和正式事实读取 |
| `collection-controller` | 到期计算、创建 Run、Lease、派发、重试、超时、状态回收和告警 |
| `collection-worker` | 执行 RSS、RSSHub、HTTP、Eastmoney、Local Backfill 等固定 Connector |
| `tidewise-ai-agentrun` | 作为远程 AI Executor 执行 Eino AI 采集任务 |

以上运行入口可以位于两个仓库，但不得共享进程内状态或跨仓库直接访问数据库。Data 内部运行入口可以共享同一 Go module 和领域实现。

### PostgreSQL 隔离

agent-run 的权威运行数据库固定命名为 `tidewise_ai_server`。允许与 Tidewise Data 共用 PostgreSQL instance，以降低初期运维成本，但必须满足：

- 使用独立 database 和最小权限数据库用户。
- agent-run 数据库用户没有 Tidewise Data 数据库的连接或对象权限，Data 用户也没有 agent-run 数据库权限。
- agent-run migration 只作用于 `tidewise_ai_server`，不得加入 Tidewise Data migration 序列。
- 两侧不使用跨库 SQL、FDW、`dblink`、共享 schema 或逻辑复制形成运行依赖。
- 数据交换只经过认证、版本化 HTTP Interface；同一 instance 不改变这一 Seam。
- 两个数据库能够独立备份、恢复、扩容和迁移到不同 instance。

## 目标运行流

1. `collection-controller` 查询到期且有效的 Source Execution Profile。
2. Controller 获取 Source 级 Lease，创建 Collection Run 和首次 Attempt。
3. Controller 冻结 Source 配置、Prompt Version 和变量，生成 Run Spec。
4. Controller 根据 `executor_kind` 选择 Executor Adapter：
   - `local_deterministic`：派发给 `collection-worker`。
   - `remote_ai`：派发给 `tidewise-ai-agentrun`。
5. Executor 在自身执行账本中更新 Heartbeat，并在 Deadline 内产生 Candidate Batch；Data 的 Attempt Heartbeat 由 Remote AI Executor Adapter 根据 AgentRun 查询结果更新，两个 Heartbeat 不共享数据库记录。
6. Executor 通过 Data Import Interface 发布 Candidate，不直接写数据库。
7. Data 按严格批次合同返回 Receipt 和逐项 `created/reused`；非法批次整体失败，不返回部分接纳。
8. Executor 提交 Run Outcome。
9. Controller 根据 Outcome 和 Receipt 确定最终状态；只有满足成功条件时才推进 Watermark。
10. Controller 释放 Lease，并计算下一次执行时间或 Retry 时间。

## 核心 Module 与 Seam

### Run Coordinator

Run Coordinator 是调度实现使用的深 Module。外部 Interface 只暴露一次执行所需的最小能力：

```go
type RunCoordinator interface {
    Execute(ctx context.Context, runID string) (RunOutcome, error)
}
```

它在 Implementation 内部完成 Run Spec 加载、Executor 选择、Deadline、Heartbeat、Candidate 发布、结果判定和 Watermark 仲裁。调用者不理解具体 Connector 或 Eino Workflow。

### Executor Seam

```go
type Executor interface {
    Execute(ctx context.Context, spec RunSpec) (ExecutionResult, error)
}
```

生产 Adapter 至少包括：

- RSS Executor Adapter。
- Eastmoney Executor Adapter。
- HTTP/RSSHub Executor Adapter。
- Market Provider Executor Adapter。
- Remote AI Executor Adapter。

测试使用 In-Memory Executor Adapter。Scheduler 不通过类型判断了解各 Adapter 的内部参数。

### Candidate Sink Seam

```go
type CandidateSink interface {
    Publish(ctx context.Context, batch CandidateBatch) (ImportReceipt, error)
}
```

固定 Worker 和远程 AI Executor 都必须通过该 Interface 进入 Data。生产使用 Data HTTP Adapter；Data 内部测试使用 In-Memory Adapter。

## Source Execution Profile V1

每个 Profile 必须版本化且不可原地改变已经创建的 Run：

```yaml
source_id: 53f5f2fe-5317-4cac-b456-8b938f21b62e
config_version: 18
source_category: news
connector_key: llm_web_research
parser_key: llm_research_items
executor_kind: remote_ai
status: active

schedule:
  cron: "0 */2 * * *"
  timezone: Asia/Shanghai
  priority: 50

limits:
  timeout_seconds: 600
  max_items: 50
  max_pages: 1
  max_attempts: 3

incremental:
  watermark_type: published_at
  overlap_window_seconds: 3600
  backfill_policy: bounded

prompt_version_id: prompt-version-uuid
agent_execution_profile_key: collector-full-v1

execution_config: {}
quality_policy: {}
```

Source 的有效运行状态按以下条件计算：

```text
effective_active =
  source_active
  AND profile_valid
  AND executor_supported
  AND credentials_ready
  AND prompt_published
```

Catalog 中的 `active` 不单独构成可执行条件。

## Prompt Registry

### 当前差距

当前 `source_catalogs.source_config` 已保存 `prompt_ref`、`prompt_version` 和 `prompt_variables`，但 `prompt_ref` 指向 `agent-run://...`；完整业务提示词仍由 agent-run 本地文件维护。Data Source Metadata 只返回引用和版本，尚未提供完整、不可变的 Prompt Snapshot。

### Prompt 数据模型

`collection_prompts` 保存逻辑身份：

```text
id, key, name, purpose, status, created_at, updated_at
```

`collection_prompt_versions` 保存不可变版本：

```text
id, prompt_id, version, template_content, variables_schema,
content_hash, status, created_by, created_at, published_at
```

Prompt Version 状态至少包含：

```text
draft -> testing -> published -> retired
```

已经 published 的版本不可修改正文、变量 Schema 或 Hash。修改必须创建新版本。

### Prompt 分层

Data 下发的 Collection Prompt 包含业务目标；agent-run 的 Executor Protocol Prompt 只包含稳定执行协议。两者在 agent-run 内组合，但不得相互覆盖：

- Collection Prompt 不得改变输出 Schema、身份、Deadline、安全和工具权限。
- Executor Protocol Prompt 不得内置具体业务主题、来源偏好或时间窗口。

### Run Prompt Snapshot

创建 Collection Run 时必须冻结：

- `prompt_id`
- `prompt_version`
- `template_content` 或不可变内容地址
- 解析后的 variables
- `content_hash`

运行中的 Prompt 修改不得影响已有 Run。Run Outcome 和 Receipt 审计信息必须能追溯实际 Prompt Version、模型和 Connector 版本。

## Run Spec V1

`collector_run_request.v1` 是 Remote AI Collector 的唯一线格式，也是本节所称的 `collection_run_spec.v1`。Data 在创建 Collection Attempt 时生成完整快照并随 POST 推送；AgentRun 不使用 `run_id` 回调拉取，不存在第二套字段不同的 Schema。

```json
{
  "schema_version": "collector_run_request.v1",
  "collection_run_id": "run-uuid",
  "collection_attempt": 1,
  "source": {
    "source_id": "53f5f2fe-5317-4cac-b456-8b938f21b62e",
    "source_config_version": 18,
    "ingest_channel": "ai_web_research",
    "source_type": "news"
  },
  "agent_version": "collector.v1",
  "created_at": "2026-07-22T10:00:00Z",
  "deadline": "2026-07-22T10:10:00Z",
  "prompt": {
    "prompt_id": "prompt-uuid",
    "prompt_version": "v3",
    "content": "采集业务提示词",
    "variables": {},
    "content_hash": "sha256:..."
  },
  "execution_profile": {
    "llm_profile_key": "collector-planner-primary",
    "connector_profile_set_key": "collector-full-v1"
  },
  "limits": {
    "max_candidates": 80
  },
  "quality_policy": {},
  "output_contract": "raw_document_import.v1",
  "result_target": "tidewise-data-primary"
}
```

Data 不得在 Run Spec 中发送模型名、Provider/Connector URL、Credential、enabled Connector 清单、单 Connector candidate limit、并发或 rate limit。这些由 AgentRun 解析逻辑 Profile Key 后冻结。相同服务身份、相同 `(collection_run_id, collection_attempt)` 和相同 payload 是幂等重放；不同 payload 为 409 conflict。

## Run Outcome V1

最终状态至少包括：

- `succeeded`
- `succeeded_no_change`
- `partially_succeeded`
- `retryable_failed`
- `permanently_failed`
- `configuration_failed`
- `cancelled`
- `timed_out`

最小结果：

```json
{
  "schema_version": "collection_run_outcome.v1",
  "collection_run_id": "run-uuid",
  "collection_attempt": 1,
  "agent_execution_id": "execution-uuid",
  "status": "succeeded",
  "items_discovered": 42,
  "items_published": 31,
  "items_reused": 8,
  "candidate_terminal_counts": {
    "accepted": 39,
    "known_url": 1,
    "out_of_window": 1,
    "invalid_result": 1,
    "exact_duplicate": 0,
    "near_duplicate": 0
  },
  "connector_invocations": [],
  "receipt_ids": ["receipt-uuid"],
  "provider_failures": [],
  "watermark_candidate": {},
  "started_at": "2026-07-22T10:00:01Z",
  "finished_at": "2026-07-22T10:03:00Z"
}
```

AgentRun 使用 Outbox POST：

```http
POST /internal/data/v1/collection-runs/{collection_run_id}/attempts/{collection_attempt}/outcome
```

Data 以调用主体和 `(collection_run_id, collection_attempt)` 幂等，并校验 `agent_execution_id`。八个 V1 Connector 均为 best-effort，但至少一个必须完成；部分失败且至少一个完成、全部 Candidate 终态、全部 accepted Candidate 已获 Receipt 时为 `partially_succeeded`。所有 Provider/Connector 都失败、未取得必需 Receipt、`results_pending != 0` 或输出违反合同，不得返回成功语义。

## 调度、Lease 与重试

- 同一 Source 同一时间默认只能有一个有效 Lease。
- Lease 必须有过期时间并支持崩溃后接管。
- Attempt 是同一个 Run 的重试，不为每次重试创建新的业务 Run 身份。
- 每个 Collection Attempt 创建一个新的 Agent Execution；Data 不复用上一 Attempt 的 `execution_id`。
- Controller 决定跨 Attempt 的重试和退避。Collector V1 的 Connector 外部调用固定 `max_attempts=1`；AgentRun 只重试幂等且可对账的 Outbox 投递。
- Heartbeat 只证明 Executor 存活，不表示业务成功。
- Watermark 只能在符合 Source Quality Policy 的最终成功后推进。
- `partially_succeeded` 是否推进 Watermark由 Profile 明确规定，默认不推进。
- Scheduler 必须按 Executor/Provider 使用隔离并发池，至少区分 RSS、Eastmoney、Market Provider 和 AI。

## 数据导入通路

### 内容型数据

RSS、网页、新闻、研报、搜索结果和 AI 调研材料继续使用 Raw Document Import；从中提炼的 Event 通过 Event Import 进入正式事实。

AgentRun 复用 `POST /internal/data/v1/raw-document-imports` 和按 idempotency key 的 GET 查询。每批必须同时满足 `<=100` items 和完整 UTF-8 JSON body `<=1,048,576` bytes；幂等键为 `collector:{agent_execution_id}:raw:{batch_number}`。Raw Import 是整批严格事务，成功只产生 `created/reused`；超时或未知响应必须先用原 key 查询，禁止换 key 重发。AgentRun 的 `invalid_result` 等是建批前的本地 Candidate 终态，不是 Data 的逐项拒绝结果。

### 结构化市场数据

K 线、指数点位、证券列表、板块成分和结构化财务指标不得长期以 Raw Document 作为唯一正式模型。启用对应 Source 前，应定义专用的结构化导入合同，例如：

- Market Observation Import。
- Instrument Import。
- Board Constituent Import。
- Corporate Metric Import。

## 安全

- Data 只保存和下发 agent-run 的逻辑 Execution Profile Key，不保存 LLM 或搜索 Connector 的凭证值。
- agent-run 在 `tidewise_ai_server` 中保存 LLM/Connector Profile 和加密 Credential；解密主密钥必须位于数据库之外。
- Profile 的 Base URL、模型、Endpoint、限制和 Credential 引用在创建 Agent Execution 时冻结为非敏感快照，解密后的密钥不得进入快照。
- Credential 轮换创建不可变 Credential Version。每版本使用随机 DEK 和 AES-256-GCM；DEK 由数据库外的版本化 KEK 包装，数据库不得保存 KEK。
- Run Spec 只能包含 allowlisted 非敏感配置。
- Prompt、模型原始响应和密钥不得写入普通错误日志。
- agent-run 只能执行其被授权的 Run Spec，并只能提交对应 `collection_run_id/collection_attempt/source_id` 的结果。
- V1 Run Spec 由 Data 主动推送；上一条中的“读取”仅指校验请求内容，不授权 AgentRun 回调 Data 拉取 Prompt 或访问 Data DB。
- Data 必须验证发布主体、Run 所有权、Source 身份和 Output Contract。

## 可观测性

至少采集以下指标：

- Source 到期延迟和调度积压。
- Run 各状态数量、持续时间和重试次数。
- Lease 过期和重复派发。
- 每个 Connector/Provider 的成功率、延迟、限流和熔断状态。
- discovered、published、reused、rejected 数量。
- Source freshness 和 Watermark 落后时间。
- AI Run 的模型调用量、token、搜索次数和单位有效 Candidate 成本。
- Prompt Version 的有效来源率、重复率、拒绝率和失败率。

所有日志和追踪至少关联 `collection_run_id`、`collection_attempt`、`source_id`、`source_config_version`；AI Run 额外关联 `agent_execution_id` 和 `prompt_version`。

## 迁移策略

### 控制合同

- 用新 ADR 取代 ADR-0002 中采集 ownership 的局部决策。
- 更新 Data Context 的 Owns/Does Not Own 和 External Agent 关系。
- 冻结 Source Execution Profile、Run Spec、Run Outcome 和 Executor Interface。

### 采集控制能力

- 建立 Prompt Registry、Schedule、Run、Attempt、Lease 和 Watermark 数据模型。
- 建立 Controller 与 Worker 独立运行入口。
- 建立固定 Executor 和 Remote AI Executor 的能力注册表。

### 固定采集恢复

- 从历史实现迁移 RSS、RSSHub、HTTP、Local Backfill、Eastmoney 和 Parser 的有效逻辑与测试。
- 不恢复旧 Scheduler 的包结构和数据库直耦合。
- RSS 按少量 Source、分组、全量三个阶段灰度激活。

### AI Executor 接入

- 建立独立 `tidewise_ai_server` 数据库、agent-run migration 和最小权限数据库身份。
- 建立 Agent Definition/Version、Agent Execution/Attempt、LLM Provider Profile、Connector Profile、加密 Credential 和 Delivery Outbox。
- 暴露独立 Collector Agent Run Interface，并在内部复用通用 Agent Execution Core。
- 完整迁移 Codex 原始采集的 Connector fan-out、直接结果、合并、时间门禁、URL/正文/SimHash 去重和全结果终态语义。
- agent-run 删除本地业务 `--prompt-file` 作为事实来源的机制。
- agent-run 接收 Data 随创建请求推送的完整不可变 Run Spec，不回调拉取。
- 保留 Executor Protocol Prompt，并与 Collection Prompt 分层组合。
- Candidate 通过 Data Import Interface 发布，文件输出降为可选审计 Artifact。

### 结构化来源激活

- 先冻结对应结构化导入合同，再逐个启用 Eastmoney 和 Market Provider Source。
- `market_provider` Catalog 记录在 Adapter 和合同通过验收前保持 inactive。

## 整体验收标准

- 固定 Worker 与 agent-run 能消费相同版本的 Run Spec，并返回相同结构的 Run Outcome。
- Source、Profile、Prompt、Run、Attempt、Receipt 和正式数据可以端到端追溯。
- 同一 Source 的重复调度不会产生并发双跑；Worker 崩溃后 Lease 可安全恢复。
- 相同 Candidate 重放不会重复创建正式数据。
- 已创建 Run 不受后续 Source 配置或 Prompt 修改影响。
- agent-run 不再维护业务采集提示词或独立 Schedule。
- 所有 AI Provider 失败时 Run 明确失败。
- Data API 在 Connector 阻塞、限流或 AI 超时时仍可独立提供服务。
- 固定 Connector 与 AI Executor 可以独立扩缩容和停用。
- 结构化行情不会被迫以 Raw Document 作为最终事实模型。
- 两仓共享 `collector_run_request.v1`、Raw Import consumer 和 `collection_run_outcome.v1` golden fixtures；Data 侧没有第二套漂移 Schema。
- Collector V1 八个 Adapter（Parallel、Tavily、Bocha、Brave live search、财联社电报、东方财富快讯、东方财富个股新闻、证券时报快讯）都通过合同测试；Brave 未配置或未 smoke 时不得宣称 production-ready，也不得宣称与 Codex 索引/排序等价。

## 与当前文档和实现的冲突

1. `docs/contexts/data/CONTEXT.md` 当前明确写明 Data 不拥有 connector、parser、采集 prompt 和调度执行。
2. ADR-0002 当前明确将以上职责全部交给外部 agent-run。
3. Data 当前仅在 `source_catalogs.source_config` 保存 Prompt 引用、版本和变量，没有 Prompt Registry 和不可变 Prompt Version。
4. Source Metadata Interface 只返回少量 allowlisted 配置，不足以构造完整 Run Spec。
5. agent-run 当前从本地 `query_planner_v1.md` 读取业务提示词，没有 Data Run Spec Adapter。
6. 当前 Data API、Controller、Worker 和远程 Executor 尚未形成独立运行入口和统一 Executor Interface。

在新 ADR 被接受并完成 Data Context 修订前，本 Spec 只表示 Proposed 目标，不得把现有实现描述为已经符合该架构。

## 已关闭选择与跨仓交付

- AgentRun 队列：自身 PostgreSQL lease，不使用外部消息队列。
- Data Prompt Registry V1：migration、Repository、内部 Admin CRUD/发布 Interface 均需交付；published 版本不可修改。
- Run Spec：完整快照随 POST 推送。
- Live Search：Brave Web Search Adapter；行为对齐，不称结果等价。
- Credential：AgentRun 应用层 envelope encryption，外部版本化 KEK。

Data 仓库必须以独立 Change/PR 交付 ADR/Context 修订、Prompt Registry、Collection Run/Attempt/Lease、Remote AI Executor Adapter、Outcome Interface、AgentRun service principal 和共享 golden fixtures。AgentRun 仓库以 fake Data server 完成纵切测试；在 Data Change 合并、两侧 migration/身份/Profile/Credential 配置及八 Connector smoke 完成前，端到端上线验收不成立。两仓任何实现都不得以测试便利为由访问对方数据库。
