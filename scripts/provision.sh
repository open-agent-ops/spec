#!/bin/sh
set -eu
AOM_TOOLS_DIR=${AOM_TOOLS_DIR:-${TMPDIR:-/tmp}/aom-spec-tools-1.26.10}
export AOM_TOOLS_DIR
case "$(uname -s)/$(uname -m)" in
  Linux/x86_64) go_url='https://go.dev/dl/go1.26.10.linux-amd64.tar.gz'; go_sha='ccfe7ae822d2c2f41f4fe7c956199967a4dac22d0ee0d2b2bfdb440aab3b20c5' ;;
  Linux/aarch64) go_url='https://go.dev/dl/go1.26.10.linux-arm64.tar.gz'; go_sha='3f4227ce7b206a513e794f1393adb473a07b46e94d18bd15d1c684bbb7319797' ;;
  Darwin/x86_64) go_url='https://go.dev/dl/go1.26.10.darwin-amd64.tar.gz'; go_sha='e820ab780c273ae4a340b79d0f3c4313b74ee029aa3874792145d44b9f4f91da' ;;
  Darwin/arm64) go_url='https://go.dev/dl/go1.26.10.darwin-arm64.tar.gz'; go_sha='0b5183492292ada170f9b561a834863808dd02edba599e4ad0e89899330020b3' ;;
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
[ "$(go env GOVERSION)" = go1.26.10 ]
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
