package formal

import "errors"

type EffectKnowledge string

const (
	EffectUnknown    EffectKnowledge = "unknown"
	EffectApplied    EffectKnowledge = "applied"
	EffectNotApplied EffectKnowledge = "not_applied"
	EffectPartial    EffectKnowledge = "partial"
)

type OutcomeState string

const (
	OutcomeUnknown   OutcomeState = "unknown"
	OutcomeSucceeded OutcomeState = "succeeded"
	OutcomeFailed    OutcomeState = "failed"
)

type C4Event string

const (
	C4CreatePackage       C4Event = "create_package"
	C4Approve             C4Event = "approve"
	C4HumanStart          C4Event = "human_start"
	C4Reserve             C4Event = "reserve"
	C4Revoke              C4Event = "revoke"
	C4MutatePackage       C4Event = "mutate_package"
	C4TargetABA           C4Event = "target_aba"
	C4AdvancePastExpiry   C4Event = "advance_past_expiry"
	C4Admit               C4Event = "admit"
	C4BypassWriterEffect  C4Event = "bypass_writer_effect"
	C4LoseResponse        C4Event = "lose_response"
	C4ObserveApplied      C4Event = "observe_applied"
	C4ObserveNotApplied   C4Event = "observe_not_applied"
	C4ReplaySameKey       C4Event = "replay_same_key"
	C4Retry               C4Event = "retry"
	C4LeaseExpire         C4Event = "lease_expire"
	C4StaleWriterEffect   C4Event = "stale_writer_effect"
	C4CheckpointOmit      C4Event = "checkpoint_omit"
	C4Restart             C4Event = "restart"
	C4IntentSatisfied     C4Event = "intent_satisfied"
	C4OutcomeSuccess      C4Event = "outcome_success"
	C4OutcomeSuccessFalse C4Event = "outcome_success_false_intent"
	C4Reconcile           C4Event = "reconcile"
	C4Compensate          C4Event = "compensate"
	C4Release             C4Event = "release"
)

type C4Action struct {
	Kind     C4Event
	Evidence EvidenceFacts
}

type C4State struct {
	PackageDigest    string
	PackageTarget    TargetRef
	CurrentTarget    TargetRef
	ApprovedDigest   string
	ApprovedTarget   TargetRef
	PolicyEpoch      uint64
	ApprovalEpoch    uint64
	CurrentPolicy    uint64
	CurrentApproval  uint64
	NotBefore        int64
	NotAfter         int64
	UseTime          int64
	ClockUncertainty int64
	Approved         bool
	Started          bool
	Revoked          bool
	Reservation      bool
	Fence            uint64
	BackendFence     uint64
	Budget           uint8
	Consumed         uint8
	Admitted         bool
	Released         bool
	Effect           EffectKnowledge
	EffectSupported  bool
	Outcome          OutcomeState
	IntentOK         bool
	Obligations      uint8
	Reconciled       bool
	Compensated      bool
	P1Violation      bool
	P2Violation      bool
	P3Violation      bool
	P4Violation      bool
	P5Violation      bool
	P6Violation      bool
	P7Violation      bool
	RetryViolation   bool
}

func InitialC4State() C4State {
	return C4State{
		Budget:           2,
		Effect:           EffectUnknown,
		Outcome:          OutcomeUnknown,
		UseTime:          10,
		ClockUncertainty: 1,
	}
}

func ApplyC4(state C4State, action C4Action, assumptions Assumptions) (C4State, error) {
	next := state
	switch action.Kind {
	case C4CreatePackage:
		if next.PackageDigest != "" {
			return state, errors.New("package already created")
		}
		next.PackageDigest = "sha256:package-1"
		next.PackageTarget = TargetRef{UID: "target-1", Generation: 1, Epoch: 1, Versions: "resource=1;policy=1;profile=1"}
		next.CurrentTarget = next.PackageTarget
		next.PolicyEpoch = 1
		next.ApprovalEpoch = 1
		next.CurrentPolicy = 1
		next.CurrentApproval = 1
		next.NotBefore = 5
		next.NotAfter = 15
	case C4Approve:
		if next.PackageDigest == "" {
			return state, errors.New("package missing")
		}
		next.Approved = true
		next.ApprovedDigest = next.PackageDigest
		next.ApprovedTarget = next.PackageTarget
	case C4HumanStart:
		if !next.Approved || next.Revoked {
			return state, errors.New("approval unavailable")
		}
		next.Started = true
	case C4Reserve:
		if !next.Started {
			return state, errors.New("start missing")
		}
		next.Reservation = true
		next.Fence++
		next.BackendFence = next.Fence
	case C4Revoke:
		if !next.Approved {
			return state, errors.New("approval missing")
		}
		next.Revoked = true
	case C4MutatePackage:
		if next.PackageDigest == "" {
			return state, errors.New("package missing")
		}
		next.PackageDigest = "sha256:package-2"
	case C4TargetABA:
		if next.CurrentTarget.UID == "" {
			return state, errors.New("target missing")
		}
		next.CurrentTarget.Generation++
		next.CurrentTarget.Versions = "resource=2;policy=1;profile=1"
	case C4AdvancePastExpiry:
		next.UseTime = 20
	case C4Admit:
		if next.PackageDigest == "" {
			return state, errors.New("package missing")
		}
		authority := next.Approved && next.Started && !next.Revoked &&
			next.PolicyEpoch == next.CurrentPolicy && next.ApprovalEpoch == next.CurrentApproval
		packageBound := next.PackageDigest == next.ApprovedDigest && next.PackageTarget == next.ApprovedTarget &&
			next.CurrentTarget == next.PackageTarget
		timeValid := next.UseTime-next.ClockUncertainty >= next.NotBefore && next.UseTime+next.ClockUncertainty <= next.NotAfter
		reservationValid := next.Reservation && next.Fence > 0 && next.Fence == next.BackendFence
		budgetValid := next.Consumed < next.Budget
		if !authority && assumptions.Has(C4A03) {
			return state, errors.New("authority invalid")
		}
		if !packageBound && assumptions.Has(C4A04) && assumptions.Has(C4A18) {
			return state, errors.New("package or target mismatch")
		}
		if !timeValid && assumptions.Has(C4A10) {
			return state, errors.New("authority outside interval")
		}
		if !reservationValid && assumptions.Has(C4A02) && assumptions.Has(C4A07) {
			return state, errors.New("reservation or fence invalid")
		}
		if !budgetValid && assumptions.Has(C4A06) && assumptions.Has(C4A17) {
			return state, errors.New("attempt budget exhausted")
		}
		next.P1Violation = next.P1Violation || !authority || !reservationValid
		next.P2Violation = next.P2Violation || !packageBound
		next.P3Violation = next.P3Violation || !budgetValid
		next.P4Violation = next.P4Violation || !reservationValid
		next.P6Violation = next.P6Violation || !timeValid
		next.Consumed++
		next.Admitted = true
		next.Effect = EffectUnknown
		next.EffectSupported = false
		next.Obligations++
	case C4BypassWriterEffect:
		if assumptions.Has(C4A02) {
			return state, errors.New("out-of-domain writer excluded")
		}
		next.Effect = EffectApplied
		next.P1Violation = true
	case C4LoseResponse:
		if !next.Admitted {
			return state, errors.New("attempt not admitted")
		}
		next.Effect = EffectUnknown
		next.EffectSupported = false
	case C4ObserveApplied, C4ObserveNotApplied:
		if !next.Admitted {
			return state, errors.New("attempt not admitted")
		}
		disposition := DispositionSupported
		valid := action.Evidence.SupportsEffect(next.CurrentTarget, disposition)
		if !valid && assumptions.Has(C4A11) && assumptions.Has(C4A13) {
			return state, errors.New("effect evidence incomplete")
		}
		if action.Kind == C4ObserveApplied {
			next.Effect = EffectApplied
		} else {
			next.Effect = EffectNotApplied
			next.P5Violation = next.P5Violation || !valid
		}
		next.EffectSupported = valid
	case C4ReplaySameKey:
		if !next.Admitted {
			return state, errors.New("attempt missing")
		}
		// Replaying the same backend idempotency key observes the existing
		// attempt. It consumes neither a new attempt nor a new obligation.
	case C4Retry:
		if !next.Admitted {
			return state, errors.New("attempt missing")
		}
		budgetValid := next.Consumed < next.Budget
		retrySafe := next.Effect == EffectNotApplied && next.EffectSupported || next.Reconciled
		if !budgetValid && assumptions.Has(C4A06) && assumptions.Has(C4A17) {
			return state, errors.New("attempt budget exhausted")
		}
		if !retrySafe && assumptions.Has(C4A05) {
			return state, errors.New("retry safety unknown")
		}
		next.P3Violation = next.P3Violation || !budgetValid
		next.RetryViolation = next.RetryViolation || !retrySafe
		next.Consumed++
		next.Obligations++
		next.Effect = EffectUnknown
		next.EffectSupported = false
	case C4LeaseExpire:
		if !next.Reservation {
			return state, errors.New("reservation missing")
		}
		next.Reservation = false
	case C4StaleWriterEffect:
		if assumptions.Has(C4A05) && assumptions.Has(C4A07) {
			return state, errors.New("stale fence rejected")
		}
		next.P4Violation = true
		next.Effect = EffectPartial
	case C4CheckpointOmit:
		if next.Consumed == 0 && next.Obligations == 0 {
			return state, errors.New("nothing to omit")
		}
		if assumptions.Has(C4A06) && assumptions.Has(C4A17) {
			return state, errors.New("checkpoint is not a refinement")
		}
		next.Consumed = 0
		next.Obligations = 0
		next.P3Violation = true
	case C4Restart:
		// A conforming restart preserves the full state. The explicit omission
		// action above represents the minimized C4-A06/A17 counterexample.
	case C4IntentSatisfied:
		next.IntentOK = true
	case C4OutcomeSuccess, C4OutcomeSuccessFalse:
		if !next.Admitted {
			return state, errors.New("attempt not admitted")
		}
		intentOK := next.IntentOK && action.Kind != C4OutcomeSuccessFalse
		evidenceOK := action.Evidence.SupportsOutcome(next.CurrentTarget)
		if (!intentOK || !evidenceOK) && assumptions.Has(C4A12) {
			return state, errors.New("outcome evidence insufficient")
		}
		next.P7Violation = next.P7Violation || !intentOK || !evidenceOK
		next.Outcome = OutcomeSucceeded
	case C4Reconcile:
		if next.Obligations == 0 {
			return state, errors.New("obligation missing")
		}
		next.Obligations = 0
		next.Reconciled = true
	case C4Compensate:
		if next.Obligations == 0 {
			return state, errors.New("obligation missing")
		}
		next.Obligations = 0
		next.Effect = EffectPartial
		next.Compensated = true
	case C4Release:
		if next.Obligations != 0 {
			return state, errors.New("unresolved obligation")
		}
		next.Released = true
	default:
		return state, errors.New("unknown C4 event")
	}
	return next, nil
}

func C4Violations(s C4State, assumptions Assumptions) []string {
	var out []string
	if s.P1Violation {
		out = append(out, "P1")
	}
	if s.P2Violation {
		out = append(out, "P2")
	}
	if s.P3Violation {
		out = append(out, "P3")
	}
	if s.P4Violation {
		out = append(out, "P4")
	}
	if s.P5Violation {
		out = append(out, "P5")
	}
	if s.P6Violation {
		out = append(out, "P6")
	}
	if s.P7Violation {
		out = append(out, "P7")
	}
	if s.RetryViolation {
		out = append(out, "C4-G03")
	}
	if s.Obligations > 0 && (!assumptions.Has(C4A14) || !assumptions.Has(C4A15)) {
		out = append(out, "L1")
	}
	return out
}

func HasAuthorizedResolutionPath(s C4State, assumptions Assumptions) bool {
	return s.Obligations == 0 || assumptions.Has(C4A14) && assumptions.Has(C4A15)
}
