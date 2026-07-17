# Package 10.1 Fresh Read-Only Review Refresh

## Scope And Result

- Code baseline: accepted clean checkpoint `18e615deef5a026f0f37f419d83266b0b9bc0676`.
- Target: the same healthy local-only `postgres:16` container and `tidewise_local` database used by the original Package 10.1 review.
- **Overall result: PASS.** All eight required refresh areas match the frozen review baseline. Task 10.1 may be completed.
- This refresh authorizes no Package 10.2/10.3 operation. No `CREATE`, `ALTER`, `GRANT`, `REVOKE`, owner/password/credential change, migration, DML, seed, import, Neo4j, deployment, UAT/prod/shared action, restore, retry or forward-fix ran.

All database queries were direct catalog/count queries inside server-enforced `BEGIN READ ONLY` transactions. Connection material was not printed or persisted.

## Per-Requirement PASS/FAIL

| # | Requirement | Result | Fresh evidence |
|---:|---|---|---|
| 1 | Local identity/scope | PASS | Healthy container `tidewise-local-postgres`, image `postgres:16`; PostgreSQL `16.14`; database `tidewise_local`; current/session role `tidewise`; container-local Unix socket; `transaction_read_only=on`. Explicitly local, not UAT/prod/shared. |
| 2 | Ledger, repository migrations, hashes and `000022` schema | PASS | Applied set exactly `1..22`, version 22 exactly once, no higher row; repository files exactly 22. Historical 21 aggregate and `000022` hashes exact. Seven columns, seven constraints, three indexes, function and enabled statement trigger exact; structural counts remain `43/373/203/104/1/1`. |
| 3 | Business counts/hash and receipt zero | PASS | Fresh sorted manifest contains exactly 41 tables and every count matches Package 5/original Package 10 review. SHA-256 remains `9da3142af32821d54869eaac63fd1bad5a4a2ab8bcb0e1ca0cef295c0ddf6549`; raw receipt rows remain `0`. |
| 4 | Roles/attributes/membership/owners/grants/PUBLIC | PASS | Target role count remains zero. Existing roles, three built-in memberships, database/schema/table/sequence/function owners, raw ACLs and expanded grants exactly match the original complete before manifest. |
| 5 | Receipt locking compatibility | PASS | Accepted checkpoint `18e615d` retains the transaction-scoped idempotency advisory lock before an anchored plain receipt `SELECT` in both adapters. Fresh sqlmock tests pass; static scan finds no row-locking clause. |
| 6 | Existing backup | PASS | Original archive exists at the exact path, is `1,379,080` bytes, SHA-256 exact, and PostgreSQL 16 `pg_restore --list` returns 262 lines. It was not replaced or restored. |
| 7 | Candidate 10.2 DCL and 10.3 cutover audit | PASS | Exact allowlists, negative assertions, environment-only secrets, old-credential cutback and stop rules remain sufficient after removal of redundant row locks. Runtime needs no UPDATE, sequence or function privilege. |
| 8 | Stop/partial-write boundary | PASS | Any future identity/count/hash/schema/role/grant/backup drift stops before DCL; pre-commit DCL failure rolls back; post-commit mismatch is partial state and stops without retry, restore or forward-fix. No future layer is implicitly authorized. |

## Fresh Identity, Ledger And Schema Results

```text
transaction_read_only: on
container/image: tidewise-local-postgres / postgres:16
database: tidewise_local
current/session role: tidewise
server: PostgreSQL 16.14, container-local Unix socket
environment classification: local only; not UAT/prod/shared

applied positive migrations: 22
applied set: 1..22
version 22 applied rows: 1
higher applied rows: 0
repository migration files: 22
historical 000001..000021 aggregate SHA-256:
  2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc
000022 SHA-256:
  3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26

public tables: 43
columns: 373
constraints: 203
indexes: 104
public functions: 1
non-internal public triggers: 1
raw_document_import_receipts rows: 0
```

`raw_document_import_receipts` remains the exact seven-column immutable contract:

- `id uuid`, `caller_identity text`, `idempotency_key text`, `payload_hash character(64)`, `raw_document_ids uuid[]`, `result_payload jsonb`, `imported_at timestamptz DEFAULT now()`; all NOT NULL;
- constraints `raw_document_import_receipts_pkey`, `uq_raw_document_import_receipts_caller_key`, `chk_raw_document_import_receipts_caller`, `chk_raw_document_import_receipts_key`, `chk_raw_document_import_receipts_payload_hash`, `chk_raw_document_import_receipts_raw_ids`, `chk_raw_document_import_receipts_result`;
- PK/unique backing indexes plus `idx_raw_document_import_receipts_imported_at`;
- function `prevent_raw_document_import_receipt_mutation()` owned by `tidewise`, definition MD5 `10d2fc92cc384550044e1959ffed4a92`;
- enabled `trg_raw_document_import_receipts_immutable`, `BEFORE DELETE OR UPDATE OR TRUNCATE FOR EACH STATEMENT`.

## Fresh 41-Table Count Manifest

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

Fresh temporary manifest: `/private/tmp/tidewise-p10-refresh-18e615d-business-counts.tsv`, 41 lines, SHA-256 `9da3142af32821d54869eaac63fd1bad5a4a2ab8bcb0e1ca0cef295c0ddf6549`. It contains only table names and counts and is outside the repository.

## Fresh Role, Owner And Grant Before Manifest

- Target roles `data_service_migrate`, `data_service_rw`, `data_service_ro`: exactly `0` existing roles.
- Existing project role `tidewise`: unchanged `SUPERUSER INHERIT CREATEROLE CREATEDB LOGIN REPLICATION BYPASSRLS`, unlimited connections, no expiry.
- Fourteen PostgreSQL predefined roles remain unchanged, all NOLOGIN/non-superuser/non-createrole/non-createdb/non-replication/non-bypassrls.
- Memberships remain only the three built-in memberships of `pg_monitor` in `pg_read_all_settings`, `pg_read_all_stats` and `pg_stat_scan_tables`; no project membership exists.
- Database `tidewise_local`: owner `tidewise`, null ACL.
- Schema `public`: owner `pg_database_owner`; `pg_database_owner=USAGE,CREATE`, PUBLIC=`USAGE`.
- All 43 public tables: owner `tidewise`, null ACL. This is the exact table set recorded in `package-10-role-credential-review.md`, with no addition, removal or owner/ACL drift.
- Sequence `goose_db_version_id_seq`: owner `tidewise`, null ACL.
- Function `prevent_raw_document_import_receipt_mutation()`: owner `tidewise`, null ACL.
- Expanded effective PUBLIC grants remain exactly database `CONNECT,TEMPORARY`, schema `USAGE`, function `EXECUTE`, and no table/sequence privilege.
- Expanded `tidewise` grants remain database `CONNECT,TEMPORARY,CREATE`, owner table privileges on all 43 tables, sequence `USAGE` and function `EXECUTE`.

This exactly matches the complete original before manifest; no target role or security catalog mutation occurred between reviews.

## Accepted Receipt Compatibility Evidence

- Accepted code checkpoint: `18e615deef5a026f0f37f419d83266b0b9bc0676`.
- Raw import: caller/key `pg_advisory_xact_lock` runs first in the same transaction, followed by plain `receiptSelectSQL`.
- Reviewed-event import: idempotency-key `pg_advisory_xact_lock` runs first in the same transaction, followed by plain receipt `SELECT`.
- Fresh anchored sqlmock tests for both adapters: PASS.
- Fresh static scan for `FOR UPDATE`, `FOR SHARE`, `FOR KEY SHARE`, `FOR NO KEY UPDATE` in both adapters: zero matches.
- Receipt key/hash/result, transaction, winner re-read, replay, conflict, rollback and error contracts remain unchanged.

## Reused Backup Verification

```text
path: /private/tmp/tidewise-p10-review-119f735e6263-pre-roles.dump
bytes: 1379080
sha256: 434e1eb2cc64ba145160d20a36b8d9852b77ea0309009d060337212f231c1e66
format/readability: PostgreSQL custom archive; pg_restore --list = 262 lines
new backup created: no
restore executed: no
restore-tested claim: no
```

The original isolated-database recovery procedure and old process-local credential cutback remain unchanged. Any mismatch would have stopped this refresh without replacement, restore or repair.

## Candidate 10.2/10.3 Audit

The exact candidate transaction in `package-10-role-credential-review.md` was re-read against the accepted code and current schema.

`data_service_rw` positive allowlist remains exact:

- database `CONNECT`;
- schema `USAGE`;
- SELECT on 18 tables: `goose_db_version`, `source_catalogs`, `raw_documents`, `raw_document_import_receipts`, `event_import_receipts`, `events`, `event_sources`, `event_tag_defs`, `event_tag_maps`, `research_themes`, `research_theme_chain_nodes`, `research_theme_indices`, `research_theme_events`, `research_anchors`, `research_anchor_chain_nodes`, `research_anchor_indices`, `research_anchor_events`, `entity_nodes`;
- INSERT on six tables: `raw_documents`, `raw_document_import_receipts`, `events`, `event_sources`, `event_tag_maps`, `event_import_receipts`.

Negative assertions remain exact: no database/schema CREATE or TEMP, no sequence/function privilege, no UPDATE/DELETE/TRUNCATE/REFERENCES/TRIGGER on any runtime table, no ownership and no role membership. Removing the two redundant row locks means both receipt paths now operate with this SELECT/INSERT-only contract; no UPDATE grant is needed.

The candidate still excludes business DML, migration, seed/import/projection, Neo4j, deployment, UAT/prod/shared and any BFF/Agent database credential. Package 10.3 still injects Data's `data_service_rw` secret only through temporary process environment, preserves the old process-local credential for cutback, and verifies BFF/Agent remain DB-free.

Future execution remains fail-closed: any before drift stops without DCL; any transaction error rolls back; any post-commit mismatch is partial state and stops without restore, retry, reverse DCL or forward-fix. This Review completion does not itself authorize either future layer.

## Checkpoint Validation

- PASS: OpenSpec strict validate.
- PASS: explicit `OPENSPEC_TASK_LINT_CHANGE=establish-data-service-bff-boundaries` task-design lint.
- PASS: artifact status 4/4; task progress `43/49`, with only 10.1 newly completed.
- PASS: anchored raw/reviewed-event plain-SELECT sqlmock tests with `TIDEWISE_TEST_DATABASE_URL` unset.
- PASS: `git diff --check`, scoped manifest and secret scan.
- Scoped change: this refresh evidence, the historical Review addendum status, and task 10.1 checkbox only. No production, migration, infra, deployment or environment file changed.
