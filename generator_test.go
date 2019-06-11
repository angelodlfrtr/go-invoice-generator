package generator

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestNew(t *testing.T) {
	doc, _ := New(DELIVERY_NOTE, &Options{
		TextTypeInvoice: "FACTURE",
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

	doc.SetRef("testref")
	doc.SetVersion("someversion")

	doc.SetDescription("A description")
	doc.SetNotes("Des notes des nores sdqd qsd qsd sqd fgh fgh fg df  rty rty rty rty rty rty rty rty rty rty try rty rty rty try rty rt yrt yrty rty rty rty rty rty !")

	doc.SetDate("23/12/1992")
	doc.SetPaymentTerm("23/12/1992")

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
			Name:     "Test",
			UnitCost: "99876.89",
			Quantity: "2",
			Tax: &Tax{
				Percent: "20",
			},
		})
	}

	pdf, err := doc.Build()

	if err != nil {
		t.Errorf(err.Error())
	}

	err = pdf.OutputFileAndClose("out.pdf")

	if err != nil {
		t.Errorf(err.Error())
	}
}

func ExampleNew() {
	doc, _ := New(DELIVERY_NOTE, &Options{
		TextTypeInvoice: "FACTURE",
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

	doc.SetRef("testref")
	doc.SetVersion("someversion")

	doc.SetDescription("A description")

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

	for i := 0; i < 15; i++ {
		doc.AppendItem(&Item{
			Name:     "Test",
			UnitCost: "99876.89",
			Quantity: "2",
			Tax: &Tax{
				Percent: "20",
			},
		})
	}

	pdf, err := doc.Build()

	if err != nil {
		fmt.Println(err.Error())
	}

	err = pdf.OutputFileAndClose("out.pdf")

	if err != nil {
		fmt.Println(err.Error())
	}
}
