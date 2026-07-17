# Package 5 Order 2 Apply And Rollback-Only Integration Evidence

## Scope And Outcome

Package 5 Order 2 completed from checkpoint `1469bb4317b97394613dd4b28bf5733615dd804a` against the same redacted, approved local PostgreSQL identity used by Order 1. The only persistent database changes are the single Goose ledger application of `000022` and the frozen raw-receipt schema objects. All synthetic integration DML rolled back.

**Result: PASS for tasks 5.2–5.4. STOP before Package 6.** Task 5.5 and all later packages remain pending.

No database identity, role, grant or credential change, seed, import, Neo4j, UAT/prod/shared, deployment, restore, migration retry, forward-fix, or committed race fixture was used. Connection material remained process-environment injected and was neither printed nor persisted.

## Single Migration Apply

The approved command body was run from `backend/` exactly once:

```text
go run ./cmd/dbmigrate -apply -target-version 000022
```

Credential injection was redacted. The command exited `0` and Goose reported:

```text
OK 000022_add_raw_document_import_receipts.sql
successfully migrated database to version: 22
current_version: 22
applied: 000022_add_raw_document_import_receipts.sql
remaining: none
```

`dbmigrate` was not invoked again during either integration-layer failure review or recovery.

## Post-Apply Read-Only Assertions

Every catalog/count assertion ran inside server-enforced `BEGIN READ ONLY` transactions.

| Assertion | Result | Evidence |
|---|---|---|
| Local identity/scope | PASS | approved local database and principal classifications over the existing container-local Unix socket |
| Ledger version | PASS | one Goose version-0 sentinel; exact positive migration set `1..22`; `000022` applied once; no version above 22 |
| Initial receipt rows | PASS | `0` |
| Allowed structural delta | PASS | public tables `42→43`, columns `366→373`, constraints `196→203`, indexes `101→104`, public functions `0→1`, user triggers `0→1` |
| Seven-column contract | PASS | exact order, types, NOT NULL state and default shown below |
| Named schema objects | PASS | seven constraints, three table indexes including the named time index, fixed function and statement trigger exact |
| Historical migration hash | PASS | 21 files; aggregate SHA-256 `2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc` |
| `000022` hash | PASS | SHA-256 `3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26` |
| Existing business rows | PASS | all 41 pre-existing business tables matched the Order 1 baseline before integration and in the final assertion |

Exact columns:

```text
1 id                uuid                     NOT NULL no default
2 caller_identity   text                     NOT NULL no default
3 idempotency_key   text                     NOT NULL no default
4 payload_hash      character(64)            NOT NULL no default
5 raw_document_ids  uuid[]                   NOT NULL no default
6 result_payload    jsonb                    NOT NULL no default
7 imported_at       timestamp with time zone NOT NULL now()
```

Exact named objects:

- `raw_document_import_receipts_pkey`;
- `uq_raw_document_import_receipts_caller_key`;
- `chk_raw_document_import_receipts_caller`;
- `chk_raw_document_import_receipts_key`;
- `chk_raw_document_import_receipts_payload_hash`;
- `chk_raw_document_import_receipts_raw_ids`;
- `chk_raw_document_import_receipts_result`;
- `idx_raw_document_import_receipts_imported_at` plus the PK/unique backing indexes;
- no-argument PL/pgSQL trigger function `prevent_raw_document_import_receipt_mutation()` with SQLSTATE `55000`;
- `trg_raw_document_import_receipts_immutable`, `BEFORE DELETE OR UPDATE OR TRUNCATE FOR EACH STATEMENT`.

## Stateless Harness Failure And Recovery Record

The dedicated harness remained outside the repository at `/private/tmp/tidewise_p5_raw_receipt_integration.go`. It used the backend's existing pgx dependency and had no migration path. Every synthetic write was enclosed by an outer transaction or savepoint with rollback; both advisory-lock connections rolled back.

### Initial attempt

The first run stopped during its valid-row cross-field read with:

```text
ERROR: operator does not exist: text = uuid (SQLSTATE 42883)
```

The open transaction's deferred rollback ran. A subsequent approved read-only cleanup query returned `raw_document_import_receipts rows=0`. No harness repair or rerun occurred until Leader review.

### Leader review 1 and second attempt

Leader independently classified the issue as a stateless harness-only parameter typing error and authorized one line to cast the `$1`/`$3` text and JSON-text comparisons while retaining the UUID array cast and `WHERE id=$1`. The patch plus static target-line check confirmed no other harness case was changed.

The second run stopped with:

```text
ERROR: operator does not exist: uuid = text (SQLSTATE 42883)
```

Again, the open transaction's deferred rollback ran. The fail-closed gate prevented any additional query, edit or rerun until a second Leader review.

### Leader review 2 and final affected-layer attempt

Leader independently confirmed the remaining ambiguity was only the same temporary SELECT predicate and authorized exactly:

```text
WHERE id=$1
→ WHERE id=$1::uuid
```

A full before/after comparison reported `differing_lines=1` and `only_expected_change=true`; the rest of the harness was byte-identical. The final, explicitly authorized affected-layer run exited `0`:

```text
PASS valid/read/cross-field and UPDATE/DELETE/TRUNCATE immutability (rolled back)
PASS constraint rejection cases (savepoints, outer rollback)
PASS raw-document plus invalid-receipt atomic rollback
PASS two-connection blocking/release with sorted overlapping identity locks (both rolled back)
PASS final raw_document_import_receipts rows=0
```

The constraint cases covered empty caller, uppercase/noncanonical hash, empty raw membership and result/membership mismatch. Immutability operations returned the frozen SQLSTATE and were rolled back to savepoints. The lock case used sorted overlapping raw-identity lock sets, proved blocking until the first transaction released, and rolled back both connections. It did not claim or persist a winner-commit/loser-replay race fixture.

## Final Read-Only Zero-Residue Assertions

The final server-enforced read-only transaction returned:

```text
transaction_read_only: on
raw_document_import_receipts rows: 0
synthetic raw rows with source_external_id prefix p5:package-5-order-2-: 0
positive applied migration rows: 22
000022 applied rows: 1
higher applied rows: 0
max applied version: 22
pre-existing business tables matching baseline: 41/41
```

Final exact row-count manifest:

```text
alliance_org_profiles                    45
benchmark_observations                    0
benchmark_profiles                       10
chain_node_physical_constraints           0
chain_node_profiles                     842
chain_node_relations                    212
commodity_profiles                       45
company_profiles                         77
economy_profiles                         94
entity_edges                            241
entity_external_identifiers           1169
entity_nodes                           1387
event_entity_links                        0
event_import_receipts                   203
event_sources                           204
event_tag_defs                           22
event_tag_maps                          277
events                                  203
graph_projection_run_items                2
graph_projection_runs                    18
index_profiles                           43
ingestion_run_sources                    22
ingestion_runs                           13
ingestion_scheduler_configs               1
instrument_profiles                       4
market_profiles                          47
metric_profiles                          43
person_profiles                          30
policy_body_profiles                     30
raw_documents                           590
research_anchor_chain_nodes               0
research_anchor_events                    0
research_anchor_indices                   0
research_anchors                          0
research_theme_chain_nodes                0
research_theme_events                     0
research_theme_indices                    0
research_themes                           0
security_profiles                        77
source_catalogs                         215
theme_profiles                            0
```

## Recovery Evidence And Stop Boundary

The Order 1 archive remains preserved and was not restored:

```text
path: /private/tmp/tidewise-p5-order1-0ea453db377ec8e0-pre-000022.dump
sha256: 47b0da126ee92a2278d7db417ffaed89626f9b0ea89df117db71cb4dc8222065
```

Package 5 tasks 5.2–5.4 are complete. No authorization is inferred for Package 6, Package 10, another migration operation, database security changes, or any environment/deployment action.
