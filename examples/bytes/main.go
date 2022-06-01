package main

import (
	"bytes"
	"fmt"

	generator "github.com/angelodlfrtr/go-invoice-generator"
)

func main() {
	doc, _ := generator.New(generator.Invoice, &generator.Options{
		TextTypeInvoice: "FACTURE",
		TextRefTitle:    "Ref",
		AutoPrint:       true,
		BaseTextColor:   []int{6, 63, 156},
		GreyTextColor:   []int{161, 96, 149},
		GreyBgColor:     []int{171, 240, 129},
		DarkBgColor:     []int{176, 12, 20},
	})

	doc.SetHeader(&generator.HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon.</center>",
		Pagination: true,
	})

	doc.SetFooter(&generator.HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon.</center>",
		Pagination: true,
	})

	doc.SetRef("testràf")
	doc.SetVersion("someversion")

	doc.SetDescription("A description àç")
	doc.SetNotes("I léove croissant cotton candy.")

	doc.SetDate("02/03/2021")
	doc.SetPaymentTerm("02/04/2021")

	doc.SetCompany(&generator.Contact{
		Name: "Test Company",
		Address: &generator.Address{
			Address:    "89 Rue de Brest",
			Address2:   "Appartement 2",
			PostalCode: "75000",
			City:       "Paris",
			Country:    "France",
		},
	})

	doc.SetCustomer(&generator.Contact{
		Name: "Test Customer",
		Address: &generator.Address{
			Address:    "89 Rue de Paris",
			PostalCode: "29200",
			City:       "Brest",
			Country:    "France",
		},
	})

	for i := 0; i < 10; i++ {
		doc.AppendItem(&generator.Item{
			Name:        "Cupcake ipsum dolor sit amet bonbon, coucou bonbon lala jojo, mama titi toto",
			Description: "Cupcake ipsum dolor sit amet bonbon, Cupcake ipsum dolor sit",
			UnitCost:    "99876.89",
			Quantity:    "2",
			Tax: &generator.Tax{
				Percent: "20",
			},
		})
	}

	doc.AppendItem(&generator.Item{
		Name:     "Test",
		UnitCost: "99876.89",
		Quantity: "2",
		Tax: &generator.Tax{
			Amount: "89",
		},
		Discount: &generator.Discount{
			Percent: "30",
		},
	})

	doc.AppendItem(&generator.Item{
		Name:     "Test",
		UnitCost: "889.89",
		Quantity: "2",
		Discount: &generator.Discount{
			Amount: "234.67",
		},
	})

	doc.SetDefaultTax(&generator.Tax{
		Percent: "10",
	})

	doc.SetDiscount(&generator.Discount{
		Amount: "1340",
	})

	pdf, err := doc.Build()
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}
	if err := pdf.Output(buf); err != nil {
		panic(err)
	}

	// Convert to byte slice
	docAsBytes := buf.Bytes()

	// Print to STDOUT
	fmt.Printf("%v\n", docAsBytes)
}
