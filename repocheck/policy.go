package repocheck

import (
	"encoding/json"
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
	if p.Schema != "aom04a.policy.v1" || p.Repository != Repository || p.Owners != "open-agent-ops/owners" || p.Maintainers != "open-agent-ops/maintainers" || len(p.Files) == 0 || len(p.Files) > 256 {
		return ErrInput
	}
	seen := map[string]bool{}
	for _, f := range p.Files {
		if !ExportPath(f.Path) || seen[f.Path] || f.License == "" || (f.ProtectedHash != "" && !Digest(f.ProtectedHash)) {
			return ErrInput
		}
		seen[f.Path] = true
	}
	required := []string{"policy", "conformance", "docs", "supply-chain", "aggregate"}
	if len(p.Required) != len(required) || len(p.Full) != 2 || p.Full[0] != "heavy" || p.Full[1] != "reproducibility" {
		return ErrInput
	}
	for i, s := range required {
		if p.Required[i] != s {
			return ErrInput
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
	if p.Validate() != nil || !Commit(b.Candidate) || !Commit(b.Base) || !Commit(b.Workflow) || !Digest(b.Policy) || !Digest(b.Lock) || b.Run == "" || b.Attempt < 1 {
		return ErrInput
	}
	needed := append([]string(nil), p.Required...)
	if pre {
		needed = needed[:len(needed)-1]
	}
	if b.Full {
		needed = append(needed, p.Full...)
	}
	if len(records) != len(needed) {
		return ErrRejected
	}
	seen := map[string]bool{}
	for _, r := range records {
		if r.Schema != "aom04a.gate-record.v1" || r.Repository != Repository || r.Candidate != b.Candidate || r.Base != b.Base || r.Policy != b.Policy || r.Lock != b.Lock || r.Workflow != b.Workflow || r.Run != b.Run || r.Attempt != b.Attempt || r.Result != "success" || seen[r.Job] {
			return ErrRejected
		}
		out, ok := streams[r.Job+".stdout"]
		if !ok || Hash(out) != r.Stdout {
			return ErrRejected
		}
		errout, ok := streams[r.Job+".stderr"]
		if !ok || Hash(errout) != r.Stderr {
			return ErrRejected
		}
		seen[r.Job] = true
	}
	for _, n := range needed {
		if !seen[n] {
			return ErrRejected
		}
	}
	return nil
}

// Snapshot contains only candidate regular-file bytes supplied by a bounded
// caller. Symlinks are rejected at the file-system boundary before reading.
type Snapshot map[string][]byte

func Composition(s Snapshot, p Policy) error {
	if p.Validate() != nil || len(s) != len(p.Files) {
		return ErrRejected
	}
	for _, f := range p.Files {
		b, ok := s[f.Path]
		if !ok || len(b) == 0 {
			return ErrRejected
		}
		limit := 4 << 20
		if strings.HasSuffix(f.Path, ".pdf") && f.ProtectedHash != "" {
			limit = 32 << 20
		}
		if len(b) > limit || (f.ProtectedHash != "" && Hash(b) != f.ProtectedHash) {
			return ErrRejected
		}
		for _, part := range strings.Split(f.Path, "/") {
			if part == "internal" || part == "aidlc-docs" || part == ".git" || part == ".env" {
				return ErrRejected
			}
		}
	}
	var manifest struct {
		Maturity string   `json:"maturity"`
		Paths    []string `json:"paths"`
	}
	if json.Unmarshal(s["composition-manifest.json"], &manifest) != nil || len(manifest.Paths) != len(s) {
		return ErrRejected
	}
	seen := map[string]bool{}
	for _, path := range manifest.Paths {
		if _, ok := s[path]; !ok || seen[path] {
			return ErrRejected
		}
		seen[path] = true
	}
	return nil
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
