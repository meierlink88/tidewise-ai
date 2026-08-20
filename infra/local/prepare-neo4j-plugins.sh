#!/usr/bin/env bash

set -euo pipefail

neo4j_image='neo4j:5.26.28-community@sha256:ff32db30b2baff97971e441b46bfd9c832c1b62c970398ef579244c06b21d357'
plugin_volume='tidewise-reason_neo4j-5.26-plugins'
gds_version=2.13.4
gds_url="https://graphdatascience.ninja/neo4j-graph-data-science-${gds_version}.jar"
gds_sha256='10e072f73992224f1159f246c9d6a89da5f3b3434aeffa5be42647edda13a8d8'

docker volume inspect "$plugin_volume" >/dev/null

docker run --rm \
  --volume "$plugin_volume:/plugins" \
  --env GDS_VERSION="$gds_version" \
  --env GDS_URL="$gds_url" \
  --env GDS_SHA256="$gds_sha256" \
  --entrypoint bash \
  "$neo4j_image" -c '
    set -euo pipefail

    apoc_source=/var/lib/neo4j/labs/apoc-5.26.28-core.jar
    apoc_target=/plugins/apoc-5.26.28-core.jar
    gds_target="/plugins/neo4j-graph-data-science-${GDS_VERSION}.jar"
    gds_staging="${gds_target}.download"

    if [ -f "$apoc_target" ]; then
      cmp --silent "$apoc_source" "$apoc_target"
    else
      install -m 0644 "$apoc_source" "$apoc_target"
    fi

    if [ ! -f "$gds_target" ]; then
      rm -f "$gds_staging"
      wget --quiet --timeout=300 --tries=8 --output-document="$gds_staging" "$GDS_URL"
      printf "%s  %s\n" "$GDS_SHA256" "$gds_staging" | sha256sum --check --status
      mv "$gds_staging" "$gds_target"
      chmod 0644 "$gds_target"
    fi
    printf "%s  %s\n" "$GDS_SHA256" "$gds_target" | sha256sum --check --status

    unexpected="$(find /plugins -maxdepth 1 -type f -name "*.jar" \
      ! -name "apoc-5.26.28-core.jar" \
      ! -name "neo4j-graph-data-science-${GDS_VERSION}.jar" -print)"
    [ -z "$unexpected" ] || {
      printf "Unexpected Neo4j plugin artifacts:\n%s\n" "$unexpected" >&2
      exit 1
    }
  '

echo "PASS prepared Neo4j APOC 5.26.28 and GDS $gds_version"
