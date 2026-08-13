# Product delivery workflow

This repository uses a standards-and-ticket workflow. OpenSpec and repository Architecture Specs are retired.

## Default ownership

The current Codex task owns exploration, specification, implementation, verification, commit, and pull-request delivery. Do not create a separate task, subagent, feature worktree, or delegation merely because a change is large. Delegate only when the user explicitly asks for it.

When delegation is explicitly requested, use `gpt-5.6-sol` with `medium` reasoning unless the user selects another model. The delegated task still follows this document and does not use OpenSpec.

## Lifecycle

### 1. Explore and preserve

- Use `$ganchaojia-development-standard` to discover the applicable repository standards, then read
  `CONTEXT-MAP.md`, the affected Context and relevant ADRs. Inspect executable contracts and current
  implementation only when the task touches them.
- Inspect the current branch, HEAD, remote baseline, and dirty/untracked files before writing.
- Treat existing changes as user-owned. Preserve them and keep unrelated files out of the task commit.

### Stack reference gates

When the routed stack standard requires reference-first work, complete that audit before design or
implementation and record the required evidence in the GitHub Issue or PR. The stack standard
owns source locations and evidence fields; this workflow does not duplicate them.

### 2. Grill with docs

For new capabilities, architecture, domain decisions, or materially ambiguous behavior, complete
the design gate in `engineering-standard.md`. Use `$grill-with-docs` when it is available; the Skill
assists the gate but is not repository authority.

- Resolve contradictions, missing ownership, state transitions, failure semantics, security boundaries, and acceptance criteria before implementation.
- Write resolved acceptance criteria into the GitHub Issue, and update Context or a sparse ADR when
  the decision changes domain language or an architectural boundary; chat summaries are not sufficient.
- If the user has already frozen the decisions and the Issue, Context or ADR captures them, record
  that the grill is complete instead of repeating the interview.
- Small workflow, documentation, wording, or mechanical maintenance changes may use the direct path and skip grilling.

### 3. Publish the implementation ticket

After the design is sufficiently resolved, publish the reviewed outcome, boundaries, acceptance
criteria and testing fields directly to the GitHub Issue; do not create a repository Architecture Spec.

- Prefer one high, observable test seam. Confirm it with the user unless their frozen acceptance criteria already establish it.
- Create or update a GitHub Issue using the repository change template and apply only the `ready-for-agent` triage label.
- Create the Issue automatically without requesting separate user approval. `gh issue create` may
  run outside the workspace sandbox to read the macOS Keychain.
- The Issue is the implementation ticket and PR traceability anchor. Do not create an OpenSpec change or parallel local planning format.
- If GitHub is temporarily unavailable, report the delivery blocker and do not invent a substitute
  Issue identifier or local Spec.

### 4. Implement

Implement against the reviewed Issue acceptance criteria. Use `$implement` when it is available; otherwise
follow the same repository standards directly.

- Work directly in the current checkout on a `codex/<issue-or-change-name>` feature branch unless the user explicitly requests another arrangement.
- Use `$tdd` where practical at the pre-agreed seams: establish a real failing test, add the minimum implementation, then refactor.
- Run focused tests and typechecking regularly, then the complete relevant test suite once at the end.
- Use `$code-review` after implementation. Resolve actionable Standards and Issue findings before delivery.
- Verify diff scope, formatting, credentials, generated/runtime artifacts, and migration safety as applicable.
- Commit only task-owned files to the current branch.

### 5. Deliver and merge

- After implementation, verification, and code review pass, automatically push the feature branch
  and create a ready-for-review PR linked to the Issue. Do not request separate user approval for
  `gh pr create`; it may run outside the workspace sandbox to read the macOS Keychain.
- Use a draft PR only when the user explicitly asks for an early review or the implementation is
  intentionally incomplete. Keep multiline PR text in a temporary body file.
- Do not merge the PR. The user controls merge.
- After the user confirms merge, fetch `origin/main`, verify the commit is present, return the main checkout to a clean updated state, and remove obsolete local/remote feature branches when authorized.

## Direct path

Repository workflow files, skills, development standards, documentation-only corrections, tiny
configuration adjustments, and explicit runtime verification may use the direct path when they do
not add or alter product capability. Development-standard content changes
require format, reference, and diff review only; they do not trigger product tests or CI gates.

## Forbidden legacy flow

Do not run or recreate any of the following unless the user explicitly reverses this repository decision:

- `openspec new change`
- OpenSpec proposal/design/tasks artifacts
- OpenSpec Apply/Validate/Sync/Archive lifecycle
- mandatory Leader-to-executor delegation
- mandatory independent worktree for every product change
