[![Go Report Card](https://goreportcard.com/badge/github.com/angelodlfrtr/go-invoice-generator)](https://goreportcard.com/report/github.com/angelodlfrtr/go-invoice-generator)
[![GoDoc](https://godoc.org/github.com/angelodlfrtr/go-invoice-generator?status.svg)](https://godoc.org/github.com/angelodlfrtr/go-invoice-generator)

# Golang invoice generator

A super fast golang package to generate invoices, delivery notes and quotations as pdf
using https://github.com/jung-kurt/gofpdf.

## Download from Github

```
go get -u github.com/angelodlfrtr/go-invoice-generator
```

## Exemple output

![DeliveryNoteExample](example.png)

## Quick start

```golang
package main

import (
	"log"

	generator "github.com/angelodlfrtr/go-invoice-generator"
)

func main() {
	doc, _ := generator.New(generator.Invoice, &generator.Options{
		TextTypeInvoice: "FACTURE",
		AutoPrint:       true,
	})

	doc.SetHeader(&generator.HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon. I love croissant cotton candy. Carrot cake sweet I love sweet roll cake powder.</center>",
		Pagination: true,
	})

	doc.SetFooter(&generator.HeaderFooter{
		Text:       "<center>Cupcake ipsum dolor sit amet bonbon. I love croissant cotton candy. Carrot cake sweet I love sweet roll cake powder.</center>",
		Pagination: true,
	})

	doc.SetRef("testref")
	doc.SetVersion("someversion")

	doc.SetDescription("A description")
	doc.SetNotes("Cupcake ipsum dolor sit amet. I love carrot cake sugar plum muffin jelly liquorice ice cream. Tootsie roll tootsie roll lemon drops oat cake liquorice.")

	doc.SetDate("23/12/1992")
	doc.SetPaymentTerm("23/12/1992")

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
			Name:     "Test",
			UnitCost: "99876.89",
			Quantity: "2",
			Tax: &generator.Tax{
				Percent: "20",
			},
		})
	}

	pdf, err := doc.Build()

	if err != nil {
		log.Fatal(err)
	}

	if err := pdf.OutputFileAndClose("out.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

## License

This SDK is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0),
see [LICENSE](./LICENSE) and [NOTICE](./NOTICE) for more information.
