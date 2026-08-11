---
status: accepted
---

# Use Docker as the only Tidewise service runtime

Tidewise AI uses Docker images and Docker Compose as the only supported way to run application
services and service-owned operational commands in local and deployed environments. Host-native
`go run`, Vite, migration, seed, projector, configuration and Artifact processes are not supported
runtime entrypoints.

The deployable services are Data, AgentRun, Miniapp Backend, Admin Portal Backend and Admin Portal
Frontend. PostgreSQL, Neo4j and Qdrant are always independently operated infrastructure and are
excluded from Tidewise application Compose/release artifacts. Miniapp Frontend is not a deployable
service: Docker provides its reproducible Taro build/watch tool, which writes platform output to the
host-mounted `dist` directory. WeChat or Douyin developer tools and platform publishing stay on the
host and keep their existing release contract.

Each Backend keeps one non-secret YAML file per application environment under its service-owned
`configs/` directory. Local or development YAML uses container-reachable infrastructure endpoints;
Tidewise service-to-service calls use Compose DNS names. A second YAML tree for host-native runtime
is forbidden. Secrets remain environment-injected, and service ownership, API,
database, migration and frontend/backend boundaries do not change.

Native compilers and test runners remain valid implementation and CI tools. This decision governs
application runtime and operational processes, not the internal mechanism used to compile or test
source code.
