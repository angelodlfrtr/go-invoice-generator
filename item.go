package generator

import (
	"fmt"

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

func (i *Item) appendColTo(options *Options, doc *Document) {
	ac := accounting.Accounting{
		Symbol:    options.CurrencySymbol,
		Precision: options.CurrencyPrecision,
		Thousand:  options.CurrencyThousand,
		Decimal:   options.CurrencyDecimal,
	}

	// Get base Y (top of line)
	baseY := doc.pdf.GetY()

	// Name
	doc.pdf.SetX(ItemColNameOffset)
	doc.pdf.MultiCell(
		ItemColUnitPriceOffset-ItemColNameOffset,
		3,
		doc.encodeString(i.Name),
		"",
		"",
		false,
	)

	// Description
	if len(i.Description) > 0 {
		doc.pdf.SetX(ItemColNameOffset)
		doc.pdf.SetY(doc.pdf.GetY() + 1)

		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.MultiCell(
			ItemColUnitPriceOffset-ItemColNameOffset,
			3,
			doc.encodeString(i.Description),
			"",
			"",
			false,
		)

		// Reset font
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
	}

	// Compute line height
	colHeight := doc.pdf.GetY() - baseY

	// Unit price
	doc.pdf.SetY(baseY)
	doc.pdf.SetX(ItemColUnitPriceOffset)
	doc.pdf.CellFormat(
		ItemColQuantityOffset-ItemColUnitPriceOffset,
		colHeight,
		doc.encodeString(ac.FormatMoneyDecimal(i.unitCost())),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Quantity
	doc.pdf.SetX(ItemColQuantityOffset)
	doc.pdf.CellFormat(
		ItemColTaxOffset-ItemColQuantityOffset,
		colHeight,
		doc.encodeString(i.quantity().String()),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Total HT
	doc.pdf.SetX(ItemColTotalHTOffset)
	doc.pdf.CellFormat(
		ItemColTaxOffset-ItemColTotalHTOffset,
		colHeight,
		doc.encodeString(ac.FormatMoneyDecimal(i.totalWithoutTax())),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Discount
	doc.pdf.SetX(ItemColDiscountOffset)
	if i.Discount == nil {
		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight,
			doc.encodeString("--"),
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
			discountTitle = fmt.Sprintf("%s %s", discountAmount, doc.encodeString("%"))
			// get amount from percent
			dCost := i.totalWithoutTax()
			dAmount := dCost.Mul(discountAmount.Div(decimal.NewFromFloat(100)))
			discountDesc = fmt.Sprintf("-%s", ac.FormatMoneyDecimal(dAmount))
		} else {
			discountTitle = fmt.Sprintf("%s %s", discountAmount, doc.encodeString("€"))
			dCost := i.totalWithoutTax()
			dPerc := discountAmount.Mul(decimal.NewFromFloat(100))
			dPerc = dPerc.Div(dCost)
			// get percent from amount
			discountDesc = fmt.Sprintf("-%s %%", dPerc.StringFixed(2))
		}

		// discount title
		// lastY := doc.pdf.GetY()
		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			doc.encodeString(discountTitle),
			"0",
			0,
			"LB",
			false,
			0,
			"",
		)

		// discount desc
		doc.pdf.SetXY(ItemColDiscountOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			doc.encodeString(discountDesc),
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
		doc.pdf.SetY(baseY)
	}

	// Tax
	doc.pdf.SetX(ItemColTaxOffset)
	if i.Tax == nil {
		// If no tax
		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight,
			doc.encodeString("--"),
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
		// lastY := doc.pdf.GetY()
		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			doc.encodeString(taxTitle),
			"0",
			0,
			"LB",
			false,
			0,
			"",
		)

		// tax desc
		doc.pdf.SetXY(ItemColTaxOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			doc.encodeString(taxDesc),
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
		doc.pdf.SetY(baseY)
	}

	// TOTAL TTC
	doc.pdf.SetX(ItemColTotalTTCOffset)
	doc.pdf.CellFormat(
		190-ItemColTotalTTCOffset,
		colHeight,
		doc.encodeString(ac.FormatMoneyDecimal(i.totalWithTaxAndDiscount())),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Set Y for next line
	doc.pdf.SetY(baseY + colHeight)
}
