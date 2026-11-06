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

	"github.com/shopspring/decimal"

	"github.com/angelodlfrtr/go-invoice-generator/generator"
	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

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

// ---- Fixtures ----

func newTestDoc(t *testing.T) *invoice.Document {
	t.Helper()

	doc, err := invoice.New(invoice.Invoice)
	if err != nil {
		t.Fatalf("invoice.New: %v", err)
	}

	doc.SetRef("INV-2024-001")
	doc.SetDate("01/01/2024")
	doc.SetDueDate("01/02/2024")

	doc.SetCompany(&invoice.Contact{
		Name:  "Acme Corp",
		VATID: "FR12345678901",
		Address: &invoice.Address{
			Address:     "1 Rue de la Paix",
			PostalCode:  "75001",
			City:        "Paris",
			Country:     "France",
			CountryCode: "FR",
		},
	})

	doc.SetCustomer(&invoice.Contact{
		Name: "John Doe",
		Address: &invoice.Address{
			Address:    "42 Main Street",
			PostalCode: "10001",
			City:       "New York",
			Country:    "US",
		},
	})

	return doc
}

// buildTestDoc is the simple fixture: two standard-rated lines, one item discount.
func buildTestDoc(t *testing.T) *invoice.Document {
	t.Helper()
	doc := newTestDoc(t)

	doc.AppendItem(&invoice.Item{
		Name:     "Consulting",
		UnitCost: "150.00",
		Quantity: "8",
		Tax:      &invoice.Tax{Percent: "20"},
	})

	doc.AppendItem(&invoice.Item{
		Name:     "Support",
		UnitCost: "75.00",
		Quantity: "4",
		Tax:      &invoice.Tax{Percent: "20"},
		Discount: &invoice.Discount{Percent: "10"},
	})

	return doc
}

// buildFullTestDoc exercises mixed rates, an exempt line, an untaxed line,
// item discounts, a document discount, notes and every business field.
func buildFullTestDoc(t *testing.T) *invoice.Document {
	t.Helper()
	doc := newTestDoc(t)
	doc.SetNotes("Thank you for your business.")
	doc.SetDiscount(&invoice.Discount{Percent: "5"})
	doc.SetClientRef("COST-CENTRE-42")
	doc.SetPurchaseOrderRef("PO-2023-0099")
	doc.SetPaymentTerm("Payable within 30 days")

	doc.Company.LegalID = "123456789"
	doc.Company.LegalIDScheme = "0002"
	doc.Company.ElectronicAddress = "billing@acme.example"
	doc.Company.ContactName = "Jane Smith"
	doc.Company.ContactPhone = "+33 1 23 45 67 89"
	doc.Company.ContactEmail = "jane@acme.example"

	doc.Customer.VATID = "US123456789"
	doc.Customer.LegalID = "987654321"
	doc.Customer.ElectronicAddress = "ap@johndoe.example"

	doc.SetDelivery(&invoice.Delivery{
		Date:        "2023-12-31",
		PeriodStart: "01/12/2023",
		PeriodEnd:   "31/12/2023",
		ShipTo: &invoice.Contact{
			Name: "John Doe Warehouse",
			Address: &invoice.Address{
				Address:    "7 Industrial Way",
				PostalCode: "07001",
				City:       "Newark",
				Country:    "US",
			},
		},
	})
	doc.SetPayment(&invoice.Payment{
		IBAN:      "FR7630006000011234567890189",
		BIC:       "BNPAFRPP",
		Reference: "INV-2024-001",
	})

	doc.AppendItem(&invoice.Item{
		Name:        "Consulting",
		Description: "Senior consultant, daily rate",
		UnitCost:    "150.00",
		Quantity:    "8",
		Unit:        "DAY",
		Tax:         &invoice.Tax{Percent: "20"},
	})
	doc.AppendItem(&invoice.Item{
		Name:     "Support",
		UnitCost: "75.00",
		Quantity: "4",
		Unit:     "HUR",
		Tax:      &invoice.Tax{Percent: "20"},
		Discount: &invoice.Discount{Percent: "10"},
	})
	doc.AppendItem(&invoice.Item{
		Name:     "Books",
		UnitCost: "12.33",
		Quantity: "3",
		Tax:      &invoice.Tax{Percent: "5.5", Category: "S"},
		Discount: &invoice.Discount{Amount: "2.50"},
	})
	doc.AppendItem(&invoice.Item{
		Name:     "Training",
		UnitCost: "499.99",
		Quantity: "1",
		Tax: &invoice.Tax{
			Percent:             "0",
			Category:            "E",
			ExemptionReason:     "Exempt from VAT, art. 261-4-4 CGI",
			ExemptionReasonCode: "VATEX-EU-132",
		},
	})
	doc.AppendItem(&invoice.Item{
		Name:     "Shipping",
		UnitCost: "9.90",
		Quantity: "1",
	})

	return doc
}

func fullOptions(profile Profile) Options {
	return Options{
		Profile:                profile,
		BusinessProcessID:      "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0",
		NoTaxExemptionReason:   "VAT not applicable, art. 293 B CGI",
		DocumentDiscountReason: "Loyalty discount",
		VATPointDateCode:       "5",
	}
}

func simpleOptions(profile Profile) Options {
	return Options{Profile: profile, TaxCategoryCode: "S"}
}

// buildFrenchDoc is a credit note between two French companies.
func buildFrenchDoc(t *testing.T) *invoice.Document {
	t.Helper()
	doc := newTestDoc(t)
	doc.SetType(invoice.CreditNote)
	doc.SetRef("AV-2024-001")
	doc.SetPaymentTerm("Remboursement sous 30 jours")
	doc.SetPrecedingInvoice("INV-2023-120", "2023-12-01")

	doc.Company.LegalID = "123456789"
	doc.Company.LegalIDScheme = "0002"

	doc.SetCustomer(&invoice.Contact{
		Name:          "Client SARL",
		VATID:         "FR98765432109",
		LegalID:       "987654321",
		LegalIDScheme: "0002",
		Address: &invoice.Address{
			Address:    "5 Rue de la République",
			PostalCode: "69001",
			City:       "Lyon",
			Country:    "FR",
		},
	})
	doc.SetDelivery(&invoice.Delivery{
		ShipTo: &invoice.Contact{
			Name: "Client SARL - Entrepôt",
			Address: &invoice.Address{
				Address:    "ZI des Chênes",
				PostalCode: "69800",
				City:       "Saint-Priest",
				Country:    "FR",
			},
		},
	})
	for _, n := range frenchNotes() {
		doc.AddNote(n.Content, n.SubjectCode)
	}

	doc.AppendItem(&invoice.Item{
		Name:     "Consulting (refund)",
		UnitCost: "150.00",
		Quantity: "2",
		Tax:      &invoice.Tax{Percent: "20"},
	})
	doc.AppendItem(&invoice.Item{
		Name:     "Training (refund)",
		UnitCost: "100.00",
		Quantity: "1",
		Tax: &invoice.Tax{
			Percent:         "0",
			Category:        "E",
			ExemptionReason: "TVA non applicable, art. 261-4-4 du CGI",
		},
	})
	return doc
}

// frenchNotes are the three legal mentions French invoices must carry.
func frenchNotes() []Note {
	return []Note{
		{Content: "Pénalités de retard : 3 fois le taux d'intérêt légal", SubjectCode: "PMD"},
		{Content: "Indemnité forfaitaire pour frais de recouvrement : 40 €", SubjectCode: "PMT"},
		{Content: "Pas d'escompte pour paiement anticipé", SubjectCode: "AAB"},
	}
}

func frenchOptions(profile Profile) Options {
	return Options{
		Profile:           profile,
		FrenchRules:       true,
		BusinessProcessID: "S1",
		VATPointDateCode:  "72",
	}
}

// usdOptions invoices in USD with the VAT total also given in EUR.
func usdOptions(profile Profile) Options {
	o := simpleOptions(profile)
	o.TaxCurrencyCode = "EUR"
	o.TaxCurrencyRate = "0.9234"
	return o
}

func usdDoc(t *testing.T) *invoice.Document {
	t.Helper()
	doc := buildTestDoc(t)
	doc.SetCurrency("USD", 2)
	return doc
}

type fixture struct {
	doc  *invoice.Document
	opts Options
}

// extraFixtures lets test files register more fixtures.
var extraFixtures []func(t *testing.T, profile Profile, fx map[string]fixture)

// fixtures lists every (document, options) pair rendered for each profile.
// The Makefile validation targets glob out/facturx*.pdf.
func fixtures(t *testing.T, profile Profile) map[string]fixture {
	t.Helper()
	fx := map[string]fixture{
		"facturx":      {buildTestDoc(t), simpleOptions(profile)},
		"facturx_full": {buildFullTestDoc(t), fullOptions(profile)},
	}
	if profile != ProfileMinimum {
		fx["facturx_usd"] = fixture{usdDoc(t), usdOptions(profile)}
	}
	if profile == ProfileEN16931 || profile == ProfileExtended {
		fx["facturx_fr"] = fixture{buildFrenchDoc(t), frenchOptions(profile)}
	}
	for _, add := range extraFixtures {
		add(t, profile, fx)
	}
	return fx
}

func buildPDF(t *testing.T, doc *invoice.Document) []byte {
	t.Helper()

	pdf, err := generator.NewGenerator(&generator.Options{CurrencySymbol: "€ "}).Build(doc)
	if err != nil {
		t.Fatalf("Build: %v", err)
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

// ---- PDF tests ----

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

type xmlURI struct {
	Value  string `xml:",chardata"`
	Scheme string `xml:"schemeID,attr"`
}

type xmlNote struct {
	Content     string `xml:"Content"`
	SubjectCode string `xml:"SubjectCode"`
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
	ReasonCode      string `xml:"ReasonCode"`
	CategoryCode    string `xml:"CategoryTradeTax>CategoryCode"`
	Rate            string `xml:"CategoryTradeTax>RateApplicablePercent"`
}

type xmlQuantity struct {
	Value string `xml:",chardata"`
	Unit  string `xml:"unitCode,attr"`
}

type xmlLine struct {
	Name        string      `xml:"SpecifiedTradeProduct>Name"`
	Description string      `xml:"SpecifiedTradeProduct>Description"`
	GrossPrice  string      `xml:"SpecifiedLineTradeAgreement>GrossPriceProductTradePrice>ChargeAmount"`
	NetPrice    string      `xml:"SpecifiedLineTradeAgreement>NetPriceProductTradePrice>ChargeAmount"`
	Quantity    xmlQuantity `xml:"SpecifiedLineTradeDelivery>BilledQuantity"`
	Category    string      `xml:"SpecifiedLineTradeSettlement>ApplicableTradeTax>CategoryCode"`
	Rate        string      `xml:"SpecifiedLineTradeSettlement>ApplicableTradeTax>RateApplicablePercent"`
	LineTotal   string      `xml:"SpecifiedLineTradeSettlement>SpecifiedTradeSettlementLineMonetarySummation>LineTotalAmount"`
}

type xmlInvoice struct {
	BusinessProcess string         `xml:"ExchangedDocumentContext>BusinessProcessSpecifiedDocumentContextParameter>ID"`
	TypeCode        string         `xml:"ExchangedDocument>TypeCode"`
	NoteEntries     []xmlNote      `xml:"ExchangedDocument>IncludedNote"`
	Lines           []xmlLine      `xml:"SupplyChainTradeTransaction>IncludedSupplyChainTradeLineItem"`
	BuyerReference  string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerReference"`
	SellerLegalID   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>SpecifiedLegalOrganization>ID"`
	SellerContact   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>DefinedTradeContact>PersonName"`
	SellerEndpoint  xmlURI         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>URIUniversalCommunication>URIID"`
	SellerLineOne   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>PostalTradeAddress>LineOne"`
	SellerCountry   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>SellerTradeParty>PostalTradeAddress>CountryID"`
	BuyerTaxID      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerTradeParty>SpecifiedTaxRegistration>ID"`
	BuyerEndpoint   xmlURI         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerTradeParty>URIUniversalCommunication>URIID"`
	PurchaseOrder   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeAgreement>BuyerOrderReferencedDocument>IssuerAssignedID"`
	ShipToName      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ShipToTradeParty>Name"`
	ShipToCountry   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ShipToTradeParty>PostalTradeAddress>CountryID"`
	DeliveryDate    string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeDelivery>ActualDeliverySupplyChainEvent>OccurrenceDateTime>DateTimeString"`
	TaxCurrency     string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>TaxCurrencyCode"`
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
	ChargeTotal     string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>ChargeTotalAmount"`
	Prepaid         string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TotalPrepaidAmount"`
	TaxBasisTotal   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TaxBasisTotalAmount"`
	TaxTotals       []xmlAmount    `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>TaxTotalAmount"`
	GrandTotal      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>GrandTotalAmount"`
	DuePayable      string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>SpecifiedTradeSettlementHeaderMonetarySummation>DuePayableAmount"`
	PrecedingRef    string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>InvoiceReferencedDocument>IssuerAssignedID"`
	PrecedingDate   string         `xml:"SupplyChainTradeTransaction>ApplicableHeaderTradeSettlement>InvoiceReferencedDocument>FormattedIssueDateTime>DateTimeString"`
}

// taxTotal returns the VAT total in the invoice currency.
func (inv xmlInvoice) taxTotal() xmlAmount {
	if len(inv.TaxTotals) == 0 {
		return xmlAmount{}
	}
	return inv.TaxTotals[0]
}

func parseInvoice(t *testing.T, doc *invoice.Document, opts Options) xmlInvoice {
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
// for every profile that carries a VAT breakdown, and that the XML prints
// exactly what the PDF prints.
func TestTotalsReconcile(t *testing.T) {
	for _, profile := range []Profile{ProfileBasicWL, ProfileBasic, ProfileEN16931, ProfileExtended} {
		t.Run(string(profile), func(t *testing.T) {
			doc := buildFullTestDoc(t)
			inv := parseInvoice(t, doc, fullOptions(profile))

			if inv.taxTotal().Currency != "EUR" {
				t.Errorf("TaxTotalAmount currencyID: got %q", inv.taxTotal().Currency)
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
			assertEqual(t, "BR-CO-14 VAT total", dec(t, inv.taxTotal().Value), sumDec(t, taxes...))
			assertEqual(t, "sum of bases", sumDec(t, bases...), dec(t, inv.TaxBasisTotal))

			// BR-CO-15 / BR-CO-16: grand total and amount due.
			assertEqual(t, "BR-CO-15 grand total",
				dec(t, inv.GrandTotal), dec(t, inv.TaxBasisTotal).Add(dec(t, inv.taxTotal().Value)))
			assertEqual(t, "BR-CO-16 due payable", dec(t, inv.DuePayable), dec(t, inv.GrandTotal))

			// The XML must print exactly what the PDF prints.
			assertEqual(t, "PDF line total", dec(t, inv.LineTotal), doc.TotalWithoutTaxAndWithoutDocumentDiscount())
			assertEqual(t, "PDF discount", dec(t, inv.AllowanceTotal), doc.DocumentDiscount())
			assertEqual(t, "PDF total without tax", dec(t, inv.TaxBasisTotal), doc.TotalWithoutTax())
			assertEqual(t, "PDF tax", dec(t, inv.taxTotal().Value), doc.Tax())
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
	minimum := parseInvoice(t, buildFullTestDoc(t), fullOptions(ProfileMinimum))
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
	if len(minimum.NoteEntries) != 0 || len(minimum.Taxes) != 0 || len(minimum.Lines) != 0 || minimum.IBAN != "" ||
		minimum.DueDate != "" || minimum.LineTotal != "" || minimum.SellerEndpoint.Value != "" || minimum.BuyerTaxID != "" ||
		minimum.ShipToName != "" {
		t.Errorf("MINIMUM carries fields above its profile: %+v", minimum)
	}
	assertEqual(t, "MINIMUM BR-CO-15",
		dec(t, minimum.GrandTotal), dec(t, minimum.TaxBasisTotal).Add(dec(t, minimum.taxTotal().Value)))

	// BASIC-WL: header data, no lines, no BIC, no contact.
	if len(basicWL.Lines) != 0 {
		t.Errorf("BASIC-WL must not carry line items")
	}
	if basicWL.SellerEndpoint.Value != "billing@acme.example" || basicWL.SellerEndpoint.Scheme != "EM" ||
		basicWL.BuyerTaxID == "" || basicWL.DeliveryDate != "20231231" ||
		basicWL.PeriodStart != "20231201" || basicWL.PaymentRef == "" || basicWL.IBAN == "" ||
		basicWL.PaymentTerms != "Payable within 30 days" || basicWL.DueDate != "20240201" || len(basicWL.NoteEntries) != 1 {
		t.Errorf("BASIC-WL misses header fields: %+v", basicWL)
	}
	if basicWL.BIC != "" || basicWL.SellerContact != "" {
		t.Errorf("BASIC-WL must not carry BIC or seller contact")
	}
	if basicWL.SellerLineOne != "1 Rue de la Paix" {
		t.Errorf("BASIC-WL seller address line: %q", basicWL.SellerLineOne)
	}
	if basicWL.ShipToName != "John Doe Warehouse" || basicWL.ShipToCountry != "US" {
		t.Errorf("BASIC-WL ship-to: %q %q", basicWL.ShipToName, basicWL.ShipToCountry)
	}
	for _, tx := range basicWL.Taxes {
		if tx.DueDateTypeCode != "5" {
			t.Errorf("BASIC-WL VAT point date code: got %q, want 5", tx.DueDateTypeCode)
		}
	}

	// BASIC: lines without description, gross price for discounted items, units.
	if len(basic.Lines) != 5 {
		t.Fatalf("BASIC expected 5 lines, got %d", len(basic.Lines))
	}
	if basic.Lines[0].Description != "" {
		t.Errorf("BASIC must not carry product descriptions")
	}
	if basic.Lines[0].Quantity.Unit != "DAY" || basic.Lines[1].Quantity.Unit != "HUR" || basic.Lines[2].Quantity.Unit != "C62" {
		t.Errorf("unit codes: %q %q %q", basic.Lines[0].Quantity.Unit, basic.Lines[1].Quantity.Unit, basic.Lines[2].Quantity.Unit)
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
	// A payment term that is a date acts as the due date.
	doc := buildTestDoc(t)
	doc.SetDueDate("")
	doc.SetPaymentTerm("15/03/2024")
	inv := parseInvoice(t, doc, simpleOptions(ProfileBasic))
	if inv.DueDate != "20240315" || inv.PaymentTerms != "" {
		t.Errorf("due date from PaymentTerm: got due %q terms %q", inv.DueDate, inv.PaymentTerms)
	}

	// Free-text payment terms are exported as the description.
	doc = buildTestDoc(t)
	doc.SetDueDate("")
	doc.SetPaymentTerm("30 days end of month")
	inv = parseInvoice(t, doc, simpleOptions(ProfileBasic))
	if inv.DueDate != "" || inv.PaymentTerms != "30 days end of month" {
		t.Errorf("free-text payment term: due %q terms %q", inv.DueDate, inv.PaymentTerms)
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
	if inv.SellerEndpoint.Value != "123456789" || inv.SellerEndpoint.Scheme != "0225" {
		t.Errorf("seller electronic address: %+v", inv.SellerEndpoint)
	}
	if inv.BuyerEndpoint.Value != "987654321" || inv.BuyerEndpoint.Scheme != "0225" {
		t.Errorf("buyer electronic address: %+v", inv.BuyerEndpoint)
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
		mut  func(o *Options, doc *invoice.Document)
	}{
		{"profile below EN 16931", func(o *Options, _ *invoice.Document) { o.Profile = ProfileBasic }},
		{"missing PMD note", func(_ *Options, doc *invoice.Document) { doc.Notes = doc.Notes[1:] }},
		{"duplicate AAB note", func(_ *Options, doc *invoice.Document) { doc.AddNote("x", "AAB") }},
		{"user-supplied BAR note", func(_ *Options, doc *invoice.Document) { doc.AddNote("B2C", "BAR") }},
		{"invalid flow", func(o *Options, _ *invoice.Document) { o.FrenchFlow = "B2G" }},
		{"invalid cadre", func(o *Options, _ *invoice.Document) { o.BusinessProcessID = "X1" }},
		{"no cadre de facturation", func(o *Options, _ *invoice.Document) { o.BusinessProcessID = "" }},
		{"prepaid cadre unsupported", func(o *Options, _ *invoice.Document) { o.BusinessProcessID = "S2" }},
		{"invalid type code", func(o *Options, _ *invoice.Document) { o.TypeCode = "999" }},
		{"seller not in France", func(_ *Options, doc *invoice.Document) { doc.Company.Address.CountryCode = "BE" }},
		{"seller SIREN missing", func(_ *Options, doc *invoice.Document) { doc.Company.LegalID = "" }},
		{"seller SIREN malformed", func(_ *Options, doc *invoice.Document) { doc.Company.LegalID = "12345678A" }},
		{"seller SIREN wrong scheme", func(_ *Options, doc *invoice.Document) { doc.Company.LegalIDScheme = "0009" }},
		{"french buyer without SIREN", func(_ *Options, doc *invoice.Document) { doc.Customer.LegalID = "" }},
		{"credit note without preceding invoice", func(_ *Options, doc *invoice.Document) { doc.PrecedingInvoice = nil }},
		{"credit note without preceding date", func(_ *Options, doc *invoice.Document) { doc.PrecedingInvoice.Date = "" }},
		{"invoice number with spaces", func(_ *Options, doc *invoice.Document) { doc.SetRef("AV 2024 001") }},
		{"email as electronic address", func(_ *Options, doc *invoice.Document) {
			doc.Customer.ElectronicAddress = "ap@client.example"
			doc.Customer.ElectronicAddressScheme = "EM"
		}},
		{"non-french VAT rate", func(_ *Options, doc *invoice.Document) { doc.Items[0].Tax.Percent = "19" }},
		{"non-EUR without EUR tax currency", func(_ *Options, doc *invoice.Document) { doc.SetCurrency("USD", 2) }},
		{"due date before issue date", func(_ *Options, doc *invoice.Document) { doc.SetDueDate("2023-12-31") }},
		{"quantity with 5 decimals", func(_ *Options, doc *invoice.Document) { doc.Items[0].Quantity = "1.00001" }},
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
	doc.Customer.LegalID = ""
	doc.Customer.LegalIDScheme = ""
	doc.Customer.ElectronicAddress = "DE123456789"
	doc.Customer.ElectronicAddressScheme = "9930"
	doc.Delivery.ShipTo.Address.Country = "DE"
	opts := frenchOptions(ProfileEN16931)
	opts.FrenchFlow = "B2BINT"
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildXML(doc, opts); err != nil {
		t.Fatalf("foreign buyer without SIREN should pass: %v", err)
	}
}

func TestTaxCurrency(t *testing.T) {
	doc := usdDoc(t)
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

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		doc  func(t *testing.T) *invoice.Document
		opts func() Options
		want error
	}{
		{
			name: "fixed-amount tax above MINIMUM",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&invoice.Item{Name: "Eco tax", UnitCost: "10", Quantity: "1", Tax: &invoice.Tax{Amount: "2"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrFixedAmountTax,
		},
		{
			name: "missing seller identifier",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.Company.VATID = ""
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileMinimum) },
			want: ErrMissingSellerID,
		},
		{
			name: "invalid country code",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.Customer.Address.CountryCode = "USA"
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidCountryCode,
		},
		{
			name: "missing buyer country above MINIMUM",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.Customer.Address.Country = "United States"
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasicWL) },
			want: ErrMissingCountry,
		},
		{
			name: "invalid document date",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.SetDate("January 1st")
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileMinimum) },
			want: ErrInvalidDate,
		},
		{
			name: "invalid delivery date",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.SetDelivery(&invoice.Delivery{Date: "2024-13-45"})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidDate,
		},
		{
			name: "untaxed item without exemption reason",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&invoice.Item{Name: "Shipping", UnitCost: "10", Quantity: "1"})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrMissingExemptionReason,
		},
		{
			name: "standard category with zero rate",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&invoice.Item{Name: "Free", UnitCost: "10", Quantity: "1", Tax: &invoice.Tax{Percent: "0"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidTaxCategory,
		},
		{
			name: "exempt category with non-zero rate",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&invoice.Item{Name: "Odd", UnitCost: "10", Quantity: "1",
					Tax: &invoice.Tax{Percent: "20", Category: "E", ExemptionReason: "x"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidTaxCategory,
		},
		{
			name: "unknown category",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.AppendItem(&invoice.Item{Name: "Odd", UnitCost: "10", Quantity: "1",
					Tax: &invoice.Tax{Percent: "20", Category: "XX"}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
			want: ErrInvalidTaxCategory,
		},
		{
			name: "no payment terms above MINIMUM",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.SetDueDate("")
				doc.SetPaymentTerm("")
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasicWL) },
			want: ErrMissingPaymentTerms,
		},
		{
			name: "half invoicing period",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.SetDelivery(&invoice.Delivery{PeriodStart: "01/01/2024"})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
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
			doc:  usdDoc,
			opts: func() Options { o := usdOptions(ProfileBasic); o.TaxCurrencyCode = "USD"; return o },
			want: ErrInvalidTaxCurrency,
		},
		{
			name: "tax currency without rate",
			doc:  usdDoc,
			opts: func() Options { o := usdOptions(ProfileBasic); o.TaxCurrencyRate = ""; return o },
			want: ErrInvalidTaxCurrency,
		},
		{
			name: "ship-to without country",
			doc: func(t *testing.T) *invoice.Document {
				doc := buildTestDoc(t)
				doc.SetDelivery(&invoice.Delivery{ShipTo: &invoice.Contact{Address: &invoice.Address{Address: "Somewhere"}}})
				return doc
			},
			opts: func() Options { return simpleOptions(ProfileBasic) },
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

func TestPrecisionAboveTwoRejected(t *testing.T) {
	doc := buildTestDoc(t)
	doc.SetCurrency("EUR", 3)
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildXML(doc, simpleOptions(ProfileMinimum)); !errors.Is(err, ErrUnsupportedPrecision) {
		t.Fatalf("got %v, want ErrUnsupportedPrecision", err)
	}
}

func TestFixedAmountTaxAllowedInMinimum(t *testing.T) {
	doc := buildTestDoc(t)
	doc.AppendItem(&invoice.Item{Name: "Eco tax", UnitCost: "10", Quantity: "1", Tax: &invoice.Tax{Amount: "2"}})
	inv := parseInvoice(t, doc, simpleOptions(ProfileMinimum))
	assertEqual(t, "MINIMUM BR-CO-15 with fixed tax",
		dec(t, inv.GrandTotal), dec(t, inv.TaxBasisTotal).Add(dec(t, inv.taxTotal().Value)))
	// 1200 + 270 + 10 = 1480 basis; 20% of 1470 = 294 + 2 fixed = 296.
	assertEqual(t, "fixed tax total", dec(t, inv.taxTotal().Value), decimal.RequireFromString("296.00"))
}

// ---- Charges, prepaid, document types ----

// buildChargesDoc adds document-level charges, a discount and a prepayment.
func buildChargesDoc(t *testing.T) *invoice.Document {
	t.Helper()
	doc := buildTestDoc(t)
	doc.SetDiscount(&invoice.Discount{Percent: "5"})
	doc.AddCharge(&invoice.Charge{Name: "Shipping", Amount: "15.00", ReasonCode: "FC", Tax: &invoice.Tax{Percent: "20"}})
	doc.AddCharge(&invoice.Charge{Name: "Eco-contribution", Amount: "2.00"})
	doc.SetPrepaid("100.00")
	return doc
}

func chargesOptions(profile Profile) Options {
	o := simpleOptions(profile)
	o.NoTaxExemptionReason = "Not subject to VAT"
	return o
}

func init() {
	extraFixtures = append(extraFixtures, func(t *testing.T, profile Profile, fx map[string]fixture) {
		fx["facturx_charges"] = fixture{buildChargesDoc(t), chargesOptions(profile)}
	})
}

func TestChargesAndPrepaid(t *testing.T) {
	doc := buildChargesDoc(t)
	inv := parseInvoice(t, doc, chargesOptions(ProfileEN16931))

	// 1470 lines - 73.50 discount + 17 charges = 1413.50 basis
	assertEqual(t, "line total", dec(t, inv.LineTotal), decimal.RequireFromString("1470.00"))
	assertEqual(t, "allowance total", dec(t, inv.AllowanceTotal), decimal.RequireFromString("73.50"))
	assertEqual(t, "charge total", dec(t, inv.ChargeTotal), decimal.RequireFromString("17.00"))
	assertEqual(t, "BR-CO-13 basis", dec(t, inv.TaxBasisTotal),
		dec(t, inv.LineTotal).Sub(dec(t, inv.AllowanceTotal)).Add(dec(t, inv.ChargeTotal)))
	assertEqual(t, "prepaid", dec(t, inv.Prepaid), decimal.RequireFromString("100.00"))
	assertEqual(t, "BR-CO-16 due", dec(t, inv.DuePayable), dec(t, inv.GrandTotal).Sub(dec(t, inv.Prepaid)))
	assertEqual(t, "PDF amount due", dec(t, inv.DuePayable), doc.AmountDue())

	charges := 0
	for _, ac := range inv.Allowances {
		if ac.ChargeIndicator == "true" {
			charges++
			if ac.Reason == "Shipping" && (ac.ReasonCode != "FC" || ac.CategoryCode != "S" || ac.Rate != "20.00") {
				t.Errorf("shipping charge: %+v", ac)
			}
			if ac.Reason == "Eco-contribution" && (ac.CategoryCode != "E" || ac.Rate != "0.00") {
				t.Errorf("eco charge: %+v", ac)
			}
		}
	}
	if charges != 2 {
		t.Errorf("expected 2 charges, got %d", charges)
	}

	// MINIMUM carries no prepaid element but the due amount still nets it.
	minimum := parseInvoice(t, buildChargesDoc(t), chargesOptions(ProfileMinimum))
	if minimum.Prepaid != "" {
		t.Error("MINIMUM must not carry TotalPrepaidAmount")
	}
	assertEqual(t, "MINIMUM due", dec(t, minimum.DuePayable), dec(t, minimum.GrandTotal).Sub(decimal.RequireFromString("100")))
}

func TestUnsupportedDocumentType(t *testing.T) {
	for _, typ := range []string{invoice.Quotation, invoice.DeliveryNote} {
		doc := buildTestDoc(t)
		doc.SetType(typ)
		if err := doc.Validate(); err != nil {
			t.Fatal(err)
		}
		if _, err := BuildXML(doc, simpleOptions(ProfileMinimum)); !errors.Is(err, ErrUnsupportedDocumentType) {
			t.Errorf("%s: got %v, want ErrUnsupportedDocumentType", typ, err)
		}
		// An explicit type code overrides the check.
		o := simpleOptions(ProfileMinimum)
		o.TypeCode = "380"
		if _, err := BuildXML(doc, o); err != nil {
			t.Errorf("%s with TypeCode: %v", typ, err)
		}
	}
}

func TestFrenchPrepaidCadre(t *testing.T) {
	// S2: already paid, prepaid equals the total and the due date is the payment date.
	doc := buildFrenchDoc(t)
	doc.SetType(invoice.Invoice)
	doc.PrecedingInvoice = nil
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	doc.SetPrepaid(doc.TotalWithTax().String())
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	opts := frenchOptions(ProfileEN16931)
	opts.BusinessProcessID = "S2"
	if _, err := BuildXML(doc, opts); err != nil {
		t.Fatalf("already-paid cadre with full prepayment should pass: %v", err)
	}

	doc.SetPrepaid("1.00")
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildXML(doc, opts); !errors.Is(err, ErrFrenchRules) {
		t.Fatalf("partial prepayment under S2: got %v, want ErrFrenchRules", err)
	}

	// S4: final invoice after deposit cannot itself be a deposit invoice.
	opts.BusinessProcessID = "S4"
	opts.TypeCode = "386"
	doc.SetPrepaid("")
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildXML(doc, opts); !errors.Is(err, ErrFrenchRules) {
		t.Fatalf("deposit type under S4: got %v, want ErrFrenchRules", err)
	}
}

// ---- Reception ----

func TestReadRoundTrip(t *testing.T) {
	doc := buildFullTestDoc(t)
	pdf, err := Attach(buildPDF(t, doc), doc, fullOptions(ProfileEN16931))
	if err != nil {
		t.Fatal(err)
	}

	p, err := Read(pdf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if p.Profile != ProfileEN16931 || p.TypeCode != "380" || p.BusinessProcessID == "" || p.VATPointDateCode != "5" {
		t.Errorf("metadata: %+v", p)
	}
	got := p.Document
	if got.Ref != "INV-2024-001" || got.Date != "20240101" || got.DueDate != "20240201" || got.Type != invoice.Invoice {
		t.Errorf("header: ref %q date %q due %q type %q", got.Ref, got.Date, got.DueDate, got.Type)
	}
	if got.Company.VATID != "FR12345678901" || got.Company.LegalID != "123456789" || got.Company.LegalIDScheme != "0002" ||
		got.Company.Address.CountryCode != "FR" || got.Company.ContactName != "Jane Smith" {
		t.Errorf("seller: %+v", got.Company)
	}
	if got.Customer.VATID != "US123456789" || got.Customer.ElectronicAddress != "ap@johndoe.example" {
		t.Errorf("buyer: %+v", got.Customer)
	}
	if got.Delivery == nil || got.Delivery.ShipTo == nil || got.Delivery.ShipTo.Name != "John Doe Warehouse" ||
		got.Delivery.Date != "20231231" || got.Delivery.PeriodStart != "20231201" {
		t.Errorf("delivery: %+v", got.Delivery)
	}
	if got.Payment == nil || got.Payment.IBAN != "FR7630006000011234567890189" || got.Payment.BIC != "BNPAFRPP" {
		t.Errorf("payment: %+v", got.Payment)
	}
	if len(got.Items) != 5 || got.Items[0].Unit != "DAY" || got.Items[0].Description == "" {
		t.Fatalf("items: %d", len(got.Items))
	}
	if got.Items[1].UnitCost != "75.00" || got.Items[1].Discount == nil || got.Items[1].Discount.Amount != "30" {
		t.Errorf("discounted line: cost %q discount %+v", got.Items[1].UnitCost, got.Items[1].Discount)
	}
	if got.Items[3].Tax == nil || got.Items[3].Tax.Category != "E" || got.Items[3].Tax.ExemptionReason == "" {
		t.Errorf("exempt line tax: %+v", got.Items[3].Tax)
	}
	if got.Discount == nil || len(got.Notes) != 1 || got.ClientRef != "COST-CENTRE-42" || got.PurchaseOrderRef != "PO-2023-0099" {
		t.Errorf("discount %+v notes %d refs %q %q", got.Discount, len(got.Notes), got.ClientRef, got.PurchaseOrderRef)
	}

	assertEqual(t, "stated grand total", p.Totals.GrandTotal, doc.TotalWithTax())
	if err := p.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	// The parsed document exports again to the same totals.
	again := parseInvoice(t, got, fullOptions(ProfileEN16931))
	assertEqual(t, "re-exported grand total", dec(t, again.GrandTotal), doc.TotalWithTax())
	assertEqual(t, "re-exported tax", dec(t, again.taxTotal().Value), doc.Tax())
}

func TestReadChargesAndFrench(t *testing.T) {
	doc := buildChargesDoc(t)
	pdf, err := Attach(buildPDF(t, doc), doc, chargesOptions(ProfileBasic))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Read(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Document.Charges) != 2 || p.Document.Charges[0].ReasonCode != "FC" || p.Document.Prepaid != "100.00" {
		t.Errorf("charges %+v prepaid %q", p.Document.Charges, p.Document.Prepaid)
	}
	if err := p.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	fr := buildFrenchDoc(t)
	pdf, err = Attach(buildPDF(t, fr), fr, frenchOptions(ProfileEN16931))
	if err != nil {
		t.Fatal(err)
	}
	p, err = Read(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if p.FrenchFlow != "B2B" || len(p.Document.Notes) != 3 || p.Document.Type != invoice.CreditNote ||
		p.Document.PrecedingInvoice == nil || p.Document.PrecedingInvoice.Date != "20231201" {
		t.Errorf("french: flow %q notes %d type %q preceding %+v", p.FrenchFlow, len(p.Document.Notes), p.Document.Type, p.Document.PrecedingInvoice)
	}
	if err := p.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	// Re-exporting under the French rules works without duplicating the BAR note.
	if _, err := BuildXML(p.Document, frenchOptions(ProfileEN16931)); err != nil {
		t.Fatalf("re-export: %v", err)
	}
}

func TestReadErrors(t *testing.T) {
	if _, err := Extract(buildPDF(t, buildTestDoc(t))); !errors.Is(err, ErrNoXML) {
		t.Errorf("plain PDF: got %v, want ErrNoXML", err)
	}
	if _, err := Parse([]byte(`<?xml version="1.0"?><Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"/>`)); !errors.Is(err, ErrNotCII) {
		t.Errorf("UBL: got %v, want ErrNotCII", err)
	}
	if _, err := Parse([]byte("not xml")); err == nil {
		t.Error("garbage should fail")
	}

	// A tampered total is caught by Verify.
	doc := buildTestDoc(t)
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	xmlBytes, err := BuildXML(doc, simpleOptions(ProfileBasic))
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(xmlBytes), "<ram:GrandTotalAmount>1764.00</ram:GrandTotalAmount>", "<ram:GrandTotalAmount>1700.00</ram:GrandTotalAmount>", 1)
	if tampered == string(xmlBytes) {
		t.Fatalf("fixture changed, grand total not found:\n%s", xmlBytes)
	}
	p, err := Parse([]byte(tampered))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(); !errors.Is(err, ErrTotalsMismatch) {
		t.Errorf("tampered: got %v, want ErrTotalsMismatch", err)
	}
}
