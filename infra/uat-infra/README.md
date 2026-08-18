# UAT independent infrastructure

This directory owns the independently managed MySQL and MinIO services on the Tidewise UAT ECS.
It is not part of the four-service Tidewise application release and never owns Huawei RDS,
host-native Neo4j, Qdrant, AgentOS, or Reason Server.

## Topology

```text
Huawei ECS
├── tidewise-uat network
│   ├── mysql:3306              # OpenSPG metadata, loopback host binding
│   └── minio:9000              # shared S3 API, loopback host binding
├── host Neo4j 127.0.0.1:7474/7687
└── Nginx :443
    └── /raw-evidence/* -> 127.0.0.1:9000/raw-evidence/*

Huawei RDS PostgreSQL              # application and AgentOS databases
```

The Compose project is `tidewise-infra-uat`. It creates the persistent named volumes
`tidewise-infra-uat-mysql-data` and `tidewise-infra-uat-minio-data`. Normal deployment and rollback
never remove either volume.

## One-time root bootstrap

Install the reviewed Nginx include in the existing `tideai.tripwise.cn` HTTPS server block, then run:

```bash
sudo bash infra/uat-infra/bootstrap-ecs.sh
```

The bootstrap creates `/opt/tidewise/infra-uat`, installs the versioned Nginx snippet, validates
Nginx and reloads it. It does not create credentials, containers, buckets or volumes.

## GitHub `uat` Environment

Existing SWR and runner variables are reused. Add:

- Variable `RAW_EVIDENCE_PUBLIC_BASE_URL=https://tideai.tripwise.cn`.
- Secrets `OPENSPG_MYSQL_ROOT_PASSWORD`, `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`.
- Secrets `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY` for the bucket-scoped AgentOS runtime user. Set the
  same generated pair in the `tidewise-agent-os` repository's UAT Environment.

Never reuse the local demo credentials. The deployment creates `raw-evidence`, attaches a dedicated
bucket policy to the AgentOS user, grants anonymous `GetObject` only (no public bucket listing), and
proves authenticated write/read/delete, browser read, anonymous-write denial, and that existing UAT
application, AgentOS, Neo4j and Qdrant workloads remain healthy and retain their start fingerprints.
The Data container's `dbmigrate` command runs without `-apply` to prove a real read-only Huawei RDS
connection and migration-readiness check without changing the database.

CI runs `scripts/ci/smoke-uat-infra.sh` against isolated Docker volumes and a temporary network. It
exercises the rendered Compose images, policy drift reconciliation, authenticated object lifecycle,
anonymous-read/write-denial behavior, DNS aliases and restart persistence.

## Deployment and rollback

Run **Deploy UAT Infrastructure** manually from `main`. The selected commit must belong to `main`
and have a successful CI run. The workflow builds a digest-addressed deployment bundle, pulls it on
the self-hosted ECS runner, takes the shared UAT deployment lock, and updates only MySQL and MinIO.

Successful state is recorded under `/opt/tidewise/infra-uat/state`. A failed later candidate restores
the previous runtime/Compose snapshot. A failed first candidate stops the candidate containers while
preserving both named volumes. The manual rollback command uses the recorded snapshot:

```bash
docker compose \
  --env-file /opt/tidewise/infra-uat/state/previous.runtime.env \
  -f /opt/tidewise/infra-uat/state/previous.compose.yaml \
  up -d --wait mysql minio
```
