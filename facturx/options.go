package facturx

// Options holds Factur-X-specific data that cannot be derived from the generator.Document.
type Options struct {
	// Profile is the Factur-X conformance level. Defaults to ProfileMinimum.
	Profile Profile

	// CurrencyCode is the ISO 4217 currency code. Defaults to "EUR".
	CurrencyCode string

	// SellerTaxID is the seller's VAT registration number (e.g. "FR12345678901").
	// Required for most profiles.
	SellerTaxID string

	// BuyerReference is the buyer's internal reference (e.g. a purchase order number).
	BuyerReference string

	// PaymentDueDate is the payment due date in "YYYYMMDD" format.
	PaymentDueDate string

	// PaymentIBAN is the seller's IBAN for bank transfer payment.
	PaymentIBAN string

	// PaymentBIC is the seller's BIC/SWIFT code for bank transfer payment.
	PaymentBIC string

	// TaxCategoryCode is the default VAT category code applied when a tax rate has no
	// explicit category. Common values: "S" (standard), "E" (exempt), "Z" (zero-rated),
	// "G" (export). Defaults to "S".
	TaxCategoryCode string

	// TypeCode is the UN/CEFACT document type code. Defaults to "380" (invoice).
	// Other common values: "381" (credit note), "84" (delivery note).
	TypeCode string
}

func (o Options) profile() Profile {
	if o.Profile == "" {
		return ProfileMinimum
	}
	return o.Profile
}

func (o Options) currencyCode() string {
	if o.CurrencyCode != "" {
		return o.CurrencyCode
	}
	return "EUR"
}

func (o Options) taxCategoryCode() string {
	if o.TaxCategoryCode != "" {
		return o.TaxCategoryCode
	}
	return "S"
}

func (o Options) typeCode() string {
	if o.TypeCode != "" {
		return o.TypeCode
	}
	return "380"
}
