# Package 10 Receipt Lock Compatibility Evidence

## Scope

- Input checkpoint: `f9081e9d1d286923960bc23a3c081b9a2b2d429b`.
- R1-only compatibility correction: remove the redundant receipt `FOR UPDATE` suffix after the existing idempotency-key advisory transaction lock in the raw-document and reviewed-event import adapters.
- No receipt key/hash/result, validation, transaction, winner re-read, replay, conflict, rollback or error contract changes.
- No PostgreSQL connection, `TIDEWISE_TEST_DATABASE_URL`, synthetic DML, schema, migration, seed, data, role, owner, grant, credential, backup, Neo4j, deployment or environment operation.
- Package 10.1 remains incomplete; Package 10.2/10.3 were not entered.

## TDD Evidence

RED:

- `TestLockReceiptUsesAdvisoryTransactionLockBeforePlainSelect` first matched the raw caller/key `pg_advisory_xact_lock`, then rejected the actual receipt query because it had the unapproved trailing `FOR UPDATE`.
- `TestEventImportLockReceiptUsesAdvisoryTransactionLockBeforePlainSelect` first matched the reviewed-event key `pg_advisory_xact_lock`, then rejected the actual receipt query for the same trailing clause.

GREEN contract:

- Both sqlmock tests enforce statement order and anchor the full receipt query at the end of plain `SELECT`; any `FOR UPDATE`, `FOR SHARE`, `FOR KEY SHARE`, `FOR NO KEY UPDATE` or other suffix fails.
- Existing raw import replay/conflict/ordered transaction tests and reviewed-event same-key replay/different-hash conflict/rollback tests remain unchanged.
- The real PostgreSQL event-import integration file remains compiled but its write path is skipped because `TIDEWISE_TEST_DATABASE_URL` is deliberately unset.

## Stop Boundary

This checkpoint only removes a redundant row lock already dominated by the same transaction's advisory idempotency lock. If tests showed the advisory lock did not precede the query or did not cover the transaction, work would stop rather than expand grants or scope. No such drift was found. Package 10.1 must be independently refreshed and accepted before any R2 action.

## Validation Results

- PASS: raw sqlmock RED failed only because the query contained trailing `FOR UPDATE`; GREEN passed after the one-line removal.
- PASS: reviewed-event sqlmock RED failed only because the query contained trailing `FOR UPDATE`; GREEN passed after the one-line removal.
- PASS: `env -u TIDEWISE_TEST_DATABASE_URL go test ./services/data/rawimport/... -count=1`.
- PASS: `env -u TIDEWISE_TEST_DATABASE_URL go test ./internal/apps/ingestion/eventimport -count=1`.
- PASS: full `env -u TIDEWISE_TEST_DATABASE_URL go test ./internal/repositories -count=1`.
- PASS/SKIP as authorized: the explicit repository selection compiled `TestEventImportPostgresIntegration` and reported its expected missing-DSN skip; no real PostgreSQL path ran.
- PASS: static scan found no row-locking clause in either production receipt adapter.
- PASS: OpenSpec strict validate, explicit task-design lint, artifact status 4/4, diff/whitespace/scope/secret checks. Post-commit `git show --check` is recorded in the handoff.
