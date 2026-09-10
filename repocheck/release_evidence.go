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
		return ErrInput
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	files := map[string][]byte{}
	for {
		h, e := tr.Next()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			return ErrInput
		}
		if h.Typeflag != tar.TypeReg || !SafePath(h.Name) || h.Size < 0 || h.Size > 8<<20 || len(files) >= 21 {
			return ErrInput
		}
		if _, exists := files[h.Name]; exists {
			return ErrInput
		}
		b, e := io.ReadAll(io.LimitReader(tr, (8<<20)+1))
		if e != nil || int64(len(b)) != h.Size {
			return ErrInput
		}
		files[h.Name] = b
	}
	if len(files) != 21 {
		return ErrRejected
	}
	names := []string{"policy", "conformance", "docs", "supply-chain", "aggregate", "heavy", "reproducibility"}
	records := make([]Gate, 0, 7)
	streams := map[string][]byte{}
	for _, name := range names {
		b, ok := files[name+".json"]
		if !ok || ValidateRecord(schema, b) != nil {
			return ErrRejected
		}
		var g Gate
		if Decode(b, &g) != nil || g.Job != name {
			return ErrRejected
		}
		records = append(records, g)
		for _, suffix := range []string{".stdout", ".stderr"} {
			b, ok := files[name+suffix]
			if !ok {
				return ErrRejected
			}
			streams[name+suffix] = b
		}
	}
	return EvidenceReady(records, p, EvidenceBinding{Candidate: proposal.Candidate, Base: base, Policy: proposal.Policy, Lock: proposal.Lock, Workflow: proposal.Workflow, Run: proposal.Run, Attempt: proposal.Attempt, Full: true}, streams)
}

// ReleaseSource compares every archive byte to the reviewed checkout. A fresh
// digest supplied by the archive itself cannot authorize different source.
func ReleaseSource(raw []byte, trusted Snapshot, p Policy, proposal Proposal) error {
	if len(raw) == 0 || len(raw) > 64<<20 || Composition(trusted, p) != nil || Hash(trusted["composition-manifest.json"]) != proposal.Inventory {
		return ErrRejected
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	seen := map[string]bool{}
	for {
		h, e := tr.Next()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil || h.Typeflag != tar.TypeReg || !ExportPath(h.Name) || h.Size < 0 || h.Size > 32<<20 || seen[h.Name] || len(seen) >= 256 {
			return ErrInput
		}
		want, ok := trusted[h.Name]
		if !ok || h.Size != int64(len(want)) {
			return ErrRejected
		}
		data, e := io.ReadAll(io.LimitReader(tr, (32<<20)+1))
		if e != nil || !bytes.Equal(data, want) {
			return ErrRejected
		}
		seen[h.Name] = true
	}
	if len(seen) != len(trusted) {
		return ErrRejected
	}
	return nil
}
