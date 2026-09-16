package conformance

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
)

//go:embed catalogs/evaluation-reproducibility-v1.0.0.json
var evaluationProtocolBytes []byte

type EvaluationVersionPins struct {
	C4ProfileID         string `json:"c4_profile_id"`
	EvidenceProfileID   string `json:"evidence_profile_id"`
	InterfaceID         string `json:"interface_id"`
	C4RelationsID       string `json:"c4_relations_id"`
	EvidenceRelationsID string `json:"evidence_relations_id"`
	C4ValidatorID       string `json:"c4_validator_id"`
	EvidenceValidatorID string `json:"evidence_validator_id"`
	IntegratedValidator string `json:"integrated_validator_id"`
}

type EvaluationProtocol struct {
	ProtocolID          string                `json:"protocol_id"`
	Status              string                `json:"status"`
	Seed                int                   `json:"seed"`
	CaseSelection       string                `json:"case_selection"`
	VersionPins         EvaluationVersionPins `json:"version_pins"`
	DenominatorOutcomes []string              `json:"denominator_outcomes"`
	NamedBarriers       []string              `json:"named_barriers"`
	PrimaryMetrics      []string              `json:"primary_metrics"`
	SecondaryMetrics    []string              `json:"secondary_metrics"`
	RequiredRecords     []string              `json:"required_records"`
	RequiredArtifacts   []string              `json:"required_artifacts"`
	BlockingRule        string                `json:"blocking_rule"`
	ClaimLimit          string                `json:"claim_limit"`
}

func OpenEvaluationProtocol() (EvaluationProtocol, error) {
	var protocol EvaluationProtocol
	d := json.NewDecoder(strings.NewReader(string(evaluationProtocolBytes)))
	d.DisallowUnknownFields()
	if err := d.Decode(&protocol); err != nil {
		return EvaluationProtocol{}, err
	}
	pins := protocol.VersionPins
	if protocol.ProtocolID != EvaluationProtocolID || protocol.Status != "pre-runtime-frozen" || protocol.Seed != 30404 ||
		protocol.CaseSelection != "all_assigned_cases" ||
		pins.C4ProfileID != "agent-ops.c4-human-approved-apply@1.0.0" ||
		pins.EvidenceProfileID != "agent-ops.evidence-acquisition@1.0.0" ||
		pins.InterfaceID != "agent-ops.c4-evidence-interface@1.0.0" ||
		pins.C4RelationsID != "agent-ops.c4-relations@1.0.0" ||
		pins.EvidenceRelationsID != "agent-ops.evidence-relations@1.0.0" ||
		pins.C4ValidatorID != RelationCatalogID || pins.EvidenceValidatorID != EvidenceRelationCatalogID ||
		pins.IntegratedValidator != IntegratedCatalogID ||
		!sameStrings(protocol.DenominatorOutcomes, []string{"succeeded", "failed", "timeout", "partial", "unknown"}) ||
		len(protocol.NamedBarriers) < 6 || len(protocol.PrimaryMetrics) < 14 || len(protocol.SecondaryMetrics) < 13 ||
		len(protocol.RequiredRecords) < 13 || len(protocol.RequiredArtifacts) < 11 ||
		protocol.BlockingRule == "" || protocol.ClaimLimit == "" {
		return EvaluationProtocol{}, errors.New("evaluation_protocol_invalid")
	}
	for _, values := range [][]string{protocol.NamedBarriers, protocol.PrimaryMetrics, protocol.SecondaryMetrics, protocol.RequiredRecords, protocol.RequiredArtifacts} {
		if hasBlankOrDuplicate(values) {
			return EvaluationProtocol{}, errors.New("evaluation_protocol_invalid")
		}
	}
	return protocol, nil
}

func hasBlankOrDuplicate(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}
