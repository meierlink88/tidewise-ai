# Package 10.1 Local Data DB Role And Credential Review

## Compatibility Resolution Addendum

Leader independently accepted the blocker diagnosis and authorized a separate minimal R1 compatibility checkpoint from `f9081e9d1d286923960bc23a3c081b9a2b2d429b`. That checkpoint removes only the redundant `FOR UPDATE` suffix from raw-document and reviewed-event receipt lookups, while retaining the preceding idempotency-key `pg_advisory_xact_lock` and all replay/conflict/transaction behavior. Exact sqlmock tests require advisory-lock-first ordering followed by an anchored plain `SELECT`, so `FOR UPDATE`, `FOR SHARE` and any other trailing row-locking clause fail the contract.

The original verdict below remains the historical Package 10.1 review result at `f9081e9`; it is not retroactively rewritten. Package 10.1 remains unchecked until this R1 checkpoint is independently accepted and the read-only Review evidence is refreshed. No role/grant/credential or database operation is authorized by the compatibility correction.

## Review Verdict And Stop Boundary

**Overall: FAIL-CLOSED / BLOCKED before Package 10.2.** Database identity, migration, schema, count, owner/grant and backup assertions all pass. The proposed least-privilege runtime contract cannot yet be applied because both retained receipt import paths issue `SELECT ... FOR UPDATE`, which PostgreSQL 16 requires to have `UPDATE` privilege. The approved contract requires runtime to receive only necessary `SELECT/INSERT` and forbids receipt `UPDATE/DELETE/TRUNCATE` privilege.

- Raw receipt caller: `backend/services/data/rawimport/postgresstore/store.go:60-64` acquires the caller/key advisory transaction lock and then appends `FOR UPDATE` to the receipt query.
- Reviewed-event receipt caller: `backend/internal/repositories/event_import_postgres.go:60-72` acquires its idempotency advisory transaction lock and then selects `event_import_receipts ... FOR UPDATE`.
- PostgreSQL 16 documents that every locking clause including `FOR UPDATE`, `FOR SHARE` and their variants additionally requires `UPDATE` privilege on at least one selected-table column: <https://www.postgresql.org/docs/16/sql-select.html>.

Granting column or table `UPDATE` would weaken the frozen privilege contract. `raw_document_import_receipts` has an immutable trigger, but granting `UPDATE` would still violate the required negative privilege assertion; `event_import_receipts` has no equivalent immutable trigger. This review therefore does not propose an UPDATE grant as a workaround.

Before Package 10.2 can be authorized, a separately reviewed R1 correction must remove the redundant receipt row-locking clauses while preserving the already-present caller/key advisory transaction locks, receipt replay, winner re-read and race tests. This review did not make that production change. Task 10.1 remains unchecked until that blocker is resolved and the Review package is refreshed.

No role, membership, owner, grant, default privilege, password, credential, schema, migration, table, row or service configuration was changed. No Neo4j, seed, import, deployment, UAT/prod/shared, Package 11, restore or service start occurred.

## Per-Requirement Results

| # | Requirement | Result | Evidence |
|---:|---|---|---|
| 1 | Local identity and scope | PASS | Existing healthy `postgres:16` container `tidewise-local-postgres`; database `tidewise_local`; current/session role `tidewise`; PostgreSQL `16.14`; container-local Unix socket; `transaction_read_only=on`. Name, container and port classification are explicitly local and not UAT/prod/shared. |
| 2 | Ledger, repository migrations, hashes and `000022` objects | PASS | Applied positive set exactly `1..22`; version 22 exactly once; no higher row; repository exactly 22 SQL files. Historical 21 aggregate SHA-256 `2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`; `000022` SHA-256 `3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26`. Exact seven columns, seven constraints, three indexes, function and enabled statement trigger are present. |
| 3 | Business counts/hash and receipt count | PASS | All 41 frozen business-table counts exactly equal Package 5 before and final baselines. Sorted TSV SHA-256 before and after backup is `9da3142af32821d54869eaac63fd1bad5a4a2ab8bcb0e1ca0cef295c0ddf6549`; byte comparison exact. `raw_document_import_receipts=0`. |
| 4 | Complete roles/owners/grants/PUBLIC before manifest | PASS | Captured below in server-enforced read-only transactions. The three target roles are absent; all project relations/function are owned by `tidewise`; PUBLIC database/schema/function defaults remain visible. |
| 5 | Fresh readable backup and recovery/cutback | PASS | One new repo-external custom archive, exact path/size/hash below; PostgreSQL 16 `pg_restore --list` parsed 262 lines. Not restored and not claimed restore-tested. Recovery and old credential cutback are documented below. |
| 6 | Exact 10.2 transaction plan | FAIL / BLOCKED | Candidate DCL is frozen below, but it is conditional on resolving the `FOR UPDATE`/UPDATE-privilege conflict without weakening the approved runtime contract. No DCL was executed. |
| 7 | Exact 10.3 credential cutover plan | BLOCKED BY #6 | Cutover plan is complete below, secrets remain environment-only, Data uses `data_service_rw`, and BFF/Agent remain DB-free. It cannot be authorized until #6 is compatible and independently accepted. |
| 8 | Before/after assertions, stop and partial-write rules | PASS AS PLAN | Exact assertions and fail-closed behavior are documented. Any pre-commit assertion rolls back the single DCL transaction; any post-commit mismatch is a partial-write condition and stops without restore/retry/forward-fix. |

One read-only ACL expansion query initially failed before producing results because a `CASE` expression inferred SQL `character` rather than internal `"char"` for `acldefault`. The transaction aborted with no state change. The query was split into table and sequence branches and rerun successfully. The host lacked `pg_restore`; the same archive was then listed through the PostgreSQL 16 container from stdin. Neither diagnostic created database state or a second backup.

## Identity, Migration And Schema Manifest

```text
target classification: approved local container/database
container: tidewise-local-postgres (postgres:16, healthy)
database: tidewise_local
current/session role: tidewise
server: PostgreSQL 16.14, container-local Unix socket
transaction_read_only: on

applied positive migrations: 22
applied set: 1..22
version 22 rows: 1
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

Frozen raw receipt objects are exact:

- columns in order: `id uuid`, `caller_identity text`, `idempotency_key text`, `payload_hash character(64)`, `raw_document_ids uuid[]`, `result_payload jsonb`, `imported_at timestamptz DEFAULT now()`; all NOT NULL;
- constraints: `raw_document_import_receipts_pkey`, `uq_raw_document_import_receipts_caller_key`, `chk_raw_document_import_receipts_caller`, `chk_raw_document_import_receipts_key`, `chk_raw_document_import_receipts_payload_hash`, `chk_raw_document_import_receipts_raw_ids`, `chk_raw_document_import_receipts_result`;
- indexes: PK and caller/key unique backing indexes plus `idx_raw_document_import_receipts_imported_at`;
- `prevent_raw_document_import_receipt_mutation()` is PL/pgSQL, owned by `tidewise`, raises SQLSTATE `55000`;
- `trg_raw_document_import_receipts_immutable` is enabled and is `BEFORE DELETE OR UPDATE OR TRUNCATE FOR EACH STATEMENT`.

## Fresh Business Count Manifest

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

Sorted tab-separated manifest SHA-256: `9da3142af32821d54869eaac63fd1bad5a4a2ab8bcb0e1ca0cef295c0ddf6549`.

## Complete Before Role, Owner And Grant Manifest

Project and predefined roles:

- Project role `tidewise`: `SUPERUSER INHERIT CREATEROLE CREATEDB LOGIN REPLICATION BYPASSRLS`, unlimited connections, no expiry.
- PostgreSQL predefined NOLOGIN roles: `pg_checkpoint`, `pg_create_subscription`, `pg_database_owner`, `pg_execute_server_program`, `pg_monitor`, `pg_read_all_data`, `pg_read_all_settings`, `pg_read_all_stats`, `pg_read_server_files`, `pg_signal_backend`, `pg_stat_scan_tables`, `pg_use_reserved_connections`, `pg_write_all_data`, `pg_write_server_files`. Each is non-superuser, non-createrole, non-createdb, non-replication, non-bypassrls, INHERIT, unlimited, no expiry.
- Memberships are only PostgreSQL built-ins: `pg_monitor` is a member of `pg_read_all_settings`, `pg_read_all_stats`, and `pg_stat_scan_tables`, without admin option. No project-role membership exists.
- `data_service_migrate`, `data_service_rw`, and `data_service_ro` do not exist.

Ownership and raw ACLs:

- database `tidewise_local`: owner `tidewise`, null ACL (default owner plus PUBLIC database privileges);
- schema `public`: owner `pg_database_owner`, ACL `pg_database_owner=UC`, `PUBLIC=U`;
- all 43 public tables, including Goose and `raw_document_import_receipts`: owner `tidewise`, null ACL;
- sequence `goose_db_version_id_seq`: owner `tidewise`, null ACL;
- function `prevent_raw_document_import_receipt_mutation()`: owner `tidewise`, null ACL.

Expanded effective grants:

- PUBLIC: database `CONNECT,TEMPORARY`; schema `USAGE`; function `EXECUTE`; no table or sequence privilege.
- `pg_database_owner`: schema `USAGE,CREATE`.
- `tidewise`: database `CONNECT,TEMPORARY,CREATE`; all seven table privileges on all 43 tables; sequence `USAGE`; function `EXECUTE`. Owner powers additionally apply independently of ACL display.

The exact 43-table owner set is:

```text
alliance_org_profiles, benchmark_observations, benchmark_profiles,
chain_node_physical_constraints, chain_node_profiles, chain_node_relations,
commodity_profiles, company_profiles, economy_profiles, entity_edges,
entity_external_identifiers, entity_nodes, event_entity_links,
event_import_receipts, event_sources, event_tag_defs, event_tag_maps, events,
goose_db_version, graph_projection_run_items, graph_projection_runs,
index_profiles, ingestion_run_sources, ingestion_runs,
ingestion_scheduler_configs, instrument_profiles, market_profiles,
metric_profiles, person_profiles, policy_body_profiles,
raw_document_import_receipts, raw_documents,
research_anchor_chain_nodes, research_anchor_events,
research_anchor_indices, research_anchors, research_theme_chain_nodes,
research_theme_events, research_theme_indices, research_themes,
security_profiles, source_catalogs, theme_profiles
```

## Backup And Recovery Evidence

```text
path: /private/tmp/tidewise-p10-review-119f735e6263-pre-roles.dump
format: PostgreSQL custom archive, --no-owner --no-privileges
bytes: 1379080
sha256: 434e1eb2cc64ba145160d20a36b8d9852b77ea0309009d060337212f231c1e66
pg_restore --list: 262 lines, exit 0 via the PostgreSQL 16 container
restore executed: no
restore-tested claim: no
```

Recovery procedure requiring separate authorization:

1. Stop local writers and verify the archive path, byte count and SHA-256 again.
2. Create a new, empty, explicitly named isolated local recovery database; never target `tidewise_local` in the first recovery attempt.
3. List the archive again, then run `pg_restore --no-owner --no-privileges --exit-on-error --dbname=<isolated-local-recovery-db> <archive>` using environment-injected connection material.
4. Re-run ledger, seven-column/schema objects, 41-table count/hash and receipt-zero assertions against the isolated database.
5. Submit any in-place restore or cutover decision for a new explicit authorization. No restore was performed in this Review.

Old credential cutback for a future Package 10.3 failure does not require restore: stop the newly started local Data process, replace only its process-local `TIDEWISE_DATABASE_URL` with the preserved pre-cutover environment value, restart Data, and recheck `/healthz` and `/readyz`. The existing `tidewise` role/credential is not altered or dropped by the candidate plan. Do not reverse DCL, restore, retry or forward-fix without a new review.

## Candidate Package 10.2 Role And Grant Contract

This is an exact candidate only after the receipt-lock blocker is resolved and independently reviewed. It was not executed.

Role attributes:

| Role | Login | Purpose | Attributes |
|---|---|---|---|
| `data_service_migrate` | LOGIN | database/schema/object owner and migration job only | `NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS NOINHERIT`, unlimited connection, no expiry, SCRAM password from temporary environment |
| `data_service_rw` | LOGIN | Data Service runtime only | same negative attributes and temporary-environment SCRAM password |
| `data_service_ro` | NOLOGIN | controlled audit role reached only by separately authorized `SET ROLE`; never a BFF credential | same negative attributes; no password |

No membership is granted between these roles. The existing `tidewise` role remains unchanged and is not dropped. The candidate transfers the local Data database and every frozen public table/sequence/function owner to `data_service_migrate`, making it the actual DDL owner; this includes the mandatory raw receipt table/function transfer. It does not drop or rename any object.

Runtime SELECT set, derived from Data startup, Admin/Research reads and both retained import paths:

```text
goose_db_version,
source_catalogs, raw_documents, raw_document_import_receipts,
event_import_receipts, events, event_sources, event_tag_defs, event_tag_maps,
research_themes, research_theme_chain_nodes, research_theme_indices,
research_theme_events, research_anchors, research_anchor_chain_nodes,
research_anchor_indices, research_anchor_events, entity_nodes
```

Runtime INSERT set:

```text
raw_documents, raw_document_import_receipts,
events, event_sources, event_tag_maps, event_import_receipts
```

Runtime receives no sequence, function, schema CREATE, database CREATE/TEMP, table UPDATE/DELETE/TRUNCATE/REFERENCES/TRIGGER or owner privilege. `data_service_ro` receives SELECT on all 43 public tables only. `data_service_migrate` owns the database, public objects and Goose sequence/function, and is the only new role with DDL/owner power.

Exact transaction body, with secrets read by psql `\getenv` from process-only environment variables:

```sql
\getenv migrate_password DATA_SERVICE_MIGRATE_PASSWORD
\getenv rw_password DATA_SERVICE_RW_PASSWORD

BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SET LOCAL password_encryption = 'scram-sha-256';
SELECT pg_advisory_xact_lock(hashtextextended('tidewise:package10:local-data-db-role-boundary', 0));

CREATE ROLE data_service_migrate LOGIN NOINHERIT NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'migrate_password';
CREATE ROLE data_service_rw LOGIN NOINHERIT NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'rw_password';
CREATE ROLE data_service_ro NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOREPLICATION NOBYPASSRLS;

ALTER DATABASE tidewise_local OWNER TO data_service_migrate;

ALTER TABLE public.alliance_org_profiles OWNER TO data_service_migrate;
ALTER TABLE public.benchmark_observations OWNER TO data_service_migrate;
ALTER TABLE public.benchmark_profiles OWNER TO data_service_migrate;
ALTER TABLE public.chain_node_physical_constraints OWNER TO data_service_migrate;
ALTER TABLE public.chain_node_profiles OWNER TO data_service_migrate;
ALTER TABLE public.chain_node_relations OWNER TO data_service_migrate;
ALTER TABLE public.commodity_profiles OWNER TO data_service_migrate;
ALTER TABLE public.company_profiles OWNER TO data_service_migrate;
ALTER TABLE public.economy_profiles OWNER TO data_service_migrate;
ALTER TABLE public.entity_edges OWNER TO data_service_migrate;
ALTER TABLE public.entity_external_identifiers OWNER TO data_service_migrate;
ALTER TABLE public.entity_nodes OWNER TO data_service_migrate;
ALTER TABLE public.event_entity_links OWNER TO data_service_migrate;
ALTER TABLE public.event_import_receipts OWNER TO data_service_migrate;
ALTER TABLE public.event_sources OWNER TO data_service_migrate;
ALTER TABLE public.event_tag_defs OWNER TO data_service_migrate;
ALTER TABLE public.event_tag_maps OWNER TO data_service_migrate;
ALTER TABLE public.events OWNER TO data_service_migrate;
ALTER TABLE public.goose_db_version OWNER TO data_service_migrate;
ALTER TABLE public.graph_projection_run_items OWNER TO data_service_migrate;
ALTER TABLE public.graph_projection_runs OWNER TO data_service_migrate;
ALTER TABLE public.index_profiles OWNER TO data_service_migrate;
ALTER TABLE public.ingestion_run_sources OWNER TO data_service_migrate;
ALTER TABLE public.ingestion_runs OWNER TO data_service_migrate;
ALTER TABLE public.ingestion_scheduler_configs OWNER TO data_service_migrate;
ALTER TABLE public.instrument_profiles OWNER TO data_service_migrate;
ALTER TABLE public.market_profiles OWNER TO data_service_migrate;
ALTER TABLE public.metric_profiles OWNER TO data_service_migrate;
ALTER TABLE public.person_profiles OWNER TO data_service_migrate;
ALTER TABLE public.policy_body_profiles OWNER TO data_service_migrate;
ALTER TABLE public.raw_document_import_receipts OWNER TO data_service_migrate;
ALTER TABLE public.raw_documents OWNER TO data_service_migrate;
ALTER TABLE public.research_anchor_chain_nodes OWNER TO data_service_migrate;
ALTER TABLE public.research_anchor_events OWNER TO data_service_migrate;
ALTER TABLE public.research_anchor_indices OWNER TO data_service_migrate;
ALTER TABLE public.research_anchors OWNER TO data_service_migrate;
ALTER TABLE public.research_theme_chain_nodes OWNER TO data_service_migrate;
ALTER TABLE public.research_theme_events OWNER TO data_service_migrate;
ALTER TABLE public.research_theme_indices OWNER TO data_service_migrate;
ALTER TABLE public.research_themes OWNER TO data_service_migrate;
ALTER TABLE public.security_profiles OWNER TO data_service_migrate;
ALTER TABLE public.source_catalogs OWNER TO data_service_migrate;
ALTER TABLE public.theme_profiles OWNER TO data_service_migrate;
ALTER SEQUENCE public.goose_db_version_id_seq OWNER TO data_service_migrate;
ALTER FUNCTION public.prevent_raw_document_import_receipt_mutation()
  OWNER TO data_service_migrate;

REVOKE ALL ON DATABASE tidewise_local FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;

GRANT CONNECT, TEMPORARY, CREATE ON DATABASE tidewise_local TO data_service_migrate;
GRANT CONNECT ON DATABASE tidewise_local TO data_service_rw, data_service_ro;
GRANT USAGE, CREATE ON SCHEMA public TO data_service_migrate;
GRANT USAGE ON SCHEMA public TO data_service_rw, data_service_ro;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO data_service_ro;
GRANT SELECT ON TABLE
  public.goose_db_version,
  public.source_catalogs, public.raw_documents,
  public.raw_document_import_receipts, public.event_import_receipts,
  public.events, public.event_sources, public.event_tag_defs,
  public.event_tag_maps, public.research_themes,
  public.research_theme_chain_nodes, public.research_theme_indices,
  public.research_theme_events, public.research_anchors,
  public.research_anchor_chain_nodes, public.research_anchor_indices,
  public.research_anchor_events, public.entity_nodes
TO data_service_rw;
GRANT INSERT ON TABLE
  public.raw_documents, public.raw_document_import_receipts,
  public.events, public.event_sources, public.event_tag_maps,
  public.event_import_receipts
TO data_service_rw;

ALTER DEFAULT PRIVILEGES FOR ROLE data_service_migrate IN SCHEMA public
  REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

-- The approved execution package must run owner/grant/negative assertions here.
-- Any mismatch raises before COMMIT so the entire transaction rolls back.
COMMIT;
```

The planned invocation consumes secrets only from temporary environment variables and a reviewed SQL file containing no values:

```text
DATA_SERVICE_MIGRATE_PASSWORD=<temporary> DATA_SERVICE_RW_PASSWORD=<temporary> \
  psql --no-psqlrc --set=ON_ERROR_STOP=1 --file=<reviewed-package10-sql>
```

The actual connection identity/DSN and secret values must not be printed, committed, written into config, shell history or evidence.

## Candidate Package 10.3 Credential Cutover

Only after Package 10.2 passes and receives a separate authorization:

1. Re-run the same local identity, ledger, migration hashes, schema, count hash, receipt-zero, owner/grant and backup hash assertions.
2. Preserve the existing process-local Data database environment value for immediate cutback without printing it.
3. Export the new `data_service_rw` connection secret only in the Data process environment and set Data's `TIDEWISE_DATABASE_URL` to the local database. Do not edit committed YAML, compose, `.env`, shell profile or secret files.
4. Confirm Miniapp/Admin/Agent/agent-run process environments and service assets contain no PostgreSQL URL/user/password. Their only Data access remains HTTP identity tokens.
5. Start only the local Data service with its required service-identity tokens; verify `/healthz` and `/readyz` and the read-only migration readiness path. No migration, seed, import, projection or business write is allowed.
6. Positive privilege checks: `CONNECT`, schema `USAGE`, and the exact SELECT/INSERT allowlists are true. Negative checks: database/schema CREATE/TEMP, sequence/function privileges and UPDATE/DELETE/TRUNCATE/REFERENCES/TRIGGER on every runtime table are false; raw receipt mutation privileges are all false.
7. Stop the new process and restore the old process-local Data connection value if health/readiness or any privilege assertion fails. Do not alter roles/grants again, restore the archive, retry, or forward-fix.

## Exact Before/After Assertions And Stop Conditions

Before Package 10.2:

- exact local identity and PostgreSQL 16 target;
- clean accepted code checkpoint and resolved receipt-lock compatibility;
- target roles absent, or an independently reviewed exact matching manifest if the checkpoint is refreshed;
- ledger exact `1..22`, version 22 once, no higher row;
- historical and `000022` hashes exact;
- 43/373/203/104/1/1 structural counts and exact frozen raw receipt objects;
- all 41 business counts and SHA-256 exact; receipt rows zero;
- before owners/grants/PUBLIC manifest exact;
- backup path, size, SHA-256 and 262-line readability exact.

After Package 10.2, before commit and again read-only after commit:

- exact three roles and attributes; no unexpected membership;
- database, all 43 tables, Goose sequence and receipt function owned by `data_service_migrate`;
- PUBLIC has no database/schema/table/sequence/function privilege in the reviewed scope;
- `data_service_ro` has only SELECT on all 43 tables;
- `data_service_rw` has only database CONNECT, schema USAGE and the exact SELECT/INSERT allowlists;
- `has_table_privilege(data_service_rw,'raw_document_import_receipts','UPDATE,DELETE,TRUNCATE')` is false for every mutation privilege, and equivalent negative assertions hold for all runtime tables;
- no runtime function/sequence privilege;
- ledger, migration hashes, schema object definitions, all 41 counts/hash and receipt zero unchanged.

Stop rules:

- Any identity, role collision, lock timeout, owner/grant, PUBLIC, ledger, count/hash, schema, receipt-row or backup mismatch before COMMIT causes transaction rollback and immediate stop.
- Any error or timeout during DCL causes rollback and immediate stop; no second execution.
- Any mismatch discovered after COMMIT is explicitly a partial-write state: preserve exact catalogs/logs and backup, stop all new-role processes, use only the old credential cutback, and report. Do not restore, retry, reverse DCL, drop roles, forward-fix or enter Package 10.3.
- Package 10.3 failure stops after old credential cutback; it does not authorize Package 11, database restore, migration, business write, Neo4j or deployment.
