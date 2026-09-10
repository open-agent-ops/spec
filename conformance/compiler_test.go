package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/open-agent-ops/spec/schemas"
	"os"
	"strings"
	"testing"
)

func TestAllSchemasCompileOffline(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if len(v.compiled) != 61 {
		t.Fatal("incomplete compilation")
	}
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"https://example.org/x", "file:///etc/passwd", "../x"} {
		if _, err := (localLoader{r}).Load(id); err == nil {
			t.Fatal("external loader accepted")
		}
		if v.Validate(id, []byte(`{}`)).Accepted {
			t.Fatal("unknown schema accepted")
		}
	}
	count := 0
	if checkRefs(map[string]any{"$ref": "../schemas/foundation_object_ref.schema.json"}, r.Entries()[0].ID, r, &count) == nil {
		t.Fatal("traversal accepted")
	}
}

type fixture struct {
	Path     string `json:"path"`
	Schema   string `json:"schema"`
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason"`
}

func TestReasonSpecificFixtures(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/index.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []fixture
	if json.Unmarshal(b, &fixtures) != nil || len(fixtures) != 28 {
		t.Fatal("fixture inventory")
	}
	seen := map[string]bool{}
	valid, invalid := 0, 0
	for _, f := range fixtures {
		if seen[f.Path] || !ValidPath(f.Path) {
			t.Fatal("fixture path")
		}
		seen[f.Path] = true
		b, err := os.ReadFile("testdata/" + f.Path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Decode(b); err != nil {
			t.Fatal("negative must parse", f.Path)
		}
		r := v.Validate(f.Schema, b)
		if r.Accepted != f.Accepted {
			t.Fatalf("%s accepted=%v reasons=%v", f.Path, r.Accepted, r.Reasons)
		}
		if !f.Accepted && !contains(r.Reasons, f.Reason) {
			t.Fatalf("%s wrong reason %v expected %s", f.Path, r.Reasons, f.Reason)
		}
		if !strings.HasPrefix(f.Path, "run_request/") {
			if f.Accepted {
				valid++
			} else {
				invalid++
			}
		}
	}
	if valid != 6 || invalid != 18 {
		t.Fatal("core fixture set incomplete")
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestDeliveredFixturePinsIntegration(t *testing.T) {
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/index.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []fixture
	if json.Unmarshal(b, &fixtures) != nil {
		t.Fatal("index")
	}
	checked := 0
	for _, f := range fixtures {
		if !f.Accepted {
			continue
		}
		b, err := os.ReadFile("testdata/" + f.Path)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if json.Unmarshal(b, &doc) != nil {
			t.Fatal("fixture")
		}
		pin, ok := doc["document_schema"].(map[string]any)
		if !ok {
			continue
		}
		id, ok := pin["id"].(string)
		if !ok {
			t.Fatal("pin id")
		}
		resource, err := r.Lookup(id)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(resource)
		if pin["sha256"] != hex.EncodeToString(sum[:]) {
			t.Fatal("stale delivered pin", f.Path)
		}
		checked++
	}
	if checked != 5 {
		t.Fatal("pin coverage")
	}
}
