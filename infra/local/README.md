# Local Docker Runtime

Local Docker Compose is the only supported runtime for Tidewise application services and
service-owned operational commands. Host-native `go run` and Vite service hosting are not runtime
entrypoints. Miniapp Frontend is not a service; its repository-pinned Taro build/watch commands run
directly and hand platform output to the native developer tools.

The `tidewise-app` stack contains Data, Miniapp Backend, Admin Backend and Admin Web.
Reason Server and Agent OS join the same Docker Desktop project only when started from their own
repositories. The independently operated `tidewise-infra` stack contains PostgreSQL, MySQL, Neo4j,
MinIO and Qdrant. Both projects use the `tidewise-local` network. Data and Agent OS keep
independent PostgreSQL databases and roles; sharing the engine does not share data ownership.

## Start and stop

Create the ignored runtime environment file and replace every placeholder:

```bash
cp infra/local/.env.example infra/local/.env.local
```

Resolve both Compose contracts without starting containers:

```bash
npm run runtime:config
```

Provision the service-owned PostgreSQL databases and roles before first application startup. The
infrastructure lifecycle creates missing named volumes, then creates the engine containers and
network; application migrations still own their schemas. Start or intentionally reconcile middleware only when first bootstrapping,
recovering Docker state or changing infrastructure configuration:

```bash
npm run infra:up
npm run infra:status
```

Required application inputs are explicit: Data uses `TIDEWISE_DB_HOST` and
`TIDEWISW_DB_PASSWORD`. Service identities use `DATA_SERVICE_TOKEN` and `ADMIN_SERVICE_TOKEN`.
Admin Evidence 详情使用 `RAW_EVIDENCE_PUBLIC_BASE_URL` 拼接公开采集文档地址；本地默认值为
`http://127.0.0.1:9000`。

Build and start the four Tidewise AI application services. This first ensures existing middleware
containers are running with `--no-recreate`, builds the candidate Data image, applies its migration
ledger in an ephemeral `docker compose run --rm` container, and only then starts application
services:

```bash
npm run runtime:up
```

Data migration is an explicit pre-start run and does not create or retain a `data-migrate`
container. Any failed pre-start operation stops the canonical npm command before its dependent
application startup. Normal shutdown never stops infrastructure, Reason Server or Agent OS.

```bash
npm run runtime:down
```

Do not add `-v` to the normal shutdown command. Volume deletion is destructive and is not a normal
reset or rollback mechanism.

Infrastructure has a separate, explicit shutdown command and should normally remain running:

```bash
npm run infra:down
```

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
| Miniapp H5 profile | `http://127.0.0.1:10086` |

Application-to-application traffic uses Compose DNS (`data`, `miniapp`, `adminportal`).
Applications reach local infrastructure through `postgres`, `qdrant`, `mysql`,
`neo4j` and `minio` on the shared network.

Docker Desktop shows the long-running application containers as `admin-portal-web`,
`admin-portal-service`, `miniapp-service`, and `data-service`. The transient Data migration
container is automatically removed.

## Backend services

Start or rebuild one backend with its declared dependencies:

```bash
npm run backend:dev:data
npm run backend:dev:miniapp
npm run backend:dev:admin
```

All Backend images read one service-owned YAML per environment from `/app/configs`. Secrets come
only from `infra/local/.env.local`. Miniapp/Admin BFF containers never receive Data database
credentials.

## Admin Web

Admin Web runs from the same unprivileged nginx image shape used in UAT:

```bash
npm run dev:admin
```

The browser calls relative `/api/admin/*` on the Admin Web origin. nginx proxies those requests to
the internal `adminportal:9013` service. Enter `ADMIN_SERVICE_TOKEN` in the UI when the current
Admin contract requires it; no downstream service token is embedded in frontend files.

## Miniapp frontend

Install the repository lockfile with Node 22, then run the repository-pinned Taro commands directly.
They write to `miniapp/frontend/dist/<platform>`:

```bash
TARO_APP_RESEARCH_SOURCE=mock npm run dev:weapp
TARO_APP_RESEARCH_SOURCE=mock npm run dev:tt
TARO_APP_RESEARCH_SOURCE=mock npm run dev:h5
```

`TARO_APP_RESEARCH_SOURCE` selects `api` or `mock` for each command. WeChat and Douyin developer
tools open `miniapp/frontend/dist/weapp` or `dist/tt`. Mock mode needs no Backend, secrets,
PostgreSQL, Neo4j or Qdrant. In API mode, start the Miniapp Backend separately with
`npm run backend:dev:miniapp` and provide the approved Miniapp Backend URL.

## Data operations

Apply migrations explicitly when needed; normal npm startup commands already build the candidate
Data image and perform this step before starting services:

```bash
npm run runtime:migrate:data
```

PostgreSQL remains the Data fact source. Data no longer seeds Entity packages or writes Neo4j/Qdrant
projections.

## Failure diagnosis

- Data migration failed: inspect the `runtime:migrate:data` command output; its ephemeral container is removed automatically.
- `pending migrations exist`: rebuild the image and rerun the owning migration service.
- Admin CORS failure: `ADMIN_ALLOWED_ORIGIN` must match the mapped Admin Web origin, normally
  `http://127.0.0.1:9014`.
- Miniapp output missing: inspect the selected profile logs and confirm the host `dist` mount is
  writable.
