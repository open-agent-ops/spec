package releaseops

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
)

type attemptLogKey struct{}
type attemptLog struct {
	writer                io.Writer
	actor, run, candidate string
	count                 int
}

func withAttemptLog(ctx context.Context, p repocheck.Proposal, actor string) context.Context {
	return context.WithValue(ctx, attemptLogKey{}, &attemptLog{writer: os.Stderr, actor: actor, run: p.Run, candidate: p.Candidate})
}
func recordAttempt(ctx context.Context, action, target string, attempt int, err error) error {
	log, ok := ctx.Value(attemptLogKey{}).(*attemptLog)
	if !ok {
		return nil
	}
	if log.count >= 1000 || len(target) > 4096 || !opaqueID.MatchString(log.actor) || !opaqueID.MatchString(log.run) || !repocheck.Commit(log.candidate) {
		return ErrAuthority
	}
	result := "observed"
	if err != nil {
		result = "unknown"
		var tr *transportError
		if errors.As(err, &tr) {
			result = tr.category
		} else if errors.Is(err, ErrAuthority) {
			result = "denied"
		} else if errors.Is(err, ErrConflict) {
			result = "conflict"
		}
	}
	row := struct {
		Timestamp, Actor, Run, Candidate, Action, Target, Result string
		Attempt                                                  int
	}{time.Now().UTC().Format(time.RFC3339Nano), log.actor, log.run, log.candidate, action, target, result, attempt}
	raw, e := json.Marshal(row)
	if e != nil {
		return ErrUnknown
	}
	log.count++
	if _, e = log.writer.Write(append(raw, '\n')); e != nil {
		return ErrUnknown
	}
	return nil
}
