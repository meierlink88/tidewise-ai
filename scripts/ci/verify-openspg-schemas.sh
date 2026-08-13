#!/usr/bin/env bash
set -euo pipefail

repository_root="$(git rev-parse --show-toplevel)"
kag_commit="fdab15b3929d2ee40dfcdd388f90233096a6afc9"
kag_checkout="$(mktemp -d)"
parser_environment="$(mktemp -d)"

cleanup() {
  rm -rf "$kag_checkout" "$parser_environment"
}
trap cleanup EXIT

git -C "$kag_checkout" init --quiet
git -C "$kag_checkout" fetch --quiet --depth=1 https://github.com/OpenSPG/KAG.git "$kag_commit"
git -C "$kag_checkout" checkout --quiet --detach FETCH_HEAD

python3 -m venv "$parser_environment"
"$parser_environment/bin/pip" install --quiet --requirement "$repository_root/scripts/ci/openspg-parser-requirements.txt"
"$parser_environment/bin/python" "$repository_root/scripts/ci/verify-openspg-schemas.py" \
  --kag-root "$kag_checkout" \
  --schema-root "$repository_root/data-service/doctype"
