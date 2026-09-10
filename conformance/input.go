// Package conformance checks the declared public schema contracts offline.
// Successful schema validation does not authenticate data or authorize actions.
package conformance

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

const MaxBytes = 4194304

var ErrInput = errors.New("input_invalid_or_over_bound")

type profile struct{ depth, nodes, collection, text int }

var inputProfile = profile{32, 4096, 1024, 16384}

// Decode rejects malformed, duplicate-key, trailing and over-bound JSON.
func Decode(data []byte) (any, error) { return decode(data, inputProfile) }

func decode(data []byte, limits profile) (any, error) {
	if len(data) == 0 || len(data) > MaxBytes || !utf8.Valid(data) {
		return nil, ErrInput
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	nodes := 0
	v, err := value(d, 1, &nodes, limits)
	if err != nil {
		return nil, ErrInput
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, ErrInput
	}
	return v, nil
}

func value(d *json.Decoder, depth int, nodes *int, p profile) (any, error) {
	*nodes++
	if depth > p.depth || *nodes > p.nodes {
		return nil, ErrInput
	}
	tok, err := d.Token()
	if err != nil {
		return nil, ErrInput
	}
	switch t := tok.(type) {
	case string:
		if len(t) > p.text {
			return nil, ErrInput
		}
		return t, nil
	case json.Delim:
		switch t {
		case '{':
			out := map[string]any{}
			for d.More() {
				if len(out) >= p.collection {
					return nil, ErrInput
				}
				key, err := d.Token()
				if err != nil {
					return nil, ErrInput
				}
				k, ok := key.(string)
				if !ok || len(k) > p.text {
					return nil, ErrInput
				}
				if _, exists := out[k]; exists {
					return nil, ErrInput
				}
				v, err := value(d, depth+1, nodes, p)
				if err != nil {
					return nil, err
				}
				out[k] = v
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return nil, ErrInput
			}
			return out, nil
		case '[':
			out := []any{}
			for d.More() {
				if len(out) >= p.collection {
					return nil, ErrInput
				}
				v, err := value(d, depth+1, nodes, p)
				if err != nil {
					return nil, err
				}
				out = append(out, v)
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return nil, ErrInput
			}
			return out, nil
		default:
			return nil, ErrInput
		}
	default:
		return t, nil
	}
}

// Read closes the supplied reader on success and failure.
func Read(r io.ReadCloser) (data []byte, err error) {
	if r == nil {
		return nil, ErrInput
	}
	defer func() {
		if r.Close() != nil {
			data = nil
			err = ErrInput
		}
	}()
	data, err = io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil || len(data) > MaxBytes {
		return nil, ErrInput
	}
	return data, nil
}

// ValidPath accepts a bounded normalized relative path, never a host path.
func ValidPath(s string) bool {
	if s == "" || len(s) > 256 || !utf8.ValidString(s) || strings.ContainsAny(s, "\\\x00:") {
		return false
	}
	parts := strings.Split(s, "/")
	if len(parts) > 16 {
		return false
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return false
		}
	}
	return true
}
