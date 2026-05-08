package generator

import (
	"errors"
	"os"
	"testing"
)

func TestNewWithNamedTaxes(t *testing.T) {
	doc, err := New(Invoice, &Options{
		TextTypeInvoice:   "INVOICE",
		CurrencySymbol:    "€ ",
		CurrencyPrecision: 2,
	})
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	doc.SetRef("INV-TAX-001")
	doc.SetDate("01/05/2025")
	doc.SetPaymentTerm("01/06/2025")

	doc.SetCompany(&Contact{
		Name: "Acme Inc",
		Address: &Address{
			Address:    "12 Rue de la Paix",
			PostalCode: "75001",
			City:       "Paris",
			Country:    "FR",
		},
	})

	doc.SetCustomer(&Contact{
		Name: "Client Corp",
		Address: &Address{
			Address:    "5 Rue de la République",
			PostalCode: "69001",
			City:       "Lyon",
			Country:    "FR",
		},
	})

	// Standard rate VAT 20%
	doc.AppendItem(&Item{
		Name:     "Consulting services",
		UnitCost: "1500.00",
		Quantity: "3",
		Tax:      &Tax{Name: "VAT 20%", Percent: "20"},
	})
	doc.AppendItem(&Item{
		Name:     "Software development",
		UnitCost: "800.00",
		Quantity: "5",
		Tax:      &Tax{Name: "VAT 20%", Percent: "20"},
	})

	// Reduced rate VAT 10%
	doc.AppendItem(&Item{
		Name:     "Professional training",
		UnitCost: "600.00",
		Quantity: "2",
		Tax:      &Tax{Name: "VAT 10%", Percent: "10"},
	})

	// Super-reduced rate VAT 5.5%
	doc.AppendItem(&Item{
		Name:     "Technical books",
		UnitCost: "45.00",
		Quantity: "10",
		Tax:      &Tax{Name: "VAT 5.5%", Percent: "5.5"},
	})

	// Fixed tax
	doc.AppendItem(&Item{
		Name:     "Hardware component",
		UnitCost: "45.00",
		Quantity: "10",
		Tax:      &Tax{Name: "Eco tax", Amount: "5.67"},
	})

	// Item with no tax name — should appear under "Other" in the breakdown
	doc.AppendItem(&Item{
		Name:     "Shipping",
		UnitCost: "12.50",
		Quantity: "1",
		Tax:      &Tax{Percent: "20"},
	})

	pdf, err := doc.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if err := os.MkdirAll("../out", 0o750); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := pdf.OutputFileAndClose("../out/invoice_named_taxes.pdf"); err != nil {
		t.Fatalf("OutputFileAndClose: %v", err)
	}
}

func TestNewWithInvalidType(t *testing.T) {
	_, err := New("INVALID", &Options{})

	if errors.Is(err, ErrInvalidDocumentType) {
		return
	}

	t.Fatalf("expected ErrInvalidDocumentType, got %v", err)
}

func TestNew(t *testing.T) {
	doc, err := New(Invoice, &Options{
		TextTypeInvoice: "INVOICE",
		TextRefTitle:    "Réàf.",
		// BaseTextColor:     []int{6, 63, 156},
		// GreyTextColor:     []int{161, 96, 149},
		// GreyBgColor:       []int{171, 240, 129},
		// DarkBgColor:       []int{176, 12, 20},
		CurrencyPrecision: 2,
	})
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	doc.SetHeader(&HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon. I love croissant cotton candy. Carrot cake sweet I love sweet roll cake powder.</center>",
		Pagination: true,
	})

	doc.SetFooter(&HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon. I love croissant cotton candy. Carrot cake sweet I love sweet roll cake powder.</center>",
		Pagination: true,
	})

	doc.SetRef("testràf")
	doc.SetVersion("someversion")

	doc.SetDescription("A description àç")
	doc.SetNotes("I léove croissant cotton candy. Carrot cake sweet Ià love sweet roll cake powder! I love croissant cotton candy. Carrot cake sweet I love sweet roll cake powder! I love croissant cotton candy. Carrot cake sweet I love sweet roll cake powder! I love croissant cotton candy. Carrot cake sweet I love sweet roll cake powder! ")

	doc.SetDate("02/03/2021")
	doc.SetPaymentTerm("02/04/2021")

	logoBytes, _ := os.ReadFile("../support/example_logo.png")

	doc.SetCompany(&Contact{
		Name: "Test Company",
		Logo: logoBytes,
		Address: &Address{
			Address:    "89 Avenue Victor Hugo",
			Address2:   "Appartement 2",
			PostalCode: "75000",
			City:       "Paris",
			Country:    "FR",
		},
		AddtionnalInfo: []string{"Cupcake: ipsum dolor"},
	})

	doc.SetCustomer(&Contact{
		Name: "Test Customer",
		Address: &Address{
			Address:    "89 Rue de Paris",
			PostalCode: "13001",
			City:       "Marseille",
			Country:    "FR",
		},
		AddtionnalInfo: []string{
			"Cupcake: ipsum dolor",
			"Cupcake: ipsum dolo r",
		},
	})

	for range 3 {
		doc.AppendItem(&Item{
			Name:        "Cupcake ipsum dolor sit amet bonbon, coucou bonbon lala jojo, mama titi toto",
			Description: "Cupcake ipsum dolor sit amet bonbon, Cupcake ipsum dolor sit amet bonbon, Cupcake ipsum dolor sit amet bonbon",
			UnitCost:    "99876.89",
			Quantity:    "2",
			Tax: &Tax{
				Percent: "20",
			},
		})
	}

	doc.AppendItem(&Item{
		Name:        "Cupcake ipsum dolor sit amet bonbon, coucou bonbon lala jojo, mama titi toto",
		Description: "Cupcake ipsum dolor sit amet bonbon, Cupcake ipsum dolor sit amet bonbon, Cupcake ipsum dolor sit amet bonbon",
		UnitCost:    "1000.99",
		Quantity:    "10",
		Tax: &Tax{
			Percent: "15.6",
		},
	})

	doc.AppendItem(&Item{
		Name:     "Test",
		UnitCost: "99876.89",
		Quantity: "2",
		Tax: &Tax{
			Amount: "89",
		},
		Discount: &Discount{
			Percent: "30",
		},
	})

	doc.AppendItem(&Item{
		Name:     "Test",
		UnitCost: "3576.89",
		Quantity: "2",
		Discount: &Discount{
			Percent: "50",
		},
	})

	doc.AppendItem(&Item{
		Name:     "Test",
		UnitCost: "889.89",
		Quantity: "2",
		Discount: &Discount{
			Amount: "234.67",
		},
	})

	doc.SetDefaultTax(&Tax{
		Percent: "10",
	})

	doc.SetDiscount(&Discount{
		Percent: "90",
	})
	// doc.SetDiscount(&Discount{
	// 	Amount: "1340",
	// })

	pdf, err := doc.Build()
	if err != nil {
		t.Errorf("%v", err.Error())
	}

	if err := os.MkdirAll("../out", 0o750); err != nil {
		t.Errorf("%v", err.Error())
	}

	err = pdf.OutputFileAndClose("../out/invoice.pdf")
	if err != nil {
		t.Errorf("%v", err.Error())
	}
}
