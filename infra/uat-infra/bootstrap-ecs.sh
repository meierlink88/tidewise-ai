#!/usr/bin/env bash

set -euo pipefail

[ "$(id -u)" -eq 0 ] || { echo "bootstrap-ecs.sh must be run as root" >&2; exit 1; }
[ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] \
  || { echo "UAT infrastructure requires Linux x86_64" >&2; exit 1; }

deploy_user="${TIDEWISE_DEPLOY_USER:-tidewise-deploy}"
deploy_root="${TIDEWISE_INFRA_DEPLOY_ROOT:-/opt/tidewise/infra-uat}"
nginx_server_config="${NGINX_SERVER_CONFIG:-/etc/nginx/sites-enabled/tideai.tripwise.cn}"
snippet_source="$(cd "$(dirname "$0")" && pwd)/nginx-raw-evidence-location.conf"
snippet_target=/etc/nginx/snippets/tidewise-raw-evidence-uat.conf

id "$deploy_user" >/dev/null
command -v docker >/dev/null
command -v nginx >/dev/null
docker network inspect tidewise-uat >/dev/null
[ -f "$nginx_server_config" ] || { echo "$nginx_server_config is not a file" >&2; exit 1; }

install -d -m 0750 -o "$deploy_user" -g "$deploy_user" \
  "$deploy_root" "$deploy_root/state"
install -d -m 0755 /etc/nginx/snippets
install -m 0644 "$snippet_source" "$snippet_target"

include_line="    include ${snippet_target};"
grep -Fq "$include_line" "$nginx_server_config" || {
  echo "Add this line to the existing tideai.tripwise.cn HTTPS server block, then rerun bootstrap:" >&2
  echo "$include_line" >&2
  exit 1
}

nginx -t
systemctl reload nginx

echo "PASS uat-infrastructure-bootstrap"
