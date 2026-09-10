package repocheck_test

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
	"go.yaml.in/yaml/v3"
)

func policy() repocheck.Policy {
	return repocheck.Policy{Schema: "aom04a.policy.v1", Repository: repocheck.Repository, Files: []repocheck.FileRule{{Path: "README.md", Class: "documentation", License: "CC-BY-4.0"}, {Path: "GOVERNANCE.md", Class: "documentation", License: "CC-BY-4.0", Normative: true}}, Required: []string{"policy", "conformance", "docs", "supply-chain", "aggregate"}, Full: []string{"heavy", "reproducibility"}, Owners: "open-agent-ops/owners", Maintainers: "open-agent-ops/maintainers"}
}
func submission() repocheck.Submission {
	return repocheck.Submission{Schema: "aom04a.submission.v1", Kind: "fix", Head: strings.Repeat("a", 40), Base: strings.Repeat("b", 40), Paths: []string{"README.md"}, Actor: "11", Link: "PR1", Revision: "v0.4.0", Description: "reproduce", Compatibility: "unchanged", Bilingual: "unchanged", LicenseAck: true}
}
func TestRule01(t *testing.T) {
	s := submission()
	if repocheck.Intake(s, policy()).State != "received" {
		t.Fatal("fix not received")
	}
	s.LicenseAck = false
	if repocheck.Intake(s, policy()).State != "needs_info" {
		t.Fatal("missing license")
	}
	t.Log("fixture-only cases=small_fix_received,missing_license_needs_info writes=0")
}
func TestRule02(t *testing.T) {
	p := policy()
	if n, e := repocheck.Classify([]string{"GOVERNANCE.md"}, p); e != nil || !n {
		t.Fatal("authority downgraded")
	}
	for _, paths := range [][]string{{"unknown.go"}, {"README.md", "README.md"}, {"../README.md"}} {
		if _, e := repocheck.Classify(paths, p); e == nil {
			t.Fatal("unclassified path admitted")
		}
	}
	t.Log("fixture-only cases=authority_requires_owner,unknown_duplicate_traversal_denied")
}
func TestRule03(t *testing.T) {
	s := submission()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	b := repocheck.ReviewBindings{Roles: map[string]string{"12": "owner"}, Now: now, HeadTime: now.Add(-time.Hour)}
	r := repocheck.Review{Schema: "aom04a.review-decision.v1", Reviewer: "12", Role: "owner", Candidate: s.Head, Decision: "approve", Timestamp: now.Add(-time.Minute), SourceReceipt: strings.Repeat("c", 64)}
	if repocheck.ReviewReady(s, policy(), []repocheck.Review{r}, b) != nil {
		t.Fatal("independent approval denied")
	}
	for _, mutate := range []func(*repocheck.Review){func(r *repocheck.Review) { r.Reviewer = s.Actor }, func(r *repocheck.Review) { r.Candidate = s.Base }, func(r *repocheck.Review) { r.Decision = "request_changes" }, func(r *repocheck.Review) { r.Timestamp = now.Add(-2 * time.Hour) }, func(r *repocheck.Review) { r.Reviewer = "99" }} {
		bad := r
		mutate(&bad)
		if repocheck.ReviewReady(s, policy(), []repocheck.Review{bad}, b) == nil {
			t.Fatal("invalid review admitted")
		}
	}
	t.Log("fixture-only cases=independent_current_pass,self_stale_unbound_changes_requested_denied")
}
func gateSet() (repocheck.EvidenceBinding, []repocheck.Gate, map[string][]byte) {
	b := repocheck.EvidenceBinding{Candidate: strings.Repeat("a", 40), Base: strings.Repeat("b", 40), Policy: strings.Repeat("c", 64), Lock: strings.Repeat("d", 64), Workflow: strings.Repeat("a", 40), Run: "1", Attempt: 1}
	records := []repocheck.Gate{}
	streams := map[string][]byte{}
	for _, name := range policy().Required {
		stdout := []byte("complete " + name)
		stderr := []byte{}
		streams[name+".stdout"] = stdout
		streams[name+".stderr"] = stderr
		records = append(records, repocheck.Gate{Schema: "aom04a.gate-record.v1", Repository: repocheck.Repository, Candidate: b.Candidate, Base: b.Base, Policy: b.Policy, Lock: b.Lock, Workflow: b.Workflow, Run: b.Run, Job: name, Attempt: 1, Result: "success", Stdout: repocheck.Hash(stdout), Stderr: repocheck.Hash(stderr)})
	}
	return b, records, streams
}
func TestRule04(t *testing.T) {
	b, records, streams := gateSet()
	if repocheck.EvidenceReady(records, policy(), b, streams) != nil {
		t.Fatal("complete evidence denied")
	}
	for _, status := range []string{"skipped", "neutral", "failure", "unknown", "cancelled"} {
		bad := append([]repocheck.Gate(nil), records...)
		bad[0].Result = status
		if repocheck.EvidenceReady(bad, policy(), b, streams) == nil {
			t.Fatal("non-success admitted")
		}
	}
	if repocheck.EvidenceReady(records[1:], policy(), b, streams) == nil {
		t.Fatal("missing admitted")
	}
	streams["policy.stdout"] = []byte("truncated")
	if repocheck.EvidenceReady(records, policy(), b, streams) == nil {
		t.Fatal("truncated admitted")
	}
	t.Log("fixture-only cases=complete_pass,missing_non_success_truncated_denied")
}
func toolLock(t *testing.T) repocheck.ToolLock {
	t.Helper()
	b, e := os.ReadFile("../process/toolchain.lock.json")
	if e != nil {
		t.Fatal(e)
	}
	var l repocheck.ToolLock
	if json.Unmarshal(b, &l) != nil {
		t.Fatal("lock")
	}
	return l
}
func TestRule05(t *testing.T) {
	l := toolLock(t)
	pins := map[string]string{}
	for _, a := range l.Actions {
		pins[a.Repository] = a.Repository + "@" + a.Commit
	}
	for _, release := range []bool{false, true} {
		m := repocheck.WorkflowModel(l, pins, release)
		b, e := yaml.Marshal(m)
		if e != nil || repocheck.Workflow(b, l, release) != nil {
			t.Fatal("valid workflow denied")
		}
		jobs := m["jobs"].(map[string]any)
		for _, value := range jobs {
			job := value.(map[string]any)
			for _, key := range []string{"container", "services"} {
				job[key] = map[string]any{"image": "golang:latest"}
				altered, _ := yaml.Marshal(m)
				if repocheck.Workflow(altered, l, release) == nil {
					t.Fatal("container execution admitted", key)
				}
				delete(job, key)
			}
		}
		m["permissions"] = map[string]any{"contents": "write"}
		bad, _ := yaml.Marshal(m)
		if repocheck.Workflow(bad, l, release) == nil {
			t.Fatal("write permission admitted")
		}
	}
	m := repocheck.WorkflowModel(l, pins, false)
	m["on"] = map[string]any{"pull_request_target": nil}
	b, _ := yaml.Marshal(m)
	if repocheck.Workflow(b, l, false) == nil {
		t.Fatal("privileged trigger")
	}
	t.Log("fixture-only cases=closed_ci_release_pass,write_and_privileged_trigger_denied")
}
func TestRule06(t *testing.T) {
	p := policy()
	s := repocheck.Snapshot{"README.md": []byte("readme"), "GOVERNANCE.md": []byte("governance")}
	p.Files = append(p.Files, repocheck.FileRule{Path: "composition-manifest.json", Class: "metadata", License: "Apache-2.0"})
	s["composition-manifest.json"] = []byte(`{"paths":["README.md","GOVERNANCE.md","composition-manifest.json"]}`)
	if repocheck.Composition(s, p) != nil {
		t.Fatal("complete snapshot")
	}
	s["extra"] = []byte("unknown")
	if repocheck.Composition(s, p) == nil {
		t.Fatal("unknown file")
	}
	for _, name := range []string{"nested/internal/x.go", "nested/review/x.json", strings.Repeat("x", 257), "nested/docs/foundation/x.md"} {
		if repocheck.ExportPath(name) {
			t.Fatal("exclusion", name)
		}
	}
	t.Log("fixture-only cases=closed_inventory_pass,extra_file_recursive_exclusion_denied")
}
func TestRule11(t *testing.T) {
	for _, root := range []string{".", "../schemas", "../conformance"} {
		e := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, e := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if e != nil {
				return e
			}
			for _, i := range f.Imports {
				v := strings.Trim(i.Path.Value, "\"")
				if strings.Contains(v, "releaseops") || strings.Contains(v, "gitinsky") || v == "net/http" || v == "os/exec" || v == "os" {
					t.Errorf("capability dependency %s", path)
				}
			}
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
	}
	t.Log("source_import_scan=offline_public_boundary_pass")
}
func TestRule12(t *testing.T) {
	s := submission()
	s.Description = "SYNTHETIC_SECRET_CANARY"
	s.Paths = []string{"unknown"}
	out := repocheck.Intake(s, policy())
	b, _ := json.Marshal(out)
	if strings.Contains(string(b), s.Description) || out.Reason == "" || out.Role == "" || out.Next == "" {
		t.Fatal("unsanitized outcome")
	}
	t.Log("fixture-only cases=sanitized_reason_role_next_action_pass")
}
