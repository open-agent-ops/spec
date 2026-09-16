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

The profile contracts fix scope, assumptions, claim vocabulary, trusted
computing bases, quality axes, and the coordinated model boundary. Two bounded
compositional executable models use `agent-ops.c4-evidence-interface@1.0.0` and independently versioned
`agent-ops.c4-relations@1.0.0` and
`agent-ops.evidence-relations@1.0.0` catalogs. These artifacts can falsify a
property inside their stated bound; they supply no complete runtime conformance
target or production-safety proof.

The bounded C4 implementation slice adds six new schema identities for
`ChangePackage`, `ExecutionRecord`, authority/admission receipt, `Outcome`, C4
lifecycle event, and checkpoint, plus
`agent-ops.relation-validation@1.0.0`. It reuses `HumanDecision` and
`OutcomeCheck` by exact versioned reference and digest. These additions do not
change any existing schema resource ID, including `foundation_evidence_bundle`
v1, and do not establish full lifecycle or runtime conformance. An incompatible
field, relation, or semantic change requires a new schema revision or ID and an
explicit migration note.
