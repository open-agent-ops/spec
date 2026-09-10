package releaseops

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func localGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.invalid")
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %v %s", args, e, b)
	}
	return strings.TrimSpace(string(b))
}
func TestMirrorMainAtomicLifecycle(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	dst := filepath.Join(root, "mirror.git")
	if e := os.Mkdir(src, 0700); e != nil {
		t.Fatal(e)
	}
	localGit(t, src, "init", "-q")
	localGit(t, src, "commit", "--allow-empty", "-qm", "first")
	first := localGit(t, src, "rev-parse", "HEAD")
	localGit(t, root, "init", "--bare", "-q", dst)
	localGit(t, src, "push", dst, first+":refs/tags/v0.4.0")
	observe := func(version string) mirrorRefs {
		t.Helper()
		refs, e := parseMirrorRefs([]byte(localGit(t, src, "ls-remote", "--refs", dst, "refs/tags/"+version, "refs/heads/main")), version)
		if e != nil {
			t.Fatal(e)
		}
		return refs
	}
	ancestor := func(old, next string) error {
		return exec.Command("git", "-C", src, "merge-base", "--is-ancestor", old, next).Run()
	}
	push := func(candidate, version string) {
		t.Helper()
		args, e := mirrorPushArgs(dst, candidate, version, observe(version), ancestor)
		if e != nil {
			t.Fatal(e)
		}
		localGit(t, src, args...)
	}
	refs := observe("v0.4.0")
	if refs.tag != first || refs.main != "" {
		t.Fatal(refs)
	}
	push(first, "v0.4.0")
	push(first, "v0.4.0")
	refs = observe("v0.4.0")
	if refs.tag != first || refs.main != first {
		t.Fatal(refs)
	}
	localGit(t, dst, "symbolic-ref", "HEAD", "refs/heads/main")
	if !mainIsDefault([]byte(localGit(t, src, "ls-remote", "--symref", dst, "HEAD")), first) {
		t.Fatal("main not default")
	}
	localGit(t, src, "commit", "--allow-empty", "-qm", "second")
	second := localGit(t, src, "rev-parse", "HEAD")
	push(second, "v0.4.1")
	if got := observe("v0.4.1"); got.tag != second || got.main != second {
		t.Fatal(got)
	}
	if _, e := mirrorPushArgs(dst, first, "v0.4.0", observe("v0.4.0"), ancestor); !errors.Is(e, ErrConflict) {
		t.Fatal("rollback allowed", e)
	}
	if _, e := mirrorPushArgs(dst, second, "v0.4.0", observe("v0.4.0"), ancestor); !errors.Is(e, ErrConflict) {
		t.Fatal("immutable tag changed", e)
	}
	localGit(t, src, "checkout", "--detach", first)
	localGit(t, src, "commit", "--allow-empty", "-qm", "divergent")
	divergent := localGit(t, src, "rev-parse", "HEAD")
	if _, e := mirrorPushArgs(dst, divergent, "v0.4.2", observe("v0.4.2"), ancestor); !errors.Is(e, ErrConflict) {
		t.Fatal("divergence allowed", e)
	}
	localGit(t, src, "checkout", "--detach", second)
	localGit(t, src, "commit", "--allow-empty", "-qm", "third")
	third := localGit(t, src, "rev-parse", "HEAD")
	// Simulate a server rejecting main: atomic push must not leave the new tag.
	hook := []byte("#!/bin/sh\n[ \"$1\" != refs/heads/main ]\n")
	if e := os.WriteFile(filepath.Join(dst, "hooks", "update"), hook, 0700); e != nil {
		t.Fatal(e)
	}
	args, e := mirrorPushArgs(dst, third, "v0.4.2", observe("v0.4.2"), ancestor)
	if e != nil {
		t.Fatal(e)
	}
	if e := exec.Command("git", append([]string{"-C", src}, args...)...).Run(); e == nil {
		t.Fatal("atomic rejection accepted")
	}
	if got := observe("v0.4.2"); got.tag != "" || got.main != second {
		t.Fatal("partial atomic update", got)
	}
	// Even a valid pair must be rejected when the server lacks atomic support.
	if e := os.Remove(filepath.Join(dst, "hooks", "update")); e != nil {
		t.Fatal(e)
	}
	localGit(t, dst, "config", "receive.advertiseAtomic", "false")
	if e := exec.Command("git", append([]string{"-C", src}, args...)...).Run(); e == nil {
		t.Fatal("non-atomic fallback")
	}
	if got := observe("v0.4.2"); got.tag != "" || got.main != second {
		t.Fatal("non-atomic mutation", got)
	}
}
func TestMirrorRefParsingAndDefault(t *testing.T) {
	sha := strings.Repeat("a", 40)
	for _, raw := range []string{sha + " refs/heads/other", sha + " refs/heads/main\n" + sha + " refs/heads/main", "invalid refs/heads/main", sha + " refs/tags/v0.4.0^{}"} {
		if _, e := parseMirrorRefs([]byte(raw), "v0.4.0"); e == nil {
			t.Fatal("bad ref output accepted", raw)
		}
	}
	for _, raw := range []string{"", sha + " HEAD", "ref: refs/heads/master HEAD\n" + sha + " HEAD", "ref: refs/heads/main HEAD\n" + strings.Repeat("b", 40) + " HEAD"} {
		if mainIsDefault([]byte(raw), sha) {
			t.Fatal("wrong HEAD accepted", raw)
		}
	}
}
func TestMirrorPushProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		present := rapid.Bool().Draw(t, "tag_present")
		sameMain := rapid.Bool().Draw(t, "main_present")
		sha := strings.Repeat("a", 40)
		refs := mirrorRefs{}
		if present {
			refs.tag = sha
		}
		if sameMain {
			refs.main = sha
		}
		args, e := mirrorPushArgs("https://gitverse.ru/open-agent-ops/spec.git", sha, "v0.4.0", refs, func(string, string) error { t.Fatal("unnecessary ancestry check"); return ErrConflict })
		if e != nil || strings.Join(args, " ") != "push --atomic --porcelain https://gitverse.ru/open-agent-ops/spec.git "+sha+":refs/tags/v0.4.0 "+sha+":refs/heads/main" {
			t.Fatal(args, e)
		}
	})
}

// An existing tag with a missing main must reach the write/reconciliation path.
type partialMirror struct {
	tag, main string
	writes    int
}

func (f *partialMirror) Observe(context.Context, string) (string, bool, error) {
	return f.tag, f.tag != "" && f.tag == f.main, nil
}
func (f *partialMirror) Push(_ context.Context, sha, _ string) error {
	f.writes++
	f.tag = sha
	f.main = sha
	return nil
}
func TestExistingTagDoesNotSkipMainRepair(t *testing.T) {
	p, _, _, _ := candidate()
	p.ExpiresAt = time.Now().Add(time.Hour)
	f := &partialMirror{tag: p.Candidate}
	state, e := mirror(context.Background(), p, f, func(context.Context) error { return nil })
	if e != nil || state != "verified" || f.writes != 1 || f.main != p.Candidate {
		t.Fatal(state, e, f)
	}
}
