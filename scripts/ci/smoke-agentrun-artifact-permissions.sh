#!/usr/bin/env bash

set -euo pipefail

image="${1:-tidewise-agentrun:ci}"
smoke_root="${RUNNER_TEMP:-/tmp}"
artifact_dir="$(mktemp -d "${smoke_root%/}/agentrun-artifact-smoke.XXXXXX")"
host_uid="$(id -u)"
host_gid="$(id -g)"

cleanup() {
  docker run --rm \
    --user 0 \
    --entrypoint /bin/sh \
    --volume "${artifact_dir}:/artifact" \
    "$image" \
    -c 'chown "$1:$2" /artifact' cleanup "$host_uid" "$host_gid"
  rmdir "$artifact_dir"
}
trap cleanup EXIT

identity="$(docker run --rm --entrypoint /usr/bin/id "$image")"
case "$identity" in
  *"uid=10001(agentrun)"*"gid=10001(agentrun)"*) ;;
  *)
    echo "AgentRun image must run as uid=10001 gid=10001: $identity" >&2
    exit 1
    ;;
esac

docker run --rm \
  --user 0 \
  --entrypoint /bin/sh \
  --volume "${artifact_dir}:/artifact" \
  "$image" \
  -c 'chown 20000:10001 /artifact && chmod 2770 /artifact'

docker run --rm \
  --entrypoint /bin/sh \
  --volume "${artifact_dir}:/app/data" \
  "$image" \
  -c 'probe="$(mktemp /app/data/.ci-write-probe.XXXXXX)" && rm -f "$probe"'

echo "PASS AgentRun Artifact bind mount is writable through shared GID 10001"
