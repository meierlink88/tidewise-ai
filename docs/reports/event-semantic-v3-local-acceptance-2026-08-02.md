# Event Semantic V3 local acceptance report

Date: 2026-08-02
Issue: #164
Environment: local only; no UAT deployment

## Outcome

The Entity-first V3 workflow completed the fixed 100 Event sample with 20 accepted submissions, 80
terminal rejected submissions and zero execution failures. A rejected submission is an auditable
abstention: after candidate-level isolation, the Event retained no acceptable EventEntityLink. It is not
a model-contract or transport failure.

The sample extracted 237 grounded raw mentions, accepted 23 EventEntityLinks and rejected one unsafe
binding. Nineteen accepted sample Events were valid link-only Events; one accepted Event also produced
one VariableSignal and one natural-language Measurement. The separately executed fixed NVIDIA/Amkor
Event accepted one EntityLink, one VariableSignal and one natural-language Measurement. No run produced
DirectImpact.

Compared with V2's 16 accepted, 57 rejected and 27 model-contract failures, V3 raised completed workflow
coverage from 73 to 100 Events while preserving conservative `no_match` behavior.

## Frozen architecture and projection

- PostgreSQL remains the only fact authority. Qdrant is a rebuildable retrieval projection and owns no
  accepted state.
- Data owns PostgreSQL-to-Qdrant projection and uses an ordinary OpenAI-compatible HTTP adapter. Data's
  module has no Eino/eino-ext dependency.
- AgentRun owns Qdrant lookup. Its Event-batch adapter receives an injected official Eino
  `embedding.Embedder`, embeds unresolved mentions once, and performs one Qdrant query batch.
- Qdrant version is v1.15.5. Embedding is DashScope `text-embedding-v4`, 1024 dimensions, cosine.
- `entity_semantic_v1`: green, 4,788 active/event-link-allowed Entity points.
- `variable_definition_semantic_v1`: green, 12 active/current VariableDefinition points.
- Points record `projection_version=event-semantic-projection.v1` and
  `embedding_model=text-embedding-v4`; full rebuild uses idempotent upsert.

Only formal Entity identity/retrieval text and VariableDefinition contract data are vectorized. Events,
Evidence, accepted facts, DirectImpact and the relationship graph are not vectorized.

## Data TBox migration

Migration `000037_event_semantic_entity_first_v3.sql` extends the existing
`entity_type_definitions` table with `name_zh`, `name_en`, `business_definition`,
`inclusion_criteria`, `exclusion_criteria` and `event_link_allowed`. Data domain models, PostgreSQL
repositories, API/OpenAPI contracts, Context manifests, Research Analysis consumers and the projector
all read and preserve the fields.

The migration contains a one-time, manually authored backfill for all 12 active definitions and validates
content before adding non-null/content constraints. No Curator, generation Agent, management UI or
continuous TBox synchronization was added.

## Final 100 Event result

The run reused the fixed sample whose ID CSV SHA-256 is
`820a9a280a86808a1df481273140f40fbd0963a641d7f29a4e8db06ae85044b6`.
The authoritative full-batch report is
`/private/tmp/event-semantic-v3-acceptance-final-v3.json`, SHA-256
`7fdfb44774b3bf9db464e6f049bfcc28795e6c69262ae9b3675eb697db28044d`.
It contains 100 comparison records plus the separately marked fixed Event. One record initially returned
`EVENT_SEMANTICS_CONFLICT` because a prior diagnostic embedding failure still owned an unexpired Context
Lease. After that lease expired, the same Event was rerun from
`/private/tmp/event-semantic-v3-acceptance-retry.json`, SHA-256
`ba5b8099f751161c18c02d1d61892fe04fe2e9692ab041e9a9e3cf9c5c6abf2f`, and terminated `rejected`
without an execution error. All effective 100-Event metrics below replace the conflict record with this
retry; the transient and retry remain separately auditable.

| Metric | Result |
| --- | ---: |
| Accepted / rejected / failed Events | 20 / 80 / 0 |
| Raw mentions | 237 |
| Exact-hit mentions | 19 |
| Vector fallbacks | 221 |
| `no_match` | 213 |
| EventEntityLink accepted / rejected | 23 / 1 |
| VariableSignal accepted / rejected | 1 / 0 |
| Measurement emitted / accepted / rejected | 1 / 1 / 0 |
| Accepted link-only Events | 19 |
| Projected type / PG type mismatches | 0 |
| DirectImpact rows | 0 |
| Direct Target / transmission-rule calls | 0 / 0 |
| Data entity-search calls | 0 |

The accepted Measurement remained natural-language `measurement_text` plus Evidence lineage; no
numeric/value-shape interpretation gate was involved.

## Retrieval coverage and candidate isolation

Across 93 Event-level exact batches, 237 Mention lookups produced 218 empty, 16 unique and three
ambiguous candidate sets. Ambiguous exact mentions were still sent through the single Event-level vector
batch; exact and vector results were merged before selection. This prevents a non-unique alias from
silently becoming a binding.

Cross-type vector retrieval returned 2,210 audited candidates across ten formal types:

| Entity Type | Candidates |
| --- | ---: |
| `alliance_org` | 252 |
| `chain_node` | 1,031 |
| `commodity` | 12 |
| `company` | 119 |
| `concept` | 53 |
| `industry` | 90 |
| `industry_chain` | 31 |
| `person` | 297 |
| `policy_body` | 210 |
| `security` | 115 |

There were 188 multi-type candidate sets. Retrieval no longer excludes a correct candidate using an LLM
predicted type. The selected candidate's projected type is submitted to Data and deterministically
compared with the PG Entity type; this batch recorded zero mismatches.

Two candidates were isolated without terminating their Events: one
`mention_evidence_support_invalid / model` and one `duplicate_entity_link / agentrun`. Among selector
`no_match` decisions, 44 were classified as `stage_a_non_entity_mention / model_extraction`, 166 as
`identity_projection_gap / abox_or_retrieval`, and two as
`selector_rejected_exact_candidates / model_selection`. The candidate lists plus later PG audit allow the
combined ABox/retrieval category to be split without giving AgentRun PG access. Final JSON-envelope,
Context, transport and unknown-outcome failures were all zero after the one lease-expiry retry.

## Manual EntityLink audit

All 24 persisted sample EventEntityLink decisions were inspected against mention text, selected PG Entity,
formal type and Event context.

- The 23 accepted links were direct identities or defensible normal forms in this sample. Examples include
  `央行` -> `中国人民银行`, `欧洲央行` -> `欧洲中央银行`, `福特` -> `福特汽车`, and direct
  people/company/policy-body matches. No accepted wrong-object or wrong-type binding was observed.
- The compound mention `中国人民银行副行长邹澜` was proposed as the institution
  `中国人民银行`; independent review rejected it as an object mismatch.

This is a bounded audit of the fixed sample, not a global precision claim. Cross-type recall increased the
candidate surface without increasing observed accepted wrong bindings in this run.

## Mention completeness proxy

The earlier system-external DeepSeek analysis is not gold truth: it includes amounts, periods, metrics and
other objects outside Stage A's frozen formal-Entity scope. After per-Event normalization and de-duplication,
it contains 261 mentions. V3 produced 236 normalized unique mentions (237 raw Stage A items):

- exact text overlap: 155 / 261 (59.4% reference-side proxy);
- lexical containment overlap: 203 / 261 (77.8% reference-side proxy);
- exact overlap over V3 unique mentions: 155 / 236 (65.7% precision-like proxy).

These are taxonomy-sensitive lexical proxies, not semantic precision/recall measurements. Remaining gaps
must be classified by ABox/TBox coverage and manual relevance before being treated as model omissions.

## VariableSignal and Measurement quality

The 100 comparison Events accepted one Signal for the formal `汽车` industry:
`production_volume / decrease / actual`, with narrative Measurement
`2026年1-6月汽车生产1510万台，同比降4%`. The Signal was grounded in the Event's Evidence even though
the title foregrounded industry profit rate. No Signal or Measurement was rejected in the effective run.

The fixed NVIDIA/Amkor Event accepted `order_value / increase / actual` and natural-language Measurement
`15亿美元`. The result demonstrates that the post-resolution Signal stage can use the chosen Entity's
formal type and pinned complete VariableDefinition directory without numeric parsing. Direction-only
Signals and link-only Events remain legal.

## Size and latency

| Metric | Result |
| --- | ---: |
| Prompt bytes total / average / min / max | 1,607,593 / 16,075 / 2,021 / 40,773 |
| Prompt bytes p50 / p95 | 14,734 / 30,188 |
| Context bytes total / average / min / max | 1,511,027 / 15,110 / 14,867 / 15,779 |
| Context bytes p50 / p95 | 15,067 / 15,421 |
| Model calls / total model latency | 220 / 308,594 ms |
| Event latency p50 / p95 | 3,112 / 6,202 ms |
| Qdrant exact Event-batch calls | 93 |
| Qdrant vector Event-batch calls | 88 |
| Qdrant candidates returned | 2,232 |
| Qdrant latency samples / p50 / p95 | 181 / 5 / 337 ms |
| Data API calls / request bytes | 421 / 135,603 |
| Data latency p50 / p95 | 8 / 15 ms |

Events with no extracted Mention make no Qdrant call. Every Event with mentions performs one exact batch;
every Event with a non-unique or unresolved exact result performs one Eino `EmbedStrings` batch and one
Qdrant query batch, regardless of mention count. There is no mention-level N+1.

## Fixed NVIDIA / Amkor Event

Event:

> 2026年7月，英伟达与全球第二大半导体封测厂商安靠科技达成了一项价值15亿美元的战略合作，并首次将“预付款锁定产能”的模式延伸至第三方封测厂。

The dedicated run was accepted in 6,679 ms with four model calls:

- Stage A extracted only `英伟达` and `安靠科技`, both with Evidence lineage; the amount was not emitted
  as an Entity Mention.
- Exact lookup for `英伟达` returned both the company and `英伟达生态` concept. The ambiguous result also
  entered vector retrieval; the selector chose the formal NVIDIA company.
- `安靠科技` has no formal ABox Entity. Its ten cross-type vector candidates were unrelated, so the
  selector returned `no_match` instead of substituting Onsemi or another similar-name company.
- NVIDIA received `order_value / increase / actual` with narrative Measurement `15亿美元`.
- Accepted EventEntityLink / VariableSignal / Measurement counts were `1 / 1 / 1`.
- No packaging-chain Entity, DirectImpact, cross-Entity propagation, Theme or investment conclusion was
  emitted.

## Comparison with V2

| Metric | V2 | V3 |
| --- | ---: | ---: |
| Accepted / rejected / failed Events | 16 / 57 / 27 | 20 / 80 / 0 |
| Mentions | 212 | 237 |
| Exact / fallback / `no_match` | 15 / 197 / 153 | 19 / 221 / 213 |
| EventEntityLink accepted / rejected | 18 / 0 | 23 / 1 |
| Signal accepted / rejected | 1 / 0 | 1 / 0 |
| Prompt bytes total / average | 1,839,781 / 18,398 | 1,607,593 / 16,075 |
| Event latency p50 / p95 | 3,844 / 8,133 ms | 3,112 / 6,202 ms |
| Qdrant candidates | 830 | 2,232 |
| Data calls / request bytes | 389 / 107,595 | 421 / 135,603 |

V3 Context is larger because it dynamically includes authored Entity Type boundaries and cross-type
candidates. Total prompt size and Event latency nevertheless fell. Candidate count rose as expected after
removing predicted-type hard filtering; the audited sample did not show an accepted wrong-type increase.

## Remaining differences by owner

### ABox gaps

- Amkor and other mentioned organizations/products remain absent and correctly end as `no_match`.
- V3 never creates a missing Entity or VariableDefinition.

### TBox gaps

- Only 12 active VariableDefinitions exist, explaining the low Signal yield.
- Some formal entities have Entity Types that are not active/event-link-allowed for this workflow and are
  intentionally excluded from the projection.
- Instrument-versus-commodity and product-versus-company boundaries now prefer safe rejection until the
  correct formal object exists.

### Model errors

- One Mention/Evidence grounding violation and one duplicate Entity selection were isolated.
- One unsafe compound-person-to-institution selection was rejected by review.
- These errors no longer terminate unrelated candidates in the same Event.

### Workflow result

- Stage A emits only raw grounded Mention plus Evidence IDs.
- Entity Type no longer comes from model prediction or filters retrieval before candidate selection.
- Selector chooses only a supplied formal Entity ID or `no_match` and assigns role; Data revalidates the
  selected ID, status, projected type and evidence lineage against PG.
- Signal generation is a separate post-resolution stage and cannot revoke an accepted EventEntityLink.
- The 27 V2 whole-Event model-contract failures became zero V3 execution failures.

## Verification

- Data migrations 000001 through 000037 on a fresh PostgreSQL database: pass.
- AgentRun migrations 001 through 012 on a fresh PostgreSQL database: pass.
- Data TBox migration contract and active-definition completeness: pass.
- Data Event Semantic and AgentRun stage-audit PostgreSQL integration seams: pass.
- Data-owned real PostgreSQL-to-Qdrant full rebuild: pass, 4,788 Entity and 12 VariableDefinition points.
- AgentRun cross-type exact/vector batching and injected Eino Embedder tests: pass.
- Data and AgentRun `go test ./...` and `go vet ./...`: pass.
- Repository architecture/contract tests: pass.
- DirectImpact rows and prohibited dependency/call audit: zero.
- No UAT or production deployment was performed.
