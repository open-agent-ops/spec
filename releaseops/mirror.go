package releaseops

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
)

var mirrorPath = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,99}/[A-Za-z0-9][A-Za-z0-9_.-]{0,99}$`)

type mirrorProvider interface {
	Observe(context.Context, string) (string, bool, error)
	Push(context.Context, string, string) error
}

func mirror(ctx context.Context, p repocheck.Proposal, provider mirrorProvider, wait func(context.Context) error) (string, error) {
	observed, e := reconcile(ctx, time.Now, wait, func() bool { return time.Now().Before(p.ExpiresAt) },
		func() (Observation, error) {
			sha, exists, err := provider.Observe(ctx, p.Version)
			return Observation{Exists: exists, Identity: sha}, err
		},
		func() error { return provider.Push(ctx, p.Candidate, p.Version) }, p.Candidate, transferTimeout)
	if e != nil {
		if errors.Is(e, ErrConflict) {
			return "conflict", e
		}
		return "unknown", e
	}
	if !observed.Exists {
		return "unknown", ErrUnknown
	}
	return "verified", nil
}

type gitMirror struct{ dir, remote, token, askpass string }

func (m *gitMirror) gitOnce(ctx context.Context, credential bool, args ...string) ([]byte, error) {
	duration := metadataTimeout
	if len(args) > 0 && (args[0] == "fetch" || args[0] == "push") {
		duration = transferTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/git", args...)
	if configureGitCancellation(cmd) != nil {
		return nil, ErrAuthority
	}
	cmd.Dir = m.dir
	cmd.Env = []string{"PATH=/usr/bin:/bin", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=https", "GIT_CONFIG_COUNT=2", "GIT_CONFIG_KEY_0=credential.helper", "GIT_CONFIG_VALUE_0=", "GIT_CONFIG_KEY_1=http.followRedirects", "GIT_CONFIG_VALUE_1=false"}
	if credential {
		cmd.Env = append(cmd.Env, "GIT_ASKPASS="+m.askpass, "AOM_MIRROR_TOKEN="+m.token)
	}
	out := boundedOutput{remaining: 8 << 20}
	cmd.Stdout = &out
	diagnostic := boundedOutput{remaining: 64 << 10}
	cmd.Stderr = &diagnostic
	if cmd.Run() != nil {
		return nil, transientGitDiagnostic(diagnostic.Bytes(), ctx.Err() != nil)
	}
	if out.Len() > 8<<20 {
		return nil, ErrUnknown
	}
	return out.Bytes(), nil
}

func (m *gitMirror) git(ctx context.Context, credential bool, args ...string) ([]byte, error) {
	attempts := 1
	duration := metadataTimeout
	if len(args) > 0 && (args[0] == "ls-remote" || args[0] == "fetch") {
		attempts = maxAttempts
	}
	if len(args) > 0 && (args[0] == "fetch" || args[0] == "push") {
		duration = transferTimeout
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		if !room(ctx, time.Now(), duration) {
			return nil, ErrUnknown
		}
		out, e := m.gitOnce(ctx, credential, args...)
		action := "git"
		if len(args) > 0 {
			action = "git_" + args[0]
		}
		if logErr := recordAttempt(ctx, action, m.remote, attempt, e); logErr != nil {
			return nil, logErr
		}
		if e == nil {
			return out, nil
		}
		if attempt == attempts {
			return nil, e
		}
		if _, ok := retryDelay(e, attempt); !ok {
			return nil, e
		}
		if e = waitRetry(ctx, time.Now, backoff, e, attempt, duration); e != nil {
			return nil, e
		}
	}
	return nil, ErrUnknown
}

// Retain at most the fixed provider response budget while the command runs.
type boundedOutput struct {
	bytes.Buffer
	remaining int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, ErrUnknown
	}
	n, e := b.Buffer.Write(p)
	b.remaining -= n
	return n, e
}

type discardWriter struct{}

func (*discardWriter) Write(p []byte) (int, error) { return len(p), nil }

type mirrorRefs struct{ tag, main string }

// Only the two requested direct refs are admissible. Partial state is never a
// completed mirror, even when the immutable tag already exists.
func parseMirrorRefs(data []byte, version string) (mirrorRefs, error) {
	var refs mirrorRefs
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !repocheck.Commit(fields[0]) || seen[fields[1]] {
			return refs, ErrUnknown
		}
		seen[fields[1]] = true
		switch fields[1] {
		case "refs/tags/" + version:
			refs.tag = fields[0]
		case "refs/heads/main":
			refs.main = fields[0]
		default:
			return refs, ErrUnknown
		}
	}
	return refs, nil
}
func (m *gitMirror) refs(ctx context.Context, version string) (mirrorRefs, error) {
	data, err := m.git(ctx, true, "ls-remote", "--refs", m.remote, "refs/tags/"+version, "refs/heads/main")
	if err != nil {
		return mirrorRefs{}, err
	}
	return parseMirrorRefs(data, version)
}
func (m *gitMirror) Observe(ctx context.Context, version string) (string, bool, error) {
	if !repocheck.Version(version) {
		return "", false, ErrAuthority
	}
	refs, err := m.refs(ctx, version)
	if err != nil {
		return "", false, err
	}
	return refs.tag, refs.tag != "" && refs.tag == refs.main, nil
}

// The candidate's complete canonical history is fetched before this check.
// No mirror history is imported to justify an update. The server must support
// atomic push; there is deliberately no sequential or force-push fallback.
func mirrorPushArgs(remote, candidate, version string, refs mirrorRefs, ancestor func(string, string) error) ([]string, error) {
	if !repocheck.Commit(candidate) || !repocheck.Version(version) {
		return nil, ErrAuthority
	}
	if refs.tag != "" && refs.tag != candidate {
		return nil, ErrConflict
	}
	if refs.main != "" && refs.main != candidate {
		if !repocheck.Commit(refs.main) || ancestor(refs.main, candidate) != nil {
			return nil, ErrConflict
		}
	}
	return []string{"push", "--atomic", "--porcelain", remote, candidate + ":refs/tags/" + version, candidate + ":refs/heads/main"}, nil
}
func (m *gitMirror) Push(ctx context.Context, candidate, version string) error {
	if !repocheck.Commit(candidate) || !repocheck.Version(version) {
		return ErrAuthority
	}
	refs, err := m.refs(ctx, version)
	if err != nil {
		return err
	}
	args, err := mirrorPushArgs(m.remote, candidate, version, refs, func(old, next string) error {
		_, e := m.git(ctx, false, "merge-base", "--is-ancestor", old, next)
		return e
	})
	if err != nil {
		return err
	}
	_, err = m.git(ctx, true, args...)
	return err
}

func mainIsDefault(data []byte, candidate string) bool {
	fields := strings.Fields(string(data))
	return len(fields) == 5 && fields[0] == "ref:" && fields[1] == "refs/heads/main" && fields[2] == "HEAD" && fields[3] == candidate && fields[4] == "HEAD"
}

// MirrorApproved creates an isolated repository fetched only from the fixed
// canonical public host. Private checkout history never enters the Git push.
func MirrorApproved(ctx context.Context, p repocheck.Proposal) (out repocheck.Receipt, resultErr error) {
	out = repocheck.Receipt{Schema: "aom04a.publication-receipt.v1", Repository: repocheck.Repository, Candidate: p.Candidate, Version: p.Version, Canonical: "unknown", Mirror: "not_started", Assets: append([]repocheck.Asset(nil), p.Assets...), ObservedIDs: []string{}}
	digest, e := repocheck.ProposalDigest(p)
	out.Proposal = digest
	if e != nil || repocheck.ValidateProposal(p, time.Now()) != nil || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("AOM_REF") != "refs/heads/main" || os.Getenv("AOM_SHA") != p.Candidate || os.Getenv("AOM_RUN") != p.Run || os.Getenv("AOM_ATTEMPT") != "1" {
		return out, ErrAuthority
	}
	destination := os.Getenv("AOM_MIRROR_REPOSITORY")
	if !mirrorPath.MatchString(destination) || strings.Contains(destination, "..") || strings.HasSuffix(destination, ".git") {
		return out, ErrAuthority
	}
	token, e := credential("AOM_MIRROR_TOKEN")
	if e != nil {
		return out, e
	}
	ctx = withAttemptLog(ctx, p, os.Getenv("AOM_INITIATOR"))
	// Canonical readback is authenticated with the read-only built-in token so
	// it is not subject to the anonymous per-IP rate limit shared by runners,
	// and so a 403 rate-limit reply is recognised and retried rather than
	// classified as an authority failure.
	readToken, e := credential("AOM_GITHUB_TOKEN")
	if e != nil {
		return out, e
	}
	canonical := newGitHub(readToken)
	jobCtx, cancel, e := canonical.jobContext(ctx, p, "AOM / mirror")
	if e != nil {
		return out, e
	}
	defer cancel()
	ctx = jobCtx
	tag, e := canonical.ReadTag(ctx, p.Version)
	if e != nil || !tag.Exists || tag.Identity != p.Candidate {
		return out, ErrAuthority
	}
	release, ok, e := canonical.release(ctx, p.Version)
	if e != nil || !ok || len(release.Assets) != len(p.Assets) {
		return out, ErrUnknown
	}
	for _, a := range p.Assets {
		o, e := canonical.ReadAsset(ctx, p.Version, a.Name)
		if e != nil || !o.Exists || o.Identity != a.SHA256 {
			return out, ErrUnknown
		}
		out.ObservedIDs = append(out.ObservedIDs, o.ID)
	}
	out.Canonical = "verified"
	dir, e := os.MkdirTemp("", "aom-public-mirror-")
	if e != nil {
		return out, ErrUnknown
	}
	defer func() {
		if os.RemoveAll(dir) != nil {
			out.Mirror = "unknown"
			resultErr = ErrUnknown
		}
	}()
	askpass := filepath.Join(dir, "askpass")
	script := []byte("#!/bin/sh\ncase \"$1\" in\n*Username*) printf '%s\\n' \"$AOM_MIRROR_TOKEN\" ;;\n*Password*) printf '%s\\n' '' ;;\n*) exit 1 ;;\nesac\n")
	if os.WriteFile(askpass, script, 0700) != nil {
		return out, ErrUnknown
	}
	m := &gitMirror{dir, "https://gitverse.ru/" + destination + ".git", token, askpass}
	if _, e = m.git(ctx, false, "init", "--quiet"); e != nil {
		return out, e
	}
	if _, e = m.git(ctx, false, "fetch", "--no-tags", "https://github.com/open-agent-ops/spec.git", p.Candidate); e != nil {
		return out, e
	}
	actual, e := m.git(ctx, false, "rev-parse", "--verify", "FETCH_HEAD^{commit}")
	if e != nil || strings.TrimSpace(string(actual)) != p.Candidate {
		return out, ErrConflict
	}
	out.Mirror, e = mirror(ctx, p, m, backoff)
	if e == nil {
		head, err := m.git(ctx, true, "ls-remote", "--symref", m.remote, "HEAD")
		if err != nil || !mainIsDefault(head, p.Candidate) {
			out.Mirror = "unknown"
			if err != nil {
				return out, err
			}
			return out, ErrAuthority
		}
	}
	return out, e
}
