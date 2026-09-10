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
	return out.Bytes(), errout.Bytes(), e
}
func snapshot(strict bool) (repocheck.Snapshot, error) {
	// Git archive excludes VCS data and untracked CI scratch by construction.
	top, _, gitErr := command("git", "rev-parse", "--show-toplevel")
	if gitErr == nil && strict {
		dirty, _, e := command("git", "status", "--porcelain", "--untracked-files=no", "--", ".")
		if e != nil || len(dirty) != 0 {
			return nil, repocheck.ErrRejected
		}
		cwd, e := os.Getwd()
		if e != nil {
			return nil, e
		}
		prefix, e := filepath.Rel(strings.TrimSpace(string(top)), cwd)
		if e != nil || strings.HasPrefix(prefix, "..") {
			return nil, repocheck.ErrInput
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
				return nil, e
			}
			if h.Typeflag == tar.TypeDir {
				continue
			}
			if h.Typeflag != tar.TypeReg || !repocheck.ExportPath(h.Name) || len(out) >= 256 || h.Size > 32<<20 {
				return nil, repocheck.ErrRejected
			}
			b, e := io.ReadAll(io.LimitReader(tr, (32<<20)+1))
			if e != nil {
				return nil, e
			}
			if _, exists := out[h.Name]; exists {
				return nil, repocheck.ErrRejected
			}
			out[h.Name] = b
		}
		return out, nil
	}
	root, e := os.OpenRoot(".")
	if e != nil {
		return nil, e
	}
	defer root.Close()
	out := repocheck.Snapshot{}
	e = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() || !repocheck.ExportPath(path) || len(out) >= 256 {
			return repocheck.ErrRejected
		}
		info, err := d.Info()
		if err != nil || info.Size() > 32<<20 {
			return repocheck.ErrRejected
		}
		f, err := root.Open(path)
		if err != nil {
			return err
		}
		b, readErr := io.ReadAll(io.LimitReader(f, (32<<20)+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil {
			return repocheck.ErrUnknown
		}
		out[path] = b
		return nil
	})
	return out, e
}
func inputs(s repocheck.Snapshot) (repocheck.Policy, repocheck.ToolLock, error) {
	var p repocheck.Policy
	var lock repocheck.ToolLock
	if repocheck.Decode(s["process/policy.json"], &p) != nil || p.Validate() != nil {
		return p, lock, repocheck.ErrInput
	}
	// The full lock additionally carries archive and module records; this typed
	// projection is used only by the workflow policy, after duplicate validation.
	if _, e := repocheck.JSON(s["process/toolchain.lock.json"]); e != nil {
		return p, lock, e
	}
	if json.Unmarshal(s["process/toolchain.lock.json"], &lock) != nil {
		return p, lock, repocheck.ErrInput
	}
	return p, lock, nil
}
func check(name string, s repocheck.Snapshot, p repocheck.Policy, lock repocheck.ToolLock) error {
	switch name {
	case "policy":
		if repocheck.Composition(s, p) != nil {
			return repocheck.ErrRejected
		}
		if repocheck.Workflow(s[".github/workflows/ci.yml"], lock, false) != nil || repocheck.Workflow(s[".github/workflows/release.yml"], lock, true) != nil {
			return repocheck.ErrRejected
		}
		return repocheck.Docs(s, p)
	case "docs":
		return repocheck.Docs(s, p)
	default:
		return repocheck.ErrInput
	}
}
func main() {
	if e := run(os.Args[1:]); e != nil {
		code := 3
		if errors.Is(e, repocheck.ErrInput) {
			code = 2
		} else if errors.Is(e, repocheck.ErrUnknown) {
			code = 4
		}
		fmt.Fprintln(os.Stderr, "spec_check_failed")
		os.Exit(code)
	}
}
func simpleRun(args []string) error {
	if len(args) != 1 || (args[0] != "policy" && args[0] != "docs" && args[0] != "composition") {
		return repocheck.ErrInput
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		head, _, err := command("git", "rev-parse", "HEAD")
		if err != nil || strings.TrimSpace(string(head)) != os.Getenv("AOM_SHA") {
			return repocheck.ErrRejected
		}
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
		if tw.WriteHeader(h) != nil {
			return nil, repocheck.ErrUnknown
		}
		if _, e := tw.Write(s[n]); e != nil {
			return nil, e
		}
	}
	if tw.Close() != nil {
		return nil, repocheck.ErrUnknown
	}
	return b.Bytes(), nil
}
func gate(name string) error {
	allowed := map[string]bool{"policy": true, "conformance": true, "docs": true, "supply-chain": true, "aggregate": true, "heavy": true, "reproducibility": true}
	if !allowed[name] {
		return repocheck.ErrInput
	}
	dir := evidenceDir()
	if os.MkdirAll(dir, 0700) != nil {
		return repocheck.ErrUnknown
	}
	out, e := os.Create(filepath.Join(dir, name+".stdout"))
	if e != nil {
		return e
	}
	defer out.Close()
	errout, e := os.Create(filepath.Join(dir, name+".stderr"))
	if e != nil {
		return e
	}
	defer errout.Close()
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		head, _, err := command("git", "rev-parse", "HEAD")
		if err != nil || strings.TrimSpace(string(head)) != os.Getenv("AOM_SHA") {
			return repocheck.ErrRejected
		}
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
		if cmd.Run() != nil {
			return repocheck.ErrRejected
		}
		return nil
	}
	switch name {
	case "policy":
		e = check(name, s, p, lock)
		if e == nil {
			e = execCheck(tool("actionlint"), ".github/workflows/ci.yml", ".github/workflows/release.yml")
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
		if err != nil || !bytes.Equal(first, second) {
			e = repocheck.ErrRejected
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
		fmt.Fprintln(errout, "gate_failed")
	} else {
		fmt.Fprintln(out, "gate_passed="+name)
	}
	if out.Sync() != nil || errout.Sync() != nil {
		return repocheck.ErrUnknown
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		stdout, readErr := os.ReadFile(filepath.Join(dir, name+".stdout"))
		if readErr != nil {
			return readErr
		}
		stderr, readErr := os.ReadFile(filepath.Join(dir, name+".stderr"))
		if readErr != nil {
			return readErr
		}
		attempt, err := strconv.Atoi(os.Getenv("AOM_ATTEMPT"))
		if err != nil {
			return repocheck.ErrInput
		}
		result := "success"
		if e != nil {
			result = "failure"
		}
		record := repocheck.Gate{Schema: "aom04a.gate-record.v1", Repository: repocheck.Repository, Candidate: os.Getenv("AOM_SHA"), Base: os.Getenv("AOM_BASE"), Policy: repocheck.Hash(s["process/policy.json"]), Lock: repocheck.Hash(s["process/toolchain.lock.json"]), Workflow: os.Getenv("AOM_WORKFLOW"), Run: os.Getenv("AOM_RUN"), Job: name, Attempt: attempt, Result: result, Stdout: repocheck.Hash(stdout), Stderr: repocheck.Hash(stderr)}
		b, err := json.Marshal(record)
		if err != nil || repocheck.ValidateRecord(s["process/schemas/gate-record.schema.json"], b) != nil {
			return repocheck.ErrInput
		}
		if os.WriteFile(filepath.Join(dir, name+".json"), append(b, '\n'), 0600) != nil {
			return repocheck.ErrUnknown
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
	if json.Unmarshal(s["process/toolchain.lock.json"], &lock) != nil {
		return repocheck.ErrInput
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
		if e != nil || m.Replace != nil {
			return repocheck.ErrRejected
		}
		if m.Main {
			continue
		}
		if seen[m.Path] {
			return repocheck.ErrRejected
		}
		seen[m.Path] = true
		match := false
		for _, l := range lock.Modules {
			if l.Name == m.Path && l.Version == m.Version {
				// Lazy module loading may omit Sum for graph-only modules. Resolve the
				// exact pinned module outside the candidate, then verify all lock facts.
				info, err := downloadModule(m.Path, m.Version)
				if err != nil || info.Sum != l.Sum || info.GoModSum != l.GoModSum {
					return repocheck.ErrRejected
				}
				license, err := os.ReadFile(filepath.Join(info.Dir, "LICENSE"))
				if err != nil || len(license) > 1<<20 || repocheck.Hash(license) != l.License {
					return repocheck.ErrRejected
				}
				match = true
			}
		}
		if !match {
			return repocheck.ErrRejected
		}
	}
	if len(seen) != len(lock.Modules) {
		return repocheck.ErrRejected
	}
	return nil
}
func prepareRelease() error {
	candidate, version := os.Getenv("AOM_CANDIDATE"), os.Getenv("AOM_VERSION")
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("AOM_REF") != "refs/heads/main" || os.Getenv("AOM_ATTEMPT") != "1" || candidate != os.Getenv("AOM_SHA") || candidate != os.Getenv("AOM_WORKFLOW") || !repocheck.Commit(candidate) || !repocheck.Version(version) {
		return repocheck.ErrRejected
	}
	head, _, e := command("git", "rev-parse", "HEAD")
	if e != nil || strings.TrimSpace(string(head)) != candidate {
		return repocheck.ErrRejected
	}
	for _, name := range []string{"policy", "conformance", "docs", "supply-chain", "heavy", "reproducibility", "aggregate"} {
		if gate(name) != nil {
			return repocheck.ErrRejected
		}
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		head, _, err := command("git", "rev-parse", "HEAD")
		if err != nil || strings.TrimSpace(string(head)) != os.Getenv("AOM_SHA") {
			return repocheck.ErrRejected
		}
	}
	s, e := snapshot(true)
	if e != nil {
		return e
	}
	archive, e := normalizedArchive(s)
	if e != nil {
		return e
	}
	if os.MkdirAll(".aom-release", 0700) != nil {
		return repocheck.ErrUnknown
	}
	if os.WriteFile(".aom-release/source.tar", archive, 0600) != nil {
		return repocheck.ErrUnknown
	}
	sbom, e := releaseSBOM(s)
	if e != nil {
		return e
	}
	if os.WriteFile(".aom-release/sbom.json", sbom, 0600) != nil {
		return repocheck.ErrUnknown
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
				return err
			}
			evidence[name+suffix] = b
		}
	}
	evidenceArchive, err := normalizedArchive(evidence)
	if err != nil {
		return err
	}
	if os.WriteFile(".aom-release/evidence.tar", evidenceArchive, 0600) != nil {
		return repocheck.ErrUnknown
	}
	for _, name := range []string{"source.tar", "sbom.json", "spec-release", "evidence.tar"} {
		b, e := os.ReadFile(filepath.Join(".aom-release", name))
		if e != nil {
			return e
		}
		h := sha256.Sum256(b)
		p.Assets = append(p.Assets, repocheck.Asset{Name: name, SHA256: hex.EncodeToString(h[:]), Size: int64(len(b))})
	}
	b, e := json.MarshalIndent(p, "", "  ")
	if e != nil || repocheck.ValidateRecord(s["process/schemas/release-proposal.schema.json"], b) != nil {
		return repocheck.ErrInput
	}
	return os.WriteFile(".aom-release/proposal.json", append(b, '\n'), 0600)
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
		if e != nil || repocheck.ValidateRecord(s["process/schemas/gate-record.schema.json"], raw) != nil {
			return repocheck.ErrRejected
		}
		var record repocheck.Gate
		if repocheck.Decode(raw, &record) != nil {
			return repocheck.ErrRejected
		}
		records = append(records, record)
		for _, suffix := range []string{".stdout", ".stderr"} {
			b, e := os.ReadFile(filepath.Join(dir, n+suffix))
			if e != nil {
				return repocheck.ErrRejected
			}
			streams[n+suffix] = b
		}
	}
	attempt, e := strconv.Atoi(os.Getenv("AOM_ATTEMPT"))
	if e != nil {
		return repocheck.ErrRejected
	}
	b := repocheck.EvidenceBinding{Candidate: os.Getenv("AOM_SHA"), Base: os.Getenv("AOM_BASE"), Policy: repocheck.Hash(s["process/policy.json"]), Lock: repocheck.Hash(s["process/toolchain.lock.json"]), Workflow: os.Getenv("AOM_WORKFLOW"), Run: os.Getenv("AOM_RUN"), Attempt: attempt, Full: full}
	return repocheck.PreAggregateReady(records, p, b, streams)
}

func releaseSBOM(s repocheck.Snapshot) ([]byte, error) {
	// Preserve the complete approved lock inventory in a single valid JSON object.
	// Runtime module verification is performed independently before packaging.
	if moduleGraph(s) != nil {
		return nil, repocheck.ErrRejected
	}
	var lock any
	if json.Unmarshal(s["process/toolchain.lock.json"], &lock) != nil {
		return nil, repocheck.ErrInput
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
			return nil, repocheck.ErrUnknown
		}
		st, statErr := f.Stat()
		if statErr != nil || !st.Mode().IsRegular() || st.Size() <= 0 || st.Size() > 128<<20 {
			f.Close()
			return nil, repocheck.ErrInput
		}
		data, readErr := io.ReadAll(io.LimitReader(f, (128<<20)+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(data) > 128<<20 || int64(len(data)) != st.Size() {
			return nil, repocheck.ErrUnknown
		}
		info, e := buildinfo.Read(bytes.NewReader(data))
		if e != nil || info.GoVersion != "go1.26.8" || info.Main.Replace != nil ||
			(pin.module != "" && (info.Main.Path != pin.module || info.Main.Version != pin.version)) {
			return nil, repocheck.ErrRejected
		}
		result[pin.name] = map[string]any{"sha256": repocheck.Hash(data), "size": len(data), "build": info}
	}
	return result, nil
}

func reproducibleBinary(s repocheck.Snapshot) (digest string, result error) {
	dir, e := os.MkdirTemp("", "aom-repro-")
	if e != nil {
		return "", repocheck.ErrUnknown
	}
	defer func() {
		if os.RemoveAll(dir) != nil {
			result = repocheck.ErrUnknown
		}
	}()
	var first []byte
	for _, copyName := range []string{"first", "second"} {
		root := filepath.Join(dir, copyName)
		for n, b := range s {
			if !repocheck.ExportPath(n) {
				return "", repocheck.ErrRejected
			}
			target := filepath.Join(root, filepath.FromSlash(n))
			if os.MkdirAll(filepath.Dir(target), 0700) != nil || os.WriteFile(target, b, 0600) != nil {
				return "", repocheck.ErrUnknown
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
			return "", repocheck.ErrRejected
		}
		b, e := os.ReadFile(output)
		if e != nil {
			return "", repocheck.ErrUnknown
		}
		if first == nil {
			first = b
		} else if !bytes.Equal(first, b) {
			return "", repocheck.ErrRejected
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
		return result, repocheck.ErrUnknown
	}
	defer func() {
		if os.RemoveAll(dir) != nil {
			resultErr = repocheck.ErrUnknown
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, tool("go"), "mod", "download", "-json", name+"@"+version)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	raw, e := cmd.Output()
	if e != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &result) != nil || result.Error != nil {
		return result, repocheck.ErrRejected
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
		return repocheck.ErrUnknown
	}
	if e != nil {
		return repocheck.ErrRejected
	}
	for _, target := range []string{"source", "actionlint", "govulncheck"} {
		args := []string{"-format=json", "./..."}
		presence := string(packages)
		if target != "source" {
			args = []string{"-mode=binary", "-format=json", tool(target)}
			symbols, stderr, err := command(tool("go"), "tool", "nm", tool(target))
			if _, e := errout.Write(stderr); e != nil {
				return repocheck.ErrUnknown
			}
			if err != nil {
				return repocheck.ErrRejected
			}
			presence = string(symbols)
		}
		raw, stderr, err := command(tool("govulncheck"), args...)
		if _, e := out.Write(raw); e != nil {
			return repocheck.ErrUnknown
		}
		if _, e := errout.Write(stderr); e != nil {
			return repocheck.ErrUnknown
		}
		if err != nil {
			return repocheck.ErrRejected
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
			return e
		}
		for _, fact := range cleared {
			fmt.Fprintln(out, target+":"+fact)
		}
	}
	return nil
}
