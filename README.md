# Agent-Ops Specification

Vendor-neutral public foundation candidate for Agent-Ops v0.5.0. The public
identity is `agent-ops.ru`; `open-agent-ops` is the distribution repository
name. Repository presence does not mean that a candidate has been released.

English is normative. Documentation uses CC BY 4.0 (`LICENSE-docs`);
schemas, Go code, conformance and machine metadata use Apache-2.0 (`LICENSE`).
See `COMPATIBILITY.md` for the public namespace and live schema revisions.
From this module directory, run `go test ./...` after provisioning the pinned
Go toolchain and dependencies. The conformance suite runs offline.

[English white paper](docs/whitepaper/Agent-Ops_White_Paper_EN_v0.5.0.md) ·
[Русский white paper](docs/whitepaper/Agent-Ops_White_Paper_RU_v0.5.0.md)

## Normative publication set

The closed bilingual catalog is `bilingual-pairs.json`. It pairs the white
paper, standards map, and glossary.
The v0.5.0 candidate uses source revision `aom-04-r2` and supersedes the
v0.4.0 text candidate with Stage 1 of the independently versioned
`agent-ops.c4-human-approved-apply@1.0.0` and
`agent-ops.evidence-acquisition@1.0.0` profiles. Their Stage 2 models will
compose through a small versioned semantic interface while preserving separate
authority and evidence-quality domains. `review-summary.json`
retains historical review evidence for r12; it does not approve this candidate.
The two v0.4.0 PDFs retain predecessor bytes and do not render v0.5.0. Fresh
exact-target owner and bilingual review and new PDF Build/Test remain pending.
`composition-manifest.json` lists the full current public file set;
`publication-manifest.json` tracks only the six paired Markdown sources and two
PDF artifacts. Private review decisions are represented only by the safe
public projection; they are not distributed with this module.
English governance documents and paired changelogs are auxiliary, not
additional normative pairs. Russian-only guides and raw review records are
outside this publication set.

- English: [standards map](docs/normative/standards-map.en.md), [glossary](docs/normative/glossary.en.md).
- Russian: [карта стандартов](docs/normative/standards-map.ru.md), [глоссарий](docs/normative/glossary.ru.md).

Structural parity is machine-verifiable but is not semantic approval. The
v0.5.0 candidate remains pending until its complete conformance,
composition and supply-chain evidence is bound to its exact target.

## Public boundary

The specification defines contracts, evidence semantics, and lifecycle
controls. The offline specification and conformance core supplies no credentials or
inferred approval. The separate protected workflow executable implements an
explicit artifact publication capability; it is inactive until host admission. See the documentation
[license](LICENSE-docs), [compatibility](COMPATIBILITY.md),
[governance](GOVERNANCE.md), [security policy](SECURITY.md), and the
[conformance-oriented standards map](docs/normative/standards-map.en.md).

## Offline conformance

Use pinned Go 1.26.8 for public checks; Go 1.25.13 remains the compatibility
minimum. Provision the exact dependencies from `go.mod` and `go.sum`
with the standard verified Go download mechanism before disconnecting.
Then run `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go test ./...` with the
installed toolchain on PATH. A missing cache input is a prerequisite failure.
To replay generated cases, pass `-rapid.seed=30404 -rapid.checks=100` to
`go test ./...`; failure output retains the seed and minimized counterexample.
For the focused compiler and fixture suite, run `go test ./conformance`.

The registry embeds exactly 61 schemas and returns defensive copies. Lookup
uses full public resource IDs. Revision dispatch accepts the two family names
`foundation_object_ref` and `foundation_semantic_object` with exact versions
`1.0.0`, `1.1.0`, `1.2.0`, `1.3.0`. Text v0.5.0, the independently versioned
C4 and Evidence Acquisition profiles, and release tags are separate version
dimensions. New private
namespace aliases are rejected.

The suite compiles every schema, checks six positive and eighteen distinct
negative core fixtures, and includes four Run Request cases. Seven declared
properties are verified across public and private release checks; private
review-authentication and export checks are not part of this public module.
Public schema checks do not authenticate provenance, grant authority or prove
operational safety, full Agent-Ops implementation conformance or runtime readiness.
Fixed fixtures use synthetic references and digest values, never live targets.

## Contributing and releasing

Read the [contributor guide](docs/contributing.md) for issue/PR intake, local
checks and independent review, and the [release guide](docs/releasing.md) for
exact approved publication, mirror readback and immutable recovery. New public
process tooling is a corrective candidate; local fixtures do not establish
actual host readiness or an approved release.
