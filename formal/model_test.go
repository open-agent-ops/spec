package formal_test

import (
	"slices"
	"sort"
	"testing"

	"github.com/open-agent-ops/spec/formal"
)

func TestCatalogs(t *testing.T) {
	want := map[string]int{
		"agent-ops.c4-evidence-interface@1.0.0": 7,
		"agent-ops.c4-relations@1.0.0":          13,
		"agent-ops.evidence-relations@1.0.0":    14,
	}
	for _, name := range formal.CatalogNames() {
		catalog, err := formal.OpenCatalog(name)
		if err != nil {
			t.Fatal(name, err)
		}
		if len(catalog.Relations) != want[catalog.CatalogID] {
			t.Fatalf("%s has %d relations", catalog.CatalogID, len(catalog.Relations))
		}
	}
}

func TestBoundedModels(t *testing.T) {
	for name, result := range map[string]formal.CheckResult{
		"evidence": formal.CheckEvidenceModel(8),
		"c4":       formal.CheckC4Model(8),
	} {
		if result.Violation != "" {
			t.Fatalf("%s violation %s trace=%v", name, result.Violation, result.Trace)
		}
		if result.States < 10 || result.Transitions < 10 {
			t.Fatalf("%s exploration unexpectedly small: %+v", name, result)
		}
		t.Logf("model=%s depth=%d states=%d transitions=%d", name, result.Depth, result.States, result.Transitions)
	}
}

func TestRetainedCounterexamples(t *testing.T) {
	corpus, err := formal.Counterexamples()
	if err != nil {
		t.Fatal(err)
	}
	for _, counterexample := range corpus.Cases {
		counterexample := counterexample
		t.Run(counterexample.ID, func(t *testing.T) {
			got, err := formal.ReplayCounterexample(counterexample)
			if err != nil {
				t.Fatal(err)
			}
			want := append([]string(nil), counterexample.Violates...)
			sort.Strings(want)
			if !slices.Equal(got, want) {
				t.Fatalf("violations=%v want=%v", got, want)
			}
			t.Logf("removed=%v trace=%v violations=%v", counterexample.RemovedAssumptions, counterexample.Events, got)
		})
	}
}

func TestInterfaceDoesNotGrantAuthority(t *testing.T) {
	facts := formal.EvidenceFacts{
		InterfaceID:           formal.InterfaceID,
		RequirementID:         "effect-observation",
		ConsumerRule:          "c4-effect-rule@1.0.0",
		Applicable:            true,
		Target:                formal.TargetRef{UID: "target-1", Generation: 1, Epoch: 1, Versions: "resource=1"},
		FreshAtUse:            true,
		Coverage:              formal.CoverageComplete,
		DetectionCapable:      true,
		Disposition:           formal.DispositionSupported,
		Sufficiency:           formal.SufficiencySufficient,
		ProvenanceBound:       true,
		CommonCausesDisclosed: true,
	}
	state := formal.InitialC4State()
	if _, err := formal.ApplyC4(state, formal.C4Action{Kind: formal.C4Admit, Evidence: facts}, formal.DefaultC4Assumptions()); err == nil {
		t.Fatal("evidence facts created C4 authority")
	}
}
