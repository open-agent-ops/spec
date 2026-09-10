// Package releaseops owns the explicit publication capability. It is never
// imported by the offline checker or the specification/conformance core.
package releaseops

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
)

var ErrAuthority = errors.New("release_authority_rejected")
var ErrConflict = errors.New("publication_conflict")
var ErrUnknown = errors.New("publication_unknown")

type Observation struct {
	Exists   bool
	Identity string
	ID       string
}

// Provider methods encode only the admitted GitHub tag/release/asset operations.
// There is no arbitrary URL, method, command or payload dispatch interface.
type provider interface {
	ReadTag(context.Context, string) (Observation, error)
	CreateTag(context.Context, string, string) error
	ReadRelease(context.Context, string) (Observation, error)
	CreateRelease(context.Context, string, string) error
	ReadAsset(context.Context, string, string) (Observation, error)
	CreateAsset(context.Context, string, string, []byte) error
}
type Grant struct {
	proposal, run, actor string
	expires              time.Time
	fixture              bool
}

// ApprovalFacts are fixture input only. Production authority comes from the
// concrete GitHub client's authenticated run/environment review history.
type ApprovalFacts struct {
	Proposal, Run, Initiator, Reviewer       string
	Environment                              string
	Attempt                                  int
	RunStart, BuildCompleted, PublishStarted time.Time
	OwnerIDs                                 []string
	Decision                                 string
}

func approve(p repocheck.Proposal, f ApprovalFacts, now time.Time, fixture bool) (Grant, error) {
	digest, e := repocheck.ProposalDigest(p)
	if e != nil || repocheck.ValidateProposal(p, now) != nil {
		return Grant{}, ErrAuthority
	}
	if f.Proposal != digest || f.Run != p.Run || f.Attempt != 1 || f.Initiator == "" || f.Reviewer == "" || f.Initiator == f.Reviewer || f.Environment != "public-release" || f.Decision != "approved" || f.RunStart.IsZero() || f.RunStart.After(p.CheckedAt) || f.BuildCompleted.IsZero() || f.PublishStarted.IsZero() || f.BuildCompleted.Before(p.CheckedAt) || f.PublishStarted.Before(f.BuildCompleted) || f.PublishStarted.After(now) {
		return Grant{}, ErrAuthority
	}
	count := 0
	for _, id := range f.OwnerIDs {
		if id == f.Reviewer {
			count++
		}
	}
	if count != 1 {
		return Grant{}, ErrAuthority
	}
	return Grant{digest, p.Run, f.Initiator, p.ExpiresAt, fixture}, nil
}

type Publisher struct {
	provider provider
	now      func() time.Time
	wait     func(context.Context) error
	fixture  bool
}

func (p *Publisher) Publish(ctx context.Context, proposal repocheck.Proposal, g Grant, assets map[string][]byte) (repocheck.Receipt, error) {
	if p == nil || p.provider == nil || p.now == nil || p.wait == nil {
		return repocheck.Receipt{}, ErrAuthority
	}
	out := repocheck.Receipt{Schema: "aom04a.publication-receipt.v1", Repository: repocheck.Repository, Candidate: proposal.Candidate, Version: proposal.Version, Canonical: "not_started", Mirror: "not_started", Assets: append([]repocheck.Asset(nil), proposal.Assets...), ObservedIDs: []string{}, FixtureOnly: p.fixture}
	digest, e := repocheck.ProposalDigest(proposal)
	out.Proposal = digest
	if e != nil || repocheck.ValidateProposal(proposal, p.now()) != nil || g.proposal != digest || g.run != proposal.Run || g.fixture != p.fixture || !p.now().Before(g.expires) || len(assets) != len(proposal.Assets) {
		return out, ErrAuthority
	}
	for _, a := range proposal.Assets {
		b, ok := assets[a.Name]
		if !ok || int64(len(b)) != a.Size || repocheck.Hash(b) != a.SHA256 {
			return out, ErrAuthority
		}
	}
	ensure := func(read func() (Observation, error), write func() error, want string, duration time.Duration) error {
		observed, err := reconcile(ctx, p.now, p.wait, func() bool { return p.now().Before(g.expires) }, read, write, want, duration)
		if err == nil && observed.ID != "" {
			out.ObservedIDs = append(out.ObservedIDs, observed.ID)
		}
		return err
	}
	fail := func(e error) (repocheck.Receipt, error) {
		out.Canonical = "unknown"
		if errors.Is(e, ErrConflict) {
			out.Canonical = "conflict"
		}
		return out, e
	}
	if e = ensure(func() (Observation, error) { return p.provider.ReadTag(ctx, proposal.Version) }, func() error { return p.provider.CreateTag(ctx, proposal.Version, proposal.Candidate) }, proposal.Candidate, metadataTimeout); e != nil {
		return fail(e)
	}
	if e = ensure(func() (Observation, error) { return p.provider.ReadRelease(ctx, proposal.Version) }, func() error { return p.provider.CreateRelease(ctx, proposal.Version, proposal.Candidate) }, proposal.Version, metadataTimeout); e != nil {
		return fail(e)
	}
	for _, a := range proposal.Assets {
		a := a
		if e = ensure(func() (Observation, error) { return p.provider.ReadAsset(ctx, proposal.Version, a.Name) }, func() error { return p.provider.CreateAsset(ctx, proposal.Version, a.Name, assets[a.Name]) }, a.SHA256, transferTimeout); e != nil {
			return fail(e)
		}
	}
	sort.Strings(out.ObservedIDs)
	out.Canonical = "verified"
	out.Mirror = "incomplete"
	return out, nil
}
