# Collector Agent V1 复刻补齐与可靠性 Spec

状态：Ready for implementation
日期：2026-07-23
Issue：#17
修订：2026-07-24 Tavily 同日时间窗兼容及零结果参数修正（Issue #25）
修订：2026-07-27 Web Search 最近一天召回策略（Issue #121）
涉及系统：`tidewise-ai-agentrun`
基线：AgentRun `4306eaa`；Collector Platform Foundation Issue #13

## Problem Statement

Collector Agent V1 已具备异步 HTTP 接口、PostgreSQL 执行控制面、DeepSeek 查询规划、七个固定 Connector、Candidate 终态和本地 Artifact，但尚未完整复刻当前 Codex Raw Collector 的真实执行合同，并已在真实采集结果中产生可证明的数据损坏。

财联社数字 ID 被通用 JSON 解码为浮点数后写成科学计数法，导致来源 URL、来源外部 ID、URL hash 和去重身份错误。专业来源中的 HTML 高亮标签被替换为空格，导致相邻中文和数字出现断词。Tavily 没有启用 Codex 基线中的原始 Markdown 返回，结果上限也从 10 降为 5。URL、时间、正文选择和 hash 合同缺少跨语言确定性基准。

共享 dedup index 当前被当作不可缺失状态；索引丢失时系统会直接按空索引运行，历史 Markdown 去重能力随之消失。运行 manifest 和 summary 不能完整复核 Connector、Candidate、内容等级和 Artifact，失败或跳过的运行通常没有文件审计。

更严重的是，当前系统先发布文档、共享 index 和 manifest，随后才提交 PostgreSQL 成功终态。若数据库完成写失败、结果未知或进程在两者之间退出，已发布文档不会通过 GET 暴露，但共享 index 会阻止后续任务重新接收它们，形成永久不可达数据。

## Solution

在不推翻 Eino Workflow 和 AgentRun 平台模型的前提下，补齐 Collector V1 的直接结果、Tavily、确定性处理、可重建 index、批后审计和可恢复发布合同。

Connector Adapter 忠实保留数字身份和相邻文本，Tavily 使用 explicit news、advanced、确定性时间参数、三段 chunk、原始 Markdown 和每轮 10 条结果。Collector 使用一套跨语言 golden contract 统一 canonical URL、同 URL 正文选择、发布时间解析、normalized body、SHA-256、document ID 和 SimHash64；近似重复唯一权威半径固定为 3。

`data/documents` 中 accepted Markdown 继续作为事实载体，TSV 只作为可重建缓存。索引缺失时从 Markdown 自动重建；显式 verify/rebuild 能发现和修复损坏或 stale index；健康索引不在每轮全量扫描 Markdown 正文。

每个已持久化 Execution，包括成功、部分成功、失败和活动冲突跳过，都具有可查询终态和完整安全审计。Artifact 发布采用最小 prepare/publish/reconcile 协议：先持久化发布计划和 PostgreSQL prepared 状态，再幂等发布文件与 index，manifest 最后发布，最后幂等提交数据库终态。服务启动只恢复已 prepared 的发布，不重新运行 Planner 或 Connector。

已有污染 Artifact 在本变更中保持只读。本变更生成审计清单和可显式执行的迁移能力；只有用户另行授权后，才可将污染文档按原字节隔离、记录迁移 manifest、重建 index 并重新采集。

## User Stories

1. As a Collector user, I want all seven non-Codex Connectors to retain their current fixed fan-out behavior, so that reliability fixes do not reduce source coverage.
2. As a Collector user, I want each enabled Connector invoked once after successful planning, so that the LLM cannot choose or skip channels.
3. As a Collector user, I want at most three Connector calls in flight, so that external calls remain bounded.
4. As a Collector user, I want one Connector failure isolated from the others, so that usable direct results can still complete.
5. As a data consumer, I want Connector response text preserved without rewriting, so that Raw Documents remain faithful to the source channel.
6. As a data consumer, I want numeric source IDs preserved exactly, so that URLs and provenance do not change through JSON decoding.
7. As a data consumer, I want HTML highlight tags removed without inserting artificial spaces, so that adjacent Chinese characters and numbers remain intact.
8. As an auditor, I want real provider payload shapes represented by regression fixtures, so that fixes cover observed failures rather than synthetic examples only.
9. As a Tavily user, I want advanced search with explicit news and deterministic time parameters, so that Tavily returns auditable recent-news results instead of silently changing search modes.
10. As a Tavily user, I want three chunks per source and raw Markdown content requested, so that direct full text is available when the Provider returns it.
11. As a Tavily user, I want raw content preferred over snippets, so that the richest direct Connector result is retained.
12. As a Tavily user, I want at most 10 direct results, so that Tavily has the same per-Connector budget as the other V1 channels.
13. As a data consumer, I want the Collector to continue avoiding secondary URL, PDF and attachment fetches, so that content origin remains `connector_response`.
14. As a data steward, I want canonical URL behavior identical across Go and the Codex reference contract, so that duplicate identity is portable.
15. As a data steward, I want default ports, fragments and tracking parameters removed deterministically, so that cosmetic URL differences do not create separate documents.
16. As a data steward, I want remaining query pairs sorted stably, so that canonical URL hashes do not depend on input order.
17. As a data steward, I want equal-level content compared by non-whitespace Unicode character count, so that formatting noise does not select the winner.
18. As a data steward, I want exact ties resolved by an explicit stable rule, so that Go map iteration cannot change `primary_connector` or document hashes.
19. As a data steward, I want common ISO timestamps including space-separated date-time forms parsed, so that known old material cannot bypass the time gate.
20. As a data steward, I want unparseable timestamps treated as unknown rather than invented, so that collection time does not overwrite source time.
21. As a data steward, I want normalized body, content SHA-256, document ID and SimHash64 validated by golden fixtures, so that future refactors preserve identity.
22. As a data steward, I want the SimHash Hamming radius fixed at 3, so that implementation follows current real scheduled behavior rather than the conflicting catalog declaration of 10.
23. As an operator, I want accepted Markdown to remain the durable fact carrier, so that loss of a cache does not lose collected material.
24. As an operator, I want a missing dedup index rebuilt automatically, so that historical deduplication does not silently disappear.
25. As an operator, I want an explicit index verify command, so that malformed rows, missing files and hash mismatches are discoverable.
26. As an operator, I want an explicit atomic index rebuild command, so that a damaged or stale cache can be repaired from accepted Markdown.
27. As an operator, I want a healthy index loaded without scanning all Markdown bodies on every run, so that normal execution cost remains bounded.
28. As an auditor, I want every merged Candidate to reach one of six terminal dispositions, so that no result disappears without an outcome.
29. As an auditor, I want successful-class persistence to reject `results_pending != 0`, so that success cannot hide unfinished Candidate work.
30. As an auditor, I want actual window start and end recorded, so that time-gate decisions can be reproduced.
31. As an auditor, I want attempted, completed and failed Connector counts separated, so that all-failed runs are not reported as completed.
32. As an auditor, I want per-Connector status, result count and safe failure reason recorded, so that a batch can be reviewed without logs.
33. As an auditor, I want raw, merged, terminal, pending and six disposition counts recorded, so that Candidate conservation can be checked.
34. As an auditor, I want all four content-level counts recorded, so that direct-result quality can be inspected.
35. As an auditor, I want accepted paths and hashes plus ledger, index, summary and manifest paths recorded, so that every output can be reconciled.
36. As an auditor, I want `merged_results = results_terminal + results_pending` enforced, so that the batch summary is internally consistent.
37. As an operator, I want successful, partial, all-failed, Planner-failed, timed-out and skipped runs to have a durable audit outcome, so that audit is not limited to the happy path.
38. As an operator, I want an active-run conflict represented as a skipped Execution, so that `skipped_previous_run_active` is queryable by ID.
39. As a caller, I want a conflict response to identify both the skipped request and the active Execution, so that I can inspect both outcomes.
40. As a caller, I want replay of the skipped request's idempotency key to return the same skipped Execution, so that transport retries remain idempotent.
41. As an operator, I want all-Connector failure to remain an Execution failure while retaining `completed_with_connector_failures` as the Collector stop reason, so that service success and batch mechanics are not conflated.
42. As an operator, I want Planner failure and execution interruption mapped to `agent_or_tool_limit`, so that the stop reason matches incomplete orchestration.
43. As an operator, I want Artifact publication prepared before shared files or index are changed, so that recovery always has a durable plan.
44. As an operator, I want publication steps idempotent, so that restart can finish a partially published plan safely.
45. As an operator, I want manifest published last, so that it remains the completed file-set marker.
46. As an operator, I want PostgreSQL terminal completion idempotent and read back after unknown results, so that a timeout cannot incorrectly turn published success into failure.
47. As an operator, I want startup to reconcile prepared publication only, so that reliability does not become general task replay.
48. As a caller, I want GET to expose formal Artifact paths only after terminal database completion, so that the HTTP contract never advertises an uncommitted publication.
49. As an operator, I want Model Provider and Connector Base URLs validated as absolute HTTP(S) URLs with hosts, so that readiness does not accept unusable endpoints.
50. As a security-conscious operator, I want remote plaintext HTTP model and Connector endpoints rejected, so that credentials are not sent without transport protection.
51. As a developer, I want loopback HTTP endpoints allowed in development tests, so that fake external services remain practical.
52. As a security-conscious operator, I want model and Connector keys excluded from errors, logs, HTTP responses and all Artifacts, so that failure evidence does not leak credentials.
53. As an owner of historical data, I want existing polluted Artifacts inventoried without mutation, so that immutable history is not silently rewritten.
54. As an owner of historical data, I want any future quarantine operation to preserve exact bytes and hashes, so that corrupted history remains auditable.
55. As an owner of historical data, I want index migration and re-collection performed only after explicit authorization, so that repair does not imply data deletion.
56. As a Raw Collector owner, I want factual verification kept out of this Agent, so that Event evidence review remains a separate downstream responsibility.
57. As a Raw Collector owner, I want the optional official-source registry excluded from this change, so that the task does not add behavior absent from the mandatory Codex execution path.
58. As a maintainer, I want deterministic and fault-injection tests before implementation, so that each reliability change is proven by a failing test first.
59. As a maintainer, I want PostgreSQL-backed tests to use an isolated test database, so that skipped tests are not accepted as evidence.
60. As a maintainer, I want one honest seven-Connector smoke when local configuration permits, so that deterministic tests are complemented by current Provider evidence.
61. As a Collector user, I want a sub-day Tavily window to remain a valid Connector request, so that same-day `start_date` and `end_date` do not fail the Connector before the exact downstream time gate runs.
62. As a Collector user, I want Tavily recent-news retrieval to use explicit news and relative-time semantics, so that automatic finance classification plus an absolute date does not silently return zero direct results.
63. As a Collector user, I want Tavily RFC1123 publication timestamps recognized by the exact downstream time gate, so that old search results cannot pass as unknown-time Candidates.
64. As a Collector user, I want Bocha, Tavily and Parallel Search to limit provider-side discovery to the most recent day, so that older search results do not consume the fixed per-Connector result budget.
65. As an auditor, I want the exact downstream timestamp gate preserved after provider-side freshness filtering, so that provider date granularity cannot silently decide Candidate acceptance.

## Implementation Decisions

### Scope and preserved architecture

- The change modifies only AgentRun. It does not modify Tidewise Data or access the Tidewise Data database.
- The existing Agent Definition, Agent Version, Agent Execution, Connector Invocation, HTTP API and capability-first package direction remain. Current configuration is represented by separate Model Provider Configuration and Connector Configuration resources.
- Eino remains the typed orchestration boundary for `Planner -> seven Connector fan-out -> materialization`. The change does not introduce AgenticModel, tool calling, multi-Agent delegation or conversational state.
- The fixed Connector set remains Parallel, Tavily, Bocha, CLS Telegraph, Eastmoney Fast News, Eastmoney Stock News and STCN Quick News.
- Codex-only `live_search` remains excluded.
- The optional official-source registry and classifier remain excluded because they are not part of the mandatory current Codex execution path. This exclusion does not prevent DeepSeek from generating official-source queries from the submitted Prompt.
- Raw Collector does not perform factual evidence matching, conflict resolution, officialness approval, independent-source counting, evidence grading, Event extraction or investment analysis.

### Reference baseline and conflict resolution

- The parity baseline is the current local `collect-global-raw-documents` skill, Prompt, source contracts, stager, materializer, dedup implementation, Provider adapters and tests.
- Golden provenance records SHA-256 identities of the audited Python reference files because the reference directory is not a Git checkout.
- The source catalog declares SimHash Hamming radius 10, while the actual scheduled materializer calls near-duplicate matching with radius 3. Radius 3 is the only authoritative Collector V1 value.
- Candidate limits are 10 for every Connector.
- Connector direct results remain the only content source; no URL is opened after a search result or feed item is returned.

### Direct-result fidelity

- Generic JSON number decoding must preserve integer and decimal token text without conversion through binary floating point. Numeric IDs used in URLs or provenance are rendered from their original JSON number lexeme.
- String IDs remain unchanged after surrounding whitespace removal.
- HTML-to-text normalization follows the Python `HTMLParser` golden behavior: character references are decoded, tags are removed, adjacent data fragments are concatenated before whitespace normalization, and no synthetic space is inserted merely because a tag occurred.
- The normalizer does not summarize, translate, correct or rewrite returned text.
- Regression fixtures include observed CLS numeric IDs and Eastmoney highlight markup patterns that previously produced scientific notation and broken Chinese/numeric text.

### Tavily parity

- Tavily receives one `combined_query` request per Execution.
- The request explicitly uses the `news` topic, disables automatic parameters, enables advanced search and three chunks per source, requests no generated answer, requests raw content in Markdown form and limits direct results to 10.
- Every request sends `time_range: "day"` and no `start_date` or `end_date`, independently of the Planner's effective time window.
- Explicit news classification is authoritative because the real Tavily API returned zero results when automatic parameters selected `finance` together with an absolute `start_date`, while the same query returned results with explicit `news`.
- Response normalization prefers nonblank raw content and marks it `full_text`; otherwise it uses the direct content snippet and marks it `snippet`; otherwise it uses the title and marks it `title_only`.
- The Connector never follows returned URLs.

### Web Search freshness

- Issue #121 is a documented `collector.v1` conformance fix rather than a new Agent Version. It corrects the intended V1 provider freshness rules; Executions before and after deployment may therefore differ while retaining the `collector.v1` runtime identity.
- Web Search provider-side discovery prioritizes the most recent day independently of the Planner's effective time window.
- Bocha always sends `freshness: "oneDay"`; it no longer maps wider Planner windows to `oneWeek`, `oneMonth` or `oneYear`.
- Tavily always sends `time_range: "day"` as defined above.
- Parallel Search sends `advanced_settings.source_policy.after_date` as the UTC collection date minus 24 hours, formatted as `YYYY-MM-DD`.
- Provider date filtering is a recall constraint, not the Candidate acceptance authority. Collector materialization continues applying its exact inclusive timestamp window after merge.
- When the Planner window exceeds 24 hours, Web Search does not actively cover the older portion of that window. The four fixed professional feeds and their existing downstream gate are unchanged.

### Deterministic Candidate contract

- Canonical URL lowercases scheme and host, removes the fragment, supplies `/` for an empty path, removes default ports 80 and 443, removes all trailing slashes from non-root paths, removes case-insensitive `utm_*`, `fbclid`, `gclid`, `ref`, `mc_cid` and `mc_eid`, and stably sorts remaining query pairs.
- Invalid or hostless URLs remain invalid Candidates rather than being hashed.
- Same-URL selection first compares content level, then the number of non-whitespace Unicode code points.
- An exact richness tie uses a stable tuple independent of Go map iteration: fixed Connector order, original Connector result position, source external ID and content bytes. The first tuple wins; bodies are never concatenated.
- Common ISO forms accepted by Python `datetime.fromisoformat`, including `YYYY-MM-DD HH:MM:SS`, and Tavily's observed RFC1123 HTTP-date form are supported. A naive timestamp at the materializer boundary is interpreted as UTC; Provider adapters continue converting known local source times before materialization.
- An unparseable timestamp is unknown. A successfully parsed timestamp outside the inclusive window is `out_of_window`.
- Body normalization uses NFC, LF line endings, trailing Unicode whitespace removal per line, outer trim and collapse of three or more newlines to two.
- Content SHA-256 hashes the normalized Markdown body beginning with the level-one title.
- Document ID is `sha256:` plus SHA-256 of canonical URL, one newline and lowercase content SHA-256.
- SimHash64 tokenizes normalized NFKC lowercase Unicode word tokens, weights duplicate tokens, hashes tokens with BLAKE2b-64, sets zero-score bits and renders 16 lowercase hexadecimal characters.
- Near duplicate classification uses Hamming distance at most 3.

### Dedup index as cache

- Accepted Markdown under the document root is the fact carrier. The TSV is a derived cache.
- A missing index is rebuilt atomically from accepted Markdown before Candidate processing.
- Normal loading validates the exact header, six-column row schema, hash formats, unique document IDs and resolvable document paths without reading every Markdown body.
- Explicit verify scans accepted Markdown and validates path coverage, canonical URL hash, normalized content hash, SimHash64 and document ID in both directions.
- Explicit rebuild scans accepted Markdown, rejects duplicate document IDs, fails on invalid source documents and atomically replaces the index only after a complete valid replacement is prepared.
- Malformed or stale indexes fail closed and identify the need for verify/rebuild; they are not silently treated as empty.
- Healthy normal runs do not perform full Markdown body verification.

### Complete audit and stop reasons

- Every persisted Execution has a Collector stop reason distinct from its platform Execution status.
- Migration backfills historical `succeeded`, `succeeded_no_change`, `partially_succeeded` and `failed` rows to their deterministic legacy-compatible stop reasons before startup audit reconciliation.
- `connectors_completed` applies only when all seven Connectors complete successfully and Candidate processing is terminal.
- `completed_with_connector_failures` applies when all Connector Invocations reach terminal state but at least one fails. A run with seven failed Connectors has this stop reason and platform status `failed`.
- `agent_or_tool_limit` applies to Planner failure, execution timeout, state interruption or Candidate processing that cannot reach its terminal invariant.
- `skipped_previous_run_active` applies to a new request rejected by the one-active-Execution rule.
- `connectors_attempted` counts Connector HTTP calls that started. `connectors_completed` counts successful Connector responses. `connectors_failed` counts started Connector calls that ended in failure; `not_invoked` is not counted as failed. Each Connector also records terminal status, direct-result count and safe failure code/summary.
- Manifest and summary include the effective window start/end, Connector attempted/completed/failed counts, all per-Connector outcomes, Candidate conservation counts, six dispositions, four content levels, accepted paths/hashes, all audit Artifact paths and the stop reason. The Markdown summary lists every accepted path beside its SHA-256.
- `merged_results = results_terminal + results_pending` is validated before publication.
- Successful, no-change and partially successful database transitions require `results_pending=0`.
- Planner failure, all-Connector failure, timeout and skipped active conflict receive safe audit payloads. A run that stops before Planner output records the Collector default 48-hour effective window ending at Execution creation time. Ordinary failure first atomically commits the PostgreSQL terminal state and Invocation outcomes, then writes and attaches its audit; a restart between those steps rebuilds the missing audit from the terminal database row. An existing audit is reusable only when its Execution, status, Prompt hash, stop reason and safe error identity match. If the Artifact volume is unavailable before a durable plan can be prepared, PostgreSQL retains the minimum terminal audit payload but does not promise later file reconstruction. Only already prepared plans are publication-recoverable.

### Active conflict audit

- A non-idempotent request arriving during another active Execution creates one terminal `skipped` Execution rather than a separate Attempt subsystem.
- The skipped Execution preserves the request idempotency key and Prompt hash/length, records seven terminal `not_invoked` Connector Invocations and produces an empty Candidate ledger plus summary/manifest.
- The HTTP response remains `409 active_execution_exists` and includes both the skipped Execution ID and the active Execution ID.
- Replaying the same idempotency key and exact Prompt returns the same skipped Execution. A later real retry uses a new idempotency key.

### Minimal recoverable publication

- General task replay remains out of scope. Only a completed materialization's publication is recoverable.
- Before any accepted document or shared index mutation, the materializer writes all outputs and a hash-addressed publication plan into a durable pending run area.
- PostgreSQL records a minimal prepared publication identity before shared publication begins.
- A definite failure before PostgreSQL prepare removes its unreferenced pending directory. If prepare may have committed but its result is unknown, an independent read-back preserves a matching plan; an unavailable read-back also preserves the plan conservatively. After prepare, the directory remains durable until commit or startup reconciliation. Once PostgreSQL is readable at startup, pending directories with no prepared row are removed as orphans.
- Publication is idempotent: each target is accepted only when absent or when its bytes match the planned hash; mismatched existing targets fail as integrity errors.
- Accepted documents, Candidate ledger and summary are published before the shared index. The index is atomically replaced. Manifest is published last.
- After manifest publication, terminal PostgreSQL completion is idempotent. A timeout or unknown database result triggers read-back and safe retry, never an immediate downgrade to failed.
- Successful, no-change and partially successful terminal states have no direct persistence bypass; they can be reached only by committing a matching prepared publication. Manifest, publication plan and PostgreSQL completion use one captured completion timestamp.
- GET exposes completed Artifact references only after PostgreSQL confirms the terminal state.
- Startup first reconciles prepared publications. A fully or partially published plan is completed and its database terminal state is committed.
- Executions interrupted before publication preparation are not replayed; they are marked failed as before.
- Pending publication state stores only the minimum plan identity, safe counts, paths and hashes needed for reconciliation. It is not a generic queue, lease, worker or retry framework.

### Model Provider and Connector readiness and secret safety

- Every Model Provider and Connector Base URL must parse as an absolute HTTP(S) URL with a non-empty host and no embedded credentials.
- UAT requires HTTPS.
- Development permits HTTP only for loopback hosts used by local services and fake external-service tests; non-loopback development endpoints require HTTPS.
- The DeepSeek Model Provider key is required. Connector keys follow one uniform optional rule and do not block readiness; an external endpoint that rejects anonymous requests produces the ordinary safe Connector Invocation failure.
- Errors, logs, HTTP responses, publication plans, manifests, summaries, ledgers and Markdown must not contain keys, Authorization headers, full Prompt text or raw model error/response bodies.

### Existing polluted Artifact migration

- The implementation performs a read-only audit of existing accepted Markdown, Candidate ledgers, summaries, manifests and the dedup index. It records the exact path, kind and SHA-256 of every audited file, the index row count, and detected pollution reasons as a finding subset.
- The latest read-only inventory contains 131 accepted Markdown documents, 131 index rows and 3 Candidate ledgers. It includes 11 documents with scientific-notation CLS IDs and at least 11 documents with obvious highlight-tag spacing damage; later smoke Artifacts use lossless integer IDs and remain unmodified.
- The product change does not modify, delete, move or rebuild these existing user Artifacts automatically.
- A future explicitly authorized migration preserves polluted files byte-for-byte under a quarantine run, records old path, new path, hash, index identity and reason in an immutable migration manifest, atomically rebuilds the active index from the remaining accepted documents, then performs a new collection.
- Original run manifests and Candidate ledgers remain immutable. The migration manifest supplies the additional path history.

### Eino reference audit

- `cloudwego/eino` at `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`: audited `components/model/interface.go`, `compose/workflow.go` and `compose/workflow_test.go`. These confirm typed Workflow DAG and all-predecessor fan-in remain appropriate. No checkpoint-based task replay is introduced because publication reconciliation is an application persistence concern.
- `cloudwego/eino-ext` OpenAI-compatible ChatModel is used for DeepSeek's compatible API with explicit Base URL, model, key, timeout and JSON response format. The former DeepSeek-specific component was removed because its SDK dependency pulled the unrelated vulnerable Ollama server module into AgentRun. The reference contains general search tools but no Tavily, Bocha, CLS, Eastmoney or STCN adapter matching this direct-result contract.
- `cloudwego/eino-examples` at `171220631fb7068ead50b7cd964b8c471647117d`: audited `quickstart/chat/{main.go,generate.go,template.go}` and `compose/workflow/{1_simple,2_field_mapping,3_data_only,4_control_only_branch,5_static_values,6_stream_field_map}/main.go`. These confirm composition at the application edge and explicit Workflow dependencies. They do not supply AgentRun's HTTP idempotency, Artifact/index or publication transaction boundary.
- Connector fidelity, deterministic Candidate processing, index verification and publication reconciliation remain project-specific adapters and domain services.

## Testing Decisions

- TDD is mandatory. Each behavior group begins with a meaningful failing test and records RED, GREEN and REFACTOR evidence.
- The highest integration seam is the existing black-box Collector HTTP path using the real Handler, PostgreSQL Repository, Eino Workflow, Artifact writer and GET response with fake DeepSeek and fake Provider HTTP servers.
- The black-box matrix covers successful, no-change, partial, all-Connector-failed, Planner-failed, timeout, active-conflict-skipped and publication-reconciled executions.
- PostgreSQL-backed tests always run with an isolated `AGENTRUN_TEST_DATABASE_URL`. A skipped database test is not acceptance evidence.
- Focused Connector tests assert lossless CLS number handling, HTML text extraction, Tavily request shape, raw-content priority, content level and limit 10 using fixtures shaped like observed real responses.
- Deterministic golden fixtures cover canonical URL, equal-level merge and tie-break, timestamp parsing, normalized body, content SHA-256, document ID and SimHash64. Fixture provenance records the audited Python reference hashes.
- Golden cases include Unicode, repeated query keys, tracking parameters, default ports, empty paths, multiple trailing slashes, space-separated timestamps, naive timestamps, Tavily RFC1123 timestamps and equal-richness candidates.
- Index tests cover missing-index automatic rebuild, healthy-index no-body-scan behavior, malformed header/row, duplicate identity, missing path, extra row, missing row, content/hash mismatch, explicit verify and atomic rebuild failure.
- Audit tests assert actual window bounds, attempted/completed counts, every Connector outcome, Candidate and content-level conservation, accepted path/hash integrity and all four stop reasons.
- Publication fault injection covers failure or process interruption after prepare, after partial document publication, after summary/ledger publication, after index replacement, after manifest publication, before PostgreSQL terminal commit, during a failed commit and after an unknown commit result.
- A PostgreSQL-backed HTTP restart test forces all in-process publication commit attempts to fail, then proves startup reconciles the prepared publication before stale failure, invokes no second model or Connector call, clears prepared state and exposes the recovered Artifact through GET. Separate tests prove pre-prepare stale executions fail without replay.
- Security tests inject keys, Authorization values, full Prompt and raw Provider/model payloads into underlying errors and assert no observable output contains them.
- Model Provider and Connector Configuration tests cover malformed, relative, hostless, credential-bearing, remote plaintext and valid loopback/HTTPS URLs in dev and UAT.
- Existing polluted Artifact tests use read-only fixture copies. Repository `data` is never edited by automated tests.
- Final local verification runs gofmt, go vet, command builds, `go test -count=1 ./...`, race tests, isolated PostgreSQL repository and HTTP tests, migration checks, diff checks, credential scans and forbidden-path scans.
- When all local Model Provider and Connector Configurations are available, one real seven-Connector smoke is run after deterministic tests. The report distinguishes zero results, external-service failures and missing configuration; it never reports a false pass.
- Final review runs Standards and Spec axes independently against the merge base and includes untracked task files while excluding the pre-existing user-owned research Spec.

### 2026-07-24 Tavily 同日时间窗修复证据

- RED: 原同日请求测试失败并捕获请求仍含 `end_date: "2026-07-24"`，与当时的同日窗口合同冲突。
- GREEN: 同日用例与原有跨日合同用例共同通过；完整 Connector package 通过。
- REFACTOR: 日期计算统一到 UTC，并使用 `startDate`、`endDate` 表达 Provider 日期边界；没有引入日期类型、重试、fallback 或额外抽象。
- Regression: 配置隔离 PostgreSQL 后执行 `go test -count=1 ./...` 全部通过；`go vet ./...`、`go build ./cmd/...` 和 `git diff --check` 通过。
- Real smoke: API Execution `de79800b-db3b-4075-aded-535f7d54173c` 与 Schedule Execution `69b7df1b-58c6-472e-96ec-4108f8a6c1bc` 均为 `succeeded` / `connectors_completed`；七个 Connector 全部 `completed`，60 个 Candidate 全部终态且 `results_pending=0`。Tavily 连续返回 0 条成为后续参数诊断的输入，而不再被视为质量通过。

### 2026-07-24 Tavily 零结果与 RFC1123 时间门禁修复证据

- Tavily API diagnosis: 固定同一 Planner query 与日期后，`auto_parameters=true` 选择的 `finance` topic 加绝对 `start_date` 返回 0 条；去掉日期、改用 `time_range: "day"`、关闭自动参数，或显式使用 `news` 均返回 10 条。最终选择显式 `news`、关闭自动参数，并对 24 小时内窗口使用相对 `day`。
- RED 1: Tavily request-shape 测试先失败，捕获缺少 `topic: "news"` 且仍发送 `auto_parameters: true`。
- GREEN 1: 显式 news、关闭自动参数后，原始 Markdown、advanced、三段 chunk、10 条上限及 48 小时绝对日期合同共同通过。
- RED 2: 两小时窗口测试先失败，捕获请求仍发送绝对 `start_date` 且未发送 `time_range: "day"`。
- GREEN 2: 24 小时内窗口改为相对 `day` 且不发送绝对日期；长窗口仍发送 UTC `start_date`/`end_date`。
- First smoke: API Execution `10c2af38-3ef9-4d37-a4f3-8129ee8e207b` 为 `succeeded` / `connectors_completed`，七个 Connector 全部完成且 Tavily 从 0 恢复为 10 条；该批同时暴露 Tavily 的 RFC1123 `published_at` 未被解析，导致部分旧闻误记为 accepted。该历史 Artifact 与共享 index 未被静默删除或改写。
- RED 3: 使用真实形态 `Fri, 24 Jul 2026 04:55:35 GMT` 的 materializer 回归测试失败，旧闻被计为 `accepted: 1` 而不是 `out_of_window: 1`。
- GREEN 3: 增加 RFC1123 解析后，同一测试进入 `out_of_window`，Candidate ledger 原因为 `published_at_outside_time_window`。
- Final smoke: Schedule Execution `4f8cebf0-ca8c-40ee-b71f-efbf9e07f531` 为 `succeeded` / `connectors_completed`；七个 Connector 各返回 10 条，70 个 Candidate 全部终态，`accepted=16`、`known_url=27`、`out_of_window=27`、`results_pending=0`。Tavily 的 10 条 RFC1123 结果全部按精确两小时时间门禁进入 `out_of_window`。
- Regression: 使用隔离数据库 `tidewise_ai_server_test` 执行 `go test -count=1 ./...` 全部通过；服务在 smoke 后关闭。

## Out of Scope

- Tidewise Data changes, Data Raw Document Import, Outcome callback, Delivery Outbox or direct Tidewise Data database access.
- Codex `live_search`, Brave or another replacement Connector.
- Optional official-source registry query expansion or URL authority classification.
- Factual evidence verification, source conflict review, evidence grading, Event extraction, industrial-chain analysis or investment conclusions.
- LLM result filtering, relevance review, spam filtering, summarization, translation or rewriting.
- Opening result URLs, PDF files, attachments or secondary pages.
- General Execution restart, Planner/Connector replay, task queue, lease, heartbeat, Worker Attempt, cancellation or multi-instance worker claims.
- Automatic Provider retry, fallback or runtime configuration of fixed limits, timeouts and concurrency.
- Candidate body API, Candidate PostgreSQL persistence or Tidewise Raw Document persistence.
- Admin HTTP API, configuration history, credential versioning, encryption or production credential migration.
- Automatic mutation, deletion, movement, quarantine or index migration of existing user Artifacts.

## Further Notes

- This Spec revises the Connector, deterministic Candidate, dedup index, audit, failure and Artifact publication sections of the Collector Platform Foundation contract. Unchanged platform and HTTP decisions from Issue #13 remain in force.
- The grill decisions intentionally choose the smallest recoverable publication mechanism rather than a general workflow recovery platform.
- `Run Artifact Manifest` remains the completed file-set marker; PostgreSQL remains the HTTP-visible execution control plane.
- The existing user-owned untracked research Spec is unrelated to this change and must remain untouched and uncommitted.
