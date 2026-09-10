package repocheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const Repository = "github.com/open-agent-ops/spec"

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var versionPattern = regexp.MustCompile(`^v(0|1)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func Commit(s string) bool  { return commitPattern.MatchString(s) }
func Digest(s string) bool  { return digestPattern.MatchString(s) }
func Version(s string) bool { return len(s) <= 64 && versionPattern.MatchString(s) }
func Hash(b []byte) string  { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func Canonical(b []byte) ([]byte, error) {
	v, e := JSON(b)
	if e != nil {
		return nil, e
	}
	out, e := json.Marshal(v)
	if e != nil {
		return nil, ErrInput
	}
	return out, nil
}

type denyLoader struct{}

func (denyLoader) Load(string) (any, error) { return nil, ErrInput }

// ValidateRecord compiles only the supplied closed process schema, never URLs.
// Schema bytes must come from the trusted candidate policy snapshot.
func ValidateRecord(schema, data []byte) error {
	s, e := JSON(schema)
	if e != nil {
		return e
	}
	v, e := JSON(data)
	if e != nil {
		return e
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(denyLoader{})
	c.AssertFormat()
	if c.AddResource("https://open-agent-ops.org/process/input", s) != nil {
		return ErrInput
	}
	compiled, e := c.Compile("https://open-agent-ops.org/process/input")
	if e != nil {
		return ErrInput
	}
	if compiled.Validate(v) != nil {
		return ErrInput
	}
	return nil
}

type Asset struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type Submission struct {
	Schema        string   `json:"schema"`
	Kind          string   `json:"kind"`
	Head          string   `json:"head"`
	Base          string   `json:"base"`
	Paths         []string `json:"paths"`
	Actor         string   `json:"actor"`
	Link          string   `json:"link"`
	Revision      string   `json:"revision"`
	Description   string   `json:"description"`
	Compatibility string   `json:"compatibility"`
	Bilingual     string   `json:"bilingual"`
	LicenseAck    bool     `json:"license_ack"`
}
type Review struct {
	Schema        string    `json:"schema"`
	Reviewer      string    `json:"reviewer"`
	Role          string    `json:"role"`
	Candidate     string    `json:"candidate"`
	Decision      string    `json:"decision"`
	Timestamp     time.Time `json:"timestamp"`
	SourceReceipt string    `json:"source_receipt"`
}
type Gate struct {
	Schema     string `json:"schema"`
	Repository string `json:"repository"`
	Candidate  string `json:"candidate"`
	Base       string `json:"base"`
	Policy     string `json:"policy"`
	Lock       string `json:"lock"`
	Workflow   string `json:"workflow"`
	Run        string `json:"run"`
	Job        string `json:"job"`
	Attempt    int    `json:"attempt"`
	Result     string `json:"result"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
}
type Proposal struct {
	Schema     string    `json:"schema"`
	Repository string    `json:"repository"`
	Candidate  string    `json:"candidate"`
	Version    string    `json:"version"`
	Tree       string    `json:"tree"`
	Inventory  string    `json:"inventory"`
	Assets     []Asset   `json:"assets"`
	SBOM       string    `json:"sbom"`
	Workflow   string    `json:"workflow"`
	Run        string    `json:"run"`
	Attempt    int       `json:"attempt"`
	Lock       string    `json:"lock"`
	Policy     string    `json:"policy"`
	CheckedAt  time.Time `json:"checked_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}
type Receipt struct {
	Schema      string   `json:"schema"`
	Repository  string   `json:"repository"`
	Candidate   string   `json:"candidate"`
	Version     string   `json:"version"`
	Proposal    string   `json:"proposal"`
	Canonical   string   `json:"canonical"`
	Mirror      string   `json:"mirror"`
	Assets      []Asset  `json:"assets"`
	ObservedIDs []string `json:"observed_ids"`
	FixtureOnly bool     `json:"fixture_only"`
}

func ValidateProposal(p Proposal, now time.Time) error {
	if p.Schema != "aom04a.release-proposal.v1" || p.Repository != Repository || !Commit(p.Candidate) || p.Candidate != p.Workflow || !Version(p.Version) || !Commit(p.Tree) || !Digest(p.Inventory) || !Digest(p.SBOM) || !Digest(p.Lock) || !Digest(p.Policy) || p.Run == "" || p.Attempt != 1 {
		return ErrInput
	}
	if p.CheckedAt.IsZero() || p.CheckedAt.After(now) || !now.Before(p.ExpiresAt) || p.ExpiresAt.After(p.CheckedAt.Add(24*time.Hour)) || !p.ExpiresAt.After(p.CheckedAt) {
		return ErrRejected
	}
	if len(p.Assets) == 0 || len(p.Assets) > 16 {
		return ErrInput
	}
	total := int64(0)
	seen := map[string]bool{}
	for _, a := range p.Assets {
		if !SafePath(a.Name) || !Digest(a.SHA256) || a.Size <= 0 || a.Size > 64<<20 || seen[a.Name] {
			return ErrInput
		}
		seen[a.Name] = true
		total += a.Size
	}
	if total > 256<<20 {
		return ErrInput
	}
	return nil
}
func ProposalDigest(p Proposal) (string, error) {
	copyP := p
	copyP.Assets = append([]Asset(nil), p.Assets...)
	sort.Slice(copyP.Assets, func(i, j int) bool { return copyP.Assets[i].Name < copyP.Assets[j].Name })
	b, e := json.Marshal(copyP)
	if e != nil {
		return "", ErrInput
	}
	canonical, e := Canonical(b)
	if e != nil {
		return "", e
	}
	return Hash(canonical), nil
}
