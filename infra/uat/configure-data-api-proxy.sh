#!/usr/bin/env bash

set -euo pipefail

test_root="${TIDEWISE_NGINX_TEST_ROOT:-}"
if [ -n "$test_root" ]; then
  [ -f "$test_root/.tidewise-nginx-test-root" ] \
    || { echo "invalid TIDEWISE_NGINX_TEST_ROOT" >&2; exit 1; }
  nginx_server_config="$test_root/etc/nginx/sites-enabled/tideai.tripwise.cn"
  snippet_target="$test_root/etc/nginx/snippets/tidewise-data-api-uat.conf"
else
  [ "$(id -u)" -eq 0 ] || { echo "configure-data-api-proxy.sh must be run as root" >&2; exit 1; }
  [ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] \
    || { echo "UAT Data proxy requires Linux x86_64" >&2; exit 1; }
  nginx_server_config="${NGINX_SERVER_CONFIG:-/etc/nginx/sites-enabled/tideai.tripwise.cn}"
  snippet_target=/etc/nginx/snippets/tidewise-data-api-uat.conf
fi
snippet_source="$(cd "$(dirname "$0")" && pwd)/nginx-data-api-location.conf"
operation="${1:-install}"

command -v nginx >/dev/null
command -v systemctl >/dev/null
[ -f "$nginx_server_config" ] || { echo "$nginx_server_config is not a file" >&2; exit 1; }

include_line="    include ${snippet_target};"

case "$operation" in
  install)
    install -d -m 0755 "$(dirname "$snippet_target")"
    backup="$(mktemp)"
    previous_snippet_exists=false
    if [ -f "$snippet_target" ]; then
      cp -p "$snippet_target" "$backup"
      previous_snippet_exists=true
    fi
    restore_snippet() {
      if [ "$previous_snippet_exists" = true ]; then
        cp -p "$backup" "$snippet_target"
      else
        rm -f "$snippet_target"
      fi
    }
    cleanup_install() {
      rm -f "$backup"
    }
    trap cleanup_install EXIT
    if ! install -m 0644 "$snippet_source" "$snippet_target"; then
      restore_snippet
      echo "Data API snippet installation failed; restored the previous snippet" >&2
      exit 1
    fi
    if ! grep -Fq "$include_line" "$nginx_server_config"; then
      restore_snippet
      echo "Add this line to the existing tideai.tripwise.cn HTTPS server block, then rerun:" >&2
      echo "$include_line" >&2
      exit 1
    fi
    if ! nginx -t; then
      restore_snippet
      nginx -t >/dev/null 2>&1 || true
      echo "Nginx validation failed; restored the previous Data API snippet" >&2
      exit 1
    fi
    if ! systemctl reload nginx; then
      restore_snippet
      if nginx -t >/dev/null 2>&1; then
        systemctl reload nginx >/dev/null 2>&1 || true
      fi
      echo "Nginx reload failed; restored the previous Data API snippet" >&2
      exit 1
    fi
    rm -f "$backup"
    trap - EXIT
    echo "PASS uat-data-api-proxy"
    ;;
  remove)
    backup="$(mktemp)"
    candidate="$(mktemp)"
    cleanup_remove() {
      rm -f "$backup" "$candidate"
    }
    trap cleanup_remove EXIT
    cp -p "$nginx_server_config" "$backup"
    sed "\|^[[:space:]]*include ${snippet_target};[[:space:]]*$|d" "$nginx_server_config" > "$candidate"
    if ! cp "$candidate" "$nginx_server_config"; then
      cp -p "$backup" "$nginx_server_config"
      echo "Nginx server update failed; restored the previous server configuration" >&2
      exit 1
    fi
    if ! nginx -t; then
      cp -p "$backup" "$nginx_server_config"
      echo "Nginx validation failed; restored the previous server configuration" >&2
      exit 1
    fi
    if ! systemctl reload nginx; then
      cp -p "$backup" "$nginx_server_config"
      if nginx -t >/dev/null 2>&1; then
        systemctl reload nginx >/dev/null 2>&1 || true
      fi
      echo "Nginx reload failed; restored the previous server configuration" >&2
      exit 1
    fi
    rm -f "$snippet_target" "$backup" "$candidate"
    trap - EXIT
    echo "PASS uat-data-api-proxy-removed"
    ;;
  *)
    echo "usage: configure-data-api-proxy.sh [install|remove]" >&2
    exit 2
    ;;
esac
