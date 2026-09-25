# Agent-Ops v0.5.1 release-candidate notes

Status: public release candidate, source revision `aom-04-r8`. It supersedes
released edition v0.5.0 (`aom-04-r7`). Repository presence is not a release.
Exact-target owner and bilingual review, complete release and reproducibility
evidence, and separate authorization for the protected release environment
remain required.

## Changed in v0.5.1

- related work relating Agent-Ops to NIST SP 800-207 and to Teleport's
  *From Zero Trust to Agent Trust* (2026), without a superiority claim;
- agent objective or behavioral drift as a named threat, with a session
  baseline, a governed session stop, and a session-stop signal;
- explicit rejection of agreement among agents or a meta-agent assessment as
  independent verification;
- policy-level aggregate bounds on an action class within a target domain;
- attestable subject identity bound to the grantor and harness version, and a
  session-scoped reasoning-component runtime;
- two Appendix C sources, a derived Teleport Agent Trust section in the
  standards map, and three glossary terms.

The new requirements are `target`: they add no schema, validator, runtime, or
other implementation slice.

## Included

- independently versioned limited C4 Human-approved Apply and Evidence
  Acquisition profiles;
- two bounded executable formal models, independently versioned relation
  catalogs, and their versioned semantic interface;
- six C4 and five Evidence Acquisition schemas, bounded component validators,
  paired negative fixtures, and the composed offline conformance checker;
- a frozen pre-runtime evaluation and reproducibility protocol;
- paired English and Russian source documents and deterministic PDF renderings
  of source revision `aom-04-r8`.

## Explicit non-goals

This candidate does not provide a collector or Executor runtime, backend
adapters, a complete lifecycle validator, production validation, a runtime or
pilot evaluation result, source authentication, universal conformance, or a
production-safety guarantee. Synthetic fixtures and bounded models cannot
establish those claims.

## Compatibility and migration

Existing Evidence Bundle v1 bytes and their local contract remain unchanged.
A producer that needs the stronger acquisition and sufficiency semantics
creates new requirement, profile, receipt/artifact, assertion, and Evidence
Bundle v2 records; it does not rewrite or promote v1 bytes. Source revision
`aom-04-r8` changes no schema, profile, catalog, interface, fixture, Go-module,
runtime, or tag identity; records valid under v0.5.0 need no migration.
