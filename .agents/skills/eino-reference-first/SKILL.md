---
name: eino-reference-first
description: Audit the local cloudwego/eino-ext, cloudwego/eino-examples, and cloudwego/eino repositories before producing solutions or changes. Use before proposing or making any change in this repository, including source code, tests, configuration, documentation, prompts, skills, scripts, dependencies, generated artifacts, project structure, reviews, and debugging fixes, whether or not the task initially appears to require Eino.
---

# Eino Reference First

Before proposing or making any change, use the three local upstream clones as
the source of truth. Audit all task-relevant code and documentation in each
repository; do not claim to have read every unrelated file in all three
repositories.

## Mandatory workflow

### 1. Verify the references

From the project root, run:

```bash
python3 .agents/skills/eino-reference-first/scripts/check_references.py --root .
```

Stop and report the missing reference if any clone is unavailable. Record the
reported absolute `path` and commit ID for each repository. The checker resolves
the reference root in this order: explicit `--reference-root`, shared Git config
`tidewise.referenceRoot`, the main checkout inferred from Git's common directory,
then the current checkout. A linked worktree must reuse the main checkout's
read-only clones without creating per-worktree symlinks. Never modify or update a
clone as part of an implementation task unless the user explicitly asks to
refresh it.

For an uncommon repository layout where Git common-directory inference is not
available, configure one absolute shared path once from any worktree:

```bash
git config tidewise.referenceRoot /absolute/path/to/.reference/cloudwego
```

### 2. Search `eino-ext` first

Translate the task into concrete capability keywords, package names, and Go
interfaces. Use `rg` and `rg --files` under the `eino-ext` absolute path reported
by the checker to find an existing component before designing custom
infrastructure. Do not assume the clone is inside the current worktree.

Read the complete task-relevant component directory: its README, exported API,
configuration, constructors, tests, and dependency declarations when they
affect behavior. Prefer an `eino-ext` component when it meets the requirement.

### 3. Find the official pattern in `eino-examples`

Search the checker-reported absolute `eino-examples` path for the nearest
working example.
Read the complete relevant example flow, including initialization, composition,
callbacks, streaming, state, error handling, and configuration files that the
example uses. Treat examples as implementation guidance, not as production code
to copy without adapting boundaries, errors, configuration, and observability.

### 4. Confirm semantics in `eino`

Search the checker-reported absolute `eino` path for every core interface and
runtime behavior the proposal depends on. Read the public API, implementation, comments, and
focused tests needed to understand graph/chain compilation, component contracts,
callbacks, streaming, state, interrupts, checkpointing, or concurrency.

Do not infer core behavior solely from examples.

### 5. Decide and implement

Apply this priority:

1. Configure or compose an existing `eino-ext` component.
2. Adapt a demonstrated `eino-examples` pattern using stable `eino` APIs.
3. Build a thin project adapter around those interfaces.
4. Implement a new component only when the audit proves a real capability gap.

Keep product-specific prompts, skills, domain rules, and storage contracts in
the project. Keep provider SDK details behind adapters. Do not copy or edit files
inside `.reference/cloudwego/`.

### 6. Verify and report

Run tests appropriate to the changed scope. End every response that proposes or
makes a change with a concise `Eino reference audit` containing:

- `eino-ext`: components inspected and reuse decision.
- `eino-examples`: example paths inspected and adopted pattern.
- `eino`: core packages or interfaces inspected and semantic conclusion.
- `gap`: any custom implementation and why upstream did not cover it.
- `commits`: the three commit IDs from the reference check.

If no relevant match exists in a repository, state the search terms and paths
checked instead of silently omitting that repository.
