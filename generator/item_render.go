package generator

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// appendColTo renders the item as a row in the PDF items table.
// It must be called with the target document — page break handling is the
// caller's responsibility (see appendItems which wraps this in pageTxn).
func (i *Item) appendColTo(doc *Document) {
	baseY := doc.pdf.GetY()

	// Name
	doc.pdf.SetX(ItemColNameOffset)
	doc.pdf.MultiCell(ItemColUnitPriceOffset-ItemColNameOffset, 3, doc.encodeString(i.Name), "", "", false)

	// Description
	if len(i.Description) > 0 {
		doc.pdf.SetX(ItemColNameOffset)
		doc.pdf.SetY(doc.pdf.GetY() + 1)
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(doc.Options.GreyTextColor[0], doc.Options.GreyTextColor[1], doc.Options.GreyTextColor[2])
		doc.pdf.MultiCell(ItemColUnitPriceOffset-ItemColNameOffset, 3, doc.encodeString(i.Description), "", "", false)
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(doc.Options.BaseTextColor[0], doc.Options.BaseTextColor[1], doc.Options.BaseTextColor[2])
	}

	colHeight := doc.pdf.GetY() - baseY

	// Unit price
	doc.pdf.SetY(baseY)
	doc.pdf.SetX(ItemColUnitPriceOffset)
	doc.pdf.CellFormat(ItemColQuantityOffset-ItemColUnitPriceOffset, colHeight, doc.encodeString(doc.ac.FormatMoneyDecimal(i._unitCost)), "0", 0, "", false, 0, "")

	// Quantity
	doc.pdf.SetX(ItemColQuantityOffset)
	doc.pdf.CellFormat(ItemColTaxOffset-ItemColQuantityOffset, colHeight, doc.encodeString(i._quantity.String()), "0", 0, "", false, 0, "")

	// Total HT (before discount)
	doc.pdf.SetX(ItemColTotalHTOffset)
	doc.pdf.CellFormat(ItemColTaxOffset-ItemColTotalHTOffset, colHeight, doc.encodeString(doc.ac.FormatMoneyDecimal(i.TotalWithoutTaxAndWithoutDiscount())), "0", 0, "", false, 0, "")

	// Discount
	doc.pdf.SetX(ItemColDiscountOffset)
	if i.Discount == nil {
		doc.pdf.CellFormat(ItemColTotalTTCOffset-ItemColDiscountOffset, colHeight, doc.encodeString("--"), "0", 0, "", false, 0, "")
	} else {
		discountType, discountAmount := i.Discount.getDiscount()
		dCost := i.TotalWithoutTaxAndWithoutDiscount()

		var discountTitle, discountDesc string
		if discountType == DiscountTypePercent {
			discountTitle = fmt.Sprintf("%s %s", discountAmount, doc.encodeString("%"))
			dAmount := dCost.Mul(discountAmount.Div(decimal.NewFromFloat(100)))
			discountDesc = fmt.Sprintf("-%s", doc.ac.FormatMoneyDecimal(dAmount))
		} else {
			discountTitle = fmt.Sprintf("%s %s", discountAmount, doc.encodeString("€"))
			dPerc := discountAmount.Mul(decimal.NewFromFloat(100)).Div(dCost)
			discountDesc = fmt.Sprintf("-%s %%", dPerc.StringFixed(2))
		}

		doc.pdf.CellFormat(ItemColTotalTTCOffset-ItemColDiscountOffset, colHeight/2, doc.encodeString(discountTitle), "0", 0, "LB", false, 0, "")

		doc.pdf.SetXY(ItemColDiscountOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(doc.Options.GreyTextColor[0], doc.Options.GreyTextColor[1], doc.Options.GreyTextColor[2])
		doc.pdf.CellFormat(ItemColTotalTTCOffset-ItemColDiscountOffset, colHeight/2, doc.encodeString(discountDesc), "0", 0, "LT", false, 0, "")

		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(doc.Options.BaseTextColor[0], doc.Options.BaseTextColor[1], doc.Options.BaseTextColor[2])
		doc.pdf.SetY(baseY)
	}

	// Tax
	doc.pdf.SetX(ItemColTaxOffset)
	if i.Tax == nil {
		doc.pdf.CellFormat(ItemColDiscountOffset-ItemColTaxOffset, colHeight, doc.encodeString("--"), "0", 0, "", false, 0, "")
	} else {
		taxType, taxAmount := i.Tax.getTax()
		dCost := i.TotalWithoutTaxAndWithDiscount()

		var taxTitle, taxDesc string
		if taxType == TaxTypePercent {
			taxTitle = fmt.Sprintf("%s %s", taxAmount, "%")
			dAmount := dCost.Mul(taxAmount.Div(decimal.NewFromFloat(100)))
			taxDesc = doc.ac.FormatMoneyDecimal(dAmount)
		} else {
			taxTitle = fmt.Sprintf("%s %s", doc.ac.Symbol, taxAmount)
			dPerc := taxAmount.Mul(decimal.NewFromFloat(100)).Div(dCost)
			taxDesc = fmt.Sprintf("%s %%", dPerc.StringFixed(2))
		}

		doc.pdf.CellFormat(ItemColDiscountOffset-ItemColTaxOffset, colHeight/2, doc.encodeString(taxTitle), "0", 0, "LB", false, 0, "")

		doc.pdf.SetXY(ItemColTaxOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(doc.Options.GreyTextColor[0], doc.Options.GreyTextColor[1], doc.Options.GreyTextColor[2])
		doc.pdf.CellFormat(ItemColDiscountOffset-ItemColTaxOffset, colHeight/2, doc.encodeString(taxDesc), "0", 0, "LT", false, 0, "")

		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(doc.Options.BaseTextColor[0], doc.Options.BaseTextColor[1], doc.Options.BaseTextColor[2])
		doc.pdf.SetY(baseY)
	}

	// Total TTC
	doc.pdf.SetX(ItemColTotalTTCOffset)
	doc.pdf.CellFormat(190-ItemColTotalTTCOffset, colHeight, doc.encodeString(doc.ac.FormatMoneyDecimal(i.TotalWithTaxAndDiscount())), "0", 0, "", false, 0, "")

	doc.pdf.SetY(baseY + colHeight)
}
