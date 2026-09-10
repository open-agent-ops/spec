package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestConformanceAssessmentRequiresCompleteEvidence(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/index.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []fixture
	if json.Unmarshal(b, &fixtures) != nil {
		t.Fatal("index")
	}
	cases := []Case{}
	for _, f := range fixtures {
		b, err := os.ReadFile("testdata/" + f.Path)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, Case{f.Path, f.Schema, b, f.Accepted, f.Reason})
	}
	properties := []PropertyResult{}
	for i := 1; i <= 7; i++ {
		properties = append(properties, PropertyResult{ID: fmt.Sprintf("AOM04-FD-PROP-%02d", i), Registry: v.registry, Passed: true, Seed: "30404", Evidence: v.registry})
	}
	// Synthetic property records exercise the admission contract, not PBT evidence.
	if v.Assess(cases, properties).Status != "ok" {
		t.Fatal("complete fixture/evidence model")
	}
	if v.Assess(cases, properties[:6]).Status == "ok" {
		t.Fatal("missing property accepted")
	}
	properties[0].Registry = "stale"
	if v.Assess(cases, properties).Status == "ok" {
		t.Fatal("stale property accepted")
	}
	if v.Assess(cases[:27], nil).Status == "ok" {
		t.Fatal("partial conformance")
	}
}

// The private exact-target harness admits real property-run evidence separately.
// This record deliberately remains partial rather than inventing those results.
func TestConformanceCaseEvidence(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/index.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []fixture
	if json.Unmarshal(b, &fixtures) != nil || len(fixtures) != 28 {
		t.Fatal("index")
	}
	cases := []Case{}
	for _, f := range fixtures {
		if !ValidPath(f.Path) {
			t.Fatal("fixture path")
		}
		b, err := os.ReadFile("testdata/" + f.Path)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, Case{f.Path, f.Schema, b, f.Accepted, f.Reason})
	}
	a := v.Assess(cases, nil)
	if a.Compiled != 61 || len(a.Cases) != 28 || a.Status != "partial" {
		t.Fatal("case evidence incomplete")
	}
	for _, c := range a.Cases {
		if !c.Matched {
			t.Fatal("fixture mismatch", c.ID)
		}
	}
	encoded, err := a.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("AOM04_CASE_EVIDENCE=" + string(encoded))
}
