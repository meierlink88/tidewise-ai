# Product delivery workflow

This repository uses a document-and-ticket workflow. OpenSpec is retired.

## Default ownership

The current Codex task owns exploration, specification, implementation, verification, commit, and pull-request delivery. Do not create a separate task, subagent, feature worktree, or delegation merely because a change is large. Delegate only when the user explicitly asks for it.

When delegation is explicitly requested, use `gpt-5.6-sol` with `medium` reasoning unless the user selects another model. The delegated task still follows this document and does not use OpenSpec.

## Lifecycle

### 1. Explore and preserve

- Read `AGENTS.md`, `CONTEXT.md`, relevant ADRs, authoritative specs, and the affected implementation.
- Inspect the current branch, HEAD, remote baseline, and dirty/untracked files before writing.
- Treat existing changes as user-owned. Preserve them and keep unrelated files out of the task commit.
- For Eino changes, inspect the shared read-only references under the main checkout's `.reference/cloudwego/` when available. A worktree-local clone is not required.

### 2. Grill with docs

Use `$grill-with-docs` for new capabilities, architecture, domain decisions, or materially ambiguous behavior.

- Resolve contradictions, missing ownership, state transitions, failure semantics, security boundaries, and acceptance criteria before implementation.
- Write resolved decisions into the authoritative spec, glossary, and sparse ADRs where appropriate; chat summaries are not sufficient.
- If the user has already frozen the decisions and the authoritative documents capture them, record that the grill is complete instead of repeating the interview.
- Small workflow, documentation, wording, or mechanical maintenance changes may use the direct path and skip grilling.

### 3. Publish with to-spec

Use `$to-spec` after the design is sufficiently resolved.

- Prefer one high, observable test seam. Confirm it with the user unless their frozen acceptance criteria already establish it.
- Create or update a GitHub Issue using the repository spec template and apply only the `ready-for-agent` triage label.
- The Issue is the implementation ticket and PR traceability anchor. Do not create an OpenSpec change or parallel local planning format.
- If GitHub is temporarily unavailable, keep the reviewed authoritative spec as the source of truth, report the delivery blocker, and do not invent a substitute Issue identifier.

### 4. Implement

Use `$implement` against the reviewed spec and Issue.

- Work directly in the current checkout on a `codex/<issue-or-change-name>` feature branch unless the user explicitly requests another arrangement.
- Use `$tdd` where practical at the pre-agreed seams: establish a real failing test, add the minimum implementation, then refactor.
- Run focused tests and typechecking regularly, then the complete relevant test suite once at the end.
- Use `$code-review` after implementation. Resolve actionable Standards and Spec findings before delivery.
- Verify diff scope, formatting, credentials, generated/runtime artifacts, and migration safety as applicable.
- Commit only task-owned files to the current branch.

### 5. Deliver and merge

- When the requested outcome includes delivery, push the feature branch and create a draft PR linked to the Issue. Keep multiline PR text in a temporary body file.
- Do not merge the PR. The user controls merge.
- After the user confirms merge, fetch `origin/main`, verify the commit is present, return the main checkout to a clean updated state, and remove obsolete local/remote feature branches when authorized.

## Direct path

Repository workflow files, skills, documentation-only corrections, tiny configuration adjustments, and explicit runtime verification may be changed directly without `$to-spec` when they do not add or alter product capability. They still require scope review, secret safety, and an intentional commit/PR when delivery is requested.

## Forbidden legacy flow

Do not run or recreate any of the following unless the user explicitly reverses this repository decision:

- `openspec new change`
- OpenSpec proposal/design/tasks artifacts
- OpenSpec Apply/Validate/Sync/Archive lifecycle
- mandatory Leader-to-executor delegation
- mandatory independent worktree for every product change
