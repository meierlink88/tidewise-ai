# Event Semantic V3 local corrective acceptance report

Date: 2026-08-02

Issue: #164

PR: #165

Environment: local only; no UAT deployment

## Outcome

The fixed 100 Event sample completed with **45 accepted, 55 rejected and 0 execution failures**. It
extracted 240 grounded mentions and produced 61 accepted EventEntityLinks. Manual inspection found no
accepted wrong-object binding. The three role regressions named by the corrective review all passed.

The separately executed NVIDIA/Amkor Event was accepted with one NVIDIA EventEntityLink, one
VariableSignal and one natural-language Measurement. Amkor remained `no_match`; it was not incorrectly
bound to Onsemi. DirectImpact and all prohibited downstream reasoning remained zero.

## Corrective scope

- Stage envelopes now require their named top-level arrays. `null`, `{}`, missing/null/wrong-type fields,
  duplicate keys, unknown fields and trailing JSON receive one bounded repair and then fail as a model
  contract error.
- A malformed item inside a valid top-level array is isolated with its candidate key (or array position),
  while valid sibling Mention, Selection, Signal or Review items continue. This is deliberately a V3-only
  parser seam, not a generic tolerant JSON framework.
- Stage A extracts only grounded raw mentions and Evidence IDs. Entity Type, role, Signal and Measurement
  are no longer generated there.
- Exact and vector recall run across active, event-link-allowed Entity Types. A model-predicted type can no
  longer exclude the correct candidate before search.
- The Selector chooses only a supplied formal candidate or `no_match`; an independent Reviewer checks
  same-object identity. Unique canonical/alias exact matches remain deterministic.
- No hand-written abbreviation or substring identity rules were added. Legitimate alias gaps belong in
  formal alias data; related-but-different objects remain rejected.
- Role whitelists remain strict. A role failure is not automatically classified as a Selector false reject.
- Signal extraction runs only after entity resolution. Signal or Measurement rejection does not revoke an
  accepted EventEntityLink.
- AgentRun validates Qdrant outer point ID against payload `entity_id` and checks projection provenance.
  Payload does not duplicate `point_id`; Data still validates formal ID, active status and type against PG.
- `EMBEDDING_API_KEY` is validated at startup, before a Work Item or Context Lease is acquired.
- OpenAPI review candidate types and decimal confidence validation now match runtime behavior.
- AgentRun retrieval and Data projection HTTP adapters preserve `context.Canceled` and
  `context.DeadlineExceeded`; cancellation is no longer disguised as a retryable remote failure.

## Ownership and dependency boundaries

- PostgreSQL is the fact authority; Qdrant is a rebuildable recall projection and owns no accepted state.
- Data owns PG-to-Qdrant projection, uses a plain OpenAI-compatible HTTP embedding adapter and has no
  Eino/eino-ext dependency.
- AgentRun owns direct Qdrant lookup. Its thin Event-batch adapter receives an injected official Eino
  `embedding.Embedder`, performs one batch embedding and one Qdrant query batch for unresolved mentions.
- Both sides use DashScope `text-embedding-v4`, 1024 dimensions and cosine distance.
- Entity and active/current VariableDefinition data are vectorized. Event, Evidence, accepted facts,
  DirectImpact and relationship graphs are not vectorized.

## Effective 100 Event metrics

The fixed sample ID CSV SHA-256 is
`820a9a280a86808a1df481273140f40fbd0963a641d7f29a4e8db06ae85044b6`.

| Metric | Result |
| --- | ---: |
| Accepted / rejected / failed Events | 45 / 55 / 0 |
| Raw mentions | 240 |
| Exact hits / vector fallbacks / no_match | 67 / 175 / 156 |
| EventEntityLink accepted / rejected | 61 / 23 |
| VariableSignal accepted / rejected | 1 / 0 |
| Measurement emitted / accepted / rejected | 1 / 1 / 0 |
| Candidate-level isolations / stage violations | 48 / 0 |
| Model-contract failures | 0 |
| DirectImpact / Direct Target / transmission-rule calls | 0 / 0 / 0 |
| Theme / Reason Tree outputs | 0 / 0 |

Artifacts:

- Final run: `/private/tmp/event-semantic-v3-corrective-rerun-final.json`, SHA-256
  `dd9cd1cc8d03ffa88b45b4590e7e486570c20c2c3852a5f4ffc6e7ea394e40e6`.
- Frozen system-external Mention reference: `/private/tmp/uat_event_mentions_deepseek.json`, SHA-256
  `06ba99e669f5d5bba4a2cba993e5f74bcf3b7ef8af1ddaaab92584f2c1345bc7`.
- Deterministic reject audit:
  `/private/tmp/event-semantic-v3-corrective-rerun-final-audit-v2.json`, SHA-256
  `57384c6dced9b523febc58fd891317267fc9253228c8248f5fc58de0a3e73630`.

## Rejection classification

The Data-owned read-only audit command cross-checks the frozen external Mention reference, Stage output,
Qdrant candidates, active TBox and formal PG identity. Its input hashes and per-Event verdicts make the
classification reproducible. It does not collapse missing formal data and retrieval failures, and does
not treat every role rejection as a Selector error.

| Required category | Events | Meaning in this sample |
| --- | ---: | --- |
| `correct_reject` | 2 | No acceptable formal identity after deterministic candidate audit |
| `abox_missing` | 28 | Entity-like mentions lacked a canonical/alias identity in formal PG data |
| `tbox_out_of_scope` | 0 | Newly available active types were included in this run |
| `mention_extraction_miss` | 15 | The frozen reference/PG showed an in-scope formal identity absent from Stage A |
| `retrieval_miss` | 0 | No formal exact identity was absent from the audited candidate path |
| `selector_false_reject` | 0 | No confirmed same-object candidate remained wrongly rejected |
| `review_reject` | 10 | Candidate proposed, then rejected by deterministic or AI same-object/role review |
| `model_contract_failure` | 0 | All strict stage envelopes parsed after at most one repair |

`review_reject` includes safety decisions such as rejecting `菲律宾警方 -> 菲律宾` and rejecting the
withdrawal action phrase as an Israel Entity. These are related concepts, not the same formal object, so
loosening the role whitelist would be incorrect.

## Accepted link quality

All 61 accepted links were inspected. No wrong-object binding was found. Previously observed wrong
bindings remain absent: Amkor was not bound to Onsemi; Lula was not bound to von der Leyen; eurozone was
not bound to the EU; CIEB was not bound to MOFCOM; Robotaxi was not bound to an autonomous-driving
system; and `国新办` was not bound to `国务院`.

The three required role goldens passed in the real run:

- `特朗普发布对伊朗48小时通牒`: Iran = `event_object`.
- `美国暂停对伊朗军事打击`: Iran = `affected_entity`.
- `巴西对原产于中国的钢瓶发起调查`: China = `event_object`.

Broader role inspection still found model variability in context versus subject/object choices; this is
recorded as a remaining quality risk rather than hidden in the reject classification. The accepted-object
precision in this sample is 61/61; the three blocking role regressions are 0/3 erroneous after correction.

## Signal and Measurement

The 100 comparison Events emitted and accepted one Signal and one natural-language Measurement. The 48
recorded item-level isolations did not become whole-Event failures or revoke accepted sibling links.

Measurement remains analyst-readable natural language with Evidence lineage; no numeric-shape database
gate was reintroduced. The fixed NVIDIA Event separately accepted `order_value / increase / actual` with
Measurement `15亿美元`.

## Fixed NVIDIA / Amkor Event

Event:

> 2026年7月，英伟达与全球第二大半导体封测厂商安靠科技达成了一项价值15亿美元的战略合作，并首次将“预付款锁定产能”的模式延伸至第三方封测厂。

Result:

- Stage A extracted `英伟达` and `安靠科技` with Evidence lineage.
- NVIDIA resolved to the formal company and produced one accepted EventEntityLink.
- Amkor had no formal same-object candidate and remained `no_match`.
- One `order_value` VariableSignal and natural-language Measurement `15亿美元` were accepted.
- No packaging-chain propagation, DirectImpact, Theme or investment conclusion was emitted.

## Size and latency

| Metric | Result |
| --- | ---: |
| Prompt bytes total / average | 1,955,952 / 19,559 |
| Prompt bytes min / p50 / p95 / max | 2,568 / 18,237 / 31,067 / 38,057 |
| Context bytes total / average | 1,793,127 / 17,931 |
| Context bytes min / p50 / p95 / max | 17,688 / 17,888 / 18,242 / 18,600 |
| Model calls / total latency | 273 / 330,248 ms |
| Event latency p50 / p95 | 3,501 / 5,773 ms |
| Qdrant exact / vector Event-batch calls | 99 / 82 |
| Qdrant candidates / latency p50 / p95 | 1,819 / 3 / 300 ms |
| Data API calls / request bytes | 459 / 169,086 |
| Data API latency p50 / p95 | 8 / 18 ms |

There is no mention-level embedding/query N+1. The final run had no execution or model-contract failure.

## Comparison

| Run | Accepted | Rejected | Failed |
| --- | ---: | ---: | ---: |
| V2 baseline | 16 | 57 | 27 |
| Earlier V3 acceptance | 20 | 80 | 0 |
| Corrective V3 final result | 45 | 55 | 0 |

The correction kept V3's elimination of whole-Event model-contract failures while recovering valid links
previously lost to Stage A misses, type-filtered retrieval and over-conservative selection. Cross-type
recall did not introduce an observed accepted wrong-object binding in the manually audited sample.

## Remaining risks

- The largest gaps are formal ABox coverage (28 Events) and Stage A Mention recall (15 Events).
- Ten Events ended after candidate review; these require future per-case data/model review, not a
  blanket relaxation of identity or role constraints.
- Role selection beyond the three blocking goldens remains model-variable, especially for contextual
  economies and institutions; further prompt/data evaluation is warranted before treating role accuracy
  as globally solved.
- Alias governance should add confirmed short names to formal Entity aliases instead of accumulating code
  heuristics.
- The fixed sample is an auditable regression set, not a global precision/recall guarantee.
- The audit depends on the frozen external Mention reference and current formal PG identities; its hashes
  must remain attached whenever the result is compared or reproduced.

## Verification

- AgentRun `go test ./...` and `go vet ./...`: pass.
- Data `go test ./...` and `go vet ./...`: pass.
- Repository contract `go test ./...`: pass.
- Strict stage-envelope, startup config, Qdrant provenance, batch retrieval, OpenAPI confidence and Data
  precheck tests: pass.
- Data projector test confirms stable outer Point IDs and no duplicate payload `point_id`: pass.
- `git diff --check`: pass.
- No UAT/production deployment and no PR merge were performed.
