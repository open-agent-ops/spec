package releaseops

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
)

type fakeProvider struct {
	state        map[string]Observation
	writes       int
	timeoutAfter bool
	unknownRead  bool
	absentFirst  bool
}

func (f *fakeProvider) read(key string) (Observation, error) {
	if f.unknownRead {
		return Observation{}, ErrUnknown
	}
	return f.state[key], nil
}
func (f *fakeProvider) write(key, id string) error {
	f.writes++
	if f.absentFirst {
		f.absentFirst = false
		return temporary("timeout")
	}
	f.state[key] = Observation{true, id, "12"}
	if f.timeoutAfter {
		return ErrUnknown
	}
	return nil
}
func (f *fakeProvider) ReadTag(_ context.Context, v string) (Observation, error) {
	return f.read("tag/" + v)
}
func (f *fakeProvider) CreateTag(_ context.Context, v, sha string) error {
	return f.write("tag/"+v, sha)
}
func (f *fakeProvider) ReadRelease(_ context.Context, v string) (Observation, error) {
	return f.read("release/" + v)
}
func (f *fakeProvider) CreateRelease(_ context.Context, v, _ string) error {
	return f.write("release/"+v, v)
}
func (f *fakeProvider) ReadAsset(_ context.Context, v, n string) (Observation, error) {
	return f.read(v + "/" + n)
}
func (f *fakeProvider) CreateAsset(_ context.Context, v, n string, b []byte) error {
	return f.write(v+"/"+n, repocheck.Hash(b))
}
func candidate() (repocheck.Proposal, ApprovalFacts, time.Time, map[string][]byte) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	data := []byte("synthetic-source")
	p := repocheck.Proposal{Schema: "aom04a.release-proposal.v1", Repository: repocheck.Repository, Candidate: strings.Repeat("a", 40), Version: "v0.4.0", Tree: strings.Repeat("b", 40), Inventory: strings.Repeat("c", 64), Assets: []repocheck.Asset{{Name: "source.tar.gz", SHA256: repocheck.Hash(data), Size: int64(len(data))}}, SBOM: strings.Repeat("d", 64), Workflow: strings.Repeat("a", 40), Run: "123", Attempt: 1, Lock: strings.Repeat("e", 64), Policy: strings.Repeat("f", 64), CheckedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}
	digest, _ := repocheck.ProposalDigest(p)
	f := ApprovalFacts{Proposal: digest, Run: p.Run, Initiator: "11", Reviewer: "12", Environment: "public-release", Attempt: 1, RunStart: now.Add(-2 * time.Hour), BuildCompleted: now.Add(-30 * time.Minute), PublishStarted: now.Add(-time.Minute), OwnerIDs: []string{"12"}, Decision: "approved"}
	return p, f, now, map[string][]byte{"source.tar.gz": data}
}
func TestRule07(t *testing.T) {
	p, f, now, _ := candidate()
	if _, e := SimulateApproval(p, f, now); e != nil {
		t.Fatal("synthetic positive")
	}
	for _, change := range []func(*ApprovalFacts){func(f *ApprovalFacts) { f.Reviewer = f.Initiator }, func(f *ApprovalFacts) { f.Attempt = 2 }, func(f *ApprovalFacts) { f.Run = "999" }, func(f *ApprovalFacts) { f.Proposal = strings.Repeat("0", 64) }, func(f *ApprovalFacts) { f.PublishStarted = time.Time{} }, func(f *ApprovalFacts) { f.OwnerIDs = []string{} }} {
		bad := f
		change(&bad)
		if _, e := SimulateApproval(p, bad, now); e == nil {
			t.Fatal("invalid authority")
		}
	}
	t.Log("fixture-only cases=self_run_attempt_proposal_missing_time_nonowner_denied; production_host_authority=not_implemented")
}
func TestRule09(t *testing.T) {
	p, f, now, data := candidate()
	grant, e := SimulateApproval(p, f, now)
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"normal", "timeout_after_write", "confirmed_absence_retry", "conflict", "unknown_read"} {
		t.Run(mode, func(t *testing.T) {
			provider := &fakeProvider{state: map[string]Observation{}, timeoutAfter: mode == "timeout_after_write", absentFirst: mode == "confirmed_absence_retry", unknownRead: mode == "unknown_read"}
			if mode == "conflict" {
				provider.state["tag/"+p.Version] = Observation{true, strings.Repeat("b", 40), "9"}
			}
			publisher := NewSimulation(provider)
			publisher.now = func() time.Time { return now }
			waits := 0
			publisher.wait = func(context.Context) error { waits++; return nil }
			out, e := publisher.Publish(context.Background(), p, grant, data)
			if mode == "conflict" {
				if !errors.Is(e, ErrConflict) || provider.writes != 0 {
					t.Fatal("conflict mutated")
				}
				return
			}
			if mode == "unknown_read" {
				if !errors.Is(e, ErrUnknown) || provider.writes != 0 {
					t.Fatal("unknown mutated")
				}
				return
			}
			if e != nil || out.Canonical != "verified" || out.Mirror != "incomplete" || !out.FixtureOnly {
				t.Fatal("receipt", e, out)
			}
			expected := 3
			if mode == "confirmed_absence_retry" {
				expected = 4
				if waits != 1 {
					t.Fatal("retry count")
				}
			}
			if provider.writes != expected {
				t.Fatal("write count", provider.writes)
			}
			replay, e := publisher.Publish(context.Background(), p, grant, data)
			if e != nil || provider.writes != expected || !reflect.DeepEqual(out, replay) {
				t.Fatal("replay drift")
			}
			t.Logf("fixture-only mode=%s writes=%d retry_waits=%d canonical=verified mirror=incomplete replay_writes=0", mode, provider.writes, waits)
		})
	}
}

// SimulateApproval cannot produce a production grant. Controlled fixture
// providers are the only permitted consumer of this explicit simulation path.
func SimulateApproval(p repocheck.Proposal, f ApprovalFacts, now time.Time) (Grant, error) {
	return approve(p, f, now, true)
}

func NewSimulation(p provider) *Publisher { return &Publisher{p, time.Now, backoff, true} }
