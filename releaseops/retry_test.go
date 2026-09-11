package releaseops

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestRetryStateMachine(t *testing.T) {
	for _, tc := range []struct {
		name                                    string
		failures                                int
		writeError                              error
		afterSuccess, unknown, conflict, expire bool
		want                                    int
		success                                 bool
	}{
		{name: "third", failures: 2, writeError: temporary("timeout"), want: 3, success: true},
		{name: "exhausted", failures: 3, writeError: temporary("timeout"), want: 3},
		{name: "timeout_committed", afterSuccess: true, writeError: temporary("timeout"), want: 1, success: true},
		{name: "unknown_readback", unknown: true, writeError: temporary("timeout"), want: 1},
		{name: "conflict_readback", conflict: true, writeError: temporary("timeout"), want: 1},
		{name: "denied", failures: 3, writeError: ErrAuthority, want: 1},
		{name: "unclassified", failures: 3, writeError: ErrUnknown, want: 1},
		{name: "expiry", failures: 3, writeError: temporary("timeout"), expire: true, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			present := false
			valid := true
			delays := []time.Duration{}
			read := func() (Observation, error) {
				if writes > 0 && tc.unknown {
					return Observation{}, ErrUnknown
				}
				if writes > 0 && tc.conflict {
					return Observation{Exists: true, Identity: "foreign"}, nil
				}
				return Observation{Exists: present, Identity: "wanted"}, nil
			}
			write := func() error {
				writes++
				if tc.afterSuccess {
					present = true
					return tc.writeError
				}
				if writes <= tc.failures {
					return tc.writeError
				}
				present = true
				return nil
			}
			wait := func(ctx context.Context) error {
				delays = append(delays, ctx.Value(delayKey{}).(time.Duration))
				if tc.expire {
					valid = false
				}
				return nil
			}
			_, err := reconcile(context.Background(), time.Now, wait, func() bool { return valid }, read, write, "wanted", metadataTimeout)
			if (err == nil) != tc.success || writes != tc.want {
				t.Fatalf("writes=%d err=%v", writes, err)
			}
			for i, delay := range delays {
				want := 5 * time.Second
				if i == 1 {
					want = 15 * time.Second
				}
				if delay != want {
					t.Fatal("backoff", delays)
				}
			}
		})
	}
}
func TestGeneratedRetryTraces(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failures := rapid.IntRange(0, 4).Draw(t, "transient failures")
		conflict := rapid.Bool().Draw(t, "existing conflict")
		committedTimeout := rapid.Bool().Draw(t, "commit before timeout")
		writes := 0
		state := ""
		if conflict {
			state = "other"
		}
		read := func() (Observation, error) { return Observation{Exists: state != "", Identity: state}, nil }
		write := func() error {
			writes++
			if committedTimeout {
				state = "target"
				return temporary("timeout")
			}
			if writes <= failures {
				return temporary("timeout")
			}
			state = "target"
			return nil
		}
		wait := func(context.Context) error { return nil }
		_, err := reconcile(context.Background(), time.Now, wait, func() bool { return true }, read, write, "target", metadataTimeout)
		expected := failures + 1
		if expected > 3 {
			expected = 3
		}
		if committedTimeout {
			expected = 1
		}
		if conflict {
			expected = 0
		}
		success := !conflict && (failures < 3 || committedTimeout)
		if writes != expected || (err == nil) != success {
			t.Fatalf("model mismatch writes=%d expected=%d err=%v", writes, expected, err)
		}
		if conflict && state != "other" {
			t.Fatal("replaced foreign identity")
		}
		if success {
			before := writes
			_, err = reconcile(context.Background(), time.Now, wait, func() bool { return true }, read, write, "target", metadataTimeout)
			if err != nil || writes != before {
				t.Fatal("replay wrote")
			}
		}
	})
}
func TestRetryAfterAndDeadline(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, bad := range []string{"-1", "18446744073709551615", "nonsense", now.Add(-time.Hour).Format(http.TimeFormat)} {
		if _, e := retryAfter(bad, now); e == nil {
			t.Fatal("accepted", bad)
		}
	}
	for _, value := range []string{"20", now.Add(20 * time.Second).Format(http.TimeFormat)} {
		d, e := retryAfter(value, now)
		if e != nil || d != 20*time.Second {
			t.Fatal(value, d, e)
		}
	}
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(64*time.Second))
	defer cancel()
	called := false
	e := waitRetry(ctx, func() time.Time { return now }, func(context.Context) error { called = true; return nil }, temporary("timeout"), 1, metadataTimeout)
	if e == nil || called {
		t.Fatal("wait crossed remaining budget")
	}
	e = waitRetry(context.Background(), time.Now, func(context.Context) error { called = true; return nil }, &transportError{after: time.Duration(1<<63 - 1)}, 1, metadataTimeout)
	if e == nil || called {
		t.Fatal("overflow admitted")
	}
}

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestHTTPAttemptsAndWriteOwnership(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		g := newGitHub("synthetic")
		calls := 0
		waits := 0
		g.wait = func(context.Context) error { waits++; return nil }
		g.http.Transport = responseTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			d, ok := r.Context().Deadline()
			if !ok || time.Until(d) > metadataTimeout {
				t.Error("metadata deadline")
			}
			status := 503
			if calls == 3 {
				status = 200
			}
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})
		_, _, e := g.request(context.Background(), method, api+"/releases", nil)
		if method == "GET" {
			if e != nil || calls != 3 || waits != 2 {
				t.Fatal(calls, waits, e)
			}
		} else {
			if e == nil || calls != 1 || waits != 0 {
				t.Fatal("nested write retry")
			}
		}
	}
}
func TestHTTPStatusClassification(t *testing.T) {
	for _, status := range []int{401, 403, 408, 429, 500, 502, 503, 302} {
		g := newGitHub("synthetic")
		calls := 0
		g.wait = func(context.Context) error { return nil }
		g.http.Transport = responseTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("SYNTHETIC_CANARY"))}, nil
		})
		_, _, e := g.request(context.Background(), "GET", api+"/releases", nil)
		want := 1
		if status == 408 || status == 429 || status >= 500 {
			want = 3
		}
		if e == nil || calls != want || strings.Contains(e.Error(), "CANARY") {
			t.Fatal(status, calls, e)
		}
	}
}
func TestGitDiagnosticClassification(t *testing.T) {
	for _, s := range []string{
		"authentication failed",
		"SSL certificate problem: unable to get local issuer certificate",
		"unknown exit",
		"repository not found",
		"The requested URL returned error: 403",
		"fatal: unable to access 'https://x/': HTTP 401",
		"could not read Username for 'https://x'",
	} {
		e := transientGitDiagnostic([]byte(s), false)
		if _, ok := retryDelay(e, 1); ok {
			t.Fatal("retry unsafe exit", s)
		}
		if s != "unknown exit" && !errors.Is(e, ErrAuthority) {
			t.Fatal("authority expected", s, e)
		}
	}
	for _, s := range []string{
		"Connection reset by peer",
		"HTTP 503",
		// curl wording for a dropped TLS session: contains "ssl" but is a
		// transport fault, not a certificate or authority failure.
		"error: RPC failed; curl 56 OpenSSL SSL_read: Connection reset by peer, errno 104",
		// digits that happen to spell a status code must not deny a retry
		"fatal: the remote end hung up unexpectedly after 1403 bytes: connection timed out",
	} {
		if _, ok := retryDelay(transientGitDiagnostic([]byte(s), false), 1); !ok {
			t.Fatal("transient denied", s)
		}
	}
	// A bare status code inside unrelated text is neither transient nor an
	// authority verdict; it stays unknown and is not retried.
	if e := transientGitDiagnostic([]byte("received 403 objects"), false); !errors.Is(e, ErrUnknown) {
		t.Fatal("bare digits classified", e)
	}
	if !errors.Is(transientGitDiagnostic(nil, true), ErrUnknown) {
		t.Fatal("timeout")
	}
}

type canceledBody struct {
	ctx    context.Context
	closed bool
}

func (b *canceledBody) Read([]byte) (int, error) { <-b.ctx.Done(); return 0, b.ctx.Err() }
func (b *canceledBody) Close() error             { b.closed = true; return nil }
func TestHTTPBodyCancellation(t *testing.T) {
	g := newGitHub("")
	var body *canceledBody
	g.http.Transport = responseTransport(func(r *http.Request) (*http.Response, error) {
		body = &canceledBody{ctx: r.Context()}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
	})
	_, _, e := g.requestOnce(context.Background(), "GET", api+"/releases", nil, 10*time.Millisecond)
	if e == nil || body == nil || !body.closed {
		t.Fatal("body not canceled/closed")
	}
}
func TestHTTPResponseBound(t *testing.T) {
	g := newGitHub("")
	g.http.Transport = responseTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("x", (8<<20)+1)))}, nil
	})
	if _, _, e := g.request(context.Background(), "GET", api+"/releases", nil); !errors.Is(e, ErrAuthority) {
		t.Fatal("oversize")
	}
}
func TestJobDeadline(t *testing.T) {
	p, _, _, _ := candidate()
	now := time.Now().UTC()
	g := newGitHub("")
	g.now = func() time.Time { return now }
	g.http.Transport = responseTransport(func(*http.Request) (*http.Response, error) {
		raw := `{"total_count":1,"jobs":[{"run_id":123,"head_sha":"` + p.Candidate + `","name":"AOM / publish","status":"in_progress","started_at":"` + now.Add(-20*time.Minute).Format(time.RFC3339Nano) + `"}]}`
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(raw))}, nil
	})
	ctx, cancel, e := g.jobContext(context.Background(), p, "AOM / publish")
	if e != nil {
		t.Fatal(e)
	}
	defer cancel()
	d, _ := ctx.Deadline()
	if !d.Equal(now.Add(10*time.Minute - 10*time.Second)) {
		t.Fatal("job budget reset", d)
	}
	if _, _, e = g.jobContext(context.Background(), p, "AOM / mirror"); e == nil {
		t.Fatal("wrong job")
	}
}
func TestGitCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	m := &gitMirror{dir: t.TempDir()}
	start := time.Now()
	_, e := m.gitOnce(ctx, false, "-c", "alias.aom05wait=!sleep 10", "aom05wait")
	if e == nil || time.Since(start) > 2*time.Second {
		t.Fatal("Git subprocess cancellation", e)
	}
}
func TestAttemptLog(t *testing.T) {
	p, _, _, _ := candidate()
	ctx := withAttemptLog(context.Background(), p, "11")
	log := ctx.Value(attemptLogKey{}).(*attemptLog)
	var output strings.Builder
	log.writer = &output
	if e := recordAttempt(ctx, "write_result", p.Candidate, 1, errors.New("SYNTHETIC_SECRET")); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(output.String(), "SYNTHETIC_SECRET") || !strings.Contains(output.String(), p.Candidate) {
		t.Fatal("unsafe/missing evidence")
	}
}
