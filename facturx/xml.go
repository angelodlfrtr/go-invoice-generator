package facturx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"text/template"
	"time"

	generator "github.com/angelodlfrtr/go-invoice-generator/generator"
	"github.com/shopspring/decimal"
)

const ciiXMLTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">

	<rsm:ExchangedDocumentContext>
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
	</rsm:ExchangedDocument>

	<rsm:SupplyChainTradeTransaction>
		{{- if .HasLineItems}}
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
					{{- if .LineDiscount}}
					<ram:AppliedTradeAllowanceCharge>
						<ram:ChargeIndicator><udt:Indicator>false</udt:Indicator></ram:ChargeIndicator>
						<ram:ActualAmount>{{.LineDiscount}}</ram:ActualAmount>
					</ram:AppliedTradeAllowanceCharge>
					{{- end}}
				</ram:GrossPriceProductTradePrice>
				{{- end}}
				<ram:NetPriceProductTradePrice>
					<ram:ChargeAmount>{{.UnitPrice}}</ram:ChargeAmount>
				</ram:NetPriceProductTradePrice>
			</ram:SpecifiedLineTradeAgreement>
			<ram:SpecifiedLineTradeDelivery>
				<ram:BilledQuantity unitCode="{{$.UnitCode}}">{{.Quantity}}</ram:BilledQuantity>
			</ram:SpecifiedLineTradeDelivery>
			<ram:SpecifiedLineTradeSettlement>
				<ram:ApplicableTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:CategoryCode>{{.TaxCategoryCode}}</ram:CategoryCode>
					{{- if .TaxPercent}}
					<ram:RateApplicablePercent>{{.TaxPercent}}</ram:RateApplicablePercent>
					{{- end}}
				</ram:ApplicableTradeTax>
				<ram:SpecifiedTradeSettlementLineMonetarySummation>
					<ram:LineTotalAmount>{{.LineTotal}}</ram:LineTotalAmount>
				</ram:SpecifiedTradeSettlementLineMonetarySummation>
			</ram:SpecifiedLineTradeSettlement>
		</ram:IncludedSupplyChainTradeLineItem>
		{{- end}}
		{{- end}}

		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty>
				<ram:Name>{{xe .SellerName}}</ram:Name>
				{{- if .SellerAddress}}
				<ram:PostalTradeAddress>
					{{- if .SellerAddress.PostalCode}}<ram:PostcodeCode>{{xe .SellerAddress.PostalCode}}</ram:PostcodeCode>{{- end}}
					{{- if .SellerAddress.Address}}<ram:LineOne>{{xe .SellerAddress.Address}}</ram:LineOne>{{- end}}
					{{- if .SellerAddress.Address2}}<ram:LineTwo>{{xe .SellerAddress.Address2}}</ram:LineTwo>{{- end}}
					{{- if .SellerAddress.City}}<ram:CityName>{{xe .SellerAddress.City}}</ram:CityName>{{- end}}
					{{- if .SellerAddress.Country}}<ram:CountryID>{{xe .SellerAddress.Country}}</ram:CountryID>{{- end}}
				</ram:PostalTradeAddress>
				{{- end}}
				{{- if .SellerTaxID}}
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">{{xe .SellerTaxID}}</ram:ID>
				</ram:SpecifiedTaxRegistration>
				{{- end}}
			</ram:SellerTradeParty>
			<ram:BuyerTradeParty>
				<ram:Name>{{xe .BuyerName}}</ram:Name>
				{{- if .BuyerAddress}}
				<ram:PostalTradeAddress>
					{{- if .BuyerAddress.PostalCode}}<ram:PostcodeCode>{{xe .BuyerAddress.PostalCode}}</ram:PostcodeCode>{{- end}}
					{{- if .BuyerAddress.Address}}<ram:LineOne>{{xe .BuyerAddress.Address}}</ram:LineOne>{{- end}}
					{{- if .BuyerAddress.Address2}}<ram:LineTwo>{{xe .BuyerAddress.Address2}}</ram:LineTwo>{{- end}}
					{{- if .BuyerAddress.City}}<ram:CityName>{{xe .BuyerAddress.City}}</ram:CityName>{{- end}}
					{{- if .BuyerAddress.Country}}<ram:CountryID>{{xe .BuyerAddress.Country}}</ram:CountryID>{{- end}}
				</ram:PostalTradeAddress>
				{{- end}}
				{{- if .BuyerTaxID}}
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">{{xe .BuyerTaxID}}</ram:ID>
				</ram:SpecifiedTaxRegistration>
				{{- end}}
			</ram:BuyerTradeParty>
			{{- if .BuyerReference}}
			<ram:BuyerReference>{{xe .BuyerReference}}</ram:BuyerReference>
			{{- end}}
		</ram:ApplicableHeaderTradeAgreement>

		<ram:ApplicableHeaderTradeDelivery/>

		<ram:ApplicableHeaderTradeSettlement>
			{{- if .PaymentMeansCode}}
			<ram:SpecifiedTradeSettlementPaymentMeans>
				<ram:TypeCode>{{.PaymentMeansCode}}</ram:TypeCode>
				{{- if .PaymentIBAN}}
				<ram:PayeePartyCreditorFinancialAccount>
					<ram:IBANID>{{xe .PaymentIBAN}}</ram:IBANID>
				</ram:PayeePartyCreditorFinancialAccount>
				{{- if .PaymentBIC}}
				<ram:PayeeSpecifiedCreditorFinancialInstitution>
					<ram:BICID>{{xe .PaymentBIC}}</ram:BICID>
				</ram:PayeeSpecifiedCreditorFinancialInstitution>
				{{- end}}
				{{- end}}
			</ram:SpecifiedTradeSettlementPaymentMeans>
			{{- end}}
			<ram:InvoiceCurrencyCode>{{.CurrencyCode}}</ram:InvoiceCurrencyCode>
			{{- range .TaxBreakdown}}
			<ram:ApplicableTradeTax>
				<ram:CalculatedAmount>{{.TaxAmount}}</ram:CalculatedAmount>
				<ram:TypeCode>VAT</ram:TypeCode>
				<ram:BasisAmount>{{.BasisAmount}}</ram:BasisAmount>
				<ram:CategoryCode>{{.CategoryCode}}</ram:CategoryCode>
				{{- if .Percent}}
				<ram:RateApplicablePercent>{{.Percent}}</ram:RateApplicablePercent>
				{{- end}}
			</ram:ApplicableTradeTax>
			{{- end}}
			{{- range .DocAllowances}}
			<ram:SpecifiedTradeAllowanceCharge>
				<ram:ChargeIndicator><udt:Indicator>false</udt:Indicator></ram:ChargeIndicator>
				<ram:ActualAmount>{{.ActualAmount}}</ram:ActualAmount>
				<ram:CategoryTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:CategoryCode>{{.CategoryCode}}</ram:CategoryCode>
					{{- if .Percent}}
					<ram:RateApplicablePercent>{{.Percent}}</ram:RateApplicablePercent>
					{{- end}}
				</ram:CategoryTradeTax>
			</ram:SpecifiedTradeAllowanceCharge>
			{{- end}}
			{{- if .PaymentDueDate}}
			<ram:SpecifiedTradePaymentTerms>
				<ram:DueDateDateTime>
					<udt:DateTimeString format="102">{{.PaymentDueDate}}</udt:DateTimeString>
				</ram:DueDateDateTime>
			</ram:SpecifiedTradePaymentTerms>
			{{- end}}
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				{{- if .HasLineTotalAmount}}
				<ram:LineTotalAmount>{{.LineTotalAmount}}</ram:LineTotalAmount>
				{{- end}}
				{{- if .HasAllowance}}
				<ram:AllowanceTotalAmount>{{.AllowanceTotalAmount}}</ram:AllowanceTotalAmount>
				{{- end}}
				<ram:TaxBasisTotalAmount>{{.TaxBasisTotalAmount}}</ram:TaxBasisTotalAmount>
				<ram:TaxTotalAmount currencyID="{{.CurrencyCode}}">{{.TaxTotalAmount}}</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>{{.GrandTotalAmount}}</ram:GrandTotalAmount>
				<ram:DuePayableAmount>{{.GrandTotalAmount}}</ram:DuePayableAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`

type ciiAddress struct {
	Address    string
	Address2   string
	PostalCode string
	City       string
	Country    string
}

type ciiTaxLine struct {
	TaxAmount    string
	BasisAmount  string
	CategoryCode string
	Percent      string // empty for fixed-amount taxes
}

type ciiAllowanceCharge struct {
	ActualAmount string
	CategoryCode string
	Percent      string // empty for fixed-amount or no-tax groups
}

type ciiLineItem struct {
	LineID          string
	Name            string
	Description     string
	GrossUnitPrice  string // non-empty for EN16931+ when item has a discount
	LineDiscount    string // discount per unit (EN16931+ only)
	UnitPrice       string // net price per unit
	Quantity        string
	TaxPercent      string
	TaxCategoryCode string
	LineTotal       string
}

type ciiData struct {
	GuidelineID          string
	TypeCode             string
	ID                   string
	IssueDate            string
	SellerName           string
	SellerAddress        *ciiAddress
	SellerTaxID          string
	BuyerName            string
	BuyerAddress         *ciiAddress
	BuyerTaxID           string
	BuyerReference       string
	CurrencyCode         string
	PaymentMeansCode     string
	PaymentIBAN          string
	PaymentBIC           string
	PaymentDueDate       string
	TaxCategoryCode      string
	UnitCode             string
	TaxBreakdown         []ciiTaxLine
	DocAllowances        []ciiAllowanceCharge
	HasAllowance         bool
	AllowanceTotalAmount string
	HasLineTotalAmount   bool
	LineTotalAmount      string
	TaxBasisTotalAmount  string
	TaxTotalAmount       string
	GrandTotalAmount     string
	HasLineItems         bool
	LineItems            []ciiLineItem
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
// doc.Validate() (called by doc.Build()) must have run before calling this.
func BuildXML(doc *generator.Document, opts Options) ([]byte, error) {
	data, err := buildCIIData(doc, opts)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := ciiTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("facturx: XML template: %w", err)
	}

	// Collapse excessive blank lines produced by template whitespace.
	out := strings.TrimSpace(buf.String())
	return []byte(out), nil
}

func buildCIIData(doc *generator.Document, opts Options) (*ciiData, error) {
	issueDate, err := formatDate(doc.Date)
	if err != nil {
		issueDate = time.Now().Format("20060102")
	}

	profile := opts.profile()
	isEN16931Plus := profile == ProfileEN16931 || profile == ProfileExtended

	d := &ciiData{
		GuidelineID:     profile.guidelineID(),
		TypeCode:        opts.typeCode(),
		ID:              doc.Ref,
		IssueDate:       issueDate,
		SellerName:      doc.Company.Name,
		SellerTaxID:     opts.SellerTaxID,
		BuyerName:       doc.Customer.Name,
		BuyerTaxID:      opts.BuyerTaxID,
		BuyerReference:  opts.BuyerReference,
		CurrencyCode:    opts.currencyCode(),
		PaymentMeansCode: opts.paymentMeansCode(),
		PaymentIBAN:     opts.PaymentIBAN,
		PaymentBIC:      opts.PaymentBIC,
		PaymentDueDate:  opts.PaymentDueDate,
		TaxCategoryCode: opts.taxCategoryCode(),
		UnitCode:        opts.itemDefaultUnitCode(),
	}

	// Seller address — MINIMUM only gets CountryID.
	if doc.Company.Address != nil {
		a := &ciiAddress{Country: opts.sellerCountryCode(doc)}
		if profile != ProfileMinimum {
			a.Address = doc.Company.Address.Address
			a.Address2 = doc.Company.Address.Address2
			a.PostalCode = doc.Company.Address.PostalCode
			a.City = doc.Company.Address.City
		}
		d.SellerAddress = a
	}

	// Buyer address — MINIMUM only gets CountryID.
	if doc.Customer.Address != nil {
		a := &ciiAddress{Country: opts.buyerCountryCode(doc)}
		if profile != ProfileMinimum {
			a.Address = doc.Customer.Address.Address
			a.Address2 = doc.Customer.Address.Address2
			a.PostalCode = doc.Customer.Address.PostalCode
			a.City = doc.Customer.Address.City
		}
		d.BuyerAddress = a
	}

	// Monetary totals.
	lineTotal := doc.TotalWithoutTaxAndWithoutDocumentDiscount()
	taxBasis := doc.TotalWithoutTax()
	taxTotal := doc.Tax()
	grandTotal := doc.TotalWithTax()

	d.LineTotalAmount = lineTotal.StringFixed(2)
	d.TaxBasisTotalAmount = taxBasis.StringFixed(2)
	d.TaxTotalAmount = taxTotal.StringFixed(2)
	d.GrandTotalAmount = grandTotal.StringFixed(2)

	// MINIMUM profile omits tax breakdown, payment terms, line total.
	if profile == ProfileMinimum {
		d.PaymentDueDate = ""
		d.PaymentIBAN = ""
		d.PaymentBIC = ""
		d.PaymentMeansCode = ""
		return d, nil
	}

	d.HasLineTotalAmount = true
	d.TaxBreakdown = buildTaxBreakdown(doc, opts.taxCategoryCode())

	// Document-level allowances (EN16931+).
	if isEN16931Plus && doc.Discount != nil {
		allowances, total := buildDocAllowances(doc, opts.taxCategoryCode())
		if len(allowances) > 0 {
			d.DocAllowances = allowances
			d.HasAllowance = true
			d.AllowanceTotalAmount = total.StringFixed(2)
		}
	}

	// Line items — BASIC and above.
	if profile != ProfileBasicWL {
		d.HasLineItems = true
		d.LineItems = buildLineItems(doc, opts.taxCategoryCode(), isEN16931Plus)
	}

	return d, nil
}

// buildTaxBreakdown groups items by effective tax rate and computes per-rate
// basis/tax amounts, accounting for the document-level discount.
func buildTaxBreakdown(doc *generator.Document, categoryCode string) []ciiTaxLine {
	type group struct {
		percent     decimal.Decimal
		isFixed     bool
		basisAmount decimal.Decimal
		taxAmount   decimal.Decimal
	}

	groups := make(map[string]*group)
	totalPreDiscount := decimal.Zero

	for _, item := range doc.Items {
		basis := item.TotalWithoutTaxAndWithDiscount()
		totalPreDiscount = totalPreDiscount.Add(basis)

		if item.Tax == nil {
			continue
		}

		switch {
		case item.Tax.Percent != "":
			key := item.Tax.Percent
			if _, ok := groups[key]; !ok {
				p, _ := decimal.NewFromString(item.Tax.Percent)
				groups[key] = &group{percent: p}
			}
			g := groups[key]
			g.basisAmount = g.basisAmount.Add(basis)
			g.taxAmount = g.taxAmount.Add(item.TaxWithTotalDiscounted())

		case item.Tax.Amount != "":
			key := "__fixed__"
			if _, ok := groups[key]; !ok {
				groups[key] = &group{isFixed: true}
			}
			g := groups[key]
			g.basisAmount = g.basisAmount.Add(basis)
			g.taxAmount = g.taxAmount.Add(item.TaxWithTotalDiscounted())
		}
	}

	// Apply document-level discount proportionally.
	if doc.Discount != nil && !totalPreDiscount.IsZero() {
		discountedTotal := doc.TotalWithoutTax()
		discountFactor := discountedTotal.Div(totalPreDiscount)
		for _, g := range groups {
			if g.isFixed {
				g.basisAmount = g.basisAmount.Mul(discountFactor)
			} else {
				discountedBasis := g.basisAmount.Mul(discountFactor)
				g.taxAmount = discountedBasis.Mul(g.percent).Div(decimal.NewFromFloat(100))
				g.basisAmount = discountedBasis
			}
		}
	}

	// Stable ordering: percent-based rates first (sorted), then fixed.
	var percentKeys []string
	hasFixed := false
	for k, g := range groups {
		if g.isFixed {
			hasFixed = true
		} else {
			percentKeys = append(percentKeys, k)
		}
	}
	for i := 0; i < len(percentKeys)-1; i++ {
		for j := i + 1; j < len(percentKeys); j++ {
			if percentKeys[i] > percentKeys[j] {
				percentKeys[i], percentKeys[j] = percentKeys[j], percentKeys[i]
			}
		}
	}

	var lines []ciiTaxLine
	for _, k := range percentKeys {
		g := groups[k]
		lines = append(lines, ciiTaxLine{
			TaxAmount:    g.taxAmount.StringFixed(2),
			BasisAmount:  g.basisAmount.StringFixed(2),
			CategoryCode: categoryCode,
			Percent:      g.percent.StringFixed(2),
		})
	}
	if hasFixed {
		g := groups["__fixed__"]
		lines = append(lines, ciiTaxLine{
			TaxAmount:    g.taxAmount.StringFixed(2),
			BasisAmount:  g.basisAmount.StringFixed(2),
			CategoryCode: categoryCode,
		})
	}

	return lines
}

// buildDocAllowances computes document-level allowance charges per tax rate,
// required for EN16931+ when a document discount is present.
func buildDocAllowances(doc *generator.Document, categoryCode string) ([]ciiAllowanceCharge, decimal.Decimal) {
	if doc.Discount == nil {
		return nil, decimal.Zero
	}

	type group struct {
		percent decimal.Decimal
		isFixed bool
		basis   decimal.Decimal // pre-discount basis for this rate group
	}

	groups := make(map[string]*group)
	for _, item := range doc.Items {
		basis := item.TotalWithoutTaxAndWithDiscount()
		if item.Tax == nil || (item.Tax.Percent == "" && item.Tax.Amount == "") {
			key := "__notax__"
			if _, ok := groups[key]; !ok {
				groups[key] = &group{isFixed: true}
			}
			groups[key].basis = groups[key].basis.Add(basis)
			continue
		}
		if item.Tax.Percent != "" {
			key := item.Tax.Percent
			if _, ok := groups[key]; !ok {
				p, _ := decimal.NewFromString(item.Tax.Percent)
				groups[key] = &group{percent: p}
			}
			groups[key].basis = groups[key].basis.Add(basis)
		} else {
			key := "__fixed__"
			if _, ok := groups[key]; !ok {
				groups[key] = &group{isFixed: true}
			}
			groups[key].basis = groups[key].basis.Add(basis)
		}
	}

	totalPreDiscount := doc.TotalWithoutTaxAndWithoutDocumentDiscount()
	totalPostDiscount := doc.TotalWithoutTax()
	if totalPreDiscount.IsZero() {
		return nil, decimal.Zero
	}

	totalDiscount := totalPreDiscount.Sub(totalPostDiscount)
	discountRatio := totalDiscount.Div(totalPreDiscount)

	var allowances []ciiAllowanceCharge
	for key, g := range groups {
		amount := g.basis.Mul(discountRatio)
		ac := ciiAllowanceCharge{
			ActualAmount: amount.StringFixed(2),
			CategoryCode: categoryCode,
		}
		if !g.isFixed && key != "__notax__" {
			ac.Percent = g.percent.StringFixed(2)
		}
		allowances = append(allowances, ac)
	}

	return allowances, totalDiscount
}

func buildLineItems(doc *generator.Document, categoryCode string, withGrossPrice bool) []ciiLineItem {
	items := make([]ciiLineItem, len(doc.Items))
	for i, item := range doc.Items {
		unitCost, _ := decimal.NewFromString(item.UnitCost)
		qty, _ := decimal.NewFromString(item.Quantity)
		lineTotal := item.TotalWithoutTaxAndWithDiscount()

		var netUnitPrice decimal.Decimal
		if !qty.IsZero() {
			netUnitPrice = lineTotal.Div(qty)
		}

		li := ciiLineItem{
			LineID:          fmt.Sprintf("%d", i+1),
			Name:            item.Name,
			Description:     item.Description,
			UnitPrice:       netUnitPrice.StringFixed(2),
			Quantity:        item.Quantity,
			TaxCategoryCode: categoryCode,
			LineTotal:       lineTotal.StringFixed(2),
		}

		// Gross price and per-unit discount for EN16931+.
		if withGrossPrice && item.Discount != nil {
			li.GrossUnitPrice = unitCost.StringFixed(2)
			discountPerUnit := unitCost.Sub(netUnitPrice)
			li.LineDiscount = discountPerUnit.StringFixed(2)
		}

		if item.Tax != nil && item.Tax.Percent != "" {
			li.TaxPercent = item.Tax.Percent
		}

		items[i] = li
	}
	return items
}

// formatDate converts "DD/MM/YYYY" or "YYYY-MM-DD" to "YYYYMMDD" (CII format 102).
func formatDate(date string) (string, error) {
	if t, err := time.Parse("02/01/2006", date); err == nil {
		return t.Format("20060102"), nil
	}
	if t, err := time.Parse("2006-01-02", date); err == nil {
		return t.Format("20060102"), nil
	}
	if len(date) == 8 {
		return date, nil
	}
	return "", fmt.Errorf("facturx: cannot parse date %q", date)
}
