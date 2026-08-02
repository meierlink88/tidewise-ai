# Event Semantic V2 local acceptance report

Date: 2026-08-01
Issue: #164
Environment: local only; no UAT deployment

## Outcome

The V2 Data and AgentRun path completed real local submissions through PostgreSQL, DashScope
`text-embedding-v4`, Qdrant and the configured generator/reviewer models. The authoritative final 100
Event run finished with 16 accepted submissions, 57 terminal rejected submissions and 27 terminal
model-contract failures. It produced no DirectImpact and made no Direct Target, transmission-rule or
Data entity-search calls.

The architecture boundary is working, but semantic quality is not ready for broad production enablement.
The workflow conservatively returned `no_match` for 153 of 212 mentions, while 27 Events terminated after
the model still violated a bounded vocabulary or candidate-coverage contract after its single repair.
Manual review of all 18 accepted EntityLinks found one definite wrong binding: `腾讯QQ` was bound to
`腾讯控股`. The remaining 17 accepted links were direct identity or normal-form matches.

## Frozen configuration and projection

- PostgreSQL remains the only fact authority. Qdrant owns no accepted state.
- Qdrant version: v1.15.5.
- Embedding: DashScope OpenAI-compatible API, `text-embedding-v4`, 1024-dimensional dense vectors,
  cosine distance.
- Data owns the full rebuild projector and uses an ordinary HTTP embedding adapter. Its deployable
  package closure contains no Eino/eino-ext dependency.
- AgentRun owns queries and injects the official Eino `embedding.Embedder`, backed by
  `github.com/cloudwego/eino-ext/components/embedding/openai`, into a thin Event-batch Qdrant adapter.
- `entity_semantic_v1`: green, 5,029 points, 1024/Cosine.
- `variable_definition_semantic_v1`: green, 12 points, 1024/Cosine.
- The projector read only active/current formal facts and passed a real idempotent rebuild test, including
  stale-point removal. No CDC, incremental synchronization, scheduler or UAT/production deployment was
  added.

Only the following formal facts are vectorized:

1. Entity identity and retrieval text: UUID, Entity Type, canonical/name, aliases and a bounded formal
   type/layer description.
2. VariableDefinition identity and retrieval text: ID/key/version, names, business definition/domain,
   applicable Entity Types, value type, allowed units/directions and status.

Events, Evidence, EventEntityLink, VariableSignal, Measurement, DirectImpact, accepted state and full
EntityRelation/industry graphs are not vectorized by this projector.

## Final 100 Event result

The run reused the fixed local sample whose ID CSV SHA-256 is
`820a9a280a86808a1df481273140f40fbd0963a641d7f29a4e8db06ae85044b6`.
The authoritative raw report is `/private/tmp/event-semantic-v2-acceptance-final-v6.json`, SHA-256
`e88effd06d3496fdb45af2eb58ec593e92e9b3adf25a7776f2c7fc7aa254611d`.

| Metric | Result |
| --- | ---: |
| Accepted / rejected / failed Events | 16 / 57 / 27 |
| Native mentions | 212 |
| Qdrant exact hits | 15 |
| Vector fallbacks | 197 |
| `no_match` | 153 |
| EntityLink accepted / rejected | 18 / 0 |
| VariableSignal accepted / rejected | 1 / 0 |
| Measurement emitted | 1 |
| Measurement accepted / rejected with parent Signal | 1 / 0 |
| DirectImpact rows | 0 |
| Direct Target / transmission-rule calls | 0 / 0 |
| Data entity resolution/search calls | 0 |

The 27 failures are all terminal model-contract failures after the one allowed repair:

| Failure | Count |
| --- | ---: |
| `selection_coverage_invalid` | 10 |
| `mention_role_invalid` | 5 |
| `mention_evidence_support_invalid` | 4 |
| `mention_entity_type_invalid` | 3 |
| `signal_key_invalid` | 2 |
| `mention_evidence_ids_invalid` | 1 |
| `mention_evidence_support_invalid,mention_entity_type_invalid` | 1 |
| `mention_role_invalid,mention_evidence_support_invalid` | 1 |

There were no Data 500, embedding transport or Qdrant transport failures in the authoritative batch.
Two preceding diagnostic runs were discarded: one used a stale running Data binary and one encountered
the still-active leases left by that failed run. Neither is merged into the final metrics.

## Entity quality review

All 18 EntityLinks accepted in the authoritative run were manually inspected by raw mention, formal
canonical name and Entity Type. Seventeen were direct identity or normal-form matches, including:

- `央行` / `人民银行` → `中国人民银行`;
- `日本央行` → `日本银行`;
- `欧洲央行` → `欧洲中央银行`;
- `欧洲央行行长拉加德` → `克里斯蒂娜·拉加德`;
- `WTI原油期货` → `WTI原油`;
- `福特` → `福特汽车`;
- direct Trump, Zelensky, Tesla, Fed and White House matches.

One accepted binding is a definite false positive:

| Mention | Bound formal Entity | Classification |
| --- | --- | --- |
| 腾讯QQ | 腾讯控股 (`company`) | wrong object/type normalization; the product/service mention must not become its parent company |

This means the independent reviewer is materially safer than blind similarity binding but is not yet
sufficient by itself: a formal candidate whitelist prevents invented IDs, not an incorrect choice among
real IDs.

A title-level omission sample was checked against the formal ABox:

| Mention | Formal exact/alias match | Classification |
| --- | --- | --- |
| 大和资本市场 | none | ABox gap |
| 杰富瑞 | none | ABox gap |
| QQ宠物 | none | ABox gap |
| 安靠科技 | none | ABox gap |
| 伊朗 | `伊朗` (`economy`) exists | model/workflow miss |
| 美国 | `美国` (`economy`) exists | model/workflow miss |
| 腾讯QQ | none | ABox gap followed by unsafe parent-company binding |

The total omission count remains the measured 153 `no_match` outcomes. This sample does not claim that
all 153 are errors; it demonstrates that they contain both correct abstentions caused by missing ABox
objects and false negatives where a formal identity already exists.

## VariableSignal and Measurement quality

The authoritative 100 Event batch accepted one Signal:

- `WTI原油` / `market_price` / decrease / actual, with narrative Measurement
  `跌逾6%至每桶83.84美元`.

The Measurement was stored as natural-language text plus Evidence IDs; the legacy structured numeric,
range, unit and normalization columns remained null. Data performed no semantic or numeric interpretation.
Reviewer acceptance applied to the parent Signal after checking Evidence fidelity. Direction-only Signals
remain valid, and multiple narrative Measurements per Signal are covered by contracts and tests.

One accepted Signal is too small a sample to estimate broad VariableDefinition quality. The low yield is a
model/TBox coverage problem rather than a numeric Measurement rejection: the final batch recorded no
rejected Signal or Measurement.

## Size and latency

| Metric | Result |
| --- | ---: |
| Prompt bytes total / average / min / max per Event | 1,839,781 / 18,398 / 10,277 / 36,729 |
| Context bytes total / average / min / max per Event | 905,519 / 9,055 / 8,812 / 9,724 |
| Model calls | 218 |
| Model latency total / average / p50 / p95 per Event | 409,148 / 4,091 / 3,623 / 7,788 ms |
| Event latency p50 / p95 | 3,844 / 8,133 ms |
| Qdrant exact Event-batch calls | 76 |
| Qdrant vector Event-batch calls | 72 |
| Qdrant candidates returned | 830 |
| Qdrant latency samples / p50 / p95 | 148 / 3 / 328 ms |
| Data API calls | 389 |
| Data request payload bytes | 107,595 |
| Data latency p50 / p95 | 7 / 15 ms |

Every Event with unresolved mentions used one `EmbedStrings` batch and one Qdrant query batch regardless
of mention count. Exact lookup is also Event-batched. There is no mention-level embedding or Qdrant N+1.

## Fixed NVIDIA / Amkor Event

Event:

> 2026年7月，英伟达与全球第二大半导体封测厂商安靠科技达成了一项价值15亿美元的战略合作，并首次将“预付款锁定产能”的模式延伸至第三方封测厂。

The final dedicated run completed in 6,632 ms with three model calls and was accepted:

- `英伟达` exact-matched and was accepted as the formal NVIDIA company Entity.
- `半导体封测` vector-matched and was accepted as the formal `集成电路封装测试产业链` Entity.
- `安靠科技` has no formal ABox Entity and ended as `no_match`; no similar company was substituted.
- NVIDIA received an accepted `order_value / increase / actual` VariableSignal with narrative Measurement
  `价值15亿美元`.
- EntityLink / Signal / Measurement accepted counts were `2 / 1 / 1`; DirectImpact remained zero.
- No advanced-packaging investment conclusion, cross-Entity transmission or Theme inference was emitted.

The raw fixed report is `/private/tmp/event-semantic-v2-fixed-final-v5.json`, SHA-256
`3b1ca9c014548282d6781e218282714f816f39e804609d14c1dbc5385de6ae12`.

## Comparison with prior behavior and external reference

- Old V1 formal history for the same sample covered only 39 Events: 24 rejected and 15 superseded
  submissions, 10 links (all superseded), and zero accepted Signals, Measurements or DirectImpacts. V2
  produces current formal links and narrative Measurements without requiring DirectImpact.
- A pre-final V2 run that used a noncompliant local selector-coverage fallback appeared to produce
  23 accepted, 65 rejected and 12 failed Events. Removing that fallback reduced apparent completion to
  16 / 57 / 27 but restored the frozen failure semantics: the second invalid model response is terminal
  and omitted candidate decisions are never synthesized locally.
- The earlier system-external DeepSeek mention analysis produced 265 mentions over the same 100 Events,
  compared with V2's 212. It also used 86 mentions with types such as economy, index, instrument, market
  and metric that are not all available under the current formal dynamic Entity Type directory. V2
  correctly refuses to turn an unconstrained external taxonomy into formal TBox.
- Similarity scores for correct and dangerous wrong candidates overlap. The `腾讯QQ` false positive shows
  why lowering a global threshold or auto-binding the top candidate would reduce safety rather than solve
  the remaining recall problem.

## Remaining differences by owner

### ABox gaps

- Missing formal companies and aliases, including Amkor, Daiwa Capital Markets and Jefferies.
- Missing product/service identities such as QQ Pet and Tencent QQ encourage either correct abstention or,
  if reviewer judgment fails, unsafe parent-company normalization.

### TBox gaps

- Only 12 active/current VariableDefinitions were projected.
- The formal Entity Type directory does not cover all categories used by the external reference analysis.

### Model errors

- Twenty-seven hard controlled-vocabulary, Evidence-lineage, Signal-key or selector-coverage failures remain.
- One of 18 accepted EntityLinks is a definite wrong-object binding.
- Formal `伊朗` and `美国` identities were missed in the title-level omission sample.

### Workflow issues corrected in this change

- Removed DirectImpact, Direct Target, transmission-rule and graph-propagation dependencies.
- Replaced mention-level retrieval with Event-batched exact and vector Qdrant calls.
- Added strict candidate whitelist/no-match behavior; a second invalid or incomplete selector response now
  terminates rather than being locally completed.
- Deduplicated multiple mentions resolving to the same formal Entity before Data submission.
- Froze reviewer and adjudicator prompt/model identities and resumed unknown outcomes from persisted review
  snapshots.
- Added formal resolved Entity identity to the independent reviewer package.
- Kept historical audit reads tolerant of older persisted packages while requiring complete resolved Entity
  snapshots for resumable V2 review work.

## Verification

### Existing test cleanup audit

- `agent-run/backend/cmd/server/event_semantics_synthetic_e2e_test.go`: `obsolete`. It encoded the V1
  DirectImpact, Data resolution/search and synthetic anchor-routing workflow that V2 explicitly removes.
- `analyse-data-service/backend/internal/biz/eventsemantics/precheck_test.go`: `consolidated`. V2 formal-ID,
  Evidence, TBox, narrative Measurement and zero-DirectImpact behavior is covered by
  `precheck_v2_test.go` plus the real PostgreSQL V2 seam.
- `analyse-data-service/backend/internal/biz/eventsemantics/service_pagination_test.go`:
  `duplicated-by-stronger-seam`. Supported cursor pagination remains covered at the API/HTTP seam; the
  removed assertions exercised V1 service internals and retired resolution capabilities.
- `analyse-data-service/backend/internal/service/event_semantics_pagination_compatibility_test.go`:
  `obsolete`. It existed solely for the old compatibility surface; Issue #164 rejects V1 behavior
  compatibility.
- `analyse-data-service/backend/internal/service/event_semantics_postgres_integration_test.go`:
  `consolidated`. The smaller V2 PostgreSQL integration covers retained transactional risks, narrative
  persistence, review/adjudication, lineage supersession and historical-table preservation without keeping
  the retired DirectImpact/search matrix.

### Completed checks

- Root, AgentRun and Data `go test ./...`: pass.
- AgentRun and Data `go vet ./...`: pass.
- Relevant server, acceptance, projector and migration binaries: build pass.
- PostgreSQL migration 000001 through 000036 on fresh real test databases: pass, including 35→36
  historical Measurement Evidence backfill and a post-migration legacy `evidence_id`-only write.
- AgentRun migration 001 through 011 on a fresh real test database, including V2 Agent Version seed and
  rollback-preparation replay: pass.
- Real PostgreSQL V2 integration: narrative Measurement with structured numeric fields null, zero
  DirectImpact, reviewer/adjudicator finalization, resume semantics and supersession: pass.
- Real PostgreSQL → Qdrant projector: active/current filtering, 1024/Cosine, stable IDs, repeated rebuild and
  stale-point deletion: pass.
- Real DashScope/Eino Embedder/Qdrant fixed retrieval seam: pass.
- Repository dependency/architecture contracts: pass, including no Eino in the Data deployable closure,
  official Eino Embedder injection in AgentRun, one batch `EmbedStrings` plus one Qdrant batch query, and no
  custom AgentRun `/v1/embeddings` wire codec.
- `git diff --check`: pass.

The frozen Issue #164 Spec was corrected to make `text-embedding-v4`/1024 the formal contract and retain
BAAI/512 only as the research-spike baseline.
