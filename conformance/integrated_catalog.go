package conformance

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
)

const (
	IntegratedCatalogID       = "agent-ops.integrated-conformance@1.0.0"
	IntegratedProfileID       = "agent-ops.c4-evidence-integrated@1.0.0"
	EvaluationProtocolID      = "agent-ops.c4-evidence-evaluation@1.0.0"
	integratedFixtureFormatID = "agent-ops.integrated-conformance-traces@1.0.0"
)

//go:embed catalogs/integrated-conformance-v1.0.0.json
var integratedCatalogBytes []byte

type IntegratedCatalog struct {
	CatalogID           string                  `json:"catalog_id"`
	ProfileID           string                  `json:"profile_id"`
	C4ProfileID         string                  `json:"c4_profile_id"`
	EvidenceProfileID   string                  `json:"evidence_profile_id"`
	C4ValidatorID       string                  `json:"c4_validator_id"`
	EvidenceValidatorID string                  `json:"evidence_validator_id"`
	InterfaceID         string                  `json:"interface_id"`
	EvaluationProtocol  string                  `json:"evaluation_protocol_id"`
	Validators          []RelationValidatorSpec `json:"validators"`
	Claim               string                  `json:"claim"`
}

func OpenIntegratedCatalog() (IntegratedCatalog, error) {
	var catalog IntegratedCatalog
	d := json.NewDecoder(strings.NewReader(string(integratedCatalogBytes)))
	d.DisallowUnknownFields()
	if err := d.Decode(&catalog); err != nil {
		return IntegratedCatalog{}, err
	}
	if catalog.CatalogID != IntegratedCatalogID || catalog.ProfileID != IntegratedProfileID ||
		catalog.C4ProfileID != "agent-ops.c4-human-approved-apply@1.0.0" ||
		catalog.EvidenceProfileID != "agent-ops.evidence-acquisition@1.0.0" ||
		catalog.C4ValidatorID != RelationCatalogID || catalog.EvidenceValidatorID != EvidenceRelationCatalogID ||
		catalog.InterfaceID != "agent-ops.c4-evidence-interface@1.0.0" ||
		catalog.EvaluationProtocol != EvaluationProtocolID || len(catalog.Validators) != 2 || catalog.Claim == "" {
		return IntegratedCatalog{}, errors.New("integrated_catalog_invalid")
	}
	validators := map[string]bool{}
	codes := map[string]bool{}
	for _, validator := range catalog.Validators {
		if validator.ID == "" || validators[validator.ID] || validator.Level < 1 || validator.Level > 5 || len(validator.Relations) == 0 || len(validator.Codes) == 0 {
			return IntegratedCatalog{}, errors.New("integrated_catalog_invalid")
		}
		validators[validator.ID] = true
		for _, relation := range validator.Relations {
			if !relationID(relation) {
				return IntegratedCatalog{}, errors.New("integrated_catalog_invalid")
			}
		}
		for _, code := range validator.Codes {
			if code == "" || codes[code] {
				return IntegratedCatalog{}, errors.New("integrated_catalog_invalid")
			}
			codes[code] = true
		}
	}
	return catalog, nil
}
