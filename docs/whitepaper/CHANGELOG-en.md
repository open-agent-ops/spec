# Agent-Ops. White Paper Edition History

Build/Test correction (aom-03-r12): embed PNG figures, stabilize H1/H2 bookmarks and multi-pass contents, keep captions and table headings together; pin approved pybind11 and conditional typing_extensions dependencies. Fresh exact-target review and complete Build/Test remain required.

Build/Test correction (aom-03-r11): align two explicit MAY markers with the Russian edition; restore the missing Russian §17.1 roles table; correct normative-token, complete-tree link and per-image accessibility validation. Fresh exact-target external review and PDF validation remain pending.

Editorial review revision (aom-03-r10): incorporate the external editor corrections, Russian-first terminology guidance, and eight paired infographics; update the public composition without changing the three-pair normative set. PDFs and independent exact-target semantic review remain pending.

Publication-scope correction (aom-03-r9): retain exactly three normative EN/RU pairs; exclude Russian-only guides and raw review records; distinguish provenance from instruction admission. PDFs and independent exact-target semantic review remain pending.

A companion to "Agent-Ops. A Methodology for AI-Agent-Based Infrastructure Operations and Technical Support," public edition EN.

The white paper itself is normative. This file records what changed between editions; it creates, modifies, and revokes no contract. A documented `target` does not raise implementation status: a capability becomes `current` only after implementation and commit-bound validation evidence.

| Edition | Summary |
| --- | --- |
| 0.4.0 | Three-plane lifecycle structure plus scoped shadow promotion, Controlled Change attempt budgets and circuit breakers, agent-runtime observability, reproducible context assembly, and exact-integration admission. |
| 0.3.3 | Closed bilingual normative set, explicit English precedence, paired methodology/maps/glossary/diagrams, deterministic PDF toolchain contract, and external semantic-review gate. |
| 0.3.2 | Agent skills added to the Context Supply Chain; agent artifact inventory, admission and outcome receipts, coverage record, and a conformance mapping to external standards. Requirements sourced from the OWASP Agentic Skills Top 10. |
| 0.3.1 | Observability Readiness promoted to `current foundation`; the normative contracts of edition 0.3.0 were unchanged. |
| 0.3.0 | Operational Intent and Outcome Contract, Unified Evidence Bundle, separated R/C/I/S/M taxonomies, Validation Spine, Policy Hooks, and Agent Context Lifecycle. |

The English edition moved directly from 0.3.0 to 0.3.2 and carries the 0.3.1 changes listed below.

## Edition 0.4.0 | September 2026

- Restructured the White Paper around the data, governance and policy, and independent-assurance and trust planes while preserving the complete `Intent -> Evidence -> Diagnosis -> Plan -> Approval -> Controlled Change -> Verification -> Learning` lifecycle.
- Added case-scoped Controlled Change attempt budgets and circuit breakers: failed or `unknown` Verification consumes an attempt, a revised plan does not reset the budget, and an open breaker blocks Executor authority and escalates.
- Added `ShadowEvaluationRecord` and promotion scoped to an exact action class, environment, impact band, and agent/model/toolchain versions. Metrics cannot self-promote a capability; an explicit human or policy decision remains required.
- Added agent-runtime observability and an external watchdog in a separate failure domain with no execution authority.
- Added machine-readable `ContextAssemblyProfile` / `ContextManifest` semantics and preserved retrieval output as derived, untrusted evidence.
- Added exact connector/version/environment/operation-class integration admission without introducing another maturity scale.
- Clarified that typed specialist agents are optional topology; typed I/O, authority isolation, and Guardian independence are normative.
- Inserted shadow operations into phased adoption, added promotion-readiness measurement, updated the practical scenario, and added the Ganesh Gurudu SRE-agent article as an informative implementation source.
- Renamed the paired White Paper candidate files from v0.3.3 to v0.4.0. After completed owner review, the candidate is formalized as source revision `aom-03-r6`; its immutable successor target, external bilingual review, and generated PDFs remain pending.
- Advanced the shared candidate identity to source revision `aom-03-r7` after paired English and Russian proofreading corrected grammar, terminology, punctuation, and clarity without changing the v0.4.0 publication set or design. A new immutable target, review packets, external bilingual review, and generated PDFs remain pending.
- Advanced the shared candidate identity to source revision `aom-03-r8` after a paired agent-safety revision added model-node handoff assurance, admitted identifiers, serialized conflicting mutation, deterministic untrusted-context demarcation, and constructed-state evaluation. The source is informative and creates no product-stack dependency. A new immutable target, review packets, external bilingual review, and generated PDFs remain pending.
- Removed the PDLC map pair, the redundant standalone methodology pair, and the software-delivery conformance section from the normative publication after owner review identified an unintended conflation of Agent-Ops with AI-DLC.
- Removed the standalone diagram pair because its four selected diagrams did not form a coherent or complete visual model of Agent-Ops. Publication infographics will be designed separately after the White Paper content stabilizes.
- Rebuilt Section 20 around its three stated subjects: applied security controls for the exact operational case, an explicit non-transitive trust model, and the complete agent-context lifecycle from assembly through closure and disposal. Moved agent-artifact coverage records to the Section 16.5 inventory, removed duplicated governance/evaluation risks, and replaced software-implementation test techniques with runtime evidence of the no-mutation boundary.
- Rebuilt Section 21 around its three stated subjects: operational quality verification with applied check procedures, reproducibility of deterministic and model-assisted results, and full case-level AI economics. Removed the software-delivery test table and the separate ADLC loop; moved runtime observability and the external watchdog to the independent-assurance plane.

## Edition 0.3.3 | August 2026

- Established one closed EN/RU catalog with English semantic precedence and source revision `aom-03-r1`.
- Advanced the source revision to `aom-03-r2`; localized Russian normative markers as `ДОЛЖЕН`, `НЕ ДОЛЖЕН`, `СЛЕДУЕТ`, `НЕ СЛЕДУЕТ`, and `МОЖЕТ` without changing English precedence or normative strength.
- Advanced the source revision to `aom-03-r3`; restored the complete v0.3.2 methodology, sections and appendices as the v0.3.3 baseline, retained the bilingual publication and exact-target conformance additions, and restored the complete Russian pair.
- Advanced the source revision to `aom-03-r4`; retained the full methodology body while moving the controlled glossary and external standards mapping to their planned standalone bilingual files, removing the product-specific projection, and expressing implementation status without a vendor command or internal closure identifier.
- Advanced the source revision to `aom-03-r5`; clarified the standalone Appendix B link and verifiable Appendix C sources, measurement methods, planning-only remediation, unknown and explicit-network semantics, repeated Russian terminology explanations, Outcome production during Verification, and a mandatory append-only trace for every lifecycle transition, decision, and material engineer or agent message.
- Added paired methodology, PDLC map, standards map, glossary, and four diagrams with complete text alternatives.
- Renamed the current Russian white paper without the obsolete `Generic` suffix.
- Added deterministic PDF build and bookmark-verification contracts; rendered v0.3.3 PDFs remain Build/Test outputs.
- Kept semantic acceptance pending for an owner-designated external bilingual reviewer on one immutable candidate target.

## Edition 0.3.2 | August 2026

- Agent skills (SKILL.md bundles with accompanying resources, which an agent loads into context and executes with its own permissions) are included in the Context Supply Chain alongside MCP servers and prompt libraries. Until this edition the context supply chain did not name skills, although a skill is a distinct form of distributable artifact with its own publication, installation, and update path.
- Added section 8.5, Agent Artifact Inventory, as a Harness area.
- Added section 13.1, Admission and Outcome Receipts: two linked signed records for every Policy Hook decision and every execution. A `deny` with no execution also produces a signed receipt.
- Added section 16.3, Coverage Record: a scan of an agent artifact must publish a machine-readable record of what was examined, with the outcomes `pass`, `fail`, and `incomplete`. Zero findings do not read as `pass` when coverage is incomplete.
- Section 7 extended to agent artifact metadata: declared permissions and risk tier are declared state and are reconciled against observed behavior.
- Four rows added to the section 16 risk table: the skill and tool supply chain; loss of declared constraints when an artifact is ported between runtimes; injection resistance as a property of the executing model rather than the artifact; and repository configuration files as an execution path. The prompt/context injection control was tightened for agent chains.
- Implementation statuses in Appendix G reconciled with delivery evidence: Project Operations Harness moved from `planned` to `current foundation` based on exact-target delivery evidence, and a Foundation Contract Plane (F01-F04) row was added as `current foundation`. The English edition also carries the 0.3.1 promotion of Observability Readiness.
- Added Appendix J, External Sources and Conformance Mapping: the methodology mapped to ISO/IEC 42001, Regulation (EU) 2024/1689, and the OWASP risk catalogs, with primary-source links.
- Requirements sourced from the OWASP Agentic Skills Top 10 (AST01-AST10), version 1.0, 2026 - <https://owasp.org/www-project-agentic-skills-top-10/>. The added contracts carry `target` status and raise the implementation status of no component.

## Edition 0.3.1 | August 2026

- Observability Readiness promoted from `planned` to `current foundation`: offline readiness assessment from the artifacts of an accepted run is implemented (UOW-I3-04). A readiness status scale was added, along with the separation of authority between observed inventory, declared intent, and readiness.
- Deep Observability Discovery Audit added to Appendix G with `future` status.
- The change concerns implementation status only; the normative contracts of edition 0.3.0 were unchanged.

## Edition 0.3.0 | July 2026

- Operational Intent and Outcome Contract: a verifiable goal and completion criteria.
- Unified Evidence Bundle: one content-addressed chain of facts, decisions, and outcomes.
- Separated taxonomies: autonomy R0-R5, capability C0-C5, impact I0-I5, severity S0-S5, and process maturity M0-M6.
- Validation Spine, four eval classes, and a versioned Evidence Bundle lifecycle.
- Policy Hooks, Agent Context Lifecycle, Session Handoff, and an independent Guardian/evaluator.
- Restored normative result, package/CVE, Kubernetes, and eBPF contracts.
- Package and update semantics are a breaking normative correction: new runtime schemas require a new version and an explicit migration, while historical artifacts remain immutable.
- Quality, cost, and accepted-outcome metrics for AI operations.
- A refined roadmap where foundational governance precedes agent workflows.
