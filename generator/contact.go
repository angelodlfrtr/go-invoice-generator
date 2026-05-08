package generator

// Address represents a postal address
type Address struct {
	Address    string `json:"address,omitempty" validate:"required"`
	Address2   string `json:"address_2,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	City       string `json:"city,omitempty"`
	Country    string `json:"country,omitempty"`
}

// ToString formats the address as a multiline string
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

// Contact holds company or customer information
type Contact struct {
	Name    string   `json:"name,omitempty" validate:"required,min=1,max=256"`
	Logo    []byte   `json:"logo,omitempty"`
	Address *Address `json:"address,omitempty"`

	// AddtionnalInfo lines appended after contact info; basic HTML (bold, italic) is supported
	AddtionnalInfo []string `json:"additional_info,omitempty"`
}
