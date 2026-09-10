# Contributor guide

1. Use the public bug/proposal form, or the PR template for a small fix. State the
   affected version, reproduction or motivation, compatibility and EN/RU impact.
   Acknowledge Apache-2.0 for code/schemas/metadata and CC BY 4.0 for documentation.
2. A maintainer records triage, a reason and the next action. Normative or
   authority changes need an accepted owner proposal before implementation.
3. Provision the exact tools with `sh scripts/provision.sh`. Public checks use
   Go 1.26.8; Go 1.25.13 remains the module compatibility minimum. Provisioning
   needs network access to the verified official sources. It verifies pins and
   checksums and builds the selected actionlint/go vulnerability tools with
   Go 1.26.8, outside the public payload.
4. Run `sh scripts/check.sh fast`; run `heavy`, `vulnerability` and `release`
   profiles before proposing a release. Heavy uses race detection and 1,000
   generated cases; fast uses 100. Replay seed is 30404 and failures retain the
   minimized counterexample. Vulnerability scanning needs the public advisory
   database; ordinary conformance is offline after provisioning dependencies.
5. Commit candidate changes before composition/release checks. These inspect a
   clean exact Git tree. A history-free exported directory is inspected directly.
   Ordinary unit tests can run while editing: `go test -mod=readonly ./...`.
6. Open a PR and link the issue/proposal. Required checks are AOM / policy,
   conformance, docs, supply-chain and aggregate. Main/scheduled/release checks
   additionally include heavy and reproducibility. Missing, stale, failed,
   skipped, neutral or canceled required checks block merge.
7. Obtain independent current review, and owner review for normative/authority
   changes. A maintainer merges manually after all requirements hold. A new head
   requires new evidence and review.

Public commands work from the public module root on Linux and macOS, with no
private repository, credential or service. Windows is not an admitted local
profile. The public checker cannot authenticate private editorial evidence or
prove host protection settings. Host admission remains a separate prerequisite.

Use [private security reporting](../SECURITY.md) for vulnerabilities. See
[governance](../GOVERNANCE.md) and [versioning](../VERSIONING.md).

## Build environment

Public CI runs directly on GitHub-hosted ubuntu-24.04 runners. Provisioning
verifies the pinned Go archive and builds pinned checking tools with that Go
version. It persists the verified tool paths for subsequent workflow steps.
Supply-chain checks cover Go dependencies and checking tools. No workload
container or OCI scanner is required. Runner OS maintenance belongs to GitHub;
the managed runner environment is not an immutable input controlled by this
repository. The run's runner image identity remains part of its provenance.
