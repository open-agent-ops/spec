package formal

const (
	C4A01 = "C4-A01"
	C4A02 = "C4-A02"
	C4A03 = "C4-A03"
	C4A04 = "C4-A04"
	C4A05 = "C4-A05"
	C4A06 = "C4-A06"
	C4A07 = "C4-A07"
	C4A08 = "C4-A08"
	C4A09 = "C4-A09"
	C4A10 = "C4-A10"
	C4A11 = "C4-A11"
	C4A12 = "C4-A12"
	C4A13 = "C4-A13"
	C4A14 = "C4-A14"
	C4A15 = "C4-A15"
	C4A16 = "C4-A16"
	C4A17 = "C4-A17"
	C4A18 = "C4-A18"
	C4A19 = "C4-A19"

	EvidenceA01 = "EVD-A01-decision-version-binding"
	EvidenceA02 = "EVD-A02-negative-coverage"
	EvidenceA03 = "EVD-A03-acquisition-failure-visibility"
	EvidenceA04 = "EVD-A04-source-authority"
	EvidenceA05 = "EVD-A05-common-cause-disclosure"
	EvidenceA06 = "EVD-A06-append-only-lineage"
	EvidenceA07 = "EVD-A07-authority-separation"
)

type Assumptions map[string]bool

func (a Assumptions) Has(id string) bool { return a[id] }

func (a Assumptions) Without(ids ...string) Assumptions {
	out := make(Assumptions, len(a))
	for id, value := range a {
		out[id] = value
	}
	for _, id := range ids {
		out[id] = false
	}
	return out
}

func DefaultEvidenceAssumptions() Assumptions {
	return Assumptions{
		EvidenceA01: true,
		EvidenceA02: true,
		EvidenceA03: true,
		EvidenceA04: true,
		EvidenceA05: true,
		EvidenceA06: true,
		EvidenceA07: true,
	}
}

func DefaultC4Assumptions() Assumptions {
	return Assumptions{
		C4A01: true,
		C4A02: true,
		C4A03: true,
		C4A04: true,
		C4A05: true,
		C4A06: true,
		C4A07: true,
		C4A08: true,
		C4A09: true,
		C4A10: true,
		C4A11: true,
		C4A12: true,
		C4A13: true,
		C4A14: true,
		C4A15: true,
		C4A16: true,
		C4A17: true,
		C4A18: true,
		C4A19: true,
	}
}
