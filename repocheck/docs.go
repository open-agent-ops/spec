package repocheck

import (
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode"
)

var inlineLink = regexp.MustCompile(`!?\[[^\]\n]*\]\(([^)\n]+)\)`)
var explicitAnchor = regexp.MustCompile(`(?:id|name)=["']([^"']+)["']`)

func anchors(data []byte) map[string]bool {
	out := map[string]bool{}
	counts := map[string]int{}
	fence := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		for _, m := range explicitAnchor.FindAllStringSubmatch(line, -1) {
			out[m[1]] = true
		}
		if !strings.HasPrefix(line, "#") {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(line, "#"))
		var b strings.Builder
		for _, r := range strings.ToLower(title) {
			if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' {
				b.WriteRune(r)
			} else if unicode.IsSpace(r) {
				b.WriteByte('-')
			}
		}
		key := b.String()
		n := counts[key]
		counts[key]++
		if n > 0 {
			key += stringNumber(n)
		}
		out[key] = true
	}
	return out
}
func stringNumber(n int) string {
	if n == 0 {
		return "-0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return "-" + digits
}
func Docs(s Snapshot, p Policy) error {
	for _, f := range p.Files {
		wanted := "Apache-2.0"
		if strings.HasSuffix(f.Path, ".md") || strings.HasPrefix(f.Path, "docs/") || f.Path == "LICENSE-docs" {
			wanted = "CC-BY-4.0"
		}
		if f.License != wanted {
			return Rejectedf("docs: %s licensed %q, want %q", f.Path, f.License, wanted)
		}
		if !strings.HasSuffix(f.Path, ".md") {
			continue
		}
		data, ok := s[f.Path]
		if !ok {
			return Rejectedf("docs: %s missing from snapshot", f.Path)
		}
		for _, m := range inlineLink.FindAllSubmatch(data, -1) {
			target := strings.Trim(string(m[1]), "<>")
			if i := strings.Index(target, " \""); i >= 0 {
				target = target[:i]
			}
			u, e := url.Parse(target)
			if e != nil {
				return Rejectedf("docs: %s has an unparsable link target", f.Path)
			}
			if u.IsAbs() {
				if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "mailto" {
					return Rejectedf("docs: %s links with scheme %q", f.Path, u.Scheme)
				}
				continue
			}
			if u.Host != "" || strings.HasPrefix(u.Path, "/") {
				return Rejectedf("docs: %s has a host-relative or absolute-path link %q", f.Path, target)
			}
			dest := f.Path
			if u.Path != "" {
				dest = path.Join(path.Dir(f.Path), u.Path)
			}
			b, ok := s[dest]
			if !ok {
				return Rejectedf("docs: %s links to missing file %s", f.Path, dest)
			}
			if u.Fragment != "" && strings.HasSuffix(dest, ".md") && !anchors(b)[u.Fragment] {
				return Rejectedf("docs: %s links to missing anchor %s#%s", f.Path, dest, u.Fragment)
			}
		}
	}
	for _, name := range []string{"LICENSE", "LICENSE-docs", "NOTICE", "CONTRIBUTING.md", "GOVERNANCE.md", "SECURITY.md", "VERSIONING.md", "docs/contributing.md", "docs/releasing.md", ".github/PULL_REQUEST_TEMPLATE.md"} {
		if len(s[name]) == 0 {
			return Rejectedf("docs: required file missing or empty: %s", name)
		}
	}
	if !strings.Contains(string(s["SECURITY.md"]), "https://github.com/open-agent-ops/spec/security/advisories/new") {
		return Rejectedf("docs: SECURITY.md lacks the advisory intake URL")
	}
	var publication struct {
		Version   string `json:"version"`
		Source    string `json:"source_revision"`
		English   bool   `json:"english_precedence"`
		Artifacts []struct {
			Path     string `json:"path"`
			Hash     string `json:"sha256"`
			Language string `json:"language"`
			Pair     string `json:"pair_id"`
		} `json:"artifacts"`
	}
	var revision struct {
		Version string `json:"version"`
		Source  string `json:"source_revision"`
		English bool   `json:"english_precedence"`
	}
	if e := json.Unmarshal(s["publication-manifest.json"], &publication); e != nil {
		return Rejectedf("publication-manifest.json: %v", e)
	}
	if e := json.Unmarshal(s["revision-manifest.json"], &revision); e != nil {
		return Rejectedf("revision-manifest.json: %v", e)
	}
	switch {
	case !Version(publication.Version):
		return Rejectedf("publication-manifest.json version %q malformed", publication.Version)
	case publication.Version != revision.Version:
		return Rejectedf("publication version %q differs from revision version %q", publication.Version, revision.Version)
	case publication.Source == "" || publication.Source != revision.Source:
		return Rejectedf("publication source_revision %q differs from revision %q", publication.Source, revision.Source)
	case !publication.English || !revision.English:
		return Rejectedf("english_precedence must be true in publication and revision manifests")
	case len(publication.Artifacts) != 8:
		return Rejectedf("publication-manifest.json lists %d artifacts, want 8", len(publication.Artifacts))
	}
	for _, a := range publication.Artifacts {
		b, ok := s[a.Path]
		if !ok {
			return Rejectedf("publication artifact missing from snapshot: %s", a.Path)
		}
		if Hash(b) != a.Hash {
			return Rejectedf("publication artifact digest differs: %s", a.Path)
		}
	}
	var pairs struct {
		Entries []struct {
			English    string `json:"english_path"`
			Russian    string `json:"russian_path"`
			Source     string `json:"source_revision"`
			Required   bool   `json:"required"`
			Precedence bool   `json:"english_precedence"`
		} `json:"entries"`
	}
	if e := json.Unmarshal(s["bilingual-pairs.json"], &pairs); e != nil {
		return Rejectedf("bilingual-pairs.json: %v", e)
	}
	if len(pairs.Entries) != 3 {
		return Rejectedf("bilingual-pairs.json lists %d entries, want 3", len(pairs.Entries))
	}
	for _, a := range pairs.Entries {
		switch {
		case !a.Required || !a.Precedence:
			return Rejectedf("bilingual pair %s must be required with english precedence", a.English)
		case a.Source != revision.Source:
			return Rejectedf("bilingual pair %s source_revision %q differs from revision %q", a.English, a.Source, revision.Source)
		case len(s[a.English]) == 0 || len(s[a.Russian]) == 0:
			return Rejectedf("bilingual pair %s / %s: a side is missing or empty", a.English, a.Russian)
		}
	}
	for _, name := range []string{"bug", "proposal"} {
		file := ".github/ISSUE_TEMPLATE/" + name + ".yml"
		n, e := YAML(s[file])
		if e != nil {
			return Rejectedf("%s: %v", file, e)
		}
		var form struct {
			Name string `yaml:"name"`
			Body []struct {
				ID          string `yaml:"id"`
				Type        string `yaml:"type"`
				Validations struct {
					Required bool `yaml:"required"`
				} `yaml:"validations"`
			} `yaml:"body"`
		}
		if n.Decode(&form) != nil || form.Name == "" {
			return Rejectedf("%s: form has no name or unexpected shape", file)
		}
		ids := map[string]bool{}
		for _, f := range form.Body {
			if f.ID != "" {
				if ids[f.ID] {
					return Rejectedf("%s: field id %q duplicated", file, f.ID)
				}
				if !f.Validations.Required {
					return Rejectedf("%s: field %q is not required", file, f.ID)
				}
				ids[f.ID] = true
			}
		}
		for _, id := range []string{"kind", "version", "description", "compatibility", "bilingual", "license"} {
			if !ids[id] {
				return Rejectedf("%s: required field %q missing", file, id)
			}
		}
	}
	return nil
}
