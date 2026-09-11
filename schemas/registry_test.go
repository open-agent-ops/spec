package schemas_test

import (
	"bytes"
	"encoding/json"
	"github.com/open-agent-ops/spec/schemas"
	"testing"
)

func TestRegistryMembershipAndCopies(t *testing.T) {
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	entries := r.Entries()
	if len(entries) != 61 || len(r.Digest()) != 64 {
		t.Fatal("incomplete registry")
	}
	resources := make([]schemas.Resource, 0, len(entries))
	for _, e := range entries {
		b, err := r.Lookup(e.ID)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, []byte("https://gitinsky.com/schemas/")) || bytes.Contains(b, []byte(`"gitinsky.`)) {
			t.Fatal("legacy identity", e.Name)
		}
		resources = append(resources, schemas.Resource{Name: e.Name, Bytes: b})
	}
	if _, err := schemas.New(resources[:60]); err == nil {
		t.Fatal("missing accepted")
	}
	if _, err := schemas.New(append(append([]schemas.Resource(nil), resources...), resources[0])); err == nil {
		t.Fatal("extra accepted")
	}
	original := resources[0]
	resources[0] = schemas.Resource{Name: "internal.schema.json", Bytes: original.Bytes}
	if _, err := schemas.New(resources); err == nil {
		t.Fatal("internal entry accepted")
	}
	resources[0] = schemas.Resource{Name: original.Name, Bytes: append(append([]byte(nil), original.Bytes...), '\n')}
	if _, err := schemas.New(resources); err == nil {
		t.Fatal("byte drift accepted")
	}
	resources[0] = original
	resources[0] = resources[1]
	if _, err := schemas.New(resources); err == nil {
		t.Fatal("duplicate accepted")
	}
	before, _ := r.Lookup(entries[0].ID)
	copyBytes, _ := r.Lookup(entries[0].ID)
	copyBytes[0] ^= 1
	entries[0].ID = "changed"
	after, _ := r.Lookup(r.Entries()[0].ID)
	if !bytes.Equal(before, after) {
		t.Fatal("shared state")
	}
	for _, id := range []string{"", "file:///etc/passwd", "https://example.org/schema", "../schema", "gitinsky.object_ref"} {
		if _, err := r.Lookup(id); err == nil {
			t.Fatal("unknown accepted")
		}
	}
}

func TestExactLiveDispatch(t *testing.T) {
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	versions := []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0"}
	suffixes := []string{"", "_v1_1", "_v1_2", "_v1_3"}
	for _, family := range []string{"foundation_object_ref", "foundation_semantic_object"} {
		for i, version := range versions {
			e, err := r.Resolve(family, version)
			if err != nil || e.Name != family+suffixes[i]+".schema.json" {
				t.Fatal(family, version, err)
			}
		}
		for _, version := range []string{"", "v1.0", "1.4.0", "0.4.0"} {
			if _, err := r.Resolve(family, version); err == nil {
				t.Fatal("unsupported version")
			}
		}
	}
}

// Each resolved semantic object revision must declare its own format_version;
// a family member that only accepts 1.0.0 makes version dispatch meaningless.
func TestResolvedSemanticObjectDeclaresItsVersion(t *testing.T) {
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0"} {
		e, err := r.Resolve("foundation_semantic_object", version)
		if err != nil {
			t.Fatal(version, err)
		}
		b, err := r.Lookup(e.ID)
		if err != nil {
			t.Fatal(version, err)
		}
		var doc struct {
			Properties struct {
				FormatVersion struct{ Const string } `json:"format_version"`
			}
		}
		if json.Unmarshal(b, &doc) != nil {
			t.Fatal(version, "decode")
		}
		if doc.Properties.FormatVersion.Const != version {
			t.Fatalf("%s resolves to %s declaring format_version %q", version, e.Name, doc.Properties.FormatVersion.Const)
		}
	}
}
