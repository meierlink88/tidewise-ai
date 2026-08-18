---
status: accepted
date: 2026-08-18
issue: 280
amended_by: 284
---

# Operate UAT MySQL and MinIO independently from application releases

## Decision

The UAT ECS runs OpenSPG MySQL and shared MinIO in a separately managed
`tidewise-infra-uat` Compose project on the existing `tidewise-uat` network. The four Tidewise
application services, external AgentOS and future Reason Server releases may consume those services
through stable network aliases, but none of their release transactions owns the infrastructure
containers or volumes.

- MySQL owns OpenSPG product metadata and persists in `tidewise-infra-uat-mysql-data`.
- MinIO owns shared S3 objects and persists in `tidewise-infra-uat-minio-data`.
- MySQL and the MinIO S3 API bind host ports only to loopback. MinIO Console publishes native host
  port `9001` for office-network operators; the existing Huawei Cloud source-IP allowlist is the
  outer UAT access boundary.
- Nginx exposes anonymous object `GetObject` only under `/raw-evidence/` with no public bucket
  listing; authenticated writes use the
  internal `http://minio:9000` endpoint and a bucket-scoped AgentOS identity.
- UAT administrator and service credentials are GitHub Environment Secrets and never reuse local
  demo defaults.
- Existing Huawei RDS PostgreSQL, host-native Neo4j and independently operated Qdrant remain outside
  this Compose project and are not migrated, restarted or rolled back by its deployment.

## Delivery and failure boundary

The infrastructure has a manual, CI-gated, digest-addressed GitHub Action and shares the ECS UAT
deployment lock with application and AgentOS releases. Persistent volumes survive every normal
deployment and rollback. A failed later candidate restores the previous Compose/runtime snapshot; a
failed first candidate stops only its containers and preserves both volumes for inspection.

The deployment creates the `raw-evidence` bucket, installs the reviewed bucket policy, enables
anonymous object download, and proves authenticated write/read/delete plus browser read through the
TLS ingress. It does not deploy OpenSPG Server or KAG; Reason Server remains a separate release unit.

## Consequences

Local `tidewise-infra` remains a developer bootstrap and is not promoted byte-for-byte. UAT instead
has an explicit environment-owned infrastructure contract with production-style credentials,
office-allowlisted Console access, private MySQL/S3 API exposure, observability and rollback. The
native HTTP Console exception is limited to UAT and is removed when a managed HTTPS/VPN operator
ingress exists. Adding Reason Server later may depend on `mysql`, `minio` and existing Neo4j, but it
cannot absorb their lifecycle or data ownership.
