# Event Semantic Anchor Resolution MVP V1

Status: frozen implementation spec

> Superseded 2026-08-01 by `event-semantic-qdrant-retrieval-v2.md` and ADR-0009.
> New Event Semantic executions do not use route/anchor/candidate APIs or resolution bindings;
> this document remains only as the V1 historical record.

Date: 2026-07-31

Owners: Data Service (formal facts and resolution interface), AgentRun (Event Semantic workflow)

Supersedes for this MVP: `event-semantic-agent-catalog-mvp-v1.md`

## 1. Outcome

Restore the Event Semantic Agent without sending the complete Entity and EntityRelation catalog to
AgentRun or to the LLM.

The MVP must run this bounded path end to end:

```text
formal Event + Evidence
→ LLM extracts raw mention, likely Entity Type and Event-native Signal
→ AgentRun selects a formal routing branch
→ Data returns bounded formal anchor candidates
→ LLM selects formal anchor IDs
→ Data traverses approved formal relations and returns bounded target candidates
→ LLM selects one formal Entity ID or leaves the mention unresolved
→ existing Direct Target, Submission and Review flow completes
```

The first vertical slice resolves `chain_node`. Other target Entity Types are not required for this
MVP acceptance.

## 2. Problem

The current leased Semantic Context contains approximately 5,000 active Entities and 1,500
EntityRelations. Its response is approximately 1.297 MB and is copied into the first LLM prompt.
AgentRun rejects it at the 1 MB response limit; increasing the limit only moves the failure to a
180-second model timeout.

The complete catalog is also redundant because AgentRun already calls separate Data APIs for Entity
Resolution and Direct Target lookup.

Root cause: Data's complete internal lease snapshot is being used as an external inference context.

## 3. Scope

### Included

1. Replace the complete Context Snapshot with a compact Context Manifest and on-demand resolution
   bindings.
2. Return a compact Semantic Context containing only Event-level evidence and required TBox.
3. Add a versioned, bounded anchor-resolution interface owned by Data.
4. Change AgentRun to perform mention extraction, route selection, anchor selection and target
   disambiguation as separate stages.
5. Preserve the existing Direct Target, Submission and Review responsibilities.
6. Run one local mock Event through the complete Event Semantic flow.

### Explicitly excluded

- Wiki or generated semantic files;
- embeddings or a vector database;
- a new search engine or middleware;
- AgentRun direct PostgreSQL access;
- AgentRun direct Neo4j access;
- exporting all Entities, relations or the complete industry graph;
- automatic creation of Entity, alias or relation facts;
- recursive graph search controlled by the LLM;
- changing Theme or Reason Tree logic;
- raising the 1 MB AgentRun response safety limit as the fix.

## 4. Ownership and authority

| Concern | Owner | Rule |
| --- | --- | --- |
| Formal Entity, alias and relation facts | Data | PostgreSQL remains the source of truth |
| Context Lease and consistency | Data | Lease stores a compact manifest, not a copied ABox catalog |
| Resolution route contract | Data | Versioned TBox contract; caller cannot invent graph paths |
| Work Item, retries and LLM orchestration | AgentRun | Data does not own Agent execution |
| Candidate selection | AgentRun/LLM | Selection is limited to candidates returned by Data |
| Final validation and accepted semantic facts | Data | Unknown or invalid IDs are rejected deterministically |

Neo4j remains a Data projection and is not a new runtime dependency for this slice. Data may use its
own approved persistence adapters behind the interface, but the first implementation should use the
existing formal PostgreSQL facts unless an accepted repository design says otherwise.

## 5. Compact Semantic Context contract

The existing leased context endpoint remains the entry point, but its external response and persisted
lease manifest must not contain complete `entities` or `relations` arrays.

It must contain:

- Event identity, title, summary, occurred/statement time and status required by the current contract;
- Evidence ID, verbatim excerpt, RawDocument lineage ID and source metadata already owned by Data;
- allowed Event Entity roles and Entity Type definitions;
- active Variable Definitions applicable to Event Semantic;
- Signal modality and measurement policy;
- approved Direct Transmission Rules required by the existing Direct Impact stage;
- Context Lease ID, context fingerprint and TBox/route contract versions.

It must not contain:

- the full Entity catalog;
- the full EntityRelation catalog;
- the full industry-chain graph;
- RawDocument complete body;
- AgentRun Artifact or Collector source files.

Target acceptance: the mock fixture's compact response is below 100 KB and remains below AgentRun's
existing 1 MB safety limit. Tests must also assert that complete Entity and relation arrays cannot
reappear in the wire response.

## 6. Resolution route TBox

The MVP publishes one route contract, for example `event-semantic-anchor-routing@1`:

```text
target: chain_node

primary route:
Industry
← MAPPED_TO_INDUSTRY — IndustryChain
— HAS_NODE → ChainNode

supplementary route:
Concept
← MAPPED_TO_CONCEPT — IndustryChain
— HAS_NODE → ChainNode
```

The physical PostgreSQL relation names may differ from the display names above. Data owns that
mapping. AgentRun sees the route ID, allowed anchor types, direction and plain-language purpose, not
SQL tables or Cypher.

Industry is the primary anchor because current formal data covers approximately 2,924 of 3,051
active ChainNodes through Industry → IndustryChain → membership. Concept is supplementary and covers
approximately 1,351 nodes. These counts are observations, not frozen acceptance constants.

The route registry can be a versioned Data domain contract in this MVP. A new PostgreSQL table and
migration are not required unless implementation discovery proves that repository authority already
requires persisted routes.

## 7. Data resolution interface

The exact generated types and route naming follow the existing Data v1 conventions. The semantic
interface must expose the following three capabilities.

### 7.1 Get resolution routes

Input:

```json
{
  "context_lease_id": "uuid",
  "target_entity_type": "chain_node"
}
```

Output contains:

- `route_contract_version`;
- route IDs and allowed formal anchor Entity Types;
- bounded routing partitions: active level-1 Industries and controlled Concept types;
- permitted next operation and stable ordering contract.

The response does not contain every Industry, Concept, IndustryChain or ChainNode.
Each route exposes at most 50 partitions, ordered by canonical name and formal identity. Partitions
beyond that deterministic Data-owned budget are outside the current resolution window and therefore
produce an unresolved semantic outcome; AgentRun must not bypass the budget with another catalog
query or a model-controlled loop.

### 7.2 List formal anchors

Input supports one bounded branch at a time:

```json
{
  "context_lease_id": "uuid",
  "route_id": "chain-node-by-industry",
  "partition_key": "selected level-1 industry ID or concept type",
  "parent_anchor_ids": ["optional formal parent UUID"],
  "page_size": 50,
  "cursor": "opaque"
}
```

Output contains formal anchor ID, Entity Type, canonical name, concise description, hierarchy
identity and stable cursor. For the Industry route, an omitted `parent_anchor_ids` returns only
approved descendant Industry anchors inside the selected level-1 partition that have a direct,
active `mapped_to_industry` path to an approved IndustryChain with an approved ChainNode membership.
This bounded leaf-anchor view intentionally makes level-3 mappings reachable without returning the
complete hierarchy or introducing a model-controlled hierarchy loop. Supplying
`parent_anchor_ids` narrows the same reachable-anchor view to those approved subtrees. The Concept
route returns only approved Concepts in the selected controlled type that have the corresponding
formal path. AgentRun must choose an ID from this response; it must not query by a model-invented
canonical name.

Both anchor and candidate pages use database-level keyset pagination ordered by canonical name and
formal Entity ID. The repository query applies `LIMIT page_size + 1`; an opaque cursor carries the
last ordering key plus the lease/request snapshot identity. Data must not load or fingerprint the
complete matching ABox before slicing a page. A cursor whose lease/request snapshot identity no
longer matches returns `409 EVENT_SEMANTIC_CONTEXT_DRIFT`.

### 7.3 Resolve candidates by anchors

Input:

```json
{
  "context_lease_id": "uuid",
  "target_entity_type": "chain_node",
  "anchor_entity_ids": ["one or more formal Industry/Concept UUIDs"],
  "match_mode": "any",
  "page_size": 50,
  "cursor": "opaque"
}
```

Output contains only formal candidates reachable through an approved route:

- target Entity ID, type, canonical name and concise description;
- matched anchor IDs;
- traversed IndustryChain IDs and names;
- route ID and bounded path evidence;
- stable ordering and opaque cursor.

Data must reject unknown lease IDs, expired leases, wrong Entity Types, inactive/superseded IDs,
unapproved route combinations and oversized page requests. An empty result is a valid response.

Name/alias search may remain as an acceleration path, but it cannot be the only resolution path and
cannot bypass the formal anchor route.

## 8. Context Lease and compact manifest

The Context Lease remains necessary, but the current complete `context_snapshot` is not. The lease
coordinates one Work Item attempt, fixes its authoritative input identities and prevents mixed-version
submission. It is not a copy of every fact that the attempt might possibly query.

### 8.1 Persisted compact manifest

Persist only the bounded material required to identify and validate the attempt:

- Event ID and an Event input fingerprint;
- Evidence IDs and Evidence content/lineage fingerprints;
- ontology version;
- acceptance policy key/version;
- applicable Variable Definition keys/versions;
- approved Direct Transmission Rule IDs/versions needed by the current stage;
- Entity Resolution route contract version;
- Agent Execution ID, worker identity, lease status and expiry;
- manifest contract version and canonical manifest fingerprint.

The canonical manifest fingerprint covers every persisted manifest field except the fingerprint
field itself, including lease status and expiry. Renewing a lease rewrites the expiry and recomputes
the fingerprint atomically with the lease row update.

Do not persist in the manifest:

- every active Entity;
- every EntityRelation;
- complete Industry, Concept, IndustryChain or ChainNode catalogs;
- the complete industry graph;
- Evidence excerpts or other Evidence content;
- complete Entity Type, Variable Definition or Direct Transmission Rule objects;
- data that the attempt never queried.

The compact Context API response is generated from this manifest and its pinned Event/Evidence/TBox
identities. The target fixture's persisted manifest must be below 100 KB; normal operation should be
substantially smaller.

### 8.2 On-demand resolution bindings

When an anchor or target candidate is actually selected, Data records or returns a bounded resolution
binding tied to the lease. A binding contains only:

- lease ID and route contract version;
- selected formal anchor Entity IDs;
- selected target Entity ID;
- traversed IndustryChain and relation/membership identities;
- path fingerprint and the formal object versions when available.

Submission precheck validates the exact binding used by the Agent. It must not rebuild validation from
a complete lease snapshot. Implementations may persist accepted/selected bindings in a dedicated
bounded structure or encode them in a tamper-safe Data-owned receipt, according to existing repository
contract conventions. Raw model output is not a valid binding.

### 8.3 Consistency and retry

- Route and candidate calls must carry the same `context_lease_id` as the compact context.
- All calls must return the same route/TBox contract version for that lease.
- Data queries formal facts on demand and validates the selected path fingerprint when the candidate
  is bound or submitted.
- If a selected Entity, relation or membership changes during the short lease, Data returns a stable
  retryable conflict. AgentRun starts a new attempt/lease; Data does not silently accept drift.
- AgentRun retries the attempt according to its existing Work Item policy and must not mark a
  temporary transport/model failure as Event data non-compliance.

### 8.4 Schema migration

Migration `000032` has already entered local/UAT history and must not be edited. Implement this change
with a new forward migration. The preferred domain name is `context_manifest`, not
`context_snapshot`. The implementation must define safe backfill/mixed-version behavior and must not
rewrite historical large snapshots merely to preserve an obsolete representation. Removal of the old
column may be staged if mixed-version rollout requires it.

## 9. AgentRun workflow

The Event Semantic Workflow becomes:

```text
1. claim Event Semantic Work Item
2. create Context Lease
3. fetch compact Semantic Context
4. LLM extracts:
   - raw entity mention and evidence span
   - likely Entity Type
   - Event Entity role
   - Event-native Variable Signal candidate
5. for each mention needing formal resolution:
   a. get allowed routes for predicted target type
   b. LLM selects one routing partition
   c. list bounded formal anchors
   d. LLM selects one or more formal anchor IDs
   e. resolve formal target candidates by anchors
   f. LLM selects one formal target ID or unresolved
6. call existing Direct Target lookup only when a resolved Signal requires it
7. generate Direct Impact candidates under existing rules
8. submit candidates
9. run existing AI review path when required
10. complete the Work Item
```

### LLM constraints

- The model may infer type, partition and candidate choice from Event Evidence.
- It may only emit formal IDs present in the immediately preceding Data response.
- It cannot invent aliases, relations, Entity IDs or graph paths.
- It must preserve the raw mention and exact supporting Evidence reference.
- If no candidate is supported, it returns `unresolved`; it must not bind from model memory.
- A wrong predicted Entity Type or a missing formal Entity is an acceptable unresolved semantic
  outcome, not permission to load the complete catalog.

### Bounded execution

For each mention, the MVP permits a bounded number of branch/anchor/candidate calls. No open-ended
agentic loop is allowed. The exact numeric limits are AgentRun configuration and must be covered by a
consumer test; they are not model-controlled.

## 10. Failure semantics

| Failure | Classification | Required behavior |
| --- | --- | --- |
| Data network/timeout/5xx | retryable execution failure | keep Event eligible under existing retry policy |
| LLM timeout/provider failure | retryable execution failure | do not mark Event non-compliant |
| Expired lease or changed selected path | retryable context conflict | create a new attempt/lease according to existing policy |
| Unknown route/ID/type mismatch | deterministic contract error | fail the candidate call; do not submit invented facts |
| No anchors or no target candidates | valid unresolved result | continue with other mentions; no forced EntityLink |
| Evidence cannot support mention | semantic rejection | candidate is not accepted |
| Missing formal Entity in Data | out of current semantic scope | unresolved; no automatic Entity creation |

Existing AgentRun Work Item ownership and Data disposition rules remain unchanged.

## 11. Highest observable acceptance seams

Implementation follows red → green vertical slices at these confirmed seams:

1. **Data HTTP provider seam**
   - compact context omits all-Entity/all-relation payloads;
   - route, anchor and candidate contracts validate identity, pagination and errors;
   - a formal Industry/Concept anchor returns only reachable ChainNode candidates.
2. **Data persistence seam**
   - a new lease persists only the compact manifest, never the full Entity/relation catalog;
   - only actually selected resolution bindings are retained for deterministic precheck;
   - changing a selected path produces a retryable conflict rather than silent drift.
3. **AgentRun consumer/workflow seam**
   - the workflow never passes the complete catalog to the model;
   - the model can only select IDs returned by Data;
   - empty candidates produce unresolved output, not invented links.
4. **Local cross-service E2E seam**
   - one mock Event completes Context Lease → semantic extraction → formal ChainNode resolution →
     Submission/Review;
   - the Event phrase is intentionally not an exact canonical-name lookup;
   - the selected ChainNode already exists in local Data and is reachable through the fixture's
     formal anchor route;
   - accepted EventEntityLink and VariableSignal reference the expected formal IDs and Evidence;
   - no request exceeds 1 MB and the compact context is below 100 KB.

The mock must be built from local test/fixture data and must not promote discussion examples such as a
particular company or `光模块` into permanent product acceptance rules.

## 12. Delivery order

1. Freeze this Spec, wire DTO examples and provider/consumer tests.
2. Data: replace the complete Context Snapshot with a compact manifest through a forward migration.
3. Data: implement bounded selected-path bindings for submission precheck.
4. Data: implement route, anchor and candidate modules behind a small versioned interface.
5. AgentRun: update the consumer port/HTTP adapter.
6. AgentRun: split the Eino workflow into bounded extraction and resolution stages.
7. Add the local mock fixture and run cross-service E2E.
8. Run Go formatting, affected tests, `go vet`, OpenAPI/contract checks and repository-required
   Standards/Spec review.

Rollout order is Data provider first, AgentRun consumer second. Until both versions are deployed,
Event Semantic must remain paused or use an explicitly compatible response contract; it must not fall
back to the full context. Rollback restores both service versions and the previous contract together.

## 13. Definition of done

- Data no longer exposes complete Entity/EntityRelation arrays in Event Semantic Context.
- New leases no longer persist a complete Entity/EntityRelation catalog in `context_snapshot`.
- Submission validation uses the compact manifest and exact selected resolution bindings.
- AgentRun no longer serializes the complete catalog into an LLM prompt.
- ChainNode resolution works through formal Industry/Concept anchor IDs and approved relation paths.
- Existing Direct Target, Submission, Review and Data acceptance rules still pass.
- One non-exact-name local mock Event completes with formal Evidence lineage.
- Empty and retryable failure cases behave as specified.
- OpenAPI, Context docs and provider/consumer fixtures are synchronized.
- No Wiki, vector database, search middleware, direct DB access or Neo4j direct connection is added.

## 14. Implementation decisions

These decisions resolve the implementation degrees of freedom in this frozen scope without changing
the outcome or non-goals.

### 14.1 Persistence and mixed-version rollout

- Forward migration `000035` adds nullable `context_manifest` storage and selected resolution
  bindings; migration `000032` remains unchanged.
- Historical `context_snapshot` values are not rewritten. New Data code creates only compact
  manifests and never falls back to returning a legacy full snapshot through the Context API. If
  the same execution idempotently replays a legacy lease, Data creates a compact manifest for that
  one lease while preserving its historical snapshot; this prevents a renewed-but-unreadable lease.
- The legacy snapshot column remains during the rollout window so schema rollback is additive. Event
  Semantic stays paused while Data and AgentRun versions are mixed and while legacy active leases
  expire.
- A selected binding is persisted only in the Submission transaction. Candidate-list responses are
  read-only and do not retain every offered target.

### 14.2 Data-owned resolution receipt

- Each resolved ChainNode candidate contains a bounded Data-generated resolution receipt: route
  version/ID, selected anchor IDs, target ID, traversed IndustryChain and mapping/membership
  identities, plus a canonical path fingerprint.
- The LLM emits only a candidate target ID. AgentRun deterministically copies the matching Data
  receipt into the EntityLink submission; model output is never accepted as a receipt.
- Data recomputes the selected path under the active lease at Submission time. A changed or missing
  path returns the retryable context-drift conflict and no semantic facts are committed.

### 14.3 Bounded orchestration

- The existing typed `compose.Workflow` remains the smallest suitable Eino primitive: the new stages
  are a finite DAG with typed state and no model-controlled loop.
- Industry route partitions are formal approved level-1 Industry UUIDs paired with Data-owned display
  labels; Concept partitions are controlled Concept types. Routes also expose direction and a
  plain-language purpose so the model never chooses among opaque UUIDs alone. Anchor pages expose
  description and hierarchy identity, accept bounded
  `parent_anchor_ids`, and candidates require `target_entity_type=chain_node` plus `match_mode=any`.
  Route responses declare the next operation and stable ordering contract; candidate responses name
  all matched anchors and the traversed IndustryChain while retaining one deterministic receipt path.
- The ChainNode MVP permits at most one route lookup, one bounded reachable-leaf anchor page and one
  bounded candidate page per mention. The Industry leaf page is restricted to the selected level-1
  partition and may therefore contain formally mapped level-2 or level-3 Industries without an
  intermediate model-driven descent. AgentRun does not follow pagination cursors or hierarchy links;
  empty pages resolve the mention as `unresolved`.
- Existing exact-name resolution may remain for non-ChainNode compatibility, but the ChainNode
  acceptance path must use formal anchor routing and cannot fall back to the complete catalog.

### 14.4 Migration and rollout risk checklist

- `000035` is immutable once observed in local/UAT history; all later schema corrections use a new
  forward migration. This remediation changes application behavior and tests, not that migration.
- A pre-`000035` row with only `context_snapshot` remains readable as historical JSON after migration.
  Idempotent replay upgrades only that lease to a reference-only manifest and preserves the legacy
  snapshot; the executed migration test covers old-row preservation, replay and a new manifest write.
- Data provider and AgentRun consumer are deployed together while Event Semantic processing is
  paused. Processing resumes only after both report the same Context/route contract versions; this
  avoids an old consumer interpreting the compact response as the retired full catalog.
- Rollback is application-first and schema-additive: the nullable legacy column remains, and neither
  historical snapshots nor accepted bindings are deleted. No Down migration is run in UAT/prod.
- Formal path changes during a lease are not hidden by pagination. Submission recomputes the selected
  receipt and returns retryable `EVENT_SEMANTIC_CONTEXT_DRIFT` before writing semantic facts.

## 15. Eino reference-first audit

| Repository | Exact commit | Relevant evidence inspected |
| --- | --- | --- |
| `cloudwego/eino` | `922b6a8a233b5233fe47eecee6cd2c005e8c39cd` | `compose/workflow.go`, typed `Workflow`, `InvokableLambda`, all-predecessor DAG semantics and compile boundary |
| `cloudwego/eino-ext` | `9137edd89e72b72735ede69db1c5ae29178a6e41` | `components/model/openai` structured JSON response and immutable model adapter boundary used by the pinned DeepSeek adapter |
| `cloudwego/eino-examples` | `171220631fb7068ead50b7cd964b8c471647117d` | typed compose workflow/model examples; used only as orchestration and test-pattern evidence |

Adopted: typed workflow input/output, explicit deterministic Lambda stages, caller `context.Context`,
compile-once execution and fake boundary models in tests.

Rejected: Tool-calling, ADK/multi-Agent, open loops, dynamic graph construction, Provider code in Biz,
example deployment/layout and model-owned persistence or retry. Project-owned Work Items, leases,
idempotency, Data validation and retry remain outside Eino.

Version compatibility is fixed by root `go.mod`: `eino v0.9.12` and
`eino-ext/components/model/openai v0.1.13`. The project-specific gap is formal-ID constrained anchor
resolution; it is implemented as Data consumer Port calls and deterministic receipt validation, not
as an Eino Tool or retriever.
