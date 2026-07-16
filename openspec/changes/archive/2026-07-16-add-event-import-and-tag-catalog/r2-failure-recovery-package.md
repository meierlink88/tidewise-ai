# R2 Synthetic Integration Failure-Recovery Review Package

Status: prepared only. The prior R2 run applied `000020` after
backup/preflight/postflight, then failed synthetic replay while scanning receipt
UUID[] values. A later verification run also closed `*sql.DB` before its
`t.Cleanup` fixture cleanup. Either run may have left only rows whose identifiers
use the synthetic prefixes below; this package never assumes a residual count.

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

Only if diagnosis reports nonzero scoped synthetic rows, and after separate
authorization, run `r2-failure-recovery-cleanup.sql` once. If diagnosis is
already zero, do not run cleanup. The script deletes only the above synthetic
prefix rows in one transaction and FK order: receipts → tag maps → sources →
events → raw documents. It never deletes the fixed source, frozen Tags, real
Events, or `event_entity_links`, and it fails if any scoped row remains.

Run the post-cleanup diagnosis in tuple-only, tab-separated mode and require
each expected table/key to be exactly zero; do not parse aligned psql tables:

```bash
docker exec -i tidewise-local-postgres sh -lc 'exec psql "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@127.0.0.1:5432/${POSTGRES_DB}?sslmode=disable" "$@"' sh \
  -X -qAt -F $'\t' -v ON_ERROR_STOP=1 < r2-failure-recovery-diagnosis.sql |
  awk -F $'\t' '
    $1=="event_import_receipts" || $1=="events" || $1=="event_sources" || $1=="event_tag_maps" || $1=="raw_documents" { if (!($1 in seen)) count++; seen[$1]=$2 }
    END { for (key in seen) if (seen[key] != "0") exit 1; exit (count == 5 ? 0 : 1) }'
```

## Reverification boundary

After diagnosis is zero (or cleanup passes), rerun only the fixed
`TestEventImportPostgresIntegration`, confirm zero synthetic residual rows, and
then run the reviewed-outbox CLI dry-run plus five-table zero-write check. Do
not repeat `000020`. Any diagnosis, cleanup, test, residual, or zero-write
failure stops the package without automatic restore/retry/forward-fix.
