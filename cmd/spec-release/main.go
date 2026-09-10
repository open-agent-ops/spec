package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/open-agent-ops/spec/releaseops"
	"github.com/open-agent-ops/spec/repocheck"
)

func load(root *os.Root, name string, limit int64) ([]byte, error) {
	if !repocheck.SafePath(name) {
		return nil, repocheck.ErrInput
	}
	st, e := root.Lstat(name)
	if e != nil || !st.Mode().IsRegular() || st.Size() > limit {
		return nil, repocheck.ErrInput
	}
	f, e := root.Open(name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil || int64(len(b)) > limit {
		return nil, repocheck.ErrInput
	}
	return b, nil
}
func run() error {
	if len(os.Args) != 2 || (os.Args[1] != "publish" && os.Args[1] != "mirror") {
		return repocheck.ErrInput
	}
	root, e := os.OpenRoot(".aom-release")
	if e != nil {
		return e
	}
	defer root.Close()
	raw, e := load(root, "proposal.json", repocheck.MaxInput)
	if e != nil {
		return e
	}
	var p repocheck.Proposal
	if repocheck.Decode(raw, &p) != nil || repocheck.ValidateProposal(p, time.Now()) != nil {
		return repocheck.ErrInput
	}
	schema, e := os.ReadFile("process/schemas/release-proposal.schema.json")
	if e != nil || repocheck.ValidateRecord(schema, raw) != nil {
		return repocheck.ErrInput
	}
	lock, e := os.ReadFile("process/toolchain.lock.json")
	if e != nil || repocheck.Hash(lock) != p.Lock {
		return repocheck.ErrRejected
	}
	policy, e := os.ReadFile("process/policy.json")
	if e != nil || repocheck.Hash(policy) != p.Policy {
		return repocheck.ErrRejected
	}
	assets := map[string][]byte{}
	for _, a := range p.Assets {
		b, e := load(root, a.Name, 64<<20)
		if e != nil || int64(len(b)) != a.Size || repocheck.Hash(b) != a.SHA256 {
			return repocheck.ErrRejected
		}
		assets[a.Name] = b
	}
	var trustedPolicy repocheck.Policy
	gateSchema, schemaErr := os.ReadFile("process/schemas/gate-record.schema.json")
	if repocheck.Decode(policy, &trustedPolicy) != nil || schemaErr != nil || repocheck.ReleaseEvidence(assets["evidence.tar"], trustedPolicy, p, os.Getenv("AOM_BASE"), gateSchema) != nil {
		return repocheck.ErrRejected
	}
	checkout, openErr := os.OpenRoot(".")
	if openErr != nil {
		return repocheck.ErrUnknown
	}
	defer checkout.Close()
	trusted := repocheck.Snapshot{}
	for _, f := range trustedPolicy.Files {
		limit := int64(4 << 20)
		if strings.HasSuffix(f.Path, ".pdf") && f.ProtectedHash != "" {
			limit = 32 << 20
		}
		b, err := load(checkout, f.Path, limit)
		if err != nil {
			return repocheck.ErrRejected
		}
		trusted[f.Path] = b
	}
	if repocheck.ReleaseSource(assets["source.tar"], trusted, trustedPolicy, p) != nil || repocheck.Hash(assets["sbom.json"]) != p.SBOM {
		return repocheck.ErrRejected
	}
	if _, err := repocheck.JSON(assets["sbom.json"]); err != nil {
		return repocheck.ErrRejected
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var receipt repocheck.Receipt
	if os.Args[1] == "publish" {
		receipt, e = releaseops.PublishApproved(ctx, p, assets)
	} else {
		receipt, e = releaseops.MirrorApproved(ctx, p)
	}
	if receipt.Schema == "" {
		if e != nil {
			return e
		}
		return repocheck.ErrUnknown
	}
	out, marshalErr := json.Marshal(receipt)
	if marshalErr != nil {
		return repocheck.ErrUnknown
	}
	receiptSchema, schemaErr := os.ReadFile("process/schemas/publication-receipt.schema.json")
	if schemaErr != nil || repocheck.ValidateRecord(receiptSchema, out) != nil {
		return repocheck.ErrUnknown
	}
	if _, writeErr := os.Stdout.Write(append(out, '\n')); writeErr != nil {
		return repocheck.ErrUnknown
	}
	return e
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "release_rejected_or_incomplete")
		code := 4
		if errors.Is(e, repocheck.ErrInput) {
			code = 2
		} else if errors.Is(e, repocheck.ErrRejected) || errors.Is(e, releaseops.ErrAuthority) || errors.Is(e, releaseops.ErrConflict) {
			code = 3
		}
		os.Exit(code)
	}
}
