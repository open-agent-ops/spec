# Release and recovery

This repository releases specification artifacts and a Go module. It does not
deploy a runtime service. Actual GitHub/GitVerse settings and credentials must
be admitted by the owner before this workflow is activated. Local tests use
synthetic providers and cannot prove public host readiness.

Builds run natively on GitHub-hosted ubuntu-24.04 with SHA256-verified Go.
No workload container or OCI scanner is required. Dependency and tool
vulnerability findings still block the supply-chain gate.

## Preconditions

Confirm intended maintainer/owner teams and immutable owner IDs, independent
reviewers, protected main, exact required checks, no direct/force/delete bypass,
private security reporting and both protected environments. Configure
`AOM_OWNER_IDS` in public-release as the admitted comma-separated immutable
owner IDs. Configure `AOM_MIRROR_REPOSITORY` and the narrowly scoped
`AOM_MIRROR_TOKEN` only in public-mirror. Missing or ambiguous configuration
blocks execution. No source edit or invented principal supplies host authority.

## Exact release

1. Review the clean main candidate and run all public checks. Dispatch AOM release
   from main with its full lowercase 40-hex SHA and stable version. The candidate
   must equal the dispatch workflow SHA. Only run attempt 1 is accepted.
2. Read-only build jobs repeat full checks and produce deterministic source,
   hashes, SBOM and provenance. The immutable proposal binds source tree,
   inventory, assets, lock, policy, version and this run. It expires within 24h.
3. An owner distinct from the initiator reviews that concrete proposal and
   approves public-release. Host admission must prevent self-review/admin bypass.
   JSON fields, labels, comments and old approvals grant no authority.
4. The publisher verifies same-run artifact digests, main ancestry and current
   owner review history, then creates or reconciles the exact tag/release/assets.
   It does not execute install hooks or PR code. Canonical readback must match.
5. The separate protected mirror job sends only the public commit and exact tag
   to the admitted GitVerse repository, then reads back both identities. Full
   publication requires both canonical and mirror verification.

## Interrupted operations

Observe remote state before retrying a timed-out write. Matching immutable
objects are reused; differing tags or asset bytes are conflicts. Confirmed
absence after a classified transient failure permits at most three total attempts
per operation, with five- and fifteen-second backoffs. Respect Retry-After and
the remaining job deadline. Unknown or conflicting readback stops further writes.
HTTP connection/TLS establishment is bounded to30 seconds, metadata to60 seconds
and transfers to180 seconds. Git commands have total60/180-second limits,
including connection setup; no separate Git connection-phase bound is claimed.
Publish and mirror jobs have30-minute budgets, including setup and readback.
Unknown observations remain unknown and block success. A failed mirror leaves
canonical verification intact and reports mirror incomplete, never published.

Do not rerun an old publication attempt. Recovery after a terminated run needs a
new dispatch, immutable proposal and independent environment decision. Never
force a tag or overwrite assets. Consumers can restore a previous immutable
release, or adopt a new reviewed fix through their normal version-pinned PR.
Public release does not update a private development repository.

Retain full stdout/stderr, record hashes and same-run identity for 90 days;
release source/SBOM/provenance remain immutable release assets. Check actual host
retention and notification ownership at admission. Secrets never belong in
logs, URLs, archives, receipts or persistent Git configuration.
