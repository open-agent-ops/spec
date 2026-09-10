package conformance

import (
	"bytes"
	"encoding/json"
	"pgregory.net/rapid"
	"testing"
)

func TestResultRoundTripProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		result := Result{Registry: rapid.StringMatching("[a-f0-9]{64}").Draw(t, "registry"), Schema: "https://agent-ops.ru/schemas/foundation_object_ref.schema.json", Accepted: false, Reasons: []string{"schema_unknown"}}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		var decoded Result
		if json.Unmarshal(encoded, &decoded) != nil {
			t.Fatal("decode")
		}
		again, err := json.Marshal(decoded)
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatal("PROP-01 result")
		}
	})
}

func TestMalformedInputNeverImprovesProperty(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	rapid.Check(t, func(t *rapid.T) {
		text := rapid.StringMatching("[a-z0-9]{0,64}").Draw(t, "payload")
		b, _ := json.Marshal(map[string]any{"unknown": text})
		for _, input := range [][]byte{b, append(append([]byte(nil), b...), []byte(` {"private":true}`)...)} {
			if v.Validate("https://agent-ops.ru/schemas/foundation_object_ref.schema.json", input).Accepted {
				t.Fatal("PROP-07 invalid improved")
			}
		}
	})
}
