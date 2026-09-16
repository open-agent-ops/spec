package formal

import "errors"

type EvidenceEvent string

const (
	EvidenceDeclareRequirement  EvidenceEvent = "declare_requirement"
	EvidenceBindProfile         EvidenceEvent = "bind_profile"
	EvidenceAttemptComplete     EvidenceEvent = "attempt_complete"
	EvidenceAttemptPartial      EvidenceEvent = "attempt_partial"
	EvidenceAttemptTruncated    EvidenceEvent = "attempt_truncated"
	EvidenceAttemptParseFailed  EvidenceEvent = "attempt_parse_failed"
	EvidenceAttemptUnauthorized EvidenceEvent = "attempt_unauthorized"
	EvidenceRecordArtifact      EvidenceEvent = "record_artifact"
	EvidenceAssertPositive      EvidenceEvent = "assert_positive"
	EvidenceAssertNegative      EvidenceEvent = "assert_negative"
	EvidenceAssertConflict      EvidenceEvent = "assert_conflict"
	EvidenceTargetVersionChange EvidenceEvent = "target_version_change"
	EvidenceShareCommonCause    EvidenceEvent = "share_common_cause"
	EvidenceBorrowC4Authority   EvidenceEvent = "borrow_c4_authority"
	EvidenceCheckpointOmit      EvidenceEvent = "checkpoint_omit_attempt"
	EvidenceSeal                EvidenceEvent = "seal"
)

type EvidenceState struct {
	RequirementID         string
	ConsumerRule          string
	Required              TargetRef
	Current               TargetRef
	Applicable            bool
	ProfileBound          bool
	Attempts              uint8
	Receipts              uint8
	TransportOK           bool
	ParseOK               bool
	Truncated             bool
	ArtifactBound         bool
	IntegrityOK           bool
	LineageOK             bool
	SourceAuthorized      bool
	Coverage              Coverage
	DetectionCapable      bool
	Disposition           Disposition
	NegativeClaim         bool
	TTLValid              bool
	CommonCausesDisclosed bool
	Independent           bool
	Sealed                bool
	LineageViolation      bool
	AuthorityBorrowed     bool
}

func InitialEvidenceState() EvidenceState {
	return EvidenceState{
		Coverage:    CoverageUnknown,
		Disposition: DispositionUnknown,
		TTLValid:    true,
	}
}

func ApplyEvidence(state EvidenceState, event EvidenceEvent, assumptions Assumptions) (EvidenceState, error) {
	next := state
	switch event {
	case EvidenceDeclareRequirement:
		if next.RequirementID != "" {
			return state, errors.New("requirement already declared")
		}
		next.RequirementID = "effect-observation"
		next.ConsumerRule = "c4-effect-rule@1.0.0"
		next.Required = TargetRef{UID: "target-1", Generation: 1, Epoch: 1, Versions: "resource=1;policy=1;profile=1"}
		next.Current = next.Required
		next.Applicable = true
	case EvidenceBindProfile:
		if next.RequirementID == "" {
			return state, errors.New("requirement missing")
		}
		next.ProfileBound = true
	case EvidenceAttemptComplete, EvidenceAttemptPartial, EvidenceAttemptTruncated, EvidenceAttemptParseFailed, EvidenceAttemptUnauthorized:
		if !next.ProfileBound {
			return state, errors.New("profile missing")
		}
		next.Attempts++
		next.Receipts++
		next.TransportOK = true
		next.ParseOK = true
		next.Truncated = false
		next.Coverage = CoverageComplete
		next.DetectionCapable = true
		next.SourceAuthorized = true
		next.CommonCausesDisclosed = true
		next.Independent = true
		next.ArtifactBound = false
		next.IntegrityOK = false
		next.LineageOK = false
		next.Disposition = DispositionUnknown
		next.NegativeClaim = false
		if event == EvidenceAttemptPartial {
			next.Coverage = CoveragePartial
		}
		if event == EvidenceAttemptTruncated {
			next.Truncated = true
			next.Coverage = CoveragePartial
		}
		if event == EvidenceAttemptParseFailed {
			next.ParseOK = false
			next.Coverage = CoverageUnknown
			next.DetectionCapable = false
		}
		if event == EvidenceAttemptUnauthorized {
			next.SourceAuthorized = false
		}
	case EvidenceRecordArtifact:
		if next.Receipts == 0 {
			return state, errors.New("receipt missing")
		}
		if assumptions.Has(EvidenceA03) && (!next.TransportOK || !next.ParseOK || next.Truncated) {
			return state, errors.New("acquisition incomplete")
		}
		next.ArtifactBound = true
		next.IntegrityOK = true
		next.LineageOK = true
	case EvidenceAssertPositive:
		if !next.ArtifactBound {
			return state, errors.New("artifact missing")
		}
		next.Disposition = DispositionSupported
		next.NegativeClaim = false
	case EvidenceAssertNegative:
		if !next.ArtifactBound {
			return state, errors.New("artifact missing")
		}
		if assumptions.Has(EvidenceA02) && (next.Coverage != CoverageComplete || !next.DetectionCapable) {
			next.Disposition = DispositionDoesNotTest
		} else {
			next.Disposition = DispositionSupported
		}
		next.NegativeClaim = true
	case EvidenceAssertConflict:
		if !next.ArtifactBound {
			return state, errors.New("artifact missing")
		}
		next.Disposition = DispositionConflicted
	case EvidenceTargetVersionChange:
		if next.Current.UID == "" {
			return state, errors.New("target missing")
		}
		next.Current.Versions = "resource=2;policy=1;profile=1"
	case EvidenceShareCommonCause:
		next.Independent = false
		next.CommonCausesDisclosed = false
	case EvidenceBorrowC4Authority:
		if assumptions.Has(EvidenceA07) {
			return state, errors.New("C4 authority is not evidence truth")
		}
		next.AuthorityBorrowed = true
		next.Disposition = DispositionSupported
	case EvidenceCheckpointOmit:
		if next.Attempts == 0 {
			return state, errors.New("attempt missing")
		}
		if assumptions.Has(EvidenceA06) {
			return state, errors.New("checkpoint cannot omit attempt")
		}
		next.Attempts--
		next.LineageViolation = true
	case EvidenceSeal:
		facts := next.Facts(assumptions)
		if facts.Sufficiency != SufficiencySufficient {
			return state, errors.New("evidence insufficient")
		}
		next.Sealed = true
	default:
		return state, errors.New("unknown evidence event")
	}
	return next, nil
}

func (s EvidenceState) Facts(assumptions Assumptions) EvidenceFacts {
	fresh := s.TTLValid
	if assumptions.Has(EvidenceA01) {
		fresh = fresh && s.Required == s.Current
	}
	acquisitionOK := s.TransportOK && s.ParseOK && !s.Truncated
	if !assumptions.Has(EvidenceA03) {
		acquisitionOK = s.TransportOK
	}
	sourceOK := s.SourceAuthorized
	if !assumptions.Has(EvidenceA04) {
		sourceOK = true
	}
	independenceOK := s.CommonCausesDisclosed && s.Independent
	if !assumptions.Has(EvidenceA05) {
		independenceOK = true
	}
	disposition := s.Disposition
	coverage := s.Coverage
	detection := s.DetectionCapable
	if !assumptions.Has(EvidenceA03) && (!s.ParseOK || s.Truncated) {
		coverage = CoverageComplete
		detection = true
	}
	if s.NegativeClaim && !assumptions.Has(EvidenceA02) {
		coverage = CoverageComplete
		detection = true
	}
	authoritySeparated := !s.AuthorityBorrowed
	if !assumptions.Has(EvidenceA07) {
		authoritySeparated = true
	}
	complete := s.Applicable && s.ProfileBound && s.Attempts > 0 && s.Receipts > 0 &&
		acquisitionOK && s.ArtifactBound && s.IntegrityOK && s.LineageOK && sourceOK &&
		fresh && coverage == CoverageComplete && detection &&
		disposition == DispositionSupported && independenceOK && authoritySeparated && !s.LineageViolation
	sufficiency := SufficiencyInsufficient
	if complete {
		sufficiency = SufficiencySufficient
	} else if !s.Applicable || s.RequirementID == "" || disposition == DispositionUnknown {
		sufficiency = SufficiencyUnknown
	}
	return EvidenceFacts{
		InterfaceID:           InterfaceID,
		RequirementID:         s.RequirementID,
		ConsumerRule:          s.ConsumerRule,
		Applicable:            s.Applicable,
		Target:                s.Required,
		FreshAtUse:            fresh,
		Coverage:              coverage,
		DetectionCapable:      detection,
		Disposition:           disposition,
		Sufficiency:           sufficiency,
		ProvenanceBound:       s.ArtifactBound && s.IntegrityOK && s.LineageOK,
		CommonCausesDisclosed: independenceOK,
	}
}

func EvidenceViolations(s EvidenceState, assumptions Assumptions) []string {
	f := s.Facts(assumptions)
	var out []string
	if f.FreshAtUse && s.Required != s.Current {
		out = append(out, "E1")
	}
	if s.NegativeClaim && f.Disposition == DispositionSupported && (s.Coverage != CoverageComplete || !s.DetectionCapable) {
		out = append(out, "E2")
	}
	if f.Sufficiency == SufficiencySufficient && (!s.TransportOK || !s.ParseOK || s.Truncated) {
		out = append(out, "E3")
	}
	if f.Sufficiency == SufficiencySufficient && !s.SourceAuthorized {
		out = append(out, "E4")
	}
	if f.Sufficiency == SufficiencySufficient && (!s.CommonCausesDisclosed || !s.Independent) {
		out = append(out, "E5")
	}
	if s.LineageViolation {
		out = append(out, "E6")
	}
	if f.Sufficiency == SufficiencySufficient && s.AuthorityBorrowed {
		out = append(out, "E7")
	}
	return out
}
