# Docker-only Runtime V1

## Outcome

All Tidewise deployable application services and service-owned operational processes have one
supported runtime: service-owned images orchestrated by Docker Compose. Local service development,
UAT deployment and one-shot operational commands use the same image entrypoints and container
network semantics. Miniapp Frontend build/watch is development and publishing tooling, not a
deployable application process.

## Scope

| Process                                  | Docker contract                                                                        |
| ---------------------------------------- | -------------------------------------------------------------------------------------- |
| Data Domain Service                      | `data` image and Compose service                                                       |
| Data migrations and tools                | binaries in the Data image, invoked with `docker compose run --rm`                     |
| AgentRun                                 | `agentrun` image and Compose service                                                   |
| AgentRun migration/config/Artifact tools | binaries in the AgentRun image, invoked with Compose run/exec                          |
| Miniapp Application Backend              | `miniapp` image and Compose service                                                    |
| Admin Application Backend                | `adminportal` image and Compose service                                                |
| Admin Portal Frontend                    | unprivileged nginx `admin` image and Compose service                                   |
| Miniapp Frontend                         | non-service repository-pinned Node/Taro build; excluded from application Compose       |
| PostgreSQL, Neo4j and Qdrant             | external infrastructure endpoints; excluded from application Compose/release artifacts |

Tests, lint, typecheck and source compilation may still execute directly in CI or a developer tool.
Miniapp Frontend build/watch also runs directly because it produces platform artifacts rather than
hosting a Tidewise service. These are tooling contracts, not deployable application runtime
entrypoints.

## Non-goals

- No API, authentication, domain, database schema or ownership change.
- No replacement of Kratos, Taro, Vite, nginx, PostgreSQL, Neo4j or Qdrant.
- No Kubernetes, service discovery, remote configuration center or new deployment platform.
- No ownership, packaging or lifecycle management of PostgreSQL, Neo4j or Qdrant by any application Compose release.
- No attempt to run WeChat or Douyin desktop developer tools inside Docker.
- No change to Miniapp preview, upload, review or platform publishing.
- No removal of service binaries or native test/build commands from CI.

## Runtime contract

### Configuration

- Every Backend owns `configs/config.<environment>.yaml` only.
- Data `local` and AgentRun `dev` use `host.docker.internal` defaults for external infrastructure;
  environment endpoint overrides support other infrastructure-owned addresses. Tidewise services
  still use application Compose DNS such as `data`.
- `configs/compose/` is removed; host-specific YAML is not retained.
- Every Backend image copies its YAML directory to `/app/configs` and defaults its config-directory
  environment variable to `/app/configs`.
- Secret values remain in ignored local environment files or the approved UAT secret source.
- Environment variables may keep their existing approved endpoint overrides; they do not recreate a
  second checked-in host-runtime contract.

### Local Compose

- A normal `up` starts migrations before long-running services and includes Admin Portal Frontend.
- Stable local application browser/host ports remain Data `9011`, Miniapp `9012`, Admin Backend
  `9013`, Admin Web `9014` and AgentRun `9080`. UAT publishes Miniapp `9012` and Admin Web `9014`,
  but keeps Admin Backend `9013` internal. Infrastructure ports are outside this contract.
- Backend-to-backend traffic uses Compose DNS; browser and Miniapp build output use mapped host URLs.
- Only the service-owned AgentRun Artifact volume is declared. Infrastructure data volumes are not
  part of the application package. Documentation must never recommend `docker compose down -v` as
  a normal reset or rollback operation.

### Frontend processes

- Admin local runtime uses the same nginx image shape as UAT. The browser calls relative
  `/api/admin/*`; nginx proxies to `http://adminportal:9013` on the Compose network.
- Miniapp H5/weapp/tt development and CI use the repository-pinned Node and Taro dependencies
  directly. Platform developer tools read the generated host `dist` output.
- Miniapp mock mode runs without Backend dependencies; API mode expects the Miniapp Backend to be
  started separately.
- Taro `outputRoot=dist/<platform>` and current API/mock selection remain unchanged.

### Operational commands

- Data and AgentRun Dockerfiles contain every supported service-owned CLI binary referenced by
  runtime documentation.
- Local migrations run as Compose one-shot services before Data or AgentRun starts.
- Seed, reset, projector, audit, config and Artifact operations use the relevant service image and
  the same YAML/network/secret contract as the service.
- Destructive commands retain their existing explicit confirmation and environment/database guards;
  local guards recognize only the approved container-to-infrastructure target in addition to the
  local environment and database identity.

## Failure and security

- A failed migration prevents its dependent service from starting.
- A missing required secret or invalid YAML fails the owning container at startup.
- Health/readiness endpoints remain the startup and dependency gates.
- BFF and frontend containers never receive Data/AgentRun database credentials.
- Admin browser runtime receives no Backend origin or downstream service token; the Admin Web
  container owns the fixed internal Backend route.
- Miniapp build output contains only the existing public Miniapp Backend URL, never a downstream
  service token.

## Rollout and compatibility

1. Add the canonical image/Compose paths and container contract tests.
2. Move local/development YAML to container-reachable infrastructure endpoints and remove duplicate YAML trees.
3. Replace documented and root-script native runtime commands with Compose commands.
4. Validate local Compose resolution, build images, connect to externally provisioned infrastructure,
   run migrations, start services and check health.
5. Deploy UAT images and Compose config together where `/app/configs` path changes.

There is no schema migration. Rollback restores the previous images and Compose/YAML files while
leaving named volumes intact. Native runtime and Docker-only runtime are not maintained as a
mixed-version compatibility window.

## Acceptance seams

- Repository contract: exactly one YAML per Backend environment; no `configs/compose`; runtime docs
  and root dev scripts contain no host-native service commands.
- Compose contract: Local and UAT resolve with example env files; all application images, config
  paths, dependency gates, ports, secrets and healthchecks are present.
- Container build: the five deployable application images build successfully; no Miniapp Frontend
  image exists.
- Runtime smoke: Data and AgentRun migrations complete; Data, AgentRun, Miniapp, Admin Backend and
  Admin Web reach their existing health/readiness contracts.
- Miniapp build seam: direct CI weapp and tt builds write their existing `dist/<platform>` output
  without changing frontend behavior.

## Reference evidence

- Taro version: repository Taro `4.2.0` platform plugins with React 18.
- Selected references: Taro 4.x [Installation and Usage](https://docs.taro.zone/docs/GETTING-STARTED),
  [Compile Configuration](https://docs.taro.zone/docs/config/) and
  [`outputRoot`](https://docs.taro.zone/docs/config-detail/#outputroot) contracts.
- Adopted: the documented `taro build --type <platform> [--watch]` flow and configured
  `outputRoot` remain the build and developer-tool handoff contract. The repository's pinned Node
  dependencies run directly and generate the platform-specific `dist` directory.
- Version/platform limits: the repository pins Taro/plugin `4.2.0` and React 18; both `weapp` and
  `tt` are built independently and keep their existing `dist/weapp` and `dist/tt` output. Neither
  Docker nor CI replaces either platform's developer tools, preview, upload, review or publishing.
- Rejected: copying an example project, changing page behavior, platform APIs, project structure,
  dependencies, or moving platform publishing into Docker.
- Project landing: the root npm workspace commands provide finite builds and watch modes;
  mock-source builds can run without any Backend, while API-source use expects the separately
  started Miniapp Backend.
