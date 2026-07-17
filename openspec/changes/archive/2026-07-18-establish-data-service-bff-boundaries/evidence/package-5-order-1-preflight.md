# Package 5 Order 1 Read-Only Preflight Evidence

## Scope And Stop Point

Package 5 Order 1 completed against the already-running, approved local PostgreSQL container at Package 4 commit `0ea453db377ec8e0e51c3ccecc8ec4b42ad62fdf`. This evidence records read-only queries plus creation and inspection of exactly one backup archive. Database and role names are represented only by approved-scope classifications; no credential or DSN was printed or persisted.

Order 2 was not started. No migration command, SQL apply, DDL, DML, seed/import, integration write, grant/role/credential change, Neo4j access, restore, deployment, retry, repair, or forward-fix ran.

## Per-Assertion Results

| Assertion | Result | Evidence |
|---|---|---|
| Package 4 commit | PASS | HEAD is `0ea453db377ec8e0e51c3ccecc8ec4b42ad62fdf` |
| Initial worktree | PASS | clean |
| Local target classification | PASS | existing `postgres:16` container `tidewise-local-postgres` was healthy; the query connection used its local Unix socket |
| Server-enforced read-only transaction | PASS | `current_setting('transaction_read_only') = on` inside `BEGIN READ ONLY` |
| Database classification | PASS | `approved_local_database` |
| Principal classification | PASS | `approved_local_identity` for both current and session identity |
| Goose ledger exists | PASS | `to_regclass('public.goose_db_version')` was non-null |
| Applied migration baseline | PASS | one standard Goose version-0 sentinel plus exactly 21 applied migration rows; exact positive set `1..21`; max applied `21`; no negative or `>21` applied row |
| Repository pending set | PASS | repository files are exactly `000001..000022`; ledger migrations are exactly `000001..000021`; pending is exactly `{000022}` |
| Historical migration hash | PASS | `000001..000021` sorted per-file manifest aggregate SHA-256 is `2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc` |
| `000022` artifact hash | PASS | SHA-256 is `3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26` |
| Package 4 OpenAPI hash | PASS | SHA-256 is `e2fbb4328d8a8f16107ed46f01a7f2dbee12872abd6b9109f2c789f1744fd6fd` |
| Frozen `000022` objects absent | PASS | table, seven named constraints, named index, named function and named trigger all absent |
| Target schema privileges | PASS | effective schema ownership, `USAGE` and `CREATE` all true; effective local database ownership true |
| Goose ledger privileges | PASS | table `SELECT/INSERT/UPDATE/DELETE` and backing sequence `USAGE/SELECT/UPDATE` all true |
| Before schema/count baseline | PASS | captured below without selecting business payloads |
| Backup creation | PASS | exactly one custom-format archive created at the explicit path below; `pg_dump` exit 0 |
| Backup hash/readability | PASS | exact SHA-256 and size below; full `pg_restore --list` parse returned 255 lines and exit 0 |
| Post-backup invariants | PASS | all seven structural counts and all 41 business-table row counts equal the before baseline; ledger remains at 21 |

An initial ledger aggregation included the standard applied version-0 Goose sentinel and therefore counted 22 ledger rows. The immediately following read-only detail assertion separated that sentinel and proved the frozen migration condition: one sentinel, exactly 21 positive applied migrations with set `1..21`, and no higher version. This was diagnostic normalization, not database drift or a migration retry.

## Frozen Object Absence

The following names had zero matching pre-apply catalog objects:

- table `raw_document_import_receipts`;
- constraints `raw_document_import_receipts_pkey`, `uq_raw_document_import_receipts_caller_key`, `chk_raw_document_import_receipts_caller`, `chk_raw_document_import_receipts_key`, `chk_raw_document_import_receipts_payload_hash`, `chk_raw_document_import_receipts_raw_ids`, and `chk_raw_document_import_receipts_result`;
- index `idx_raw_document_import_receipts_imported_at`;
- function `prevent_raw_document_import_receipt_mutation()`;
- trigger `trg_raw_document_import_receipts_immutable`.

## Business Schema And Exact Row-Count Baseline

Structural baseline, unchanged after backup:

```text
public tables including Goose ledger: 42
business tables excluding Goose ledger: 41
views/materialized views: 0
columns: 366
constraints: 196
indexes: 101
public functions: 0
non-internal public triggers: 0
```

Exact business-table row counts, all unchanged after backup:

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

## Single Backup Artifact

```text
path: /private/tmp/tidewise-p5-order1-0ea453db377ec8e0-pre-000022.dump
format: PostgreSQL custom archive
bytes: 1373756
sha256: 47b0da126ee92a2278d7db417ffaed89626f9b0ea89df117db71cb4dc8222065
pg_restore --list lines: 255
archive list exit: 0
```

No checksum sidecar or second backup was created. The archive was listed/read only; it was not restored and is not claimed as restore-tested.

Documented recovery procedure, requiring separate authorization: preserve the archive and verify its SHA-256; stop all local writers; independently reconfirm the destination is a new empty local recovery database; list the archive again; restore into that isolated database using environment-injected identity; validate schema and business count invariants; only then submit any cutover or in-place recovery decision for a new review. This procedure was documented but not executed.

## Redacted Command Classes

- local container status inspection;
- `psql` through the existing container-local Unix socket with identity arguments redacted, every assertion wrapped by `BEGIN READ ONLY ... COMMIT`;
- `pg_dump --format=custom --no-owner --no-privileges` streamed once to the explicit temporary archive path;
- local SHA-256/byte-count reads and `pg_restore --list` parsing of that same archive.

`dbmigrate` check/apply and Goose `EnsureDBVersion` paths were not invoked.

## Decision

**ORDER 1 PASS. STOP before Package 5 Order 2.** The conditional user authorization remains recorded, but the current Leader gate requires independent acceptance of this evidence before any local migration apply or integration write.
