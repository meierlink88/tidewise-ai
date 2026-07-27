# UAT Industry Graph Projection V1

Status: approved for implementation

Date: 2026-07-28

Owner: Data context

## Outcome

Project the approved Industry relationship package from UAT PostgreSQL into the
dedicated UAT Neo4j database as an auditable, repeatable deployment operation.
The resulting graph is suitable for traversing Industry, Concept,
IndustryChain, and ChainNode entities without changing the approved business
semantics.

The frozen V1 contract is:

- package SHA-256:
  `7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608`
- namespace: `tidewise-industry-v1`
- contract version: `industry-graph-projection-v1`
- nodes: 4,449
- relationships: 7,867

## Non-goals

- Neo4j does not become a source of truth. PostgreSQL remains authoritative.
- This change does not add new chains, nodes, mappings, or relationship types.
- The long-running Data Service does not receive Neo4j credentials or acquire a
  runtime dependency on Neo4j.
- Ordinary UAT deployments do not automatically rewrite the graph.
- This change does not introduce a general-purpose remote graph writer.

## Invocation contract

`Deploy UAT` exposes two independent, default-off controls:

- `apply_industry_graph_projection`
- `industry_graph_package_sha`

When projection is disabled, supplying a package SHA is rejected. When
projection is enabled, a lowercase 64-character package SHA is required.
Relationship import and graph projection remain independently selectable. This
allows projecting a package already present in UAT PostgreSQL without replaying
the PostgreSQL import.

Only when projection is enabled, the workflow injects the following GitHub
`uat` Environment values into the deployment orchestration step and forwards
them by variable name only to the one-shot projection container:

- Variable: `NEO4J_URI`
- Variable: `NEO4J_USERNAME`
- Variable: `NEO4J_DATABASE`
- Secret: `NEO4J_PASSWORD`

When projection is disabled, the step receives empty values. They are never
written into `runtime.env`, the Compose service definition, the deployment
state, reports, or diagnostics, and no other container receives them.

## Target safety

Apply mode requires `-allow-env uat`, and the requested environment must match
the loaded application configuration.

The UAT projector accepts only:

- `APP_ENV=uat`
- PostgreSQL database `tidewise_uat`
- PostgreSQL role `tidewise_uat`
- PostgreSQL `ssl_mode=require`
- the repository-controlled UAT PostgreSQL host
- credential-free `bolt://host.docker.internal:7687`
- Neo4j database `neo4j`
- non-empty Neo4j username and password

The UAT Data service maps `host.docker.internal` to Docker's `host-gateway`.
This keeps the one-shot projector on the host-local route to the systemd-managed
Neo4j service and avoids unsupported public-IP NAT hairpinning from the
deployment container.

Local projection remains restricted to loopback PostgreSQL
`tidewise_local`/`ssl_mode=disable` and a loopback Bolt URI. Production and
arbitrary remote targets are rejected.

## Projection transaction

The candidate Data image contains `industry-graph-projector` and the frozen
relationship package. Before candidate services are started, the deployment
script runs:

1. `-dry-run`
2. `-apply -allow-env uat`
3. the same apply command again as an idempotency replay

The projector first reconstructs the approved baseline from the package, reads
the source graph from UAT PostgreSQL, and requires exact semantic equality.
Neo4j replacement remains transactional inside the fixed namespace.

The deployment gate accepts the three reports only when:

- every report identifies the requested package SHA;
- dry-run and apply flags are correct;
- source, current/final projection fingerprints and type counts satisfy the
  frozen V1 contract;
- final integrity violation count is zero;
- replay reports `unchanged=true`.

Failure occurs before the candidate service release is promoted. PostgreSQL is
not mutated by graph projection. A failed Neo4j transaction retains the
previous namespace state.

## Rollout and rollback

The first UAT projection is dispatched manually from `main` with the projection
flag and exact package SHA. The Actions summary records counts and validation
status without credentials.

Application rollback does not rewrite Neo4j because the graph is a derived,
versioned namespace. If recovery is needed, rerun the last approved package
projection. A future incompatible graph contract must use a new namespace and
an explicit consumer cutover.

## Verification seams

- CLI and target-safety unit tests cover local/UAT acceptance and arbitrary
  target rejection.
- Repository contract tests cover workflow inputs, secret scoping, image
  contents, and the deploy script gate.
- Executor tests use fake Docker to prove dry-run/apply/replay order and that
  invalid inputs fail before any graph write.
- Existing Biz and Neo4j adapter tests remain authoritative for graph semantics,
  transactional replacement, constraints, and idempotency.
