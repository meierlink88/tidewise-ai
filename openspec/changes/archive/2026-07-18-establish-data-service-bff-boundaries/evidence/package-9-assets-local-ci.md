# Package 9 Assets, Local, CI, And Docs Evidence

## Scope And Base

- Accepted Package 8 base: `30c4afb902f148a1b713b884684f10571a34ed2a`.
- Executed only tasks 9.1-9.3. Package 10 roles/credentials and later lifecycle stages were not entered.
- No PostgreSQL or Neo4j connection, migration, seed, import, business write, service start, image run, image push, deployment, or UAT/prod/shared operation was performed.
- `.github/workflows/deploy-uat.yml`, `infra/uat/**`, `backend/migrations/**`, prod, and shared assets have zero diff.

## Consumer And Asset Manifest

Before Package 9:

- `backend/Dockerfile` built only the retired combined `admin-api`/`dbmigrate` image path.
- `infra/local/docker-compose.postgres.yaml` and `infra/local/docker-compose.neo4j.yaml` were independent datastore-only compose files.
- CI ran the generic backend test and Admin frontend checks, but did not explicitly build or image-test the three service boundaries.

After Package 9:

- Added service-owned assets:
  - `backend/services/data/Dockerfile` and `backend/services/data/config/config.local.yaml`.
  - `backend/services/miniapp/Dockerfile` and `backend/services/miniapp/config/config.local.yaml`.
  - `backend/services/adminportal/Dockerfile` and `backend/services/adminportal/config/config.local.yaml`.
- Each image has a unique service binary/CMD and checks both `/healthz` and `/readyz`; only Data carries migrations and datastore configuration.
- Added `infra/local/docker-compose.yaml` with exactly `postgres`, `neo4j`, `data`, `miniapp`, and `adminportal`, plus the existing local network/volumes. Miniapp/Admin receive Data service identity/URL only and no PostgreSQL/Neo4j credential.
- Updated local `.env.example` and README for the three-service topology and dry validation.
- Updated CI to run the Data OpenAPI/client drift contract, architecture/reference contracts, build all three binaries, and build all three service-owned images.
- Added/updated architecture consumer tests that fail if the old image, split compose, DB-coupled BFF assets, retired runtime paths, or incomplete CI consumers return.
- Removed `backend/Dockerfile` and the two split compose files only after local and CI consumers referenced the three service-owned assets.

## TDD And Validation

RED:

- New service-asset, local-compose, and CI-consumer architecture tests failed against the Package 8 tree because the service-owned assets and consumers did not yet exist.

GREEN:

- `go test ./internal/architecture -run 'TestServiceOwnedDockerAssets|TestLocalCompose|TestCIConsumesThreeServiceOwnedImagesAndBoundaryContracts|TestLocalInfra' -count=1`: PASS.
- `go test ./internal/architecture -count=1`: PASS.
- `go test ./services/data/... ./services/miniapp/... ./services/adminportal/... -count=1`: PASS (loopback-enabled execution for HTTP client tests).
- `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci.yml")'`: PASS. An earlier invocation used an option unsupported by the host Ruby 2.6 YAML API; the compatible syntax check passed without changing the workflow.
- `docker compose --env-file infra/local/.env.example -f infra/local/docker-compose.yaml config`: PASS. This was configuration rendering only; `up`, `start`, and service execution were not run.
- Host binary builds for Data, Miniapp, and Adminportal to `/tmp`: PASS; no repository artifact was created.
- Local image builds, without run or push: PASS, all `linux/amd64` with distinct CMD and health/readiness checks:
  - Data: `sha256:e932fd640d3f5d389e2cb054645e886b7d1022eb17498acc929b46228d826ab1`.
  - Miniapp: `sha256:8bc620ae9d825ae2823f056b79f4803acddafb741595546165eebb6ad2497107`.
  - Adminportal: `sha256:5c4ec88ce7a4937b88f95b4f0cadef7140fe1ff643ccd30fae6582ed9e26c3c8`.
- `openspec validate establish-data-service-bff-boundaries --strict`: PASS.
- Explicit `OPENSPEC_TASK_LINT_CHANGE=establish-data-service-bff-boundaries` task-design lint: PASS.
- `git diff --check`: PASS.
- Scoped scans: BFF assets/config contain no datastore credentials; service/compose/CI assets contain no retired scheduler/runtime reference; excluded deploy/UAT/migration paths have zero diff; no credential value or private-key material was added.

## Stop Boundary

Package 9 is complete at an R1 code/assets checkpoint. Package 10 local Data DB roles/credentials remains pending and requires its separate R2 authorization. No database, graph, deployment, Sync, Archive, or Deliver action was taken.
