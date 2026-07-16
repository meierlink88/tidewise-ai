# R2 Synthetic Integration Failure-Recovery Review Package

Status: prepared only. This package is not authorized or executed. The prior R2
run applied `000020` after backup/preflight/postflight, then failed the synthetic
integration replay while scanning receipt UUID[] values. It may have left only
rows whose identifiers use the synthetic prefixes below.

## Required separate authorization

The user must separately authorize this exact sequence in local `tidewise_local`
only: read-only diagnosis → scoped synthetic cleanup → fixed integration rerun →
CLI dry-run/receipt zero-write recheck. No migration rerun, real Event import,
source/Tag master mutation, `event_entity_links`, Neo4j, UAT/prod/shared,
automatic restore, retry, or forward-fix is authorized by preparing this file.

Recovery baseline is the existing custom backup
`backend/.data/r2-backups/tidewise_local_event_import_20260716T122217Z_pre-000020.dump`.
It is evidence only: do not automatically restore it.

## Read-only diagnosis

Run `r2-failure-recovery-diagnosis.sql` with the same local loopback credential
route. It fail-closes unless migration version is 20, receipt schema exists,
the fixed source is active/correct, and all 22 frozen Tag IDs are active. It
reports only `event-import-integration-%` receipts, `synthetic:event-import-integration-%`
events/source external IDs, and their event-source/tag-map rows.

## Scoped cleanup write

Only after diagnosis is reviewed and separately authorized, run
`r2-failure-recovery-cleanup.sql` once. It deletes only the above synthetic
prefix rows in one transaction and FK order: receipts → tag maps → sources →
events → raw documents. It never deletes the fixed source, frozen Tags, real
Events, or `event_entity_links`, and it fails if any scoped row remains.

## Reverification boundary

After cleanup passes, rerun only `TestEventImportPostgresIntegration` and the
reviewed-outbox CLI dry-run plus receipt zero-write check. Do not repeat
`000020`. Any diagnosis, cleanup, test, or zero-write failure stops the package
without automatic restore/retry/forward-fix.
