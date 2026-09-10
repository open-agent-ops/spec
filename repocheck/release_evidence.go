package repocheck

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
)

// ReleaseEvidence independently verifies all same-run gates and their complete
// streams immediately before entering the publication capability boundary.
// No archive member is extracted to the filesystem.
func ReleaseEvidence(raw []byte, p Policy, proposal Proposal, base string, schema []byte) error {
	if len(raw) == 0 || len(raw) > 64<<20 {
		return Invalidf("evidence archive is empty or over 64 MiB")
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	files := map[string][]byte{}
	for {
		h, e := tr.Next()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			return Invalidf("evidence archive: %v", e)
		}
		if h.Typeflag != tar.TypeReg || !SafePath(h.Name) || h.Size < 0 || h.Size > 8<<20 || len(files) >= 21 {
			return Invalidf("evidence archive member rejected (type, path, size or count): %s", h.Name)
		}
		if _, exists := files[h.Name]; exists {
			return Invalidf("evidence archive member duplicated: %s", h.Name)
		}
		b, e := io.ReadAll(io.LimitReader(tr, (8<<20)+1))
		if e != nil || int64(len(b)) != h.Size {
			return Invalidf("evidence archive member truncated: %s", h.Name)
		}
		files[h.Name] = b
	}
	if len(files) != 21 {
		return Rejectedf("evidence archive holds %d members, want 21", len(files))
	}
	names := []string{"policy", "conformance", "docs", "supply-chain", "aggregate", "heavy", "reproducibility"}
	records := make([]Gate, 0, 7)
	streams := map[string][]byte{}
	for _, name := range names {
		b, ok := files[name+".json"]
		if !ok {
			return Rejectedf("evidence: %s.json missing", name)
		}
		if e := ValidateRecord(schema, b); e != nil {
			return Rejectedf("evidence: %s.json: %v", name, e)
		}
		var g Gate
		if Decode(b, &g) != nil || g.Job != name {
			return Rejectedf("evidence: %s.json does not decode as a %s gate record", name, name)
		}
		records = append(records, g)
		for _, suffix := range []string{".stdout", ".stderr"} {
			b, ok := files[name+suffix]
			if !ok {
				return Rejectedf("evidence: %s%s missing", name, suffix)
			}
			streams[name+suffix] = b
		}
	}
	return EvidenceReady(records, p, EvidenceBinding{Candidate: proposal.Candidate, Base: base, Policy: proposal.Policy, Lock: proposal.Lock, Workflow: proposal.Workflow, Run: proposal.Run, Attempt: proposal.Attempt, Full: true}, streams)
}

// ReleaseSource compares every archive byte to the reviewed checkout. A fresh
// digest supplied by the archive itself cannot authorize different source.
func ReleaseSource(raw []byte, trusted Snapshot, p Policy, proposal Proposal) error {
	if len(raw) == 0 || len(raw) > 64<<20 {
		return Rejectedf("source archive is empty or over 64 MiB")
	}
	if e := Composition(trusted, p); e != nil {
		return Rejectedf("source: trusted checkout: %v", e)
	}
	if Hash(trusted["composition-manifest.json"]) != proposal.Inventory {
		return Rejectedf("source: composition-manifest.json digest differs from proposal inventory")
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	seen := map[string]bool{}
	for {
		h, e := tr.Next()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			return Invalidf("source archive: %v", e)
		}
		if h.Typeflag != tar.TypeReg || !ExportPath(h.Name) || h.Size < 0 || h.Size > 32<<20 || seen[h.Name] || len(seen) >= 256 {
			return Invalidf("source archive member rejected (type, path, size, duplicate or count): %s", h.Name)
		}
		want, ok := trusted[h.Name]
		if !ok {
			return Rejectedf("source archive member not in trusted checkout: %s", h.Name)
		}
		if h.Size != int64(len(want)) {
			return Rejectedf("source archive member size differs: %s", h.Name)
		}
		data, e := io.ReadAll(io.LimitReader(tr, (32<<20)+1))
		if e != nil || !bytes.Equal(data, want) {
			return Rejectedf("source archive member bytes differ: %s", h.Name)
		}
		seen[h.Name] = true
	}
	if len(seen) != len(trusted) {
		return Rejectedf("source archive holds %d members, trusted checkout has %d", len(seen), len(trusted))
	}
	return nil
}
