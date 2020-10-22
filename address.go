package generator

// Address represent an address
type Address struct {
	Address    string `validate:"required" json:"address"`
	Address2   string `json:"address2"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Country    string `json:"country"`
}

// ToString output address as string
// Line break are added for new lines
func (a *Address) ToString() string {
	var addrString string = a.Address

	if len(a.Address2) > 0 {
		addrString += "\n"
		addrString += a.Address2
	}

	if len(a.PostalCode) > 0 {
		addrString += "\n"
		addrString += a.PostalCode
	} else {
		addrString += "\n"
	}

	if len(a.City) > 0 {
		addrString += " "
		addrString += a.City
	}

	if len(a.Country) > 0 {
		addrString += "\n"
		addrString += a.Country
	}

	return addrString
}
