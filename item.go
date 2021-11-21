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

func (i *Item) unitCost() decimal.Decimal {
	unitCost, _ := decimal.NewFromString(i.UnitCost)
	return unitCost
}

func (i *Item) quantity() decimal.Decimal {
	quantity, _ := decimal.NewFromString(i.Quantity)
	return quantity
}

func (i *Item) totalWithoutTax() decimal.Decimal {
	quantity, _ := decimal.NewFromString(i.Quantity)
	price, _ := decimal.NewFromString(i.UnitCost)
	total := price.Mul(quantity)

	return total
}

func (i *Item) totalWithoutTaxAndWithDiscount() decimal.Decimal {
	total := i.totalWithoutTax()

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

func (i *Item) totalWithTaxAndDiscount() decimal.Decimal {
	return i.totalWithoutTaxAndWithDiscount().Add(i.taxWithDiscount())
}

func (i *Item) taxWithDiscount() decimal.Decimal {
	result := decimal.NewFromFloat(0)

	if i.Tax == nil {
		return result
	}

	totalHT := i.totalWithoutTaxAndWithDiscount()
	taxType, taxAmount := i.Tax.getTax()

	if taxType == "amount" {
		result = taxAmount
	} else {
		divider := decimal.NewFromFloat(100)
		result = totalHT.Mul(taxAmount.Div(divider))
	}

	return result
}

func (i *Item) appendColTo(options *Options, pdf *gofpdf.Fpdf) {
	ac := accounting.Accounting{
		Symbol:    options.CurrencySymbol,
		Precision: options.CurrencyPrecision,
		Thousand:  options.CurrencyThousand,
		Decimal:   options.CurrencyDecimal,
	}

	// Get base Y (top of line)
	baseY := pdf.GetY()

	// Name
	pdf.SetX(ItemColNameOffset)
	pdf.MultiCell(
		ItemColUnitPriceOffset-ItemColNameOffset,
		3,
		i.Name,
		"",
		"",
		false,
	)

	// Description
	if len(i.Description) > 0 {
		pdf.SetX(ItemColNameOffset)
		pdf.SetY(pdf.GetY() + 1)

		pdf.SetFont("dejavu", "", SmallTextFontSize)
		pdf.SetTextColor(GreyTextColor[0], GreyTextColor[1], GreyTextColor[2])

		pdf.MultiCell(
			ItemColUnitPriceOffset-ItemColNameOffset,
			3,
			i.Description,
			"",
			"",
			false,
		)

		// Reset font
		pdf.SetFont("dejavu", "", BaseTextFontSize)
		pdf.SetTextColor(BaseTextColor[0], BaseTextColor[1], BaseTextColor[2])
	}

	// Compute line height
	colHeight := pdf.GetY() - baseY

	// Unit price
	pdf.SetY(baseY)
	pdf.SetX(ItemColUnitPriceOffset)
	pdf.CellFormat(
		ItemColQuantityOffset-ItemColUnitPriceOffset,
		colHeight,
		ac.FormatMoneyDecimal(i.unitCost()),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Quantity
	pdf.SetX(ItemColQuantityOffset)
	pdf.CellFormat(
		ItemColTaxOffset-ItemColQuantityOffset,
		colHeight,
		i.quantity().String(),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Total HT
	pdf.SetX(ItemColTotalHTOffset)
	pdf.CellFormat(
		ItemColTaxOffset-ItemColTotalHTOffset,
		colHeight,
		ac.FormatMoneyDecimal(i.totalWithoutTax()),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Discount
	pdf.SetX(ItemColDiscountOffset)
	if i.Discount == nil {
		pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight,
			"--",
			"0",
			0,
			"",
			false,
			0,
			"",
		)
	} else {
		// If discount
		discountType, discountAmount := i.Discount.getDiscount()
		var discountTitle string
		var discountDesc string

		if discountType == "percent" {
			discountTitle = fmt.Sprintf("%s %s", discountAmount, "%")
			// get amount from percent
			dCost := i.totalWithoutTax()
			dAmount := dCost.Mul(discountAmount.Div(decimal.NewFromFloat(100)))
			discountDesc = fmt.Sprintf("-%s", ac.FormatMoneyDecimal(dAmount))
		} else {
			discountTitle = fmt.Sprintf("%s %s", discountAmount, "€")
			dCost := i.totalWithoutTax()
			dPerc := discountAmount.Mul(decimal.NewFromFloat(100))
			dPerc = dPerc.Div(dCost)
			// get percent from amount
			discountDesc = fmt.Sprintf("-%s %%", dPerc.StringFixed(2))
		}

		// discount title
		// lastY := pdf.GetY()
		pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			discountTitle,
			"0",
			0,
			"LB",
			false,
			0,
			"",
		)

		// discount desc
		pdf.SetXY(ItemColDiscountOffset, baseY+(colHeight/2))
		pdf.SetFont("dejavu", "", SmallTextFontSize)
		pdf.SetTextColor(GreyTextColor[0], GreyTextColor[1], GreyTextColor[2])

		pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			discountDesc,
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		pdf.SetFont("dejavu", "", BaseTextFontSize)
		pdf.SetTextColor(BaseTextColor[0], BaseTextColor[1], BaseTextColor[2])
		pdf.SetY(baseY)
	}

	// Tax
	pdf.SetX(ItemColTaxOffset)
	if i.Tax == nil {
		// If no tax
		pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight,
			"--",
			"0",
			0,
			"",
			false,
			0,
			"",
		)
	} else {
		// If tax
		taxType, taxAmount := i.Tax.getTax()
		var taxTitle string
		var taxDesc string

		if taxType == "percent" {
			taxTitle = fmt.Sprintf("%s %s", taxAmount, "%")
			// get amount from percent
			dCost := i.totalWithoutTaxAndWithDiscount()
			dAmount := dCost.Mul(taxAmount.Div(decimal.NewFromFloat(100)))
			taxDesc = ac.FormatMoneyDecimal(dAmount)
		} else {
			taxTitle = fmt.Sprintf("%s %s", taxAmount, "€")
			dCost := i.totalWithoutTaxAndWithDiscount()
			dPerc := taxAmount.Mul(decimal.NewFromFloat(100))
			dPerc = dPerc.Div(dCost)
			// get percent from amount
			taxDesc = fmt.Sprintf("%s %%", dPerc.StringFixed(2))
		}

		// tax title
		// lastY := pdf.GetY()
		pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			taxTitle,
			"0",
			0,
			"LB",
			false,
			0,
			"",
		)

		// tax desc
		pdf.SetXY(ItemColTaxOffset, baseY+(colHeight/2))
		pdf.SetFont("dejavu", "", SmallTextFontSize)
		pdf.SetTextColor(GreyTextColor[0], GreyTextColor[1], GreyTextColor[2])

		pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			taxDesc,
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		pdf.SetFont("dejavu", "", BaseTextFontSize)
		pdf.SetTextColor(BaseTextColor[0], BaseTextColor[1], BaseTextColor[2])
		pdf.SetY(baseY)
	}

	// TOTAL TTC
	pdf.SetX(ItemColTotalTTCOffset)
	pdf.CellFormat(
		190-ItemColTotalTTCOffset,
		colHeight,
		ac.FormatMoneyDecimal(i.totalWithTaxAndDiscount()),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Set Y for next line
	pdf.SetY(baseY + colHeight)
}
