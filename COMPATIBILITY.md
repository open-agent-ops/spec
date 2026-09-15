# Compatibility

Compatibility is evaluated against the public contract and independent version
axes. This document names those axes, lists the live schema families and their
exact revisions, and defines the process terms that the governance documents
use without defining them elsewhere. English is normative.

## Version axes

| Axis | Current value | Where it is declared |
| --- | --- | --- |
| Normative text | v0.4.0, source revision `aom-03-r12` | `publication-manifest.json`, `revision-manifest.json` |
| Schema namespace | `https://agent-ops.ru/schemas/` | `$id` of every embedded schema |
| Schema revision | per family, see below | `schemas/inventory.go`, `Registry.Resolve` |
| Document format | `format_version` inside a document | the schema revision that admits it |
| Go module | `go 1.25.13` minimum, Go 1.26.8 pinned for public checks | `go.mod`, `process/toolchain.lock.json` |
| Release tag | stable SemVer `vMAJOR.MINOR.PATCH`, major 0 or 1 | [versioning](VERSIONING.md) |

No axis implies another. Text v0.4.0 does not change a schema revision, a
schema revision does not imply a release, and a release tag does not restate
the text version.

## Schema namespace

Every schema declares JSON Schema draft 2020-12 and a `$id` under
`https://agent-ops.ru/schemas/<name>.schema.json`. The embedded registry holds
exactly 61 schemas, verifies each against its recorded digest before use, and
answers lookups by full `$id` only. Relative, private or aliased identifiers are
not resolved. The registry digest is the identity of the whole schema set and
appears in every conformance result as `registry`.

`agent-ops.ru` is the public identity of the methodology. `open-agent-ops` is a
repository name. Process record schemas under `process/schemas/` use the
`https://open-agent-ops.org/process/` prefix; they describe repository process
records, not Agent-Ops objects, and are outside the public contract.

## Live schema families

Two families are versioned and dispatched by `Registry.Resolve(family, version)`.
Dispatch accepts only the exact versions below. A version that is not listed,
including a range or a prefix, is unknown. The two families select a revision
differently:

- `foundation_semantic_object` documents carry a revision discriminator. Each
  revision declares its own `format_version` constant, so a document is
  admitted only by the revision it names.
- `foundation_object_ref` documents carry no discriminator. The revision is
  selected out of band: a consumer pins it, or an embedding schema names it by
  `$ref`, and passes that version to `Registry.Resolve`. A reference that is
  valid under 1.0.0 is also valid under every later revision.

### `foundation_object_ref`

| Version | Schema | Change relative to the previous revision |
| --- | --- | --- |
| 1.0.0 | `foundation_object_ref.schema.json` | 12 `kind` values: `project`, `environment`, `service`, `owner`, `runbook`, `policy`, `tool_contract`, `evidence_record`, `evidence_bundle`, `operational_intent`, `human_decision`, `outcome_check` |
| 1.1.0 | `foundation_object_ref_v1_1.schema.json` | adds `decision_map`, `discovery_assessment` |
| 1.2.0 | `foundation_object_ref_v1_2.schema.json` | adds `completion_profile`, `eval_descriptor`, `eval_result` |
| 1.3.0 | `foundation_object_ref_v1_3.schema.json` | adds `governance_result` |

`id`, `namespace`, `version` and their patterns are unchanged across revisions;
`version` is the referenced object's version, not the schema revision.
`run_request.schema.json` references revision 1.0.0 by `$ref`.

### `foundation_semantic_object`

| Version | Schema | Change relative to the previous revision |
| --- | --- | --- |
| 1.0.0 | `foundation_semantic_object.schema.json` | `format_version` `1.0.0`; embedded object reference with the 12 kinds of `foundation_object_ref` 1.0.0; `source.size_bytes` at most 1 MiB |
| 1.1.0 | `foundation_semantic_object_v1_1.schema.json` | `format_version` `1.1.0`; kinds as `foundation_object_ref` 1.1.0 |
| 1.2.0 | `foundation_semantic_object_v1_2.schema.json` | `format_version` `1.2.0`; kinds as `foundation_object_ref` 1.2.0; `source.size_bytes` at most 8 MiB |
| 1.3.0 | `foundation_semantic_object_v1_3.schema.json` | `format_version` `1.3.0`; kinds as `foundation_object_ref` 1.3.0 |

### Other revised schemas

`harness_observability_ref.schema.json` (`schema_version` `1.0.0`) and
`harness_observability_ref_v1_1.schema.json` (`schema_version` `1.1.0`) are both
embedded and compiled but are not dispatched by `Registry.Resolve`; consumers
select them by full `$id`. Revision 1.1 is not a drop-in replacement for 1.0.
A document valid under 1.0 is rejected by 1.1 unless it is changed as follows:

| Location | 1.0.0 | 1.1.0 |
| --- | --- | --- |
| root `criticality` | absent | required, enumerated |
| root `freshness` | absent | required object with `max_age_days` (1..3650) |
| root `queries` | `log_query_ref`, `trace_query_ref`: single token each, both required | `log_query_refs`, `trace_query_refs`: arrays of tokens, both required |
| alert `paging` | absent | required boolean |
| alert `runbook_ref` | required local reference (no `..`) | optional token |
| alert `exception_ref` | absent | optional token |
| identifier definition | `identifier` | renamed `id`; same shape |

Adding the required fields and renaming the two query keys is sufficient to
move a 1.0 document to 1.1; nothing else valid in 1.0 is rejected.

All remaining schemas exist in a single revision under their unversioned `$id`.

## Compatibility rules

- Within a dispatched family, a later minor revision only adds enumeration
  values, properties or capacity to the payload constraints. It never removes
  or narrows a payload constraint of an earlier revision. The schema identity
  and, where present, the `format_version` discriminator are excluded from
  this guarantee: they change with every revision by definition.
- For `foundation_object_ref`, the additive rule means a reference valid under
  revision X is valid under every later revision. Consumers pin the revision
  they read and pass it to `Registry.Resolve`; an embedding schema fixes it by
  `$ref`.
- For `foundation_semantic_object`, a document that names `format_version`
  `X.Y.0` is validated against revision `X.Y.0` and against nothing else.
  Validating it against another revision is a consumer error, not a
  compatibility failure. A producer moving to a newer revision changes
  `format_version`; consumers that have not adopted it reject the document as
  unknown rather than reading it partially.
- `harness_observability_ref` revisions are independent contracts selected by
  `$id`; the table above lists what moves a document between them.
- Changing a schema `$id`, removing a revision, or changing what an existing
  revision admits is a normative change. It requires an accepted owner proposal
  and current review, and a new registry digest.
- Declared `format` keywords (`date-time`, `date`, `uri`) are assertions in this
  registry. Values that fail them are rejected.

## Process terms

These terms appear in the governance, contributor and release documents. The
[glossary](docs/normative/glossary.en.md) defines methodology vocabulary; the
terms below concern publication of this repository.

| Term | Meaning |
| --- | --- |
| host admission | The set of hosting-platform facts that source files cannot prove: team membership and immutable account IDs, branch and tag protection, required checks, protected environments and their reviewers, an active private security channel. The public checker treats every one of them as a separate prerequisite that a human verifies before the process is opened to contributors or a release is dispatched. |
| exact target | The immutable content identity a check or review binds to: a commit, tree, tag or file digest. A result about one exact target says nothing about another. See the glossary. |
| protected file | A path whose `protected_hash` is recorded in `process/policy.json`. The policy gate rejects a candidate whose bytes differ from the recorded digest, so such a file changes only together with its policy record. |
| safe public projection | The subset of a private review record that may be published: identifiers, digests, verdicts and revisions, never the review text, private paths or reviewer identity. `review-summary.json` is a safe public projection. |
| corrective candidate | Public process tooling published to repair or complete an earlier candidate. Its presence in the repository does not establish that the host is ready or that a release was approved. |
| candidate | A repository state under evaluation. `maturity`, `release_status` and `status` fields marked `candidate` or `pending` mean that evidence for that exact target is not yet complete, whatever tags exist. |

## What this document does not establish

Passing the public schema and conformance checks proves that documents match
the embedded contracts. It does not authenticate provenance, grant authority,
prove operational safety, or establish full Agent-Ops implementation
conformance or runtime readiness. Those remain host admission and review
prerequisites.
