# Bounded conformance profiles

The public package exposes three independently versioned offline validators:

- `agent-ops.relation-validation@1.0.0` validates the bounded C4 record slice;
- `agent-ops.evidence-relation-validation@1.0.0` validates the bounded Evidence Acquisition record slice;
- `agent-ops.integrated-conformance@1.0.0` composes those exact results through `agent-ops.c4-evidence-interface@1.0.0` without changing either component catalog.

The integrated result reports local syntax, C4 relation validity, Evidence relation validity, consumer-specific Evidence sufficiency, cross-boundary validity, and the consuming decision separately. `allowed` requires every component result and the interface projection to pass. A malformed component produces `unknown`; a complete but invalid component produces `blocked`. No aggregate score can override a blocking finding.

The five paired cross-boundary fixtures cover an Executor response used as effect proof, `not_applied` under partial coverage, Outcome success from stale or insufficient Evidence, retry from an incomplete negative observation, and checkpoint plus supersession hiding acquisition failure and an unresolved effect. The tests also check order invariance, monotonic propagation of `unknown`, malformed-input safety, and stable catalog pins.

`agent-ops.c4-evidence-evaluation@1.0.0` freezes the pre-runtime evaluation and reproducibility protocol. It pins the profiles, relation catalogs, validators, interface, deterministic seed, complete assigned-case denominator, named barriers, primary and secondary metrics, required records, required artifacts, blocking rule, and claim limit. The protocol records how a future separately authorized evaluation must be measured; it contains no runtime result and authorizes no collector, Executor, pilot, deployment, or release.
