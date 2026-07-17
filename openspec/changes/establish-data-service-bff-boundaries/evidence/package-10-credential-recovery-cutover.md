# Package 10.3 Local Credential Recovery And Cutover Evidence

## Scope And Result

- Input checkpoint: independently accepted clean `7c74947b49e7d89eb5e81d2157eefa4735440b50`.
- Authorization: the user explicitly approved Package 10.3; Leader independently accepted the evidence-only recovery Review and its frozen one-transaction plan.
- Target: the same approved local-only PostgreSQL 16 `tidewise_local`; explicitly not UAT, prod or shared.
- Overall result: **PASS**. One runtime-password recovery transaction committed, the complete database contract remained unchanged, actual password authentication and exact runtime privileges passed, one temporary loopback Data Service process passed health/readiness/read-only endpoint checks, and the process/listener/secret were cleaned up.
- Package 10.3 is complete. Package 11 was not entered.

No migration, seed, import, projection, synthetic or business DML, schema change, owner/grant change, migrate-secret rotation, existing `tidewise` credential change, Neo4j action, deployment, restore, retry, reverse DCL, forward-fix, or UAT/prod/shared access occurred.

## Repo-Outside Harness Preparation

All runtime artifacts remained under `/private/tmp`; no credential or DSN was stored in them.

| Artifact | SHA-256 | Contract |
|---|---|---|
| password recovery SQL | `bb592524233f1429aa3e48947463e9defae5e0f6775328053a81f319872af1a6` | Exactly one state statement: `ALTER ROLE data_service_rw PASSWORD`; same advisory transaction lock, bounded timeouts, SCRAM setting and complete pre-commit assertions. |
| exact rw login/privilege SQL | `5db2eed0bdd77eda19746cdcab9ee8b6fae3ec418306f15e3a376ebe76bab5b9` | Server-enforced read-only actual-login and exact positive/negative privilege assertions; no state statement. |
| accepted post-contract SQL | `84058a6ba461e7c027761201a29b3249fa703973ab252cce0fdfce600f0248c9` | Complete `P10_POST_DCL_PASS` role/owner/grant/PUBLIC/ledger/schema/count/receipt audit. |
| current-checkpoint Data binary | `c212d7d5e37f51210a751d3d052f33d683856cdc957bdac6bf6e394faae971f2` | Built from clean `7c74947`; used only for this temporary loopback validation. |

The temporary Data config bound only `127.0.0.1:18082`, set migration auto-apply false, disabled Neo4j and contained no password, DSN or real service token. Production Miniapp/Admin sources had zero PostgreSQL credential references. The unified local compose contract still has exactly one Data-owned `TIDEWISE_DATABASE_URL` entry; BFF access remains HTTP-only.

## Continuous Read-Only Preflight

Immediately before generating the runtime password or executing the recovery transaction:

| Assertion | Result | Evidence |
|---|---|---|
| code/worktree | PASS | HEAD exact `7c74947b49e7d89eb5e81d2157eefa4735440b50`; worktree clean. |
| local identity | PASS | Healthy `postgres:16`; database/session role `tidewise_local`/`tidewise`; PostgreSQL major 16; server-enforced `transaction_read_only=on`. |
| complete post-DCL contract | PASS | Existing `/private/tmp/tidewise_p10_post.sql` returned `P10_POST_DCL_PASS`. |
| roles/attributes/membership | PASS | Exact three Data roles, frozen LOGIN/NOLOGIN and negative attributes, no unexpected membership. |
| owners/PUBLIC/grants | PASS | Database, 43 tables, Goose sequence and receipt function owned by migrate role; PUBLIC convergence exact; rw 18 SELECT + six INSERT only; ro 43 SELECT only; all negative grants exact. |
| ledger/schema/data | PASS | Ledger 22; `43/373/203/104/1/1`; all frozen 41-table counts exact; raw receipt rows zero. |
| migration files | PASS | Historical 21-file aggregate `2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`; `000022` `3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26`. |
| process boundary | PASS | Port 18082 had no existing listener; no Data/BFF/Agent service was started before the write. |

## Single Password Recovery Transaction

- Exactly one new 64-character hexadecimal, high-entropy `DATA_SERVICE_RW_PASSWORD` value was generated inside the single controlled shell process.
- No migrate password was generated or rotated. The existing `tidewise` credential was not altered.
- The runtime password was never printed, committed, or written into repository, `.env`, compose, config, profile, evidence, response body or service log.
- Invocation evidence: `P10_PASSWORD_TRANSACTION_INVOCATION=1`.
- The transaction set `lock_timeout=5s`, `statement_timeout=30s`, `password_encryption=scram-sha-256`, then acquired `pg_advisory_xact_lock(hashtextextended('tidewise:package10:local-data-db-role-boundary',0))`.
- Its only state-changing statement was `ALTER ROLE data_service_rw PASSWORD`, with the value read from the process-only psql variable.
- Before commit, the complete role attributes/membership, owners, PUBLIC, exact rw/ro positive and negative grants, ledger, `43/373/203/104/1/1`, 41-table counts and raw-receipt-zero assertions passed.
- Result: `P10_RW_PASSWORD_RECOVERY_COMMITTED_PASS`.
- The recovery transaction was not repeated.

## Post-Commit And Actual Login Verification

Immediately after commit:

- The `tidewise` server-enforced read-only full audit returned `P10_POST_DCL_PASS`.
- Both migration hashes were freshly reproduced and remained exact.
- A TCP connection using the new process-only password authenticated as both `current_user` and `session_user` = `data_service_rw`; the transaction was server-enforced read-only.
- Database CONNECT and schema USAGE were true; database CREATE/TEMP and schema CREATE were false.
- Direct ACL catalogs matched exactly the 18-table SELECT and six-table INSERT allowlists, with no extra table privilege.
- UPDATE, DELETE, TRUNCATE, REFERENCES and TRIGGER were false across all 43 tables; sequence USAGE/SELECT and receipt-function EXECUTE were false.
- Result: `P10_RW_LOGIN_EXACT_PRIVILEGE_PASS`.
- No DML was issued by either verification path.

## Temporary Data Service Validation

- Only one Data Service process was started, using the repo-outside loopback config and the new runtime password through process environment.
- Three fixed non-production, loopback-only identity placeholders satisfied the required agent/Miniapp/Admin authentication boundaries; no additional high-entropy credential was generated and no real service token was read or persisted.
- The bounded startup poll observed two connection-refused results before the same process opened its listener. This was ordinary pre-listen polling: the process was not restarted and no database transaction or password operation was retried.
- `/healthz`: HTTP 200.
- `/readyz`: HTTP 200, after the Data command's fail-closed existing-ledger read-only migration readiness completed.
- Side-effect-free `GET /internal/data/v1/admin/source-catalogs`: authenticated HTTP 200 with an `items` envelope.
- Production Miniapp/Admin sources still contain no PostgreSQL credential; local compose still injects a database URL only into Data. Miniapp/Admin/Agent/agent-run were not started or given database environment values.
- No migration, import, seed, projection, business write or Neo4j connection occurred during service validation.

## Cleanup And Final State

- The temporary Data process was terminated after successful validation; there was no pre-existing port-18082 process to restore.
- Port 18082 had no listener after shutdown.
- `DATA_SERVICE_RW_PASSWORD`, the local password variable and the constructed process-only DSN were unset; process exit provides the final environment boundary.
- No password value, DSN or authenticated response body was printed or added to this artifact.
- Database roles, owners and grants remain exactly those accepted after Package 10.2; ledger/schema/41-table counts/raw receipt state remain unchanged.
- Stop conditions did not trigger. No restore, retry, reverse DCL, second rotation or forward-fix was required or performed.

## Checkpoint Boundary

This checkpoint may modify only this evidence file and the task 10.3 checkbox/annotation. It does not authorize or start Package 11, Sync, Archive, Deliver or deployment.

Checkpoint validation:

- PASS: `openspec validate establish-data-service-bff-boundaries --strict`.
- PASS: explicit `OPENSPEC_TASK_LINT_CHANGE=establish-data-service-bff-boundaries` task-design lint.
- PASS: artifact status 4/4 complete and task progress 45/49, with Package 11's four tasks still pending.
- PASS: port-18082 listener cleanup plus temporary service-log/response DSN and secret scans.
- PASS: exact two-file staged manifest, diff/whitespace check and scoped secret scan.
- No production, migration, infrastructure or config file is part of the checkpoint diff.
