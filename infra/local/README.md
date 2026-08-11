# Local Docker Runtime

Local Docker Compose is the only supported runtime for Tidewise application services and
service-owned operational commands. Host-native `go run`, Vite and Taro processes are not runtime
entrypoints.

The application stack contains Data, AgentRun, Miniapp Backend, Admin Backend and Admin Web.
PostgreSQL, Neo4j and Qdrant are externally provisioned infrastructure: this Compose file does not
create, upgrade or persist them. Data and AgentRun keep independent externally provisioned
databases; AgentRun keeps only its service-owned Artifact volume. Miniapp frontend builders are
optional Compose profiles.

## Start and stop

Create the ignored runtime environment file and replace every placeholder:

```bash
cp infra/local/.env.example infra/local/.env.local
```

Resolve the complete Compose contract without starting containers:

```bash
npm run runtime:config
```

Before application startup, provision the required PostgreSQL databases/users, Neo4j database and
Qdrant endpoint outside the Tidewise application deployment. Their container-reachable addresses
are configured in `.env.local`; the defaults use `host.docker.internal`.

Required application inputs are explicit: Data uses `TIDEWISE_DB_HOST`,
`TIDEWISW_DB_PASSWORD`, `DATA_NEO4J_HEALTH_URI`, `DATA_NEO4J_HEALTH_USERNAME` and
`DATA_NEO4J_HEALTH_PASSWORD`; AgentRun uses `AGENTRUN_DB_HOST`, `AGENTRUN_DB_PASSWORD`,
`AGENTRUN_QDRANT_URL`, `QDRANT_API_KEY` and `EMBEDDING_API_KEY`. Service identities use
`DATA_SERVICE_TOKEN`, `AGENTRUN_SERVICE_TOKEN` and `ADMIN_SERVICE_TOKEN`. The AgentRun image is
built from `agent-run/backend`; none of these inputs cause Compose to provision the middleware.
The isolated graph projection tool additionally receives `NEO4J_USERNAME` and `NEO4J_PASSWORD`.

Build and start the application stack:

```bash
npm run runtime:up
```

Data and AgentRun migrations run as one-shot dependency services before their servers. A failed
migration prevents the dependent service from starting. Normal shutdown preserves the
service-owned Artifact volume and never manages infrastructure data:

```bash
npm run runtime:down
```

Do not add `-v` to the normal shutdown command. Volume deletion is destructive and is not a normal
reset or rollback mechanism.

Follow logs with:

```bash
npm run runtime:logs
```

## Endpoints

| Process            | Host endpoint            |
| ------------------ | ------------------------ |
| Data               | `http://127.0.0.1:9011`  |
| Miniapp Backend    | `http://127.0.0.1:9012`  |
| Admin Backend      | `http://127.0.0.1:9013`  |
| Admin Web          | `http://127.0.0.1:9014`  |
| AgentRun           | `http://127.0.0.1:9080`  |
| Miniapp H5 profile | `http://127.0.0.1:10086` |

Application-to-application traffic uses Compose DNS (`data`, `agentrun`, `miniapp`,
`adminportal`). Infrastructure endpoints are external inputs; `host.docker.internal` is only the
default bridge to developer-owned local infrastructure and can be replaced in `.env.local`.

## Backend services

Start or rebuild one backend with its declared dependencies:

```bash
npm run backend:dev:data
npm run backend:dev:agentrun
npm run backend:dev:miniapp
npm run backend:dev:admin
```

All Backend images read one service-owned YAML per environment from `/app/configs`. Secrets come
only from `infra/local/.env.local`. Miniapp/Admin BFF containers never receive Data or AgentRun
database credentials.

## Admin Web

Admin Web runs from the same unprivileged nginx image shape used in UAT:

```bash
npm run dev:admin
```

The browser receives only the public Admin Backend URL. Enter `ADMIN_SERVICE_TOKEN` in the UI when
the current Admin contract requires it; no downstream service token is embedded in frontend files.

## Miniapp frontend

Taro and Node run inside the Miniapp frontend builder image. The source directory is bind-mounted,
and the container writes to the existing host `miniapp/frontend/dist/<platform>` directory.

```bash
npm run dev:weapp
npm run dev:tt
npm run dev:h5
```

`TARO_APP_RESEARCH_SOURCE` in `.env.local` selects `api` or `mock`. WeChat and Douyin developer
tools remain host applications and open `miniapp/frontend/dist/weapp` or `dist/tt`; they do not run
inside Docker. The builder uses its own Compose file, so `mock` mode starts without Backend
services, Backend secrets, PostgreSQL, Neo4j or Qdrant. In `api` mode, start the Miniapp Backend
separately with `npm run backend:dev:miniapp`.

## Data operations

Apply migrations explicitly when needed; normal stack startup already performs this step:

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm data-migrate
```

Run Data-owned commands from the Data image:

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm --entrypoint /usr/local/bin/entity-seed data
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm --entrypoint /usr/local/bin/research-theme-dev-seed data
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm --entrypoint /usr/local/bin/research-theme-dev-reset data
```

The destructive Research Theme reset still requires both explicit flags:

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm \
  --entrypoint /usr/local/bin/research-theme-dev-reset data \
  --execute --confirm-database tidewise_local
```

Industry graph projection uses a separate one-shot service so the long-running Data container does
not receive projection credentials:

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml \
  --profile tools run --rm data-industry-projector \
  -expected-sha256 7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608 \
  -dry-run

docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml \
  --profile tools run --rm data-industry-projector \
  -expected-sha256 7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608 \
  -apply -allow-env local
```

Event Semantic Qdrant projection is also isolated:

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml \
  --profile tools run --rm data-semantic-projector -apply -allow-env local
```

PostgreSQL remains the fact source. Neo4j and Qdrant are rebuildable projections.

## AgentRun operations

The infrastructure owner must create the AgentRun database identity before startup. Compose applies
AgentRun-owned migrations and then starts AgentRun. Run AgentRun-owned commands from the same image
and persistent Artifact volume:

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml \
  run --rm agentrun-migrate --check-only

docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml \
  run --rm --no-deps --entrypoint /app/agentrun-config agentrun check

docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml \
  run --rm --no-deps --entrypoint /app/agentrun-artifacts agentrun verify-index --root /app/data
```

Provider keys should be piped to `/app/agentrun-config ... --api-key-stdin`; they must not be placed
in YAML, command arguments, logs or committed environment files.

## Failure diagnosis

- `data-migrate` or `agentrun-migrate` failed: inspect the one-shot container logs before restarting.
- `pending migrations exist`: rebuild the image and rerun the owning migration service.
- `configuration_not_ready`: configure the required AgentRun Model Provider/Connector records.
- Admin CORS failure: `ADMIN_ALLOWED_ORIGIN` must match the mapped Admin Web origin, normally
  `http://127.0.0.1:9014`.
- Miniapp output missing: inspect the selected profile logs and confirm the host `dist` mount is
  writable.
