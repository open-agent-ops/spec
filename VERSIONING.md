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

Stage 2 relation catalogs and the small semantic interface between the two
profiles are independently versioned. Their acceptance does not silently
change Evidence Bundle v1. A stronger bundle with incompatible acquisition or
sufficiency semantics requires a new schema identity or major version and an
explicit migration note.

Release input is stable SemVer `vMAJOR.MINOR.PATCH`, with no leading zeroes,
prerelease or build suffix. This module admits major 0 or 1. Major 2 requires a
separately reviewed module-path migration. Changes to normative text, schema
identity or compatibility require an accepted owner proposal and current review.
English is normative; English/Russian impact and editorial status are explicit.

An exact reviewed main SHA is bound to one release run and proposal. Rebuilding
or changing a version, source, asset, lock or policy invalidates approval. Tags
and assets are immutable: a differing existing object is a conflict. Recovery
uses a previous immutable release or a new reviewed fix, never a moved tag.
