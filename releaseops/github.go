package releaseops

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
)

const api = "https://api.github.com/repos/open-agent-ops/spec"
const uploads = "https://uploads.github.com/repos/open-agent-ops/spec"

type githubClient struct {
	http  *http.Client
	token string
	now   func() time.Time
	wait  func(context.Context) error
}

func newGitHub(token string) *githubClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
	transport.TLSHandshakeTimeout = connectTimeout
	transport.Proxy = nil
	transport.DialTLSContext = (&tls.Dialer{NetDialer: &net.Dialer{Timeout: connectTimeout}, Config: &tls.Config{MinVersion: tls.VersionTLS12}}).DialContext
	return &githubClient{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return ErrAuthority }}, token: token, now: time.Now, wait: backoff}
}
func (g *githubClient) request(ctx context.Context, method, endpoint string, body []byte) ([]byte, int, error) {
	u, e := url.Parse(endpoint)
	if e != nil || u.Scheme != "https" || (u.Host != "api.github.com" && u.Host != "uploads.github.com") || (u.Path != "/repos/open-agent-ops/spec" && !strings.HasPrefix(u.Path, "/repos/open-agent-ops/spec/")) || u.User != nil || u.Fragment != "" || (method != http.MethodGet && method != http.MethodPost) {
		return nil, 0, ErrAuthority
	}
	duration := metadataTimeout
	if u.Host == "uploads.github.com" {
		duration = transferTimeout
	}
	attempts := 1
	if method == http.MethodGet {
		attempts = maxAttempts
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		if !room(ctx, g.now(), duration) {
			return nil, 0, ErrUnknown
		}
		raw, status, err := g.requestOnce(ctx, method, endpoint, body, duration)
		if e := recordAttempt(ctx, "http_"+method, endpoint, attempt, err); e != nil {
			return nil, status, e
		}
		if err == nil {
			return raw, status, nil
		}
		if attempt == attempts {
			return nil, status, err
		}
		if _, ok := retryDelay(err, attempt); !ok {
			return nil, status, err
		}
		if e = g.waitRetry(ctx, err, attempt, duration); e != nil {
			return nil, status, e
		}
	}
	return nil, 0, ErrUnknown
}
func (g *githubClient) waitRetry(ctx context.Context, err error, attempt int, duration time.Duration) error {
	return waitRetry(ctx, g.now, g.wait, err, attempt, duration)
}
func (g *githubClient) requestOnce(ctx context.Context, method, endpoint string, body []byte, duration time.Duration) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if e != nil {
		return nil, 0, ErrAuthority
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("Content-Type", "application/json")
	if req.URL.Host == "uploads.github.com" {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, e := g.http.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return nil, 0, temporary("timeout")
		}
		var n net.Error
		if errors.As(e, &n) && (n.Timeout() || n.Temporary()) {
			return nil, 0, temporary("network")
		}
		return nil, 0, ErrAuthority
	}
	defer resp.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if len(raw) > 8<<20 {
		return nil, resp.StatusCode, ErrAuthority
	}
	if e != nil {
		if ctx.Err() != nil {
			return nil, resp.StatusCode, temporary("timeout")
		}
		return nil, resp.StatusCode, ErrUnknown
	}
	status := resp.StatusCode
	if status == 404 {
		return nil, status, nil
	}
	rateLimited := status == 429 || (status == 403 && g.token != "" && resp.Header.Get("X-RateLimit-Remaining") == "0" && resp.Header.Get("X-RateLimit-Reset") != "")
	if status == 408 || status >= 500 && status <= 599 || rateLimited {
		delay, e := retryAfter(resp.Header.Get("Retry-After"), g.now())
		if e != nil {
			return nil, status, e
		}
		if status == 403 {
			reset, e := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
			if e != nil || reset <= g.now().Unix() {
				return nil, status, ErrAuthority
			}
			until := time.Unix(reset, 0).Sub(g.now())
			if until > delay {
				delay = until
			}
		}
		return nil, status, &transportError{category: "http_transient", after: delay}
	}
	if status < 200 || status >= 300 {
		return nil, status, ErrAuthority
	}
	return raw, status, nil
}

// A404 only proves absence when the exact repository remains independently
// readable. Public read access or authenticated pull permission must be shown.
func (g *githubClient) repositoryReadable(ctx context.Context) bool {
	raw, status, e := g.request(ctx, http.MethodGet, api, nil)
	if e != nil || status != 200 {
		return false
	}
	if _, e = repocheck.JSON(raw); e != nil {
		return false
	}
	var repo struct {
		ID          int64
		FullName    string `json:"full_name"`
		Private     bool
		Permissions struct{ Pull bool }
		Owner       struct{ ID int64 }
	}
	if json.Unmarshal(raw, &repo) != nil {
		return false
	}
	return repo.ID == 1345286136 && repo.Owner.ID == 320649715 && repo.FullName == "open-agent-ops/spec" && (!repo.Private || g.token != "" && repo.Permissions.Pull)
}
func (g *githubClient) get(ctx context.Context, path string, out any) (bool, error) {
	raw, status, e := g.request(ctx, http.MethodGet, api+path, nil)
	if e != nil {
		return false, e
	}
	if status == 404 {
		if !g.repositoryReadable(ctx) {
			return false, ErrAuthority
		}
		return false, nil
	}
	if _, e = repocheck.JSON(raw); e != nil {
		return false, ErrUnknown
	}
	if json.Unmarshal(raw, out) != nil {
		return false, ErrUnknown
	}
	return true, nil
}
func (g *githubClient) post(ctx context.Context, path string, v any) error {
	raw, e := json.Marshal(v)
	if e != nil {
		return ErrAuthority
	}
	_, status, e := g.request(ctx, http.MethodPost, api+path, raw)
	if e != nil {
		return e
	}
	if status != 201 {
		return ErrUnknown
	}
	return nil
}
func (g *githubClient) ReadTag(ctx context.Context, v string) (Observation, error) {
	if !repocheck.Version(v) {
		return Observation{}, ErrAuthority
	}
	var ref struct {
		Ref    string                     `json:"ref"`
		Object struct{ Type, SHA string } `json:"object"`
	}
	ok, e := g.get(ctx, "/git/ref/tags/"+v, &ref)
	if e != nil || !ok {
		return Observation{}, e
	}
	if ref.Ref != "refs/tags/"+v || ref.Object.Type != "commit" || !repocheck.Commit(ref.Object.SHA) {
		return Observation{}, ErrConflict
	}
	return Observation{true, ref.Object.SHA, ""}, nil
}
func (g *githubClient) CreateTag(ctx context.Context, v, sha string) error {
	if !repocheck.Version(v) || !repocheck.Commit(sha) {
		return ErrAuthority
	}
	return g.post(ctx, "/git/refs", map[string]string{"ref": "refs/tags/" + v, "sha": sha})
}

type releaseRecord struct {
	ID         int64  `json:"id"`
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
		State  string `json:"state"`
	} `json:"assets"`
}

func (g *githubClient) release(ctx context.Context, v string) (releaseRecord, bool, error) {
	var r releaseRecord
	if !repocheck.Version(v) {
		return r, false, ErrAuthority
	}
	ok, e := g.get(ctx, "/releases/tags/"+v, &r)
	if e != nil || !ok {
		return r, ok, e
	}
	if r.ID <= 0 || r.Tag != v || r.Draft || r.Prerelease || len(r.Assets) > 16 {
		return r, false, ErrConflict
	}
	return r, true, nil
}
func (g *githubClient) ReadRelease(ctx context.Context, v string) (Observation, error) {
	r, ok, e := g.release(ctx, v)
	return Observation{ok, r.Tag, strconv.FormatInt(r.ID, 10)}, e
}
func (g *githubClient) CreateRelease(ctx context.Context, v, sha string) error {
	if !repocheck.Version(v) || !repocheck.Commit(sha) {
		return ErrAuthority
	}
	return g.post(ctx, "/releases", map[string]any{"tag_name": v, "target_commitish": sha, "name": v, "draft": false, "prerelease": false, "generate_release_notes": false})
}
func (g *githubClient) ReadAsset(ctx context.Context, v, n string) (Observation, error) {
	r, ok, e := g.release(ctx, v)
	if e != nil || !ok {
		return Observation{}, e
	}
	var out Observation
	for _, a := range r.Assets {
		if a.Name != n {
			continue
		}
		if out.Exists || a.ID <= 0 || a.State != "uploaded" || !strings.HasPrefix(a.Digest, "sha256:") || !repocheck.Digest(strings.TrimPrefix(a.Digest, "sha256:")) || a.Size <= 0 || a.Size > 64<<20 {
			return Observation{}, ErrConflict
		}
		out = Observation{true, strings.TrimPrefix(a.Digest, "sha256:"), strconv.FormatInt(a.ID, 10)}
	}
	return out, nil
}
func (g *githubClient) CreateAsset(ctx context.Context, v, n string, b []byte) error {
	if !repocheck.SafePath(n) || len(b) == 0 || len(b) > 64<<20 {
		return ErrAuthority
	}
	r, ok, e := g.release(ctx, v)
	if e != nil {
		return e
	}
	if !ok {
		return ErrUnknown
	}
	_, status, e := g.request(ctx, http.MethodPost, uploads+"/releases/"+strconv.FormatInt(r.ID, 10)+"/assets?name="+url.QueryEscape(n), b)
	if e != nil {
		return e
	}
	if status != 201 {
		return ErrUnknown
	}
	return nil
}
func (g *githubClient) mainAncestor(ctx context.Context, sha string) error {
	if !repocheck.Commit(sha) {
		return ErrAuthority
	}
	var comparison struct {
		Status    string `json:"status"`
		MergeBase struct {
			SHA string `json:"sha"`
		} `json:"merge_base_commit"`
	}
	ok, e := g.get(ctx, "/compare/"+sha+"...main", &comparison)
	if e != nil || !ok {
		return ErrUnknown
	}
	if (comparison.Status != "ahead" && comparison.Status != "identical") || comparison.MergeBase.SHA != sha {
		return ErrAuthority
	}
	return nil
}
func (g *githubClient) grant(ctx context.Context, p repocheck.Proposal, owners []string, initiator string, now time.Time) (Grant, error) {
	if repocheck.ValidateProposal(p, now) != nil || !opaqueID.MatchString(p.Run) || !opaqueID.MatchString(initiator) {
		return Grant{}, ErrAuthority
	}
	var run struct {
		ID      int64     `json:"id"`
		SHA     string    `json:"head_sha"`
		Branch  string    `json:"head_branch"`
		Event   string    `json:"event"`
		Attempt int       `json:"run_attempt"`
		Started time.Time `json:"run_started_at"`
		Actor   struct {
			ID int64 `json:"id"`
		} `json:"actor"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	ok, e := g.get(ctx, "/actions/runs/"+p.Run, &run)
	if e != nil || !ok {
		return Grant{}, ErrUnknown
	}
	if strconv.FormatInt(run.ID, 10) != p.Run || run.SHA != p.Candidate || run.Branch != "main" || run.Event != "workflow_dispatch" || run.Attempt != 1 || strconv.FormatInt(run.Actor.ID, 10) != initiator || run.Repository.FullName != "open-agent-ops/spec" {
		return Grant{}, ErrAuthority
	}
	var history []struct {
		State string `json:"state"`
		User  struct {
			ID int64 `json:"id"`
		} `json:"user"`
		Environments []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"environments"`
	}
	ok, e = g.get(ctx, "/actions/runs/"+p.Run+"/approvals", &history)
	if e != nil || !ok {
		return Grant{}, ErrUnknown
	}
	count := 0
	reviewer := ""
	for _, h := range history {
		for _, env := range h.Environments {
			if env.Name == "public-release" {
				count++
				if env.ID <= 0 || h.State != "approved" || h.User.ID <= 0 {
					return Grant{}, ErrAuthority
				}
				reviewer = strconv.FormatInt(h.User.ID, 10)
			}
		}
	}
	if count != 1 {
		return Grant{}, ErrAuthority
	}
	// Bounded single-page enumeration: this fixed workflow has three jobs. A
	// response with more than 100 jobs, pagination or incomplete facts is denied.
	var jobs struct {
		Total int `json:"total_count"`
		Jobs  []struct {
			ID                       int64  `json:"id"`
			RunID                    int64  `json:"run_id"`
			SHA                      string `json:"head_sha"`
			Name, Status, Conclusion string
			Completed                time.Time `json:"completed_at"`
			Steps                    []struct {
				Number  int       `json:"number"`
				Status  string    `json:"status"`
				Started time.Time `json:"started_at"`
			} `json:"steps"`
		} `json:"jobs"`
	}
	ok, e = g.get(ctx, "/actions/runs/"+p.Run+"/attempts/1/jobs?per_page=100", &jobs)
	if e != nil || !ok {
		return Grant{}, ErrUnknown
	}
	if jobs.Total != len(jobs.Jobs) || jobs.Total > 100 {
		return Grant{}, ErrAuthority
	}
	buildCount, publishCount := 0, 0
	var completed, started time.Time
	for _, j := range jobs.Jobs {
		if strconv.FormatInt(j.RunID, 10) != p.Run || j.SHA != p.Candidate {
			return Grant{}, ErrAuthority
		}
		switch j.Name {
		case "AOM / release-build":
			buildCount++
			if j.Status != "completed" || j.Conclusion != "success" {
				return Grant{}, ErrAuthority
			}
			completed = j.Completed
		case "AOM / publish":
			publishCount++
			if j.Status != "in_progress" || len(j.Steps) == 0 {
				return Grant{}, ErrAuthority
			}
			first := j.Steps[0]
			if first.Number != 1 || (first.Status != "in_progress" && first.Status != "completed") {
				return Grant{}, ErrAuthority
			}
			started = first.Started
		}
	}
	if buildCount != 1 || publishCount != 1 {
		return Grant{}, ErrAuthority
	}
	digest, e := repocheck.ProposalDigest(p)
	if e != nil {
		return Grant{}, ErrAuthority
	}
	return approve(p, ApprovalFacts{Proposal: digest, Run: p.Run, Initiator: initiator, Reviewer: reviewer, Environment: "public-release", Attempt: 1, RunStart: run.Started, BuildCompleted: completed, PublishStarted: started, OwnerIDs: owners, Decision: "approved"}, now, false)
}

// jobContext binds the operation to the actual current job start, including
// runner setup time. Reserve ten seconds for a bounded final receipt.
func (g *githubClient) jobContext(ctx context.Context, p repocheck.Proposal, name string) (context.Context, context.CancelFunc, error) {
	var jobs struct {
		Total int `json:"total_count"`
		Jobs  []struct {
			RunID        int64  `json:"run_id"`
			SHA          string `json:"head_sha"`
			Name, Status string
			Started      time.Time `json:"started_at"`
		}
	}
	ok, e := g.get(ctx, "/actions/runs/"+p.Run+"/attempts/1/jobs?per_page=100", &jobs)
	if e != nil || !ok || jobs.Total != len(jobs.Jobs) || jobs.Total > 100 {
		return nil, nil, ErrAuthority
	}
	count := 0
	var started time.Time
	for _, j := range jobs.Jobs {
		if j.Name == name {
			count++
			if strconv.FormatInt(j.RunID, 10) != p.Run || j.SHA != p.Candidate || j.Status != "in_progress" {
				return nil, nil, ErrAuthority
			}
			started = j.Started
		}
	}
	now := g.now()
	if count != 1 || started.IsZero() || started.After(now) {
		return nil, nil, ErrAuthority
	}
	deadline := started.Add(30*time.Minute - 10*time.Second)
	if !now.Before(deadline) {
		return nil, nil, ErrUnknown
	}
	child, cancel := context.WithDeadline(ctx, deadline)
	return child, cancel, nil
}

// PublishApproved is the only production canonical entry. Credential and host
// observations are acquired here; callers cannot supply a synthetic Grant.
func PublishApproved(ctx context.Context, p repocheck.Proposal, assets map[string][]byte) (repocheck.Receipt, error) {
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("AOM_REF") != "refs/heads/main" || os.Getenv("AOM_SHA") != p.Candidate || os.Getenv("AOM_CANDIDATE") != p.Candidate || os.Getenv("AOM_VERSION") != p.Version || os.Getenv("AOM_RUN") != p.Run || os.Getenv("AOM_ATTEMPT") != "1" || os.Getenv("AOM_WORKFLOW") != p.Workflow {
		return repocheck.Receipt{}, ErrAuthority
	}
	token, e := credential("AOM_GITHUB_TOKEN")
	if e != nil {
		return repocheck.Receipt{}, e
	}
	owners, e := ownerIDs(os.Getenv("AOM_OWNER_IDS"))
	if e != nil {
		return repocheck.Receipt{}, e
	}
	g := newGitHub(token)
	ctx = withAttemptLog(ctx, p, os.Getenv("AOM_INITIATOR"))
	jobCtx, cancel, e := g.jobContext(ctx, p, "AOM / publish")
	if e != nil {
		return repocheck.Receipt{}, e
	}
	defer cancel()
	ctx = jobCtx
	if e = g.mainAncestor(ctx, p.Candidate); e != nil {
		return repocheck.Receipt{}, e
	}
	grant, e := g.grant(ctx, p, owners, os.Getenv("AOM_INITIATOR"), time.Now())
	if e != nil {
		return repocheck.Receipt{}, e
	}
	publisher := &Publisher{g, time.Now, backoff, false}
	out, e := publisher.Publish(ctx, p, grant, assets)
	if e != nil {
		return out, e
	}
	r, ok, e := g.release(ctx, p.Version)
	if e != nil || !ok || len(r.Assets) != len(p.Assets) {
		out.Canonical = "unknown"
		return out, ErrUnknown
	}
	for _, a := range p.Assets {
		matches := 0
		for _, remote := range r.Assets {
			if remote.Name == a.Name && remote.Digest == "sha256:"+a.SHA256 && remote.Size == a.Size {
				matches++
			}
		}
		if matches != 1 {
			out.Canonical = "conflict"
			return out, ErrConflict
		}
	}
	return out, nil
}
