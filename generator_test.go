package generator

import (
	"testing"
)

func TestNew(t *testing.T) {
	doc, _ := New(Invoice, &Options{
		TextTypeInvoice: "Faktura numer 1/01/2021",
		TextRefTitle:    "Data wystawienia",
		TextDateTitle: "Data sprzedaży",
		TextVersionTitle: "Data zapłaty",
		AutoPrint:       true,
		CurrencySymbol: "zł ",
		TextItemsQuantityTitle: "ilość",
		TextItemsUnitCostTitle: "Cena jedn. netto",
		TextItemsTotalHTTitle: "Wartość netto",
		TextItemsTaxTitle: "Stawka VAT",
		TextItemsDiscountTitle: "Wartość VAT",
		TextItemsTotalTTCTitle: "Wartość brutto",
		TextItemsNameTitle: "Nazwa",
		DisplayDiscount: false,
		TextTotalTotal: "Suma netto",
		TextTotalTax: "Suma VAT",
		TextTotalWithTax: "Suma Brutto",
		TextPaymentTermTitle: "Termin płatności",

	})

	doc.SetRef("01/02/2021")
	doc.SetVersion("02/03/2021")

	doc.SetDate("02/03/2021")
	doc.SetPaymentTerm("02/04/2021")


	doc.SetCompany(&Contact{
		Name: "Test Company",
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

	doc.AppendItem(&Item{
		Name:     "Test",
		UnitCost: "10000",
		Quantity: "1",
	})


	doc.SetDefaultTax(&Tax{
		Percent: "23",
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
