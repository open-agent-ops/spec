# C4 and Evidence formal semantics

This directory contains the executable formal specification for
`agent-ops.c4-human-approved-apply@1.0.0` and
`agent-ops.evidence-acquisition@1.0.0`. It contains two separate finite-state
models connected only by `agent-ops.c4-evidence-interface@1.0.0`.

The package is not a production gateway, collector, or backend adapter. The
separate `conformance/` package and `schemas/` directory contain the bounded C4
schema and relation-validation slice. Neither artifact set changes Evidence
Bundle v1 or establishes full runtime conformance or production safety.

## Versioned contracts

| Contract | Public identity | Content |
| --- | --- | --- |
| C4 relations | `agent-ops.c4-relations@1.0.0` | Package, authority, admission, budget, fencing, retry, effect, Outcome, obligation, checkpoint, and release relations. |
| Evidence relations | `agent-ops.evidence-relations@1.0.0` | Applicability, collection, receipt, artifact, lineage, assertion, negative evidence, freshness, independence, completion, sealing, sufficiency, and invalidation relations. |
| Cross-profile interface | `agent-ops.c4-evidence-interface@1.0.0` | Applicability, exact target/version binding, freshness at use, coverage and detection capability, assertion disposition, consumer-specific sufficiency, provenance, and common-cause disclosure. |
| Counterexamples | `agent-ops.formal-counterexamples@1.0.0` | Retained minimal traces tied to named removed assumptions and violated properties. |

Each catalog relation names the authority that supplies the fact, the exact
check time, and the failure disposition. Evidence interface facts never grant
approval or execution authority. C4 authority never establishes evidence
truth, completeness, freshness, or sufficiency.

## Model boundaries

The Evidence model preserves requirement and profile binding, every attempt and
receipt, acquisition failures, artifact integrity and derivation lineage,
coverage and detection capability, source authority, freshness, assertion
disposition, common causes, consumer-specific sufficiency, and sealing. A new
attempt invalidates the preceding attempt's derived artifact and assertion in
this bounded model. Retry never overwrites the earlier receipt.

The C4 model preserves canonical package and exact target binding, policy and
approval epochs, bounded authority time, separate approval and human start,
reservation and fence state, attempt budget, admission, effect knowledge,
Outcome, reconciliation or compensation obligations, checkpoint refinement,
and terminal release. Admission, effect knowledge, and Outcome remain separate
states. Compensation is a new effect and does not make an unsafe retry safe.
Same-key replay preserves the existing attempt without consuming budget or
creating another obligation.

Safety properties are P1-P7 from Appendix G. L1 is checked as existence of an
authorized resolution path under C4-A14 availability and C4-A15 reversibility;
the bounded checker does not claim unbounded liveness. `C4-G03` separately
records an unsafe retry counterexample. Evidence properties E1-E7 reject stale
TTL-only reuse, incomplete negative evidence, hidden acquisition loss,
unauthorized sources, false independence, non-monotonic attempt lineage, and
the use of C4 authority as evidence truth.

## Reproducible checker record

The checker is deterministic exhaustive exploration, not random simulation. It
uses Go 1.26.8 from `process/toolchain.lock.json`, depth 8, the finite event sets
in `check.go`, and no seed. Run:

```text
go test -mod=readonly -run TestBoundedModels -v ./formal
go test -mod=readonly -run TestRetainedCounterexamples -v ./formal
```

On this candidate before publication, the default-assumption exploration
visited 332 Evidence states over 2,325 accepted transitions and 1,233 C4 states
over 6,014 accepted transitions, with no P1-P7, E1-E7, or C4-G03 violation.
The retained corpus contains the required assumption-removal and cross-boundary
counterexamples, including lost response plus partial observation plus retry,
target ABA plus stale evidence, and a checkpoint that omits both an acquisition
failure and an unresolved effect.

These bounded results can falsify a universal claim when they find a trace.
Their success does not prove an implementation, an unbounded environment, or a
production system safe. The published C4 schemas and relation validators cover
only the named offline cross-record slice; runtime gateway, effect observation,
production evaluation, and complete lifecycle conformance remain outside it.
