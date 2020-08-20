package generator

// Address represent an address
type Address struct {
	Address    string `validate:"required" json:"address"`
	Address2   string `json:"address2"`
	PostalCode string `validate:"required" json:"postalCode"`
	City       string `validate:"required" json:"city"`
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

	addrString += "\n"
	addrString += a.PostalCode
	addrString += " "
	addrString += a.City

	if len(a.Country) > 0 {
		addrString += "\n"
		addrString += a.Country
	}

	return addrString
}
