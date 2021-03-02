package generator

// SetType set type of document
func (d *Document) SetType(docType string) *Document {
	d.Type = docType
	return d
}

// SetHeader set header of document
func (d *Document) SetHeader(header *HeaderFooter) *Document {
	d.Header = header
	return d
}

// SetFooter set footer of document
func (d *Document) SetFooter(footer *HeaderFooter) *Document {
	d.Footer = footer
	return d
}

// SetRef of document
func (d *Document) SetRef(ref string) *Document {
	d.Ref = ref
	return d
}

// SetVersion of document
func (d *Document) SetVersion(version string) *Document {
	d.Version = version
	return d
}

// SetDescription of document
func (d *Document) SetDescription(desc string) *Document {
	d.Description = desc
	return d
}

// SetNotes of document
func (d *Document) SetNotes(notes string) *Document {
	d.Notes = notes
	return d
}

// SetCompany of document
func (d *Document) SetCompany(company *Contact) *Document {
	d.Company = company
	return d
}

// SetCustomer of document
func (d *Document) SetCustomer(customer *Contact) *Document {
	d.Customer = customer
	return d
}

// AppendItem to document items
func (d *Document) AppendItem(item *Item) *Document {
	d.Items = append(d.Items, item)
	return d
}

// SetDate of document
func (d *Document) SetDate(date string) *Document {
	d.Date = date
	return d
}

// SetPaymentTerm of document
func (d *Document) SetPaymentTerm(term string) *Document {
	d.PaymentTerm = term
	return d
}

// SetDefaultTax of document
func (d *Document) SetDefaultTax(tax *Tax) *Document {
	d.DefaultTax = tax
	return d
}

// SetDiscount of document
func (d *Document) SetDiscount(discount *Discount) *Document {
	d.Discount = discount
	return d
}
