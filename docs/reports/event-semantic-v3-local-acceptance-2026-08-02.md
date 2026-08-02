# Event Semantic V3 local corrective acceptance report

Date: 2026-08-02

Issue: #164

PR: #165

Environment: local only; no UAT deployment

## Outcome

The effective fixed 100 Event sample completed with **41 accepted, 59 rejected and 0 execution
failures**. It extracted 237 grounded mentions and produced 54 accepted EventEntityLinks. The accepted
links were independently checked: 47 were canonical/alias identities and seven were defensible short-name
or normal-form matches. No accepted wrong-object binding was found in the sample.

The separately executed NVIDIA/Amkor Event was accepted with one NVIDIA EventEntityLink, one
VariableSignal and one natural-language Measurement. Amkor remained `no_match`; it was not incorrectly
bound to Onsemi. DirectImpact and all prohibited downstream reasoning remained zero.

## Corrective scope

- Stage envelopes now require their named top-level arrays. `null`, `{}`, missing/null/wrong-type fields,
  duplicate keys, unknown fields and trailing JSON receive one bounded repair and then fail as a model
  contract error.
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
| Accepted / rejected / failed Events | 41 / 59 / 0 |
| Raw mentions | 237 |
| Exact hits / vector fallbacks / no_match | 53 / 186 / 154 |
| EventEntityLink accepted / rejected | 54 / 29 |
| VariableSignal accepted / rejected | 0 / 2 |
| Measurement emitted / accepted / rejected | 1 / 0 / 1 |
| Primary / secondary selection decisions | 233 / 3 |
| Stage violations / model-contract failures | 0 / 0 |
| DirectImpact / Direct Target / transmission-rule calls | 0 / 0 / 0 |
| Theme / Reason Tree outputs | 0 / 0 |

The full run initially had seven 10-second embedding/Qdrant deadline failures. With explicit user
authorization, only those seven local acceptance leases were expired and rerun at concurrency one. No
rows were deleted. The final effective result replaces those records and later targeted regressions in
the original 100 records; it does not count retries as additional Events.

Artifacts:

- Base run: `/private/tmp/event-semantic-v3-corrective-final.json`, SHA-256
  `77b7a3655e1fee22f48b68061eed8c5e9d3dcf0f6fc969d617ec32adeaa51e7a`.
- Timeout retry: `/private/tmp/event-semantic-v3-corrective-retry.json`, SHA-256
  `a4742448851c947f1874c4b9601c29e306877ff252b31cc388ff1db06e663155`.
- Final no-handwritten-guard overrides:
  `/private/tmp/event-semantic-v3-corrective-final-noguard.json`, SHA-256
  `58d90d0d8e6bf5fdfed695f586b672c3d2beb163d2f462c0bec22fc3b6546ddf`.
- Related-institution regression:
  `/private/tmp/event-semantic-v3-related-institution-regression.json`, SHA-256
  `422d051d0caa79b27c564a69d376b8ac0ffccc97dc8633370961cde789bc7dea`.
- Effective independent audit:
  `/private/tmp/event-semantic-v3-corrective-final-effective-audit.json`, SHA-256
  `1c17c1ac4f5698101e1418400fff5fd9545f9378cb3121a3eb82260702a6da9c`.

## Rejection classification

The effective audit cross-checked Stage output, Qdrant candidates, active TBox and formal PG identity.
It does not collapse missing formal data and retrieval failures, and does not treat every role rejection
as a Selector error.

| Required category | Events | Meaning in this sample |
| --- | ---: | --- |
| `correct_reject` | 0 | No Event was placed in this residual bucket |
| `abox_missing` | 41 | Entity-like mentions lacked a canonical/alias identity in formal PG data |
| `tbox_out_of_scope` | 0 | Newly available active types were included in this run |
| `mention_extraction_miss` | 0 | No audited whole-Event miss remained |
| `retrieval_miss` | 0 | No formal exact identity was absent from the audited candidate path |
| `selector_false_reject` | 0 | No confirmed same-object candidate remained wrongly rejected |
| `review_reject` | 18 | Candidate proposed, then rejected by deterministic or AI same-object/role review |
| `model_contract_failure` | 0 | All strict stage envelopes parsed after at most one repair |

`review_reject` includes safety decisions such as rejecting `菲律宾警方 -> 菲律宾` and rejecting the
withdrawal action phrase as an Israel Entity. These are related concepts, not the same formal object, so
loosening the role whitelist would be incorrect.

## Accepted link quality

All 54 accepted links were inspected. Forty-seven were formal canonical/alias matches. Seven required
manual semantic confirmation and were accepted as the same object:

- `福特 -> 福特汽车`
- `丰田 -> 丰田汽车`
- `乌克兰总统泽连斯基 -> 泽连斯基`
- `欧洲央行 -> 欧洲中央银行` in two Events
- `拉加德 -> 克里斯蒂娜·拉加德`
- `人民银行 -> 中国人民银行`

Previously observed wrong bindings are absent: Lula was not bound to von der Leyen, eurozone was not
bound to the EU, CIEB was not bound to MOFCOM, Robotaxi was not bound to an autonomous-driving system,
Amkor was not bound to Onsemi, and `国新办` was not bound to `国务院`.

Role review also confirmed that statement sources and discussed objects are no longer mechanically both
labelled `actor`. The remaining role risk is model variability on context versus event-subject roles, not
an observed wrong-object acceptance in this sample.

## Signal and Measurement

The 100 comparison Events emitted two Signals, both rejected because their parent entity candidates were
rejected. One natural-language Measurement was emitted and rejected with its parent Signal. This confirms
candidate-level isolation: Signal/Measurement failure did not remove other valid links.

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
| Prompt bytes total / average | 1,897,567 / 18,975 |
| Prompt bytes min / p50 / p95 / max | 9,606 / 17,290 / 31,015 / 38,363 |
| Context bytes total / average | 1,793,129 / 17,931 |
| Context bytes min / p50 / p95 / max | 17,688 / 17,888 / 18,242 / 18,600 |
| Model calls / total latency | 274 / 589,141 ms |
| Event latency p50 / p95 | 5,329 / 21,887 ms |
| Qdrant exact / vector Event-batch calls | 100 / 84 |
| Qdrant candidates / latency p50 / p95 | 1,915 / 3 / 2,366 ms |
| Data API calls / request bytes | 459 / 168,761 |
| Data API latency p50 / p95 | 8 / 17 ms |

There is no mention-level embedding/query N+1. The seven initial deadline failures show that the local
embedding path is sensitive to high concurrent acceptance load; this is a performance risk, not a final
workflow failure.

## Comparison

| Run | Accepted | Rejected | Failed |
| --- | ---: | ---: | ---: |
| V2 baseline | 16 | 57 | 27 |
| Earlier V3 acceptance | 20 | 80 | 0 |
| Corrective V3 effective result | 41 | 59 | 0 |

The correction kept V3's elimination of whole-Event model-contract failures while recovering valid links
previously lost to Stage A misses, type-filtered retrieval and over-conservative selection. Cross-type
recall did not introduce an observed accepted wrong-object binding in the manually audited sample.

## Remaining risks

- The largest gap is formal ABox coverage: 41 rejected Events contain entity-like mentions without a
  formal canonical/alias identity.
- Eighteen Events ended after candidate review; these require future per-case data/model review, not a
  blanket relaxation of identity or role constraints.
- Alias governance should add confirmed short names to formal Entity aliases instead of accumulating code
  heuristics.
- The fixed sample is an auditable regression set, not a global precision/recall guarantee.
- Local concurrent embedding showed a long-tail latency issue; production resilience is outside this
  one-time local acceptance scope.

## Verification

- AgentRun `go test ./...` and `go vet ./...`: pass.
- Data `go test ./...` and `go vet ./...`: pass.
- Repository contract `go test ./...`: pass.
- Strict stage-envelope, startup config, Qdrant provenance, batch retrieval, OpenAPI confidence and Data
  precheck tests: pass.
- Data projector test confirms stable outer Point IDs and no duplicate payload `point_id`: pass.
- `git diff --check`: pass.
- No UAT/production deployment and no PR merge were performed.
