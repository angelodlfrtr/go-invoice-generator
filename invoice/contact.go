package invoice

import "strings"

// Address is a postal address.
type Address struct {
	Address    string `json:"address,omitempty" validate:"required"`
	Address2   string `json:"address_2,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	City       string `json:"city,omitempty"`
	// Country is printed on the PDF as given, e.g. "France".
	Country string `json:"country,omitempty"`
	// CountryCode is the ISO 3166-1 alpha-2 code used by exporters. When
	// empty, Country is used if it already is a two-letter code.
	CountryCode string `json:"country_code,omitempty"`
}

// ISOCountry returns the ISO 3166-1 alpha-2 code of the address, upper-cased,
// or "" when none can be derived.
func (a *Address) ISOCountry() string {
	if a == nil {
		return ""
	}
	if c := strings.ToUpper(strings.TrimSpace(a.CountryCode)); c != "" {
		return c
	}
	if c := strings.ToUpper(strings.TrimSpace(a.Country)); len(c) == 2 {
		return c
	}
	return ""
}

// ToString formats the address as a multiline string.
func (a *Address) ToString() string {
	addrString := a.Address

	if len(a.Address2) > 0 {
		addrString += "\n" + a.Address2
	}

	if len(a.PostalCode) > 0 {
		addrString += "\n" + a.PostalCode
	} else {
		addrString += "\n"
	}

	if len(a.City) > 0 {
		addrString += " " + a.City
	}

	if len(a.Country) > 0 {
		addrString += "\n" + a.Country
	}

	return addrString
}

// Contact is a party: the seller, the buyer or a deliver-to address.
type Contact struct {
	Name    string   `json:"name,omitempty" validate:"max=256"`
	Logo    []byte   `json:"logo,omitempty"`
	Address *Address `json:"address,omitempty"`

	// VATID is the VAT registration number (BT-31 / BT-48), e.g. "FR12345678901".
	VATID string `json:"vat_id,omitempty"`
	// LegalID is the legal registration identifier (BT-30 / BT-47), e.g. a
	// SIREN, with LegalIDScheme its ISO 6523 ICD ("0002" SIRENE, "0088" GLN).
	LegalID       string `json:"legal_id,omitempty"`
	LegalIDScheme string `json:"legal_id_scheme,omitempty"`
	// ElectronicAddress is the e-invoicing endpoint (BT-34 / BT-49) with its
	// EAS scheme ("EM" e-mail, "0225" French SIREN routing, "0088" GLN...).
	ElectronicAddress       string `json:"electronic_address,omitempty"`
	ElectronicAddressScheme string `json:"electronic_address_scheme,omitempty"`
	// ContactName, ContactPhone and ContactEmail identify a person (BG-6 / BG-9).
	ContactName  string `json:"contact_name,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`

	// AdditionalInfo lines are printed below the address; basic HTML
	// (bold, italic) is supported.
	AdditionalInfo []string `json:"additional_info,omitempty"`
}
