package releaseops

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
	"pgregory.net/rapid"
)

func TestPropertyReplay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p, f, now, data := candidate()
		p.Candidate = rapid.StringMatching(`[0-9a-f]{40}`).Draw(t, "candidate SHA")
		p.Workflow = p.Candidate
		content := []byte(rapid.StringMatching(`[a-zA-Z0-9_-]{1,1000}`).Draw(t, "synthetic asset"))
		data["source.tar.gz"] = content
		p.Assets[0].SHA256 = repocheck.Hash(content)
		p.Assets[0].Size = int64(len(content))
		f.Proposal, _ = repocheck.ProposalDigest(p)
		g, e := SimulateApproval(p, f, now)
		if e != nil {
			t.Fatal(e)
		}
		provider := &fakeProvider{state: map[string]Observation{}, timeoutAfter: rapid.Bool().Draw(t, "timeout after write")}
		publisher := NewSimulation(provider)
		publisher.now = func() time.Time { return now }
		publisher.wait = func(context.Context) error { return nil }
		first, e := publisher.Publish(context.Background(), p, g, data)
		if e != nil {
			t.Fatal(e)
		}
		writes := provider.writes
		count := rapid.IntRange(1, 10).Draw(t, "repeat count")
		for i := 0; i < count; i++ {
			again, e := publisher.Publish(context.Background(), p, g, data)
			if e != nil || provider.writes != writes || !reflect.DeepEqual(first, again) {
				t.Fatal("replay changed receipt or remote writes")
			}
		}
	})
}
