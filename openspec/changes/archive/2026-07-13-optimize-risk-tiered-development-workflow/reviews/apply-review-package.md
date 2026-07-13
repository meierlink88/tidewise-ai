# Apply Review Package

## Scope and risk

- Change: `optimize-risk-tiered-development-workflow`
- Base proposal checkpoint: `0d1b76d`
- Apply package risk: R1. It changes workflow rules and the existing architecture rule test only; it does not perform any R2/R3 operation.
- Included files: `AGENTS.md`, `.agents/openspec-workflow.md`, `.agents/testing-tdd.md`, `.agents/git-workflow.md`, `.agents/skill-routing.md`, `backend/internal/architecture/workflow_rules_test.go`, and this change's `tasks.md` and review package.

## Before / after

| Area | Before | After |
|---|---|---|
| Risk and gate ownership | Root rules had lifecycle/stateful hard gates but no R0—R3 or package model | Root has a short R0—R3 summary; `.agents/openspec-workflow.md` is the detailed unique source |
| Ordinary tasks | Checkbox and review semantics were not separated | Checkbox is a work unit; human gate must record risk, evidence, allowed next step and non-authorization |
| R2 recovery | Backup/recovery choice had no executable workflow rule | Each named layer selects `backup` or approved disposable recovery; shared/curated/UAT/non-replaceable data require backup; invalid evidence fail-closes |
| R3 | Existing stateful safeguards were general | Production, irreversible cleanup, Neo4j rebuild and sensitive deploy require independent authorization plus recovery/DR evidence; no cross-layer batch |
| Apply validation | TDD rules unconditionally required `go test ./...` | Complete suite follows affected delivery boundary plus shared tests; repo-wide is limited to explicit triggers and fail-closes when selection is unclear |
| Active changes | General lifecycle isolation only | Deliver-time adoption is explicit, future-gate-only and requires latest `origin/main`, scoped tasks diff and one user Review |

## TDD and internal review evidence

1. RED: after adding `TestRiskTieredWorkflowRules`, `go test ./internal/architecture -run TestRiskTieredWorkflowRules -count=1` failed because `AGENTS.md` lacked `R0—R3`.
2. GREEN: after implementing the rules, the same test passed, and `go test ./internal/architecture -count=1` passed.
3. RED remediation: after expanding the test, the focused test failed because `.agents/testing-tdd.md` still contained `对应包测试通过后运行 go test ./...`.
4. GREEN remediation: replaced the unconditional test requirement with affected-boundary validation and explicit repo-wide triggers; focused and full architecture tests passed.
5. Internal code review found the unconditional validation conflict, insufficient R2/R3 assertion coverage and missing RED evidence. The conflict was removed, assertions now cover named-operation authorization, R2 pre/post assertions, backup/disposable constraints, fail-closed behavior, R3 recovery/non-batching and repo-wide trigger rules. This document preserves the RED evidence.

## Apply final verification

| Check | Result |
|---|---|
| `go test ./internal/architecture -run TestRiskTieredWorkflowRules -count=1` | Passed after GREEN and remediation |
| `go test ./internal/architecture -count=1` | Passed |
| `go test ./...` from `backend/` | Passed; required because this change modifies shared rules and architecture tests |
| `openspec validate optimize-risk-tiered-development-workflow --strict` | Passed |
| `git diff --check` | Passed |
| Scope check | Only the listed workflow rules, architecture test, current change tasks and review package changed |
| Secret scan | No credential-like pattern found in the scoped files |
| Rule links | All `.agents` files referenced by root `AGENTS.md` exist |

## Rule ownership and root size

- `AGENTS.md`: 90 -> 97 lines, 3,477 -> 3,990 characters, 6,045 -> 6,954 bytes. The 7 new lines are routing-level summary only.
- `.agents/openspec-workflow.md`: detailed R0—R3, Review package, candidate review, conditional execution, recovery evidence and adoption rules.
- `.agents/testing-tdd.md`: affected-boundary/repo-wide validation selection and test evidence only.
- `.agents/git-workflow.md`: stage-level checkpoint and adoption branch/task-diff boundaries only.
- `.agents/skill-routing.md`: package and self-review routing only.

## Non-goals and unverified items

- No business application source, migration, seed, database, Neo4j, deployment workflow, `prototype/`, `doc/` or active change artifact was changed.
- No R2/R3 operation was executed; consequently no stateful pre/post evidence is applicable to this R1 package.
- No PR, Sync, Archive or Deliver action has been performed.

## Active adoption next steps

After this rule change is Delivered, each active change must separately fetch and update latest `origin/main`, submit a scoped future-gate-only `workflow-adoption tasks diff`, and obtain one user Review. `refactor-industry-chain-node-foundation` keeps cleanup as a separate R3 package, Phase A as named R2 layers and Phase B independent. `reinitialize-alliance-economy-foundation` remains at B candidate Review; its alliance manifest remains manual, its PostgreSQL layers may later form named R2 packages, and Neo4j remains independent R3.

## Review request

This package requests Apply-after review only. Approval is required before Sync, Archive, Deliver, PR creation or any active-change adoption. It does not authorize a database, graph or deployment operation.
