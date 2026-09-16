# Versioning

Document, schema, contract, Go module and release versions are separate dimensions.
Text v0.5.0 does not imply a released module or change every schema revision.
See [compatibility](COMPATIBILITY.md) for the current schema family versions.

Profile identifiers are independently versioned contracts of the form
`agent-ops.<profile-name>@MAJOR.MINOR.PATCH`. The limited C4 profile begins as
`agent-ops.c4-human-approved-apply@1.0.0`; the Evidence Acquisition profile
begins as `agent-ops.evidence-acquisition@1.0.0`. Changing the white-paper
edition, schema revision, Go module, release tag, or the other profile does not
silently change either profile.
An incompatible profile boundary, assumption, guarantee, enforcement domain, or
required relation needs a new profile major version and an accepted owner
proposal. Compatible additions and clarifications use the applicable minor or
patch version and still require the governed review for normative text.

The formal semantics fix independently versioned contracts
`agent-ops.c4-relations@1.0.0`,
`agent-ops.evidence-relations@1.0.0`, and
`agent-ops.c4-evidence-interface@1.0.0`. Changing one does not silently change
either profile, the other catalogs, or Evidence Bundle v1. An incompatible
relation, interface fact, authority, check time, or failure disposition needs a
new applicable major version and an accepted owner proposal. A stronger bundle
with incompatible acquisition or sufficiency semantics requires a new schema
identity or major version and an explicit migration note.

The bounded C4 implementation slice begins with six schema identities at
revision `1.0.0` and the independently versioned validator catalog
`agent-ops.relation-validation@1.0.0`. A change to required cross-record
binding, authority, temporal, retry, budget, checkpoint, effect, Outcome, or
completeness semantics requires the applicable new major version and an
accepted owner proposal. Compatible additions still require governed review.

The bounded Evidence Acquisition implementation slice begins with four record
schema identities at revision `1.0.0`, the incompatible
`evidence_bundle_v2.schema.json` identity at revision `2.0.0`, and
`agent-ops.evidence-relation-validation@1.0.0`. Evidence Bundle v1 retains its
existing identity and semantics. An incompatible change to applicability,
acquisition lineage, freshness, independence, completion, sealing,
invalidation, or consumer sufficiency requires the applicable new major
version and an accepted owner proposal.

The composed offline profile begins as
`agent-ops.c4-evidence-integrated@1.0.0`, its validator catalog as
`agent-ops.integrated-conformance@1.0.0`, and its frozen measurement contract as
`agent-ops.c4-evidence-evaluation@1.0.0`. The integrated catalog pins the exact
C4 validator, Evidence validator, and interface versions; it cannot redefine
their relations or reason codes. An incompatible change to a required interface
projection, consuming-decision rule, denominator, blocking metric, or claim
limit requires the applicable new major version and an accepted owner proposal.

Release input is stable SemVer `vMAJOR.MINOR.PATCH`, with no leading zeroes,
prerelease or build suffix. This module admits major 0 or 1. Major 2 requires a
separately reviewed module-path migration. Changes to normative text, schema
identity or compatibility require an accepted owner proposal and current review.
English is normative; English/Russian impact and editorial status are explicit.

An exact reviewed main SHA is bound to one release run and proposal. Rebuilding
or changing a version, source, asset, lock or policy invalidates approval. Tags
and assets are immutable: a differing existing object is a conflict. Recovery
uses a previous immutable release or a new reviewed fix, never a moved tag.

The v0.5.0 release candidate is source revision `aom-04-r7`. That revision is a
publication consolidation: it does not advance any profile, schema, catalog,
interface, Go-module, or release-tag version. The candidate becomes a release
only after exact-target review and release evidence satisfy the governed gates.
