package conformance

import (
	"errors"
	"github.com/open-agent-ops/spec/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"net/url"
	"sort"
	"strings"
)

var ErrCompile = errors.New("schema_compilation_failed")

type localLoader struct{ registry *schemas.Registry }

func (l localLoader) Load(id string) (any, error) {
	b, err := l.registry.Lookup(id)
	if err != nil {
		return nil, ErrCompile
	}
	return Decode(b)
}

// Validator owns one fully compiled approved registry.
type Validator struct {
	registry string
	compiled map[string]*jsonschema.Schema
}

func New() (*Validator, error) {
	r, err := schemas.Open()
	if err != nil {
		return nil, ErrCompile
	}
	c := jsonschema.NewCompiler()
	// Formats such as date-time and uri are annotations by default in draft
	// 2020-12; every declared format is enforced as an assertion here.
	c.AssertFormat()
	c.UseLoader(localLoader{r})
	for _, e := range r.Entries() {
		b, err := r.Lookup(e.ID)
		if err != nil {
			return nil, ErrCompile
		}
		doc, err := Decode(b)
		if err != nil {
			return nil, ErrCompile
		}
		count := 0
		if checkRefs(doc, e.ID, r, &count) != nil {
			return nil, ErrCompile
		}
		if c.AddResource(e.ID, doc) != nil {
			return nil, ErrCompile
		}
	}
	v := &Validator{registry: r.Digest(), compiled: map[string]*jsonschema.Schema{}}
	for _, e := range r.Entries() {
		s, err := c.Compile(e.ID)
		if err != nil {
			return nil, ErrCompile
		}
		v.compiled[e.ID] = s
	}
	return v, nil
}

func checkRefs(v any, base string, r *schemas.Registry, count *int) error {
	switch x := v.(type) {
	case map[string]any:
		for k, v := range x {
			if k == "$ref" || k == "$dynamicRef" {
				*count++
				if *count > 1024 {
					return ErrCompile
				}
				s, ok := v.(string)
				if !ok {
					return ErrCompile
				}
				u, err := url.Parse(s)
				if err != nil {
					return ErrCompile
				}
				for _, p := range strings.Split(u.Path, "/") {
					if p == ".." || p == "." {
						return ErrCompile
					}
				}
				b, err := url.Parse(base)
				if err != nil {
					return ErrCompile
				}
				resolved := b.ResolveReference(u)
				resolved.Fragment = ""
				resolved.RawFragment = ""
				if _, err = r.Lookup(resolved.String()); err != nil {
					return ErrCompile
				}
			}
			if err := checkRefs(v, base, r, count); err != nil {
				return err
			}
		}
	case []any:
		for _, v := range x {
			if err := checkRefs(v, base, r, count); err != nil {
				return err
			}
		}
	}
	return nil
}

type Result struct {
	Registry string   `json:"registry"`
	Schema   string   `json:"schema"`
	Accepted bool     `json:"accepted"`
	Reasons  []string `json:"reasons"`
}

// Validate emits keyword classes only, never input payloads or private paths.
func (v *Validator) Validate(id string, data []byte) Result {
	out := Result{Reasons: []string{}}
	if v == nil {
		out.Reasons = []string{"validator_missing"}
		return out
	}
	out.Registry = v.registry
	s, ok := v.compiled[id]
	if !ok {
		out.Reasons = []string{"schema_unknown"}
		return out
	}
	out.Schema = id
	doc, err := Decode(data)
	if err != nil {
		out.Reasons = []string{"input_invalid"}
		return out
	}
	if err = s.Validate(doc); err != nil {
		var ve *jsonschema.ValidationError
		if !errors.As(err, &ve) {
			out.Reasons = []string{"validation_error"}
			return out
		}
		out.Reasons = validationReasons(ve, 1024)
		return out
	}
	out.Accepted = true
	return out
}

func validationReasons(ve *jsonschema.ValidationError, limit int) []string {
	seen := map[string]bool{}
	count := 0
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		count++
		if count > limit {
			seen["findings_overflow"] = true
			return
		}
		if len(e.Causes) == 0 {
			p := e.ErrorKind.KeywordPath()
			if len(p) > 0 {
				seen[p[0]] = true
			} else {
				seen["schema_violation"] = true
			}
		}
		for _, c := range e.Causes {
			if count > limit {
				break
			}
			walk(c)
		}
	}
	walk(ve)
	out := []string{}
	for reason := range seen {
		out = append(out, reason)
	}
	sort.Strings(out)
	return out
}
