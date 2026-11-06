package facturx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	generator "github.com/angelodlfrtr/go-invoice-generator/generator"
	"github.com/shopspring/decimal"
)

var hundred = decimal.NewFromInt(100)

// frRuleRe matches a French BR-FR rule reported inside a warning or error
// element (the mustang log also mentions the schematron file name).
var frRuleRe = regexp.MustCompile(`<(warning|error)[^>]*>\s*BR-FR`)

var allProfiles = []Profile{
	ProfileMinimum,
	ProfileBasicWL,
	ProfileBasic,
	ProfileEN16931,
	ProfileExtended,
}

func profileFilename(prefix string, profile Profile) string {
	return fmt.Sprintf("../out/%s_%s.pdf", prefix, strings.ReplaceAll(string(profile), " ", "_"))
}

func newTestDoc(t *testing.T) *generator.Document {
	t.Helper()

	doc, err := generator.New(generator.Invoice, &generator.Options{
		TextTypeInvoice:   "INVOICE",
		CurrencySymbol:    "€ ",
		CurrencyPrecision: 2,
	})
	if err != nil {
		t.Fatalf("generator.New: %v", err)
	}

	doc.SetRef("INV-2024-001")
	doc.SetDate("01/01/2024")
	doc.SetPaymentTerm("01/02/2024")

	doc.SetCompany(&generator.Contact{
		Name: "Acme Corp",
		Address: &generator.Address{
			Address:    "1 Rue de la Paix",
			PostalCode: "75001",
			City:       "Paris",
			Country:    "FR",
		},
		AddtionnalInfo: []string{"VAT: FR12345678901"},
	})

	doc.SetCustomer(&generator.Contact{
		Name: "John Doe",
		Address: &generator.Address{
			Address:    "42 Main Street",
			PostalCode: "10001",
			City:       "New York",
			Country:    "US",
		},
	})

	return doc
}

// buildTestDoc is the simple fixture: two standard-rated lines, one item discount.
func buildTestDoc(t *testing.T) *generator.Document {
	t.Helper()
	doc := newTestDoc(t)

	doc.AppendItem(&generator.Item{
		Name:     "Consulting",
		UnitCost: "150.00",
		Quantity: "8",
		Tax:      &generator.Tax{Percent: "20"},
	})

	doc.AppendItem(&generator.Item{
		Name:     "Support",
		UnitCost: "75.00",
		Quantity: "4",
		Tax:      &generator.Tax{Percent: "20"},
		Discount: &generator.Discount{Percent: "10"},
	})

	return doc
}

// buildFullTestDoc exercises mixed rates, an exempt line, an untaxed line,
// item discounts, a document discount and notes.
func buildFullTestDoc(t *testing.T) *generator.Document {
	t.Helper()
	doc := newTestDoc(t)
	doc.SetNotes("Thank you for your business.")
	doc.SetDiscount(&generator.Discount{Percent: "5"})

	doc.AppendItem(&generator.Item{
		Name:        "Consulting",
		Description: "Senior consultant, daily rate",
		UnitCost:    "150.00",
		Quantity:    "8",
		Tax:         &generator.Tax{Percent: "20"},
	})
	doc.AppendItem(&generator.Item{
		Name:     "Support",
		UnitCost: "75.00",
		Quantity: "4",
		Tax:      &generator.Tax{Percent: "20"},
		Discount: &generator.Discount{Percent: "10"},
	})
	doc.AppendItem(&generator.Item{
		Name:     "Books",
		UnitCost: "12.33",
		Quantity: "3",
		Tax:      &generator.Tax{Percent: "5.5", Category: "S"},
		Discount: &generator.Discount{Amount: "2.50"},
	})
	doc.AppendItem(&generator.Item{
		Name:     "Training",
		UnitCost: "499.99",
		Quantity: "1",
		Tax: &generator.Tax{
			Percent:             "0",
			Category:            "E",
			ExemptionReason:     "Exempt from VAT, art. 261-4-4 CGI",
			ExemptionReasonCode: "VATEX-EU-132",
		},
	})
	doc.AppendItem(&generator.Item{
		Name:     "Shipping",
		UnitCost: "9.90",
		Quantity: "1",
	})

	return doc
}

func fullOptions(profile Profile) Options {
	return Options{
		Profile:                         profile,
		CurrencyCode:                    "EUR",
		BusinessProcessID:               "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0",
		BuyerReference:                  "COST-CENTRE-42",
		PurchaseOrderReference:          "PO-2023-0099",
		SellerTaxID:                     "FR12345678901",
		SellerLegalID:                   "123456789",
		SellerLegalIDSchemeID:           "0002",
		SellerCountryCode:               "FR",
		SellerElectronicAddress:         "billing@acme.example",
		SellerContactName:               "Jane Smith",
		SellerContactPhone:              "+33 1 23 45 67 89",
		SellerContactEmail:              "jane@acme.example",
		BuyerTaxID:                      "US123456789",
		BuyerLegalID:                    "987654321",
		BuyerCountryCode:                "US",
		BuyerElectronicAddress:          "ap@johndoe.example",
		DeliveryDate:                    "2023-12-31",
		InvoicingPeriodStart:            "01/12/2023",
		InvoicingPeriodEnd:              "31/12/2023",
		PaymentDueDate:                  "20240201",
		PaymentTermsDescription:         "Payable within 30 days",
		PaymentReference:                "INV-2024-001",
		PaymentIBAN:                     "FR7630006000011234567890189",
		PaymentBIC:                      "BNPAFRPP",
		NoTaxExemptionReason:            "VAT not applicable, art. 293 B CGI",
		DocumentDiscountReason:          "Loyalty discount",
		SellerElectronicAddressSchemeID: "EM",
		VATPointDateCode:                "5",
		ShipToName:                      "John Doe Warehouse",
		ShipToAddress: &generator.Address{
			Address:    "7 Industrial Way",
			PostalCode: "07001",
			City:       "Newark",
			Country:    "US",
		},
	}
}

// buildFrenchDoc is a credit note between two French companies.
func buildFrenchDoc(t *testing.T) *generator.Document {
	t.Helper()
	doc := newTestDoc(t)
	doc.SetRef("AV-2024-001")
	doc.SetCustomer(&generator.Contact{
		Name: "Client SARL",
		Address: &generator.Address{
			Address:    "5 Rue de la République",
			PostalCode: "69001",
			City:       "Lyon",
			Country:    "FR",
		},
	})
	doc.AppendItem(&generator.Item{
		Name:     "Consulting (refund)",
		UnitCost: "150.00",
		Quantity: "2",
		Tax:      &generator.Tax{Percent: "20"},
	})
	doc.AppendItem(&generator.Item{
		Name:     "Training (refund)",
		UnitCost: "100.00",
		Quantity: "1",
		Tax: &generator.Tax{
			Percent:         "0",
			Category:        "E",
			ExemptionReason: "TVA non applicable, art. 261-4-4 du CGI",
		},
	})
	return doc
}

func frenchOptions(profile Profile) Options {
	return Options{
		Profile:                   profile,
		FrenchRules:               true,
		TypeCode:                  "381",
		BusinessProcessID:         "S1",
		SellerTaxID:               "FR12345678901",
		SellerLegalID:             "123456789",
		SellerLegalIDSchemeID:     "0002",
		SellerCountryCode:         "FR",
		BuyerTaxID:                "FR98765432109",
		BuyerLegalID:              "987654321",
		BuyerLegalIDSchemeID:      "0002",
		BuyerCountryCode:          "FR",
		PrecedingInvoiceReference: "INV-2023-120",
		PrecedingInvoiceDate:      "2023-12-01",
		VATPointDateCode:          "72",
		PaymentDueDate:            "20240201",
		PaymentTermsDescription:   "Remboursement sous 30 jours",
		Notes:                     frenchNotes(),
		ShipToName:                "Client SARL - Entrepôt",
		ShipToAddress: &generator.Address{
			Address:    "ZI des Chênes",
			PostalCode: "69800",
			City:       "Saint-Priest",
			Country:    "FR",
		},
	}
}

// frenchNotes are the three legal mentions French invoices must carry.
func frenchNotes() []Note {
	return []Note{
		{Content: "Pénalités de retard : 3 fois le taux d'intérêt légal", SubjectCode: "PMD"},
		{Content: "Indemnité forfaitaire pour frais de recouvrement : 40 €", SubjectCode: "PMT"},
		{Content: "Pas d'escompte pour paiement anticipé", SubjectCode: "AAB"},
	}
}

// usdOptions invoices in USD with the VAT total also given in EUR.
func usdOptions(profile Profile) Options {
	o := simpleOptions(profile)
	o.CurrencyCode = "USD"
	o.TaxCurrencyCode = "EUR"
	o.TaxCurrencyRate = "0.9234"
	return o
}

func simpleOptions(profile Profile) Options {
	return Options{
		Profile:           profile,
		SellerTaxID:       "FR12345678901",
		SellerCountryCode: "FR",
		BuyerCountryCode:  "US",
		CurrencyCode:      "EUR",
		PaymentDueDate:    "20240201",
		TaxCategoryCode:   "S",
	}
}

func buildPDF(t *testing.T, doc *generator.Document) []byte {
	t.Helper()

	pdf, err := doc.Build()
	if err != nil {
		t.Fatalf("doc.Build: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("pdf.Output: %v", err)
	}

	return buf.Bytes()
}

func writeOut(t *testing.T, filename string, data []byte) {
	t.Helper()
	if err := os.MkdirAll("../out", 0o750); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

// fixtures lists every (document, options) pair rendered for each profile.
// Keep it in sync with the Makefile validation targets, which glob out/facturx*.pdf.
func fixtures(t *testing.T, profile Profile) map[string]struct {
	doc  *generator.Document
	opts Options
} {
	t.Helper()
	type fixture = struct {
		doc  *generator.Document
		opts Options
	}
	fx := map[string]fixture{
		"facturx":      {buildTestDoc(t), simpleOptions(profile)},
		"facturx_full": {buildFullTestDoc(t), fullOptions(profile)},
	}
	if profile != ProfileMinimum {
		fx["facturx_usd"] = fixture{buildTestDoc(t), usdOptions(profile)}
	}
	if profile == ProfileEN16931 || profile == ProfileExtended {
		fx["facturx_fr"] = fixture{buildFrenchDoc(t), frenchOptions(profile)}
	}
	return fx
}

func TestAttach(t *testing.T) {
	doc := buildTestDoc(t)
	opts := simpleOptions(ProfileMinimum)
	opts.ShowIcon = true

	result, err := Attach(buildPDF(t, doc), doc, opts)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("Attach returned empty result")
	}
	writeOut(t, "../out/facturx.pdf", result)
}

func TestAttachProfiles(t *testing.T) {
	for _, profile := range allProfiles {
		for prefix, fx := range fixtures(t, profile) {
			t.Run(prefix+"/"+string(profile), func(t *testing.T) {
				result, err := Attach(buildPDF(t, fx.doc), fx.doc, fx.opts)
				if err != nil {
					t.Fatalf("Attach(%s): %v", profile, err)
				}
				if len(result) == 0 {
					t.Fatalf("Attach(%s) returned empty result", profile)
				}
				writeOut(t, profileFilename(prefix, profile), result)
			})
		}
	}
}

func TestMustangValidation(t *testing.T) {
	if _, err := exec.LookPath("mustang-cli"); err != nil {
		t.Skip("mustang-cli not in PATH")
	}

	for _, profile := range allProfiles {
		for prefix, fx := range fixtures(t, profile) {
			t.Run(prefix+"/"+string(profile), func(t *testing.T) {
				result, err := Attach(buildPDF(t, fx.doc), fx.doc, fx.opts)
				if err != nil {
					t.Fatalf("Attach(%s): %v", profile, err)
				}
				filename := profileFilename(prefix, profile)
				writeOut(t, filename, result)

				out, err := exec.Command("mustang-cli", "--action", "validate", "--source", filename).CombinedOutput()
				if err != nil {
					t.Fatalf("mustang-cli failed: %v\n%s", err, out)
				}
				// mustang reports a <pdf> summary, an <xml> summary and a global one.
				// The global summary can be "valid" while the <pdf> section is not,
				// so reject any invalid sub-summary or error element.
				if strings.Contains(string(out), `status="invalid"`) || strings.Contains(string(out), "<error") {
					t.Fatalf("profile %s not valid:\n%s", profile, out)
				}
				// mustang >= 2.26 reports the French BR-FR rules as warnings when
				// both parties are in France; treat them as failures.
				if frRuleRe.Match(out) {
					t.Fatalf("profile %s breaks French rules:\n%s", profile, out)
				}
				if !strings.Contains(string(out), `status="valid"`) {
					t.Fatalf("profile %s: no valid summary found:\n%s", profile, out)
				}
			})
		}
	}
}

func TestBuildXML(t *testing.T) {
	doc := buildFullTestDoc(t)
	if err := doc.Validate(); err != nil {
		t.Fatalf("doc.Validate: %v", err)
	}

	xmlBytes, err := BuildXML(doc, fullOptions(ProfileEN16931))
	if err != nil {
		t.Fatalf("BuildXML: %v", err)
	}
	if len(xmlBytes) == 0 {
		t.Fatal("BuildXML returned empty XML")
	}
	if err := xml.Unmarshal(xmlBytes, new(struct{})); err != nil {
		t.Fatalf("not well-formed: %v\n%s", err, xmlBytes)
	}
}

// ---- XML structure assertions ----

type xmlAmount struct {
	Value    string `xml:",chardata"`
	Currency string `xml:"currencyID,attr"`
}

type xmlTradeTax struct {
	CalculatedAmount string `xml:"CalculatedAmount"`
	BasisAmount      string `xml:"BasisAmount"`
	CategoryCode     string `xml:"CategoryCode"`
	ExemptionReason  string `xml:"ExemptionReason"`
	DueDateTypeCode  string `xml:"DueDateTypeCode"`
	Rate             string `xml:"RateApplicablePercent"`
}

type xmlAllowance struct {
	ChargeIndicator string `xml:"ChargeIndicator>Indicator"`
	ActualAmount    string `xml:"ActualAmount"`
	Reason          string `xml:"Reason"`
	CategoryCode    string `xml:"CategoryTradeTax>CategoryCode"`
	Rate            string `xml:"CategoryTradeTax>RateApplicablePercent"`
}

type xmlLine struct {
	Name        string `xml:"SpecifiedTradeProduct>Name"`
	Description string `xml:"SpecifiedTradeProduct>Description"`
	GrossPrice  string `xml:"SpecifiedLineTradeAgreement>GrossPriceProductTradePrice>ChargeAmount"`
	NetPrice    string `xml:"SpecifiedLineTradeAgreement>NetPriceProductTradePrice>ChargeAmount"`
	Quantity    string `xml:"SpecifiedLineTradeDelivery>BilledQuantity"`
	Category    string `xml:"SpecifiedLineTradeSettlement>ApplicableTradeTax>CategoryCode"`
	Rate        string `xml:"SpecifiedLineTradeSettlement>ApplicableTradeTax>RateApplicablePercent"`
	LineTotal   string `xml:"SpecifiedLineTradeSettlement>SpecifiedTradeSettlementLineMonetarySummation>LineTotalAmount"`
}

type xmlNote struct {
	Content     string `xml:"Content"`
	SubjectCode string `xml:"SubjectCode"`
}

type xmlURI struct {
	Value  string `xml:",chardata"`
	Scheme string `xml:"schemeID,attr"`
}

type xmlInvoice struct {
	BusinessProcess string         `xml:"ExchangedDocumentContext>BusinessProcessSpecifiedDocumentContextParameter>ID"`
	Notes           []string       `xml:"-"`
	NoteEntries     []xmlNote      `xml:"ExchangedDocument>IncludedNote"`
	SellerURIScheme string         `xml:"-"`
	BuyerURI        string         `xml:"-"`
	BuyerURIScheme  string         `xml:"-"`
	SellerEndpoint  xmlURI         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>URIUniversalCommunication>URIID"`
	BuyerEndpoint   xmlURI         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerTradeParty>URIUniversalCommunication>URIID"`
	Lines           []xmlLine      `xml:"SupplyChainTradeTransaction>IncludedSupplyChainTradeLineItem"`
	BuyerReference  string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerReference"`
	SellerLegalID   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>SpecifiedLegalOrganization>ID"`
	SellerContact   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>DefinedTradeContact>PersonName"`
	SellerURI       string         `xml:"-"`
	SellerLineOne   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>PostalTradeAddress>LineOne"`
	SellerCountry   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>PostalTradeAddress>CountryID"`
	BuyerTaxID      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerTradeParty>SpecifiedTaxRegistration>ID"`
	PurchaseOrder   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerOrderReferencedDocument>IssuerAssignedID"`
	TypeCode        string         `xml:"ExchangedDocument>TypeCode"`
	ShipToName      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ShipToTradeParty>Name"`
	ShipToCountry   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ShipToTradeParty>PostalTradeAddress>CountryID"`
	DeliveryDate    string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ActualDeliverySupplyChainEvent>OccurrenceDateTime>DateTimeString"`
	TaxCurrency     string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>TaxCurrencyCode"`
	PrecedingRef    string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>InvoiceReferencedDocument>IssuerAssignedID"`
	PrecedingDate   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>InvoiceReferencedDocument>FormattedIssueDateTime>DateTimeString"`
	PaymentRef      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>PaymentReference"`
	IBAN            string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementPaymentMeans>PayeePartyCreditorFinancialAccount>IBANID"`
	BIC             string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementPaymentMeans>PayeeSpecifiedCreditorFinancialInstitution>BICID"`
	Taxes           []xmlTradeTax  `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>ApplicableTradeTax"`
	PeriodStart     string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>BillingSpecifiedPeriod>StartDateTime>DateTimeString"`
	Allowances      []xmlAllowance `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeAllowanceCharge"`
	PaymentTerms    string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradePaymentTerms>Description"`
	DueDate         string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradePaymentTerms>DueDateDateTime>DateTimeString"`
	LineTotal       string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>LineTotalAmount"`
	AllowanceTotal  string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>AllowanceTotalAmount"`
	TaxBasisTotal   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TaxBasisTotalAmount"`
	TaxTotal        xmlAmount      `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TaxTotalAmount"`
	GrandTotal      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>GrandTotalAmount"`
	DuePayable      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>DuePayableAmount"`
}

func parseInvoice(t *testing.T, doc *generator.Document, opts Options) xmlInvoice {
	t.Helper()
	if err := doc.Validate(); err != nil {
		t.Fatalf("doc.Validate: %v", err)
	}
	xmlBytes, err := BuildXML(doc, opts)
	if err != nil {
		t.Fatalf("BuildXML(%s): %v", opts.Profile, err)
	}
	var inv xmlInvoice
	if err := xml.Unmarshal(xmlBytes, &inv); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, xmlBytes)
	}
	for _, n := range inv.NoteEntries {
		inv.Notes = append(inv.Notes, n.Content)
	}
	inv.SellerURI = inv.SellerEndpoint.Value
	inv.SellerURIScheme = inv.SellerEndpoint.Scheme
	inv.BuyerURI = inv.BuyerEndpoint.Value
	inv.BuyerURIScheme = inv.BuyerEndpoint.Scheme
	return inv
}

func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}

func sumDec(t *testing.T, values ...string) decimal.Decimal {
	t.Helper()
	total := decimal.Zero
	for _, v := range values {
		total = total.Add(dec(t, v))
	}
	return total
}

func assertEqual(t *testing.T, what string, got, want decimal.Decimal) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s: got %s, want %s", what, got, want)
	}
}

// TestTotalsReconcile checks the EN 16931 arithmetic rules on the full fixture
// for every profile that carries a VAT breakdown.
func TestTotalsReconcile(t *testing.T) {
	for _, profile := range []Profile{ProfileBasicWL, ProfileBasic, ProfileEN16931, ProfileExtended} {
		t.Run(string(profile), func(t *testing.T) {
			inv := parseInvoice(t, buildFullTestDoc(t), fullOptions(profile))

			if inv.TaxTotal.Currency != "EUR" {
				t.Errorf("TaxTotalAmount currencyID: got %q", inv.TaxTotal.Currency)
			}

			// BR-CO-13: tax basis = line total - allowances.
			assertEqual(t, "BR-CO-13 tax basis",
				dec(t, inv.TaxBasisTotal), dec(t, inv.LineTotal).Sub(dec(t, inv.AllowanceTotal)))

			// BR-CO-11: allowance total = sum of document allowances.
			var allowances []string
			for _, a := range inv.Allowances {
				if a.ChargeIndicator != "false" {
					t.Errorf("allowance has ChargeIndicator %q", a.ChargeIndicator)
				}
				if a.Reason != "Loyalty discount" {
					t.Errorf("allowance reason: got %q", a.Reason)
				}
				allowances = append(allowances, a.ActualAmount)
			}
			assertEqual(t, "BR-CO-11 allowance total", dec(t, inv.AllowanceTotal), sumDec(t, allowances...))

			// BR-CO-14: VAT total = sum of category VAT amounts, each BR-CO-17 consistent.
			var taxes, bases []string
			seen := map[string]bool{}
			for _, tx := range inv.Taxes {
				taxes = append(taxes, tx.CalculatedAmount)
				bases = append(bases, tx.BasisAmount)
				key := tx.CategoryCode + "|" + tx.Rate
				if seen[key] {
					t.Errorf("BR-CO-18 duplicate breakdown for %s", key)
				}
				seen[key] = true
				if tx.Rate != "" {
					want := dec(t, tx.BasisAmount).Mul(dec(t, tx.Rate)).Div(hundred).Round(2)
					assertEqual(t, "BR-CO-17 VAT for "+key, dec(t, tx.CalculatedAmount), want)
				}
				if tx.CategoryCode == "E" && tx.ExemptionReason == "" {
					t.Errorf("BR-E-10 exempt breakdown without reason")
				}
				if tx.CategoryCode == "S" && dec(t, tx.Rate).Sign() <= 0 {
					t.Errorf("BR-S-05 standard rate breakdown with rate %q", tx.Rate)
				}
			}
			assertEqual(t, "BR-CO-14 VAT total", dec(t, inv.TaxTotal.Value), sumDec(t, taxes...))
			assertEqual(t, "sum of bases", sumDec(t, bases...), dec(t, inv.TaxBasisTotal))

			// BR-CO-15 / BR-CO-16: grand total and amount due.
			assertEqual(t, "BR-CO-15 grand total",
				dec(t, inv.GrandTotal), dec(t, inv.TaxBasisTotal).Add(dec(t, inv.TaxTotal.Value)))
			assertEqual(t, "BR-CO-16 due payable", dec(t, inv.DuePayable), dec(t, inv.GrandTotal))

			// The XML must print exactly what the PDF prints.
			doc := buildFullTestDoc(t)
			if err := doc.Validate(); err != nil {
				t.Fatal(err)
			}
			assertEqual(t, "PDF line total", dec(t, inv.LineTotal), doc.TotalWithoutTaxAndWithoutDocumentDiscount())
			assertEqual(t, "PDF discount", dec(t, inv.AllowanceTotal), doc.DocumentDiscount())
			assertEqual(t, "PDF total without tax", dec(t, inv.TaxBasisTotal), doc.TotalWithoutTax())
			assertEqual(t, "PDF tax", dec(t, inv.TaxTotal.Value), doc.Tax())
			assertEqual(t, "PDF total with tax", dec(t, inv.GrandTotal), doc.TotalWithTax())

			// Expected breakdown: S 20, S 5.5 and two exempt groups merged (E 0).
			if len(inv.Taxes) != 3 {
				t.Errorf("expected 3 VAT groups (E 0, S 5.5, S 20), got %d: %+v", len(inv.Taxes), inv.Taxes)
			}

			// BR-CO-10: line total = sum of line net amounts (BASIC and above).
			if len(inv.Lines) > 0 {
				var lines []string
				for _, l := range inv.Lines {
					lines = append(lines, l.LineTotal)
					if l.Category == "S" && l.Rate == "" {
						t.Errorf("line %q: standard rate without RateApplicablePercent", l.Name)
					}
				}
				assertEqual(t, "BR-CO-10 line total", dec(t, inv.LineTotal), sumDec(t, lines...))
			}
		})
	}
}

// TestProfileGating checks that fields are only emitted for the profiles that
// carry them in the Factur-X XSDs.
func TestProfileGating(t *testing.T) {
	doc := buildFullTestDoc(t)
	minimum := parseInvoice(t, doc, fullOptions(ProfileMinimum))
	basicWL := parseInvoice(t, buildFullTestDoc(t), fullOptions(ProfileBasicWL))
	basic := parseInvoice(t, buildFullTestDoc(t), fullOptions(ProfileBasic))
	en := parseInvoice(t, buildFullTestDoc(t), fullOptions(ProfileEN16931))

	// MINIMUM: header identifiers yes, everything else no.
	if minimum.BusinessProcess == "" || minimum.BuyerReference == "" || minimum.PurchaseOrder == "" || minimum.SellerLegalID == "" {
		t.Errorf("MINIMUM should carry business process, buyer reference, PO and legal ID: %+v", minimum)
	}
	if minimum.SellerCountry != "FR" || minimum.SellerLineOne != "" {
		t.Errorf("MINIMUM seller address should only carry the country: %q %q", minimum.SellerCountry, minimum.SellerLineOne)
	}
	if len(minimum.Notes) != 0 || len(minimum.Taxes) != 0 || len(minimum.Lines) != 0 || minimum.IBAN != "" ||
		minimum.DueDate != "" || minimum.LineTotal != "" || minimum.SellerURI != "" || minimum.BuyerTaxID != "" ||
		minimum.ShipToName != "" {
		t.Errorf("MINIMUM carries fields above its profile: %+v", minimum)
	}
	assertEqual(t, "MINIMUM BR-CO-15",
		dec(t, minimum.GrandTotal), dec(t, minimum.TaxBasisTotal).Add(dec(t, minimum.TaxTotal.Value)))

	// BASIC-WL: header data, no lines, no BIC, no contact.
	if len(basicWL.Lines) != 0 {
		t.Errorf("BASIC-WL must not carry line items")
	}
	if basicWL.SellerURI == "" || basicWL.BuyerTaxID == "" || basicWL.DeliveryDate != "20231231" ||
		basicWL.PeriodStart != "20231201" || basicWL.PaymentRef == "" || basicWL.IBAN == "" ||
		basicWL.PaymentTerms == "" || basicWL.DueDate != "20240201" || len(basicWL.Notes) != 1 {
		t.Errorf("BASIC-WL misses header fields: %+v", basicWL)
	}
	if basicWL.BIC != "" || basicWL.SellerContact != "" {
		t.Errorf("BASIC-WL must not carry BIC or seller contact")
	}
	if basicWL.ShipToName != "John Doe Warehouse" || basicWL.ShipToCountry != "US" {
		t.Errorf("BASIC-WL ship-to: %q %q", basicWL.ShipToName, basicWL.ShipToCountry)
	}
	for _, tx := range basicWL.Taxes {
		if tx.DueDateTypeCode != "5" {
			t.Errorf("BASIC-WL VAT point date code: got %q, want 5", tx.DueDateTypeCode)
		}
	}
	if basicWL.SellerLineOne != "1 Rue de la Paix" {
		t.Errorf("BASIC-WL seller address line: %q", basicWL.SellerLineOne)
	}

	// BASIC: lines without description, gross price for discounted items.
	if len(basic.Lines) != 5 {
		t.Fatalf("BASIC expected 5 lines, got %d", len(basic.Lines))
	}
	if basic.Lines[0].Description != "" {
		t.Errorf("BASIC must not carry product descriptions")
	}
	if basic.Lines[1].GrossPrice != "75.00" || basic.Lines[1].NetPrice != "67.50" {
		t.Errorf("discounted line prices: gross %q net %q", basic.Lines[1].GrossPrice, basic.Lines[1].NetPrice)
	}
	if basic.Lines[4].Category != "E" || basic.Lines[4].Rate != "0.00" {
		t.Errorf("untaxed line should be E 0.00, got %s %s", basic.Lines[4].Category, basic.Lines[4].Rate)
	}
	if basic.BIC != "" {
		t.Errorf("BASIC must not carry BIC")
	}

	// EN 16931: everything.
	if en.Lines[0].Description == "" || en.BIC == "" || en.SellerContact == "" {
		t.Errorf("EN 16931 misses description, BIC or seller contact: %+v", en)
	}
}

func TestPaymentDueDateFromPaymentTerm(t *testing.T) {
	doc := buildTestDoc(t)
	doc.SetPaymentTerm("15/03/2024")
	opts := simpleOptions(ProfileBasic)
	opts.PaymentDueDate = ""

	inv := parseInvoice(t, doc, opts)
	if inv.DueDate != "20240315" {
		t.Errorf("due date from PaymentTerm: got %q, want 20240315", inv.DueDate)
	}

	// Free-text payment terms are not a date: fall back to the description.
	doc = buildTestDoc(t)
	doc.SetPaymentTerm("30 days end of month")
	opts.PaymentTermsDescription = "30 days end of month"
	inv = parseInvoice(t, doc, opts)
	if inv.DueDate != "" || inv.PaymentTerms != "30 days end of month" {
		t.Errorf("free-text payment term: due %q terms %q", inv.DueDate, inv.PaymentTerms)
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		doc  func(t *testing.T) *generator.Document
		opts func() Options
		want error
	}{
		{
			name: "fixed-amount tax above MINIMUM",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&generator.Item{Name: "Eco tax", UnitCost: "10", Quantity: "1", Tax: &generator.Tax{Amount: "2"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrFixedAmountTax,
		},
		{
			name: "missing seller identifier",
			doc:  buildTestDoc,
			opts: func() Options { o := simpleOptions(ProfileMinimum); o.SellerTaxID = ""; return o },
			want: ErrMissingSellerID,
		},
		{
			name: "invalid country code",
			doc:  buildTestDoc,
			opts: func() Options { o := simpleOptions(ProfileBasic); o.BuyerCountryCode = "France"; return o },
			want: ErrInvalidCountryCode,
		},
		{
			name: "missing buyer country above MINIMUM",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.Customer.Address.Country = ""
				return doc
			},
			opts: func() Options { o := simpleOptions(ProfileBasicWL); o.BuyerCountryCode = ""; return o },
			want: ErrMissingCountry,
		},
		{
			name: "invalid currency",
			doc:  buildTestDoc,
			opts: func() Options { o := simpleOptions(ProfileMinimum); o.CurrencyCode = "euro"; return o },
			want: ErrInvalidCurrencyCode,
		},
		{
			name: "invalid document date",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.SetDate("January 1st")
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileMinimum) },
			want: ErrInvalidDate,
		},
		{
			name: "invalid delivery date",
			doc:  buildTestDoc,
			opts: func() Options { o := simpleOptions(ProfileBasic); o.DeliveryDate = "2024-13-45"; return o },
			want: ErrInvalidDate,
		},
		{
			name: "untaxed item without exemption reason",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&generator.Item{Name: "Shipping", UnitCost: "10", Quantity: "1"})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrMissingExemptionReason,
		},
		{
			name: "standard category with zero rate",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&generator.Item{Name: "Free", UnitCost: "10", Quantity: "1", Tax: &generator.Tax{Percent: "0"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidTaxCategory,
		},
		{
			name: "exempt category with non-zero rate",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&generator.Item{Name: "Odd", UnitCost: "10", Quantity: "1",
					Tax: &generator.Tax{Percent: "20", Category: "E", ExemptionReason: "x"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidTaxCategory,
		},
		{
			name: "unknown category",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&generator.Item{Name: "Odd", UnitCost: "10", Quantity: "1",
					Tax: &generator.Tax{Percent: "20", Category: "XX"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidTaxCategory,
		},
		{
			name: "no payment terms above MINIMUM",
			doc: func(t *testing.T) *generator.Document {
				doc := buildTestDoc(t)
				doc.SetPaymentTerm("")
				return doc
			},
			opts: func() Options { o := simpleOptions(ProfileBasicWL); o.PaymentDueDate = ""; return o },
			want: ErrMissingPaymentTerms,
		},
		{
			name: "half invoicing period",
			doc:  buildTestDoc,
			opts: func() Options { o := simpleOptions(ProfileBasic); o.InvoicingPeriodStart = "01/01/2024"; return o },
			want: ErrInvalidPeriod,
		},
		{
			name: "invalid VAT point date code",
			doc:  buildTestDoc,
			opts: func() Options { o := simpleOptions(ProfileBasic); o.VATPointDateCode = "35"; return o },
			want: ErrInvalidVATPointDateCode,
		},
		{
			name: "tax currency equals invoice currency",
			doc:  buildTestDoc,
			opts: func() Options { o := usdOptions(ProfileBasic); o.TaxCurrencyCode = "USD"; return o },
			want: ErrInvalidTaxCurrency,
		},
		{
			name: "tax currency without rate",
			doc:  buildTestDoc,
			opts: func() Options { o := usdOptions(ProfileBasic); o.TaxCurrencyRate = ""; return o },
			want: ErrInvalidTaxCurrency,
		},
		{
			name: "ship-to without country",
			doc:  buildTestDoc,
			opts: func() Options {
				o := simpleOptions(ProfileBasic)
				o.ShipToAddress = &generator.Address{Address: "Somewhere"}
				return o
			},
			want: ErrMissingCountry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := tc.doc(t)
			if err := doc.Validate(); err != nil {
				t.Fatalf("doc.Validate: %v", err)
			}
			_, err := BuildXML(doc, tc.opts())
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestFrenchCreditNote(t *testing.T) {
	inv := parseInvoice(t, buildFrenchDoc(t), frenchOptions(ProfileEN16931))
	if inv.TypeCode != "381" {
		t.Errorf("type code: %q", inv.TypeCode)
	}
	if inv.PrecedingRef != "INV-2023-120" || inv.PrecedingDate != "20231201" {
		t.Errorf("preceding invoice: %q %q", inv.PrecedingRef, inv.PrecedingDate)
	}
	if inv.ShipToName != "Client SARL - Entrepôt" || inv.ShipToCountry != "FR" {
		t.Errorf("ship-to: %q %q", inv.ShipToName, inv.ShipToCountry)
	}
	if inv.BusinessProcess != "S1" || inv.SellerLegalID != "123456789" {
		t.Errorf("cadre de facturation %q, SIREN %q", inv.BusinessProcess, inv.SellerLegalID)
	}
	for _, tx := range inv.Taxes {
		if tx.DueDateTypeCode != "72" {
			t.Errorf("VAT point date code: got %q, want 72", tx.DueDateTypeCode)
		}
	}
	// Electronic addresses default to the SIREN with the French scheme.
	if inv.SellerURI != "123456789" || inv.SellerURIScheme != "0225" {
		t.Errorf("seller electronic address: %q scheme %q", inv.SellerURI, inv.SellerURIScheme)
	}
	if inv.BuyerURI != "987654321" || inv.BuyerURIScheme != "0225" {
		t.Errorf("buyer electronic address: %q scheme %q", inv.BuyerURI, inv.BuyerURIScheme)
	}
	// Three legal mentions plus the BAR flow note.
	codes := map[string]string{}
	for _, n := range inv.NoteEntries {
		codes[n.SubjectCode] = n.Content
	}
	if codes["BAR"] != "B2B" || codes["PMT"] == "" || codes["PMD"] == "" || codes["AAB"] == "" {
		t.Errorf("notes: %+v", inv.NoteEntries)
	}
}

func TestFrenchRulesErrors(t *testing.T) {
	cases := []struct {
		name string
		mut  func(o *Options, doc *generator.Document)
	}{
		{"profile below EN 16931", func(o *Options, _ *generator.Document) { o.Profile = ProfileBasic }},
		{"missing PMD note", func(o *Options, _ *generator.Document) { o.Notes = o.Notes[1:] }},
		{"duplicate AAB note", func(o *Options, _ *generator.Document) {
			o.Notes = append(o.Notes, Note{Content: "x", SubjectCode: "AAB"})
		}},
		{"user-supplied BAR note", func(o *Options, _ *generator.Document) {
			o.Notes = append(o.Notes, Note{Content: "B2C", SubjectCode: "BAR"})
		}},
		{"invalid flow", func(o *Options, _ *generator.Document) { o.FrenchFlow = "B2G" }},
		{"invalid cadre", func(o *Options, _ *generator.Document) { o.BusinessProcessID = "X1" }},
		{"prepaid cadre unsupported", func(o *Options, _ *generator.Document) { o.BusinessProcessID = "S2" }},
		{"invalid type code", func(o *Options, _ *generator.Document) { o.TypeCode = "999" }},
		{"credit note without preceding date", func(o *Options, _ *generator.Document) { o.PrecedingInvoiceDate = "" }},
		{"invoice number with spaces", func(_ *Options, doc *generator.Document) { doc.SetRef("AV 2024 001") }},
		{"email as electronic address", func(o *Options, _ *generator.Document) {
			o.BuyerElectronicAddress = "ap@client.example"
			o.BuyerElectronicAddressSchemeID = "EM"
		}},
		{"non-french VAT rate", func(_ *Options, doc *generator.Document) { doc.Items[0].Tax.Percent = "19" }},
		{"non-EUR without EUR tax currency", func(o *Options, _ *generator.Document) { o.CurrencyCode = "USD" }},
		{"due date before issue date", func(o *Options, _ *generator.Document) { o.PaymentDueDate = "2023-12-31" }},
		{"quantity with 5 decimals", func(_ *Options, doc *generator.Document) { doc.Items[0].Quantity = "1.00001" }},
		{"seller not in France", func(o *Options, _ *generator.Document) { o.SellerCountryCode = "BE" }},
		{"seller SIREN missing", func(o *Options, _ *generator.Document) { o.SellerLegalID = "" }},
		{"seller SIREN malformed", func(o *Options, _ *generator.Document) { o.SellerLegalID = "12345678A" }},
		{"seller SIREN wrong scheme", func(o *Options, _ *generator.Document) { o.SellerLegalIDSchemeID = "0009" }},
		{"french buyer without SIREN", func(o *Options, _ *generator.Document) { o.BuyerLegalID = "" }},
		{"no cadre de facturation", func(o *Options, _ *generator.Document) { o.BusinessProcessID = "" }},
		{"credit note without preceding invoice", func(o *Options, _ *generator.Document) { o.PrecedingInvoiceReference = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := buildFrenchDoc(t)
			opts := frenchOptions(ProfileEN16931)
			tc.mut(&opts, doc)
			if err := doc.Validate(); err != nil {
				t.Fatal(err)
			}
			_, err := BuildXML(doc, opts)
			if !errors.Is(err, ErrFrenchRules) {
				t.Fatalf("got %v, want ErrFrenchRules", err)
			}
		})
	}

	// A B2BINT flow to a foreign buyer needs neither a SIREN nor a French
	// electronic address scheme.
	doc := buildFrenchDoc(t)
	doc.Customer.Address.Country = "DE"
	opts := frenchOptions(ProfileEN16931)
	opts.FrenchFlow = "B2BINT"
	opts.BuyerCountryCode = "DE"
	opts.BuyerLegalID = ""
	opts.BuyerLegalIDSchemeID = ""
	opts.BuyerElectronicAddress = "DE123456789"
	opts.BuyerElectronicAddressSchemeID = "9930"
	opts.ShipToAddress.Country = "DE"
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildXML(doc, opts); err != nil {
		t.Fatalf("foreign buyer without SIREN should pass: %v", err)
	}
}

func TestTaxCurrency(t *testing.T) {
	doc := buildTestDoc(t)
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	xmlBytes, err := BuildXML(doc, usdOptions(ProfileBasic))
	if err != nil {
		t.Fatal(err)
	}
	out := string(xmlBytes)
	if !strings.Contains(out, "<ram:TaxCurrencyCode>EUR</ram:TaxCurrencyCode>") {
		t.Error("missing TaxCurrencyCode")
	}
	if !strings.Contains(out, `<ram:TaxTotalAmount currencyID="USD">294.00</ram:TaxTotalAmount>`) {
		t.Error("missing VAT total in invoice currency")
	}
	// 294.00 * 0.9234 = 271.4796 -> 271.48
	if !strings.Contains(out, `<ram:TaxTotalAmount currencyID="EUR">271.48</ram:TaxTotalAmount>`) {
		t.Errorf("missing or wrong VAT total in tax currency:\n%s", out)
	}
	if idx := strings.Index(out, "<ram:TaxCurrencyCode>"); idx < 0 || idx > strings.Index(out, "<ram:InvoiceCurrencyCode>") {
		t.Error("TaxCurrencyCode must precede InvoiceCurrencyCode")
	}

	// MINIMUM carries neither.
	xmlBytes, err = BuildXML(doc, usdOptions(ProfileMinimum))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(xmlBytes), "TaxCurrencyCode") || strings.Count(string(xmlBytes), "<ram:TaxTotalAmount") != 1 {
		t.Error("MINIMUM must not carry the tax currency")
	}
}

func TestPrecisionAboveTwoRejected(t *testing.T) {
	doc, err := generator.New(generator.Invoice, &generator.Options{CurrencyPrecision: 3})
	if err != nil {
		t.Fatal(err)
	}
	doc.SetRef("P3").SetDate("01/01/2024")
	doc.SetCompany(&generator.Contact{Name: "A", Address: &generator.Address{Address: "x", Country: "FR"}})
	doc.SetCustomer(&generator.Contact{Name: "B", Address: &generator.Address{Address: "y", Country: "FR"}})
	doc.AppendItem(&generator.Item{Name: "a", UnitCost: "10", Quantity: "1", Tax: &generator.Tax{Percent: "20"}})
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildXML(doc, simpleOptions(ProfileMinimum)); !errors.Is(err, ErrUnsupportedPrecision) {
		t.Fatalf("got %v, want ErrUnsupportedPrecision", err)
	}
}

func TestFixedAmountTaxAllowedInMinimum(t *testing.T) {
	doc := buildTestDoc(t)
	doc.AppendItem(&generator.Item{Name: "Eco tax", UnitCost: "10", Quantity: "1", Tax: &generator.Tax{Amount: "2"}})
	inv := parseInvoice(t, doc, simpleOptions(ProfileMinimum))
	assertEqual(t, "MINIMUM BR-CO-15 with fixed tax",
		dec(t, inv.GrandTotal), dec(t, inv.TaxBasisTotal).Add(dec(t, inv.TaxTotal.Value)))
	// 1200 + 270 + 10 = 1480 basis; 20% of 1470 = 294 + 2 fixed = 296.
	assertEqual(t, "fixed tax total", dec(t, inv.TaxTotal.Value), decimal.RequireFromString("296.00"))
}
