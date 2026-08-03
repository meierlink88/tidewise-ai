# UAT Event Semantic Qdrant Rollout V1

Status: approved for implementation

Date: 2026-08-03

Owners: Data context, AgentRun context, UAT deployment control plane

## Outcome

Bring the UAT Event Semantic runtime to the repository-managed contract by:

- completing the active Entity Type TBox required by the accepted V3 contract;
- running Qdrant `v1.15.5` as an internal-only UAT Compose service from an immutable SWR image;
- rebuilding the two Data-owned semantic collections through an explicit, default-off deployment gate;
- replaying the already approved Industry PostgreSQL package into the UAT Neo4j projection without
  importing or changing the PostgreSQL package again.

PostgreSQL remains the fact owner. Neo4j and Qdrant remain rebuildable projections.

## Non-goals

- Copying the local PostgreSQL database into UAT.
- Copying local Event Semantic synthetic acceptance fixtures into UAT.
- Mutating Event, RawDocument, Theme, Reason Tree, or Industry relationship package facts.
- Incremental Qdrant synchronization, CDC, scheduled rebuilds, public Qdrant ports, HA, or backups.
- Giving AgentRun access to the Data PostgreSQL database or giving the long-running Data Service
  access to the embedding secret.
- Automatically merging the implementation PR or bypassing the RDS recovery-point gate.

## Owner and dependency map

| Contract | Provider/owner | Consumer | Boundary |
| --- | --- | --- | --- |
| Entity and Variable Definition facts | Data PostgreSQL | Data projector | Existing Data database operation configuration |
| Semantic collection writes | Data projector | Qdrant | One-shot internal HTTP operation |
| Semantic collection reads | Qdrant | AgentRun | Existing typed AgentRun retrieval adapter |
| Industry graph facts | Data PostgreSQL frozen package | Data graph projector | Existing one-shot projector |
| Industry graph view | host-managed Neo4j | Data graph projector | Scoped Bolt credentials only during projection |
| Deployment state | UAT control plane | Docker Compose | Runtime/image env plus current/previous release snapshots |

No API or cross-service wire DTO changes are introduced.

## PostgreSQL contract

UAT currently has the twelve Entity Type definitions introduced by migration `000032`. The accepted
V3 catalog and local production baseline also contain `economy`, `index`, `instrument`, and `market`.
Migration `000040` is a forward-only deterministic upsert of those four authored definitions. It does
not import ABox rows and does not edit migrations already used by a shared environment.

The UAT audit established that the production ABox business fields already match local after excluding
the three local synthetic acceptance entities and their dependent fixture rows. Therefore the rollout
performs checksum/count verification only; it does not run a local-to-UAT data copy.

## Qdrant runtime contract

- Image: SWR mirror of `qdrant/qdrant:v1.15.5`, consumed as
  `repository:v1.15.5@sha256:<digest>`.
- Upstream image digest audited for the rollout:
  `sha256:0fb8897412abc81d1c0430a899b9a81eb8328aa634e7242d1bc804c1fe8fe863`.
- Network: `tidewise-uat`, service identity `qdrant`.
- Ports: `6333` and `6334` are exposed only to the internal Docker network; no host port is published.
- Storage: external-name-compatible Docker volume `tidewise-uat-qdrant-data` mounted at
  `/qdrant/storage`.
- Lifecycle: `restart: unless-stopped`, bounded health check, and the existing UAT log rotation policy.

The first repository-managed release reuses the existing named volume. Immediately before that release,
the manually created `tidewise-uat-qdrant` container must be stopped and removed without deleting the
volume. This is a one-time handoff, not a recurring CD operation.

## Projection invocation and failure contract

`Deploy UAT` adds the independent, default-off input `apply_event_semantic_projection`.

When enabled:

1. Data and AgentRun migrations complete successfully.
2. Any requested Industry graph projection completes and verifies replay.
3. Compose starts only the candidate Qdrant service and waits for health.
4. The candidate Data image runs `/usr/local/bin/event-semantic-projector -apply -allow-env uat` with:
   - `QDRANT_URL=http://qdrant:6333`;
   - the frozen DashScope-compatible embedding base URL;
   - `EMBEDDING_API_KEY` passed by environment name from the UAT Secret.
5. The deployment executor verifies the projector report and Qdrant collection metadata before it
   starts AgentRun or any public service.

The frozen first-rollout result is:

| Collection | Points | Vector contract |
| --- | ---: | --- |
| `entity_semantic_v1` | 4,973 | 1,024 dimensions, Cosine |
| `variable_definition_semantic_v1` | 12 | 1,024 dimensions, Cosine |

The local entity collection has three additional synthetic acceptance points and is intentionally not
the UAT acceptance count.

The current projector replaces each formal collection. AgentRun must therefore remain unavailable during
the rebuild. A failed rebuild blocks the candidate release; it is never treated as degraded success. The
Qdrant volume is retained so an operator can correct the failure and rerun the explicit projection.

When projection is disabled, the deployment still starts repository-managed Qdrant but does not mutate
collections. This mode is only valid after a successful initial projection.

## Secrets and configuration

- `EMBEDDING_API_KEY` is a GitHub `uat` Environment Secret.
- AgentRun receives it through the persisted `runtime.env` because retrieval requires it at startup.
- The one-shot Data projector receives it only through `docker compose run -e EMBEDDING_API_KEY`.
- The long-running Data Service and other containers do not receive it.
- Qdrant has no host exposure and does not use an API key in this UAT topology.
- `SWR_QDRANT_IMAGE` is a GitHub `uat` Environment Variable containing the full immutable SWR image
  reference; the workflow rejects a tag-only value.

Secrets must not appear in Compose, repository configuration, command output, deployment summaries,
diagnostics, or projection payloads.

## Rollout and rollback

Rollout order:

1. Confirm an RDS recovery point/PITR and pause Event Semantic scheduling.
2. Remove only the legacy manual Qdrant container, retaining `tidewise-uat-qdrant-data`.
3. Dispatch the merged `main` release with high-risk backup confirmation, Industry graph projection, and
   Event Semantic projection enabled.
4. Apply Data migrations through `000040` and the target AgentRun migration chain.
5. Project/replay Neo4j from the already present frozen Industry package.
6. Rebuild/verify Qdrant.
7. Start the complete release and pass container and host verification.
8. Verify excluded PostgreSQL fact row counts and schema-normalized full-row fact fingerprints are
   unchanged before resuming Event Semantic scheduling. The normalization accounts only for the approved
   migration `000035`/`000038` representation changes; all surviving fact fields remain compared.

Application rollback restores the previous repository-managed application release and does not run down
migrations. Migration `000040` is compatible with the previous application because it only adds complete
catalog rows to an existing table. Qdrant projection failure occurs before AgentRun/public candidate
services start. The named volume is never deleted by deployment rollback.

## Acceptance seams

- Migration/schema test proves the four definitions and authored fields after the full forward chain.
- Repository contract test proves immutable Qdrant image handling, internal-only ports, scoped secret
  injection, explicit projection input, and release-state coverage.
- Deployment executor fixture proves the semantic projector runs only when enabled, before AgentRun starts,
  rejects missing secrets and result drift, and does not print the embedding key.
- Compose config validation proves the release file resolves with the documented environment contract.
- UAT operational acceptance proves PostgreSQL migration 40, sixteen active TBox definitions, Neo4j
  4,449/7,867 with unchanged replay, Qdrant 4,973/12 with green 1,024/Cosine collections, healthy services,
  and unchanged Event/RawDocument/Theme/Reason Tree row counts and schema-normalized content fingerprints.
