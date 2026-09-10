package repocheck_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/open-agent-ops/spec/repocheck"
)

func contract(t *testing.T, name string, fields map[string]any) {
	t.Helper()
	fields["schema"] = "aom04a." + name + ".v1"
	schema, e := os.ReadFile("../process/schemas/" + name + ".schema.json")
	if e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(fields)
	if e != nil {
		t.Fatal(e)
	}
	if repocheck.ValidateRecord(schema, b) != nil {
		t.Fatal("valid record rejected", name)
	}
	t.Log("fixture-only scenario=valid_closed_record result=accepted")
	for key, value := range fields {
		delete(fields, key)
		bad, _ := json.Marshal(fields)
		if repocheck.ValidateRecord(schema, bad) == nil {
			t.Fatal("missing accepted", key)
		}
		fields[key] = value
	}
	fields["approved"] = true
	bad, _ := json.Marshal(fields)
	delete(fields, "approved")
	if repocheck.ValidateRecord(schema, bad) == nil {
		t.Fatal("unknown authority field accepted")
	}
	if repocheck.ValidateRecord(schema, append(b, []byte(" {}")...)) == nil {
		t.Fatal("trailing record accepted")
	}
	dup := append([]byte(`{"schema":"wrong",`), b[1:]...)
	if repocheck.ValidateRecord(schema, dup) == nil {
		t.Fatal("duplicate key accepted")
	}
	t.Log("fixture-only scenarios=missing_each_field,unknown_authority,duplicate_key,trailing_record result=rejected")
}
func TestSubmissionContract(t *testing.T) {
	contract(t, "submission", map[string]any{"kind": "fix", "head": strings.Repeat("a", 40), "base": strings.Repeat("b", 40), "paths": []string{"README.md"}, "actor": "11", "link": "PR1", "revision": "v0.4.0", "description": "reproduction", "compatibility": "unchanged", "bilingual": "unchanged", "license_ack": true})
}
func TestReviewContract(t *testing.T) {
	contract(t, "review-decision", map[string]any{"reviewer": "12", "role": "owner", "candidate": strings.Repeat("a", 40), "decision": "approve", "timestamp": "2026-09-08T00:00:00Z", "source_receipt": strings.Repeat("a", 64)})
}
func TestGateRecordContract(t *testing.T) {
	contract(t, "gate-record", map[string]any{"repository": repocheck.Repository, "candidate": strings.Repeat("a", 40), "base": strings.Repeat("b", 40), "policy": strings.Repeat("a", 64), "lock": strings.Repeat("a", 64), "workflow": strings.Repeat("a", 40), "run": "11", "job": "policy", "attempt": 1, "result": "success", "stdout": strings.Repeat("a", 64), "stderr": strings.Repeat("b", 64)})
}
func TestReleaseProposalContract(t *testing.T) {
	contract(t, "release-proposal", map[string]any{"repository": repocheck.Repository, "candidate": strings.Repeat("a", 40), "version": "v0.4.0", "tree": strings.Repeat("a", 40), "inventory": strings.Repeat("a", 64), "assets": []any{map[string]any{"name": "source.tar.gz", "sha256": strings.Repeat("a", 64), "size": 1}}, "sbom": strings.Repeat("a", 64), "workflow": strings.Repeat("a", 40), "run": "11", "attempt": 1, "lock": strings.Repeat("a", 64), "policy": strings.Repeat("a", 64), "checked_at": "2026-09-08T00:00:00Z", "expires_at": "2026-09-09T00:00:00Z"})
}
func TestPublicationReceiptContract(t *testing.T) {
	contract(t, "publication-receipt", map[string]any{"repository": repocheck.Repository, "candidate": strings.Repeat("a", 40), "version": "v0.4.0", "proposal": strings.Repeat("a", 64), "canonical": "verified", "mirror": "unknown", "assets": []any{}, "observed_ids": []string{"123"}, "fixture_only": true})
}
func TestInputBounds(t *testing.T) {
	for _, s := range []string{`{"x":1,"x":2}`, `{} {}`, strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34), `{"x":"` + strings.Repeat("x", 4097) + `"}`, strings.Repeat(" ", repocheck.MaxInput+1)} {
		if _, e := repocheck.JSON([]byte(s)); e == nil {
			t.Fatal("hostile JSON accepted")
		}
	}
	for _, s := range []string{"a: &a {}\nb: *a", "a: 1\na: 2", "a: !!python/object {}", "a: {}\n---\nb: {}", "a: {<<: {b: 1}}"} {
		if _, e := repocheck.YAML([]byte(s)); e == nil {
			t.Fatal("hostile YAML accepted", s)
		}
	}
	for _, s := range []string{"../a", "/a", "a/../b", "a//b", "a\\b", "a\x00b"} {
		if repocheck.SafePath(s) {
			t.Fatal("hostile path accepted")
		}
	}
}
