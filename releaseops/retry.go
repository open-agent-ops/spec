package releaseops

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const metadataTimeout = 60 * time.Second
const transferTimeout = 180 * time.Second
const connectTimeout = 30 * time.Second
const maxAttempts = 3

type transportError struct {
	category string
	after    time.Duration
}

func (e *transportError) Error() string { return "publication_transport_" + e.category }
func (e *transportError) Unwrap() error { return ErrUnknown }
func temporary(category string) error   { return &transportError{category: category} }
func retryDelay(err error, attempt int) (time.Duration, bool) {
	var e *transportError
	if !errors.As(err, &e) || attempt < 1 || attempt >= maxAttempts {
		return 0, false
	}
	delay := 5 * time.Second
	if attempt == 2 {
		delay = 15 * time.Second
	}
	if e.after > delay {
		delay = e.after
	}
	return delay, true
}
func retryAfter(value string, now time.Time) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	if n, e := strconv.ParseUint(value, 10, 64); e == nil {
		if n > uint64((1<<63-1)/int64(time.Second)) {
			return 0, ErrAuthority
		}
		return time.Duration(n) * time.Second, nil
	}
	date, e := http.ParseTime(value)
	if e != nil || date.Before(now) {
		return 0, ErrAuthority
	}
	return date.Sub(now), nil
}

type delayKey struct{}

func backoff(ctx context.Context) error {
	delay, ok := ctx.Value(delayKey{}).(time.Duration)
	if !ok || delay < 0 {
		return ErrUnknown
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ErrUnknown
	}
}
func room(ctx context.Context, now time.Time, required time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	deadline, ok := ctx.Deadline()
	return !ok || deadline.Sub(now) >= required
}
func waitRetry(ctx context.Context, now func() time.Time, wait func(context.Context) error, err error, attempt int, operation time.Duration) error {
	delay, ok := retryDelay(err, attempt)
	if !ok || delay > time.Duration(1<<63-1)-operation || !room(ctx, now(), delay+operation) {
		return ErrUnknown
	}
	return wait(context.WithValue(ctx, delayKey{}, delay))
}

// reconcile alone owns write retries. Adapters perform exactly one write; their
// read operations have a separate budget. Uncertain readback always halts writes.
func reconcile(ctx context.Context, now func() time.Time, wait func(context.Context) error, valid func() bool, read func() (Observation, error), write func() error, want string, duration time.Duration) (Observation, error) {
	observed, err := read()
	if err != nil {
		return Observation{}, err
	}
	match := func(o Observation) error {
		if o.Identity != want {
			return ErrConflict
		}
		return nil
	}
	if observed.Exists {
		return observed, match(observed)
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if !valid() {
			return observed, ErrAuthority
		}
		if !room(ctx, now(), duration) {
			return observed, ErrUnknown
		}
		if e := recordAttempt(ctx, "write_start", want, attempt, nil); e != nil {
			return observed, e
		}
		writeErr := write()
		logErr := recordAttempt(ctx, "write_result", want, attempt, writeErr)
		observed, err = read()
		if logErr != nil {
			return observed, logErr
		}
		result := err
		if result == nil && observed.Exists && observed.Identity != want {
			result = ErrConflict
		}
		action := "readback_absent"
		if observed.Exists {
			action = "readback_present"
		}
		if e := recordAttempt(ctx, action, want, attempt, result); e != nil {
			return observed, e
		}
		if err != nil {
			return observed, ErrUnknown
		}
		if observed.Exists {
			return observed, match(observed)
		}
		if writeErr == nil {
			return observed, ErrUnknown
		}
		if _, ok := retryDelay(writeErr, attempt); !ok {
			return observed, writeErr
		}
		if err = waitRetry(ctx, now, wait, writeErr, attempt, duration); err != nil {
			return observed, err
		}
	}
	return observed, ErrUnknown
}
func transientGitDiagnostic(b []byte, timedOut bool) error {
	if timedOut {
		return temporary("timeout")
	}
	// Raw diagnostics never leave this bounded classifier. Unknown exits are not
	// transient, including authentication, certificate and repository failures.
	//
	// Transport phrases are matched first: curl reports a dropped TLS session as
	// "OpenSSL SSL_read: Connection reset by peer", and a substring such as
	// "ssl" or "403" inside an otherwise transient line must not turn a retryable
	// network fault into a non-retryable authority verdict. Denied phrases are
	// anchored to the shapes git and curl actually emit rather than bare digits.
	s := strings.ToLower(string(b))
	for _, transient := range []string{"connection reset by peer", "connection timed out", "failed to connect", "temporary failure in name resolution", "http 408", "http 429", "http 500", "http 502", "http 503", "http 504"} {
		if strings.Contains(s, transient) {
			return temporary("git_transient")
		}
	}
	for _, deny := range []string{"authentication", "permission denied", "error: 403", "http 403", "error: 401", "http 401", "certificate", "not found", "could not read username", "redirect"} {
		if strings.Contains(s, deny) {
			return ErrAuthority
		}
	}
	return ErrUnknown
}
