# Package 5 R2 Raw Receipt Schema Review And Preflight Plan

## Status And Authorization Boundary

**Status: REVIEW ONLY — NOT EXECUTED.** This package is submitted from the clean Package 4 checkpoint for independent Leader acceptance. No local database identity query, read-only transaction, backup, catalog query, migration command, SQL apply, or integration test has run in Package 5.

The recorded user authorization permits one local `000022` apply only after the approved amendment and exact Order 1 evidence pass. The current Leader gate is stricter: no local DB command may run until this Review/preflight package is independently accepted. Package 10 remains a separate R2 role/credential change and is excluded.

## Frozen Inputs

- Package 4 migration artifact: `backend/migrations/000022_add_raw_document_import_receipts.sql`.
- Exact artifact SHA-256: `3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26`.
- Historical 21-file aggregate SHA-256: `2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`.
- Expected local before state: existing Goose ledger, current applied version exactly 21, no higher applied version, repository pending set exactly `{000022}`, all frozen raw receipt objects absent.
- Expected after state: applied version exactly 22 once, frozen objects exact, initial and post-rollback-test receipt row count zero, all pre-existing business schema/data invariants unchanged.

Credentials/DSNs must remain environment-injected and must not appear in commands, logs, evidence, shell history excerpts, or artifacts.

## Layer 1 — Order 1 Read-Only Preflight

After Leader approval, execute only the following ordered evidence collection against the approved local identity:

1. Confirm the exact Package 4 commit, clean worktree, local environment label, 22 repository migration files, historical aggregate hash, and exact `000022` hash. Any code/artifact/hash drift stops the package.
2. Open a server-enforced `BEGIN READ ONLY` transaction. Record only redacted `current_database()`, current user/role identity, server address classification as local, and `current_setting('transaction_read_only') = 'on'`. A non-local target or unexpected identity stops the package.
3. Directly query `to_regclass('goose_db_version')` and ledger rows. Ledger absence stops the package; do not run `dbmigrate` check mode or any helper that invokes `goose.EnsureDBVersionContext`. Assert applied version exactly 21 and no higher applied row.
4. Compare the read-only ledger with `FileSource` repository discovery. Assert the only pending file is `000022`; no missing, duplicate, unknown-applied or extra pending migration is allowed.
5. Query `pg_catalog` for the exact table/constraint/index/function/trigger names. Assert all are absent before apply. Capture target schema ownership plus current identity `USAGE`/`CREATE`, ledger DML and migration DDL privilege facts without changing grants.
6. Capture approved business schema/table/data count invariants, including existing migration/table counts. Do not select or persist business payloads.
7. Create one clean full local backup archive at a new explicit temporary path, calculate its SHA-256, verify the archive can be listed/read, and record a documented recovery procedure. Do not restore and do not claim restore-tested.
8. Roll back/end the read-only transaction and submit the complete redacted Order 1 evidence. Any query, privilege, count, backup, archive-read, identity, scope or hash failure stops; no apply, retry, repair, restore or forward-fix follows.

Layer 1 permits reads plus creation of the explicitly reviewed backup artifact only. It performs no schema/business write and does not satisfy Layer 2 by itself.

## Layer 2 — Single Approved Local Apply And Verification

Only if every Layer 1 assertion is exact and the recorded conditional authorization remains valid:

1. From `backend/`, execute exactly once:

   ```text
   go run ./cmd/dbmigrate -apply -target-version 000022
   ```

   Do not use an unbounded target, rerun it, or apply any other pending migration.
2. Immediately use read-only queries to assert ledger applied version 22 exactly once; seven columns/types/default; every frozen named PK/unique/check/index/function/statement trigger; no extra receipt abstraction; and `COUNT(*) = 0` for the new table.
3. Re-run the Layer 1 business schema/data/hash/count invariants. The only allowed persistent changes are the Goose ledger record for `000022` and the frozen `000022` objects.
4. Run the dedicated local integration suite inside rollback-only transactions/savepoints: valid receipt insert/read and cross-field result, invalid constraint cases, atomic failure rollback, UPDATE/DELETE/TRUNCATE rejection, and two-connection advisory-lock blocking/release plus sorted overlapping raw-identity guard. Both connections must roll back; final receipt rows must be zero. Winner-commit/loser-replay remains covered by Package 4 state-machine/SQL winner-reread tests and must not persist a local race fixture.
5. Record redacted commands, input hashes, before/after assertions and targeted test results, then stop for Package 5 review. Do not proceed to Package 6 automatically.

## Stop And Recovery Rules

On partial state, assertion failure, timeout, unexpected applied version/object/data delta, nonzero receipt rows, or integration cleanup failure: stop immediately, preserve logs and the clean backup, and report the exact state. This authorization does not permit restore, retry, forward-fix, seed/import, role/grant/credential edits, Neo4j, UAT/prod/shared access, deployment, or Package 10 work. Recovery choice requires a new explicit review.
