package facturx

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

// French e-invoicing reform (AFNOR XP Z12-012, "Flux 2") rules. The checks
// below mirror the BR-FR schematron published with the French CII profile so
// that a document accepted here passes a French-aware validator.

// ErrFrenchRules is returned when Options.FrenchRules is set and the
// document breaks a requirement of the French e-invoicing reform.
var ErrFrenchRules = errors.New("french e-invoicing rule violated")

const (
	// sirenScheme is the ISO 6523 ICD of the French SIRENE register.
	sirenScheme = "0002"
	// frEndpointScheme is the ISO 6523 ICD of the French e-invoicing
	// electronic address (SIREN or SIRET plus optional routing code).
	frEndpointScheme = "0225"
)

var (
	sirenRe = regexp.MustCompile(`^[0-9]{9}$`)
	// frIDRe is the character set allowed in identifiers and electronic
	// addresses (BR-FR-02, BR-FR-23).
	frIDRe = regexp.MustCompile(`^[A-Za-z0-9+\-_/]+$`)

	frTypeCodes         = set("380", "389", "393", "501", "386", "500", "384", "471", "472", "473", "261", "262", "381", "396", "502", "503")
	frCadres            = set("B1", "S1", "M1", "B2", "S2", "M2", "B4", "S4", "M4", "S5", "S6", "B7", "S7")
	frPrepaidCadres     = set("B2", "S2", "M2")
	frFinalAfterDeposit = set("B4", "S4", "M4")
	frCreditNotes       = set("261", "381", "396", "502", "503")
	frCorrectives       = set("384", "471", "472", "473")
	frDeposits          = set("386", "500", "503")
	frSelfBilling       = set("389", "501", "500", "471", "473", "261", "502")
	frFlows             = set("B2B", "B2BINT", "B2C", "OUTOFSCOPE", "ARCHIVEONLY")
	frMandatoryNotes    = []string{"PMT", "PMD", "AAB"}
	frVATRates          = mustDecimals("0", "10", "13", "20", "8.5", "19.6", "2.1", "5.5", "7", "20.6", "1.05", "0.9", "1.75", "9.2", "9.6")
)

func set(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

func mustDecimals(values ...string) []decimal.Decimal {
	out := make([]decimal.Decimal, len(values))
	for i, v := range values {
		out[i] = decimal.RequireFromString(v)
	}
	return out
}

func isFrenchVATRate(rate decimal.Decimal) bool {
	for _, r := range frVATRates {
		if r.Equal(rate) {
			return true
		}
	}
	return false
}

// frDateInRange checks a CII format-102 date for a year in 2000-2099 (BR-FR-03).
func frDateInRange(date string) bool {
	return date == "" || strings.HasPrefix(date, "20")
}

func hasSIREN(c *invoice.Contact) bool {
	return c != nil && c.LegalIDScheme == sirenScheme && sirenRe.MatchString(c.LegalID)
}

// validateFrenchRules runs after the generic build so it can inspect the
// resolved data (normalised dates, countries, endpoints, notes, VAT groups).
func validateFrenchRules(doc *invoice.Document, opts Options, profile Profile, d *ciiData, sellerCountry string, t *totals) []error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format+": %w", append(args, ErrFrenchRules)...))
	}

	// Profile and parties.
	if profile != ProfileEN16931 && profile != ProfileExtended {
		fail("profile %s is not accepted, use EN 16931 or EXTENDED", profile)
	}
	if sellerCountry != "FR" {
		fail("seller country must be FR, got %q", sellerCountry)
	}
	if !hasSIREN(doc.Company) {
		fail("BR-FR-10: the company needs a 9-digit SIREN in LegalID with LegalIDScheme %q", sirenScheme)
	}

	flow := opts.frenchFlow()
	if !frFlows[flow] {
		fail("BR-FR-20: FrenchFlow %q must be one of B2B, B2BINT, B2C, OUTOFSCOPE, ARCHIVEONLY", flow)
	}
	typeCode := d.TypeCode
	if flow == "B2B" {
		if !hasSIREN(doc.Customer) {
			fail("BR-FR-11: a B2B flow needs a 9-digit SIREN in the customer's LegalID with LegalIDScheme %q", sirenScheme)
		}
		if frSelfBilling[typeCode] {
			if d.Seller.ElectronicAddressSchemeID != frEndpointScheme || !strings.HasPrefix(d.Seller.ElectronicAddress, doc.Company.LegalID) {
				fail("BR-FR-22: on a self-billed document the seller electronic address must start with the seller SIREN and use scheme %q", frEndpointScheme)
			}
		} else if hasSIREN(doc.Customer) && (d.Buyer.ElectronicAddressSchemeID != frEndpointScheme || !strings.HasPrefix(d.Buyer.ElectronicAddress, doc.Customer.LegalID)) {
			fail("BR-FR-21: the buyer electronic address must start with the buyer SIREN and use scheme %q", frEndpointScheme)
		}
	}

	// Electronic addresses are mandatory on both sides (BR-FR-12, BR-FR-13).
	for name, addr := range map[string]string{"seller": d.Seller.ElectronicAddress, "buyer": d.Buyer.ElectronicAddress} {
		switch {
		case addr == "":
			fail("BR-FR-12/13: %s electronic address is required", name)
		case !frIDRe.MatchString(addr):
			fail("BR-FR-23: %s electronic address %q may only contain letters, digits, + - _ /", name, addr)
		case len(addr) > 125:
			fail("BR-FR-25: %s electronic address exceeds 125 characters", name)
		}
	}

	// Document header.
	cadre := opts.BusinessProcessID
	switch {
	case cadre == "":
		fail("BR-FR-08: BusinessProcessID must carry the cadre de facturation (B1, S1, M1, B2, S2, M2, B4, S4, M4, S5, S6, B7, S7)")
	case !frCadres[cadre]:
		fail("BR-FR-08: cadre de facturation %q is not accepted", cadre)
	case frPrepaidCadres[cadre] && t != nil:
		// Already paid: the prepaid amount covers the total, nothing is due,
		// and the due date states the payment date.
		if t.due.Sign() != 0 || !t.prepaid.Equal(t.grand) {
			fail("BR-FR-CO-09: cadre %s (already paid) needs Prepaid equal to the total with tax, got prepaid %s for %s", cadre, t.prepaid, t.grand)
		}
		if d.PaymentDueDate == "" {
			fail("BR-FR-CO-09: cadre %s (already paid) needs DueDate set to the payment date", cadre)
		}
	case frFinalAfterDeposit[cadre] && frDeposits[typeCode]:
		fail("BR-FR-CO-08: cadre %s (final invoice after deposit) cannot carry deposit type %s", cadre, typeCode)
	}
	if !frTypeCodes[typeCode] {
		fail("BR-FR-04: document type code %q is not accepted", typeCode)
	}
	if len(doc.Ref) > 35 {
		fail("BR-FR-01: invoice number %q exceeds 35 characters", doc.Ref)
	}
	if !frIDRe.MatchString(doc.Ref) {
		fail("BR-FR-02: invoice number %q may only contain letters, digits, + - _ /", doc.Ref)
	}

	// Preceding invoice.
	if frCreditNotes[typeCode] || frCorrectives[typeCode] {
		if d.PrecedingInvoiceReference == "" || d.PrecedingInvoiceDate == "" {
			fail("BR-FR-CO-04/05: document type %s needs a PrecedingInvoice with its date", typeCode)
		}
	}
	if ref := d.PrecedingInvoiceReference; ref != "" {
		if len(ref) > 35 {
			fail("BR-FR-01: preceding invoice reference %q exceeds 35 characters", ref)
		}
		if !frIDRe.MatchString(ref) {
			fail("BR-FR-02: preceding invoice reference %q may only contain letters, digits, + - _ /", ref)
		}
	}

	// Dates.
	for name, date := range map[string]string{
		"document": d.IssueDate, "due": d.PaymentDueDate, "delivery": d.DeliveryDate,
		"period start": d.PeriodStart, "period end": d.PeriodEnd, "preceding invoice": d.PrecedingInvoiceDate,
	} {
		if !frDateInRange(date) {
			fail("BR-FR-03: %s date %s must be between 2000 and 2099", name, date)
		}
	}
	if d.PaymentDueDate != "" && d.IssueDate != "" && d.PaymentDueDate < d.IssueDate && !frDeposits[typeCode] {
		fail("BR-FR-CO-07: due date %s is before the document date %s", d.PaymentDueDate, d.IssueDate)
	}

	// Currency.
	if d.CurrencyCode != "EUR" && d.TaxCurrencyCode != "EUR" {
		fail("BR-FR-CO-12: an invoice in %s needs TaxCurrencyCode EUR with TaxCurrencyRate", d.CurrencyCode)
	}

	// Notes.
	counts := map[string]int{}
	for _, n := range d.Notes {
		counts[n.SubjectCode]++
	}
	for _, code := range frMandatoryNotes {
		switch counts[code] {
		case 0:
			fail("BR-FR-05: a note with subject code %s is required", code)
		case 1:
		default:
			fail("BR-FR-06: subject code %s must appear only once", code)
		}
	}
	if counts["TXD"] > 1 {
		fail("BR-FR-06: subject code TXD must appear only once")
	}
	if counts["BAR"] > 1 {
		fail("BR-FR-30: the BAR note is generated from FrenchFlow, do not add one")
	}

	// VAT.
	if t != nil {
		for _, g := range t.groups {
			if g.category == "L" || g.category == "M" {
				fail("BR-FR-15: VAT category %s is not relevant in France", g.category)
			}
			if g.hasRate && !isFrenchVATRate(g.percent) {
				fail("BR-FR-16: VAT rate %s is not a French rate", g.percent)
			}
		}
	}
	for _, item := range doc.Items {
		if qty, err := decimal.NewFromString(item.Quantity); err == nil && (qty.Sign() < 0 || qty.Exponent() < -4) {
			fail("BR-FR-DEC-02: quantity %q of item %q must be positive with at most 4 decimals", item.Quantity, item.Name)
		}
		if item.Tax != nil && item.Tax.Percent != "" {
			if p, err := decimal.NewFromString(item.Tax.Percent); err == nil && p.Exponent() < -2 {
				fail("BR-FR-DEC-04: VAT rate %q of item %q has more than 2 decimals", item.Tax.Percent, item.Name)
			}
		}
	}

	return errs
}
