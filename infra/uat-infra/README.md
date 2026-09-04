# UAT raw-evidence storage

This directory owns the independently managed MinIO raw-evidence storage on the Tidewise UAT ECS.
It is not part of the four-service Tidewise application release and never owns Huawei RDS or any
AgentOS, OpenSPG, Neo4j, Qdrant, MySQL, KAG, or Reason runtime.

## Topology

```text
Huawei ECS
├── tidewise-uat network
│   └── minio:9000              # shared S3 API, loopback host binding
├── MinIO Console :9001          # office-allowlisted ECS host binding
└── Nginx :443
    └── /raw-evidence/* -> 127.0.0.1:9000/raw-evidence/*

Huawei RDS PostgreSQL              # Tidewise Data database
```

The Compose project is `tidewise-infra-uat`. It creates and preserves only the named volume
`tidewise-infra-uat-minio-data`.

The existing Huawei Cloud security-group source-IP allowlist is the outer boundary for the native
UAT operator ports. Office users reach MinIO Console at `http://124.71.201.208:9001`; port `9000`
remains loopback-only. This direct HTTP exception is UAT-only and must not be copied to production.

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
- Secrets `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`.
- Secrets `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY` for the bucket-scoped AgentOS runtime user. Set the
  same generated pair in the `tidewise-agent-os` repository's UAT Environment.

Never reuse the local demo credentials. The deployment creates `raw-evidence`, attaches a dedicated
bucket policy to the AgentOS user, grants anonymous `GetObject` only (no public bucket listing), and
proves authenticated write/read/delete, browser read, anonymous-write denial, and that the four UAT
application workloads remain healthy and retain their start fingerprints.
The Data container's `dbmigrate` command runs without `-apply` to prove a real read-only Huawei RDS
connection and migration-readiness check without changing the database.

CI runs `scripts/ci/smoke-uat-infra.sh` against isolated Docker volumes and a temporary network. It
exercises the rendered Compose images, policy drift reconciliation, authenticated object lifecycle,
anonymous-read/write-denial behavior, DNS aliases and restart persistence.

After a successful deployment, verify the office-network boundary from outside the ECS:

```bash
curl --fail --show-error --silent http://124.71.201.208:9001/ >/dev/null
if nc -z -w 3 124.71.201.208 9000; then
  echo 'MinIO S3 API must remain private' >&2
  exit 1
fi
```

The first probe must load MinIO Console and the second must prove that the S3 API remains private.

## Deployment and rollback

Run **Deploy UAT Infrastructure** manually from `main`. The selected commit must belong to `main`
and have a successful CI run. The workflow builds a digest-addressed deployment bundle, pulls it on
the self-hosted ECS runner, takes the shared UAT deployment lock, and updates only MinIO.

Successful state is recorded under `/opt/tidewise/infra-uat/state`. A failed later candidate restores
the previous runtime/Compose snapshot. A failed first candidate stops the candidate containers while
preserving the named volume. The manual rollback command uses the recorded snapshot:

```bash
docker compose \
  --env-file /opt/tidewise/infra-uat/state/previous.runtime.env \
  -f /opt/tidewise/infra-uat/state/previous.compose.yaml \
  up -d --wait minio
```
