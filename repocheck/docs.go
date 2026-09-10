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
			return ErrRejected
		}
		if !strings.HasSuffix(f.Path, ".md") {
			continue
		}
		data, ok := s[f.Path]
		if !ok {
			return ErrRejected
		}
		for _, m := range inlineLink.FindAllSubmatch(data, -1) {
			target := strings.Trim(string(m[1]), "<>")
			if i := strings.Index(target, " \""); i >= 0 {
				target = target[:i]
			}
			u, e := url.Parse(target)
			if e != nil {
				return ErrRejected
			}
			if u.IsAbs() {
				if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "mailto" {
					return ErrRejected
				}
				continue
			}
			if u.Host != "" || strings.HasPrefix(u.Path, "/") {
				return ErrRejected
			}
			dest := f.Path
			if u.Path != "" {
				dest = path.Join(path.Dir(f.Path), u.Path)
			}
			b, ok := s[dest]
			if !ok {
				return ErrRejected
			}
			if u.Fragment != "" && strings.HasSuffix(dest, ".md") && !anchors(b)[u.Fragment] {
				return ErrRejected
			}
		}
	}
	for _, name := range []string{"LICENSE", "LICENSE-docs", "NOTICE", "CONTRIBUTING.md", "GOVERNANCE.md", "SECURITY.md", "VERSIONING.md", "docs/contributing.md", "docs/releasing.md", ".github/PULL_REQUEST_TEMPLATE.md"} {
		if len(s[name]) == 0 {
			return ErrRejected
		}
	}
	if !strings.Contains(string(s["SECURITY.md"]), "https://github.com/open-agent-ops/spec/security/advisories/new") {
		return ErrRejected
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
	if json.Unmarshal(s["publication-manifest.json"], &publication) != nil || json.Unmarshal(s["revision-manifest.json"], &revision) != nil || publication.Version != revision.Version || !Version(publication.Version) || publication.Source == "" || publication.Source != revision.Source || !publication.English || !revision.English || len(publication.Artifacts) != 8 {
		return ErrRejected
	}
	for _, a := range publication.Artifacts {
		b, ok := s[a.Path]
		if !ok || Hash(b) != a.Hash {
			return ErrRejected
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
	if json.Unmarshal(s["bilingual-pairs.json"], &pairs) != nil || len(pairs.Entries) != 3 {
		return ErrRejected
	}
	for _, a := range pairs.Entries {
		if !a.Required || !a.Precedence || a.Source != revision.Source || len(s[a.English]) == 0 || len(s[a.Russian]) == 0 {
			return ErrRejected
		}
	}
	for _, name := range []string{"bug", "proposal"} {
		n, e := YAML(s[".github/ISSUE_TEMPLATE/"+name+".yml"])
		if e != nil {
			return ErrRejected
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
			return ErrRejected
		}
		ids := map[string]bool{}
		for _, f := range form.Body {
			if f.ID != "" {
				if ids[f.ID] || !f.Validations.Required {
					return ErrRejected
				}
				ids[f.ID] = true
			}
		}
		for _, id := range []string{"kind", "version", "description", "compatibility", "bilingual", "license"} {
			if !ids[id] {
				return ErrRejected
			}
		}
	}
	return nil
}
