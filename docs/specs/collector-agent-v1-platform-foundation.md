# Collector Agent V1 与 AgentRun Platform Foundation Spec

状态：Ready for implementation
日期：2026-07-22
Issue：#13
涉及系统：`tidewise-ai-agentrun`

## Problem Statement

Tidewise Data Service 需要通过稳定的 HTTP 接口异步驱动原始资讯采集，但当前 AgentRun 只有本地 Collector CLI：调用方无法创建、幂等重放或查询一次采集执行，执行状态和 Connector 结果也没有持久化控制面。现有 Collector 只接入 DeepSeek 查询规划与 Parallel、Tavily、Bocha 三个搜索通道，尚未覆盖当前 Codex 原始信息采集合同中的专业财经资讯流。

本阶段还需要建立足以承载后续 Event Extractor Agent、Analyst Agent 的最小通用 Agent 执行模型，但不能提前建设 Tidewise Data 侧能力、跨服务数据发布、回调、队列恢复或通用 Agentic 运行时。

## Solution

在 AgentRun 仓库中交付 Collector Agent V1 的单服务纵切：Data Service 或运维客户端以一段完整自然语言 Collection Prompt 调用内部 HTTP API，AgentRun 创建一个可查询的 Agent Execution，并在当前进程中异步运行 Eino Workflow。DeepSeek 只理解 Prompt、生成搜索 queries、组合查询和可选时间窗口；Go 程序固定执行七个 Connector，确定性完成结果解析、URL 合并、时间门禁、SHA-256/SimHash64 去重、Candidate 终态判定和本地 Artifact 写入。

AgentRun 使用独立 PostgreSQL database `tidewise_ai_server` 保存执行控制面和当前 Provider 配置，使用持久化本地 Artifact Volume 保存采集数据。V1 不调用 Tidewise Data Raw Document Import，也不回调 Run Outcome；后续 Tidewise Data 仓库任务通过本 Spec 的 HTTP 合同和本地 Artifact 合同完成跨服务集成。

## User Stories

1. As Tidewise Data Service, I want to submit one complete natural-language Collection Prompt, so that I do not need to know models, providers, connectors, concurrency, or collection mechanics.
2. As Tidewise Data Service, I want Collector creation to return immediately with an Execution ID, so that collection runs asynchronously.
3. As Tidewise Data Service, I want to poll an Execution by ID, so that I can observe progress and terminal outcome without a callback.
4. As Tidewise Data Service, I want a required idempotency key, so that transport retries do not create duplicate executions.
5. As Tidewise Data Service, I want reuse of an idempotency key with a different Prompt rejected, so that a submitted task cannot silently change meaning.
6. As Tidewise Data Service, I want the exact Prompt preserved with a hash, so that execution intent is auditable without returning it through status APIs.
7. As an operator, I want at most one active Collector Execution, so that this MVP does not overrun external quotas or local artifacts.
8. As an operator, I want a clear conflict response containing the active Execution ID, so that I can inspect the work already in progress.
9. As an operator, I want health and readiness endpoints, so that process liveness is distinct from database, schema, authentication, and Provider configuration readiness.
10. As an operator, I want explicit database migrations, so that service startup never mutates production schema implicitly.
11. As an operator, I want a bootstrap CLI for current Provider configuration, so that the later Admin Portal has a database-backed configuration source from day one.
12. As a security-conscious operator, I want API, CLI, logs, errors, snapshots, and Artifacts never to reveal Provider keys, even though this dev/UAT MVP stores them in plaintext.
13. As a deployment owner, I want plaintext Provider storage to make production readiness fail, so that the MVP security exception cannot be mistaken for production readiness.
14. As an Agent platform owner, I want Agent Definition, immutable Agent Version, and Agent Execution separated, so that later Agents can reuse the execution control plane.
15. As an Agent platform owner, I want the migration to seed `collector` and `collector.v1`, so that every run records an explicit immutable execution contract.
16. As a Collector user, I want DeepSeek to understand long, multiline, Markdown-capable Prompt text, so that collection intent can remain expressive and change frequently outside AgentRun source code.
17. As a Collector user, I want the Planner to generate focused queries and one combined query, so that multi-query and single-query Connectors receive suitable input.
18. As a Collector user, I want an explicitly stated time window interpreted from natural language, so that requests such as “最近一周” affect deterministic time gating.
19. As a Collector user, I want a 48-hour program default when the Prompt does not state a time window, so that missing model output has a stable meaning.
20. As a Collector user, I want all seven fixed Connectors executed once after successful planning, so that the LLM cannot silently reduce source coverage.
21. As a Collector user, I want Parallel, Tavily, Bocha, CLS Telegraph, Eastmoney Fast News, Eastmoney Stock News, and STCN Quick News covered, so that V1 reproduces the applicable Codex collection channels without Codex-only live search.
22. As a Collector user, I want bounded Connector concurrency and fixed timeouts, so that external calls cannot grow without control.
23. As a Collector user, I want one Connector failure isolated from the others, so that useful direct results can still become a partial success.
24. As an auditor, I want exactly seven Connector Invocation records for every Execution, so that attempted, completed, failed, and not-started channels are visible.
25. As a data consumer, I want only direct Connector results, so that the Collector never invents facts or opens result URLs a second time.
26. As a data consumer, I want content origin and content level recorded, so that downstream work can distinguish full text, summary, snippet, and title-only results.
27. As a data steward, I want canonical URLs merged with complete Connector provenance, so that the same page is represented once without losing source-channel attribution.
28. As a data steward, I want exact SHA-256 and conservative SimHash64 near-duplicate detection, so that repeated material is classified without aggressive false-positive deletion.
29. As an auditor, I want every post-merge Candidate to reach a terminal disposition and `results_pending=0`, so that a successful or partial run has no unaccounted Candidate.
30. As an auditor, I want rejected and duplicate Candidate metadata retained without duplicate body storage, so that quality decisions remain explainable while Artifact size stays controlled.
31. As an operator, I want accepted results written as readable Markdown documents, so that collection quality can be inspected locally before Data Import exists.
32. As an operator, I want a per-run manifest, Candidate ledger, and summary, so that counts, reasons, hashes, Connector outcomes, and document paths can be reconciled.
33. As an operator, I want Artifact completion published atomically, so that a query never advertises a half-written completed run.
34. As an operator, I want stale in-process work marked failed after restart, so that an old Execution does not remain indefinitely active even though V1 does not recover it.
35. As a maintainer, I want one black-box HTTP test seam using real AgentRun behavior with fake external providers, so that the service contract is verified without a fake Tidewise Data service.
36. As a maintainer, I want real Provider smoke collection to be optional and honestly reported, so that missing credentials or external outages do not make deterministic tests unreliable.

## Implementation Decisions

### Eino reference audit

This design completed the repository's Eino reference-first gate against the shared read-only clones in the main checkout:

- `cloudwego/eino` at `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`: inspected `compose/doc.go`, `compose/workflow.go`, the typed Runnable and Graph surfaces, and the ADK Agent/Workflow interfaces. Adopted typed `compose.Workflow`, explicit dependency edges, one compiled Runnable per Collector execution, and ordinary Go functions behind Lambda nodes. Rejected ADK Workflow Agent, Agent transfer, tool calling, checkpoint/resume, and shared Agent conversation state because Collector V1 is a deterministic application workflow rather than an autonomous or conversational multi-Agent system.
- `cloudwego/eino-ext` at `9137edd89e72b72735ede69db1c5ae29178a6e41` (`components/model/deepseek/v0.1.7`): inspected the DeepSeek README, constructor, configuration, Generate path, structured response option, and tests/examples. Adopted the official `ChatModel` adapter, explicit Base URL/model/key/timeout mapping, `ResponseFormatTypeJSONObject`, and the `model.BaseChatModel` seam used by the Planner. Rejected a custom DeepSeek HTTP client and Provider-specific types in the Collector domain model.
- `cloudwego/eino-examples` at `171220631fb7068ead50b7cd964b8c471647117d`: inspected typed Workflow field/dependency examples, parallel ADK workflows, the multi-Agent project-manager example, and the Deer-Go composition/handler/infrastructure split. Adopted explicit composition at the application edge and capability-local orchestration. Rejected example-level global model state, building complex graphs in HTTP handlers, and multi-Agent supervisor patterns for this V1.

Project-specific gaps remain intentionally implemented in AgentRun rather than Eino: HTTP idempotency, Agent Definition/Version/Execution state, Connector Invocation persistence, Provider configuration, fixed Connector policy, timeouts and bounded concurrency, Candidate accounting, deduplication, and atomic Artifact publication. Eino remains an orchestration implementation detail inside the Collector capability and does not define the repository's domain structure.

The resulting package direction is capability-first: `internal/agentrun` owns reusable platform execution concepts and persistence adapters, while `internal/collector` owns Collector application orchestration, planning, Workflow, Connectors, Artifacts, and HTTP transport. Future Agent capabilities receive sibling packages instead of adding their behavior to global technical-layer packages.

### Scope and ownership

- This change modifies only AgentRun. It does not modify Tidewise Data, access the Tidewise Data database, or include Tidewise Data code in the PR.
- AgentRun owns Agent Execution, Connector Invocation, current Provider configuration, Candidate processing, and local Raw Document Artifacts.
- Tidewise Data owns its Prompt Registry, Collection Run/Attempt, Remote AI Executor, final Run state, Watermark, and future import/callback integration.
- V1 does not implement AgentRun-to-Data Raw Document Import, Run Outcome callback, Delivery Outbox, or a fake Data server.

### Agent platform model

- The minimum reusable model is `Agent Definition -> immutable Agent Version -> Agent Execution`.
- Migrations seed one `collector` Agent Definition and one immutable `collector.v1` Agent Version. The Collector API always selects the current V1; callers cannot choose a version.
- An Agent Execution stores the selected Agent Version, generated UUID, idempotency key, exact Prompt, Prompt SHA-256, Prompt byte length, status, timestamps, safe error, Candidate counts, and completed Artifact references.
- The generic execution repository must not depend on Collector-specific Candidate or Connector implementations. Future Agents add their own typed handler and Workflow while reusing definitions, versions, executions, authentication, and persistence.
- V1 has no Registry administration API or CLI and no synthetic second Agent.

### HTTP contract

- The versioned endpoints are `POST /internal/agent-run/v1/collector/runs` and `GET /internal/agent-run/v1/collector/runs/{execution_id}`.
- Both endpoints require `Authorization: Bearer <AGENTRUN_SERVICE_TOKEN>`. Missing or incorrect credentials return `401`. Authorization headers are never logged.
- Health and readiness endpoints are anonymous and return no sensitive configuration detail.
- POST requires an `Idempotency-Key` header and JSON containing exactly one business field, `prompt`.
- Prompt accepts UTF-8, newlines, and Markdown. It is rejected when blank after `TrimSpace`, malformed, or larger than 64 KiB. The service preserves the exact accepted text.
- AgentRun generates the Execution UUID and request timestamp. The caller cannot specify time, Agent Version, model, Provider, Connector set, limits, concurrency, or collection method.
- A first valid POST persists the Execution and seven pending Connector Invocations, starts asynchronous in-process work, and returns `202` with schema identifier `collector_run.v1`, Execution ID, current status, creation time, and status URL.
- Replaying the same idempotency key with the same exact Prompt returns the existing Execution representation and does not start work again. Idempotency lookup happens before the active-run check.
- Reusing an idempotency key with different Prompt bytes returns `409 idempotency_conflict`.
- Missing idempotency key returns `400`; unavailable schema or Provider configuration returns `503 configuration_not_ready`.
- Only one Execution may be active in `queued`, `planning`, `collecting`, or `materializing`. A new non-replay POST during that period returns `409 active_execution_exists` with the active Execution ID; V1 has no waiting queue.
- GET returns schema identifier, Agent key/version, Prompt SHA-256 and byte length, Execution status, all seven Connector Invocations, Candidate counts, completed Artifact paths, timestamps, and a stable safe error code/summary. It never returns the full Prompt, Provider keys, or Candidate bodies.
- GET for an unknown UUID returns `404 execution_not_found`. All errors use stable error codes and redact model responses, endpoint bodies, Prompt content, credentials, and Authorization headers.

### Execution state and failure semantics

- Running states are `queued -> planning -> collecting -> materializing`.
- Terminal states are:
  - `succeeded`: all seven Connectors completed and at least one new Candidate was accepted.
  - `succeeded_no_change`: all seven Connectors completed and no Candidate was newly accepted.
  - `partially_succeeded`: at least one Connector completed and at least one failed; Candidate processing and completed Artifacts succeeded. Accepted count may be zero.
  - `failed`: Planner failure, all seven Connectors failed, whole-execution timeout, Artifact failure, database/state failure that prevents completion, or stale work found after process restart.
- Planner failure is fail closed: no Connector HTTP call occurs. The seven precreated Invocation rows become failed with a safe “not invoked because planning failed” code.
- A Connector failure does not cancel other Connector calls. An Invocation records only Connector key, status, start/end times, direct-result count, stable error code, and safe summary; it never stores news bodies.
- Invocation states are `pending -> running -> completed | failed`. Calls not started because of Planner failure or whole-execution timeout end as failed with a distinct not-invoked code.
- V1 does not recover or retry work after process failure. On startup, any persisted Execution in a running state and its incomplete Invocations are marked `failed/process_restarted`.
- The Data caller may create a new task with a new idempotency key. The old key always resolves to the old failed Execution.
- V1 has no cancel endpoint, cancelled state, lease, heartbeat, Worker Attempt, cross-process claim, automatic HTTP retry, or external task queue.

### Provider configuration and database boundary

- AgentRun uses an independent PostgreSQL database named `tidewise_ai_server`, with its own DSN, migrations, connection pool, database user, backup, and restore boundary.
- Sharing a PostgreSQL instance with Tidewise Data is allowed, but shared databases, schemas, business tables, FDW, `dblink`, cross-database SQL, and Tidewise Data DSNs are prohibited.
- PostgreSQL is the only source for current Provider configuration. There is no Provider-key or Provider-endpoint environment fallback.
- Current mutable configuration covers DeepSeek Base URL, model, and key; Parallel, Tavily, and Bocha Base URL and key; and Base URLs for CLS Telegraph, Eastmoney Fast News, Eastmoney Stock News, and STCN Quick News.
- Each Provider has one current row. Updates overwrite current values; V1 does not preserve profile versions, credential versions, rotation history, or configuration history. An active Execution uses the configuration snapshot already loaded in memory; a later Execution reads current values.
- A bootstrap CLI supports schema-aware upsert, list/check, and redacted inspection. It never prints a full key; output may show only configured/not-configured and a masked suffix.
- Provider keys are plaintext in PostgreSQL only for the dev/UAT MVP, as accepted by ADR 0001. HTTP responses, CLI reads, logs, errors, Execution data, Candidate ledgers, manifests, summaries, and documents never contain them.
- Only `dev` and `uat` are supported while plaintext credential storage is active. A future production change must add database-external key management, encrypted storage, and plaintext migration before production can become ready.
- Non-secret runtime configuration is selected by `APP_ENV` and loaded from `internal/agentrun/config/config.dev.yaml` or `config.uat.yaml`. Environment variables are limited to the optional config directory override, database password or complete DSN, and static inbound service token.
- Migrations run only through an explicit migration CLI. Server startup checks schema compatibility but never auto-migrates. The Provider config CLI refuses to operate against an incompatible schema.
- `/healthz` reports process liveness. `/readyz` requires database connectivity, exact supported schema, a non-empty service token, a supported dev/UAT environment, writable Artifact root, and every required Provider URL/model/key. Missing readiness requirements also make Collector POST return `503`.
- The HTTP service port is fixed at `9080` in dev and UAT. Development reuses the existing local PostgreSQL instance while retaining an independent `tidewise_ai_server` database and `agentrun` user; AgentRun does not start another PostgreSQL container. Automated database tests use an isolated temporary database and explicitly reject production and Tidewise Data database names.

### Planner and Eino workflow

- Eino Workflow expresses the typed dependency graph `Planner -> seven-Connector fan-out -> Candidate processing/Artifact materialization -> Execution terminal state`.
- Eino is an orchestration boundary only. The program owns state transitions, connector set, timeouts, concurrency, JSON validation, defaults, time gating, merge, deduplication, terminal accounting, database writes, and atomic files.
- V1 does not use AgenticModel or tool calling. The LLM never chooses, skips, or directly calls a Connector.
- DeepSeek receives the exact Collection Prompt and request timestamp. Its only task is semantic query planning.
- The strict Planner output is one JSON object containing non-empty `queries`, non-empty `combined_query`, and optional integer `time_window_hours`. Unknown fields, trailing JSON/content, wrong types, blank values, or malformed JSON are rejected.
- The Planner may return at most 12 stable, trimmed, exact-deduplicated queries; each query and `combined_query` is at most 256 Unicode code points.
- When the Prompt explicitly expresses a time window, DeepSeek may return `time_window_hours`. The program accepts only 1 through 8760 hours. If it is absent, the program uses 48 hours.
- The program computes the cutoff from the AgentRun request timestamp in UTC. It does not maintain a natural-language time parser.
- The minimal machine-output instruction needed to request the JSON contract is fixed in code. AgentRun has no local business Collection Prompt file and no separately maintained Executor Protocol Prompt artifact.
- DeepSeek does not read Connector results, generate facts, judge relevance, identify spam, verify evidence, assign content level, summarize, translate, rewrite, deduplicate, extract Events, or perform investment analysis.

### Fixed execution guardrails

- Planner timeout is 30 seconds.
- Each Connector timeout is 30 seconds.
- Whole Execution timeout is 5 minutes.
- At most three Connector calls run concurrently.
- Planner output contains at most 12 queries, each at most 256 Unicode code points.
- Tavily retains at most 5 direct results. Parallel, Bocha, and each professional finance Connector retain at most 10 direct results.
- These values belong to immutable `collector.v1`; they are not database configuration.
- V1 performs no automatic retry or Provider fallback for DeepSeek or Connector HTTP calls.

### Connector contract

- After successful planning, the fixed Connector set is exactly: Parallel, Tavily, Bocha, CLS Telegraph, Eastmoney Fast News, Eastmoney Stock News, and STCN Quick News.
- Every Connector is represented by exactly one Workflow node and exactly one Invocation record. Every Connector is called once per Execution unless planning or an execution-wide failure prevents invocation.
- Codex-only `live_search` is excluded. V1 has no Brave replacement, placeholder adapter, or equivalence claim.
- Parallel receives the complete `queries` array in one request and stores direct excerpts.
- Tavily receives `combined_query` once with fixed finance topic, advanced search depth, three chunks per source, at most five results, and date bounds derived from the time window. It does not enable auto parameters or raw content and stores only direct result content snippets.
- Bocha receives `combined_query` once with summary enabled and a freshness value derived deterministically from the time window. It prefers direct summary and otherwise uses direct snippet.
- Eastmoney Stock News receives `combined_query` once.
- CLS Telegraph, Eastmoney Fast News, and STCN Quick News each call their fixed feed once and require no query.
- Professional finance Connectors store only direct content, summary, brief, or title supplied by their API. No Connector opens a result URL, PDF, attachment, or secondary page.
- Every normalized result records `content_origin=connector_response` and one of `full_text`, `summary`, `snippet`, or `title_only`.
- Fixed feeds may return window-valid finance results weakly related to the Collection Prompt. V1 accepts them when structurally valid and nonduplicate; it adds no keyword, semantic relevance, spam, credibility, or evidence-quality gate.

### Candidate processing and Artifact contract

- A Connector direct result is normalized and grouped by canonical URL. One post-merge Collection Candidate represents each canonical URL group and carries the stable sorted set of contributing Connector keys plus one primary Connector.
- Canonicalization lowercases scheme/host, removes fragments and known tracking parameters, normalizes the remaining query, and removes a non-root trailing slash.
- Within one canonical URL group, content selection uses `full_text > summary > snippet > title_only`; at equal level the longer non-empty direct content wins. Bodies are never concatenated and facts are never synthesized.
- Every post-merge Candidate reaches exactly one terminal disposition: `accepted`, `known_url`, `out_of_window`, `invalid_result`, `exact_duplicate`, or `near_duplicate`.
- A result without a parseable published timestamp may be accepted using collection time as its time basis. A parseable timestamp outside the inclusive request window is `out_of_window`.
- Exact deduplication uses SHA-256 of the normalized Markdown body. Near deduplication uses SimHash64 with BLAKE2b-derived token hashes and Hamming distance at most 3. This conservative radius intentionally preserves the current AgentRun behavior instead of the earlier proposed radius 10.
- `results_pending` must be zero before `succeeded`, `succeeded_no_change`, or `partially_succeeded` is persisted.
- PostgreSQL does not contain Candidate rows or news bodies. The per-run Candidate ledger stores terminal metadata and reason; non-accepted entries do not duplicate body content. Accepted entries refer to their Markdown Artifact.
- The persistent Artifact root contains accepted Markdown documents partitioned by date, a per-Execution directory containing `candidates.jsonl`, `manifest.json`, and `summary.md`, and a shared `dedup-index.tsv`.
- Accepted Markdown preserves the direct selected content and provenance under schema `connector_result_md.v1`.
- The manifest records schema/version, Execution and Agent identity, Prompt SHA-256 and byte length, effective time window, Connector statistics, Candidate terminal counts, `results_pending`, accepted Artifact paths and hashes, safe errors, and start/end timestamps. It contains no full Prompt or Provider secret.
- The summary is human-readable and derived from the manifest. The dedup index stores durable URL/content fingerprints and document paths for later AgentRun executions.
- Files are written via same-directory temporary files and atomic rename. `manifest.json` is published last and acts as the completed Artifact marker. GET exposes completed Artifact paths only after manifest publication.
- V1 assumes one AgentRun instance or a shared persistent Artifact Volume. The single-active-Execution rule avoids concurrent index writers.

## Testing Decisions

- The primary seam is a black-box HTTP test that drives the real AgentRun HTTP Handler/Server, real repositories, real Eino workflow, and real Artifact writer against an isolated PostgreSQL test database, Fake DeepSeek, seven Fake Connector HTTP servers, and a temporary Artifact root.
- Tests call AgentRun directly. There is no fake Tidewise Data server because AgentRun performs no outbound Data request in this scope.
- Black-box tests cover Bearer authentication, Prompt validation and preservation, idempotent replay/conflict, active-run conflict, asynchronous status transitions, GET redaction, unknown ID, readiness failures, fixed seven-Connector calls, bounded fan-out, and completed Artifact visibility.
- Outcome matrices cover Planner failure with zero Connector calls, seven completed with accepted results, seven completed with no change, mixed completed/failed with and without accepted results, all seven failed, execution timeout, Artifact failure, and process-start stale-state marking.
- Every successful-class test asserts seven Invocation records, terminal post-merge Candidate accounting, `results_pending=0`, manifest-last publication, and no secrets or full Prompt in observable output.
- Focused tests cover strict Planner JSON and trailing-content rejection, query limits, optional/default time window, Connector request mapping, Connector parsing, URL canonicalization, content-level merge, time gating, SHA-256, BLAKE2b-based SimHash64 with radius 3, Candidate dispositions, atomic file contracts, configuration redaction, and database-name isolation.
- Persistence integration tests run migrations in an isolated temporary database and cover seeded Agent/Version, idempotency uniqueness, active-run exclusion, seven Invocation rows, startup stale-state failure, current Provider upsert, and schema-readiness checks.
- Tests assert external behavior and durable contracts rather than Eino node internals or private helper implementation.
- TDD is used at the agreed HTTP seam and focused deterministic seams: capture a meaningful failing test, implement the minimum behavior, then refactor.
- The final automated gate includes formatting, the complete Go test suite, race-sensitive concurrency tests where practical, migration verification, diff checks, credential scans, and a Standards/Spec code review.
- If all real Provider configurations are locally available, pre-merge acceptance also performs one manual network collection and inspects its manifest, summary, Candidate ledger, and accepted Markdown. This smoke is reported honestly but is not an automated gate; missing credentials or external outages do not permit a false pass and do not block deterministic code acceptance.

## Out of Scope

- Any modification to the Tidewise Data repository or direct access to its database.
- Tidewise Data Prompt Registry, Collection Run/Attempt, Remote AI Executor, Scheduler, Source Catalog, Watermark, Outcome endpoint, or dual-service integration.
- AgentRun-to-Data Raw Document Import, Run Outcome callback, Delivery Outbox, batching, receipt reconciliation, or callback retry.
- Candidate body/query API, Candidate PostgreSQL table, or formal Tidewise Raw Document persistence.
- Cancellation, waiting queue, multi-instance worker claims, PostgreSQL job queue, lease, heartbeat, Worker Attempt, crash recovery, or automatic replay after restart.
- Admin HTTP API, configuration UI, Profile Version, Credential Version, rotation history, or configuration audit history.
- Production credential storage, encryption master key, envelope encryption, or a production-ready security claim.
- Codex `live_search`, Brave or another replacement, secondary URL fetching, PDF/attachment fetching, multi-round search, tool calling, AgenticModel, Provider fallback, or automatic external-call retry.
- LLM result review, relevance/spam filtering, evidence verification, factual scoring, summarization, translation, rewriting, Event extraction, industrial-chain analysis, or investment analysis.
- Runtime configuration of timeouts, result limits, query limits, concurrency, or rate limits.
- A test-only second Agent or Registry management surface.

## Further Notes

- This Spec is the authoritative implementation contract for Issue #13 and supersedes the earlier draft that included eight Connectors, Brave, Data publishing, callback/outbox, cancellation, encrypted versioned credentials, and crash-recovery queues.
- `collector.v1` is intentionally a deterministic workflow with a narrow LLM planning role. “Agent” does not mean the model can alter source coverage or execution policy.
- Plaintext Provider storage is a conscious dev/UAT exception recorded in ADR 0001, not a recommended final architecture.
- The later Tidewise Data change may rely only on the versioned HTTP response and completed Artifact contract described here; it must not couple to AgentRun tables or query `tidewise_ai_server` directly.
