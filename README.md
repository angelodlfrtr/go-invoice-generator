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
  generator "github.com/angelodlfrtr/go-invoice-generator"
)

func main() {
	doc := generator.New(generator.INVOICE, &generator.Options{})

	doc.SetHeader(&generator.HeaderFooter{
		Text: "Some header text",
	})

	doc.SetFooter(&generator.HeaderFooter{
		Text:           "<center>Some footer text</center>",
		Pagination:     true,
	})

	doc.SetNumber("testnumber")
	doc.SetVersion("test version")

	doc.SetDescription("Some text describing document")

	doc.SetCompany(&generator.Contact{
		Name: "Test Company",
		Logo: []byte{}, // Image as byte array, supported format: png, jpeg, gif
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

	doc.AppendItem(&generator.Item{
		Name:     "Item one",
		UnitCost: "89",
		Quantity: "2",
		Tax: &generator.Tax{
			Percent: "20",
		},
	})

	doc.AppendItem(&generator.Item{
		Name:     "Item two",
		UnitCost: "5.89",
		Quantity: "11",
		Tax: &generator.Tax{
			Amount: "10",
		},
	})

	pdf, err := doc.Build()

	if err != nil {
		fmt.Println(err.Error())
	}

	err = pdf.OutputFileAndClose("./out.pdf")

	if err != nil {
		fmt.Println(err.Error())
	}
}

```

## License

This SDK is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0),
see [LICENSE](./LICENSE) and [NOTICE](./NOTICE) for more information.
