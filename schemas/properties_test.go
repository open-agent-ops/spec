package schemas_test

import (
	"bytes"
	"encoding/json"
	"github.com/open-agent-ops/spec/schemas"
	"pgregory.net/rapid"
	"reflect"
	"strconv"
	"testing"
)

func TestRegistryProperties(t *testing.T) {
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	entries := r.Entries()
	rapid.Check(t, func(t *rapid.T) {
		index := rapid.IntRange(0, len(entries)-1).Draw(t, "schema")
		e := entries[index]
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		var decoded schemas.Entry
		if json.Unmarshal(b, &decoded) != nil || decoded != e {
			t.Fatal("PROP-01 round trip")
		}
		again, err := json.Marshal(decoded)
		if err != nil || !bytes.Equal(b, again) {
			t.Fatal("PROP-01 canonical")
		}
		resources := make([]schemas.Resource, 0, len(entries))
		for i := 0; i < len(entries); i++ {
			e := entries[(i+index)%len(entries)]
			b, err := r.Lookup(e.ID)
			if err != nil {
				t.Fatal(err)
			}
			resources = append(resources, schemas.Resource{Name: e.Name, Bytes: b})
		}
		if rapid.Bool().Draw(t, "reverse") {
			for i, j := 0, len(resources)-1; i < j; i, j = i+1, j-1 {
				resources[i], resources[j] = resources[j], resources[i]
			}
		}
		other, err := schemas.New(resources)
		if err != nil || other.Digest() != r.Digest() {
			t.Fatal("PROP-02 order")
		}
		resources[index] = resources[(index+1)%len(resources)]
		if _, err := schemas.New(resources); err == nil {
			t.Fatal("PROP-02 duplicate")
		}
		original, err := r.Lookup(e.ID)
		if err != nil {
			t.Fatal(err)
		}
		mutable, err := r.Lookup(e.ID)
		if err != nil {
			t.Fatal(err)
		}
		position := rapid.IntRange(0, len(mutable)-1).Draw(t, "byte")
		mutable[position] ^= byte(rapid.IntRange(1, 255).Draw(t, "mutation"))
		after, err := r.Lookup(e.ID)
		if err != nil || !bytes.Equal(original, after) {
			t.Fatal("PROP-05 copy")
		}
		families := []string{"foundation_object_ref", "foundation_semantic_object"}
		versions := []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0"}
		suffix := []string{"", "_v1_1", "_v1_2", "_v1_3"}
		family := families[rapid.IntRange(0, 1).Draw(t, "family")]
		v := rapid.IntRange(0, 3).Draw(t, "revision")
		selected, err := r.Resolve(family, versions[v])
		if err != nil || selected.Name != family+suffix[v]+".schema.json" {
			t.Fatal("PROP-04 oracle")
		}
		if _, err := r.Resolve(family, "9.0.0"); err == nil {
			t.Fatal("PROP-04 unsupported")
		}
	})
}

func TestRevisionEnumAdditivityProperty(t *testing.T) {
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	rapid.Check(t, func(t *rapid.T) {
		families := []string{"foundation_object_ref", "foundation_semantic_object"}
		family := families[rapid.IntRange(0, 1).Draw(t, "family")]
		versions := []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0"}
		i := rapid.IntRange(0, 2).Draw(t, "edge")
		var documents [2]any
		for j := 0; j < 2; j++ {
			e, err := r.Resolve(family, versions[i+j])
			if err != nil {
				t.Fatal(err)
			}
			b, err := r.Lookup(e.ID)
			if err != nil || json.Unmarshal(b, &documents[j]) != nil {
				t.Fatal("schema")
			}
		}
		left, right := enumSets(documents[0], ""), enumSets(documents[1], "")
		if len(left) == 0 {
			t.Fatal("no enum oracle")
		}
		for path, old := range left {
			new, ok := right[path]
			if !ok {
				t.Fatal("enum missing", path)
			}
			for _, v := range old {
				found := false
				for _, candidate := range new {
					if reflect.DeepEqual(v, candidate) {
						found = true
					}
				}
				if !found {
					t.Fatal("PROP-03 enum removed", path)
				}
			}
		}
	})
}

func enumSets(v any, path string) map[string][]any {
	out := map[string][]any{}
	if items, ok := v.([]any); ok {
		for i, item := range items {
			for p, set := range enumSets(item, path+"/"+strconv.Itoa(i)) {
				out[p] = set
			}
		}
	}
	if obj, ok := v.(map[string]any); ok {
		for key, x := range obj {
			if key == "enum" {
				if set, ok := x.([]any); ok {
					out[path] = set
				}
			} else {
				for p, set := range enumSets(x, path+"/"+key) {
					out[p] = set
				}
			}
		}
	}
	return out
}
