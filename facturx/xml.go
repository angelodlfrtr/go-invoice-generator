package facturx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
	"github.com/shopspring/decimal"
)

// ciiXMLTemplate renders a UN/CEFACT CII D16B invoice. Element order follows
// the Factur-X 1.0.07 profile XSDs; profile gating is done in buildCIIData by
// leaving fields empty.
const ciiXMLTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:qdt="urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocumentContext>
		{{- if .BusinessProcessID}}
		<ram:BusinessProcessSpecifiedDocumentContextParameter>
			<ram:ID>{{xe .BusinessProcessID}}</ram:ID>
		</ram:BusinessProcessSpecifiedDocumentContextParameter>
		{{- end}}
		<ram:GuidelineSpecifiedDocumentContextParameter>
			<ram:ID>{{.GuidelineID}}</ram:ID>
		</ram:GuidelineSpecifiedDocumentContextParameter>
	</rsm:ExchangedDocumentContext>
	<rsm:ExchangedDocument>
		<ram:ID>{{xe .ID}}</ram:ID>
		<ram:TypeCode>{{.TypeCode}}</ram:TypeCode>
		<ram:IssueDateTime>
			<udt:DateTimeString format="102">{{.IssueDate}}</udt:DateTimeString>
		</ram:IssueDateTime>
		{{- range .Notes}}
		<ram:IncludedNote>
			<ram:Content>{{xe .Content}}</ram:Content>
			{{- if .SubjectCode}}
			<ram:SubjectCode>{{xe .SubjectCode}}</ram:SubjectCode>
			{{- end}}
		</ram:IncludedNote>
		{{- end}}
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>
		{{- range .LineItems}}
		<ram:IncludedSupplyChainTradeLineItem>
			<ram:AssociatedDocumentLineDocument>
				<ram:LineID>{{.LineID}}</ram:LineID>
			</ram:AssociatedDocumentLineDocument>
			<ram:SpecifiedTradeProduct>
				<ram:Name>{{xe .Name}}</ram:Name>
				{{- if .Description}}
				<ram:Description>{{xe .Description}}</ram:Description>
				{{- end}}
			</ram:SpecifiedTradeProduct>
			<ram:SpecifiedLineTradeAgreement>
				{{- if .GrossUnitPrice}}
				<ram:GrossPriceProductTradePrice>
					<ram:ChargeAmount>{{.GrossUnitPrice}}</ram:ChargeAmount>
					<ram:AppliedTradeAllowanceCharge>
						<ram:ChargeIndicator><udt:Indicator>false</udt:Indicator></ram:ChargeIndicator>
						<ram:ActualAmount>{{.UnitDiscount}}</ram:ActualAmount>
					</ram:AppliedTradeAllowanceCharge>
				</ram:GrossPriceProductTradePrice>
				{{- end}}
				<ram:NetPriceProductTradePrice>
					<ram:ChargeAmount>{{.NetUnitPrice}}</ram:ChargeAmount>
				</ram:NetPriceProductTradePrice>
			</ram:SpecifiedLineTradeAgreement>
			<ram:SpecifiedLineTradeDelivery>
				<ram:BilledQuantity unitCode="{{xe .UnitCode}}">{{.Quantity}}</ram:BilledQuantity>
			</ram:SpecifiedLineTradeDelivery>
			<ram:SpecifiedLineTradeSettlement>
				<ram:ApplicableTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:CategoryCode>{{.TaxCategoryCode}}</ram:CategoryCode>
					{{- if .HasTaxRate}}
					<ram:RateApplicablePercent>{{.TaxPercent}}</ram:RateApplicablePercent>
					{{- end}}
				</ram:ApplicableTradeTax>
				<ram:SpecifiedTradeSettlementLineMonetarySummation>
					<ram:LineTotalAmount>{{.LineTotal}}</ram:LineTotalAmount>
				</ram:SpecifiedTradeSettlementLineMonetarySummation>
			</ram:SpecifiedLineTradeSettlement>
		</ram:IncludedSupplyChainTradeLineItem>
		{{- end}}
		<ram:ApplicableHeaderTradeAgreement>
			{{- if .BuyerReference}}
			<ram:BuyerReference>{{xe .BuyerReference}}</ram:BuyerReference>
			{{- end}}
			<ram:SellerTradeParty>
				{{- template "party" .Seller}}
			</ram:SellerTradeParty>
			<ram:BuyerTradeParty>
				{{- template "party" .Buyer}}
			</ram:BuyerTradeParty>
			{{- if .PurchaseOrderReference}}
			<ram:BuyerOrderReferencedDocument>
				<ram:IssuerAssignedID>{{xe .PurchaseOrderReference}}</ram:IssuerAssignedID>
			</ram:BuyerOrderReferencedDocument>
			{{- end}}
		</ram:ApplicableHeaderTradeAgreement>
		{{- if or .ShipTo .DeliveryDate}}
		<ram:ApplicableHeaderTradeDelivery>
			{{- if .ShipTo}}
			<ram:ShipToTradeParty>
				{{- template "party" .ShipTo}}
			</ram:ShipToTradeParty>
			{{- end}}
			{{- if .DeliveryDate}}
			<ram:ActualDeliverySupplyChainEvent>
				<ram:OccurrenceDateTime>
					<udt:DateTimeString format="102">{{.DeliveryDate}}</udt:DateTimeString>
				</ram:OccurrenceDateTime>
			</ram:ActualDeliverySupplyChainEvent>
			{{- end}}
		</ram:ApplicableHeaderTradeDelivery>
		{{- else}}
		<ram:ApplicableHeaderTradeDelivery/>
		{{- end}}
		<ram:ApplicableHeaderTradeSettlement>
			{{- if .PaymentReference}}
			<ram:PaymentReference>{{xe .PaymentReference}}</ram:PaymentReference>
			{{- end}}
			{{- if .TaxCurrencyCode}}
			<ram:TaxCurrencyCode>{{.TaxCurrencyCode}}</ram:TaxCurrencyCode>
			{{- end}}
			<ram:InvoiceCurrencyCode>{{.CurrencyCode}}</ram:InvoiceCurrencyCode>
			{{- if .PaymentMeansCode}}
			<ram:SpecifiedTradeSettlementPaymentMeans>
				<ram:TypeCode>{{.PaymentMeansCode}}</ram:TypeCode>
				{{- if .PaymentIBAN}}
				<ram:PayeePartyCreditorFinancialAccount>
					<ram:IBANID>{{xe .PaymentIBAN}}</ram:IBANID>
				</ram:PayeePartyCreditorFinancialAccount>
				{{- end}}
				{{- if .PaymentBIC}}
				<ram:PayeeSpecifiedCreditorFinancialInstitution>
					<ram:BICID>{{xe .PaymentBIC}}</ram:BICID>
				</ram:PayeeSpecifiedCreditorFinancialInstitution>
				{{- end}}
			</ram:SpecifiedTradeSettlementPaymentMeans>
			{{- end}}
			{{- range .TaxBreakdown}}
			<ram:ApplicableTradeTax>
				<ram:CalculatedAmount>{{.TaxAmount}}</ram:CalculatedAmount>
				<ram:TypeCode>VAT</ram:TypeCode>
				{{- if .ExemptionReason}}
				<ram:ExemptionReason>{{xe .ExemptionReason}}</ram:ExemptionReason>
				{{- end}}
				<ram:BasisAmount>{{.BasisAmount}}</ram:BasisAmount>
				<ram:CategoryCode>{{.CategoryCode}}</ram:CategoryCode>
				{{- if .ExemptionReasonCode}}
				<ram:ExemptionReasonCode>{{xe .ExemptionReasonCode}}</ram:ExemptionReasonCode>
				{{- end}}
				{{- if $.VATPointDateCode}}
				<ram:DueDateTypeCode>{{$.VATPointDateCode}}</ram:DueDateTypeCode>
				{{- end}}
				{{- if .HasRate}}
				<ram:RateApplicablePercent>{{.Percent}}</ram:RateApplicablePercent>
				{{- end}}
			</ram:ApplicableTradeTax>
			{{- end}}
			{{- if .PeriodStart}}
			<ram:BillingSpecifiedPeriod>
				<ram:StartDateTime>
					<udt:DateTimeString format="102">{{.PeriodStart}}</udt:DateTimeString>
				</ram:StartDateTime>
				<ram:EndDateTime>
					<udt:DateTimeString format="102">{{.PeriodEnd}}</udt:DateTimeString>
				</ram:EndDateTime>
			</ram:BillingSpecifiedPeriod>
			{{- end}}
			{{- range .DocAllowances}}
			<ram:SpecifiedTradeAllowanceCharge>
				<ram:ChargeIndicator><udt:Indicator>false</udt:Indicator></ram:ChargeIndicator>
				<ram:ActualAmount>{{.ActualAmount}}</ram:ActualAmount>
				<ram:ReasonCode>95</ram:ReasonCode>
				<ram:Reason>{{xe .Reason}}</ram:Reason>
				<ram:CategoryTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:CategoryCode>{{.CategoryCode}}</ram:CategoryCode>
					{{- if .HasRate}}
					<ram:RateApplicablePercent>{{.Percent}}</ram:RateApplicablePercent>
					{{- end}}
				</ram:CategoryTradeTax>
			</ram:SpecifiedTradeAllowanceCharge>
			{{- end}}
			{{- range .DocCharges}}
			<ram:SpecifiedTradeAllowanceCharge>
				<ram:ChargeIndicator><udt:Indicator>true</udt:Indicator></ram:ChargeIndicator>
				<ram:ActualAmount>{{.ActualAmount}}</ram:ActualAmount>
				{{- if .ReasonCode}}
				<ram:ReasonCode>{{xe .ReasonCode}}</ram:ReasonCode>
				{{- end}}
				<ram:Reason>{{xe .Reason}}</ram:Reason>
				<ram:CategoryTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:CategoryCode>{{.CategoryCode}}</ram:CategoryCode>
					{{- if .HasRate}}
					<ram:RateApplicablePercent>{{.Percent}}</ram:RateApplicablePercent>
					{{- end}}
				</ram:CategoryTradeTax>
			</ram:SpecifiedTradeAllowanceCharge>
			{{- end}}
			{{- if or .PaymentTermsDescription .PaymentDueDate}}
			<ram:SpecifiedTradePaymentTerms>
				{{- if .PaymentTermsDescription}}
				<ram:Description>{{xe .PaymentTermsDescription}}</ram:Description>
				{{- end}}
				{{- if .PaymentDueDate}}
				<ram:DueDateDateTime>
					<udt:DateTimeString format="102">{{.PaymentDueDate}}</udt:DateTimeString>
				</ram:DueDateDateTime>
				{{- end}}
			</ram:SpecifiedTradePaymentTerms>
			{{- end}}
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				{{- if .LineTotalAmount}}
				<ram:LineTotalAmount>{{.LineTotalAmount}}</ram:LineTotalAmount>
				{{- end}}
				{{- if .ChargeTotalAmount}}
				<ram:ChargeTotalAmount>{{.ChargeTotalAmount}}</ram:ChargeTotalAmount>
				{{- end}}
				{{- if .AllowanceTotalAmount}}
				<ram:AllowanceTotalAmount>{{.AllowanceTotalAmount}}</ram:AllowanceTotalAmount>
				{{- end}}
				<ram:TaxBasisTotalAmount>{{.TaxBasisTotalAmount}}</ram:TaxBasisTotalAmount>
				<ram:TaxTotalAmount currencyID="{{.CurrencyCode}}">{{.TaxTotalAmount}}</ram:TaxTotalAmount>
				{{- if .TaxCurrencyCode}}
				<ram:TaxTotalAmount currencyID="{{.TaxCurrencyCode}}">{{.TaxTotalInTaxCurrency}}</ram:TaxTotalAmount>
				{{- end}}
				<ram:GrandTotalAmount>{{.GrandTotalAmount}}</ram:GrandTotalAmount>
				{{- if .PrepaidAmount}}
				<ram:TotalPrepaidAmount>{{.PrepaidAmount}}</ram:TotalPrepaidAmount>
				{{- end}}
				<ram:DuePayableAmount>{{.DuePayableAmount}}</ram:DuePayableAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
			{{- if .PrecedingInvoiceReference}}
			<ram:InvoiceReferencedDocument>
				<ram:IssuerAssignedID>{{xe .PrecedingInvoiceReference}}</ram:IssuerAssignedID>
				{{- if .PrecedingInvoiceDate}}
				<ram:FormattedIssueDateTime>
					<qdt:DateTimeString format="102">{{.PrecedingInvoiceDate}}</qdt:DateTimeString>
				</ram:FormattedIssueDateTime>
				{{- end}}
			</ram:InvoiceReferencedDocument>
			{{- end}}
		</ram:ApplicableHeaderTradeSettlement>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>
{{- define "party"}}
				<ram:Name>{{xe .Name}}</ram:Name>
				{{- if .LegalID}}
				<ram:SpecifiedLegalOrganization>
					<ram:ID{{if .LegalIDSchemeID}} schemeID="{{xe .LegalIDSchemeID}}"{{end}}>{{xe .LegalID}}</ram:ID>
				</ram:SpecifiedLegalOrganization>
				{{- end}}
				{{- if .Contact}}
				<ram:DefinedTradeContact>
					{{- if .Contact.Name}}
					<ram:PersonName>{{xe .Contact.Name}}</ram:PersonName>
					{{- end}}
					{{- if .Contact.Phone}}
					<ram:TelephoneUniversalCommunication>
						<ram:CompleteNumber>{{xe .Contact.Phone}}</ram:CompleteNumber>
					</ram:TelephoneUniversalCommunication>
					{{- end}}
					{{- if .Contact.Email}}
					<ram:EmailURIUniversalCommunication>
						<ram:URIID>{{xe .Contact.Email}}</ram:URIID>
					</ram:EmailURIUniversalCommunication>
					{{- end}}
				</ram:DefinedTradeContact>
				{{- end}}
				{{- if .Address}}
				<ram:PostalTradeAddress>
					{{- if .Address.PostalCode}}
					<ram:PostcodeCode>{{xe .Address.PostalCode}}</ram:PostcodeCode>
					{{- end}}
					{{- if .Address.Address}}
					<ram:LineOne>{{xe .Address.Address}}</ram:LineOne>
					{{- end}}
					{{- if .Address.Address2}}
					<ram:LineTwo>{{xe .Address.Address2}}</ram:LineTwo>
					{{- end}}
					{{- if .Address.City}}
					<ram:CityName>{{xe .Address.City}}</ram:CityName>
					{{- end}}
					<ram:CountryID>{{xe .Address.Country}}</ram:CountryID>
				</ram:PostalTradeAddress>
				{{- end}}
				{{- if .ElectronicAddress}}
				<ram:URIUniversalCommunication>
					<ram:URIID schemeID="{{xe .ElectronicAddressSchemeID}}">{{xe .ElectronicAddress}}</ram:URIID>
				</ram:URIUniversalCommunication>
				{{- end}}
				{{- if .TaxID}}
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">{{xe .TaxID}}</ram:ID>
				</ram:SpecifiedTaxRegistration>
				{{- end}}
{{- end}}`

type ciiAddress struct {
	Address    string
	Address2   string
	PostalCode string
	City       string
	Country    string
}

type ciiContact struct {
	Name  string
	Phone string
	Email string
}

type ciiParty struct {
	Name                      string
	LegalID                   string
	LegalIDSchemeID           string
	Contact                   *ciiContact
	Address                   *ciiAddress
	ElectronicAddress         string
	ElectronicAddressSchemeID string
	TaxID                     string
}

type ciiTaxLine struct {
	TaxAmount           string
	BasisAmount         string
	CategoryCode        string
	ExemptionReason     string
	ExemptionReasonCode string
	HasRate             bool
	Percent             string
}

type ciiAllowanceCharge struct {
	ActualAmount string
	Reason       string
	ReasonCode   string
	CategoryCode string
	HasRate      bool
	Percent      string
}

type ciiLineItem struct {
	LineID          string
	Name            string
	Description     string
	GrossUnitPrice  string // set when the item has a discount
	UnitDiscount    string // per-unit allowance, set with GrossUnitPrice
	NetUnitPrice    string
	Quantity        string
	UnitCode        string
	TaxCategoryCode string
	HasTaxRate      bool
	TaxPercent      string
	LineTotal       string
}

type ciiData struct {
	GuidelineID               string
	BusinessProcessID         string
	TypeCode                  string
	ID                        string
	IssueDate                 string
	Notes                     []Note
	BuyerReference            string
	PurchaseOrderReference    string
	Seller                    ciiParty
	Buyer                     ciiParty
	ShipTo                    *ciiParty
	DeliveryDate              string
	CurrencyCode              string
	TaxCurrencyCode           string
	TaxTotalInTaxCurrency     string
	VATPointDateCode          string
	PrecedingInvoiceReference string
	PrecedingInvoiceDate      string
	PaymentReference          string
	PaymentMeansCode          string
	PaymentIBAN               string
	PaymentBIC                string
	TaxBreakdown              []ciiTaxLine
	PeriodStart               string
	PeriodEnd                 string
	DocAllowances             []ciiAllowanceCharge
	DocCharges                []ciiAllowanceCharge
	PaymentTermsDescription   string
	PaymentDueDate            string
	LineTotalAmount           string
	ChargeTotalAmount         string
	AllowanceTotalAmount      string
	TaxBasisTotalAmount       string
	TaxTotalAmount            string
	GrandTotalAmount          string
	PrepaidAmount             string
	DuePayableAmount          string
	LineItems                 []ciiLineItem
}

var ciiTmpl = template.Must(
	template.New("cii").Funcs(template.FuncMap{
		"xe": xmlEsc,
	}).Parse(ciiXMLTemplate),
)

func xmlEsc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// BuildXML generates the CII XML bytes for the given document and options.
// doc.Validate() (called by the generator's Build) must have run before.
//
// The document and options are validated against the EN 16931 business rules
// that can be checked up front; see the Err* sentinels for what is rejected.
func BuildXML(doc *invoice.Document, opts Options) ([]byte, error) {
	data, err := buildCIIData(doc, opts)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := ciiTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("facturx: XML template: %w", err)
	}

	return []byte(strings.TrimSpace(buf.String())), nil
}

func buildCIIData(doc *invoice.Document, opts Options) (*ciiData, error) {
	profile := opts.profile()
	minimum := profile == ProfileMinimum
	basicWLPlus := !minimum
	basicPlus := basicWLPlus && profile != ProfileBasicWL
	en16931Plus := profile == ProfileEN16931 || profile == ProfileExtended

	var errs []error
	fail := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}

	// ---- Header ----

	issueDate, err := formatDate("document date", doc.Date)
	fail(err)

	if opts.TypeCode == "" && doc.Type != invoice.Invoice && doc.Type != invoice.CreditNote {
		fail(fmt.Errorf("%s: %w", doc.Type, ErrUnsupportedDocumentType))
	}

	currency := strings.ToUpper(strings.TrimSpace(doc.Currency.Code))
	if !currencyCodeRe.MatchString(currency) {
		fail(fmt.Errorf("currency %q: %w", currency, ErrInvalidCurrencyCode))
	}

	d := &ciiData{
		GuidelineID:            profile.guidelineID(),
		BusinessProcessID:      opts.BusinessProcessID,
		TypeCode:               opts.typeCode(doc),
		ID:                     doc.Ref,
		IssueDate:              issueDate,
		BuyerReference:         doc.ClientRef,
		PurchaseOrderReference: doc.PurchaseOrderRef,
		CurrencyCode:           currency,
	}

	if basicWLPlus {
		d.Notes = append(d.Notes, doc.Notes...)
		if opts.FrenchRules {
			d.Notes = append(d.Notes, Note{Content: opts.frenchFlow(), SubjectCode: "BAR"})
		}
	}

	// ---- Parties ----

	if doc.Company.VATID == "" && doc.Company.LegalID == "" {
		fail(ErrMissingSellerID)
	}

	sellerCountry, err := normalizeCountry("seller country", doc.Company.Address.ISOCountry(), true)
	fail(err)
	d.Seller = buildParty(doc.Company, sellerCountry, opts, minimum, basicWLPlus, en16931Plus)

	// Buyer address is optional in MINIMUM, mandatory above (BR-10, BR-11).
	buyerCountry, err := normalizeCountry("buyer country", doc.Customer.Address.ISOCountry(), basicWLPlus)
	fail(err)
	d.Buyer = buildParty(doc.Customer, buyerCountry, opts, minimum, basicWLPlus, false)
	if !basicWLPlus {
		d.Buyer.TaxID = ""
	}

	// ---- Delivery, period, payment (BASIC-WL and above) ----

	if basicWLPlus {
		if del := doc.Delivery; del != nil {
			d.DeliveryDate, err = optionalDate("delivery date", del.Date)
			fail(err)

			if (del.PeriodStart == "") != (del.PeriodEnd == "") {
				fail(ErrInvalidPeriod)
			}
			d.PeriodStart, err = optionalDate("invoicing period start", del.PeriodStart)
			fail(err)
			d.PeriodEnd, err = optionalDate("invoicing period end", del.PeriodEnd)
			fail(err)

			if del.ShipTo != nil {
				shipToCountry, cerr := normalizeCountry("ship-to country", del.ShipTo.Address.ISOCountry(), true)
				fail(cerr)
				d.ShipTo = &ciiParty{
					Name:    del.ShipTo.Name,
					Address: buildAddress(del.ShipTo.Address, shipToCountry, false),
				}
			}
		}

		if opts.VATPointDateCode != "" {
			if !vatPointDateCodes[opts.VATPointDateCode] {
				fail(fmt.Errorf("%q: %w", opts.VATPointDateCode, ErrInvalidVATPointDateCode))
			}
			d.VATPointDateCode = opts.VATPointDateCode
		}

		if opts.TaxCurrencyCode != "" {
			d.TaxCurrencyCode = strings.ToUpper(strings.TrimSpace(opts.TaxCurrencyCode))
			switch {
			case !currencyCodeRe.MatchString(d.TaxCurrencyCode):
				fail(fmt.Errorf("tax currency %q: %w", d.TaxCurrencyCode, ErrInvalidTaxCurrency))
			case d.TaxCurrencyCode == currency:
				fail(fmt.Errorf("tax currency equals the invoice currency %s: %w", currency, ErrInvalidTaxCurrency))
			}
			if rate, rerr := decimal.NewFromString(opts.TaxCurrencyRate); rerr != nil || rate.Sign() <= 0 {
				fail(fmt.Errorf("tax currency rate %q must be a positive number: %w", opts.TaxCurrencyRate, ErrInvalidTaxCurrency))
			}
		}

		if ref := doc.PrecedingInvoice; ref != nil && ref.Ref != "" {
			d.PrecedingInvoiceReference = ref.Ref
			d.PrecedingInvoiceDate, err = optionalDate("preceding invoice date", ref.Date)
			fail(err)
		}

		if p := doc.Payment; p != nil {
			d.PaymentReference = p.Reference
			d.PaymentMeansCode = paymentMeansCode(p)
			d.PaymentIBAN = p.IBAN
			if en16931Plus {
				d.PaymentBIC = p.BIC
			}
		}

		d.PaymentTermsDescription = doc.PaymentTermsText()
		if due, ok := doc.EffectiveDueDate(); ok {
			d.PaymentDueDate, err = formatDate("payment due date", due)
			fail(err)
		}
	}

	// ---- Lines and totals ----

	totals, err := computeTotals(doc, opts, profile)
	fail(err)

	if opts.FrenchRules {
		errs = append(errs, validateFrenchRules(doc, opts, profile, d, sellerCountry, totals)...)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("facturx: %w", errors.Join(errs...))
	}

	d.TaxBasisTotalAmount = totals.taxBasis.StringFixed(2)
	d.TaxTotalAmount = totals.tax.StringFixed(2)
	d.GrandTotalAmount = totals.grand.StringFixed(2)
	d.DuePayableAmount = totals.due.StringFixed(2)
	if d.TaxCurrencyCode != "" {
		rate, _ := decimal.NewFromString(opts.TaxCurrencyRate)
		d.TaxTotalInTaxCurrency = totals.tax.Mul(rate).Round(2).StringFixed(2)
	}

	if basicWLPlus {
		d.LineTotalAmount = totals.lineTotal.StringFixed(2)
		if totals.allowanceTotal.Sign() != 0 {
			d.AllowanceTotalAmount = totals.allowanceTotal.StringFixed(2)
		}
		if totals.chargeTotal.Sign() != 0 {
			d.ChargeTotalAmount = totals.chargeTotal.StringFixed(2)
		}
		if totals.prepaid.Sign() != 0 {
			d.PrepaidAmount = totals.prepaid.StringFixed(2)
		}
		d.DocCharges = totals.charges
		for _, g := range totals.groups {
			d.TaxBreakdown = append(d.TaxBreakdown, ciiTaxLine{
				TaxAmount:           g.tax.StringFixed(2),
				BasisAmount:         g.basis().StringFixed(2),
				CategoryCode:        g.category,
				ExemptionReason:     g.exemptionReason,
				ExemptionReasonCode: g.exemptionReasonCode,
				HasRate:             g.hasRate,
				Percent:             g.percent.StringFixed(2),
			})
			if g.allowance.Sign() != 0 {
				d.DocAllowances = append(d.DocAllowances, ciiAllowanceCharge{
					ActualAmount: g.allowance.StringFixed(2),
					Reason:       opts.documentDiscountReason(),
					CategoryCode: g.category,
					HasRate:      g.hasRate,
					Percent:      g.percent.StringFixed(2),
				})
			}
		}

		if totals.due.Sign() > 0 && d.PaymentDueDate == "" && d.PaymentTermsDescription == "" {
			return nil, fmt.Errorf("facturx: %w", ErrMissingPaymentTerms)
		}
	}

	if basicPlus {
		d.LineItems = buildLineItems(doc, opts, totals.lines, en16931Plus)
	}

	return d, nil
}

// buildParty maps a contact. MINIMUM carries name, legal ID, country and VAT
// ID; BASIC-WL adds the address and electronic address; EN 16931 adds the
// contact person (seller only).
func buildParty(c *invoice.Contact, country string, opts Options, minimum, basicWLPlus, withContact bool) ciiParty {
	p := ciiParty{
		Name:            c.Name,
		LegalID:         c.LegalID,
		LegalIDSchemeID: c.LegalIDScheme,
		TaxID:           c.VATID,
		Address:         buildAddress(c.Address, country, minimum),
	}
	if basicWLPlus {
		p.ElectronicAddress, p.ElectronicAddressSchemeID = opts.endpoint(c)
	}
	if withContact && (c.ContactName != "" || c.ContactPhone != "" || c.ContactEmail != "") {
		p.Contact = &ciiContact{Name: c.ContactName, Phone: c.ContactPhone, Email: c.ContactEmail}
	}
	return p
}

// buildAddress maps an address. MINIMUM only carries the country.
func buildAddress(a *invoice.Address, country string, minimum bool) *ciiAddress {
	if a == nil && country == "" {
		return nil
	}
	addr := &ciiAddress{Country: country}
	if a != nil && !minimum {
		addr.Address = a.Address
		addr.Address2 = a.Address2
		addr.PostalCode = a.PostalCode
		addr.City = a.City
	}
	if addr.Country == "" {
		// Country is mandatory inside PostalTradeAddress; without it the
		// address cannot be emitted at all.
		return nil
	}
	return addr
}

// ---- Monetary computation ----

// taxGroup is one VAT breakdown line: the invoice tax groups that share an
// effective VAT category and rate once option defaults apply (BR-CO-18).
type taxGroup struct {
	category            string
	percent             decimal.Decimal
	hasRate             bool
	exemptionReason     string
	exemptionReasonCode string
	lineSum             decimal.Decimal // sum of rounded line net amounts
	allowance           decimal.Decimal // share of the document discount
	charge              decimal.Decimal // document charges in the group
	tax                 decimal.Decimal
}

func (g *taxGroup) basis() decimal.Decimal { return g.lineSum.Sub(g.allowance).Add(g.charge) }

type lineAmounts struct {
	tax       lineTax
	lineTotal decimal.Decimal // rounded net amount
}

type totals struct {
	lines          []lineAmounts
	groups         []*taxGroup
	charges        []ciiAllowanceCharge
	lineTotal      decimal.Decimal
	allowanceTotal decimal.Decimal
	chargeTotal    decimal.Decimal
	taxBasis       decimal.Decimal
	tax            decimal.Decimal
	grand          decimal.Decimal
	prepaid        decimal.Decimal
	due            decimal.Decimal
}

// computeTotals maps the document's Breakdown onto the VAT breakdown. Every
// amount comes from doc.Breakdown(), the same figures the PDF prints, so the
// XML and the PDF always agree and the EN 16931 identities (BR-CO-10, 11, 13,
// 14, 15) hold by construction.
//
// Breakdown groups are keyed by the raw Tax.Category. Several of them can map
// to one effective category once defaults apply; their rounded amounts are
// then summed, which BR-CO-17 tolerates.
func computeTotals(doc *invoice.Document, opts Options, profile Profile) (*totals, error) {
	if doc.Currency.Precision > 2 {
		return nil, fmt.Errorf("currency precision %d: %w", doc.Currency.Precision, ErrUnsupportedPrecision)
	}

	bd := doc.Breakdown()
	t := &totals{
		lineTotal:      bd.LineTotal,
		allowanceTotal: bd.Allowance,
		chargeTotal:    bd.ChargeTotal,
		taxBasis:       bd.TaxBasis,
		tax:            bd.Tax,
		grand:          bd.GrandTotal,
		prepaid:        bd.Prepaid,
		due:            bd.DuePayable,
	}

	var errs []error
	for i, item := range doc.Items {
		lt, err := resolveLineTax(item, opts)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if lt.fixedAmount && profile != ProfileMinimum {
			errs = append(errs, fmt.Errorf("item %q: %w", item.Name, ErrFixedAmountTax))
		}
		t.lines = append(t.lines, lineAmounts{tax: lt, lineTotal: bd.Lines[i]})
	}
	for _, c := range doc.Charges {
		ct, err := resolveTax("charge "+c.Name, c.Tax, opts)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		t.charges = append(t.charges, ciiAllowanceCharge{
			ActualAmount: c.AmountDecimal().Round(2).StringFixed(2),
			Reason:       c.Name,
			ReasonCode:   c.ReasonCode,
			CategoryCode: ct.category,
			HasRate:      ct.hasRate,
			Percent:      ct.percent.StringFixed(2),
		})
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	byKey := map[string]*taxGroup{}
	for _, g := range bd.Groups {
		if g.Fixed {
			// Only reachable in MINIMUM, which has no breakdown; the amount is
			// already part of bd.Tax.
			continue
		}
		eg := effectiveGroup(g, opts)
		key := eg.category + "|" + eg.percent.String()
		if existing, ok := byKey[key]; ok {
			existing.lineSum = existing.lineSum.Add(eg.lineSum)
			existing.allowance = existing.allowance.Add(eg.allowance)
			existing.charge = existing.charge.Add(eg.charge)
			existing.tax = existing.tax.Add(eg.tax)
			mergeReasons(existing, eg)
			continue
		}
		byKey[key] = eg
		t.groups = append(t.groups, eg)
	}

	sort.Slice(t.groups, func(i, j int) bool {
		if t.groups[i].category != t.groups[j].category {
			return t.groups[i].category < t.groups[j].category
		}
		return t.groups[i].percent.Cmp(t.groups[j].percent) < 0
	})

	return t, nil
}

// effectiveGroup applies the option defaults to an invoice tax group.
func effectiveGroup(g *invoice.TaxGroup, opts Options) *taxGroup {
	eg := &taxGroup{
		percent:   g.Percent,
		hasRate:   true,
		lineSum:   g.LineTotal,
		allowance: g.Allowance,
		charge:    g.Charge,
		tax:       g.Tax,
	}
	if g.Untaxed {
		eg.category = opts.noTaxCategoryCode()
		eg.exemptionReason = opts.NoTaxExemptionReason
		eg.exemptionReasonCode = opts.NoTaxExemptionReasonCode
	} else {
		eg.category = g.Category
		if eg.category == "" {
			eg.category = opts.taxCategoryCode()
		}
		eg.exemptionReason = g.ExemptionReason
		eg.exemptionReasonCode = g.ExemptionReasonCode
	}
	eg.category = strings.ToUpper(strings.TrimSpace(eg.category))
	if vatCategories[eg.category].noRate {
		eg.hasRate = false
	}
	return eg
}

// mergeReasons folds src's exemption reasons into dst: every distinct reason
// text is kept, the first reason code wins.
func mergeReasons(dst, src *taxGroup) {
	if src.exemptionReason != "" && !strings.Contains(dst.exemptionReason, src.exemptionReason) {
		if dst.exemptionReason != "" {
			dst.exemptionReason += "; "
		}
		dst.exemptionReason += src.exemptionReason
	}
	if dst.exemptionReasonCode == "" {
		dst.exemptionReasonCode = src.exemptionReasonCode
	}
}

func buildLineItems(doc *invoice.Document, opts Options, lines []lineAmounts, withDescription bool) []ciiLineItem {
	items := make([]ciiLineItem, 0, len(doc.Items))
	for i, item := range doc.Items {
		la := lines[i]
		gross := item.UnitCostDecimal()
		qty := item.QuantityDecimal()

		li := ciiLineItem{
			LineID:          fmt.Sprintf("%d", i+1),
			Name:            item.Name,
			Quantity:        qty.String(),
			UnitCode:        opts.itemUnitCode(item),
			TaxCategoryCode: la.tax.category,
			HasTaxRate:      la.tax.hasRate,
			TaxPercent:      la.tax.percent.StringFixed(2),
			LineTotal:       la.lineTotal.StringFixed(2),
		}
		if withDescription {
			li.Description = item.Description
		}

		net := gross
		if item.Discount != nil && !qty.IsZero() {
			lineDiscount := item.TotalWithoutTaxAndWithoutDiscount().Sub(item.TotalWithoutTaxAndWithDiscount())
			unitDiscount := lineDiscount.Div(qty).Round(4)
			net = gross.Sub(unitDiscount)
			li.GrossUnitPrice = fmtPrice(gross)
			li.UnitDiscount = fmtPrice(unitDiscount)
		}
		li.NetUnitPrice = fmtPrice(net)

		items = append(items, li)
	}
	return items
}

// fmtPrice renders a unit price with 2 decimals, or 4 when needed to keep the
// value exact (EN 16931 allows more than 2 decimals for prices).
func fmtPrice(d decimal.Decimal) string {
	d = d.Round(4)
	if d.Equal(d.Round(2)) {
		return d.StringFixed(2)
	}
	return d.StringFixed(4)
}
