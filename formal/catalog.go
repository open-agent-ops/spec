package formal

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed catalogs/*.json
var catalogFS embed.FS

type Relation struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Authority string `json:"authority"`
	CheckTime string `json:"check_time"`
	Failure   string `json:"failure"`
}

type Catalog struct {
	CatalogID string     `json:"catalog_id"`
	AppliesTo []string   `json:"applies_to"`
	Relations []Relation `json:"relations"`
}

func CatalogNames() []string {
	return []string{
		"c4-evidence-interface-v1.0.0.json",
		"c4-relations-v1.0.0.json",
		"evidence-relations-v1.0.0.json",
	}
}

func OpenCatalog(name string) (Catalog, error) {
	if strings.Contains(name, "/") || !strings.HasSuffix(name, ".json") {
		return Catalog{}, errors.New("invalid catalog name")
	}
	b, err := fs.ReadFile(catalogFS, "catalogs/"+name)
	if err != nil {
		return Catalog{}, err
	}
	var c Catalog
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Catalog{}, err
	}
	if err := ValidateCatalog(c); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

func ValidateCatalog(c Catalog) error {
	if c.CatalogID == "" || !strings.HasSuffix(c.CatalogID, "@1.0.0") || len(c.AppliesTo) == 0 || len(c.Relations) == 0 {
		return errors.New("catalog header incomplete")
	}
	seen := map[string]bool{}
	for _, r := range c.Relations {
		if r.ID == "" || r.Name == "" || r.Authority == "" || r.CheckTime == "" || r.Failure == "" {
			return fmt.Errorf("catalog %s has incomplete relation", c.CatalogID)
		}
		if seen[r.ID] {
			return fmt.Errorf("catalog %s duplicates relation %s", c.CatalogID, r.ID)
		}
		seen[r.ID] = true
	}
	return nil
}
