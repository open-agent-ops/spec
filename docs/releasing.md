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

## Publisher App credentials

Install the organization-owned publisher GitHub App only on this repository,
with Contents write, Actions read and implicit Metadata read. In public-release,
store its client ID as AOM_PUBLISHER_CLIENT_ID and its private key as the
AOM_PUBLISHER_PRIVATE_KEY secret. The pinned token action runs after independent
environment approval, requests only this repository and these permissions, and
revokes the installation token when the job ends. Only the publish step receives
that token. The built-in workflow token has read permissions.

Grant the admitted App a bypass only for tag creation. Do not add a bypass to
main protection or tag update/deletion protection. The private key can mint App
tokens outside Actions too: retain it under owner-controlled secret custody,
never in source, logs or ordinary variables. To rotate, pause release dispatch,
create a replacement App key, replace the environment secret securely, verify
its identity through an approved read-only check, then revoke the old key.
Do not dispatch a release merely to test a credential. Record only key metadata.

## Учётные данные приложения-издателя

Установите принадлежащее организации GitHub App только в этот репозиторий:
Contents write, Actions read и обязательное Metadata read. В public-release
сохраните client ID в AOM_PUBLISHER_CLIENT_ID, приватный ключ — в секрете
AOM_PUBLISHER_PRIVATE_KEY. Закреплённый action получает токен только после
независимого одобрения окружения, ограничивает его этим репозиторием и указанными
правами и отзывает после завершения задания. Токен получает только шаг publish;
встроенный токен workflow имеет права чтения.

Разрешите приложению обход только запрета создания тегов. Защита main и запрет
изменения или удаления тегов остаются без исключений. Приватный ключ позволяет
получать токены и вне Actions: храните его под контролем владельца, исключив
исходники, логи и обычные переменные. Для ротации приостановите запуск релизов,
создайте новый ключ App, безопасно замените секрет окружения, подтвердите
идентичность разрешённой проверкой без записи и отзовите старый ключ. Не запускайте
релиз ради проверки ключа. В доказательствах сохраняйте только метаданные ключа.

## GitVerse branch and browser visibility

The mirror contains the approved release tag and a `main` branch at the same
commit. An existing tag alone is incomplete. Each release updates both refs in
one atomic, non-force push after canonical verification. Existing tags are
immutable; an existing `main` must be an ancestor of the approved candidate.
Divergence or an older release cannot overwrite or roll back the mirror branch.
If GitVerse cannot accept atomic push, publication remains incomplete; there is
no sequential fallback. Readback must confirm both refs and `HEAD` pointing to
`main` at the approved commit. GitHub release attachments remain canonical on
GitHub; this operation mirrors Git refs and their reachable objects.

For the original tag-only `v0.4.0` mirror, the fixed-purpose
`mirror-bootstrap.yml` workflow can create `main` at
`100d10849268ff1c36ddb9568b99a9fbf9ff4dbb`. It runs from reviewed canonical `main`,
requires independent `public-mirror` approval, uses the existing mirror secret,
and admits only attempt 1. It has no dispatch inputs, release creation, tag
write or forced update. A matching branch is a no-op; any other existing branch
stops the repair. This is a separately approved host action, not ordinary CI.

After bootstrap, verify the default branch in GitVerse. If the provider did not
select `main` automatically, a repository administrator must select it in the
repository settings. Do not add administration rights to the permanent mirror
token. The equivalent administrator-only API change is
`PATCH /repos/open-agent-ops/spec` with only `{"default_branch":"main"}` after
confirming repository ID 330883 and the exact branch SHA. Re-read the settings,
`git ls-remote --symref` and the public code page. A branch created successfully
with an incorrect default branch is partial progress, not complete recovery;
retain the receipt and reconcile before authorizing another execution.
