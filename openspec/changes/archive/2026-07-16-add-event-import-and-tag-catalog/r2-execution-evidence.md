# R2 Local PostgreSQL Execution Evidence

Status: executed under the explicit local R2 and failure-recovery authorizations.
Environment was verified as local loopback `tidewise_local` / `tidewise`; no
credential, DSN, password, or token is recorded here.

## Baseline and migration

- Input checkpoint before the first R2 write: `b1b9ce0`.
- Custom backup: `backend/.data/r2-backups/tidewise_local_event_import_20260716T122217Z_pre-000020.dump`.
- Backup SHA-256: `0df9818b9136eba9d5b77d61a74dc90c08f91a588e50e185131c921f304a85fa`.
- Fingerprints, dbmigrate pending set, read-only preflight, and JSON baseline passed at migration version 19.
- `000020_add_event_import_receipts_and_tag_seed.sql` applied once; final migration version is 20.
- Postflight passed: receipt schema, fixed active Event Agent source, 22 active frozen Tags, and count formulas.

## Failure recovery and final verification

The failure diagnosis found two scoped synthetic rows in each of receipts,
events, event sources, event Tag maps, and raw documents. The single authorized
FK-order cleanup removed only those prefix-scoped rows. Structured diagnosis
after cleanup reported zero for all five categories.

The repaired `TestEventImportPostgresIntegration` passed. Its post-test
structured diagnosis again reported zero synthetic rows for all five categories,
and global counts were unchanged across the test.

The reviewed-outbox CLI dry-run succeeded and produced a machine JSON artifact
at `backend/.data/r2-backups/event_import_final_recovery_dry_run_20260716T123644Z.json`.
Before/after counts were identical:

| Table | Count |
|---|---:|
| `event_import_receipts` | 0 |
| `events` | 0 |
| `event_sources` | 0 |
| `event_tag_maps` | 0 |
| `raw_documents` | 407 |

No real Event import, fixed source/Tag mutation outside `000020`,
`event_entity_links` write, Neo4j operation, restore, forward-fix, or repeat
migration was performed.
