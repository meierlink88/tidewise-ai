---
status: accepted
superseded_in_part: 0027-retire-agent-run.md
---

# Separate local Docker application and infrastructure projects

Local Docker resources use two explicit Compose projects connected by the `tidewise-local`
network. `tidewise-app` groups deployable application containers while preserving repository-owned
lifecycle boundaries. `tidewise-infra` owns the long-lived PostgreSQL, MySQL, Neo4j, MinIO and
Qdrant middleware containers and their persistent volumes.

The Tidewise AI repository owns the local orchestration entrypoints. Its application Compose file
starts only Admin Web, Admin Backend, Miniapp Backend, Data and AgentRun. Reason Server and Agent OS
keep repository-local Compose definitions and join `tidewise-app` only when explicitly started from
their owning repositories. Sharing a Compose project label does not authorize one repository to
start, stop or remove another repository's service.

Infrastructure is independently operated and low-frequency. Application startup may ensure the
existing middleware containers are running with `--no-recreate`, but normal application shutdown
does not stop infrastructure. Infrastructure shutdown and upgrades require explicit commands.
Subprojects use service-scoped lifecycle commands and must never run unscoped `docker compose down`
or `--remove-orphans` against `tidewise-app`.

The `tidewise-local` network is created by `tidewise-infra` and consumed as external by every
application Compose file. Service DNS names are stable on that network. A shared PostgreSQL engine
does not change data ownership: Data, AgentRun and Agent OS keep independent databases, roles,
migrations and credentials.

The local infrastructure Compose file is a reproducible developer-machine bootstrap artifact, not
an application release artifact and not a UAT infrastructure owner. Existing middleware volume
names remain explicit so regrouping containers cannot initialize empty replacement stores.
