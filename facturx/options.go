package facturx

import (
	"strings"

	generator "github.com/angelodlfrtr/go-invoice-generator/generator"
)

// Options holds Factur-X-specific data that cannot be derived from the generator.Document.
//
// Dates accept "DD/MM/YYYY", "YYYY-MM-DD" or "YYYYMMDD". Country codes must be
// ISO 3166-1 alpha-2 and currency codes ISO 4217; both are validated.
//
// Fields are grouped by the Factur-X profile that first carries them. Fields
// set for a lower profile than the one they belong to are silently omitted
// from the XML.
type Options struct {
	// Profile is the Factur-X conformance level. Defaults to ProfileMinimum.
	Profile Profile

	// ---- Document (all profiles) ----

	// CurrencyCode is the ISO 4217 currency code. Defaults to "EUR".
	CurrencyCode string

	// TypeCode is the UN/CEFACT document type code. Defaults to "380" (invoice).
	// Other common values: "381" (credit note), "84" (delivery note).
	TypeCode string

	// BusinessProcessID is the business process context (BT-23), e.g.
	// "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0" for Peppol / XRechnung.
	// The French e-invoicing specifications carry the "cadre de facturation"
	// here (B1 goods, S1 services, M1 mixed, and their variants).
	BusinessProcessID string

	// FrenchRules enables the rules of the French e-invoicing reform
	// (AFNOR XP Z12-012, "Flux 2") on top of EN 16931. It requires profile
	// EN 16931 or EXTENDED, a seller in France identified by a SIREN
	// (SellerLegalID with scheme "0002"), BusinessProcessID set to the cadre
	// de facturation, the three legal mentions as notes with subject codes
	// PMT, PMD and AAB, French VAT rates, and a preceding invoice with its
	// date on credit notes. It also adds a "BAR" note carrying FrenchFlow and
	// defaults the electronic addresses to the SIREN with scheme "0225".
	// Violations wrap ErrFrenchRules.
	FrenchRules bool

	// FrenchFlow is the processing flow stated in the BAR note (BT-21/22):
	// "B2B" (default), "B2BINT", "B2C", "OUTOFSCOPE" or "ARCHIVEONLY". With
	// "B2B" the buyer needs a SIREN. Only used with FrenchRules.
	FrenchFlow string

	// BuyerReference is the buyer's internal routing reference (BT-10), such
	// as a cost centre or the Leitweg-ID. For a purchase order number use
	// PurchaseOrderReference.
	BuyerReference string

	// PurchaseOrderReference is the buyer's purchase order number (BT-13).
	PurchaseOrderReference string

	// ---- Seller (all profiles unless noted) ----

	// SellerTaxID is the seller's VAT registration number (e.g. "FR12345678901").
	// Either SellerTaxID or SellerLegalID is required (BR-CO-26).
	SellerTaxID string

	// SellerLegalID is the seller's legal registration identifier (BT-30),
	// e.g. the SIREN in France. SellerLegalIDSchemeID is the ISO 6523 ICD of
	// the register, e.g. "0002" for SIRENE or "0088" for GLN; it is optional.
	SellerLegalID         string
	SellerLegalIDSchemeID string

	// SellerCountryCode is the seller's ISO 3166-1 alpha-2 country code used in
	// the CII XML address. Falls back to doc.Company.Address.Country when empty.
	SellerCountryCode string

	// SellerElectronicAddress is the seller's electronic address (BT-34), with
	// SellerElectronicAddressSchemeID the EAS scheme ("EM" for e-mail,
	// "0009" for SIRET, "0088" for GLN...). Scheme defaults to "EM".
	// BASIC-WL and above.
	SellerElectronicAddress         string
	SellerElectronicAddressSchemeID string

	// SellerContactName, SellerContactPhone and SellerContactEmail fill the
	// seller contact group (BG-6). EN 16931 and above.
	SellerContactName  string
	SellerContactPhone string
	SellerContactEmail string

	// ---- Buyer ----

	// BuyerTaxID is the buyer's VAT registration number. BASIC-WL and above.
	BuyerTaxID string

	// BuyerLegalID is the buyer's legal registration identifier (BT-47) with
	// its optional ISO 6523 scheme. All profiles.
	BuyerLegalID         string
	BuyerLegalIDSchemeID string

	// BuyerCountryCode is the buyer's ISO 3166-1 alpha-2 country code used in
	// the CII XML address. Falls back to doc.Customer.Address.Country when empty.
	BuyerCountryCode string

	// BuyerElectronicAddress is the buyer's electronic address (BT-49), see
	// SellerElectronicAddress. BASIC-WL and above.
	BuyerElectronicAddress         string
	BuyerElectronicAddressSchemeID string

	// ---- Delivery and period (BASIC-WL and above) ----

	// DeliveryDate is the actual delivery date (BT-72).
	DeliveryDate string

	// InvoicingPeriodStart and InvoicingPeriodEnd define the invoicing
	// period (BG-14). Both must be set together.
	InvoicingPeriodStart string
	InvoicingPeriodEnd   string

	// ShipToName and ShipToAddress describe the deliver-to party (BG-13,
	// BG-15) when goods go somewhere other than the buyer's address. France
	// requires it whenever the delivery address differs from the billing one.
	// ShipToCountryCode falls back to ShipToAddress.Country.
	ShipToName        string
	ShipToAddress     *generator.Address
	ShipToCountryCode string

	// ---- Payment (BASIC-WL and above) ----

	// PaymentDueDate is the payment due date. When empty, doc.PaymentTerm is
	// used if it parses as a date. EN 16931 requires either a due date or a
	// PaymentTermsDescription when an amount is due (BR-CO-25).
	PaymentDueDate string

	// PaymentTermsDescription is the free-text payment terms (BT-20).
	PaymentTermsDescription string

	// PaymentReference is the remittance information (BT-83) the buyer should
	// quote when paying.
	PaymentReference string

	// PaymentIBAN is the seller's IBAN for bank transfer payment.
	PaymentIBAN string

	// PaymentBIC is the seller's BIC/SWIFT code. EN 16931 and above.
	PaymentBIC string

	// PaymentMeansCode is the UN/ECE 4461 payment means type code (e.g. "30" for
	// credit transfer, "58" for SEPA credit transfer). Defaults to "58" when
	// PaymentIBAN is set.
	PaymentMeansCode string

	// PrecedingInvoiceReference and PrecedingInvoiceDate identify the invoice
	// a credit note (TypeCode "381") corrects (BG-3). The reference is
	// mandatory on French credit notes.
	PrecedingInvoiceReference string
	PrecedingInvoiceDate      string

	// ---- Notes (BASIC-WL and above) ----

	// Notes are the document notes (BG-1). When empty, doc.Notes is used as
	// a single note without subject code.
	Notes []Note

	// ---- Taxes ----

	// TaxCategoryCode is the VAT category code applied to items whose Tax has
	// no Category. Defaults to "S" (standard rate).
	TaxCategoryCode string

	// NoTaxCategoryCode is the VAT category code applied to items without a
	// Tax. Defaults to "E" (exempt). NoTaxExemptionReason and
	// NoTaxExemptionReasonCode give the mandatory reason for that category,
	// e.g. "TVA non applicable, art. 293 B du CGI".
	NoTaxCategoryCode        string
	NoTaxExemptionReason     string
	NoTaxExemptionReasonCode string

	// DocumentDiscountReason is the reason text for the document-level
	// allowance generated from doc.Discount (BT-97). Defaults to "Discount".
	DocumentDiscountReason string

	// VATPointDateCode states when VAT becomes chargeable (BT-8, UNTDID 2475):
	// "5" invoice date, "29" delivery date, "72" payment date. In France use
	// "5" for "TVA sur les débits" and "72" for "TVA sur les encaissements".
	// BASIC-WL and above.
	VATPointDateCode string

	// TaxCurrencyCode (BT-6) is set when the invoice currency is not the
	// currency of the seller's tax authority; the total VAT is then also
	// given in that currency, converted with TaxCurrencyRate (tax currency
	// per unit of invoice currency, e.g. "0.9234"). BASIC-WL and above.
	TaxCurrencyCode string
	TaxCurrencyRate string

	// ---- Items (BASIC and above) ----

	// ItemDefaultUnitCode is the UN/ECE recommendation 20 unit code applied to all
	// line items. Defaults to "C62" (piece/unit).
	ItemDefaultUnitCode string

	// ---- PDF ----

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

func (o Options) noTaxCategoryCode() string {
	if o.NoTaxCategoryCode != "" {
		return o.NoTaxCategoryCode
	}
	return "E"
}

func (o Options) documentDiscountReason() string {
	if o.DocumentDiscountReason != "" {
		return o.DocumentDiscountReason
	}
	return "Discount"
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

func (o Options) shipToCountryCode() string {
	if o.ShipToCountryCode != "" {
		return o.ShipToCountryCode
	}
	if o.ShipToAddress != nil {
		return o.ShipToAddress.Country
	}
	return ""
}

func (o Options) hasShipTo() bool {
	return o.ShipToName != "" || o.ShipToAddress != nil || o.ShipToCountryCode != ""
}

func (o Options) sellerElectronicAddressSchemeID() string {
	if o.SellerElectronicAddressSchemeID != "" {
		return o.SellerElectronicAddressSchemeID
	}
	return "EM"
}

func (o Options) buyerElectronicAddressSchemeID() string {
	if o.BuyerElectronicAddressSchemeID != "" {
		return o.BuyerElectronicAddressSchemeID
	}
	return "EM"
}

func (o Options) notes(doc *generator.Document) []Note {
	if len(o.Notes) > 0 {
		return o.Notes
	}
	if doc.Notes != "" {
		return []Note{{Content: doc.Notes}}
	}
	return nil
}

// Note is a document note (BG-1): free text plus an optional UNTDID 4451
// subject code. France requires three notes: "PMT" (indemnité forfaitaire
// pour frais de recouvrement), "PMD" (pénalités de retard) and "AAB"
// (conditions d'escompte, or its absence).
type Note struct {
	Content     string
	SubjectCode string
}

func (o Options) frenchFlow() string {
	if o.FrenchFlow != "" {
		return strings.ToUpper(strings.TrimSpace(o.FrenchFlow))
	}
	return "B2B"
}

// applyFrenchDefaults fills the values the French rules expect when the
// caller left them empty: electronic addresses as the SIREN with the French
// routing scheme "0225".
func applyFrenchDefaults(o Options) Options {
	if !o.FrenchRules {
		return o
	}
	if o.SellerElectronicAddress == "" && o.SellerLegalID != "" {
		o.SellerElectronicAddress = o.SellerLegalID
		o.SellerElectronicAddressSchemeID = frEndpointScheme
	}
	if o.BuyerElectronicAddress == "" && o.BuyerLegalID != "" {
		o.BuyerElectronicAddress = o.BuyerLegalID
		o.BuyerElectronicAddressSchemeID = frEndpointScheme
	}
	return o
}
