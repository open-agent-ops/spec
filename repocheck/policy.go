package repocheck

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type FileRule struct {
	Path          string `json:"path"`
	Class         string `json:"class"`
	License       string `json:"license"`
	Normative     bool   `json:"normative"`
	ProtectedHash string `json:"protected_hash"`
}
type Policy struct {
	Schema      string     `json:"schema"`
	Repository  string     `json:"repository"`
	Files       []FileRule `json:"files"`
	Required    []string   `json:"required"`
	Full        []string   `json:"full"`
	Owners      string     `json:"owners"`
	Maintainers string     `json:"maintainers"`
}
type Outcome struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
	Role   string `json:"role"`
	Next   string `json:"next_action"`
}

func (p Policy) Validate() error {
	if p.Schema != "aom04a.policy.v1" || p.Repository != Repository || p.Owners != "open-agent-ops/owners" || p.Maintainers != "open-agent-ops/maintainers" {
		return Invalidf("policy header (schema, repository, owners, maintainers)")
	}
	if len(p.Files) == 0 || len(p.Files) > 256 {
		return Invalidf("policy files count %d outside 1..256", len(p.Files))
	}
	seen := map[string]bool{}
	for _, f := range p.Files {
		switch {
		case !ExportPath(f.Path):
			return Invalidf("policy path not exportable: %s", f.Path)
		case seen[f.Path]:
			return Invalidf("policy path duplicated: %s", f.Path)
		case f.License == "":
			return Invalidf("policy license missing: %s", f.Path)
		case f.ProtectedHash != "" && !Digest(f.ProtectedHash):
			return Invalidf("policy protected_hash malformed: %s", f.Path)
		}
		seen[f.Path] = true
	}
	required := []string{"policy", "conformance", "docs", "supply-chain", "aggregate"}
	if len(p.Required) != len(required) || len(p.Full) != 2 || p.Full[0] != "heavy" || p.Full[1] != "reproducibility" {
		return Invalidf("policy required/full gate lists differ from the fixed catalog")
	}
	for i, s := range required {
		if p.Required[i] != s {
			return Invalidf("policy required[%d]=%q, want %q", i, p.Required[i], s)
		}
	}
	return nil
}

// Classification uses trusted base policy and actual diff paths. Labels and
// issue prose have no place in this authority decision.
func Classify(paths []string, p Policy) (bool, error) {
	if p.Validate() != nil || len(paths) == 0 || len(paths) > 256 {
		return false, ErrInput
	}
	known := map[string]FileRule{}
	for _, f := range p.Files {
		known[f.Path] = f
	}
	seen := map[string]bool{}
	normative := false
	for _, s := range paths {
		f, ok := known[s]
		if !ok || seen[s] || !SafePath(s) {
			return false, ErrRejected
		}
		seen[s] = true
		normative = normative || f.Normative
	}
	return normative, nil
}
func Intake(s Submission, p Policy) Outcome {
	missing := Outcome{"needs_info", "submission_incomplete", "maintainer", "supply_required_fields"}
	if s.Schema != "aom04a.submission.v1" || !Commit(s.Head) || !Commit(s.Base) || s.Actor == "" || s.Description == "" || s.Revision == "" || s.Compatibility == "" || s.Bilingual == "" || !s.LicenseAck {
		return missing
	}
	if s.Kind != "bug" && s.Kind != "proposal" && s.Kind != "fix" {
		return missing
	}
	normative, e := Classify(s.Paths, p)
	if e != nil {
		return Outcome{"needs_info", "paths_unclassified", "owner", "review_trusted_path_policy"}
	}
	if normative {
		return Outcome{"proposed", "owner_direction_required", "owner", "record_proposal_decision"}
	}
	return Outcome{"received", "triage_required", "maintainer", "record_reason_and_next_action"}
}

// ReviewBindings are trusted host observations. Self-asserted roles in a
// submission or review record never establish membership.
type ReviewBindings struct {
	Roles    map[string]string
	Now      time.Time
	HeadTime time.Time
}

func ReviewReady(s Submission, p Policy, reviews []Review, b ReviewBindings) error {
	normative, e := Classify(s.Paths, p)
	if e != nil {
		return e
	}
	if len(reviews) == 0 || len(reviews) > 1000 || b.Now.IsZero() || b.HeadTime.IsZero() || b.HeadTime.After(b.Now) {
		return ErrRejected
	}
	latest := map[string]Review{}
	for _, r := range reviews {
		if r.Schema != "aom04a.review-decision.v1" || !Digest(r.SourceReceipt) || r.Timestamp.IsZero() || r.Timestamp.After(b.Now) {
			return ErrInput
		}
		if r.Candidate != s.Head || r.Timestamp.Before(b.HeadTime) {
			continue
		}
		if r.Decision != "approve" && r.Decision != "request_changes" && r.Decision != "dismiss" {
			return ErrInput
		}
		role, ok := b.Roles[r.Reviewer]
		if !ok || role != r.Role || r.Reviewer == s.Actor {
			continue
		}
		if old, ok := latest[r.Reviewer]; ok {
			if old.Timestamp.Equal(r.Timestamp) {
				return ErrRejected
			}
			if old.Timestamp.After(r.Timestamp) {
				continue
			}
		}
		latest[r.Reviewer] = r
	}
	independent, owner := false, false
	for _, r := range latest {
		if r.Decision == "request_changes" {
			return ErrRejected
		}
		if r.Decision == "approve" {
			independent = true
			owner = owner || r.Role == "owner"
		}
	}
	if !independent || (normative && !owner) {
		return ErrRejected
	}
	return nil
}

type EvidenceBinding struct {
	Candidate, Base, Policy, Lock, Workflow, Run string
	Attempt                                      int
	Full                                         bool
}

// EvidenceReady verifies complete streams supplied independently of records.
func EvidenceReady(records []Gate, p Policy, b EvidenceBinding, streams map[string][]byte) error {
	return evidenceReady(records, p, b, streams, false)
}

// PreAggregateReady requires all independent gates before emitting aggregate itself.
func PreAggregateReady(records []Gate, p Policy, b EvidenceBinding, streams map[string][]byte) error {
	return evidenceReady(records, p, b, streams, true)
}
func evidenceReady(records []Gate, p Policy, b EvidenceBinding, streams map[string][]byte, pre bool) error {
	if e := p.Validate(); e != nil {
		return Invalidf("evidence: %v", e)
	}
	if !Commit(b.Candidate) || !Commit(b.Base) || !Commit(b.Workflow) || !Digest(b.Policy) || !Digest(b.Lock) || b.Run == "" || b.Attempt < 1 {
		return Invalidf("evidence binding malformed (candidate/base/workflow commits, policy/lock digests, run, attempt)")
	}
	needed := append([]string(nil), p.Required...)
	if pre {
		needed = needed[:len(needed)-1]
	}
	if b.Full {
		needed = append(needed, p.Full...)
	}
	if len(records) != len(needed) {
		return Rejectedf("evidence: %d gate records, %d needed", len(records), len(needed))
	}
	seen := map[string]bool{}
	for _, r := range records {
		switch {
		case r.Schema != "aom04a.gate-record.v1":
			return Rejectedf("evidence %s: unexpected record schema", r.Job)
		case r.Repository != Repository:
			return Rejectedf("evidence %s: repository differs", r.Job)
		case r.Candidate != b.Candidate, r.Base != b.Base, r.Workflow != b.Workflow, r.Run != b.Run, r.Attempt != b.Attempt:
			return Rejectedf("evidence %s: candidate/base/workflow/run/attempt differ from binding", r.Job)
		case r.Policy != b.Policy, r.Lock != b.Lock:
			return Rejectedf("evidence %s: policy or lock digest differs from binding", r.Job)
		case r.Result != "success":
			return Rejectedf("evidence %s: result %q", r.Job, r.Result)
		case seen[r.Job]:
			return Rejectedf("evidence %s: duplicate record", r.Job)
		}
		out, ok := streams[r.Job+".stdout"]
		if !ok || Hash(out) != r.Stdout {
			return Rejectedf("evidence %s: stdout stream missing or digest differs", r.Job)
		}
		errout, ok := streams[r.Job+".stderr"]
		if !ok || Hash(errout) != r.Stderr {
			return Rejectedf("evidence %s: stderr stream missing or digest differs", r.Job)
		}
		seen[r.Job] = true
	}
	for _, n := range needed {
		if !seen[n] {
			return Rejectedf("evidence: gate record missing: %s", n)
		}
	}
	return nil
}

// Snapshot contains only candidate regular-file bytes supplied by a bounded
// caller. Symlinks are rejected at the file-system boundary before reading.
type Snapshot map[string][]byte

func Composition(s Snapshot, p Policy) error {
	if e := p.Validate(); e != nil {
		return Rejectedf("composition: %v", e)
	}
	if len(s) != len(p.Files) {
		return Rejectedf("composition: snapshot has %d files, policy lists %d%s", len(s), len(p.Files), snapshotPolicyDiff(s, p))
	}
	for _, f := range p.Files {
		b, ok := s[f.Path]
		if !ok {
			return Rejectedf("composition: policy path missing from snapshot: %s", f.Path)
		}
		if len(b) == 0 {
			return Rejectedf("composition: empty file: %s", f.Path)
		}
		limit := 4 << 20
		if strings.HasSuffix(f.Path, ".pdf") && f.ProtectedHash != "" {
			limit = 32 << 20
		}
		if len(b) > limit {
			return Rejectedf("composition: %s is %d bytes, limit %d", f.Path, len(b), limit)
		}
		if f.ProtectedHash != "" && Hash(b) != f.ProtectedHash {
			return Rejectedf("composition: protected_hash mismatch: %s", f.Path)
		}
		for _, part := range strings.Split(f.Path, "/") {
			if part == "internal" || part == "aidlc-docs" || part == ".git" || part == ".env" {
				return Rejectedf("composition: forbidden path component %q in %s", part, f.Path)
			}
		}
	}
	var manifest struct {
		Maturity string   `json:"maturity"`
		Paths    []string `json:"paths"`
	}
	if e := json.Unmarshal(s["composition-manifest.json"], &manifest); e != nil {
		return Rejectedf("composition-manifest.json: %v", e)
	}
	if len(manifest.Paths) != len(s) {
		return Rejectedf("composition-manifest.json lists %d paths, snapshot has %d", len(manifest.Paths), len(s))
	}
	seen := map[string]bool{}
	for _, path := range manifest.Paths {
		if _, ok := s[path]; !ok {
			return Rejectedf("composition-manifest.json path missing from snapshot: %s", path)
		}
		if seen[path] {
			return Rejectedf("composition-manifest.json path duplicated: %s", path)
		}
		seen[path] = true
	}
	return nil
}

// snapshotPolicyDiff names the first few paths present on only one side so a
// count mismatch is actionable without dumping either inventory.
func snapshotPolicyDiff(s Snapshot, p Policy) string {
	inPolicy := map[string]bool{}
	for _, f := range p.Files {
		inPolicy[f.Path] = true
	}
	var onlySnapshot, onlyPolicy []string
	for path := range s {
		if !inPolicy[path] {
			onlySnapshot = append(onlySnapshot, path)
		}
	}
	for _, f := range p.Files {
		if _, ok := s[f.Path]; !ok {
			onlyPolicy = append(onlyPolicy, f.Path)
		}
	}
	sort.Strings(onlySnapshot)
	sort.Strings(onlyPolicy)
	const show = 5
	out := ""
	if len(onlySnapshot) > 0 {
		out += fmt.Sprintf("; not in policy: %s", strings.Join(onlySnapshot[:min(show, len(onlySnapshot))], ", "))
		if len(onlySnapshot) > show {
			out += fmt.Sprintf(" (+%d)", len(onlySnapshot)-show)
		}
	}
	if len(onlyPolicy) > 0 {
		out += fmt.Sprintf("; not in snapshot: %s", strings.Join(onlyPolicy[:min(show, len(onlyPolicy))], ", "))
		if len(onlyPolicy) > show {
			out += fmt.Sprintf(" (+%d)", len(onlyPolicy)-show)
		}
	}
	return out
}

// ExportPath retains the predecessor's smaller path/depth and recursive
// exclusion floor in addition to process parser bounds.
func ExportPath(s string) bool {
	if !SafePath(s) || len(s) > 256 || len(strings.Split(s, "/")) > 16 {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		switch part {
		case ".DS_Store", ".git", ".gitlab", "aidlc-docs", "review", "audit-method":
			return false
		}
	}
	for _, part := range []string{"examples/foundation-", "docs/foundation/", "internal/", "ai-ops-control-plane.md", "operational-scenarios.md", "owasp-ast-top10-gap-analysis.md", "ai-agent-output-contract.md", "third-party-agent-prompt.AGENTS.md", "Agent-Ops_White_Paper_EN_v0.3.2"} {
		if strings.Contains(s, part) {
			return false
		}
	}
	return true
}
