---
status: accepted
date: 2026-08-18
issue: 275
---

# Retire AgentRun from Tidewise AI

## Decision

AgentRun is fully retired from this repository because Agent OS has replaced it. The retirement is
zero-compatibility and has no AgentRun rollback path.

- Delete the application source, tests, configuration, Context documentation, database migrations,
  local Artifact data and application-owned database/role.
- Remove Admin Portal schedules, execution monitoring, model configuration, connector configuration,
  status and downstream proxy APIs. Keep the Data-backed raw-document and Event lists.
- Remove local/UAT Compose services, one-shot commands, images, secrets, Artifact mounts, CI jobs and
  deployment Action branches owned by the retired application.
- The current application release unit contains Data Service, Miniapp Backend, Admin Backend and
  Admin Web.
- Agent OS is external to this repository and can collaborate only through versioned Data APIs.

## Infrastructure boundary

Shared PostgreSQL/RDS engines and shared Qdrant infrastructure are not application-owned and remain
available to other systems. Retirement may remove only the application database, role and Artifact
directory; shared Qdrant and all of its collections remain physically untouched.

## Consequences

Historical ADRs remain as decision history but directly replaced decisions are marked superseded.
Current Context docs, API contracts, deployment scripts and repository tests describe only the
four-service baseline. Any future Agent OS integration requires a new explicit API and ownership
decision; retired routes, state and rollback machinery are not compatibility surfaces.
