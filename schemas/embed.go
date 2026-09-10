package schemas

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

//go:embed *.schema.json
var files embed.FS

// Entry identifies one approved schema. Returned entries contain no shared state.
type Entry struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

// Resource supplies bytes to a closed registry; it grants no file or URL access.
type Resource struct {
	Name  string
	Bytes []byte
}

// Registry holds the exact public schema snapshot.
type Registry struct {
	entries []Entry
	bytes   map[string][]byte
	digest  string
}

var ErrRegistry = errors.New("schema_registry_invalid")
var ErrUnknown = errors.New("schema_identity_unknown")

// Open checks the embedded resources against the approved inventory.
func Open() (*Registry, error) {
	actual, err := files.ReadDir(".")
	if err != nil || len(actual) != len(approved) {
		return nil, ErrRegistry
	}
	resources := make([]Resource, 0, len(approved))
	for _, entry := range approved {
		b, err := files.ReadFile(entry.Name)
		if err != nil {
			return nil, ErrRegistry
		}
		resources = append(resources, Resource{entry.Name, b})
	}
	return New(resources)
}

// New validates exact membership and bytes, and takes defensive copies.
func New(resources []Resource) (*Registry, error) {
	if len(resources) != len(approved) {
		return nil, ErrRegistry
	}
	expected := make(map[string]Entry, len(approved))
	for _, e := range approved {
		expected[e.Name] = e
	}
	r := &Registry{bytes: make(map[string][]byte, len(approved))}
	for _, resource := range resources {
		e, ok := expected[resource.Name]
		if !ok || len(resource.Bytes) > 4194304 {
			return nil, ErrRegistry
		}
		if _, duplicate := r.bytes[e.ID]; duplicate {
			return nil, ErrRegistry
		}
		sum := sha256.Sum256(resource.Bytes)
		if hex.EncodeToString(sum[:]) != e.SHA256 {
			return nil, ErrRegistry
		}
		var doc struct {
			ID string `json:"$id"`
		}
		if json.Unmarshal(resource.Bytes, &doc) != nil || doc.ID != e.ID {
			return nil, ErrRegistry
		}
		r.entries = append(r.entries, e)
		r.bytes[e.ID] = append([]byte(nil), resource.Bytes...)
	}
	sort.Slice(r.entries, func(i, j int) bool { return r.entries[i].Name < r.entries[j].Name })
	b, err := json.Marshal(r.entries)
	if err != nil {
		return nil, ErrRegistry
	}
	sum := sha256.Sum256(b)
	r.digest = hex.EncodeToString(sum[:])
	return r, nil
}

func (r *Registry) Entries() []Entry {
	if r == nil {
		return nil
	}
	return append([]Entry(nil), r.entries...)
}

func (r *Registry) Digest() string {
	if r == nil {
		return ""
	}
	return r.digest
}

// Lookup resolves an exact public resource ID without I/O or alias expansion.
func (r *Registry) Lookup(id string) ([]byte, error) {
	if r == nil {
		return nil, ErrUnknown
	}
	b, ok := r.bytes[id]
	if !ok {
		return nil, ErrUnknown
	}
	return append([]byte(nil), b...), nil
}
