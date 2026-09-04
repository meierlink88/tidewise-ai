#!/usr/bin/env bash

# Single reviewed inventory for the one-time UAT legacy-runtime retirement.
# Consumers must not extend these arrays from workflow inputs or environment data.
readonly -a UAT_RETAINED_CONTAINERS=(
  tidewise-uat-data-1
  tidewise-uat-miniapp-1
  tidewise-uat-adminportal-1
  tidewise-uat-admin-1
  tidewise-infra-uat-minio-1
)

readonly -a UAT_RETIRED_CONTAINERS=(
  tidewise-agentos-uat-agentos-1
  tidewise-uat-agentrun-1
  agentrun-service
  agentrun-migrate
  agentrun-agent-version
  reason-server-uat
  tidewise-uat-qdrant
  tidewise-infra-uat-mysql-1
  tidewise-uat-openspg-neo4j
)

readonly -a UAT_RETIRED_VOLUMES=(
  tidewise-app-agentrun-artifacts
  tidewise-uat-qdrant-data
  tidewise-infra-uat-mysql-data
  tidewise-uat-openspg-neo4j-data
  tidewise-uat-openspg-neo4j-logs
)

readonly -a UAT_RETIRED_PATHS=(
  /opt/tidewise/agentos-uat
  /opt/tidewise/uat/agentrun-artifacts
  /opt/tidewise/uat/logs/agentrun
  /opt/tidewise/reason-uat
  /opt/tidewise/neo4j-uat
)

readonly -a UAT_RETIRED_PORTS=(3306 6333 6334 7474 7687 8887 9081)

readonly -a UAT_RETIRED_PROJECT_UNITS=(
  actions.runner.meierlink88-tidewise-agent-os.tidewise-agentos-uat-ecs.service
  actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service
)
