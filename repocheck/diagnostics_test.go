package repocheck_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/open-agent-ops/spec/repocheck"
	"go.yaml.in/yaml/v3"
)

// Every rejection keeps its sentinel class for exit-code mapping and names the
// cause so a failed gate is actionable from its stderr alone.
func TestDiagnosticsNameTheCause(t *testing.T) {
	expect := func(t *testing.T, e error, sentinel error, fragments ...string) {
		t.Helper()
		if e == nil {
			t.Fatal("expected an error")
		}
		if !errors.Is(e, sentinel) {
			t.Fatalf("error %q is not %v", e, sentinel)
		}
		for _, f := range fragments {
			if !strings.Contains(e.Error(), f) {
				t.Fatalf("error %q lacks %q", e, f)
			}
		}
	}

	p := policy()
	p.Files = append(p.Files, repocheck.FileRule{Path: "composition-manifest.json", Class: "metadata", License: "Apache-2.0"})
	complete := func() repocheck.Snapshot {
		return repocheck.Snapshot{"README.md": []byte("readme"), "GOVERNANCE.md": []byte("governance"), "composition-manifest.json": []byte(`{"paths":["README.md","GOVERNANCE.md","composition-manifest.json"]}`)}
	}

	s := complete()
	s["extra.txt"] = []byte("x")
	expect(t, repocheck.Composition(s, p), repocheck.ErrRejected, "snapshot has 4 files, policy lists 3", "not in policy: extra.txt")

	s = complete()
	delete(s, "GOVERNANCE.md")
	s["other.md"] = []byte("x")
	expect(t, repocheck.Composition(s, p), repocheck.ErrRejected, "policy path missing from snapshot: GOVERNANCE.md")

	s = complete()
	s["README.md"] = nil
	expect(t, repocheck.Composition(s, p), repocheck.ErrRejected, "empty file: README.md")

	s = complete()
	s["composition-manifest.json"] = []byte(`{"paths":["README.md","README.md","composition-manifest.json"]}`)
	expect(t, repocheck.Composition(s, p), repocheck.ErrRejected, "composition-manifest.json path duplicated: README.md")

	protected := p
	protected.Files = append([]repocheck.FileRule(nil), p.Files...)
	protected.Files[0].ProtectedHash = strings.Repeat("0", 64)
	expect(t, repocheck.Composition(complete(), protected), repocheck.ErrRejected, "protected_hash mismatch: README.md")

	bad := p
	bad.Required = []string{"policy"}
	expect(t, bad.Validate(), repocheck.ErrInput, "required/full gate lists")

	l := toolLock(t)
	pins := map[string]string{}
	for _, a := range l.Actions {
		pins[a.Repository] = a.Repository + "@" + a.Commit
	}
	m := repocheck.WorkflowModel(l, pins, false)
	m["jobs"].(map[string]any)["docs"].(map[string]any)["timeout-minutes"] = 25
	b, err := yaml.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	expect(t, repocheck.Workflow(b, l, false), repocheck.ErrRejected, "$.jobs.docs.timeout-minutes: got 25, want 20")
	delete(m["jobs"].(map[string]any), "docs")
	b, _ = yaml.Marshal(m)
	expect(t, repocheck.Workflow(b, l, false), repocheck.ErrRejected, "$.jobs.docs: missing")

	weak := l
	weak.GoVersion = "1.0.0"
	expect(t, repocheck.Workflow(b, weak, false), repocheck.ErrInput, "toolchain lock header")

	needs := []byte(`{"policy":{"result":"success"},"conformance":{"result":"success"},"docs":{"result":"failure"},"supply-chain":{"result":"success"},"heavy":{"result":"skipped"},"reproducibility":{"result":"skipped"}}`)
	expect(t, repocheck.NeedsSuccess(needs, false), repocheck.ErrRejected, `job docs result "failure", want "success"`)
	expect(t, repocheck.NeedsSuccess(needs, true), repocheck.ErrRejected, "job docs")

	binding, records, streams := gateSet()
	records[0].Result = "failure"
	expect(t, repocheck.EvidenceReady(records, policy(), binding, streams), repocheck.ErrRejected, "evidence policy: result \"failure\"")

	schema, err := os.ReadFile("../process/schemas/gate-record.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	expect(t, repocheck.ValidateRecord(schema, []byte(`{"schema":"aom04a.gate-record.v1"}`)), repocheck.ErrInput, "record does not satisfy its process schema")
}
