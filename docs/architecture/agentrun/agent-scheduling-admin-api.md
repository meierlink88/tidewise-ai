# Agent Scheduling 与 Admin API

状态：Ready for Agent
日期：2026-07-24
涉及系统：`tidewise-ai-agentrun`
实现工单：[GitHub Issue #23](https://github.com/meierlink88/tidewise-ai-agentrun/issues/23)

## Problem Statement

AgentRun 当前只能由外部调用方按次创建 Collector Agent Execution，缺少平台拥有的周期性触发能力。Admin Portal Service 也没有 HTTP 管理面来启停 Agent Schedule、设定运行周期、分页查询执行历史，或维护当前模型和 Connector 配置。

现有 Model Provider Configuration 与 Connector Configuration 已分表保存，但 AgentRun 只在启动时加载一次，CLI 修改配置后必须重启。现有 Collector HTTP 路径包含 `/internal/agent-run` 部署含义，且只能按 Execution ID 查询单次 Collector 执行，不能提供平台通用的执行历史。

本阶段需要用简单、可扩展的方式建立 Agent Schedule 和 Admin API，使 Collector 成为第一个可定时运行的 Agent，同时让 Event Extractor、Analyst 等未来 Agent 可以复用同一平台合同。MVP 不引入外部任务队列、分布式调度或复杂补偿机制。

## Solution

AgentRun 新增平台级 Agent Schedule。每个 Agent Definition 最多拥有一个 Schedule，Schedule 固定绑定一个明确的不可变 Agent Version，并保存该版本定义的通用 Agent Input。PostgreSQL 是 Schedule 的唯一事实源；`go-co-op/gocron/v2` 只作为进程内、可从数据库重建的时间触发器。

AgentRun 同时新增由独立 Admin Bearer Token 保护的管理 API，用于：

- 管理 Agent Schedule 的创建、完整替换、部分修改、启用和停用；
- 分页查询平台通用 Agent Execution 审计记录；
- 读取和修改已注册 Model Provider 与 Connector 的当前配置；
- 对 Key 做写入后不回显、读取时只显示配置状态和安全掩码。

所有业务 API 统一使用 `/api` 前缀。现有 Collector 创建和查询接口迁移到 `/api/v1/collector/runs`，旧路径不保留兼容别名。健康探针和 OpenAPI/Swagger UI 入口保持现状。

每个新 Agent Execution 在启动时读取并冻结当前运行配置；配置修改立即影响后续手动或定时 Execution，不改变在途 Execution。Schedule 到期后调用既有 Agent Execution 能力，不取代 Eino Workflow、Connector Invocation 或各 Agent 的业务编排。

## User Stories

1. As an Admin Portal Service, I want to list all Agent Schedules, so that an administrator can see each Agent 的当前定时状态。
2. As an Admin Portal Service, I want to query one Schedule by Agent key, so that an administrator can edit the correct Agent。
3. As an Admin Portal Service, I want to create or fully replace one Schedule for an Agent Definition, so that each Agent has one unambiguous periodic policy。
4. As an Admin Portal Service, I want Schedule creation to require an explicit `enabled` value, so that saving a Schedule cannot accidentally activate it。
5. As an Admin Portal Service, I want to enable or disable a Schedule with a partial update, so that operations do not need to resend the complete definition。
6. As an Admin Portal Service, I want a Schedule to bind one explicit Agent Version, so that future Agent releases do not silently change scheduled behavior。
7. As an Admin Portal Service, I want to update the Agent Version bound to the same Schedule, so that an Agent can be upgraded without creating a second policy。
8. As an Admin Portal Service, I want to save a version-specific JSON Agent Input, so that Collector and future Agent types can use the same scheduling control plane。
9. As an Admin Portal Service, I want Collector Schedule Input to contain the same natural-language Prompt accepted by the Collector API, so that scheduled and manual collection have the same business semantics。
10. As an Admin Portal Service, I want input to be validated by the bound Agent Version, so that invalid work is rejected before a timer is activated。
11. As an Admin Portal Service, I want standard five-field Cron expressions, so that common wall-clock schedules can be represented without second-level complexity。
12. As an Admin Portal Service, I want one or more daily minute-precision times, so that several fixed collection times can be managed as one Agent Schedule。
13. As an operator, I want all schedules in one AgentRun instance to use the container deployment timezone, so that timezone ownership is clear and not repeated in each Schedule。
14. As an operator, I want service startup to reject a missing or invalid `TZ`, so that a container cannot silently execute wall-clock schedules in UTC。
15. As an operator, I want enabled Schedules restored from PostgreSQL after restart, so that in-memory gocron state is disposable。
16. As an operator, I want downtime ticks not to be backfilled, so that restarting the MVP does not create an unexpected execution burst。
17. As an operator, I want Schedule changes to affect only future triggers, so that active Agent Executions retain their original snapshots。
18. As an operator, I want different Agent Definitions to execute concurrently, so that one Agent type does not block another。
19. As an operator, I want overlapping triggers for the same Agent Definition to create a terminal `skipped` Execution, so that overlap is visible without queueing or parallel execution。
20. As an operator, I want API-created and Schedule-created Executions to use the same per-Agent overlap rule, so that concurrency does not depend on trigger source。
21. As an auditor, I want scheduled Executions to record trigger source, Schedule identity and trigger time, so that their origin is traceable。
22. As an auditor, I want skipped Executions to identify the blocking active Execution, so that the reason is reviewable。
23. As an Admin Portal Service, I want a paginated Agent Execution list, so that execution history can be browsed without loading all rows。
24. As an Admin Portal Service, I want to filter the list by optional Agent key, so that each Agent management page can show its own history。
25. As an Admin Portal Service, I want creation-time ascending or descending order with a stable tie-break, so that pagination is deterministic。
26. As an Admin Portal Service, I want execution identity, Agent Version, trigger source, state and timestamps, so that I can see when work ran and whether it succeeded。
27. As an Admin Portal Service, I want safe failure and skip reasons, so that operators can diagnose an unsuccessful run without exposing secrets or business payload。
28. As a security owner, I want the execution list to omit all Agent Input and Output Payloads, so that Prompts, Artifacts, Candidates and Connector content are not exposed through the management list。
29. As an Admin Portal Service, I want to list every registered Model Provider configuration target, including unconfigured targets, so that the portal can present the complete supported set。
30. As an Admin Portal Service, I want to read one Model Provider configuration without retrieving its full Key, so that secrets are not disclosed。
31. As an Admin Portal Service, I want to update Base URL, model and Key for a registered Model Provider, so that future Executions use current operations settings。
32. As an Admin Portal Service, I want to list every registered Connector configuration target, including unconfigured targets, so that all supported channels can be managed。
33. As an Admin Portal Service, I want to read one Connector configuration without retrieving its full Key, so that secrets are not disclosed。
34. As an Admin Portal Service, I want to update Base URL and optional Key for a registered Connector, so that future Executions use current channel settings。
35. As a platform owner, I want unknown Provider and Connector keys rejected, so that database rows cannot pretend to create capabilities that AgentRun has not implemented。
36. As a platform owner, I want registered but unconfigured targets patchable, so that their first current configuration can be created through Admin API。
37. As an operator, I want configuration updates to affect the next Execution without restart, so that credential and endpoint maintenance is immediate。
38. As an operator, I want in-flight Executions to retain their starting configuration snapshot, so that a run cannot change providers halfway through。
39. As a security owner, I want Admin API protected by an independent required token, so that the Tidewise Data Service token cannot mutate platform configuration。
40. As a security owner, I want keys excluded from logs, errors, Artifacts and responses, so that management operations do not expand credential exposure。
41. As a Tidewise Data Service, I want the Collector creation and status endpoints under `/api/v1`, so that business routes use a stable public service prefix。
42. As an API client, I want the removed legacy Collector route to return not found, so that there is only one supported path contract。
43. As an API client, I want the OpenAPI document and Swagger UI to describe the new paths, authentication and schemas, so that integration behavior is discoverable。
44. As a database operator, I want this feature delivered through the next incremental migration, so that existing databases and historical Executions are preserved。
45. As a developer, I want the same contracts verified in CI with an isolated PostgreSQL database, so that a PR cannot pass by skipping persistence behavior。

## Implementation Decisions

### Agent Schedule domain

- Agent Schedule is a platform entity, not a Collector-specific cron record.
- Each Agent Definition has zero or one Schedule. PostgreSQL enforces uniqueness by `agent_key`, not Agent Version.
- A Schedule has a stable UUID, Agent key, bound Agent Version, schedule type, type-specific policy, `input_payload`, enabled state, last actual trigger time, next planned run time, and creation/update timestamps.
- The Agent key and Agent Version relationship must be validated; the version must exist and belong to the path Agent key.
- Replacing a Schedule preserves the existing Schedule identity. There is no Schedule delete operation in this phase.
- `input_payload` is a JSON object and is copied into every triggered Agent Execution as an immutable snapshot. Its encoded size is limited to 64 KiB.
- Agent Version owns Input validation. Collector V1 accepts only its existing Prompt object and keeps the existing 64 KiB Prompt limit.
- Enabling a Schedule validates Agent registration, Agent Version, Input, timing policy and the presence of required current Model Provider and Connector configurations. It does not call an external service or prove that a Credential is valid.

### Schedule policy and time

- Supported types are `cron` and `daily`.
- `cron` accepts exactly the standard five fields. Seconds and per-expression timezone directives are rejected.
- `daily` accepts one or more `HH:MM` values at minute precision. Values are validated, deduplicated and stored in stable chronological order.
- `duration`, relative interval, one-time, weekly and monthly API variants are out of scope; equivalent recurring wall-clock behavior should use Cron where possible.
- Schedule requests and rows contain no timezone field.
- Container deployment must provide a valid IANA timezone through `TZ`; dev and UAT use `Asia/Shanghai`.
- AgentRun embeds Go timezone data, validates `TZ` during startup, loads it explicitly and gives that Location to gocron.
- The system clock comes from the deployment host/container. API timestamps, `last_triggered_at` and `next_run_at` are serialized as UTC RFC3339.

### Scheduler runtime

- `go-co-op/gocron/v2` is the selected in-process scheduler. A compatible v2 release is pinned in the Go module graph.
- PostgreSQL is the only Schedule source of truth. gocron Jobs are a runtime projection rebuilt from enabled rows at service startup.
- The first release runs one scheduler instance. Distributed election, locking and multi-instance duplicate prevention are not implemented.
- Admin writes use the accepted minimal order: validate, save PostgreSQL, then synchronize the corresponding gocron Job.
- The first release does not add an Outbox, rollback protocol, background retry or automatic periodic reconciliation between PostgreSQL and gocron.
- If runtime synchronization fails after the database write, Admin API returns a safe synchronization error. Retrying the same PUT/PATCH resynchronizes the saved target state, and service restart reloads it from PostgreSQL.
- Invalid enabled Schedule rows or an inability to register their gocron Jobs fail service startup rather than being silently ignored.
- Disabling a Schedule removes its runtime Job. Updating an enabled Schedule replaces its runtime Job and recomputes `next_run_at`.
- Shutdown or Schedule mutation does not cancel an Agent Execution already created.
- Ticks missed while AgentRun is stopped are not replayed, backfilled or represented by fake failed/skipped Executions.
- First-phase Admin API does not include “run now”; immediate Collector execution continues through the business Collector endpoint.

### Triggering and concurrency

- Schedule trigger creates a normal Agent Execution through the same registered Agent execution boundary used by API-triggered work.
- Scheduled work records `trigger_source=schedule`, Schedule UUID and actual trigger time. Business API work records `trigger_source=api` and no Schedule UUID.
- AgentRun generates the internal idempotency identity required for a Schedule trigger; Admin Portal does not supply an Idempotency-Key.
- The active Execution uniqueness rule changes from global to per Agent Definition.
- Different Agent Definitions may have active Executions concurrently.
- A trigger that overlaps an active Execution of the same Agent Definition creates a terminal `skipped` Execution with the blocking Execution ID and does not start the Agent Workflow.
- gocron concurrency options must not silently suppress the callback because that would remove the required skipped audit record.
- Schedule `last_triggered_at` records an actual callback even when the resulting Execution is skipped. `next_run_at` continues to represent the next planned callback.

### Persistence and migration

- Existing migration files remain immutable. The feature uses the next sequential incremental PostgreSQL migration.
- The migration creates the Agent Schedule table and its foreign keys, uniqueness, schedule-type checks and operational indexes.
- Agent Execution persistence gains Agent key, generic Input snapshot, trigger source, optional Schedule reference and trigger timestamp.
- Existing Collector rows are backfilled with `agent_key=collector`, `trigger_source=api` and an Input object derived from their stored Prompt.
- Existing Collector Prompt/hash/byte columns remain as a nullable Collector compatibility projection for this change; new Collector executions keep that projection synchronized with generic Input. Admin Execution APIs never expose either representation.
- The active Execution unique index is migrated from one global active row to one active row per Agent key.
- Existing Agent Execution, Connector Invocation, Artifact publication and split Provider/Connector configuration data must be preserved.
- Database constraints continue to enforce valid terminal states and successful-run invariants already owned by the Collector.

### Admin authentication and security

- Admin API uses `Authorization: Bearer ${AGENTRUN_ADMIN_TOKEN}`.
- `AGENTRUN_ADMIN_TOKEN` is independent from `AGENTRUN_SERVICE_TOKEN` and is required for service startup in every supported environment.
- Missing or invalid Admin authorization returns the same generic unauthorized response without revealing token state.
- The browser does not call AgentRun directly; Admin Portal Service is the trusted management client.
- Full Model Provider and Connector Keys are accepted only on write, never returned.
- Read responses expose whether a Key is configured and a minimal safe suffix mask only.
- Keys must not appear in logs, error messages, OpenAPI examples, Artifact files or Agent Execution list responses.
- The accepted plaintext-credential ADR remains in force: this is a dev/UAT MVP and production is unsupported until external-key encryption and migration are introduced.

### Configuration registry and updates

- AgentRun code/Agent Version registration determines supported Model Provider and Connector keys. Admin API cannot register arbitrary capabilities.
- List APIs include every supported target even when no current database row exists. Detail and PATCH for unknown keys return `404`.
- PATCH of a known but unconfigured target creates its current configuration row.
- Model Provider representation includes immutable Provider key, Base URL, model, Key configured/masked state and update time.
- Connector representation includes immutable Connector key, Base URL, Key configured/masked state and update time.
- PATCH request identity comes from the path; Provider/Connector key is not mutable in the body.
- Omitted PATCH fields preserve current values and unknown fields are rejected.
- A provided Base URL cannot be cleared and must use HTTPS, or loopback HTTP in `dev`; userinfo is forbidden and `uat` requires HTTPS. Admin validation and Agent execution readiness use this same rule.
- A provided model name must be nonblank.
- Model Provider Key is required and cannot be cleared. A nonempty supplied value replaces it.
- Connector Key is optional under one uniform rule for all Connectors; an omitted value preserves it, a nonempty value replaces it and an explicit empty string clears it.
- Timeout, Candidate limit, concurrency, rate limiting, Connector enablement, query strategy and arbitrary settings JSON remain fixed Agent Version behavior and are not configurable through this API.
- Every new Agent Execution loads current configurations from PostgreSQL and freezes them before starting the Agent Workflow. In-flight work is unaffected by later PATCH operations.
- Configuration versioning, history and Credential rotation records are not added.

### HTTP API

- Business API routes start with `/api`; Admin management routes start with `/api/admin/v1`.
- Collector routes become `POST /api/v1/collector/runs` and `GET /api/v1/collector/runs/{execution_id}`.
- The old `/internal/agent-run/v1/collector/runs` routes are removed with no compatibility alias.
- Health and documentation routes remain `/healthz`, `/readyz`, `/docs/` and `/openapi.yaml`.
- Schedule routes are:
  - `GET /api/admin/v1/agent-schedules`
  - `GET /api/admin/v1/agent-schedules/{agent_key}`
  - `PUT /api/admin/v1/agent-schedules/{agent_key}`
  - `PATCH /api/admin/v1/agent-schedules/{agent_key}`
- Schedule PUT creates or fully replaces the one Schedule for the Agent key and requires explicit `enabled`, Agent Version, type-specific policy and Input.
- Schedule PATCH changes only supplied fields. Switching schedule type requires a complete valid policy for the new type.
- Schedule list is not paginated because its maximum cardinality is one row per registered Agent Definition.
- Model Provider routes are:
  - `GET /api/admin/v1/model-providers`
  - `GET /api/admin/v1/model-providers/{provider_key}`
  - `PATCH /api/admin/v1/model-providers/{provider_key}`
- Connector routes are:
  - `GET /api/admin/v1/connectors`
  - `GET /api/admin/v1/connectors/{connector_key}`
  - `PATCH /api/admin/v1/connectors/{connector_key}`
- Agent Execution history uses `GET /api/admin/v1/agent-executions`.
- Execution query accepts optional `agent_key`, `page`, `page_size` and `sort_order`. It does not accept status or date filters.
- `page` defaults to 1. `page_size` defaults to 20 and is capped at 100. Invalid values return a safe `400`.
- Sort field is fixed to `created_at`; `sort_order` accepts `asc` or `desc` and defaults to `desc`. Execution UUID is the stable same-direction tie-break.
- List response returns `items`, `page`, `page_size`, `total_items` and `total_pages`.
- Execution items contain only Agent key/version, Execution ID, trigger source, optional Schedule/blocking Execution IDs, status, created/triggered/started/completed times and safe stop/error fields.
- Execution items omit Prompt, Input, hashes, byte counts, Candidate/Connector details, Artifact references and all Output Payloads.
- No Admin Execution detail, deletion, archival, date/status filtering or retention policy is introduced.
- Admin APIs use strict JSON decoding, reject unknown/trailing content and preserve the existing safe error envelope.
- OpenAPI and Swagger UI are updated for all new and renamed routes, schemas, query parameters and the separate Service/Admin Bearer schemes.

## Testing Decisions

- The primary acceptance seam is a full AgentRun Service black-box test using real HTTP routing, an isolated migrated PostgreSQL database, local fake model/Connector endpoints, and controllable scheduler time.
- The black-box test configures registered targets through Admin API, creates/enables a Schedule, advances scheduler time, observes a real Collector Agent Execution, and queries it through the generic paginated history API.
- The same seam verifies Admin authentication, Key redaction, strict request decoding, new Collector paths, removal of legacy paths, and immediate configuration use by the next Execution without restart.
- The same seam verifies Schedule create/replace/PATCH/disable, Cron and daily behavior, next/last trigger timestamps, startup reload, no missed-run catch-up, and no cancellation of an in-flight Execution.
- The same seam verifies same-Agent overlap produces `skipped`, while separately registered fake Agent Definitions can trigger concurrently without depending on Collector internals.
- PostgreSQL migration tests start from the previous migration level and verify forward migration, data backfill, per-Agent active uniqueness, registered-version ownership and preservation of existing execution/configuration rows.
- Focused tests cover five-field Cron rejection/acceptance, daily time canonicalization, `TZ` validation, 64 KiB Input limits, PATCH field semantics, URL validation, registered-key rejection, pagination bounds and stable ordering.
- Security tests inject recognizable fake secrets into writes and lower-level errors, then assert they do not appear in logs, HTTP responses, OpenAPI examples or persisted Artifact data.
- Existing Collector HTTP tests are adapted to the new path and continue to cover seven fake direct Connectors and successful `results_pending=0`.
- PostgreSQL-backed tests must run with `AGENTRUN_TEST_DATABASE_URL`; CI must not treat an unset database URL skip as acceptance for this feature.
- Final verification includes formatting, vet, build, `go test -count=1 ./...`, race tests, PostgreSQL integration tests, migration checks, OpenAPI contract tests, diff/credential scans and Eino dependency compatibility checks.
- No real external Provider call is required for automated acceptance; external Credential validity is outside Schedule enable-time validation.

## Out of Scope

- Event Extractor Agent, Analyst Agent or any new Agent Workflow implementation.
- Admin Portal frontend or Tidewise Data Service repository changes.
- Backward-compatible aliases for the removed Collector paths.
- Multiple Schedules for one Agent Definition.
- Per-Schedule timezone selection.
- Duration-based relative intervals, one-time jobs and missed-run catch-up.
- Admin “run now”, Agent cancellation, retry queues or execution replay.
- Status/date execution filters, Admin Execution detail, deletion, archival or retention jobs.
- Dynamic Provider/Connector capability registration.
- Provider/Connector timeout, limit, concurrency, rate-limit, enablement or arbitrary settings management.
- Configuration versions, audit history or Credential rotation records.
- Production Credential encryption; production deployment remains prohibited by the accepted plaintext Credential ADR.
- gocron persistence, distributed scheduler election/locking, multi-instance execution, River, Asynq, Temporal or a new queue.
- Outbox, cross-store transaction, runtime rollback, automatic synchronization retry or periodic reconciliation.
- Changes to Eino Agent Workflow composition.

## Further Notes

- User-controlled PR merge remains required.
- Tidewise Data Service must update its Collector client paths in its own repository before consuming the renamed API; this repository does not modify Tidewise Data.
- Eino remains responsible for one Agent Execution’s internal Workflow. Schedule persistence and periodic triggering are AgentRun platform concerns.
- Reference audit used the shared read-only clones at Eino commit `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`, Eino Ext commit `9137edd89e72b72735ede69db1c5ae29178a6e41`, and Eino Examples commit `171220631fb7068ead50b7cd964b8c471647117d`. The relevant scheduler/timer search found only internal timers and content-time filters, not a business Cron/Schedule control plane; therefore gocron is introduced outside Eino rather than altering Workflow composition.
- PostgreSQL remains the durable control plane; gocron is deliberately a small, reconstructible runtime mechanism.
