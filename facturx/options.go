package facturx

import (
	"strings"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

// Note is the document note type, see invoice.Note.
type Note = invoice.Note

// Options holds the Factur-X settings that are not part of the document
// itself. Business facts (parties, identifiers, delivery, payment, notes,
// currency) live on the invoice.Document.
//
// Fields are grouped by the Factur-X profile that first carries them;
// document data belonging to a higher profile than the selected one is
// silently omitted from the XML.
type Options struct {
	// Profile is the Factur-X conformance level. Defaults to ProfileMinimum.
	Profile Profile

	// BusinessProcessID is the business process context (BT-23), e.g.
	// "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0" for Peppol / XRechnung.
	// The French e-invoicing specifications carry the "cadre de facturation"
	// here (B1 goods, S1 services, M1 mixed, and their variants).
	BusinessProcessID string

	// TypeCode overrides the UN/CEFACT document type code derived from the
	// document type: "380" for an invoice, "381" for a credit note.
	TypeCode string

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

	// ItemDefaultUnitCode is the UN/ECE Recommendation 20 unit code applied
	// to items whose Unit is empty. Defaults to "C62" (piece/unit).
	ItemDefaultUnitCode string

	// ---- PDF ----

	// ShowIcon places the Factur-X profile icon in the bottom-right corner of
	// the first page as a compliance mark. Defaults to false.
	ShowIcon bool

	// ---- France ----

	// FrenchRules enables the rules of the French e-invoicing reform
	// (AFNOR XP Z12-012, "Flux 2") on top of EN 16931. It requires profile
	// EN 16931 or EXTENDED, a seller in France identified by a SIREN
	// (Contact.LegalID with scheme "0002"), BusinessProcessID set to the
	// cadre de facturation, the three legal mentions as notes with subject
	// codes PMT, PMD and AAB, French VAT rates, and a preceding invoice with
	// its date on credit notes. It also adds a "BAR" note carrying FrenchFlow
	// and defaults the electronic addresses to the SIREN with scheme "0225".
	// Violations wrap ErrFrenchRules.
	FrenchRules bool

	// FrenchFlow is the processing flow stated in the BAR note (BT-21/22):
	// "B2B" (default), "B2BINT", "B2C", "OUTOFSCOPE" or "ARCHIVEONLY". With
	// "B2B" the buyer needs a SIREN. Only used with FrenchRules.
	FrenchFlow string
}

func (o Options) profile() Profile {
	if o.Profile == "" {
		return ProfileMinimum
	}
	return o.Profile
}

func (o Options) typeCode(doc *invoice.Document) string {
	if o.TypeCode != "" {
		return o.TypeCode
	}
	if doc.Type == invoice.CreditNote {
		return "381"
	}
	return "380"
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

func (o Options) itemUnitCode(item *invoice.Item) string {
	if item.Unit != "" {
		return item.Unit
	}
	if o.ItemDefaultUnitCode != "" {
		return o.ItemDefaultUnitCode
	}
	return "C62"
}

func (o Options) frenchFlow() string {
	if o.FrenchFlow != "" {
		return strings.ToUpper(strings.TrimSpace(o.FrenchFlow))
	}
	return "B2B"
}

// paymentMeansCode returns the payment means code for doc.Payment: the one
// given, "58" (SEPA credit transfer) when an IBAN is set, "" otherwise.
func paymentMeansCode(p *invoice.Payment) string {
	if p == nil {
		return ""
	}
	if p.MeansCode != "" {
		return p.MeansCode
	}
	if p.IBAN != "" {
		return "58"
	}
	return ""
}

// endpoint returns the electronic address and scheme of a party. Under the
// French rules a missing address defaults to the SIREN with the French
// routing scheme "0225"; an address without scheme is an e-mail ("EM").
func (o Options) endpoint(c *invoice.Contact) (address, scheme string) {
	address, scheme = c.ElectronicAddress, c.ElectronicAddressScheme
	if address == "" && o.FrenchRules && c.LegalID != "" {
		return c.LegalID, frEndpointScheme
	}
	if address != "" && scheme == "" {
		scheme = "EM"
	}
	return address, scheme
}
