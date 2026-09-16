# Agent-Ops v0.5.0 release-candidate notes

Status: public release candidate, source revision `aom-04-r7`. Repository
presence is not a release. Exact-target owner and bilingual review, complete
release and reproducibility evidence, and separate authorization for the
protected release environment remain required.

## Included

- independently versioned limited C4 Human-approved Apply and Evidence
  Acquisition profiles;
- two bounded executable formal models, independently versioned relation
  catalogs, and their versioned semantic interface;
- six C4 and five Evidence Acquisition schemas, bounded component validators,
  paired negative fixtures, and the composed offline conformance checker;
- a frozen pre-runtime evaluation and reproducibility protocol;
- paired English and Russian source documents and deterministic PDF renderings
  of source revision `aom-04-r7`.

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
Bundle v2 records; it does not rewrite or promote v1 bytes. Existing candidate
C4 records must be regenerated against the current pinned schema hashes. Source
revision `aom-04-r7` introduces no additional schema, profile, catalog,
interface, Go-module, runtime, or tag migration.
