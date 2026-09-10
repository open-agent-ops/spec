package releaseops

import (
	"os"
	"regexp"
	"strings"
)

var opaqueID = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)

func credential(name string) (string, error) {
	if name != "AOM_GITHUB_TOKEN" && name != "AOM_MIRROR_TOKEN" {
		return "", ErrAuthority
	}
	value := os.Getenv(name)
	if len(value) < 16 || len(value) > 4096 || strings.ContainsAny(value, "\r\n\x00 ") {
		return "", ErrAuthority
	}
	return value, nil
}
func ownerIDs(value string) ([]string, error) {
	ids := strings.Split(value, ",")
	if len(ids) == 0 || len(ids) > 100 {
		return nil, ErrAuthority
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !opaqueID.MatchString(id) || seen[id] {
			return nil, ErrAuthority
		}
		seen[id] = true
	}
	return ids, nil
}
