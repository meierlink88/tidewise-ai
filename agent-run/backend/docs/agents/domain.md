# Domain Docs

This repository uses the single-context layout.

## Before exploring, read these

- `CONTEXT.md` at the repository root, when it exists.
- Relevant ADRs under `docs/adr/`, when they exist.

Missing domain documents are not errors. Proceed silently; domain-modeling
creates them lazily when terminology or architectural decisions are resolved.

## Layout

```text
/
├── CONTEXT.md
├── docs/
│   └── adr/
└── internal/
```

## Vocabulary

Use domain terms as defined in `CONTEXT.md`. Avoid synonyms that the glossary
explicitly rejects. If a required concept is absent, reconsider the terminology
or record the gap for domain modeling.

## ADR conflicts

If proposed work contradicts an existing ADR, surface the conflict explicitly
instead of silently overriding the decision.
