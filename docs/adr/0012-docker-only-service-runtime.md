---
status: accepted
---

# Use Docker as the Tidewise deployable-service runtime

Tidewise AI uses Docker images and Docker Compose as the only supported way to run deployable
application services and service-owned operational commands in local and deployed environments.
Host-native `go run`, Vite service hosting, migration, seed, projector, configuration and Artifact
processes are not supported runtime entrypoints.

The deployable services are Data, AgentRun, Miniapp Backend, Admin Portal Backend and Admin Portal
Frontend. PostgreSQL, MySQL, Neo4j, MinIO and Qdrant are independently operated infrastructure and
are excluded from Tidewise application Compose/release artifacts. The local developer bootstrap
declares them in the separate `tidewise-infra` project described by ADR-0020; this does not make
them application services or UAT/release infrastructure. Miniapp Frontend is not a deployable
service and is therefore not part of the Docker runtime contract: repository-pinned Node/Taro runs
directly for development and CI, writes `dist/weapp` or `dist/tt`, and hands those outputs to the
WeChat or Douyin developer tools and existing platform publishing flow.

Admin Portal Frontend remains a deployable static Web service. Its unprivileged nginx container is
the browser's only Admin origin: browser requests use relative `/api/admin/*`, and nginx proxies
them to the internal `adminportal:9013` service. UAT publishes Admin Web `9014` but not Admin
Backend `9013`.

Each Backend keeps one non-secret YAML file per application environment under its service-owned
`configs/` directory. Local or development YAML uses container-reachable infrastructure endpoints;
Tidewise service-to-service calls use Compose DNS names. A second YAML tree for host-native runtime
is forbidden. Secrets remain environment-injected, and service ownership, API,
database, migration and frontend/backend boundaries do not change.

Native compilers and test runners remain valid implementation and CI tools. This decision governs
deployable application runtime and operational processes, not the internal mechanism used to
compile or test source code.
