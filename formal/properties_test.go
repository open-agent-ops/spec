package formal_test

import (
	"testing"

	"github.com/open-agent-ops/spec/formal"
)

func TestEvidenceSafetyBoundaries(t *testing.T) {
	assumptions := formal.DefaultEvidenceAssumptions()

	t.Run("partial evidence cannot seal", func(t *testing.T) {
		state := evidenceTrace(t, assumptions,
			formal.EvidenceDeclareRequirement,
			formal.EvidenceBindProfile,
			formal.EvidenceAttemptPartial,
			formal.EvidenceRecordArtifact,
			formal.EvidenceAssertNegative,
		)
		if state.Facts(assumptions).Sufficiency == formal.SufficiencySufficient {
			t.Fatal("partial observation became sufficient")
		}
		if _, err := formal.ApplyEvidence(state, formal.EvidenceSeal, assumptions); err == nil {
			t.Fatal("partial observation was sealed")
		}
	})

	t.Run("critical version change invalidates use", func(t *testing.T) {
		state := evidenceTrace(t, assumptions,
			formal.EvidenceDeclareRequirement,
			formal.EvidenceBindProfile,
			formal.EvidenceAttemptComplete,
			formal.EvidenceRecordArtifact,
			formal.EvidenceAssertPositive,
			formal.EvidenceTargetVersionChange,
		)
		facts := state.Facts(assumptions)
		if facts.FreshAtUse || facts.Sufficiency == formal.SufficiencySufficient {
			t.Fatalf("stale facts remained usable: %+v", facts)
		}
	})

	t.Run("retry appends receipt and invalidates derived current facts", func(t *testing.T) {
		state := evidenceTrace(t, assumptions,
			formal.EvidenceDeclareRequirement,
			formal.EvidenceBindProfile,
			formal.EvidenceAttemptComplete,
			formal.EvidenceRecordArtifact,
			formal.EvidenceAssertPositive,
			formal.EvidenceAttemptPartial,
		)
		if state.Attempts != 2 || state.Receipts != 2 {
			t.Fatalf("attempt lineage overwritten: attempts=%d receipts=%d", state.Attempts, state.Receipts)
		}
		if state.ArtifactBound || state.Disposition != formal.DispositionUnknown {
			t.Fatal("new attempt reused prior derived artifact or assertion")
		}
	})

	t.Run("unauthorized source cannot seal", func(t *testing.T) {
		state := evidenceTrace(t, assumptions,
			formal.EvidenceDeclareRequirement,
			formal.EvidenceBindProfile,
			formal.EvidenceAttemptUnauthorized,
			formal.EvidenceRecordArtifact,
			formal.EvidenceAssertPositive,
		)
		if _, err := formal.ApplyEvidence(state, formal.EvidenceSeal, assumptions); err == nil {
			t.Fatal("unauthorized source was sealed")
		}
	})
}

func TestC4SafetyBoundaries(t *testing.T) {
	assumptions := formal.DefaultC4Assumptions()
	base := c4Trace(t, assumptions,
		formal.C4CreatePackage,
		formal.C4Approve,
		formal.C4HumanStart,
		formal.C4Reserve,
		formal.C4Admit,
		formal.C4LoseResponse,
	)

	t.Run("same key replay does not consume budget", func(t *testing.T) {
		next, err := formal.ApplyC4(base, formal.C4Action{Kind: formal.C4ReplaySameKey}, assumptions)
		if err != nil {
			t.Fatal(err)
		}
		if next.Consumed != base.Consumed || next.Obligations != base.Obligations {
			t.Fatalf("same-key replay changed counters: before=%+v after=%+v", base, next)
		}
	})

	t.Run("compensation does not authorize retry", func(t *testing.T) {
		compensated, err := formal.ApplyC4(base, formal.C4Action{Kind: formal.C4Compensate}, assumptions)
		if err != nil {
			t.Fatal(err)
		}
		if !compensated.Compensated || compensated.Reconciled {
			t.Fatalf("compensation and reconciliation were conflated: %+v", compensated)
		}
		if _, err := formal.ApplyC4(compensated, formal.C4Action{Kind: formal.C4Retry}, assumptions); err == nil {
			t.Fatal("compensation made unresolved retry safe")
		}
	})

	t.Run("admission is not outcome", func(t *testing.T) {
		if _, err := formal.ApplyC4(base, formal.C4Action{Kind: formal.C4OutcomeSuccess}, assumptions); err == nil {
			t.Fatal("admission without intent-bound evidence became Outcome success")
		}
	})
}

func evidenceTrace(t *testing.T, assumptions formal.Assumptions, events ...formal.EvidenceEvent) formal.EvidenceState {
	t.Helper()
	state := formal.InitialEvidenceState()
	for _, event := range events {
		var err error
		state, err = formal.ApplyEvidence(state, event, assumptions)
		if err != nil {
			t.Fatalf("event %s: %v", event, err)
		}
	}
	return state
}

func c4Trace(t *testing.T, assumptions formal.Assumptions, events ...formal.C4Event) formal.C4State {
	t.Helper()
	state := formal.InitialC4State()
	for _, event := range events {
		var err error
		state, err = formal.ApplyC4(state, formal.C4Action{Kind: event}, assumptions)
		if err != nil {
			t.Fatalf("event %s: %v", event, err)
		}
	}
	return state
}
