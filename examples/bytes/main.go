package main

import (
	"bytes"
	"fmt"

	"github.com/angelodlfrtr/go-invoice-generator/generator"
	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

func main() {
	// The document holds the data and the arithmetic...
	doc, err := invoice.New(invoice.Invoice)
	if err != nil {
		panic(err)
	}

	doc.SetRef("testràf")
	doc.SetVersion("someversion")
	doc.SetDescription("A description àç")
	doc.SetNotes("I léove croissant cotton candy.")
	doc.SetDate("02/03/2021")
	doc.SetDueDate("02/04/2021")

	doc.SetCompany(&invoice.Contact{
		Name:  "Test Company",
		VATID: "FR12345678901",
		Address: &invoice.Address{
			Address:     "89 Rue de Brest",
			Address2:    "Appartement 2",
			PostalCode:  "75000",
			City:        "Paris",
			Country:     "France",
			CountryCode: "FR",
		},
	})

	doc.SetCustomer(&invoice.Contact{
		Name: "Test Customer",
		Address: &invoice.Address{
			Address:     "89 Rue de Paris",
			PostalCode:  "29200",
			City:        "Brest",
			Country:     "France",
			CountryCode: "FR",
		},
	})

	for i := 0; i < 10; i++ {
		doc.AppendItem(&invoice.Item{
			Name:        "Cupcake ipsum dolor sit amet bonbon, coucou bonbon lala jojo, mama titi toto",
			Description: "Cupcake ipsum dolor sit amet bonbon, Cupcake ipsum dolor sit",
			UnitCost:    "99876.89",
			Quantity:    "2",
			Tax:         &invoice.Tax{Percent: "20"},
		})
	}

	doc.AppendItem(&invoice.Item{
		Name:     "Test",
		UnitCost: "99876.89",
		Quantity: "2",
		Tax:      &invoice.Tax{Amount: "89"},
		Discount: &invoice.Discount{Percent: "30"},
	})

	doc.AppendItem(&invoice.Item{
		Name:     "Test",
		UnitCost: "889.89",
		Quantity: "2",
		Discount: &invoice.Discount{Amount: "234.67"},
	})

	doc.SetDefaultTax(&invoice.Tax{Percent: "10"})
	doc.SetDiscount(&invoice.Discount{Amount: "1340"})

	// ...the generator only renders it.
	gen := generator.NewGenerator(&generator.Options{
		TextTypeInvoice: "FACTURE",
		TextRefTitle:    "Ref",
		BaseTextColor:   []int{6, 63, 156},
		GreyTextColor:   []int{161, 96, 149},
		GreyBgColor:     []int{171, 240, 129},
		DarkBgColor:     []int{176, 12, 20},
	})

	gen.SetHeader(&generator.HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon.</center>",
		Pagination: true,
	})

	gen.SetFooter(&generator.HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon.</center>",
		Pagination: true,
	})

	pdf, err := gen.Build(doc)
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}
	if err := pdf.Output(buf); err != nil {
		panic(err)
	}

	// Print the PDF bytes to STDOUT
	fmt.Printf("%v\n", buf.Bytes())
}
