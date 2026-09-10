package releaseops

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type routeTransport struct{ target *url.URL }

func (t routeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	u := *r.URL
	u.Scheme = t.target.Scheme
	u.Host = t.target.Host
	copy.URL = &u
	return http.DefaultTransport.RoundTrip(copy)
}
func testGitHub(t *testing.T, h http.Handler) *githubClient {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	u, e := url.Parse(server.URL)
	if e != nil {
		t.Fatal(e)
	}
	g := newGitHub("SYNTHETIC_CREDENTIAL_CANARY")
	g.http.Transport = routeTransport{u}
	return g
}
func TestRule08(t *testing.T) {
	p, _, now, _ := candidate()
	for _, mode := range []string{"valid_no_approval_timestamp", "self", "missing_approval", "ambiguous", "rejected", "attempt2", "foreign_sha", "unfinished_build", "missing_step_time", "duplicate_build", "reordered_times", "expired", "environment_times_are_not_approval"} {
		t.Run(mode, func(t *testing.T) {
			writes := 0
			g := testGitHub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					writes++
					t.Error("authority readback attempted mutation")
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/open-agent-ops/spec/actions/runs/123":
					sha := p.Candidate
					attempt := 1
					if mode == "foreign_sha" {
						sha = strings.Repeat("b", 40)
					}
					if mode == "attempt2" {
						attempt = 2
					}
					json.NewEncoder(w).Encode(map[string]any{"id": 123, "head_sha": sha, "head_branch": "main", "event": "workflow_dispatch", "run_attempt": attempt, "run_started_at": now.Add(-2 * time.Hour), "actor": map[string]any{"id": 11}, "repository": map[string]any{"full_name": "open-agent-ops/spec"}})
				case "/repos/open-agent-ops/spec/actions/runs/123/approvals":
					id := 12
					if mode == "self" {
						id = 11
					}
					state := "approved"
					if mode == "rejected" {
						state = "rejected"
					}
					h := map[string]any{"state": state, "comment": "", "user": map[string]any{"id": id}, "environments": []any{map[string]any{"id": 101, "name": "public-release", "created_at": "2000-01-01T00:00:00Z", "updated_at": "2000-01-01T00:00:00Z"}}}
					rows := []any{h}
					if mode == "missing_approval" {
						rows = nil
					}
					if mode == "ambiguous" {
						rows = append(rows, h)
					}
					json.NewEncoder(w).Encode(rows)
				case "/repos/open-agent-ops/spec/actions/runs/123/attempts/1/jobs":
					status := "completed"
					if mode == "unfinished_build" {
						status = "in_progress"
					}
					completed := now.Add(-30 * time.Minute)
					started := now.Add(-time.Minute)
					if mode == "missing_step_time" || mode == "environment_times_are_not_approval" {
						started = time.Time{}
					}
					if mode == "reordered_times" {
						completed = now
					}
					build := map[string]any{"id": 1, "run_id": 123, "head_sha": p.Candidate, "name": "AOM / release-build", "status": status, "conclusion": "success", "completed_at": completed}
					publish := map[string]any{"id": 2, "run_id": 123, "head_sha": p.Candidate, "name": "AOM / publish", "status": "in_progress", "steps": []any{map[string]any{"number": 1, "status": "completed", "started_at": started}}}
					jobs := []any{build, publish}
					if mode == "duplicate_build" {
						jobs = append(jobs, build)
					}
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(jobs), "jobs": jobs})
				default:
					t.Error("unexpected route", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			observed := now
			if mode == "expired" {
				observed = p.ExpiresAt
			}
			grant, e := g.grant(context.Background(), p, []string{"12"}, "11", observed)
			if mode == "valid_no_approval_timestamp" {
				if e != nil || grant.fixture {
					t.Fatal("valid provider-shaped proof rejected", e)
				}
			} else if e == nil {
				t.Fatal("invalid authority admitted")
			}
			if writes != 0 {
				t.Fatal("write side effect")
			}
			t.Logf("fixture-only scenario=%s accepted=%v mutation_count=%d", mode, e == nil, writes)
		})
	}
}
func TestFixedHTTPBoundary(t *testing.T) {
	calls := 0
	g := testGitHub(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Location", "https://example.invalid/stolen")
		w.WriteHeader(302)
	}))
	for _, endpoint := range []string{"https://example.invalid/repos/open-agent-ops/spec/x", api + "-foreign/x", "http://api.github.com/repos/open-agent-ops/spec/x"} {
		if _, _, e := g.request(context.Background(), "GET", endpoint, nil); e == nil {
			t.Fatal("foreign endpoint")
		}
	}
	if calls != 0 {
		t.Fatal("foreign route reached transport")
	}
	_, _, e := g.request(context.Background(), "GET", api+"/releases", nil)
	if e == nil || strings.Contains(e.Error(), g.token) || calls != 1 {
		t.Fatal("redirect/canary boundary")
	}
}

type fakeMirror struct {
	sha                                string
	present                            bool
	writes                             int
	afterTimeout, unknown, absentFirst bool
}

func (f *fakeMirror) Observe(context.Context, string) (string, bool, error) {
	if f.unknown {
		return "", false, ErrUnknown
	}
	return f.sha, f.present, nil
}
func (f *fakeMirror) Push(_ context.Context, sha, _ string) error {
	f.writes++
	if f.absentFirst {
		f.absentFirst = false
		return temporary("timeout")
	}
	f.sha = sha
	f.present = true
	if f.afterTimeout {
		return ErrUnknown
	}
	return nil
}
func TestRule10(t *testing.T) {
	p, _, _, _ := candidate()
	p.ExpiresAt = time.Now().Add(time.Hour)
	for _, mode := range []string{"matching", "new", "timeout_after_write", "absent_retry", "divergent", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			f := &fakeMirror{sha: p.Candidate, present: mode == "matching", afterTimeout: mode == "timeout_after_write", unknown: mode == "unknown", absentFirst: mode == "absent_retry"}
			if mode == "divergent" {
				f.present = true
				f.sha = strings.Repeat("b", 40)
			}
			waits := 0
			state, e := mirror(context.Background(), p, f, func(context.Context) error { waits++; return nil })
			switch mode {
			case "divergent":
				if state != "conflict" || !errors.Is(e, ErrConflict) || f.writes != 0 {
					t.Fatal(state, e)
				}
			case "unknown":
				if state != "unknown" || e == nil || f.writes != 0 {
					t.Fatal(state, e)
				}
			default:
				if state != "verified" || e != nil {
					t.Fatal(state, e)
				}
				expected := 1
				if mode == "matching" {
					expected = 0
				}
				if mode == "absent_retry" {
					expected = 2
					if waits != 1 {
						t.Fatal("retry budget")
					}
				}
				if f.writes != expected {
					t.Fatal("write count")
				}
			}
			t.Logf("fixture-only scenario=%s mirror=%s writes=%d waits=%d", mode, state, f.writes, waits)
		})
	}
}
