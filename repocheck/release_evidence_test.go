package repocheck_test

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"github.com/open-agent-ops/spec/repocheck"
	"os"
	"sort"
	"testing"
)

func evidenceArchive(t *testing.T, files map[string][]byte, extra *tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	names := []string{}
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if e := w.WriteHeader(&tar.Header{Name: n, Mode: 0600, Size: int64(len(files[n])), Typeflag: tar.TypeReg}); e != nil {
			t.Fatal(e)
		}
		if _, e := w.Write(files[n]); e != nil {
			t.Fatal(e)
		}
	}
	if extra != nil {
		if e := w.WriteHeader(extra); e != nil {
			t.Fatal(e)
		}
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestReleaseEvidenceBoundary(t *testing.T) {
	binding, records, streams := gateSet()
	p := policy()
	for _, name := range p.Full {
		g := records[0]
		g.Job = name
		streams[name+".stdout"] = []byte("complete " + name)
		streams[name+".stderr"] = []byte{}
		g.Stdout = repocheck.Hash(streams[name+".stdout"])
		records = append(records, g)
	}
	files := map[string][]byte{}
	for n, b := range streams {
		files[n] = b
	}
	for _, g := range records {
		b, e := json.Marshal(g)
		if e != nil {
			t.Fatal(e)
		}
		files[g.Job+".json"] = b
	}
	proposal := repocheck.Proposal{Candidate: binding.Candidate, Policy: binding.Policy, Lock: binding.Lock, Workflow: binding.Workflow, Run: binding.Run, Attempt: 1}
	schema, e := os.ReadFile("../process/schemas/gate-record.schema.json")
	if e != nil {
		t.Fatal(e)
	}
	check := func(files map[string][]byte, extra *tar.Header) error {
		return repocheck.ReleaseEvidence(evidenceArchive(t, files, extra), p, proposal, binding.Base, schema)
	}
	if e := check(files, nil); e != nil {
		t.Fatal("complete same-run evidence", e)
	}
	for _, scenario := range []string{"missing_stream", "truncated_stream", "foreign_run", "failure", "symlink", "duplicate", "unexpected"} {
		t.Run(scenario, func(t *testing.T) {
			bad := map[string][]byte{}
			for n, b := range files {
				bad[n] = append([]byte(nil), b...)
			}
			var extra *tar.Header
			switch scenario {
			case "missing_stream":
				delete(bad, "policy.stderr")
			case "truncated_stream":
				bad["policy.stdout"] = []byte("complete")
			case "foreign_run", "failure":
				var g repocheck.Gate
				if json.Unmarshal(bad["policy.json"], &g) != nil {
					t.Fatal("fixture")
				}
				if scenario == "foreign_run" {
					g.Run = "99"
				} else {
					g.Result = "failure"
				}
				bad["policy.json"], _ = json.Marshal(g)
			case "symlink":
				delete(bad, "policy.stderr")
				extra = &tar.Header{Name: "policy.stderr", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}
			case "duplicate":
				extra = &tar.Header{Name: "policy.stderr", Typeflag: tar.TypeReg}
			case "unexpected":
				delete(bad, "policy.stderr")
				bad["foreign.stderr"] = nil
			}
			if check(bad, extra) == nil {
				t.Fatal("incomplete/hostile release evidence admitted")
			}
		})
	}
	t.Log("same_run_complete=pass; missing,truncated,foreign_run,failure,symlink,duplicate,unexpected=denied; provider_calls=0")
}

func TestReleaseSourceBoundary(t *testing.T) {
	p := policy()
	p.Files = append(p.Files, repocheck.FileRule{Path: "composition-manifest.json", Class: "metadata", License: "Apache-2.0"})
	trusted := repocheck.Snapshot{"README.md": []byte("reviewed readme"), "GOVERNANCE.md": []byte("reviewed governance"), "composition-manifest.json": []byte(`{"maturity":"candidate","paths":["README.md","GOVERNANCE.md","composition-manifest.json"]}`)}
	proposal := repocheck.Proposal{Inventory: repocheck.Hash(trusted["composition-manifest.json"])}
	if repocheck.ReleaseSource(evidenceArchive(t, trusted, nil), trusted, p, proposal) != nil {
		t.Fatal("reviewed source denied")
	}
	bad := map[string][]byte{}
	for n, b := range trusted {
		bad[n] = b
	}
	bad["README.md"] = []byte("modified source")
	if repocheck.ReleaseSource(evidenceArchive(t, bad, nil), trusted, p, proposal) == nil {
		t.Fatal("self-hashed foreign source admitted")
	}
	delete(bad, "README.md")
	if repocheck.ReleaseSource(evidenceArchive(t, bad, nil), trusted, p, proposal) == nil {
		t.Fatal("missing source admitted")
	}
	t.Log("reviewed archive accepted; changed and omitted bytes denied before provider calls")
}
