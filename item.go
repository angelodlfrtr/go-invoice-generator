package generator

import (
	"fmt"

	"github.com/jung-kurt/gofpdf"
	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"
)

// Item represent a 'product' or a 'service'
type Item struct {
	Name        string    `json:"name,omitempty" validate:"required"`
	Description string    `json:"description,omitempty"`
	UnitCost    string    `json:"unit_cost,omitempty"`
	Quantity    string    `json:"quantity,omitempty"`
	Tax         *Tax      `json:"tax,omitempty"`
	Discount    *Discount `json:"discount,omitempty"`
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
	var taxString string
	if i.Tax != nil {
		taxType, taxAmount := i.Tax.getTax()
		if taxType == "percent" {
			taxString = fmt.Sprintf("%s %s", taxAmount, encodeString("%"))
		} else {
			taxString = fmt.Sprintf("%s %s", taxAmount, encodeString("€"))
		}
	} else {
		taxString = "--"
	}

	pdf.SetX(155)
	pdf.CellFormat(20, 6, taxString, "0", 0, "", false, 0, "")

	// TOTAL TTC
	pdf.SetX(175)
	pdf.CellFormat(25, 6, ac.FormatMoneyDecimal(i.totalTTC(nil)), "0", 0, "", false, 0, "")
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
	total := price.Mul(quantity)

	// Check discount
	if i.Discount != nil {
		dType, dNum := i.Discount.getDiscount()

		if dType == "amount" {
			total = total.Sub(dNum)
		} else {
			// Percent
			toSub := total.Mul(dNum.Div(decimal.NewFromFloat(100)))
			total = total.Sub(toSub)
		}
	}

	return total
}

func (i *Item) totalTTC(parentTax *Tax) decimal.Decimal {
	totalHT := i.totalHT()
	totalTTC := totalHT
	taxToUse := i.Tax

	if taxToUse == nil {
		taxToUse = parentTax
	}

	if taxToUse == nil {
		return totalTTC
	}

	taxType, taxAmount := taxToUse.getTax()
	if taxType == "amount" {
		totalTTC = totalHT.Add(taxAmount)
	} else {
		divider := decimal.NewFromFloat(100)
		tax := totalHT.Mul(taxAmount.Div(divider))
		totalTTC = totalHT.Add(tax)
	}

	return totalTTC
}
