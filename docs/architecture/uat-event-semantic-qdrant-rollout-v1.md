# UAT Event Semantic Qdrant Rollout V1

Status: approved for implementation

Date: 2026-08-03

Owners: Data context, AgentRun context, UAT deployment control plane

## Outcome

Bring the UAT Event Semantic runtime to the repository-managed contract by:

- completing the active Entity Type TBox required by the accepted V3 contract;
- using the independently operated, internal-only UAT Qdrant runtime without making it part of the
  application release lifecycle;
- rebuilding the two Data-owned semantic collections through an explicit, default-off deployment gate;
- replaying the already approved Industry PostgreSQL package into the UAT Neo4j projection without
  importing or changing the PostgreSQL package again.

PostgreSQL remains the fact owner. Neo4j and Qdrant remain rebuildable projections.

## Non-goals

- Copying the local PostgreSQL database into UAT.
- Copying local Event Semantic synthetic acceptance fixtures into UAT.
- Mutating Event, RawDocument, Theme, Reason Tree, or Industry relationship package facts.
- Incremental Qdrant synchronization, CDC, scheduled rebuilds, public Qdrant ports, HA, or backups.
- Installing, upgrading, restarting, removing, mirroring, or rolling back PostgreSQL, Neo4j, or Qdrant.
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

Qdrant is independently operated infrastructure, matching the ownership model of RDS PostgreSQL and
host-managed Neo4j. The application Compose/CD contract does not own its image, container, version,
restart policy, volume, installation, upgrade, rollback, or removal.

The separately maintained UAT runtime must provide:

- Docker network: `tidewise-uat`, stable network alias `qdrant`;
- internal endpoint: `http://qdrant:6333`;
- no host-published HTTP or gRPC port;
- persistent storage and lifecycle managed by the infrastructure operator.

The currently audited runtime is `qdrant/qdrant:v1.15.5` with container
`tidewise-uat-qdrant` and named volume `tidewise-uat-qdrant-data`. Those identities are operational
evidence, not application release inputs. A Qdrant version change is a separate maintenance action.

## Projection invocation and failure contract

`Deploy UAT` adds the independent, default-off input `apply_event_semantic_projection`.

When enabled:

1. Data and AgentRun migrations complete successfully.
2. Any requested Industry graph projection completes and verifies replay.
3. The candidate Data image proves the independently operated Qdrant endpoint is reachable from the
   `tidewise-uat` network.
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

Every deployment requires the external Qdrant endpoint to be healthy before database writes. When
projection is disabled, CD does not mutate collections or the Qdrant runtime. This mode is only valid
after a successful initial projection.

## Secrets and configuration

- `EMBEDDING_API_KEY` is a GitHub `uat` Environment Secret.
- AgentRun receives it through the persisted `runtime.env` because retrieval requires it at startup.
- The one-shot Data projector receives it only through `docker compose run -e EMBEDDING_API_KEY`.
- The long-running Data Service and other containers do not receive it.
- Qdrant has no host exposure and does not use an API key in this UAT topology.
- No Qdrant image or infrastructure credential is stored in the application release Environment.

Secrets must not appear in Compose, repository configuration, command output, deployment summaries,
diagnostics, or projection payloads.

## Rollout and rollback

Rollout order:

1. Confirm an RDS recovery point/PITR and pause Event Semantic scheduling.
2. Verify the independently operated Qdrant endpoint and leave its container and volume unchanged.
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
services start. Application rollback does not restart, replace, remove, or otherwise mutate the Qdrant
runtime or storage.

## Acceptance seams

- Migration/schema test proves the four definitions and authored fields after the full forward chain.
- Repository contract test proves Qdrant is absent from Compose/release state while external dependency
  probing, scoped secret injection, and the explicit projection input remain enforced.
- Deployment executor fixture proves the semantic projector runs only when enabled, before AgentRun starts,
  rejects missing secrets and result drift, and does not print the embedding key.
- Compose config validation proves the release file resolves with the documented environment contract.
- UAT operational acceptance proves PostgreSQL migration 40, sixteen active TBox definitions, Neo4j
  4,449/7,867 with unchanged replay, Qdrant 4,973/12 with green 1,024/Cosine collections, healthy services,
  and unchanged Event/RawDocument/Theme/Reason Tree row counts and schema-normalized content fingerprints.
