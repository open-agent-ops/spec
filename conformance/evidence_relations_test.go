package conformance

import (
	"encoding/json"
	"os"
	"testing"
)

type evidenceTraceCorpus struct {
	Format        string         `json:"format"`
	Base          map[string]any `json:"base"`
	RelationCases []struct {
		ID         string `json:"id"`
		PositiveID string `json:"positive_id"`
		Mutations  []struct {
			Pointer string `json:"pointer"`
			Value   any    `json:"value"`
		} `json:"mutations"`
		Code     string `json:"code"`
		Relation string `json:"relation"`
		Level    string `json:"level"`
	} `json:"relation_cases"`
	SchemaCases []struct {
		ID      string `json:"id"`
		Pointer string `json:"pointer"`
	} `json:"schema_cases"`
}

func TestEvidenceRelationCatalogAndPairedFixtures(t *testing.T) {
	catalog, err := OpenEvidenceRelationCatalog()
	if err != nil {
		t.Fatal(err)
	}
	seenRelations := map[string]bool{}
	for _, validator := range catalog.Validators {
		for _, relation := range validator.Relations {
			seenRelations[relation] = true
		}
	}
	if len(seenRelations) != 14 {
		t.Fatalf("Evidence relation catalog incomplete: %v", seenRelations)
	}
	fixtures := loadEvidenceTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.RelationCases {
		t.Run(fixture.ID, func(t *testing.T) {
			positive := cloneFixture(t, fixtures.Base)
			accepted := v.ValidateEvidenceRelations(evidenceInputFromFixture(t, positive))
			if !accepted.Accepted || !accepted.Syntactic || !accepted.Relations || accepted.Sufficiency != "sufficient" || len(accepted.Findings) != 0 {
				t.Fatalf("positive pair %s rejected: %+v", fixture.PositiveID, accepted)
			}
			negative := cloneFixture(t, fixtures.Base)
			for _, mutation := range fixture.Mutations {
				setFixturePointer(t, negative, mutation.Pointer, mutation.Value)
			}
			result := v.ValidateEvidenceRelations(evidenceInputFromFixture(t, negative))
			if result.Accepted || !result.Syntactic || result.Sufficiency == "sufficient" {
				t.Fatalf("negative fixture accepted: %+v", result)
			}
			if (fixture.Level == "sufficiency") != result.Relations {
				t.Fatalf("%s relation/sufficiency boundary incorrect: %+v", fixture.ID, result)
			}
			if !hasRelationFinding(result.Findings, fixture.Code, fixture.Relation) {
				t.Fatalf("missing %s/%s in %+v", fixture.Code, fixture.Relation, result.Findings)
			}
			if fixture.Level == "sufficiency" && !hasFindingAtLevel(result.Findings, fixture.Code, 5) {
				t.Fatalf("%s was not rejected at sufficiency level: %+v", fixture.ID, result.Findings)
			}
		})
	}
}

func TestEvidenceSchemaNegativeFixtures(t *testing.T) {
	fixtures := loadEvidenceTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.SchemaCases {
		t.Run(fixture.ID, func(t *testing.T) {
			root := cloneFixture(t, fixtures.Base)
			setFixturePointer(t, root, fixture.Pointer, true)
			result := v.ValidateEvidenceRelations(evidenceInputFromFixture(t, root))
			if result.Accepted || result.Syntactic || result.Relations || result.Sufficiency != "unknown" || !hasRelationFinding(result.Findings, "schema_invalid", "") {
				t.Fatalf("schema-negative trace accepted or misclassified: %+v", result)
			}
		})
	}
}

func TestEvidenceCompletenessAndUnknownAreMonotone(t *testing.T) {
	fixtures := loadEvidenceTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	degradations := []struct {
		pointer string
		value   any
		code    string
	}{
		{"/receipts/0/artifact/coverage", "partial", "acquisition_incomplete"},
		{"/assertions/0/fresh_at_use", false, "evidence_stale_at_use"},
		{"/assertions/0/sufficiency", "unknown", "assertion_unsupported"},
		{"/bundle/computed_complete", false, "bundle_seal_invalid"},
	}
	for _, degradation := range degradations {
		root := cloneFixture(t, fixtures.Base)
		setFixturePointer(t, root, degradation.pointer, degradation.value)
		result := v.ValidateEvidenceRelations(evidenceInputFromFixture(t, root))
		if result.Accepted || result.Sufficiency == "sufficient" || !hasRelationFinding(result.Findings, degradation.code, "") {
			t.Fatalf("degradation improved conformance: %+v", result)
		}
	}
}

func TestMalformedEvidenceInputDoesNotPanic(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{nil, []byte(`{`), []byte(`[]`), []byte(`{"unknown":true}`)} {
		result := v.ValidateEvidenceRelations(EvidenceRelationInput{Requirement: raw, Profile: raw, Receipts: []json.RawMessage{raw}, Assertions: []json.RawMessage{raw}, Bundle: raw})
		if result.Accepted || result.Syntactic || !hasRelationFinding(result.Findings, "schema_invalid", "") && !hasRelationFinding(result.Findings, "trace_incomplete", "") {
			t.Fatalf("malformed input accepted: %+v", result)
		}
	}
}

func loadEvidenceTraceFixtures(t *testing.T) evidenceTraceCorpus {
	t.Helper()
	b, err := os.ReadFile("evidencetestdata/traces.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures evidenceTraceCorpus
	if json.Unmarshal(b, &fixtures) != nil || fixtures.Format != "agent-ops.evidence-relation-traces@1.0.0" || len(fixtures.RelationCases) != 23 || len(fixtures.SchemaCases) != 5 {
		t.Fatal("Evidence relation fixture corpus invalid")
	}
	return fixtures
}

func evidenceInputFromFixture(t *testing.T, root map[string]any) EvidenceRelationInput {
	t.Helper()
	raw := func(value any) json.RawMessage {
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	rawList := func(value any) []json.RawMessage {
		values, ok := value.([]any)
		if !ok {
			t.Fatal("fixture list")
		}
		out := make([]json.RawMessage, 0, len(values))
		for _, value := range values {
			out = append(out, raw(value))
		}
		return out
	}
	return EvidenceRelationInput{
		Requirement: raw(root["requirement"]),
		Profile:     raw(root["profile"]),
		Receipts:    rawList(root["receipts"]),
		Assertions:  rawList(root["assertions"]),
		Bundle:      raw(root["bundle"]),
	}
}

func hasFindingAtLevel(findings []RelationFinding, code string, level int) bool {
	for _, finding := range findings {
		if finding.Code == code && finding.Level == level {
			return true
		}
	}
	return false
}
