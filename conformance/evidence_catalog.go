package conformance

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
)

const EvidenceRelationCatalogID = "agent-ops.evidence-relation-validation@1.0.0"

//go:embed catalogs/evidence-relation-validation-v1.0.0.json
var evidenceRelationCatalogBytes []byte

type EvidenceRelationCatalog struct {
	CatalogID        string                  `json:"catalog_id"`
	ProfileID        string                  `json:"profile_id"`
	Relations        string                  `json:"relations"`
	ConstraintLevels []ConstraintLevel       `json:"constraint_levels"`
	Validators       []RelationValidatorSpec `json:"validators"`
	Claim            string                  `json:"claim"`
}

func OpenEvidenceRelationCatalog() (EvidenceRelationCatalog, error) {
	var catalog EvidenceRelationCatalog
	d := json.NewDecoder(strings.NewReader(string(evidenceRelationCatalogBytes)))
	d.DisallowUnknownFields()
	if err := d.Decode(&catalog); err != nil {
		return EvidenceRelationCatalog{}, err
	}
	if catalog.CatalogID != EvidenceRelationCatalogID || catalog.ProfileID != "agent-ops.evidence-acquisition@1.0.0" ||
		catalog.Relations != "agent-ops.evidence-relations@1.0.0" || len(catalog.ConstraintLevels) != 5 || len(catalog.Validators) != 9 || catalog.Claim == "" {
		return EvidenceRelationCatalog{}, errors.New("evidence_relation_catalog_invalid")
	}
	levels := map[int]bool{}
	for _, level := range catalog.ConstraintLevels {
		if level.Level < 1 || level.Level > 5 || level.Name == "" || level.Meaning == "" || levels[level.Level] {
			return EvidenceRelationCatalog{}, errors.New("evidence_relation_catalog_invalid")
		}
		levels[level.Level] = true
	}
	validators := map[string]bool{}
	codes := map[string]bool{}
	for _, validator := range catalog.Validators {
		if validator.ID == "" || validators[validator.ID] || !levels[validator.Level] || len(validator.Codes) == 0 || (len(validator.Relations) == 0 && validator.ID != "AOM-EVD-SCHEMA-001") {
			return EvidenceRelationCatalog{}, errors.New("evidence_relation_catalog_invalid")
		}
		validators[validator.ID] = true
		for _, relation := range validator.Relations {
			if !strings.HasPrefix(relation, "EVD-R") || !relationID(relation) {
				return EvidenceRelationCatalog{}, errors.New("evidence_relation_catalog_invalid")
			}
		}
		for _, code := range validator.Codes {
			if code == "" || codes[code] {
				return EvidenceRelationCatalog{}, errors.New("evidence_relation_catalog_invalid")
			}
			codes[code] = true
		}
	}
	return catalog, nil
}
