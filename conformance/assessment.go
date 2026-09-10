package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type Case struct {
	ID       string
	Schema   string
	Bytes    []byte
	Expected bool
	Reason   string
}
type PropertyResult struct {
	ID       string `json:"id"`
	Registry string `json:"registry"`
	Passed   bool   `json:"passed"`
	Seed     string `json:"seed"`
	Evidence string `json:"evidence"`
}
type CaseResult struct {
	ID          string `json:"id"`
	InputSHA256 string `json:"input_sha256"`
	Result      Result `json:"result"`
	Expected    bool   `json:"expected"`
	Reason      string `json:"reason"`
	Matched     bool   `json:"matched"`
}
type Assessment struct {
	Registry   string           `json:"registry"`
	Status     string           `json:"status"`
	Compiled   int              `json:"compiled"`
	Cases      []CaseResult     `json:"cases"`
	Properties []PropertyResult `json:"properties"`
	Claim      string           `json:"claim"`
}

// Assess checks fixtures and completeness of admitted property results. Callers
// authenticate property evidence against the exact test target before admission.
func (v *Validator) Assess(cases []Case, properties []PropertyResult) Assessment {
	if len(cases) > 28 || len(properties) > 7 {
		return Assessment{Status: "partial", Claim: "Input exceeds the declared conformance set."}
	}
	out := Assessment{Status: "partial", Claim: "Schema contracts and declared properties only; no provenance, authority, operational safety or runtime readiness claim.", Cases: []CaseResult{}, Properties: append([]PropertyResult(nil), properties...)}
	if v == nil {
		return out
	}
	out.Registry = v.registry
	out.Compiled = len(v.compiled)
	complete := out.Compiled == 61 && len(cases) == 28 && len(properties) == 7
	seen := map[string]bool{}
	families := map[string][2]int{}
	reasonClasses := map[string]bool{}
	for _, c := range cases {
		if !ValidPath(c.ID) || seen[c.ID] {
			complete = false
		}
		seen[c.ID] = true
		result := v.Validate(c.Schema, c.Bytes)
		matched := result.Accepted == c.Expected
		if !c.Expected {
			matched = matched && c.Reason != "" && containsReason(result.Reasons, c.Reason) && !containsReason(result.Reasons, "input_invalid")
		}
		if !matched {
			complete = false
		}
		if !c.Expected {
			key := c.Schema + ":" + c.Reason
			if reasonClasses[key] && c.Schema != "https://agent-ops.ru/schemas/run_request.schema.json" {
				complete = false
			}
			reasonClasses[key] = true
		}
		sum := sha256.Sum256(c.Bytes)
		out.Cases = append(out.Cases, CaseResult{c.ID, hex.EncodeToString(sum[:]), result, c.Expected, c.Reason, matched})
		counts := families[c.Schema]
		if c.Expected {
			counts[0]++
		} else {
			counts[1]++
		}
		families[c.Schema] = counts
	}
	for _, family := range []string{"operational_intent", "evidence_bundle", "eval_result", "governance_result", "human_decision", "outcome_check"} {
		if families["https://agent-ops.ru/schemas/foundation_"+family+".schema.json"] != [2]int{1, 3} {
			complete = false
		}
	}
	if families["https://agent-ops.ru/schemas/run_request.schema.json"] != [2]int{1, 3} {
		complete = false
	}
	seenProps := map[string]bool{}
	for _, p := range properties {
		switch p.ID {
		case "AOM04-FD-PROP-01", "AOM04-FD-PROP-02", "AOM04-FD-PROP-03", "AOM04-FD-PROP-04", "AOM04-FD-PROP-05", "AOM04-FD-PROP-06", "AOM04-FD-PROP-07":
		default:
			complete = false
		}
		if seenProps[p.ID] || p.Registry != v.registry || !p.Passed || p.Seed == "" || !digestString(p.Evidence) {
			complete = false
		}
		seenProps[p.ID] = true
	}
	sort.Slice(out.Cases, func(i, j int) bool { return out.Cases[i].ID < out.Cases[j].ID })
	sort.Slice(out.Properties, func(i, j int) bool { return out.Properties[i].ID < out.Properties[j].ID })
	if complete {
		out.Status = "ok"
	}
	return out
}
func containsReason(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func digestString(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == s
}
func (a Assessment) Canonical() ([]byte, error) { return json.Marshal(a) }
