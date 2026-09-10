#!/bin/sh
set -eu
case "${1:-}" in
  fast|heavy|vulnerability|release) profile=$1 ;;
  *) printf '%s\n' 'usage: scripts/check.sh fast|heavy|vulnerability|release' >&2; exit 2 ;;
esac
AOM_TOOLS_DIR=${AOM_TOOLS_DIR:-${TMPDIR:-/tmp}/aom-spec-tools-1.26.8}
GOROOT="$AOM_TOOLS_DIR/go"
export AOM_TOOLS_DIR GOROOT GOTOOLCHAIN=local GOWORK=off
PATH="$AOM_TOOLS_DIR/go/bin:$AOM_TOOLS_DIR/bin:$PATH"
export PATH
[ "$(go env GOVERSION)" = go1.26.8 ] || { printf '%s\n' 'run sh scripts/provision.sh first' >&2; exit 3; }
case "$profile" in
  fast) gates='policy conformance docs' ;;
  heavy) gates='heavy reproducibility' ;;
  vulnerability) gates='supply-chain' ;;
  release) gates='policy conformance docs supply-chain heavy reproducibility' ;;
esac
for gate in $gates; do
  go run -mod=readonly ./cmd/spec-check gate "$gate"
done
