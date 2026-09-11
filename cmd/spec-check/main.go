// spec-check provides fixed read-only repository checks. It has no publisher
// dependency and never interprets contributor prose as executable commands.
package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/open-agent-ops/spec/repocheck"
)

func command(name string, args ...string) ([]byte, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "CGO_ENABLED=0")
	out, errout := captureBuffer{remaining: 256 << 20}, captureBuffer{remaining: 8 << 20}
	cmd.Stdout = &out
	cmd.Stderr = &errout
	e := cmd.Run()
	if e != nil {
		e = repocheck.Unknownf("%s %s: %v: %s", filepath.Base(name), strings.Join(args, " "), e, excerpt(errout.Bytes()))
	}
	return out.Bytes(), errout.Bytes(), e
}

// excerpt keeps the first line of a tool's stderr for diagnostics. Tool output
// is bounded process text from a pinned executable, not contributor prose.
func excerpt(b []byte) string {
	line := strings.TrimSpace(string(b))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	if len(line) > 200 {
		line = line[:200] + "..."
	}
	return line
}
func snapshot(strict bool) (repocheck.Snapshot, error) {
	// Git archive excludes VCS data and untracked CI scratch by construction.
	top, _, gitErr := command("git", "rev-parse", "--show-toplevel")
	if gitErr == nil && strict {
		dirty, _, e := command("git", "status", "--porcelain", "--untracked-files=no", "--", ".")
		if e != nil {
			return nil, e
		}
		if len(dirty) != 0 {
			return nil, repocheck.Rejectedf("snapshot: working tree has uncommitted tracked changes")
		}
		cwd, e := os.Getwd()
		if e != nil {
			return nil, repocheck.Unknownf("snapshot: getwd: %v", e)
		}
		prefix, e := filepath.Rel(strings.TrimSpace(string(top)), cwd)
		if e != nil || strings.HasPrefix(prefix, "..") {
			return nil, repocheck.Invalidf("snapshot: working directory is outside the git top level")
		}
		ref := "HEAD^{tree}"
		if prefix != "." {
			ref += ":" + filepath.ToSlash(prefix)
		}
		raw, _, e := command("git", "-C", strings.TrimSpace(string(top)), "archive", "--format=tar", ref)
		if e != nil {
			return nil, e
		}
		tr := tar.NewReader(bytes.NewReader(raw))
		out := repocheck.Snapshot{}
		for {
			h, e := tr.Next()
			if errors.Is(e, io.EOF) {
				break
			}
			if e != nil {
				return nil, repocheck.Unknownf("snapshot: git archive stream: %v", e)
			}
			if h.Typeflag == tar.TypeDir {
				continue
			}
			switch {
			case h.Typeflag != tar.TypeReg:
				return nil, repocheck.Rejectedf("snapshot: %s is not a regular file", h.Name)
			case !repocheck.ExportPath(h.Name):
				return nil, repocheck.Rejectedf("snapshot: path not exportable: %s", h.Name)
			case len(out) >= 256:
				return nil, repocheck.Rejectedf("snapshot: more than 256 files")
			case h.Size > 32<<20:
				return nil, repocheck.Rejectedf("snapshot: %s exceeds 32 MiB", h.Name)
			}
			b, e := io.ReadAll(io.LimitReader(tr, (32<<20)+1))
			if e != nil {
				return nil, repocheck.Unknownf("snapshot: reading %s: %v", h.Name, e)
			}
			if _, exists := out[h.Name]; exists {
				return nil, repocheck.Rejectedf("snapshot: duplicate archive member: %s", h.Name)
			}
			out[h.Name] = b
		}
		return out, nil
	}
	root, e := os.OpenRoot(".")
	if e != nil {
		return nil, repocheck.Unknownf("snapshot: open root: %v", e)
	}
	defer root.Close()
	out := repocheck.Snapshot{}
	e = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return repocheck.Unknownf("snapshot: walk %s: %v", path, err)
		}
		if d.IsDir() {
			return nil
		}
		switch {
		case !d.Type().IsRegular():
			return repocheck.Rejectedf("snapshot: %s is not a regular file", path)
		case !repocheck.ExportPath(path):
			return repocheck.Rejectedf("snapshot: path not exportable: %s", path)
		case len(out) >= 256:
			return repocheck.Rejectedf("snapshot: more than 256 files")
		}
		info, err := d.Info()
		if err != nil {
			return repocheck.Unknownf("snapshot: stat %s: %v", path, err)
		}
		if info.Size() > 32<<20 {
			return repocheck.Rejectedf("snapshot: %s exceeds 32 MiB", path)
		}
		f, err := root.Open(path)
		if err != nil {
			return repocheck.Unknownf("snapshot: open %s: %v", path, err)
		}
		b, readErr := io.ReadAll(io.LimitReader(f, (32<<20)+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil {
			return repocheck.Unknownf("snapshot: read %s", path)
		}
		out[path] = b
		return nil
	})
	return out, e
}
func inputs(s repocheck.Snapshot) (repocheck.Policy, repocheck.ToolLock, error) {
	var p repocheck.Policy
	var lock repocheck.ToolLock
	if e := repocheck.Decode(s["process/policy.json"], &p); e != nil {
		return p, lock, repocheck.Invalidf("process/policy.json: %v", e)
	}
	if e := p.Validate(); e != nil {
		return p, lock, e
	}
	// The full lock additionally carries archive and module records; this typed
	// projection is used only by the workflow policy, after duplicate validation.
	if _, e := repocheck.JSON(s["process/toolchain.lock.json"]); e != nil {
		return p, lock, repocheck.Invalidf("process/toolchain.lock.json: %v", e)
	}
	if e := json.Unmarshal(s["process/toolchain.lock.json"], &lock); e != nil {
		return p, lock, repocheck.Invalidf("process/toolchain.lock.json: %v", e)
	}
	return p, lock, nil
}
func check(name string, s repocheck.Snapshot, p repocheck.Policy, lock repocheck.ToolLock) error {
	switch name {
	case "policy":
		if e := repocheck.Composition(s, p); e != nil {
			return e
		}
		if e := repocheck.Workflow(s[".github/workflows/ci.yml"], lock, false); e != nil {
			return fmt.Errorf("ci.yml: %w", e)
		}
		if e := repocheck.Workflow(s[".github/workflows/release.yml"], lock, true); e != nil {
			return fmt.Errorf("release.yml: %w", e)
		}
		if e := repocheck.MirrorBootstrapWorkflow(s[".github/workflows/mirror-bootstrap.yml"], lock); e != nil {
			return fmt.Errorf("mirror-bootstrap.yml: %w", e)
		}
		return repocheck.Docs(s, p)
	case "docs":
		return repocheck.Docs(s, p)
	default:
		return repocheck.Invalidf("unknown check %q", name)
	}
}
func main() {
	if e := run(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, "spec_check_failed: "+e.Error())
		os.Exit(exitCode(e))
	}
}

// exitCode maps sentinel classes to the documented codes. An error outside the
// three classes is an unexpected host or tool failure, never a verdict on the
// candidate, so it reports as unknown rather than as rejected.
func exitCode(e error) int {
	switch {
	case errors.Is(e, repocheck.ErrInput):
		return 2
	case errors.Is(e, repocheck.ErrRejected):
		return 3
	default:
		return 4
	}
}
func simpleRun(args []string) error {
	if len(args) != 1 || (args[0] != "policy" && args[0] != "docs" && args[0] != "composition") {
		return repocheck.Invalidf("usage: spec-check policy|docs|composition | gate <name> | prepare-release")
	}
	if e := headMatchesCandidate(); e != nil {
		return e
	}
	s, e := snapshot(true)
	if e != nil {
		return e
	}
	p, lock, e := inputs(s)
	if e != nil {
		return e
	}
	if args[0] == "composition" {
		e = repocheck.Composition(s, p)
	} else {
		e = check(args[0], s, p, lock)
	}
	if e != nil {
		return e
	}
	fmt.Println("spec_check_passed")
	return nil
}

// headMatchesCandidate refuses to evaluate anything but the exact commit the
// hosting workflow was dispatched for.
func headMatchesCandidate() error {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		return nil
	}
	head, _, err := command("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if got := strings.TrimSpace(string(head)); got != os.Getenv("AOM_SHA") {
		return repocheck.Rejectedf("HEAD %s differs from AOM_SHA", got)
	}
	return nil
}

func run(args []string) error {
	if len(args) == 1 && args[0] == "prepare-release" {
		return prepareRelease()
	}
	if len(args) == 2 && args[0] == "gate" {
		return gate(args[1])
	}
	return simpleRun(args)
}
func tool(name string) string {
	dir := os.Getenv("AOM_TOOLS_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "aom-spec-tools-1.26.8")
	}
	if name == "go" {
		return filepath.Join(dir, "go", "bin", "go")
	}
	return filepath.Join(dir, "bin", name)
}
func evidenceDir() string {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return ".aom-evidence"
	}
	return filepath.Join(os.TempDir(), "aom-spec-local-evidence")
}
func normalizedArchive(s repocheck.Snapshot) ([]byte, error) {
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	names := make([]string, 0, len(s))
	for n := range s {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		h := &tar.Header{Name: n, Mode: 0644, Size: int64(len(s[n])), ModTime: time.Unix(0, 0), Format: tar.FormatUSTAR}
		if e := tw.WriteHeader(h); e != nil {
			return nil, repocheck.Unknownf("archive header %s: %v", n, e)
		}
		if _, e := tw.Write(s[n]); e != nil {
			return nil, repocheck.Unknownf("archive body %s: %v", n, e)
		}
	}
	if e := tw.Close(); e != nil {
		return nil, repocheck.Unknownf("archive close: %v", e)
	}
	return b.Bytes(), nil
}
func gate(name string) error {
	allowed := map[string]bool{"policy": true, "conformance": true, "docs": true, "supply-chain": true, "aggregate": true, "heavy": true, "reproducibility": true}
	if !allowed[name] {
		return repocheck.Invalidf("unknown gate %q", name)
	}
	dir := evidenceDir()
	if e := os.MkdirAll(dir, 0700); e != nil {
		return repocheck.Unknownf("evidence dir: %v", e)
	}
	out, e := os.Create(filepath.Join(dir, name+".stdout"))
	if e != nil {
		return repocheck.Unknownf("evidence stdout: %v", e)
	}
	defer out.Close()
	errout, e := os.Create(filepath.Join(dir, name+".stderr"))
	if e != nil {
		return repocheck.Unknownf("evidence stderr: %v", e)
	}
	defer errout.Close()
	if e := headMatchesCandidate(); e != nil {
		return e
	}
	s, e := snapshot(true)
	if e != nil {
		return e
	}
	p, lock, e := inputs(s)
	if e != nil {
		return e
	}
	execCheck := func(binary string, args ...string) error {
		timeout := 20 * time.Minute
		if name == "heavy" || name == "reproducibility" {
			timeout = 45 * time.Minute
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Stdout = io.MultiWriter(out, os.Stdout)
		cmd.Stderr = io.MultiWriter(errout, os.Stderr)
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
		if e := cmd.Run(); e != nil {
			var exit *exec.ExitError
			if errors.As(e, &exit) {
				return repocheck.Rejectedf("%s %s: %v", filepath.Base(binary), strings.Join(args, " "), e)
			}
			return repocheck.Unknownf("%s %s: %v", filepath.Base(binary), strings.Join(args, " "), e)
		}
		return nil
	}
	switch name {
	case "policy":
		e = check(name, s, p, lock)
		if e == nil {
			e = execCheck(tool("actionlint"), ".github/workflows/ci.yml", ".github/workflows/release.yml", ".github/workflows/mirror-bootstrap.yml")
		}
	case "docs":
		e = check(name, s, p, lock)
	case "conformance":
		e = execCheck(tool("go"), "test", "-mod=readonly", "./...", "-count=1", "-rapid.seed=30404", "-rapid.checks=100")
	case "heavy":
		e = execCheck(tool("go"), "test", "-mod=readonly", "-race", "./...", "-count=1", "-rapid.seed=30404", "-rapid.checks=1000")
	case "supply-chain":
		e = execCheck(tool("go"), "mod", "verify")
		if e == nil {
			e = execCheck(tool("go"), "mod", "tidy", "-diff")
		}
		if e == nil {
			e = execCheck(tool("go"), "vet", "./...")
		}
		if e == nil {
			e = moduleGraph(s)
		}
		if e == nil {
			e = vulnerabilityChecks(out, errout)
		}
	case "reproducibility":
		first, err := normalizedArchive(s)
		if err != nil {
			e = err
			break
		}
		again, err := snapshot(true)
		if err != nil {
			e = err
			break
		}
		second, err := normalizedArchive(again)
		if err != nil {
			e = err
		} else if !bytes.Equal(first, second) {
			e = repocheck.Rejectedf("reproducibility: two source archives of the same tree differ")
		}
		if e == nil {
			fmt.Fprintln(out, "reproducible_source_sha256="+repocheck.Hash(first))
			var binaryHash string
			binaryHash, e = reproducibleBinary(s)
			if e == nil {
				fmt.Fprintln(out, "reproducible_binary_sha256="+binaryHash)
			}
		}
	case "aggregate":
		if needs := os.Getenv("AOM_NEEDS"); needs != "" {
			e = repocheck.NeedsSuccess([]byte(needs), os.Getenv("AOM_EVENT") != "pull_request")
		}
		if e == nil {
			e = aggregateEvidence(dir, s, p)
		}
	}
	if e != nil {
		fmt.Fprintln(errout, "gate_failed: "+e.Error())
	} else {
		fmt.Fprintln(out, "gate_passed="+name)
	}
	if out.Sync() != nil || errout.Sync() != nil {
		return repocheck.Unknownf("evidence stream sync failed")
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		stdout, readErr := os.ReadFile(filepath.Join(dir, name+".stdout"))
		if readErr != nil {
			return repocheck.Unknownf("evidence stdout: %v", readErr)
		}
		stderr, readErr := os.ReadFile(filepath.Join(dir, name+".stderr"))
		if readErr != nil {
			return repocheck.Unknownf("evidence stderr: %v", readErr)
		}
		attempt, err := strconv.Atoi(os.Getenv("AOM_ATTEMPT"))
		if err != nil {
			return repocheck.Invalidf("AOM_ATTEMPT %q is not an integer", os.Getenv("AOM_ATTEMPT"))
		}
		result := "success"
		if e != nil {
			result = "failure"
		}
		record := repocheck.Gate{Schema: "aom04a.gate-record.v1", Repository: repocheck.Repository, Candidate: os.Getenv("AOM_SHA"), Base: os.Getenv("AOM_BASE"), Policy: repocheck.Hash(s["process/policy.json"]), Lock: repocheck.Hash(s["process/toolchain.lock.json"]), Workflow: os.Getenv("AOM_WORKFLOW"), Run: os.Getenv("AOM_RUN"), Job: name, Attempt: attempt, Result: result, Stdout: repocheck.Hash(stdout), Stderr: repocheck.Hash(stderr)}
		b, err := json.Marshal(record)
		if err != nil {
			return repocheck.Unknownf("gate record: %v", err)
		}
		if err := repocheck.ValidateRecord(s["process/schemas/gate-record.schema.json"], b); err != nil {
			return fmt.Errorf("gate record: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), append(b, '\n'), 0600); err != nil {
			return repocheck.Unknownf("gate record: %v", err)
		}
	}
	return e
}
func moduleGraph(s repocheck.Snapshot) error {
	var lock struct {
		Modules []struct {
			Name, Version, Sum string
			GoModSum           string `json:"go_mod_sum"`
			License            string `json:"license_sha256"`
		} `json:"modules"`
	}
	if e := json.Unmarshal(s["process/toolchain.lock.json"], &lock); e != nil {
		return repocheck.Invalidf("process/toolchain.lock.json modules: %v", e)
	}
	raw, _, e := command(tool("go"), "list", "-mod=readonly", "-m", "-json", "all")
	if e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	seen := map[string]bool{}
	for {
		var m struct {
			Path, Version, Sum string
			Main               bool
			Replace            any
		}
		e := d.Decode(&m)
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			return repocheck.Unknownf("go list -m output: %v", e)
		}
		if m.Replace != nil {
			return repocheck.Rejectedf("module graph: %s uses a replace directive", m.Path)
		}
		if m.Main {
			continue
		}
		if seen[m.Path] {
			return repocheck.Rejectedf("module graph: %s listed twice", m.Path)
		}
		seen[m.Path] = true
		match := false
		for _, l := range lock.Modules {
			if l.Name == m.Path && l.Version == m.Version {
				// Lazy module loading may omit Sum for graph-only modules. Resolve the
				// exact pinned module outside the candidate, then verify all lock facts.
				info, err := downloadModule(m.Path, m.Version)
				if err != nil {
					return err
				}
				if info.Sum != l.Sum || info.GoModSum != l.GoModSum {
					return repocheck.Rejectedf("module graph: %s@%s sums differ from toolchain lock", m.Path, m.Version)
				}
				license, err := os.ReadFile(filepath.Join(info.Dir, "LICENSE"))
				if err != nil {
					return repocheck.Rejectedf("module graph: %s@%s has no readable LICENSE file", m.Path, m.Version)
				}
				if len(license) > 1<<20 || repocheck.Hash(license) != l.License {
					return repocheck.Rejectedf("module graph: %s@%s LICENSE digest differs from toolchain lock", m.Path, m.Version)
				}
				match = true
			}
		}
		if !match {
			return repocheck.Rejectedf("module graph: %s@%s is not in toolchain lock", m.Path, m.Version)
		}
	}
	if len(seen) != len(lock.Modules) {
		return repocheck.Rejectedf("module graph: %d modules resolved, toolchain lock lists %d", len(seen), len(lock.Modules))
	}
	return nil
}
func prepareRelease() error {
	candidate, version := os.Getenv("AOM_CANDIDATE"), os.Getenv("AOM_VERSION")
	switch {
	case os.Getenv("GITHUB_ACTIONS") != "true":
		return repocheck.Rejectedf("prepare-release runs only under GitHub Actions")
	case os.Getenv("AOM_REF") != "refs/heads/main":
		return repocheck.Rejectedf("prepare-release: AOM_REF %q is not refs/heads/main", os.Getenv("AOM_REF"))
	case os.Getenv("AOM_ATTEMPT") != "1":
		return repocheck.Rejectedf("prepare-release: AOM_ATTEMPT %q, want 1", os.Getenv("AOM_ATTEMPT"))
	case !repocheck.Commit(candidate):
		return repocheck.Rejectedf("prepare-release: AOM_CANDIDATE is not a 40-hex commit")
	case candidate != os.Getenv("AOM_SHA") || candidate != os.Getenv("AOM_WORKFLOW"):
		return repocheck.Rejectedf("prepare-release: AOM_CANDIDATE differs from AOM_SHA or AOM_WORKFLOW")
	case !repocheck.Version(version):
		return repocheck.Rejectedf("prepare-release: AOM_VERSION %q is not a stable v0/v1 SemVer", version)
	}
	if e := headMatchesCandidate(); e != nil {
		return e
	}
	for _, name := range []string{"policy", "conformance", "docs", "supply-chain", "heavy", "reproducibility", "aggregate"} {
		if e := gate(name); e != nil {
			return fmt.Errorf("gate %s: %w", name, e)
		}
	}
	if e := headMatchesCandidate(); e != nil {
		return e
	}
	s, e := snapshot(true)
	if e != nil {
		return e
	}
	archive, e := normalizedArchive(s)
	if e != nil {
		return e
	}
	if e := os.MkdirAll(".aom-release", 0700); e != nil {
		return repocheck.Unknownf("release dir: %v", e)
	}
	if e := os.WriteFile(".aom-release/source.tar", archive, 0600); e != nil {
		return repocheck.Unknownf("source.tar: %v", e)
	}
	sbom, e := releaseSBOM(s)
	if e != nil {
		return e
	}
	if e := os.WriteFile(".aom-release/sbom.json", sbom, 0600); e != nil {
		return repocheck.Unknownf("sbom.json: %v", e)
	}
	_, _, e = command(tool("go"), "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-buildid=", "-o", ".aom-release/spec-release", "./cmd/spec-release")
	if e != nil {
		return e
	}
	tree, _, e := command("git", "rev-parse", "HEAD^{tree}")
	if e != nil {
		return e
	}
	now := time.Now().UTC()
	p := repocheck.Proposal{Schema: "aom04a.release-proposal.v1", Repository: repocheck.Repository, Candidate: candidate, Version: version, Tree: strings.TrimSpace(string(tree)), Inventory: repocheck.Hash(s["composition-manifest.json"]), SBOM: repocheck.Hash(sbom), Workflow: candidate, Run: os.Getenv("AOM_RUN"), Attempt: 1, Lock: repocheck.Hash(s["process/toolchain.lock.json"]), Policy: repocheck.Hash(s["process/policy.json"]), CheckedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	evidence := repocheck.Snapshot{}
	for _, name := range []string{"policy", "conformance", "docs", "supply-chain", "aggregate", "heavy", "reproducibility"} {
		for _, suffix := range []string{".json", ".stdout", ".stderr"} {
			b, err := os.ReadFile(filepath.Join(evidenceDir(), name+suffix))
			if err != nil {
				return repocheck.Unknownf("evidence %s%s: %v", name, suffix, err)
			}
			evidence[name+suffix] = b
		}
	}
	evidenceArchive, err := normalizedArchive(evidence)
	if err != nil {
		return err
	}
	if e := os.WriteFile(".aom-release/evidence.tar", evidenceArchive, 0600); e != nil {
		return repocheck.Unknownf("evidence.tar: %v", e)
	}
	for _, name := range []string{"source.tar", "sbom.json", "spec-release", "evidence.tar"} {
		b, e := os.ReadFile(filepath.Join(".aom-release", name))
		if e != nil {
			return repocheck.Unknownf("release asset %s: %v", name, e)
		}
		h := sha256.Sum256(b)
		p.Assets = append(p.Assets, repocheck.Asset{Name: name, SHA256: hex.EncodeToString(h[:]), Size: int64(len(b))})
	}
	b, e := json.MarshalIndent(p, "", "  ")
	if e != nil {
		return repocheck.Unknownf("proposal: %v", e)
	}
	if e := repocheck.ValidateRecord(s["process/schemas/release-proposal.schema.json"], b); e != nil {
		return fmt.Errorf("proposal: %w", e)
	}
	if e := os.WriteFile(".aom-release/proposal.json", append(b, '\n'), 0600); e != nil {
		return repocheck.Unknownf("proposal.json: %v", e)
	}
	return nil
}

func aggregateEvidence(dir string, s repocheck.Snapshot, p repocheck.Policy) error {
	names := []string{"policy", "conformance", "docs", "supply-chain"}
	full := os.Getenv("AOM_EVENT") != "pull_request"
	if full {
		names = append(names, "heavy", "reproducibility")
	}
	streams := map[string][]byte{}
	records := []repocheck.Gate{}
	for _, n := range names {
		raw, e := os.ReadFile(filepath.Join(dir, n+".json"))
		if e != nil {
			return repocheck.Rejectedf("aggregate: gate record %s.json not downloaded: %v", n, e)
		}
		if e := repocheck.ValidateRecord(s["process/schemas/gate-record.schema.json"], raw); e != nil {
			return repocheck.Rejectedf("aggregate: %s.json: %v", n, e)
		}
		var record repocheck.Gate
		if e := repocheck.Decode(raw, &record); e != nil {
			return repocheck.Rejectedf("aggregate: %s.json: %v", n, e)
		}
		records = append(records, record)
		for _, suffix := range []string{".stdout", ".stderr"} {
			b, e := os.ReadFile(filepath.Join(dir, n+suffix))
			if e != nil {
				return repocheck.Rejectedf("aggregate: stream %s%s not downloaded: %v", n, suffix, e)
			}
			streams[n+suffix] = b
		}
	}
	attempt, e := strconv.Atoi(os.Getenv("AOM_ATTEMPT"))
	if e != nil {
		return repocheck.Invalidf("AOM_ATTEMPT %q is not an integer", os.Getenv("AOM_ATTEMPT"))
	}
	b := repocheck.EvidenceBinding{Candidate: os.Getenv("AOM_SHA"), Base: os.Getenv("AOM_BASE"), Policy: repocheck.Hash(s["process/policy.json"]), Lock: repocheck.Hash(s["process/toolchain.lock.json"]), Workflow: os.Getenv("AOM_WORKFLOW"), Run: os.Getenv("AOM_RUN"), Attempt: attempt, Full: full}
	return repocheck.PreAggregateReady(records, p, b, streams)
}

func releaseSBOM(s repocheck.Snapshot) ([]byte, error) {
	// Preserve the complete approved lock inventory in a single valid JSON object.
	// Runtime module verification is performed independently before packaging.
	if e := moduleGraph(s); e != nil {
		return nil, e
	}
	var lock any
	if e := json.Unmarshal(s["process/toolchain.lock.json"], &lock); e != nil {
		return nil, repocheck.Invalidf("process/toolchain.lock.json: %v", e)
	}
	actual, _, e := command(tool("go"), "env", "GOVERSION", "GOOS", "GOARCH")
	if e != nil {
		return nil, e
	}
	observations, e := toolObservations()
	if e != nil {
		return nil, e
	}
	return json.MarshalIndent(map[string]any{"format": "aom04a-sbom-v1", "repository": repocheck.Repository, "lock_sha256": repocheck.Hash(s["process/toolchain.lock.json"]), "inventory_sha256": repocheck.Hash(s["composition-manifest.json"]), "toolchain_observation": string(actual), "tool_executables": observations, "runner_observation": map[string]string{"os": os.Getenv("RUNNER_OS"), "arch": os.Getenv("RUNNER_ARCH"), "image_os": os.Getenv("ImageOS"), "image_version": os.Getenv("ImageVersion"), "trust_boundary": "provider-managed"}, "inventory": lock}, "", "  ")
}

// toolObservations hashes and inspects the actual executables used by this run.
// No local absolute paths or environment secrets enter the release inventory.
func toolObservations() (map[string]any, error) {
	result := map[string]any{}
	for _, pin := range []struct{ name, module, version string }{
		{"go", "", ""},
		{"actionlint", "github.com/rhysd/actionlint", "v1.7.12"},
		{"govulncheck", "golang.org/x/vuln", "v1.3.0"},
	} {
		f, e := os.Open(tool(pin.name))
		if e != nil {
			return nil, repocheck.Unknownf("tool %s: %v", pin.name, e)
		}
		st, statErr := f.Stat()
		if statErr != nil || !st.Mode().IsRegular() || st.Size() <= 0 || st.Size() > 128<<20 {
			f.Close()
			return nil, repocheck.Invalidf("tool %s is not a regular file within 128 MiB", pin.name)
		}
		data, readErr := io.ReadAll(io.LimitReader(f, (128<<20)+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(data) > 128<<20 || int64(len(data)) != st.Size() {
			return nil, repocheck.Unknownf("tool %s: read failed", pin.name)
		}
		info, e := buildinfo.Read(bytes.NewReader(data))
		if e != nil {
			return nil, repocheck.Rejectedf("tool %s: no Go build info: %v", pin.name, e)
		}
		if info.GoVersion != "go1.26.8" {
			return nil, repocheck.Rejectedf("tool %s built with %s, want go1.26.8", pin.name, info.GoVersion)
		}
		if info.Main.Replace != nil || (pin.module != "" && (info.Main.Path != pin.module || info.Main.Version != pin.version)) {
			return nil, repocheck.Rejectedf("tool %s main module %s@%s, want %s@%s", pin.name, info.Main.Path, info.Main.Version, pin.module, pin.version)
		}
		result[pin.name] = map[string]any{"sha256": repocheck.Hash(data), "size": len(data), "build": info}
	}
	return result, nil
}

func reproducibleBinary(s repocheck.Snapshot) (digest string, result error) {
	dir, e := os.MkdirTemp("", "aom-repro-")
	if e != nil {
		return "", repocheck.Unknownf("reproducibility: temp dir: %v", e)
	}
	defer func() {
		if e := os.RemoveAll(dir); e != nil && result == nil {
			result = repocheck.Unknownf("reproducibility: cleanup: %v", e)
		}
	}()
	var first []byte
	for _, copyName := range []string{"first", "second"} {
		root := filepath.Join(dir, copyName)
		for n, b := range s {
			if !repocheck.ExportPath(n) {
				return "", repocheck.Rejectedf("reproducibility: path not exportable: %s", n)
			}
			target := filepath.Join(root, filepath.FromSlash(n))
			if os.MkdirAll(filepath.Dir(target), 0700) != nil || os.WriteFile(target, b, 0600) != nil {
				return "", repocheck.Unknownf("reproducibility: cannot materialize %s", n)
			}
		}
		output := filepath.Join(dir, copyName+"-spec-release")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		cmd := exec.CommandContext(ctx, tool("go"), "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-buildid=", "-o", output, "./cmd/spec-release")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "CGO_ENABLED=0")
		e = cmd.Run()
		cancel()
		if e != nil {
			return "", repocheck.Rejectedf("reproducibility: %s build failed: %v", copyName, e)
		}
		b, e := os.ReadFile(output)
		if e != nil {
			return "", repocheck.Unknownf("reproducibility: read %s build: %v", copyName, e)
		}
		if first == nil {
			first = b
		} else if !bytes.Equal(first, b) {
			return "", repocheck.Rejectedf("reproducibility: two builds of spec-release differ")
		}
	}
	return repocheck.Hash(first), nil
}

type moduleDownload struct {
	Sum, GoModSum, Dir string
	Error              *struct{ Err string }
}

func downloadModule(name, version string) (result moduleDownload, resultErr error) {
	dir, e := os.MkdirTemp("", "aom-module-check-")
	if e != nil {
		return result, repocheck.Unknownf("module download: temp dir: %v", e)
	}
	defer func() {
		if e := os.RemoveAll(dir); e != nil && resultErr == nil {
			resultErr = repocheck.Unknownf("module download: cleanup: %v", e)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, tool("go"), "mod", "download", "-json", name+"@"+version)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	raw, e := cmd.Output()
	if e != nil {
		return result, repocheck.Unknownf("go mod download %s@%s: %v", name, version, e)
	}
	if len(raw) > 1<<20 || json.Unmarshal(raw, &result) != nil {
		return result, repocheck.Unknownf("go mod download %s@%s: unreadable output", name, version)
	}
	if result.Error != nil {
		return result, repocheck.Rejectedf("go mod download %s@%s: %s", name, version, result.Error.Err)
	}
	return result, nil
}

type captureBuffer struct {
	bytes.Buffer
	remaining int
}

func (b *captureBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, repocheck.ErrUnknown
	}
	n, e := b.Buffer.Write(p)
	b.remaining -= n
	return n, e
}
func vulnerabilityChecks(out, errout io.Writer) error {
	packages, stderr, e := command(tool("go"), "list", "-mod=readonly", "-deps", "-test", "./...")
	if _, err := errout.Write(stderr); err != nil {
		return repocheck.Unknownf("evidence stderr: %v", err)
	}
	if e != nil {
		return e
	}
	for _, target := range []string{"source", "actionlint", "govulncheck"} {
		args := []string{"-format=json", "./..."}
		presence := string(packages)
		if target != "source" {
			args = []string{"-mode=binary", "-format=json", tool(target)}
			symbols, stderr, err := command(tool("go"), "tool", "nm", tool(target))
			if _, e := errout.Write(stderr); e != nil {
				return repocheck.Unknownf("evidence stderr: %v", e)
			}
			if err != nil {
				return err
			}
			presence = string(symbols)
		}
		raw, stderr, err := command(tool("govulncheck"), args...)
		if _, e := out.Write(raw); e != nil {
			return repocheck.Unknownf("evidence stdout: %v", e)
		}
		if _, e := errout.Write(stderr); e != nil {
			return repocheck.Unknownf("evidence stderr: %v", e)
		}
		if err != nil {
			// govulncheck exits non-zero when it reports findings; that is a verdict
			// on the candidate, not a tool failure.
			return repocheck.Rejectedf("govulncheck %s reported findings or failed: %v", target, err)
		}
		cleared, e := repocheck.AssessVulnerabilities(raw, func(path string) bool {
			if target == "source" {
				for _, line := range strings.Split(presence, "\n") {
					if line == path {
						return true
					}
				}
				return false
			}
			return strings.Contains(presence, " "+path+".")
		})
		if e != nil {
			return fmt.Errorf("govulncheck %s: %w", target, e)
		}
		for _, fact := range cleared {
			fmt.Fprintln(out, target+":"+fact)
		}
	}
	return nil
}
