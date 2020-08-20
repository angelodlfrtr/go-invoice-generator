package generator

import (
	"fmt"

	"github.com/jung-kurt/gofpdf"
	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"
)

// Item represent a 'product' or a 'service'
type Item struct {
	Name        string `validate:"required"`
	Description string
	UnitCost    string
	Quantity    string
	Tax         *Tax
	Discount    map[string]struct {
		Percent string // Discount in percent OR
		Amount  string // Discount in amount
	}
}

func (i *Item) appendColTo(options *Options, pdf *gofpdf.Fpdf) {
	ac := accounting.Accounting{
		Symbol:    encodeString(options.CurrencySymbol),
		Precision: options.CurrencyPrecision,
		Thousand:  options.CurrencyThousand,
		Decimal:   options.CurrencyDecimal,
	}

	// Name
	pdf.CellFormat(80, 6, encodeString(i.Name), "0", 0, "", false, 0, "")

	// Unit price
	pdf.SetX(90)
	pdf.CellFormat(30, 6, ac.FormatMoneyDecimal(i.unitCost()), "0", 0, "", false, 0, "")

	// Quantity
	pdf.SetX(120)
	pdf.CellFormat(15, 6, i.quantity().String(), "0", 0, "", false, 0, "")

	// Total HT
	pdf.SetX(135)
	pdf.CellFormat(20, 6, ac.FormatMoneyDecimal(i.totalHT()), "0", 0, "", false, 0, "")

	// Tax
	taxType, taxAmount := i.Tax.getTax()
	var taxString string
	if taxType == "percent" {
		taxString = fmt.Sprintf("%s %s", taxAmount, encodeString("%"))
	} else {
		taxString = fmt.Sprintf("%s %s", taxAmount, encodeString("€"))
	}

	pdf.SetX(155)
	pdf.CellFormat(20, 6, taxString, "0", 0, "", false, 0, "")

	// TOTAL TTC
	pdf.SetX(175)
	pdf.CellFormat(25, 6, ac.FormatMoneyDecimal(i.totalTTC()), "0", 0, "", false, 0, "")
}

func (i *Item) unitCost() decimal.Decimal {
	unitCost, _ := decimal.NewFromString(i.UnitCost)
	return unitCost
}

func (i *Item) quantity() decimal.Decimal {
	quantity, _ := decimal.NewFromString(i.Quantity)
	return quantity
}

func (i *Item) totalHT() decimal.Decimal {
	quantity, _ := decimal.NewFromString(i.Quantity)
	price, _ := decimal.NewFromString(i.UnitCost)

	return price.Mul(quantity)
}

func (i *Item) totalTTC() decimal.Decimal {
	totalHT := i.totalHT()
	taxType, taxAmount := i.Tax.getTax()
	var totalTTC decimal.Decimal

	if taxType == "amount" {
		totalTTC = totalHT.Add(taxAmount)
	} else {
		divider, _ := decimal.NewFromString("100")
		tax := totalHT.Mul(taxAmount.Div(divider))
		totalTTC = totalHT.Add(tax)
	}

	return totalTTC
}
