# Event Semantic 客观语义与 Qdrant 召回投影 V2

状态：Frozen（已通过 Codex“事件推理模型”会话复核）

日期：2026-08-01

Delivery Issue：#164

Owners：Data Service（正式事实、TBox、PG→Qdrant 投影与发布合同）、AgentRun（Execution、
Eino 编排、Qdrant 召回与模型选择）

本 Spec 取代 Event Semantic Phase One 中要求生成 DirectImpact 的部分，并完整取代
Event Semantic Anchor Resolution MVP。历史 DirectImpact 正式数据与物理表不在本次删除。

## Problem Statement

现有 Event Semantic 把“Event 原生客观语义”和“跨 Entity 因果传导”放在同一 Agent
Execution 中。它要求 Generator 生成 DirectImpact，调用 Direct Target 与
DirectTransmissionRule，并通过锚点/产业链关系解析 ChainNode。这使 Event 语义处理同时承担
事实提取、Entity Resolution、图谱传导和分析判断，扩大 Prompt、Data 合同、失败面与审核
复杂度，也把本应属于 Theme Analyst 的投资分析提前写入底层正式语义。

现有 Entity 精确解析无法覆盖简称、常用称呼和非同名语义变体；完整 ABox 不能重新进入
Context Lease，而现有有界锚点方案只覆盖特定 ChainNode 路由。已完成的本地 Qdrant spike
证明，在约 5,029 条 active Entity、`BAAI/bge-small-zh-v1.5`、512 维、cosine 配置下，按
Entity Type 过滤的 Top-5 能为语义变体提供有效候选，但相似度分数不能区分正式实体缺失与
真实语义匹配，因此不能自动形成 EventEntityLink。
该 BGE 结果只作为可行性基线；实施时用户进一步冻结正式 embedding provider 为
DashScope OpenAI-compatible API，模型为 `text-embedding-v4`，固定输出 1024 维 dense float。

Measurement 的当前表示也与实际用途不一致。现有代码与数据库约束要求解析 decimal、shape、
raw/canonical value、unit conversion 和 range，并把不满足数值合同的输出拒绝为
`measurement_value_invalid`。本阶段 Measurement 实际只供 Theme Analyst 阅读和推理，不参与
数据库计算；强行结构化会丢失 LLM 能表达但程序无法可靠归一的原文量化语义，并诱发模型伪造
normalized value。

用户需要先跑通真实 Event 流程，并以固定约 100 Event 和指定英伟达/安靠科技 Event 验证
EntityLink、VariableSignal 与 Measurement 的质量，而不是继续扩大 Event Semantic 的因果
推理职责。

## Solution

Event Semantic V2 只把正式 Event/Evidence 转换为三类 Event 原生客观语义：

1. `EventEntityLink`：能够回溯到 Event Evidence 中明确 mention、无歧义简称或可直接规范化
   类别表达的规范 Entity；
2. `VariableSignal`：Event 对已解析 Entity 的受控变量变化陈述；
3. `Measurement`：挂在 VariableSignal 下、供下游 Agent 阅读的 Evidence-grounded 自然语言
   量化注记。

AgentRun 直接查询 Qdrant：先按 projection 中的 normalized canonical name、alias 或 UUID
执行精确匹配；未唯一命中的 mention 再按单 Event 批量执行 predicted Entity Type 过滤的
向量 Top-K，并让 LLM 只从返回候选选择一个正式 Entity ID 或 `no_match`。V2 不调用 Data
Entity Resolution/Search API。Qdrant 只负责召回，不拥有正式事实或 accepted 状态。Data 在
提交时仍按 pinned Context/TBox 校验正式 Entity、Entity Type、Evidence 血缘、Variable
Definition 版本、适用 Entity Type、方向和 assertion modality。

Variable Definition 的完整 active/current 受控目录随 pinned TBox contract 动态加载，作为
正确性集合；`variable_definition_semantic_v1` 只提供排序或语义提示，Top-K 不得排除完整目录
中的合法定义。

Measurement 改为最小自然语言合同：每项只包含非空 `measurement_text` 与正式
`evidence_ids`。Data 不解析或校验数值、上下界、单位、百分比/百分点、方向一致性、比较口径、
期间或归一化结果，也不据此计算。独立 AI Review 负责判断 Measurement 是否忠实表达所引
Evidence；Measurement 不拥有独立审核状态，随所属 VariableSignal 一起 accepted、rejected
或 superseded。Data 只保留传输安全、长度预算、Evidence 存在性与 referential integrity。

Data-owned 一次性 projector 从正式 PostgreSQL 全量重建两个 Qdrant collection：
`entity_semantic_v1` 与 `variable_definition_semantic_v1`。AgentRun 直接查询 Qdrant，但不得
连接 Data PostgreSQL。正式流程不使用 spike collection。

## User Stories

1. As a Theme Analyst, I want Event Semantic to return only Event-native objective semantics, so that I can own causal transmission and investment reasoning at the correct stage.
2. As a Theme Analyst, I want every accepted VariableSignal to point to a formal Entity and Variable Definition, so that downstream reasoning starts from governed facts.
3. As a Theme Analyst, I want Measurement to preserve the LLM's natural-language quantitative meaning, so that I can read ranges, comparisons, periods and units without lossy normalization.
4. As a Theme Analyst, I want a Signal with only a direction and no Measurement to remain valid, so that the workflow never invents a number to satisfy storage rules.
5. As a Theme Analyst, I want one Signal to carry multiple Measurement texts, so that absolute value and同比/环比变化 can coexist.
6. As a Theme Analyst, I want Measurement to retain formal Evidence references, so that I can verify the quantitative statement against the Event evidence.
7. As a Data domain owner, I want PostgreSQL to remain the sole source of formal Entity and Variable Definition facts, so that vector retrieval cannot create accepted facts.
8. As a Data domain owner, I want Qdrant to be a rebuildable projection, so that it can be discarded and recreated without changing domain state.
9. As a Data operator, I want one repeatable full rebuild command, so that I can seed both production-shaped collections from active/current PG facts.
10. As a Data operator, I want stable point identities and idempotent upsert, so that rerunning the projector does not duplicate points.
11. As a Data operator, I want the projection to record projection and embedding versions, so that retrieval results are attributable to a concrete generation.
12. As an AgentRun operator, I want Qdrant lookup to batch all unresolved mentions for one Event, so that mention count does not cause N+1 network calls.
13. As an AgentRun operator, I want Entity Type filters applied inside Qdrant, so that unrelated formal types do not pollute candidate lists.
14. As an Event Semantic Agent, I want Qdrant projection exact canonical/alias/UUID resolution before vector recall, so that exact names and abbreviations are not degraded by embeddings or routed through a Data search API.
15. As an Event Semantic Agent, I want to choose only IDs returned by the immediately preceding Qdrant exact/vector response, so that I cannot invent an Entity ID.
16. As an Event Semantic Agent, I want an explicit `no_match` result, so that missing ABox facts do not become false bindings to similar companies or institutions.
17. As an Event Semantic Agent, I want Entity Type, role, direction and modality vocabularies assembled from pinned runtime TBox data, so that formal catalogs are not copied into static Prompt text.
18. As an Event Semantic Agent, I want the output JSON structure to remain fixed by the Agent Version, so that dynamic TBox does not turn orchestration into database configuration.
19. As an Event Semantic Agent, I want the full active Variable Definition catalog available for every pinned TBox contract, so that vector Top-K cannot hide a valid small-catalog definition.
20. As a Data reviewer, I want deterministic precheck to reject unknown Entity IDs, stale types, invalid Evidence references and inapplicable Variable Definitions, so that accepted facts remain governed.
21. As a Data reviewer, I do not want deterministic code to interpret Measurement prose, so that domain acceptance does not pretend to understand arbitrary quantitative language.
22. As an independent AI Reviewer, I want to review EntityLink, VariableSignal and their Measurement prose against Event Evidence, so that natural-language Measurement fidelity is checked semantically.
23. As an AgentRun platform owner, I want Work Item, lease, retry and reconciliation ownership unchanged, so that Qdrant does not become a scheduler or durable business state store.
24. As a Data consumer, I want Event Semantic submissions to contain zero DirectImpact objects, so that downstream systems do not confuse Event-native facts with analyst inference.
25. As a migration operator, I want historical DirectImpact data and tables preserved, so that rollout cannot destroy previously accepted facts.
26. As a migration operator, I want historical structured Measurement rows preserved, so that old audit data survives while new writes use natural-language Measurement.
27. As a maintainer, I want obsolete anchor-route, Direct Target and DirectTransmissionRule dependencies removed from the supported V2 flow, so that old behavior is not maintained through hidden compatibility paths.
28. As a quality reviewer, I want a 100 Event comparison report, so that failures can be attributed to ABox/TBox gaps, model errors or workflow defects.
29. As a quality reviewer, I want the fixed NVIDIA/Amkor Event evaluated separately, so that company and semiconductor-chain Entity resolution can be inspected without producing investment conclusions.
30. As a security owner, I want Qdrant and embedding credentials injected at composition roots and excluded from logs, so that retrieval does not leak secrets or Event bodies.
31. As a Data reviewer, I want every EventEntityLink to retain its raw mention and Evidence references, so that an Entity cannot be linked only from background knowledge, graph relations or analyst inference.

## Implementation Decisions

### Domain and versioning

- Introduce a new immutable Event Semantic Agent Version. This is a behavioral contract change, not a V1 conformance fix.
- V2 output contains only EventEntityLink and VariableSignal; Measurement is an optional one-to-many child of VariableSignal.
- DirectImpact, Direct Target, DirectTransmissionRule, EntityRelation transmission and analyst inference are not part of V2 generation, review or submission.
- Historical DirectImpact and structured Measurement storage is retained for audit. No destructive cleanup is authorized.
- Theme Analyst owns cross-Entity transmission, `analyst_inference`, opportunity/risk judgment and Theme/Reason Tree generation.

### Cross-service ownership and wire contract

- Data publishes the versioned Context, Submission, Review and Event Semantics read contracts. AgentRun owns matching consumer DTOs and adapters; V2 does not consume a Data Entity Resolution or search contract.
- Data Context retains the compact manifest, Event/Evidence and pinned dynamic TBox. It does not return complete Entity/EntityRelation ABox, DirectTransmissionRule objects or anchor route catalogs for V2.
- The pinned TBox contract contains active Entity Type definitions, allowed Event Entity roles, active/current Variable Definitions with versions, applicable Entity Types, value type, allowed directions/units, assertion modalities and the Measurement narrative contract.
- Agent output schema and workflow stages remain fixed in Agent Version code. Only formal vocabulary and catalog values are dynamic.
- V2 Submission carries `entity_links` and `variable_signals` only. Review work packages and decisions cover these same candidate types only.
- V2 Submission freezes both reviewer and adjudicator prompt/model identities. The first `indeterminate` review produces `needs_reanalysis`; the second pass uses the frozen adjudicator identity, and another `indeterminate` result is deterministically quarantined. Persisted review snapshots decide whether an unknown-outcome resume restarts at reviewer or adjudicator and prevent either pass from being silently repeated with a different identity.
- Every V2 EventEntityLink carries the selected formal Entity ID, role, raw `mention` and `evidence_ids`. Its link is valid only when the Evidence contains that mention, an unambiguous abbreviation or a directly normalizable category expression. Company background, EntityRelation, industry-chain membership or analyst knowledge alone cannot create a link.
- V2 VariableSignal carries `measurements[]`; each Measurement contains only `measurement_text` and `evidence_ids`.
- Data performs no numeric or semantic parsing of Measurement text. It enforces only request size, item count/length budgets, nonblank text, Evidence identity and referential integrity.
- Review work packages include the formal resolved Entity identity, type, canonical/name and aliases for every reviewable EntityLink. Independent AI Review must reject a mention-to-ID binding when the formal Entity is merely similar, same-type or industry-related rather than the same object/direct category normalization.
- Independent AI Review checks the complete meaning of each Measurement text against its cited Evidence: every stated number, range, unit, comparison basis, time, forecast/report qualifier and other limitation must be supported by at least one cited Evidence, and the text must not add a claim absent from that Evidence. A failed Measurement review rejects its parent VariableSignal rather than partially accepting or rewriting the prose.
- Measurement inherits the review status and lifecycle of its VariableSignal; it is not a standalone ABox, Observation or independently accepted object.
- Existing DirectImpact and anchor-resolution routes may be removed from the supported Event Semantic API after consumer search confirms no other owner. No V2 compatibility shim will call them.

### Measurement persistence migration

- Use a forward migration; do not edit the Phase One migration already present in shared history.
- Preserve the existing Measurement table and historical numeric columns.
- New V2 writes store natural-language Measurement in the existing raw text storage and Evidence reference; structured numeric/unit/shape columns are unused for new V2 rows.
- Relax the database constraints and nullability that require role, shape, raw/canonical numeric values, normalized units, equality conversion and range ordering for every new row.
- Do not backfill invented normalized values. Historical rows remain unchanged.
- API reads expose V2 Measurement prose and Evidence references. Any legacy structured read compatibility required for existing formal history is a read-only mapping concern, not permission to reintroduce V1 write validation.

### Entity retrieval

- The first channel is an AgentRun-owned Event-batched exact query over Qdrant projection payload: normalized canonical name, aliases and UUID return a unique projected formal candidate or unresolved/ambiguous. It does not call Data Service.
- AgentRun sends all unresolved mentions for one Event to Qdrant in a batch. Each search is filtered by the model-predicted formal Entity Type and returns bounded Top-K candidates.
- Candidate payload includes formal Entity UUID, Entity Type, canonical/name/aliases and a bounded concise description. Optional relation context is limited, deterministic projection content and is never a full graph dump.
- The model receives Event/Evidence context, the verbatim mention and only the returned candidates. It chooses one candidate ID or `no_match`.
- Before Submission, AgentRun deterministically checks that every selected ID is an exact member of that mention's immediately preceding candidate set. An ID copied from another mention, returned by an earlier retry or absent from the candidate set is an invalid model output and never reaches Data.
- Predicted Entity Type must be a member of the pinned dynamic Entity Type directory. An unknown type receives one schema-correction retry. Qdrant searches always retain a validated type filter; if the corrected type still yields no valid candidate, the result is `no_match`. V2 never falls back to untyped recall or automatic binding, and acceptance reporting attributes a demonstrably wrong predicted type to model error.
- Similarity score is ranking evidence only; there is no global auto-accept threshold.
- Unknown, inactive, type-mismatched or stale selected IDs are rejected by Data precheck.
- Data's Submission precheck is the first V2 Data call that resolves the selected Entity ID against formal PostgreSQL state; it remains authoritative even after a Qdrant exact match.
- Qdrant transport failure is a retryable Execution failure. Empty candidate results are valid unresolved outcomes. Invalid candidate IDs are deterministic contract failures.

### Variable Definition retrieval

- Data Context always supplies the complete active/current Variable Definition directory pinned to the current TBox contract because the initial catalog is small.
- The variable-definition Qdrant collection may rank or annotate likely definitions for a Signal, batched per Event and filtered by resolved subject Entity Type.
- The model may only emit a key/version present in the complete pinned directory. Qdrant omission cannot make a valid definition unavailable.
- If Qdrant ranking conflicts with the complete directory, the directory is authoritative: AgentRun preserves all directory entries applicable to the resolved subject Entity Type, treats Qdrant order only as a hint and validates the selected key/version against the complete directory before Submission.
- Data deterministically validates key/version, current status, applicable Entity Type, allowed direction, value type and other TBox constraints.

### PG→Qdrant full rebuild

- Data owns a maintenance CLI that reads Data PostgreSQL and writes Qdrant. AgentRun never receives Data PG credentials.
- The only supported operation in this scope is an explicit full rebuild/seed. There is no incremental sync, CDC, schedule or production automation.
- Use two separate collections: `entity_semantic_v1` and `variable_definition_semantic_v1`.
- Use Qdrant `v1.15.5`, 1024-dimensional cosine vectors and embedding model identity `text-embedding-v4`.
- Data projector and AgentRun query use the same DashScope OpenAI-compatible base URL (`https://dashscope.aliyuncs.com/compatible-mode/v1`), `dimensions=1024`, `encoding_format=float` and frozen model identity. The API Key is a runtime secret and MUST NOT appear in source, configuration files, artifacts, logs or projection payloads.
- Entity points use stable formal Entity UUID identity and project active/current Entity Type, canonical name/name, aliases, deterministic `normalized_names` exact keys and a bounded concise description/context.
- VariableDefinition points use stable definition identity and project formal ID/key/version, names, business definition, domain, applicable Entity Types, value type, allowed units/directions and status.
- Point payload records source identity, projection version, embedding model and content fingerprint.
- Rebuild is repeatable and uses stable point IDs with idempotent upsert. It removes stale projection points only inside the two rebuildable collections and never mutates PostgreSQL facts.
- AgentRun remains paused while a collection is being rebuilt. A failed rebuild leaves Event Semantic disabled until rerun succeeds.
- The research spike collection is never renamed or treated as a formal collection.

### Eino workflow

Reference-first audit (shared read-only clones):

| Reference | Commit | Relevant files inspected | Decision |
| --- | --- | --- | --- |
| `cloudwego/eino` | `922b6a8a233b5233fe47eecee6cd2c005e8c39cd` | `components/embedding/interface.go`; the previously audited `compose/workflow.go` typed Workflow surface | Adopt `embedding.Embedder.EmbedStrings` and retain the typed compiled Workflow. |
| `cloudwego/eino-ext` | `9137edd89e72b72735ede69db1c5ae29178a6e41` | `components/embedding/openai/embedding.go`; `components/retriever/qdrant/retriever.go` | Adopt the official OpenAI-compatible Embedder. Reject the standard Qdrant Retriever at this seam because `Retrieve(ctx, query string)` embeds exactly one string and issues one Qdrant query, which would create mention-level N+1 and cannot express per-mention typed filters in one Event batch. |
| `cloudwego/eino-examples` | `171220631fb7068ead50b7cd964b8c471647117d` | `compose/workflow/1_simple/main.go`; the previously audited typed workflow examples | Retain explicit composition/compile at the application edge. Reject example global callbacks, process-global clients and `context.Background()` during invocation. |

The project-specific gap is Event-level multi-mention retrieval: one batch embedding, per-query
Entity Type filters, Qdrant query-batch transport, exact payload matching and a candidate whitelist/
`no_match` contract. The custom AgentRun adapter is limited to that gap and does not implement an
embedding provider or model protocol.

- Keep a deterministic, typed, acyclic `compose.Workflow`; no ADK, open-ended agent loop or model-controlled Tool loop is introduced.
- Stages are: validate input/resume, generate native candidates, Qdrant exact resolve, batch Qdrant Entity retrieve, constrained Entity selection, complete-directory Variable selection, optional batch Variable ranking, create V2 submission, independent review/finalization, build result.
- Use caller Context through model, Data and Qdrant calls. Work Item retry remains outside Eino.
- Qdrant is invoked through an AgentRun-owned consumer Port. AgentRun's internal data adapter owns Qdrant retrieval; Data Service's adapter owns only PG→Qdrant projection writes. Data Service does not proxy AgentRun queries, and AgentRun never accesses Data PostgreSQL.
- Eino reference-first result: use Eino `embedding.Embedder` and eino-ext `components/embedding/openai` at the AgentRun composition root for the OpenAI-compatible DashScope call. The AgentRun Qdrant adapter receives that interface and invokes exactly one Event-batched `EmbedStrings`; it MUST NOT marshal or decode `/embeddings` wire payloads itself.
- Eino-ext's standard Qdrant Retriever remains rejected for this seam because its `Retrieve(ctx, query string)` contract embeds and queries one string at a time. The smallest justified custom adapter owns only normalized-name exact scroll, per-mention Entity Type filters, one Qdrant query batch, response mapping and candidate whitelist/no-match semantics.
- Data Service MUST NOT import Eino, eino-ext, Workflow, Retriever, Indexer or Agent components. Its Biz Port is an ordinary batch Embed interface and its internal/data adapter independently calls the same OpenAI-compatible service over HTTP. This repository intentionally has one root `go.mod`; ownership is therefore enforced by package imports and deployable binary dependency closures rather than by a nonexistent Data-local module file.
- Model output remains strict JSON for IDs, keys, directions, modality and evidence references. Measurement text is natural language inside that fixed JSON envelope.
- Every model stage is decoded with a strict Agent Version JSON Schema, rejects unknown fields and performs the stage-specific candidate/directory membership checks above. Invalid JSON or a correctable schema violation receives at most one bounded correction call within the Execution deadline. Any second invalid response, including omitted selector coverage, is a terminal model-contract failure; AgentRun does not synthesize `no_match`, submit partial facts or locally complete the model's missing decisions.
- Prompt and Workflow hashes include the new V2 schema, dynamic TBox assembly contract, Qdrant selection protocol and absence of DirectImpact.

### Failure, security and rollout

- Authentication and service-token behavior for Data Event Semantic APIs remains unchanged.
- Qdrant and embedding endpoints, keys and timeouts are typed configuration injected at composition roots. Secrets and Event/Prompt bodies are not logged.
- Read-only Data Context and Qdrant calls may retry only within one total deadline. Submission/review writes retain existing idempotency and unknown-outcome reconciliation.
- Rollout order is: Data forward migration and V2 provider contract; successful Qdrant full rebuild; AgentRun V2 consumer/runtime; enable Event Semantic. The workflow remains paused during mixed versions.
- Rollback disables Event Semantic and rolls back AgentRun/Data binaries. The relaxed forward schema and historical data remain; no down migration is assumed safe.

### Authority updates

- Replace the accepted anchor-resolution ADR with a new accepted decision for Data-owned projection and AgentRun direct Qdrant recall.
- Mark the Anchor Resolution MVP fully superseded.
- Mark DirectImpact-required portions of Phase One superseded while retaining their historical design record.
- Update Data Context so Measurement is an Event-native natural-language annotation and DirectImpact remains a historical/Theme-analysis concept rather than a V2 Event Semantic output.
- Update AgentRun Context with the V2 Agent Version, batch Qdrant retrieval, dynamic TBox and no-DirectImpact contract.

## Testing Decisions

- Tests verify observable behavior through existing Biz, HTTP, compiled Workflow and Adapter seams. They do not assert private node helpers, file layout or internal call order except external call budgets that are part of the contract.
- Data Biz seam proves a Submission with EntityLinks, VariableSignals and natural-language Measurements can pass without any DirectImpact, numeric parsing, canonical values or normalized units.
- Data Biz seam proves unknown Evidence, unknown Entity, wrong Entity Type, stale Variable Definition, inapplicable subject type, invalid direction and modality still fail deterministically.
- Data API/HTTP seam proves V2 Context, Submission, Review and Read DTOs reject unknown fields and contain no DirectImpact/DirectTarget/Rule contract.
- AgentRun integration tests prove V2 makes zero Data Entity Resolution/Search calls: exact and vector candidate retrieval both go directly to Qdrant, while Data first validates the selected Entity during Submission.
- Migration seam proves historical DirectImpact and structured Measurement rows survive, new natural-language-only Measurement rows persist, and the forward migration is safe on existing history.
- Provider/Consumer fixture seam uses identical V2 JSON fixtures in Data and AgentRun strict decoders.
- AgentRun/Eino seam invokes the real compiled Workflow with fake model, Data and Qdrant adapters. It covers exact match, batched fallback, constrained candidate choice, `no_match`, dynamic TBox, no-number Signal, multiple Measurement texts, review and resume.
- AgentRun/Eino seam also proves cross-mention/stale/invented candidate IDs are rejected before Submission, unknown predicted types receive one correction attempt, empty typed recall becomes `no_match`, and every second invalid JSON/schema response—including omitted selector coverage—terminates without accepting partial facts.
- AI Review fixtures prove multiple Measurement texts are accepted only when each cited Evidence supports every quantitative element; unsupported prose rejects the parent Signal and is never rewritten into a different measurement.
- Qdrant Adapter seam uses real Qdrant `v1.15.5` to verify collection vector configuration, stable point identity, type filter, batch request, Top-K, empty result, cancellation and sanitized failure.
- Projector Data integration seam uses a real PostgreSQL test database and Qdrant to prove active/current-only reads, deterministic documents, idempotent rerun and stale projection removal.
- Configuration seam validates Qdrant/embedding endpoint, collection names, vector size, model identity, timeout and batch limits without accepting secrets from files.
- Architecture seam proves AgentRun does not import Data implementation or access Data PostgreSQL, Data remains projection-write owner, AgentRun remains Qdrant-query owner and Data does not proxy Qdrant retrieval.
- Cross-service E2E runs Data, AgentRun, PostgreSQL, Qdrant and the configured embedding/model adapters locally. UAT is not deployed.
- The approximately 100 Event acceptance reuses the previously fixed local sample and reports completed/terminal/failed counts, failure reasons, mention counts, Qdrant exact matches, Qdrant vector fallbacks, no-match counts, accepted/rejected EntityLinks and VariableSignals, Measurement inclusion/rejection through parent Signal, zero DirectImpact, zero Direct Target/Rule calls, zero Data Entity Resolution/Search calls, prompt/context bytes, model latency, Qdrant batch counts/candidates/p50/p95, Data API payload/latency and old/reference comparison.
- Manual sampling classifies remaining differences as ABox/TBox gaps, model errors or workflow defects.
- The fixed NVIDIA/Amkor Event is reported separately and must identify supported formal companies/industry-chain-related entities and objective Signals/Measurement prose without generating advanced-packaging investment conclusions or cross-Entity DirectImpact.
- Focused red/green tests precede implementation at Data Biz and compiled Workflow seams. Completion runs affected Data and AgentRun suites, migration/fixture/architecture checks, `gofmt`, `go vet`, binary builds, cross-service E2E and Standards/Spec code review.

## Out of Scope

- PostgreSQL change-triggered incremental or real-time Qdrant synchronization.
- CDC, scheduler, production deployment automation or a continuously running projector.
- Qdrant production HA, backup, monitoring and capacity engineering.
- Production embedding hosting.
- Neo4j or graph retrieval.
- DirectImpact generation in Event Semantic.
- Theme, Reason Tree, analyst inference or investment conclusion generation.
- Automatic creation of missing Entity or Variable Definition facts.
- A general-purpose search platform or hybrid search engine.
- Deleting historical DirectImpact or structured Measurement data.
- UAT deployment or PR merge.

## Further Notes

- 2026-08-01 已把 Spec 发送到 Codex 的“事件推理模型”会话。该会话首轮结论为 `revise`：要求 V2 精确匹配也直接走 Qdrant、收紧 EventEntityLink 的 Evidence 边界，并明确 Data 只写 projection、AgentRun 自己查询 Qdrant。本版纳入三项修正后，同一会话复核结论为 `confirm`、无阻断问题，并确认可以正式冻结进入实施。
- The 2026-08-01 Qdrant spike is evidence for candidate recall only. Its earlier suggestion that AgentRun
  own projection synchronization is superseded: Data owns PG→Qdrant projection; AgentRun owns query use.
- The vector score distributions of correct and dangerous wrong candidates overlap. Any future auto-bind
  threshold requires a new reviewed decision and labeled evaluation data.
- Measurement prose is intentionally not suitable for SQL aggregation or arithmetic. A future need for
  calculation requires a separately designed Observation/structured measurement model rather than
  silently reintroducing normalization into this Event Semantic contract.
- Highest acceptance seam and the fixed 100 Event/fixed NVIDIA Event dataset were explicitly established
  by the user; no additional testing interview is required before publishing this Spec.
