package conformance

import (
	_ "embed"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

const RelationCatalogID = "agent-ops.relation-validation@1.0.0"

//go:embed catalogs/relation-validation-v1.0.0.json
var relationCatalogBytes []byte

type ConstraintLevel struct {
	Level   int    `json:"level"`
	Name    string `json:"name"`
	Meaning string `json:"meaning"`
}

type RelationValidatorSpec struct {
	ID        string   `json:"id"`
	Level     int      `json:"level"`
	Relations []string `json:"relations"`
	Codes     []string `json:"codes"`
}

type RelationCatalog struct {
	CatalogID         string                  `json:"catalog_id"`
	ProfileID         string                  `json:"profile_id"`
	C4Relations       string                  `json:"c4_relations"`
	EvidenceRelations string                  `json:"evidence_relations"`
	InterfaceID       string                  `json:"interface_id"`
	ConstraintLevels  []ConstraintLevel       `json:"constraint_levels"`
	Validators        []RelationValidatorSpec `json:"validators"`
	Claim             string                  `json:"claim"`
}

func OpenRelationCatalog() (RelationCatalog, error) {
	var catalog RelationCatalog
	d := json.NewDecoder(strings.NewReader(string(relationCatalogBytes)))
	d.DisallowUnknownFields()
	if err := d.Decode(&catalog); err != nil {
		return RelationCatalog{}, err
	}
	if catalog.CatalogID != RelationCatalogID || catalog.ProfileID != "agent-ops.c4-human-approved-apply@1.0.0" ||
		catalog.C4Relations != "agent-ops.c4-relations@1.0.0" || catalog.EvidenceRelations != "agent-ops.evidence-relations@1.0.0" ||
		catalog.InterfaceID != "agent-ops.c4-evidence-interface@1.0.0" || len(catalog.ConstraintLevels) != 5 || len(catalog.Validators) != 10 || catalog.Claim == "" {
		return RelationCatalog{}, errors.New("relation_catalog_invalid")
	}
	levels := map[int]bool{}
	for _, level := range catalog.ConstraintLevels {
		if level.Level < 1 || level.Level > 5 || level.Name == "" || level.Meaning == "" || levels[level.Level] {
			return RelationCatalog{}, errors.New("relation_catalog_invalid")
		}
		levels[level.Level] = true
	}
	validators := map[string]bool{}
	codes := map[string]bool{}
	for _, validator := range catalog.Validators {
		if validator.ID == "" || validators[validator.ID] || !levels[validator.Level] || len(validator.Codes) == 0 || (len(validator.Relations) == 0 && validator.ID != "AOM-C4-SCHEMA-001") {
			return RelationCatalog{}, errors.New("relation_catalog_invalid")
		}
		validators[validator.ID] = true
		for _, relation := range validator.Relations {
			if !relationID(relation) {
				return RelationCatalog{}, errors.New("relation_catalog_invalid")
			}
		}
		for _, code := range validator.Codes {
			if code == "" || codes[code] {
				return RelationCatalog{}, errors.New("relation_catalog_invalid")
			}
			codes[code] = true
		}
	}
	return catalog, nil
}

func relationID(value string) bool {
	for _, prefix := range []string{"C4-R", "EVD-R", "CEI-R"} {
		if strings.HasPrefix(value, prefix) && len(value) == len(prefix)+2 {
			return value[len(prefix)] >= '0' && value[len(prefix)] <= '9' && value[len(prefix)+1] >= '0' && value[len(prefix)+1] <= '9'
		}
	}
	return false
}

func catalogIndex(catalog RelationCatalog) map[string]RelationValidatorSpec {
	out := make(map[string]RelationValidatorSpec, len(catalog.Validators))
	for _, validator := range catalog.Validators {
		validator.Relations = append([]string(nil), validator.Relations...)
		validator.Codes = append([]string(nil), validator.Codes...)
		sort.Strings(validator.Relations)
		sort.Strings(validator.Codes)
		out[validator.ID] = validator
	}
	return out
}
