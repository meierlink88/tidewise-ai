---
status: accepted
date: 2026-09-03
issue: 391
supersedes_in_part: 0029, 0041
---

# Retire the migrated UAT AgentOS and reasoning runtime

## Context

AgentOS and its reasoning infrastructure have moved out of the Tidewise AI Huawei Cloud UAT ECS
and RDS. The four Tidewise applications now consume only the Data PostgreSQL facts and do not read
AgentOS, Reason/OpenSPG, KAG, MySQL, Neo4j or Qdrant. A sanitized live audit on 2026-09-03 still
found those retired containers and persistent volumes running on the old ECS.

MinIO is different: Admin raw-evidence links still resolve through
`https://tideai.tripwise.cn/raw-evidence/`, backed by the UAT MinIO volume. Removing it would break
retained evidence-document reads, so it remains a separate storage lifecycle.

## Decision

- The UAT application baseline remains Data Service, Miniapp Backend, Admin Backend and Admin Web.
- The independent `tidewise-infra-uat` baseline becomes MinIO/raw-evidence only.
- Remove the exact audited AgentOS, Reason/OpenSPG Server, Qdrant, OpenSPG MySQL and OpenSPG Neo4j
  containers from the old ECS.
- Delete only their exact audited volumes and bounded host paths after a main-only, CI-gated,
  confirmation-protected preflight proves that retained services have no legacy dependency keys.
- Stop and remove the obsolete Tidewise Reason Actions runner service and disabled host Neo4j unit.
- The daily deployment runner remains non-sudo. A checksum-verified static retirement binary runs as
  root only inside a one-shot privileged container based on the already-running immutable Data image.
  The host root is mounted read-only; only `/etc/systemd/system` and `/opt/tidewise` are writable. The
  binary accepts only `preflight` or `apply` and compiles in the two approved units and five approved
  directories rather than accepting paths or commands from workflow input.
- The retired RDS database `tidewise_ai_server` is already absent. Preserve `tidewise_uat`, its role,
  schema and facts. A residual `agentrun_uat` role is not a database and may be removed only through
  a separately authorized RDS-admin operation after dependency proof.
- Remove active UAT provisioning contracts that could recreate MySQL or Neo4j. Keep historical ADRs
  as decision history and retain container images as pullable recovery artifacts.

## Safety and verification

The retirement workflow acquires the shared deployment lock, validates exact runner identity and an
exact confirmation phrase, checks candidate mounts, verifies Data migration 81 with no pending
migration, fingerprints retained containers, and refuses broad Compose shutdown, prune, wildcard or
database/schema deletion. It removes exact targets only and then proves all retired containers,
volumes, paths and listeners absent while retained fingerprints and public health/read paths remain
unchanged.

The privileged retirement binary performs its own systemd fragment and no-symlink path preflight before
the shell removes any legacy container. It stops only the compiled units, reloads systemd, removes only
the compiled paths, and verifies both classes absent. This uses the root-equivalent Docker capability
already required by the runner instead of granting persistent sudo or changing host sudoers.

Persistent data deletion is intentionally irreversible on the old ECS. The accepted recovery path is
the externally migrated AgentOS/reasoning system; the existing RDS recovery point protects the
retained Data database but is not used to restore retired middleware state.

## Consequences

Future Tidewise UAT application or MinIO deployments cannot recreate OpenSPG MySQL, Neo4j or Qdrant.
AgentOS publishes and reads through versioned Data APIs from its external runtime. MinIO remains only
for the current raw-evidence archive boundary until a separate storage migration decision replaces
that URL and data.
