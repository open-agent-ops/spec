// Package formal contains the bounded executable Stage 2 models for the
// limited C4 and Evidence Acquisition profiles. It is a specification model,
// not a runtime gateway, collector, schema, or public relation validator.
package formal

const (
	C4ProfileID       = "agent-ops.c4-human-approved-apply@1.0.0"
	EvidenceProfileID = "agent-ops.evidence-acquisition@1.0.0"
	InterfaceID       = "agent-ops.c4-evidence-interface@1.0.0"
)

type Coverage string

const (
	CoverageUnknown  Coverage = "unknown"
	CoveragePartial  Coverage = "partial"
	CoverageComplete Coverage = "complete"
)

type Disposition string

const (
	DispositionUnknown     Disposition = "unknown"
	DispositionSupported   Disposition = "supported"
	DispositionRefuted     Disposition = "refuted"
	DispositionDoesNotTest Disposition = "does_not_test"
	DispositionConflicted  Disposition = "conflicted"
)

type Sufficiency string

const (
	SufficiencyUnknown      Sufficiency = "unknown"
	SufficiencyInsufficient Sufficiency = "insufficient"
	SufficiencySufficient   Sufficiency = "sufficient"
)

type TargetRef struct {
	UID        string
	Generation uint64
	Epoch      uint64
	Versions   string
}

// EvidenceFacts is the complete Stage 2 cross-profile interface. C4 consumes
// these facts but cannot create or strengthen them. No field grants authority.
type EvidenceFacts struct {
	InterfaceID           string
	RequirementID         string
	ConsumerRule          string
	Applicable            bool
	Target                TargetRef
	FreshAtUse            bool
	Coverage              Coverage
	DetectionCapable      bool
	Disposition           Disposition
	Sufficiency           Sufficiency
	ProvenanceBound       bool
	CommonCausesDisclosed bool
}

func (f EvidenceFacts) WellFormed() bool {
	return f.InterfaceID == InterfaceID && f.RequirementID != "" && f.ConsumerRule != "" &&
		f.Target.UID != "" && f.Target.Generation > 0 && f.Target.Epoch > 0 && f.Target.Versions != ""
}

func (f EvidenceFacts) SupportsEffect(target TargetRef, disposition Disposition) bool {
	return f.WellFormed() && f.Applicable && f.Target == target && f.FreshAtUse &&
		f.Coverage == CoverageComplete && f.DetectionCapable &&
		f.Disposition == disposition && f.Sufficiency == SufficiencySufficient &&
		f.ProvenanceBound && f.CommonCausesDisclosed
}

func (f EvidenceFacts) SupportsOutcome(target TargetRef) bool {
	return f.SupportsEffect(target, DispositionSupported)
}
