# Compatibility

Compatibility is evaluated against the public contract and independent version axes.

White Paper v0.5.0 supersedes the v0.4.0 text candidate. It introduces two
independently versioned profiles, `agent-ops.c4-human-approved-apply@1.0.0`
and `agent-ops.evidence-acquisition@1.0.0`; it does not change the meaning or
identity of an existing JSON Schema, the Go module, or a release tag.

Profile 1.0.0 intentionally rejects interpretations in which compensation alone
proves retry safety, lease expiry discards an admitted unresolved effect, TTL
alone proves freshness after a decision-critical version changes, a checkpoint
resets attempts or pending obligations, or a pre-call check atomically authorizes
a non-cooperating remote backend. An implementation depending on any of those
interpretations is not compatible with this profile.

The Evidence Acquisition profile rejects interpretations in which successful
transport, a digest, a signature, bundle sealing, or an aggregate quality score
proves source authority, complete acquisition, claim support, evidence
sufficiency, or execution authority. Sufficiency is consumer-specific and
missing blocking quality axes remain unknown or incomplete.

The Stage 1 text fixes scope, assumptions, claim vocabulary, TCBs, quality
axes, and the coordinated formal-model plan. It supplies no complete runtime
conformance target. Stage 2 requires two compositional models, a small versioned
semantic interface, and independently versioned C4 and Evidence relation
catalogs. Stage 3 schemas and relation validators require later accepted
changes. Existing schema resource IDs retain their current meanings, including
`foundation_evidence_bundle` v1; an incompatible field or semantic change
requires a new schema revision or ID and an explicit migration note.
