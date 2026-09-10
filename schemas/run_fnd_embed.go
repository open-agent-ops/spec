package schemas

// Resolve selects only the two declared live schema families and exact versions.
// Schema versions are independent of publication text and release tags.
func (r *Registry) Resolve(family, version string) (Entry, error) {
	if family != "foundation_object_ref" && family != "foundation_semantic_object" {
		return Entry{}, ErrUnknown
	}
	suffix := ""
	switch version {
	case "1.0.0":
	case "1.1.0":
		suffix = "_v1_1"
	case "1.2.0":
		suffix = "_v1_2"
	case "1.3.0":
		suffix = "_v1_3"
	default:
		return Entry{}, ErrUnknown
	}
	for _, e := range r.Entries() {
		if e.Name == family+suffix+".schema.json" {
			return e, nil
		}
	}
	return Entry{}, ErrUnknown
}
