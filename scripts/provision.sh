#!/bin/sh
set -eu
AOM_TOOLS_DIR=${AOM_TOOLS_DIR:-${TMPDIR:-/tmp}/aom-spec-tools-1.26.8}
export AOM_TOOLS_DIR
case "$(uname -s)/$(uname -m)" in
  Linux/x86_64) go_url='https://go.dev/dl/go1.26.8.linux-amd64.tar.gz'; go_sha='d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b' ;;
  Linux/aarch64) go_url='https://go.dev/dl/go1.26.8.linux-arm64.tar.gz'; go_sha='211ffced9dcb9633a55eac6364816ec0ddd951389a740e88fa8b3337971bdda0' ;;
  Darwin/x86_64) go_url='https://go.dev/dl/go1.26.8.darwin-amd64.tar.gz'; go_sha='186be014105aa6542b767d2c6ed5cca10a0214bdff809ef1724022a8c7894150' ;;
  Darwin/arm64) go_url='https://go.dev/dl/go1.26.8.darwin-arm64.tar.gz'; go_sha='a012b25b571bd0138a03dcd25375ceba866fe5ca822f426d2c66a4de56fd3f4b' ;;
  *) printf '%s\n' 'unsupported public platform' >&2; exit 2 ;;
esac
mkdir -p "$AOM_TOOLS_DIR/bin"
work=$(mktemp -d "${TMPDIR:-/tmp}/aom-provision.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
verify() {
  if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$1" | cut -d ' ' -f 1)
  else actual=$(shasum -a 256 "$1" | cut -d ' ' -f 1); fi
  test "$actual" = "$2" || { printf '%s\n' 'tool checksum mismatch' >&2; exit 3; }
}
# Every provisioning run verifies official archive bytes; no shared action cache.
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 --max-time 180 "$go_url" -o "$work/go.tar.gz"
verify "$work/go.tar.gz" "$go_sha"
tar -xzf "$work/go.tar.gz" -C "$AOM_TOOLS_DIR"
PATH="$AOM_TOOLS_DIR/go/bin:$AOM_TOOLS_DIR/bin:$PATH"
export PATH GOTOOLCHAIN=local GOWORK=off CGO_ENABLED=0
GOROOT="$AOM_TOOLS_DIR/go"
export GOROOT
[ "$(go env GOVERSION)" = go1.26.8 ]
go mod download
# Build the same pinned actionlint source with the supported compiler. The
# upstream prebuilt archive embeds Go1.26.1 and is not an admitted executable.
(cd "$work" && go mod download -json github.com/rhysd/actionlint@v1.7.12) > "$work/actionlint-module.json"
grep -F '"Sum": "h1:vQ4GeJN86C0QH+gTUQcs8McmK62OLT3kmakPMtEWYnY="' "$work/actionlint-module.json" >/dev/null
grep -F '"GoModSum": "h1:krOUhujIsJusovkaYzQ/VNH8PFexjNKqU0q5XI/4w+g="' "$work/actionlint-module.json" >/dev/null
GOBIN="$AOM_TOOLS_DIR/bin" go install -trimpath -ldflags=-buildid= github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
# Fixed tool module is outside the public runtime dependency graph.
go mod download -json golang.org/x/vuln@v1.3.0 > "$work/govulncheck-module.json"
grep -F '"Sum": "h1:hZYzR8uRhYhDSX88d+40TWbKAVw7BIvRWm26rtEn8jw="' "$work/govulncheck-module.json" >/dev/null
grep -F '"GoModSum": "h1:MIY2PaR1y52stzZM3uHBboUAdVJvSVMl5nP3OQrwQaE="' "$work/govulncheck-module.json" >/dev/null
GOBIN="$AOM_TOOLS_DIR/bin" go install -trimpath -ldflags=-buildid= golang.org/x/vuln/cmd/govulncheck@v1.3.0
go mod verify
printf '%s\n' "Verified tools: $AOM_TOOLS_DIR"

# Persist the verified compiler for subsequent native GitHub Actions steps.
# A shell export alone cannot change the environment of a later step.
if [ "${GITHUB_ACTIONS:-}" = true ]; then
  test -n "${GITHUB_PATH:-}" && test -n "${GITHUB_ENV:-}"
  printf '%s\n' "$AOM_TOOLS_DIR/go/bin" "$AOM_TOOLS_DIR/bin" >> "$GITHUB_PATH"
  printf '%s\n' "GOROOT=$AOM_TOOLS_DIR/go" "AOM_TOOLS_DIR=$AOM_TOOLS_DIR" >> "$GITHUB_ENV"
fi
