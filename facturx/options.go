package facturx

import generator "github.com/angelodlfrtr/go-invoice-generator/generator"

// Options holds Factur-X-specific data that cannot be derived from the generator.Document.
type Options struct {
	// Profile is the Factur-X conformance level. Defaults to ProfileMinimum.
	Profile Profile

	// CurrencyCode is the ISO 4217 currency code. Defaults to "EUR".
	CurrencyCode string

	// SellerTaxID is the seller's VAT registration number (e.g. "FR12345678901").
	// Required for most profiles.
	SellerTaxID string

	// SellerCountryCode is the seller's ISO 3166-1 alpha-2 country code used in
	// the CII XML address. Falls back to doc.Company.Address.Country when empty.
	SellerCountryCode string

	// BuyerCountryCode is the buyer's ISO 3166-1 alpha-2 country code used in
	// the CII XML address. Falls back to doc.Customer.Address.Country when empty.
	BuyerCountryCode string

	// BuyerReference is the buyer's internal reference (e.g. a purchase order number).
	BuyerReference string

	// BuyerTaxID is the buyer's VAT registration number. Rendered in
	// BuyerTradeParty/SpecifiedTaxRegistration for BASIC-WL and above.
	BuyerTaxID string

	// PaymentDueDate is the payment due date in "YYYYMMDD" format.
	PaymentDueDate string

	// PaymentIBAN is the seller's IBAN for bank transfer payment.
	PaymentIBAN string

	// PaymentBIC is the seller's BIC/SWIFT code for bank transfer payment.
	PaymentBIC string

	// PaymentMeansCode is the UN/ECE 4461 payment means type code (e.g. "30" for
	// credit transfer, "58" for SEPA credit transfer). Defaults to "58" when
	// PaymentIBAN is set. Required for EN16931/EXTENDED when payment means are present.
	PaymentMeansCode string

	// TaxCategoryCode is the default VAT category code applied when a tax rate has no
	// explicit category. Common values: "S" (standard), "E" (exempt), "Z" (zero-rated),
	// "G" (export). Defaults to "S".
	TaxCategoryCode string

	// TypeCode is the UN/CEFACT document type code. Defaults to "380" (invoice).
	// Other common values: "381" (credit note), "84" (delivery note).
	TypeCode string

	// ItemDefaultUnitCode is the UN/ECE recommendation 20 unit code applied to all
	// line items. Defaults to "C62" (piece/unit).
	ItemDefaultUnitCode string

	// ShowIcon places the Factur-X profile icon in the bottom-right corner of
	// the first page as a compliance mark. Defaults to false.
	ShowIcon bool
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

func (o Options) itemDefaultUnitCode() string {
	if o.ItemDefaultUnitCode != "" {
		return o.ItemDefaultUnitCode
	}
	return "C62"
}

func (o Options) paymentMeansCode() string {
	if o.PaymentMeansCode != "" {
		return o.PaymentMeansCode
	}
	if o.PaymentIBAN != "" {
		return "58"
	}
	return ""
}

func (o Options) sellerCountryCode(doc *generator.Document) string {
	if o.SellerCountryCode != "" {
		return o.SellerCountryCode
	}
	if doc.Company != nil && doc.Company.Address != nil {
		return doc.Company.Address.Country
	}
	return ""
}

func (o Options) buyerCountryCode(doc *generator.Document) string {
	if o.BuyerCountryCode != "" {
		return o.BuyerCountryCode
	}
	if doc.Customer != nil && doc.Customer.Address != nil {
		return doc.Customer.Address.Country
	}
	return ""
}
