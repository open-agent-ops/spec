package formal

import (
	_ "embed"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

//go:embed counterexamples.json
var counterexampleBytes []byte

type Counterexample struct {
	ID                 string   `json:"id"`
	Model              string   `json:"model"`
	RemovedAssumptions []string `json:"removed_assumptions"`
	Events             []string `json:"events"`
	Violates           []string `json:"violates"`
}

type CounterexampleCorpus struct {
	Format string           `json:"format"`
	Cases  []Counterexample `json:"cases"`
}

func Counterexamples() (CounterexampleCorpus, error) {
	var corpus CounterexampleCorpus
	d := json.NewDecoder(strings.NewReader(string(counterexampleBytes)))
	d.DisallowUnknownFields()
	if err := d.Decode(&corpus); err != nil {
		return CounterexampleCorpus{}, err
	}
	if corpus.Format != "agent-ops.formal-counterexamples@1.0.0" || len(corpus.Cases) == 0 {
		return CounterexampleCorpus{}, errors.New("counterexample corpus header invalid")
	}
	seen := map[string]bool{}
	for _, c := range corpus.Cases {
		if c.ID == "" || seen[c.ID] || len(c.Events) == 0 || len(c.RemovedAssumptions) == 0 || len(c.Violates) == 0 {
			return CounterexampleCorpus{}, errors.New("counterexample corpus case invalid")
		}
		seen[c.ID] = true
	}
	return corpus, nil
}

func ReplayCounterexample(c Counterexample) ([]string, error) {
	assumptions := DefaultC4Assumptions()
	for id, enabled := range DefaultEvidenceAssumptions() {
		assumptions[id] = enabled
	}
	assumptions = assumptions.Without(c.RemovedAssumptions...)
	evidence := InitialEvidenceState()
	c4 := InitialC4State()

	for _, raw := range c.Events {
		domain, name := c.Model, raw
		if before, after, ok := strings.Cut(raw, ":"); ok {
			domain, name = before, after
		}
		switch domain {
		case "evidence":
			var err error
			evidence, err = ApplyEvidence(evidence, EvidenceEvent(name), assumptions)
			if err != nil {
				return nil, err
			}
		case "c4":
			action := actionForName(name, c4, evidence.Facts(assumptions))
			var err error
			c4, err = ApplyC4(c4, action, assumptions)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unknown counterexample model")
		}
	}

	violations := append(EvidenceViolations(evidence, assumptions), C4Violations(c4, assumptions)...)
	sort.Strings(violations)
	return uniqueStrings(violations), nil
}

func actionForName(name string, state C4State, evidence EvidenceFacts) C4Action {
	switch name {
	case "observe_not_applied_partial":
		return C4Action{Kind: C4ObserveNotApplied, Evidence: partialFacts(state.CurrentTarget)}
	case "observe_not_applied_from_evidence":
		return C4Action{Kind: C4ObserveNotApplied, Evidence: evidence}
	case string(C4ObserveApplied):
		return C4Action{Kind: C4ObserveApplied, Evidence: completeFacts(state.CurrentTarget)}
	case string(C4OutcomeSuccess), string(C4OutcomeSuccessFalse):
		return C4Action{Kind: C4Event(name), Evidence: completeFacts(state.CurrentTarget)}
	default:
		return C4Action{Kind: C4Event(name)}
	}
}

func completeFacts(target TargetRef) EvidenceFacts {
	return EvidenceFacts{
		InterfaceID:           InterfaceID,
		RequirementID:         "effect-observation",
		ConsumerRule:          "c4-effect-rule@1.0.0",
		Applicable:            true,
		Target:                target,
		FreshAtUse:            true,
		Coverage:              CoverageComplete,
		DetectionCapable:      true,
		Disposition:           DispositionSupported,
		Sufficiency:           SufficiencySufficient,
		ProvenanceBound:       true,
		CommonCausesDisclosed: true,
	}
}

func partialFacts(target TargetRef) EvidenceFacts {
	f := completeFacts(target)
	f.Coverage = CoveragePartial
	f.Sufficiency = SufficiencyInsufficient
	return f
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
