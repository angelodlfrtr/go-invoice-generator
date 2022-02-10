package generator

import (
	"io/ioutil"
	"testing"
)

func TestNew(t *testing.T) {
	doc, _ := New(Invoice, &Options{
		TextTypeInvoice: "FACTURE",
		TextRefTitle:    "Réàf.",
		AutoPrint:       true,
	})

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

	logoBytes, _ := ioutil.ReadFile("./example_logo.png")

	doc.SetCompany(&Contact{
		Name: "Test Company",
		Logo: &logoBytes,
		Address: &Address{
			Address:    "89 Rue de Brest",
			Address2:   "Appartement 2",
			PostalCode: "75000",
			City:       "Paris",
			Country:    "France",
		},
	})

	doc.SetCustomer(&Contact{
		Name: "Test Customer",
		Address: &Address{
			Address:    "89 Rue de Paris",
			PostalCode: "29200",
			City:       "Brest",
			Country:    "France",
		},
	})

	for i := 0; i < 10; i++ {
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

	// doc.SetDiscount(&Discount{
	// Percent: "90",
	// })
	doc.SetDiscount(&Discount{
		Amount: "1340",
	})

	pdf, err := doc.Build()
	if err != nil {
		t.Errorf(err.Error())
	}

	err = pdf.OutputFileAndClose("out.pdf")

	if err != nil {
		t.Errorf(err.Error())
	}
}
