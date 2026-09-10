# Governance

The intended GitHub teams are `open-agent-ops/maintainers` and
`open-agent-ops/owners`. Host admission must verify their actual membership,
immutable account IDs, access and at least two distinct people before this
process is opened to contributors. Team names in source do not prove host setup.

Maintainers triage bugs, fixes and proposals with a reason, responsible role and
next action. Outcomes are received, needs information, proposed, accepted for
work, in review, changes requested, approved current, merged or rejected.
CI supplies observations; humans make semantic decisions and merge manually.

At least one reviewer independent of the author must approve the current head.
An unresolved change request blocks merge. New commits invalidate previous
approvals and checks. Normative text, schemas, compatibility/version policy,
governance, CODEOWNERS, workflows, process policy, checker and publisher changes
also need a current independent owner review and a linked accepted proposal.
Contributor labels cannot downgrade a change's classification.

Protected base-branch CODEOWNERS and required host rules enforce review. Main must
require a current PR, resolved conversations, code-owner approval, approval of the
latest push and the exact required successful checks. Direct writes, force pushes,
deletions, self-approval, automatic merge and unreviewed bypass are prohibited.
Fork workflow approval authorizes compute only. Host enforcement is separately
verified before publication; local fixtures do not prove it.

Release decisions bind an immutable candidate, version, artifact hashes, workflow
run and tool lock. An independent owner approves the protected `public-release`
environment; the initiator cannot approve their own run. Mirror access belongs to
a separate `public-mirror` environment. See [the release guide](docs/releasing.md).
