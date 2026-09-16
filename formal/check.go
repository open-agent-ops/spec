package formal

type CheckResult struct {
	Depth       int
	States      int
	Transitions int
	Violation   string
	Trace       []string
}

func CheckEvidenceModel(depth int) CheckResult {
	if depth < 0 {
		depth = 0
	}
	assumptions := DefaultEvidenceAssumptions()
	events := []EvidenceEvent{
		EvidenceDeclareRequirement,
		EvidenceBindProfile,
		EvidenceAttemptComplete,
		EvidenceAttemptPartial,
		EvidenceAttemptTruncated,
		EvidenceAttemptParseFailed,
		EvidenceAttemptUnauthorized,
		EvidenceRecordArtifact,
		EvidenceAssertPositive,
		EvidenceAssertNegative,
		EvidenceAssertConflict,
		EvidenceTargetVersionChange,
		EvidenceShareCommonCause,
		EvidenceBorrowC4Authority,
		EvidenceCheckpointOmit,
		EvidenceSeal,
	}
	initial := InitialEvidenceState()
	seen := map[EvidenceState]bool{initial: true}
	frontier := map[EvidenceState][]string{initial: nil}
	result := CheckResult{Depth: depth, States: 1}
	for level := 0; level < depth; level++ {
		nextFrontier := map[EvidenceState][]string{}
		for state, trace := range frontier {
			for _, event := range events {
				next, err := ApplyEvidence(state, event, assumptions)
				if err != nil {
					continue
				}
				result.Transitions++
				nextTrace := appendTrace(trace, string(event))
				if violations := EvidenceViolations(next, assumptions); len(violations) > 0 {
					result.Violation = violations[0]
					result.Trace = nextTrace
					return result
				}
				if !seen[next] {
					seen[next] = true
					nextFrontier[next] = nextTrace
				}
			}
		}
		frontier = nextFrontier
		result.States = len(seen)
		if len(frontier) == 0 {
			break
		}
	}
	return result
}

func CheckC4Model(depth int) CheckResult {
	if depth < 0 {
		depth = 0
	}
	assumptions := DefaultC4Assumptions()
	events := []string{
		string(C4CreatePackage), string(C4Approve), string(C4HumanStart), string(C4Reserve),
		string(C4Revoke), string(C4MutatePackage), string(C4TargetABA), string(C4AdvancePastExpiry),
		string(C4Admit), string(C4BypassWriterEffect), string(C4LoseResponse),
		string(C4ObserveApplied), "observe_not_applied_partial", string(C4ReplaySameKey), string(C4Retry),
		string(C4LeaseExpire), string(C4StaleWriterEffect), string(C4CheckpointOmit),
		string(C4Restart), string(C4IntentSatisfied), string(C4OutcomeSuccess),
		string(C4OutcomeSuccessFalse), string(C4Reconcile), string(C4Compensate), string(C4Release),
	}
	initial := InitialC4State()
	seen := map[C4State]bool{initial: true}
	frontier := map[C4State][]string{initial: nil}
	result := CheckResult{Depth: depth, States: 1}
	for level := 0; level < depth; level++ {
		nextFrontier := map[C4State][]string{}
		for state, trace := range frontier {
			for _, event := range events {
				next, err := ApplyC4(state, actionForName(event, state, EvidenceFacts{}), assumptions)
				if err != nil {
					continue
				}
				result.Transitions++
				nextTrace := appendTrace(trace, event)
				if violations := C4Violations(next, assumptions); len(violations) > 0 {
					result.Violation = violations[0]
					result.Trace = nextTrace
					return result
				}
				if !seen[next] {
					seen[next] = true
					nextFrontier[next] = nextTrace
				}
			}
		}
		frontier = nextFrontier
		result.States = len(seen)
		if len(frontier) == 0 {
			break
		}
	}
	return result
}

func appendTrace(trace []string, event string) []string {
	out := make([]string, len(trace)+1)
	copy(out, trace)
	out[len(trace)] = event
	return out
}
