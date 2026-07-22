# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues. Use the `gh` CLI for all operations.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`
- **Read an issue**: `gh issue view <number> --comments`
- **List issues**: use `gh issue list` with appropriate state and label filters
- **Comment**: `gh issue comment <number> --body "..."`
- **Apply/remove labels**: use `gh issue edit`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repository from `git remote -v`.

## Pull requests as a triage surface

**PRs as a request surface: no.**

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.

## Wayfinding operations

- A map is a GitHub issue labelled `wayfinder:map`.
- Child tickets use GitHub sub-issues where available.
- Child labels use `wayfinder:research`, `wayfinder:prototype`,
  `wayfinder:grilling`, or `wayfinder:task`.
- Use native GitHub issue dependencies for blocking relationships.
- Claim work with `gh issue edit <number> --add-assignee @me`.
- Resolve by commenting with the result, closing the issue, and recording the
  context pointer in the map's Decisions-so-far section.
