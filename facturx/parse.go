package facturx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

// Reception side: parse a CII invoice back into an invoice.Document.

var hundred = decimal.NewFromInt(100)

var (
	// ErrNotCII is returned by Parse when the XML is not a CrossIndustryInvoice.
	ErrNotCII = errors.New("not a UN/CEFACT CrossIndustryInvoice")
	// ErrTotalsMismatch is returned by Parsed.Verify when the totals stated in
	// the XML do not match the totals recomputed from its lines.
	ErrTotalsMismatch = errors.New("stated totals do not match the recomputed totals")
)

// StatedTax is one VAT breakdown line as stated in the XML.
type StatedTax struct {
	Category            string
	Percent             decimal.Decimal
	HasRate             bool
	Basis               decimal.Decimal
	Tax                 decimal.Decimal
	ExemptionReason     string
	ExemptionReasonCode string
}

// StatedTotals are the document totals as stated in the XML.
type StatedTotals struct {
	LineTotal        decimal.Decimal
	ChargeTotal      decimal.Decimal
	AllowanceTotal   decimal.Decimal
	TaxBasis         decimal.Decimal
	Tax              decimal.Decimal
	TaxInTaxCurrency decimal.Decimal
	GrandTotal       decimal.Decimal
	Prepaid          decimal.Decimal
	DuePayable       decimal.Decimal
}

// Parsed is the result of Parse: the reconstructed document plus the
// Factur-X metadata and the amounts exactly as the sender stated them.
type Parsed struct {
	// Document is reconstructed from the XML. Items, charges, discount and
	// prepaid amount are filled in so that Breakdown recomputes the totals.
	Document *invoice.Document

	GuidelineID       string
	Profile           Profile
	BusinessProcessID string
	TypeCode          string
	VATPointDateCode  string
	TaxCurrencyCode   string
	// FrenchFlow is the content of the BAR note, if any (French documents).
	FrenchFlow string

	Totals StatedTotals
	Taxes  []StatedTax
}

// ---- XML mapping (local names, namespace-agnostic) ----

type cDate struct {
	Value string `xml:"DateTimeString"`
}

type cID struct {
	Value  string `xml:",chardata"`
	Scheme string `xml:"schemeID,attr"`
}

type cAmount struct {
	Value    string `xml:",chardata"`
	Currency string `xml:"currencyID,attr"`
}

type cQuantity struct {
	Value string `xml:",chardata"`
	Unit  string `xml:"unitCode,attr"`
}

type cNote struct {
	Content     string `xml:"Content"`
	SubjectCode string `xml:"SubjectCode"`
}

type cAddress struct {
	Postcode string `xml:"PostcodeCode"`
	LineOne  string `xml:"LineOne"`
	LineTwo  string `xml:"LineTwo"`
	City     string `xml:"CityName"`
	Country  string `xml:"CountryID"`
}

type cContact struct {
	Name  string `xml:"PersonName"`
	Phone string `xml:"TelephoneUniversalCommunication>CompleteNumber"`
	Email string `xml:"EmailURIUniversalCommunication>URIID"`
}

type cParty struct {
	Name     string    `xml:"Name"`
	LegalID  cID       `xml:"SpecifiedLegalOrganization>ID"`
	Contact  *cContact `xml:"DefinedTradeContact"`
	Address  *cAddress `xml:"PostalTradeAddress"`
	Endpoint cID       `xml:"URIUniversalCommunication>URIID"`
	TaxRegs  []cID     `xml:"SpecifiedTaxRegistration>ID"`
}

type cTradeTax struct {
	Calculated          string `xml:"CalculatedAmount"`
	ExemptionReason     string `xml:"ExemptionReason"`
	Basis               string `xml:"BasisAmount"`
	Category            string `xml:"CategoryCode"`
	ExemptionReasonCode string `xml:"ExemptionReasonCode"`
	DueDateTypeCode     string `xml:"DueDateTypeCode"`
	Rate                string `xml:"RateApplicablePercent"`
}

type cAllowanceCharge struct {
	ChargeIndicator string    `xml:"ChargeIndicator>Indicator"`
	ActualAmount    string    `xml:"ActualAmount"`
	ReasonCode      string    `xml:"ReasonCode"`
	Reason          string    `xml:"Reason"`
	Tax             cTradeTax `xml:"CategoryTradeTax"`
}

type cLine struct {
	ID           string             `xml:"AssociatedDocumentLineDocument>LineID"`
	Name         string             `xml:"SpecifiedTradeProduct>Name"`
	Description  string             `xml:"SpecifiedTradeProduct>Description"`
	GrossPrice   string             `xml:"SpecifiedLineTradeAgreement>GrossPriceProductTradePrice>ChargeAmount"`
	UnitDiscount string             `xml:"SpecifiedLineTradeAgreement>GrossPriceProductTradePrice>AppliedTradeAllowanceCharge>ActualAmount"`
	NetPrice     string             `xml:"SpecifiedLineTradeAgreement>NetPriceProductTradePrice>ChargeAmount"`
	Quantity     cQuantity          `xml:"SpecifiedLineTradeDelivery>BilledQuantity"`
	Tax          cTradeTax          `xml:"SpecifiedLineTradeSettlement>ApplicableTradeTax"`
	Allowances   []cAllowanceCharge `xml:"SpecifiedLineTradeSettlement>SpecifiedTradeAllowanceCharge"`
	LineTotal    string             `xml:"SpecifiedLineTradeSettlement>SpecifiedTradeSettlementLineMonetarySummation>LineTotalAmount"`
}

type cInvoice struct {
	XMLName         xml.Name
	BusinessProcess string             `xml:"ExchangedDocumentContext>BusinessProcessSpecifiedDocumentContextParameter>ID"`
	Guideline       string             `xml:"ExchangedDocumentContext>GuidelineSpecifiedDocumentContextParameter>ID"`
	ID              string             `xml:"ExchangedDocument>ID"`
	TypeCode        string             `xml:"ExchangedDocument>TypeCode"`
	IssueDate       cDate              `xml:"ExchangedDocument>IssueDateTime"`
	Notes           []cNote            `xml:"ExchangedDocument>IncludedNote"`
	Lines           []cLine            `xml:"SupplyChainTradeTransaction>IncludedSupplyChainTradeLineItem"`
	BuyerReference  string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerReference"`
	Seller          cParty             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty"`
	Buyer           cParty             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerTradeParty"`
	PurchaseOrder   string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerOrderReferencedDocument>IssuerAssignedID"`
	ShipTo          *cParty            `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ShipToTradeParty"`
	DeliveryDate    cDate              `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ActualDeliverySupplyChainEvent>OccurrenceDateTime"`
	PaymentRef      string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>PaymentReference"`
	TaxCurrency     string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>TaxCurrencyCode"`
	Currency        string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>InvoiceCurrencyCode"`
	PaymentMeans    string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementPaymentMeans>TypeCode"`
	IBAN            string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementPaymentMeans>PayeePartyCreditorFinancialAccount>IBANID"`
	BIC             string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementPaymentMeans>PayeeSpecifiedCreditorFinancialInstitution>BICID"`
	Taxes           []cTradeTax        `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>ApplicableTradeTax"`
	PeriodStart     cDate              `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>BillingSpecifiedPeriod>StartDateTime"`
	PeriodEnd       cDate              `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>BillingSpecifiedPeriod>EndDateTime"`
	AllowCharges    []cAllowanceCharge `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeAllowanceCharge"`
	PaymentTerms    string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradePaymentTerms>Description"`
	DueDate         cDate              `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradePaymentTerms>DueDateDateTime"`
	LineTotal       string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>LineTotalAmount"`
	ChargeTotal     string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>ChargeTotalAmount"`
	AllowanceTotal  string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>AllowanceTotalAmount"`
	TaxBasis        string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TaxBasisTotalAmount"`
	TaxTotals       []cAmount          `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TaxTotalAmount"`
	GrandTotal      string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>GrandTotalAmount"`
	Prepaid         string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TotalPrepaidAmount"`
	DuePayable      string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>DuePayableAmount"`
	PrecedingRef    string             `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>InvoiceReferencedDocument>IssuerAssignedID"`
	PrecedingDate   cDate              `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>InvoiceReferencedDocument>FormattedIssueDateTime"`
}

// Parse reads a Factur-X / ZUGFeRD 2 CII XML into an invoice.Document and
// the accompanying metadata. It accepts every profile; documents below BASIC
// carry no lines, in which case Document.Items is empty and only the stated
// totals are available.
func Parse(xmlBytes []byte) (*Parsed, error) {
	var c cInvoice
	if err := xml.Unmarshal(xmlBytes, &c); err != nil {
		return nil, fmt.Errorf("facturx: parse: %w", err)
	}
	if c.XMLName.Local != "CrossIndustryInvoice" {
		return nil, fmt.Errorf("facturx: %w: root element %q", ErrNotCII, c.XMLName.Local)
	}

	p := &Parsed{
		GuidelineID:       c.Guideline,
		Profile:           profileFromGuideline(c.Guideline),
		BusinessProcessID: c.BusinessProcess,
		TypeCode:          c.TypeCode,
		TaxCurrencyCode:   c.TaxCurrency,
	}

	docType := invoice.Invoice
	if frCreditNotes[c.TypeCode] {
		docType = invoice.CreditNote
	}
	doc, err := invoice.New(docType)
	if err != nil {
		return nil, err
	}
	p.Document = doc

	doc.Ref = c.ID
	doc.Date = c.IssueDate.Value
	doc.ClientRef = c.BuyerReference
	doc.PurchaseOrderRef = c.PurchaseOrder
	doc.DueDate = c.DueDate.Value
	doc.PaymentTerm = c.PaymentTerms
	if c.Currency != "" {
		doc.Currency = invoice.Currency{Code: strings.ToUpper(c.Currency), Precision: 2}
	}

	for _, n := range c.Notes {
		if n.SubjectCode == "BAR" {
			p.FrenchFlow = strings.TrimSpace(n.Content)
			continue
		}
		doc.AddNote(n.Content, n.SubjectCode)
	}

	doc.Company = parseParty(&c.Seller)
	doc.Customer = parseParty(&c.Buyer)

	if c.ShipTo != nil || c.DeliveryDate.Value != "" || c.PeriodStart.Value != "" {
		doc.Delivery = &invoice.Delivery{
			Date:        c.DeliveryDate.Value,
			PeriodStart: c.PeriodStart.Value,
			PeriodEnd:   c.PeriodEnd.Value,
		}
		if c.ShipTo != nil {
			doc.Delivery.ShipTo = parseParty(c.ShipTo)
		}
	}

	if c.IBAN != "" || c.BIC != "" || c.PaymentMeans != "" || c.PaymentRef != "" {
		doc.Payment = &invoice.Payment{IBAN: c.IBAN, BIC: c.BIC, MeansCode: c.PaymentMeans, Reference: c.PaymentRef}
	}

	if c.PrecedingRef != "" {
		doc.PrecedingInvoice = &invoice.DocumentRef{Ref: c.PrecedingRef, Date: c.PrecedingDate.Value}
	}

	// VAT breakdown and the exemption reasons it carries per category/rate.
	reasons := map[string]cTradeTax{}
	for _, tx := range c.Taxes {
		st := StatedTax{
			Category:            tx.Category,
			Basis:               parseDec(tx.Basis),
			Tax:                 parseDec(tx.Calculated),
			ExemptionReason:     tx.ExemptionReason,
			ExemptionReasonCode: tx.ExemptionReasonCode,
		}
		if tx.Rate != "" {
			st.HasRate = true
			st.Percent = parseDec(tx.Rate)
		}
		p.Taxes = append(p.Taxes, st)
		reasons[tx.Category+"|"+st.Percent.String()] = tx
		if p.VATPointDateCode == "" {
			p.VATPointDateCode = tx.DueDateTypeCode
		}
	}

	// Lines.
	for _, l := range c.Lines {
		item := &invoice.Item{
			Name:        l.Name,
			Description: l.Description,
			Quantity:    l.Quantity.Value,
			Unit:        l.Quantity.Unit,
			Tax:         parseTax(l.Tax, reasons),
		}
		qty := parseDec(l.Quantity.Value)
		if l.GrossPrice != "" && l.UnitDiscount != "" {
			item.UnitCost = l.GrossPrice
			if disc := parseDec(l.UnitDiscount).Mul(qty).Round(2); disc.Sign() != 0 {
				item.Discount = &invoice.Discount{Amount: disc.String()}
			}
		} else {
			item.UnitCost = l.NetPrice
		}
		doc.AppendItem(item)
	}

	// Document-level allowances become the document discount; charges are
	// kept individually.
	var discount decimal.Decimal
	for _, ac := range c.AllowCharges {
		amount := parseDec(ac.ActualAmount)
		if strings.EqualFold(strings.TrimSpace(ac.ChargeIndicator), "true") {
			doc.AddCharge(&invoice.Charge{
				Name:       ac.Reason,
				Amount:     amount.String(),
				ReasonCode: ac.ReasonCode,
				Tax:        parseTax(ac.Tax, reasons),
			})
			continue
		}
		discount = discount.Add(amount)
	}
	if discount.Sign() != 0 {
		doc.Discount = &invoice.Discount{Amount: discount.String()}
	}

	if c.Prepaid != "" {
		doc.Prepaid = c.Prepaid
	}

	p.Totals = StatedTotals{
		LineTotal:      parseDec(c.LineTotal),
		ChargeTotal:    parseDec(c.ChargeTotal),
		AllowanceTotal: parseDec(c.AllowanceTotal),
		TaxBasis:       parseDec(c.TaxBasis),
		GrandTotal:     parseDec(c.GrandTotal),
		Prepaid:        parseDec(c.Prepaid),
		DuePayable:     parseDec(c.DuePayable),
	}
	for _, tt := range c.TaxTotals {
		if tt.Currency == "" || strings.EqualFold(tt.Currency, c.Currency) {
			p.Totals.Tax = parseDec(tt.Value)
		} else {
			p.Totals.TaxInTaxCurrency = parseDec(tt.Value)
		}
	}

	return p, nil
}

// Verify checks the stated totals against the totals recomputed from the
// parsed document. Below BASIC there are no lines, so only the internal
// identities between the stated totals are checked. Per-group VAT may differ
// by one cent from a sender that split the discount differently, which
// EN 16931 tolerates, so the tax total is compared with that tolerance.
func (p *Parsed) Verify() error {
	s := p.Totals
	var errs []error
	mismatch := func(what string, stated, computed decimal.Decimal) {
		errs = append(errs, fmt.Errorf("%s: stated %s, computed %s", what, stated, computed))
	}

	if !s.GrandTotal.Equal(s.TaxBasis.Add(s.Tax)) {
		mismatch("grand total = tax basis + tax", s.GrandTotal, s.TaxBasis.Add(s.Tax))
	}
	if !s.DuePayable.Equal(s.GrandTotal.Sub(s.Prepaid)) {
		mismatch("amount due = grand total - prepaid", s.DuePayable, s.GrandTotal.Sub(s.Prepaid))
	}

	if len(p.Document.Items) > 0 {
		if err := p.Document.Validate(); err != nil {
			return fmt.Errorf("facturx: %w", err)
		}
		bd := p.Document.Breakdown()
		if !bd.LineTotal.Equal(s.LineTotal) {
			mismatch("line total", s.LineTotal, bd.LineTotal)
		}
		if !bd.ChargeTotal.Equal(s.ChargeTotal) {
			mismatch("charge total", s.ChargeTotal, bd.ChargeTotal)
		}
		if !bd.Allowance.Equal(s.AllowanceTotal) {
			mismatch("allowance total", s.AllowanceTotal, bd.Allowance)
		}
		if !bd.TaxBasis.Equal(s.TaxBasis) {
			mismatch("tax basis", s.TaxBasis, bd.TaxBasis)
		}
		tolerance := decimal.NewFromInt(int64(len(bd.Groups))).Div(hundred)
		if bd.Tax.Sub(s.Tax).Abs().Cmp(tolerance) > 0 {
			mismatch("tax", s.Tax, bd.Tax)
		}
		if bd.GrandTotal.Sub(s.GrandTotal).Abs().Cmp(tolerance) > 0 {
			mismatch("grand total", s.GrandTotal, bd.GrandTotal)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("facturx: %w: %w", ErrTotalsMismatch, errors.Join(errs...))
	}
	return nil
}

func parseParty(c *cParty) *invoice.Contact {
	ct := &invoice.Contact{
		Name:                    c.Name,
		LegalID:                 c.LegalID.Value,
		LegalIDScheme:           c.LegalID.Scheme,
		ElectronicAddress:       c.Endpoint.Value,
		ElectronicAddressScheme: c.Endpoint.Scheme,
	}
	for _, reg := range c.TaxRegs {
		if strings.EqualFold(reg.Scheme, "VA") || ct.VATID == "" {
			ct.VATID = reg.Value
		}
	}
	if c.Contact != nil {
		ct.ContactName = c.Contact.Name
		ct.ContactPhone = c.Contact.Phone
		ct.ContactEmail = c.Contact.Email
	}
	if c.Address != nil {
		ct.Address = &invoice.Address{
			Address:     c.Address.LineOne,
			Address2:    c.Address.LineTwo,
			PostalCode:  c.Address.Postcode,
			City:        c.Address.City,
			Country:     c.Address.Country,
			CountryCode: c.Address.Country,
		}
	}
	return ct
}

// parseTax rebuilds an invoice.Tax from a line or charge VAT element, taking
// the exemption reason from the matching header breakdown line.
func parseTax(t cTradeTax, reasons map[string]cTradeTax) *invoice.Tax {
	if t.Category == "" && t.Rate == "" {
		return nil
	}
	rate := decimal.Zero
	if t.Rate != "" {
		rate = parseDec(t.Rate)
	}
	tax := &invoice.Tax{Percent: rate.String(), Category: t.Category}
	if hdr, ok := reasons[t.Category+"|"+rate.String()]; ok {
		tax.ExemptionReason = hdr.ExemptionReason
		tax.ExemptionReasonCode = hdr.ExemptionReasonCode
	}
	return tax
}

func profileFromGuideline(id string) Profile {
	l := strings.ToLower(id)
	switch {
	case strings.Contains(l, "minimum"):
		return ProfileMinimum
	case strings.Contains(l, "basicwl"):
		return ProfileBasicWL
	case strings.Contains(l, "basic"):
		return ProfileBasic
	case strings.Contains(l, "extended"):
		return ProfileExtended
	default:
		return ProfileEN16931
	}
}

// parseDec parses an amount, treating empty or malformed values as zero.
func parseDec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero
	}
	return d
}
