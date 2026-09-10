# Versioning

Document, schema, contract, Go module and release versions are separate dimensions.
Text v0.4.0 does not imply a released module or change every schema revision.
See [compatibility](COMPATIBILITY.md) for the current schema family versions.

Release input is stable SemVer `vMAJOR.MINOR.PATCH`, with no leading zeroes,
prerelease or build suffix. This module admits major 0 or 1. Major 2 requires a
separately reviewed module-path migration. Changes to normative text, schema
identity or compatibility require an accepted owner proposal and current review.
English is normative; English/Russian impact and editorial status are explicit.

An exact reviewed main SHA is bound to one release run and proposal. Rebuilding
or changing a version, source, asset, lock or policy invalidates approval. Tags
and assets are immutable: a differing existing object is a conflict. Recovery
uses a previous immutable release or a new reviewed fix, never a moved tag.
