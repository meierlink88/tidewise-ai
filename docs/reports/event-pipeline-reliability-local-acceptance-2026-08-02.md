# Event pipeline reliability local acceptance report

Date: 2026-08-02

Issue: #168

Environment: local only; no UAT or production deployment

## Outcome

The fixed 100 Event sample completed with **54 accepted, 46 rejected and 0 execution failures**.
It produced 72 accepted EventEntityLinks, two accepted VariableSignals and one accepted Measurement.
All 100 runs passed the deterministic boundary audit: DirectImpact, Direct Target, transmission-rule,
Theme and Reason Tree output or calls remained zero.

The previous V3 run on the same sample was 45 accepted, 55 rejected and 0 failed. Removing deterministic
source-text matching recovered nine Events without introducing an execution failure. Accepted formal
entity identities were inspected from the stage audit; no accepted wrong-object binding was observed.
Role choice remains model-variable outside the existing golden cases and is not claimed as globally
solved by this pipeline reliability change.

## Delivered behavior

- Event Publication V3 treats all Evidence links equally. It removes `is_primary` and
  `primary_source_id`, renames `evidence_excerpt` to `evidence_statement`, and preserves existing rows
  through a forward migration.
- Event Fact V2 uses four forced Eino Function Calls for extraction, duplicate judgment, tag assignment
  and review. Each stage accepts exactly one expected call, performs one bounded correction, and retains
  deterministic publication ownership outside the model.
- Event Fact, Event Semantic and Data precheck no longer use source-text substring, verbatim Mention or
  date-text matching to reject model-authored semantics. Structural, lineage, catalog, identity, status,
  authorization and transaction checks remain.
- The Event Semantic worker drains another ready item immediately after durable completion. Its 60-second
  ticker is now a recovery watchdog, while the existing single-processing permit prevents concurrent
  drains.

## Fixed 100 Event metrics

The fixed sample ID CSV SHA-256 is
`820a9a280a86808a1df481273140f40fbd0963a641d7f29a4e8db06ae85044b6`.

| Metric | Result |
| --- | ---: |
| Accepted / rejected / failed Events | 54 / 46 / 0 |
| Raw mentions | 258 |
| Exact hits / vector fallbacks / no_match | 80 / 181 / 171 |
| EventEntityLink accepted / rejected | 72 / 15 |
| VariableSignal accepted / rejected | 2 / 0 |
| Measurement accepted / rejected | 1 / 0 |
| Candidate-level isolations / stage violations | 39 / 0 |
| Model-contract failures | 0 |
| DirectImpact / Direct Target / transmission-rule calls | 0 / 0 / 0 |
| Theme / Reason Tree outputs | 0 / 0 |

Acceptance artifact:
`/private/tmp/event-pipeline-168-semantic-acceptance.json`, SHA-256
`656490cfb863f0be161f9520b552589e13b447436b628cbb256d196e31b0044d`.

## Latency and dependencies

| Metric | Result |
| --- | ---: |
| Event latency p50 / p95 / max | 4,931 / 7,665 / 10,845 ms |
| Model calls / total latency | 279 / 472,312 ms |
| Prompt bytes average / p50 / p95 | 20,257 / 18,658 / 31,784 |
| Context bytes average / p50 / p95 | 17,921 / 17,878 / 18,232 |
| Qdrant exact / vector Event-batch calls | 100 / 81 |
| Qdrant candidates / latency p50 / p95 | 1,893 / 3 / 291 ms |
| Data API calls / request bytes | 464 / 171,382 |
| Data API latency p50 / p95 | 14 / 50 ms |

No mention-level embedding/query N+1 was observed. Model latency remains the dominant per-Event cost;
Qdrant and Data latency are not systemic bottlenecks in this local run. The worker self-drain behavior is
covered separately by a long-watchdog test that completes two queued items without waiting for the next
ticker.

## Live Artifact-to-Semantic batch

A separate local collector execution (`0fd1a72f-3b7a-450e-815c-655d1922ada7`) exercised the actual
Artifact -> Event Fact V2 -> Event Publication V3 -> Event Semantic V3 -> Data path. The collector
completed in 12,775 ms and published 62 verified Artifacts from 66 merged results.

| Stage | Result |
| --- | --- |
| Event Fact V2 | 43 published / 15 no-event / 4 rejected; 0 left pending or running |
| Event Fact candidates | 70 candidates in completed unit results |
| Function calls | 105 extraction-side / 66 review-side calls |
| Event Fact latency | p50 6,508 ms / p95 21,397 ms / max 33,221 ms |
| First 20 Event Fact units | 226,341 ms wall time; p50 8,404 ms / p95 26,113 ms |
| Event Publication V3 | 43 acknowledged packages; 63 Events created / 5 reused |
| Distinct batch Events | 63 |
| Event Semantic eligibility | 54 eligible and completed / 9 structurally ineligible because `event_time` was unknown |
| Event Semantic outcome | 17 accepted / 37 rejected / 0 execution failures |
| Event Semantic facts | 29 accepted and 8 rejected EntityLinks; 4 accepted and 1 rejected Signals; 0 Measurements |
| Event Semantic latency | p50 2,880 ms / p95 6,922 ms / max 14,339 ms |
| First 20 Semantic units | p50 2,880 ms / p95 6,525 ms / max 7,936 ms |
| Immediate drain | 46 consecutive starts within 100 ms of the previous durable completion; median gap 3 ms |
| Forbidden output audit | 0 DirectImpact / 0 Theme links / 0 Reason Tree links |

The four Event Fact rejections were terminal rather than retry loops: one review `invalid_envelope`, one
extraction `arguments_contract_invalid`, and two candidates rejected by deterministic publication
validation. Input-relative Function argument checks now participate in the same single bounded correction,
so incomplete Artifact, duplicate-pair, Tag-assignment or Review coverage cannot escape as an unclassified
workflow error. Provider/model identity remains on the immutable execution snapshot; each Function stage
persists call count, final finish reason, argument byte count and safe violation classification without
persisting raw arguments.

The nine Events without Semantic submissions all had `event_time = NULL`; their Evidence links and
`evidence_statement` were present. The current Data eligibility contract deliberately requires a known
event time. This is not a worker drain failure, but it is a remaining cross-stage product decision: either
Event Fact must produce a semantically justified time or a later change must explicitly allow timeless
Events into Event Semantic. This issue does not invent a fallback time or copy publication time into the
event-time field.

## Verification

- AgentRun, Data and Admin Backend `go test ./...`: pass.
- Affected Go packages and repository contracts `go vet`: pass.
- Repository architecture/contract suite: pass.
- Admin Portal TypeScript typecheck: pass.
- Event Fact real-provider smoke exercised all four forced Function Call stages: pass.
- Data migration chain through `000038` and AgentRun migration `013`: pass against local PostgreSQL.
- `git diff --check`: pass.
