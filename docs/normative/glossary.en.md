# Agent-Ops Glossary
## Controlled vocabulary v0.5.0

**Source revision aom-04-r1 | English precedence**

> Literal identifiers and schema values stay unchanged across languages; explanations may be localized.

## 1. Core terms

| Term | Controlled meaning |
| --- | --- |
| Agent-Ops | An operations methodology combining deterministic evidence, AI analysis, and gated change. |
| Operational Intent | A versioned goal, scope, constraints, success criteria, and author identity; it is not an execution command. |
| authority | Explicit permission for an actor, action, scope, and time. |
| capability | Technical ability, which does not itself grant authority. |
| exact target | Immutable content identity evaluated by evidence. |
| evidence | Verifiable, target-bound fact or source reference with producer and result identity. |
| Evidence Bundle | An immutable package of identity, evidence, decisions, and outcome references. |
| admitted identifier | A target, resource, or object identifier issued or authoritatively confirmed by a non-model system, admitted into the current target-bound Evidence Bundle by a deterministic resolver, and fresh at the point of use. |
| oracle | A declared rule that evaluates an obligation. |
| unit of work | A bounded change with design, implementation, verification, and closure. |
| Harness | A versioned project-specific Operations-as-Code package. |
| Governance Mesh | A cross-cutting policy layer that validates process transitions. |
| Autonomy R0-R5 | A risk-adaptive permission ladder from read-only to autonomy in an isolated sandbox. |
| Capability C0-C5 | The Agent-Ops capability ladder: Observe, Explain, Plan, Prepare, Apply, and Self-Heal. |
| Impact I0-I5 | A separate scale for operational impact and required control. |
| Guardian | An independent evaluator with no execution authority. |
| Outcome | A record produced during Verification from exact-target after-state evidence, stating whether the Intent success criteria `succeeded`, `failed`, or remain `unknown`; Learning consumes this record. |
| RCA | Evidence-based ranking of testable hypotheses, not an unconditional assertion. |
| Remediation Planner | A component that prepares a change proposal without applying it. |
| Gated Executor | A separate executor for approved actions using scoped credentials. |
| cooperating writer | A writer whose admission, conflict, version-precondition, idempotency-key, and fencing behavior is enforced by the profile's declared gateway or backend domain. |
| enforcement domain | The exact set of writers, targets, operations, dependencies, and backend paths over which a named control can compel or reject behavior. |
| backend admission | Acceptance of a typed mutation by a cooperating gateway or backend at its declared linearization point after checking the bound authority predicate. A pre-call check alone is not backend admission. |
| decision-critical version | A target or dependency version whose change can invalidate the evidence, policy result, approval, or safe action selected for one decision. |
| clock uncertainty | A declared bound around reported time; validity checks conservatively use the earliest possible expiry and latest possible `not_before`. |
| effect knowledge | The classified knowledge `applied`, `not_applied`, `partial`, or `unknown` about target changes attributable to an attempt, together with observation coverage. It is distinct from attempt state and Outcome. |
| unresolved effect | An admitted operation whose actual effect is `unknown` or `partial`, or whose required reconciliation, compensation, observation, or Outcome obligation is not terminally resolved. |
| trusted computing base (TCB) | The components and people whose correct enforcement or observation is required for a stated guarantee; a TCB entry names its boundary, failure assumptions, independence, and claim limit. |
| common-cause failure | One dependency or false fact capable of defeating controls presented as separate; shared models, evidence, credentials, canonicalizers, clocks, services, and observation paths must be disclosed. |
| admission safety | The property that no operation is admitted at the declared enforcement point unless current authority holds for the exact package, target, attempt, epochs, and use time. |
| recovery | Authorized terminal resolution of every admitted obligation and required service or test-state restoration under stated availability and reversibility assumptions; a safe stop alone is not recovery. |
| Decision Log | An append-only record of decisions, approvals, expiry, evidence references, and outcomes. |
| Process Trace | An append-only ordered record of every lifecycle step entry and exit, transition attempt, decision, and material engineer or agent message, bound to its actor, authority, evidence, output, and exact target. |
| Agent skill | A `SKILL.md` bundle with accompanying resources that an agent discovers, loads into context, and executes with its own permissions. |
| Skill manifest | Declared identity, permissions, dependencies, and risk tier of a skill. |
| Risk tier | A declared risk class; an author's assertion subject to independent validation. |
| Agent artifact inventory | The list of skills, tools, and MCP servers permitted to load. |
| Backbone model | The model that actually executes an artifact at a given node. |
| Injection resistance | Resistance to executing instructions that arrive as data; a property of the executing model, not of the artifact. |
| Admission receipt | A signed record of a decision, produced before a potentially dangerous step. |
| Outcome receipt | A signed record of an execution result, linked to its admission receipt. |
| Coverage record | A record of what a scan examined, what it skipped, and where it was interrupted. |
| Manifest stripping | Loss of declared constraints when an artifact is ported between runtimes. |
| mutation conflict domain | A policy-defined set of overlapping target-mutation scopes whose attempts are mutually serialized until Verification or terminal release. |
| constructed-state evaluation | An evaluation that starts from an exact validated prior state, appends one stimulus, and grades the resulting artifacts and decision; it does not prove the transitions that created the prior state. |

## 2. Result terms

`ok` means all blocking inputs are known and satisfied. `unknown`, `partial`, `stale`, `mismatched`, `conflicting`, `unauthorized`, `unreviewed`, `blocked`, and `error` are distinct non-success results and MUST NOT be collapsed into success.

## 3. Publication terms

`agent-ops.ru` is the public methodology identity. `open-agent-ops` is a repository name. `source_revision` binds bilingual members to the same English source state.

## 4. Role terms

An implementer creates the bounded change. A verifier evaluates an exact target. A bilingual reviewer evaluates semantic equivalence. An accountable owner makes the closure decision. One person MAY hold several roles only where the applicable separation rule permits it.

## 5. Translation rule

Translations MUST preserve literal identifiers, enum values, schema names, digests, and normative force. In Russian prose, every occurrence of an English term MUST carry a Russian explanation in parentheses, not only its first occurrence. Backticked literals, registered product names, source titles, and recognized acronyms are exempt only when translation would corrupt identity; the surrounding prose still explains their role. If the English and Russian meanings conflict, English controls until an approved successor revises both members.
