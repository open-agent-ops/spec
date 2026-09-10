package repocheck_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
	"pgregory.net/rapid"
)

func submissionGen() *rapid.Generator[repocheck.Submission] {
	return rapid.Custom(func(t *rapid.T) repocheck.Submission {
		s := submission()
		s.Kind = rapid.SampledFrom([]string{"fix", "bug", "proposal"}).Draw(t, "kind")
		s.Head = rapid.StringMatching(`[0-9a-f]{40}`).Draw(t, "head")
		s.Description = rapid.StringMatching(`[a-zA-Z0-9 _-]{1,100}`).Draw(t, "description")
		s.Paths = rapid.SampledFrom([][]string{{"README.md"}, {"GOVERNANCE.md"}, {"README.md", "GOVERNANCE.md"}}).Draw(t, "paths")
		return s
	})
}
func TestPropertyCanonicalRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := submissionGen().Draw(t, "submission")
		b, e := json.Marshal(s)
		if e != nil {
			t.Fatal(e)
		}
		canonical, e := repocheck.Canonical(b)
		if e != nil {
			t.Fatal(e)
		}
		var actual repocheck.Submission
		if repocheck.Decode(canonical, &actual) != nil || !reflect.DeepEqual(s, actual) {
			t.Fatal("round trip")
		}
		again, e := repocheck.Canonical(canonical)
		if e != nil || string(again) != string(canonical) {
			t.Fatal("unstable canonical JSON")
		}
	})
}
func TestPropertyAuthority(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := submissionGen().Draw(t, "submission")
		before, e := repocheck.Classify(s.Paths, policy())
		if e != nil {
			t.Fatal(e)
		}
		s.Description = rapid.StringMatching(`[a-zA-Z0-9 _-]{1,100}`).Draw(t, "untrusted prose")
		s.Kind = rapid.SampledFrom([]string{"fix", "bug", "proposal"}).Draw(t, "untrusted kind")
		after, e := repocheck.Classify(s.Paths, policy())
		if e != nil || before != after {
			t.Fatal("untrusted metadata changed classification")
		}
		s.Paths = append(s.Paths, "unrecognized/"+rapid.StringMatching(`[a-z]{1,12}`).Draw(t, "new path"))
		if _, e = repocheck.Classify(s.Paths, policy()); e == nil {
			t.Fatal("unknown path improved eligibility")
		}
	})
}
func TestPropertyEvidenceMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		b, records, streams := gateSet()
		if repocheck.EvidenceReady(records, policy(), b, streams) != nil {
			t.Fatal("valid gate set")
		}
		i := rapid.IntRange(0, len(records)-1).Draw(t, "removed gate")
		reduced := append(append([]repocheck.Gate{}, records[:i]...), records[i+1:]...)
		if repocheck.EvidenceReady(reduced, policy(), b, streams) == nil {
			t.Fatal("missing gate improved readiness")
		}
		records[i].Candidate = rapid.StringMatching(`[0-9a-f]{40}`).Filter(func(s string) bool { return s != b.Candidate }).Draw(t, "foreign candidate")
		if repocheck.EvidenceReady(records, policy(), b, streams) == nil {
			t.Fatal("foreign candidate improved readiness")
		}
	})
}
func TestPropertyHostileBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		payload := rapid.StringMatching(`[a-z]{1,40}`).Draw(t, "path payload")
		prefix := rapid.SampledFrom([]string{"../", "/", "nested/../", "nested/internal/", "nested/review/"}).Draw(t, "hostile prefix")
		if repocheck.ExportPath(prefix + payload) {
			t.Fatal("unsafe export path")
		}
		key := rapid.StringMatching(`[a-z]{1,40}`).Draw(t, "duplicate key")
		raw := `{"` + key + `":1,"` + key + `":2}`
		if _, e := repocheck.JSON([]byte(raw)); e == nil {
			t.Fatal("duplicate JSON")
		}
		depth := rapid.IntRange(33, 40).Draw(t, "depth")
		if _, e := repocheck.JSON([]byte(strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth))); e == nil {
			t.Fatal("depth bound")
		}
	})
}

// Generated event sequences exercise current review and exact evidence against
// a separate boolean/set oracle after every event, including head invalidation.
func TestPropertyStateSequence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := submission()
		p := policy()
		now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
		bindings := repocheck.ReviewBindings{Roles: map[string]string{"12": "owner"}, Now: now, HeadTime: now.Add(-time.Hour)}
		b, records, streams := gateSet()
		approved := false
		fresh := true
		canonical := false
		mirrored := false
		review := repocheck.Review{Schema: "aom04a.review-decision.v1", Reviewer: "12", Role: "owner", Candidate: s.Head, Decision: "dismiss", Timestamp: now.Add(-time.Minute), SourceReceipt: strings.Repeat("c", 64)}
		events := rapid.SliceOfN(rapid.SampledFrom([]string{"approve", "changes", "head", "gate_pass", "gate_fail", "publish", "timeout", "mirror", "mirror_conflict"}), 0, 100).Draw(t, "workflow events")
		for eventIndex, event := range events {
			switch event {
			case "approve":
				approved = true
				review.Candidate = s.Head
				review.Decision = "approve"
			case "changes":
				approved = false
				review.Decision = "request_changes"
			case "head":
				s.Head = fmt.Sprintf("%040x", eventIndex+1)
				b.Candidate = s.Head
				approved = false
				fresh = false
				canonical = false
				mirrored = false
			case "gate_pass":
				fresh = true
				for i := range records {
					records[i].Candidate = s.Head
					records[i].Result = "success"
				}
			case "gate_fail":
				fresh = false
				records[0].Result = "failure"
			case "publish":
				if approved && fresh {
					canonical = true
				}
			case "timeout":
				canonical = false
				mirrored = false
			case "mirror":
				if canonical {
					mirrored = true
				}
			case "mirror_conflict":
				mirrored = false
			}
			reviewOK := repocheck.ReviewReady(s, p, []repocheck.Review{review}, bindings) == nil
			evidenceOK := repocheck.EvidenceReady(records, p, b, streams) == nil
			if reviewOK != approved || evidenceOK != fresh {
				t.Fatalf("sequence drift after %s", event)
			}
			// The publication receipt oracle has separate canonical/mirror observations;
			// these observations cannot compensate for stale review or incomplete gates.
			expected := approved && fresh && canonical && mirrored
			observed := reviewOK && evidenceOK && canonical && mirrored
			if observed != expected {
				t.Fatal("terminal eligibility without prerequisites")
			}
		}
	})
}
