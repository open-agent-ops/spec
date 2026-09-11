// Package repocheck evaluates repository process data without I/O or authority.
package repocheck

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const MaxInput = 1 << 20
const MaxDepth = 32
const MaxNodes = 20000
const MaxCollection = 1000
const MaxString = 4096

var ErrInput = errors.New("invalid_input")
var ErrRejected = errors.New("rejected_incomplete")
var ErrUnknown = errors.New("dependency_unknown")

// Invalidf, Rejectedf and Unknownf attach a fixed diagnostic label to the
// matching sentinel. Callers keep classifying with errors.Is; operators get a
// reason. Labels name files, keys and counts, never candidate content bytes.
func Invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInput, fmt.Sprintf(format, args...))
}
func Rejectedf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrRejected, fmt.Sprintf(format, args...))
}
func Unknownf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrUnknown, fmt.Sprintf(format, args...))
}

// Decode rejects ambiguous JSON before closed typed decoding. It never trusts
// encoding/json's last-key-wins behavior, unknown fields or replacement UTF-8.
func Decode(data []byte, target any) error {
	if _, err := JSON(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return ErrInput
	}
	return nil
}

func JSON(data []byte) (any, error) {
	if len(data) == 0 || len(data) > MaxInput || !utf8.Valid(data) {
		return nil, ErrInput
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	nodes := 0
	v, e := jsonValue(d, 0, &nodes)
	if e != nil {
		return nil, e
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, ErrInput
	}
	return v, nil
}
func jsonValue(d *json.Decoder, depth int, nodes *int) (any, error) {
	*nodes++
	if depth > MaxDepth || *nodes > MaxNodes {
		return nil, ErrInput
	}
	t, e := d.Token()
	if e != nil {
		return nil, ErrInput
	}
	switch v := t.(type) {
	case string:
		if len(v) > MaxString {
			return nil, ErrInput
		}
		return v, nil
	case json.Delim:
		switch v {
		case '{':
			m := map[string]any{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return nil, ErrInput
				}
				s, ok := k.(string)
				if !ok || len(s) > MaxString {
					return nil, ErrInput
				}
				if _, ok := m[s]; ok || len(m) >= MaxCollection {
					return nil, ErrInput
				}
				val, e := jsonValue(d, depth+1, nodes)
				if e != nil {
					return nil, e
				}
				m[s] = val
			}
			if t, e := d.Token(); e != nil || t != json.Delim('}') {
				return nil, ErrInput
			}
			return m, nil
		case '[':
			a := []any{}
			for d.More() {
				if len(a) >= MaxCollection {
					return nil, ErrInput
				}
				v, e := jsonValue(d, depth+1, nodes)
				if e != nil {
					return nil, e
				}
				a = append(a, v)
			}
			if t, e := d.Token(); e != nil || t != json.Delim(']') {
				return nil, ErrInput
			}
			return a, nil
		}
		return nil, ErrInput
	default:
		return t, nil
	}
}

func SafePath(s string) bool {
	if s == "" || len(s) > MaxString || !utf8.ValidString(s) || path.Clean(s) != s || strings.HasPrefix(s, "/") {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == ".." || part == "." || part == "" {
			return false
		}
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("_./-", c)) {
			return false
		}
	}
	return true
}

// YAML retains scalar types and rejects aliases, merges and duplicate keys.
func YAML(data []byte) (*yaml.Node, error) {
	if len(data) == 0 || len(data) > MaxInput || !utf8.Valid(data) {
		return nil, ErrInput
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	var n, extra yaml.Node
	if d.Decode(&n) != nil || d.Decode(&extra) != io.EOF {
		return nil, ErrInput
	}
	count := 0
	if yamlNode(&n, 0, &count) != nil {
		return nil, ErrInput
	}
	return &n, nil
}
func yamlNode(n *yaml.Node, depth int, count *int) error {
	*count++
	if depth > MaxDepth || *count > MaxNodes || n.Anchor != "" || n.Alias != nil || n.Kind == yaml.AliasNode || len(n.Value) > MaxString {
		return ErrInput
	}
	switch n.Tag {
	case "", "!!map", "!!seq", "!!str", "!!bool", "!!int", "!!float", "!!null":
	default:
		return ErrInput
	}
	limit := MaxCollection
	if n.Kind == yaml.MappingNode {
		limit *= 2
	}
	if len(n.Content) > limit {
		return ErrInput
	}
	if n.Kind == yaml.MappingNode {
		if len(n.Content)%2 != 0 {
			return ErrInput
		}
		seen := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" || k.Value == "<<" || seen[k.Value] {
				return ErrInput
			}
			seen[k.Value] = true
		}
	}
	for _, c := range n.Content {
		if yamlNode(c, depth+1, count) != nil {
			return ErrInput
		}
	}
	return nil
}
