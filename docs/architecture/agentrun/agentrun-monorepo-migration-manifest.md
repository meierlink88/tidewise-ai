# AgentRun Monorepo Migration Manifest

## Frozen Source

- Repository: `tidewise-ai-agentrun`
- Branch: `main`
- Commit: `cba60a53e75384901debbd3d026cf86195f72154`
- Imported with preserved Git history into `agent-run/backend/`

## Tracked Asset Disposition

| Source asset | Disposition | Monorepo destination |
| --- | --- | --- |
| `api/`, `cmd/`, `configs/`, `internal/` | moved | `agent-run/backend/` |
| `Dockerfile`, `README.md` | moved | `agent-run/backend/` |
| `CONTEXT.md` | moved | `docs/contexts/agentrun/CONTEXT.md` |
| `docs/adr/*` | moved | `docs/contexts/agentrun/adr/` |
| `docs/specs/*` | moved | `docs/architecture/agentrun/` |
| `docs/research/*` | moved | `docs/research/agentrun/` |
| `docs/agents/workflow.md` | merged | `docs/agents/workflow.md` |
| `AGENTS.md`, `docs/agents/{domain,issue-tracker,triage-labels}.md` | superseded-as-duplicate | root governance documents |
| `.env.example` | merged | `infra/local/.env.example`, `infra/uat/.env.example` |
| `.gitignore`, `.dockerignore` | merged | root ignore files |
| `.github/workflows/ci.yml` | merged | `.github/workflows/ci.yml` |
| `.codex/rules/github.rules` | superseded-as-duplicate | `.codex/rules/github.rules` |
| `go.mod`, `go.sum` | merged | root `go.mod`, `go.sum` |
| package and API `testdata` | moved | remains beside its owning AgentRun packages |

Every tracked source file from the frozen commit is listed individually in
[`agentrun-monorepo-file-disposition.tsv`](./agentrun-monorepo-file-disposition.tsv). The inventory
uses only `moved`, `merged` and `superseded-as-duplicate`, and the table above is its readable
directory-level summary.
Runtime `data/`, `.env`, `.reference/`, browser output and caches were not tracked and are not
imported into Git.

## Local Reference Baseline

AgentRun development uses ignored, read-only CloudWeGo clones at the Monorepo root:

- Eino `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`
- Eino Ext `9137edd89e72b72735ede69db1c5ae29178a6e41`
- Eino Examples `171220631fb7068ead50b7cd964b8c471647117d`

These repositories are design evidence only and are never product dependencies or committed assets.
