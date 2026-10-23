# Agent-Ops v0.5.2 release-candidate notes

Status: public release candidate, source revision `aom-04-r9`. It supersedes
white paper edition v0.5.1 (`aom-04-r8`). Repository presence is not a release.
Fresh exact-target independent owner and bilingual review, complete release and
reproducibility evidence, and protected-environment authorization remain required.

## Changed in v0.5.2

- align the executive summary with existing case-completion rules (§§1, 12.3);
- define a bounded case and its organizational interfaces (§3);
- add an informative small-team example, preserving independent review and
  the limited non-production C4 domain (§23.1);
- explain the adoption roadmap without mandatory autonomy growth (§25);
- add informative human-led continuation after a lost response, retaining
  current authority and execution controls (§20.4);
- explain full adoption costs and expand Cost-to-Outcome Ratio (CTOR), without
  changing its calculation or the completion-evidence requirements (§21.4).

The public verification toolchain advances from Go 1.26.8 to 1.26.10 to address upstream standard-library vulnerability findings, including GO-2026-6599 and GO-2026-6809. The checking-tool versions, module compatibility minimum, formal exploration bounds and public contract semantics are retained.

No PSLC comparison or integration claim is added. Figure edition labels are updated to v0.5.2; the figure content is retained.

## Included and claim limits

The candidate retains the independently versioned C4 and Evidence Acquisition
profiles, formal models, schemas, relation catalogs, semantic interface,
integrated offline checker, and frozen pre-runtime evaluation protocol.
These artifacts do not supply a collector or Executor runtime, a complete
lifecycle validator, source authentication, runtime or pilot evaluation,
production validation, or a production-safety guarantee. C4 retains its
isolated, resettable, non-production reference domain. Human approval and a
separate human start of the deterministic Executor remain required.

The paired English and Russian PDFs render source revision `aom-04-r9`;
their bounded deterministic Build/Test evidence does not authorize release or
replace independent exact-target review.

## Compatibility and migration

Profiles, schemas, relation catalogs, interface, formal models, fixtures,
Go-module, and release-tag versions are unchanged. Existing records require
no migration. Evidence Bundle v1 remains unchanged; stronger acquisition and
sufficiency semantics continue to require new v2 lineage under the existing
migration rules. Informative examples waive no authority, independent-review,
unknown-effect, attempt-budget, or completion-evidence requirement.
