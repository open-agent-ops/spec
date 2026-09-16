package conformance

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

type traceFixtureCorpus struct {
	Format        string         `json:"format"`
	Base          map[string]any `json:"base"`
	RelationCases []struct {
		ID       string `json:"id"`
		Pointer  string `json:"pointer"`
		Value    any    `json:"value"`
		Accepted bool   `json:"accepted"`
		Code     string `json:"code"`
		Relation string `json:"relation"`
	} `json:"relation_cases"`
	SchemaCases []struct {
		ID      string `json:"id"`
		Pointer string `json:"pointer"`
	} `json:"schema_cases"`
}

func TestRelationCatalogAndFixtures(t *testing.T) {
	catalog, err := OpenRelationCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogIndex(catalog)) != 10 {
		t.Fatal("validator catalog incomplete")
	}
	fixtures := loadTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.RelationCases {
		t.Run(fixture.ID, func(t *testing.T) {
			root := cloneFixture(t, fixtures.Base)
			if fixture.Pointer != "" {
				setFixturePointer(t, root, fixture.Pointer, fixture.Value)
			}
			result := v.ValidateRelations(relationInputFromFixture(t, root))
			if result.Accepted != fixture.Accepted {
				t.Fatalf("accepted=%v findings=%+v", result.Accepted, result.Findings)
			}
			if !result.Complete {
				t.Fatalf("relation fixture failed an individual schema: %+v", result.Findings)
			}
			if fixture.Accepted {
				if len(result.Findings) != 0 || result.Catalog != RelationCatalogID {
					t.Fatalf("valid trace result=%+v", result)
				}
				return
			}
			if !hasRelationFinding(result.Findings, fixture.Code, fixture.Relation) {
				t.Fatalf("missing %s/%s in %+v", fixture.Code, fixture.Relation, result.Findings)
			}
		})
	}
}

func TestRelationSchemaNegativeFixtures(t *testing.T) {
	fixtures := loadTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.SchemaCases {
		t.Run(fixture.ID, func(t *testing.T) {
			root := cloneFixture(t, fixtures.Base)
			setFixturePointer(t, root, fixture.Pointer, true)
			result := v.ValidateRelations(relationInputFromFixture(t, root))
			if result.Accepted || result.Complete || !hasRelationFinding(result.Findings, "schema_invalid", "") {
				t.Fatalf("schema-negative trace accepted: %+v", result)
			}
		})
	}
}

func TestRelationAxesRemainIndependent(t *testing.T) {
	fixtures := loadTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"autonomy":   "R2",
		"impact":     "I2",
		"capability": "C3",
		"guardian":   "T3",
	}
	for axis, value := range cases {
		t.Run(axis, func(t *testing.T) {
			root := cloneFixture(t, fixtures.Base)
			setFixturePointer(t, root, "/execution_records/1/axes/"+axis, value)
			result := v.ValidateRelations(relationInputFromFixture(t, root))
			if !result.Complete || result.Accepted || !hasRelationFinding(result.Findings, "axis_binding_mismatch", "C4-R01") {
				t.Fatalf("axis %s not independently bound: %+v", axis, result)
			}
		})
	}
}

func TestEffectOutcomeAndCompletenessRemainSeparate(t *testing.T) {
	fixtures := loadTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	root := cloneFixture(t, fixtures.Base)
	setFixturePointer(t, root, "/outcome/completeness", "unknown")
	result := v.ValidateRelations(relationInputFromFixture(t, root))
	if !result.Accepted {
		t.Fatalf("unknown completeness was conflated with applied effect or unknown outcome: %+v", result.Findings)
	}
}

func loadTraceFixtures(t *testing.T) traceFixtureCorpus {
	t.Helper()
	b, err := os.ReadFile("relationtestdata/traces.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures traceFixtureCorpus
	if json.Unmarshal(b, &fixtures) != nil || fixtures.Format != "agent-ops.relation-traces@1.0.0" || len(fixtures.RelationCases) != 13 || len(fixtures.SchemaCases) != 6 {
		t.Fatal("relation fixture corpus invalid")
	}
	return fixtures
}

func cloneFixture(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if json.Unmarshal(b, &out) != nil {
		t.Fatal("fixture clone")
	}
	return out
}

func setFixturePointer(t *testing.T, root map[string]any, pointer string, value any) {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if pointer == "" || len(parts) == 0 {
		t.Fatal("empty fixture pointer")
	}
	var current any = root
	for _, part := range parts[:len(parts)-1] {
		switch node := current.(type) {
		case map[string]any:
			current = node[part]
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(node) {
				t.Fatal("fixture pointer index", pointer)
			}
			current = node[index]
		default:
			t.Fatal("fixture pointer traversal", pointer)
		}
	}
	last := parts[len(parts)-1]
	switch node := current.(type) {
	case map[string]any:
		node[last] = value
	case []any:
		index, err := strconv.Atoi(last)
		if err != nil || index < 0 || index >= len(node) {
			t.Fatal("fixture pointer final index", pointer)
		}
		node[index] = value
	default:
		t.Fatal("fixture pointer final traversal", pointer)
	}
}

func relationInputFromFixture(t *testing.T, root map[string]any) RelationInput {
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
	return RelationInput{
		ChangePackage:     raw(root["change_package"]),
		HumanDecision:     raw(root["human_decision"]),
		ExecutionRecords:  rawList(root["execution_records"]),
		AuthorityReceipts: rawList(root["authority_receipts"]),
		Outcome:           raw(root["outcome"]),
		OutcomeCheck:      raw(root["outcome_check"]),
		LifecycleEvents:   rawList(root["lifecycle_events"]),
		Checkpoint:        raw(root["checkpoint"]),
	}
}

func hasRelationFinding(findings []RelationFinding, code, relation string) bool {
	for _, finding := range findings {
		if finding.Code == code && (relation == "" || finding.Relation == relation) {
			return true
		}
	}
	return false
}
