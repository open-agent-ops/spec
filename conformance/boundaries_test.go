package conformance

import (
	"encoding/json"
	"fmt"
	"github.com/open-agent-ops/spec/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"strings"
	"testing"
)

func TestProductionJSONBoundaries(t *testing.T) {
	for _, depth := range []int{32, 33} {
		s := strings.Repeat("[", depth-1) + "0" + strings.Repeat("]", depth-1)
		_, err := Decode([]byte(s))
		if (err == nil) != (depth == 32) {
			t.Fatal("depth boundary")
		}
	}
	for _, n := range []int{1024, 1025} {
		object := map[string]int{}
		for i := 0; i < n; i++ {
			object[fmt.Sprint(i)] = i
		}
		b, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Decode(b)
		if (err == nil) != (n == 1024) {
			t.Fatal("member boundary")
		}
	}
	for _, n := range []int{16384, 16385} {
		b, err := json.Marshal(strings.Repeat("x", n))
		if err != nil {
			t.Fatal(err)
		}
		_, err = Decode(b)
		if (err == nil) != (n == 16384) {
			t.Fatal("string boundary")
		}
	}
	for _, last := range []int{1019, 1020} {
		v := [][]int{make([]int, 1024), make([]int, 1024), make([]int, 1024), make([]int, last)}
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Decode(b)
		if (err == nil) != (last == 1019) {
			t.Fatal("node boundary")
		}
	}
	r, err := schemas.Open()
	if err != nil {
		t.Fatal(err)
	}
	id := r.Entries()[0].ID
	for _, n := range []int{1024, 1025} {
		refs := []any{}
		for i := 0; i < n; i++ {
			refs = append(refs, map[string]any{"$ref": id})
		}
		count := 0
		err := checkRefs(refs, id, r, &count)
		if (err == nil) != (n == 1024) {
			t.Fatal("reference boundary")
		}
	}
}

func TestFindingBoundary(t *testing.T) {
	for _, n := range []int{1024, 1025} {
		root := &jsonschema.ValidationError{ErrorKind: &kind.Type{}}
		for i := 1; i < n; i++ {
			root.Causes = append(root.Causes, &jsonschema.ValidationError{ErrorKind: &kind.Type{}})
		}
		reasons := validationReasons(root, 1024)
		if containsReason(reasons, "findings_overflow") != (n == 1025) {
			t.Fatal("finding boundary")
		}
	}
}
