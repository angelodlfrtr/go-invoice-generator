package facturx

// Profile represents a Factur-X conformance level.
type Profile string

const (
	ProfileMinimum  Profile = "MINIMUM"
	ProfileBasicWL  Profile = "BASIC-WL"
	ProfileBasic    Profile = "BASIC"
	ProfileEN16931  Profile = "EN 16931"
	ProfileExtended Profile = "EXTENDED"
)

func (p Profile) guidelineID() string {
	switch p {
	case ProfileBasicWL:
		return "urn:factur-x.eu:1p0:basicwl"
	case ProfileBasic:
		return "urn:factur-x.eu:1p0:basic"
	case ProfileEN16931:
		return "urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:en16931"
	case ProfileExtended:
		return "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended"
	default:
		return "urn:factur-x.eu:1p0:minimum"
	}
}

// String returns the profile name.
func (p Profile) String() string { return string(p) }
