package facturx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	generator "github.com/angelodlfrtr/go-invoice-generator/generator"
)

func buildTestDoc(t *testing.T) *generator.Document {
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

func TestAttach(t *testing.T) {
	doc := buildTestDoc(t)

	result, err := Attach(buildPDF(t, doc), doc, Options{
		Profile:           ProfileMinimum,
		SellerTaxID:       "FR12345678901",
		SellerCountryCode: "FR",
		BuyerCountryCode:  "US",
		CurrencyCode:      "EUR",
		PaymentDueDate:    "20240201",
		TaxCategoryCode:   "S",
		ShowIcon:          true,
	})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("Attach returned empty result")
	}

	if err := os.MkdirAll("../out", 0o750); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}

	if err := os.WriteFile("../out/facturx.pdf", result, 0o644); err != nil {
		t.Fatalf("write out.pdf: %v", err)
	}
}

func TestAttachProfiles(t *testing.T) {
	profiles := []Profile{
		ProfileMinimum,
		ProfileBasicWL,
		ProfileBasic,
		ProfileEN16931,
		ProfileExtended,
	}

	if err := os.MkdirAll("../out", 0o750); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}

	for _, profile := range profiles {
		t.Run(string(profile), func(t *testing.T) {
			doc := buildTestDoc(t)

			result, err := Attach(buildPDF(t, doc), doc, Options{
				Profile:           profile,
				SellerTaxID:       "FR12345678901",
				SellerCountryCode: "FR",
				BuyerCountryCode:  "US",
				CurrencyCode:      "EUR",
				PaymentDueDate:    "20240201",
				TaxCategoryCode:   "S",
			})
			if err != nil {
				t.Fatalf("Attach(%s): %v", profile, err)
			}

			if len(result) == 0 {
				t.Fatalf("Attach(%s) returned empty result", profile)
			}

			filename := fmt.Sprintf("../out/facturx_%s.pdf", strings.ReplaceAll(string(profile), " ", "_"))
			if err := os.WriteFile(filename, result, 0o644); err != nil {
				t.Fatalf("write %s: %v", filename, err)
			}
		})
	}
}

func TestMustangValidation(t *testing.T) {
	if _, err := exec.LookPath("mustang-cli"); err != nil {
		t.Skip("mustang-cli not in PATH")
	}

	profiles := []Profile{
		ProfileMinimum,
		ProfileBasicWL,
		ProfileBasic,
		ProfileEN16931,
		ProfileExtended,
	}

	if err := os.MkdirAll("../out", 0o750); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}

	// Generate all profile PDFs first.
	for _, profile := range profiles {
		doc := buildTestDoc(t)
		result, err := Attach(buildPDF(t, doc), doc, Options{
			Profile:           profile,
			SellerTaxID:       "FR12345678901",
			SellerCountryCode: "FR",
			BuyerCountryCode:  "US",
			CurrencyCode:      "EUR",
			PaymentDueDate:    "20240201",
			TaxCategoryCode:   "S",
		})
		if err != nil {
			t.Fatalf("Attach(%s): %v", profile, err)
		}
		filename := fmt.Sprintf("../out/facturx_%s.pdf", strings.ReplaceAll(string(profile), " ", "_"))
		if err := os.WriteFile(filename, result, 0o644); err != nil {
			t.Fatalf("write %s: %v", filename, err)
		}
	}

	// Validate each with mustang-cli.
	for _, profile := range profiles {
		profile := profile
		t.Run(string(profile), func(t *testing.T) {
			filename := fmt.Sprintf("../out/facturx_%s.pdf", strings.ReplaceAll(string(profile), " ", "_"))
			out, err := exec.Command("mustang-cli", "--action", "validate", "--source", filename).CombinedOutput()
			if err != nil {
				t.Fatalf("mustang-cli failed: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), `status="valid"`) {
				t.Fatalf("profile %s not valid:\n%s", profile, out)
			}
		})
	}
}

func TestBuildXML(t *testing.T) {
	doc := buildTestDoc(t)

	if err := doc.Validate(); err != nil {
		t.Fatalf("doc.Validate: %v", err)
	}

	xmlBytes, err := BuildXML(doc, Options{
		Profile:           ProfileEN16931,
		SellerTaxID:       "FR12345678901",
		SellerCountryCode: "FR",
		BuyerCountryCode:  "US",
		CurrencyCode:      "EUR",
	})
	if err != nil {
		t.Fatalf("BuildXML: %v", err)
	}

	if len(xmlBytes) == 0 {
		t.Fatal("BuildXML returned empty XML")
	}
}
