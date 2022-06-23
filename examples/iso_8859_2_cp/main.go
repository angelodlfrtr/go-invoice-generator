package main

import (
	"io/ioutil"
	"os"

	generator "github.com/angelodlfrtr/go-invoice-generator"
	"github.com/go-pdf/fpdf"
)

func main() {
	doc, _ := generator.New(generator.Invoice, &generator.Options{
		CurrencySymbol: "Ţ",
		BoldFont:       "Roboto",
		Font:           "Roboto",
	})

	// Set up translator
	func() {
		trReader, _ := os.Open("./iso-8859-2.map")
		unicodeTranslator, _ := fpdf.UnicodeTranslator(trReader)
		doc.SetUnicodeTranslator(unicodeTranslator)
	}()

	func() {
		pdf := doc.Pdf()

		// Bold
		jsonBytes, _ := ioutil.ReadFile("./Roboto-Bold.json")
		zBytes, _ := ioutil.ReadFile("./Roboto-Bold.z")
		pdf.AddFontFromBytes("Roboto", "B", jsonBytes, zBytes)

		// Regular
		jsonBytes, _ = ioutil.ReadFile("././Roboto-Regular.json")
		zBytes, _ = ioutil.ReadFile("./Roboto-Regular.z")
		pdf.AddFontFromBytes("Roboto", "", jsonBytes, zBytes)
	}()

	// Set header
	doc.SetHeader(&generator.HeaderFooter{
		Text:       "<center>şţăîâ ŞŢĂÎÂ</center>",
		Pagination: true,
	})

	doc.SetRef("one")
	doc.SetDescription("Hello, world şţăîâ ŞŢĂÎÂ")

	doc.SetCompany(&generator.Contact{
		Name: "şţăîâ",
		Address: &generator.Address{
			Address:    "89 Rue de Brest",
			Address2:   "Appartement 2",
			PostalCode: "75000",
			City:       "Paris",
			Country:    "France",
		},
	})

	doc.SetCustomer(&generator.Contact{
		Name: "şţăîâ",
		Address: &generator.Address{
			Address:    "89 Rue de Paris",
			PostalCode: "29200",
			City:       "Brest",
			Country:    "France",
		},
	})

	pdf, err := doc.Build()
	if err != nil {
		panic(err)
	}

	if err := pdf.OutputFileAndClose("out.pdf"); err != nil {
		panic(err)
	}
}
