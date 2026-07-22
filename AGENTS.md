## Agent skills

### Delivery workflow

Product changes use the repository workflow in `docs/agents/workflow.md`.

- The default lifecycle is `grill-with-docs -> to-spec -> implement`.
- OpenSpec is retired and must not be created, restored, validated, synced, or archived.
- The current task executes the work directly by default. Create or delegate another Codex task only when the user explicitly requests delegation.
- The user controls pull-request merge.

### Issue tracker

Issues and PRDs are tracked in this repository's GitHub Issues. See `docs/agents/issue-tracker.md`.

### GitHub CLI

- Run `gh` as a direct command. Do not wrap it in `bash -lc`, `zsh -lc`, command chains, substitutions, or inline multiline bodies.
- Write multiline Issue and PR bodies to a temporary file, then pass it with `--body-file` so `.codex/rules/github.rules` can match the command and allow macOS Keychain access.

### Triage labels

Use the five canonical triage labels defined for this repository. See `docs/agents/triage-labels.md`.

### Domain docs

This repository uses a single-context domain documentation layout. See `docs/agents/domain.md`.
