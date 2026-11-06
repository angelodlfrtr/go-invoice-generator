package generator

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

var hundred = decimal.NewFromInt(100)

// appendItem draws one row of the items table at the current position. Page
// break handling is the caller's responsibility (see appendItems).
func (r *renderer) appendItem(i *invoice.Item) {
	baseY := r.pdf.GetY()

	// Name
	r.pdf.SetX(ItemColNameOffset)
	r.pdf.MultiCell(ItemColUnitPriceOffset-ItemColNameOffset, 3, r.encode(i.Name), "", "", false)

	// Description
	if len(i.Description) > 0 {
		r.pdf.SetX(ItemColNameOffset)
		r.pdf.SetY(r.pdf.GetY() + 1)
		r.pdf.SetFont(r.opts.Font, "", SmallTextFontSize)
		r.setTextColor(r.opts.GreyTextColor)
		r.pdf.MultiCell(ItemColUnitPriceOffset-ItemColNameOffset, 3, r.encode(i.Description), "", "", false)
		r.pdf.SetFont(r.opts.Font, "", BaseTextFontSize)
		r.setTextColor(r.opts.BaseTextColor)
	}

	colHeight := r.pdf.GetY() - baseY

	// Unit price
	r.pdf.SetY(baseY)
	r.pdf.SetX(ItemColUnitPriceOffset)
	r.pdf.CellFormat(ItemColQuantityOffset-ItemColUnitPriceOffset, colHeight, r.encode(r.money(i.UnitCostDecimal())), "0", 0, "", false, 0, "")

	// Quantity
	r.pdf.SetX(ItemColQuantityOffset)
	r.pdf.CellFormat(ItemColTaxOffset-ItemColQuantityOffset, colHeight, r.encode(i.QuantityDecimal().String()), "0", 0, "", false, 0, "")

	// Total before discount
	r.pdf.SetX(ItemColTotalHTOffset)
	r.pdf.CellFormat(ItemColTaxOffset-ItemColTotalHTOffset, colHeight, r.encode(r.money(i.TotalWithoutTaxAndWithoutDiscount())), "0", 0, "", false, 0, "")

	// Discount
	r.pdf.SetX(ItemColDiscountOffset)
	if i.Discount == nil {
		r.pdf.CellFormat(ItemColTotalTTCOffset-ItemColDiscountOffset, colHeight, r.encode("--"), "0", 0, "", false, 0, "")
	} else {
		discountType, discountAmount := i.Discount.Value()
		dCost := i.TotalWithoutTaxAndWithoutDiscount()

		var title, desc string
		if discountType == invoice.DiscountTypePercent {
			title = fmt.Sprintf("%s %s", discountAmount, r.encode("%"))
			desc = fmt.Sprintf("-%s", r.money(dCost.Mul(discountAmount.Div(hundred))))
		} else {
			title = fmt.Sprintf("%s %s", discountAmount, r.encode(r.ac.Symbol))
			if dCost.IsZero() {
				desc = "-0.00 %"
			} else {
				desc = fmt.Sprintf("-%s %%", discountAmount.Mul(hundred).Div(dCost).StringFixed(2))
			}
		}
		r.twoLineCell(ItemColDiscountOffset, ItemColTotalTTCOffset-ItemColDiscountOffset, baseY, colHeight, title, desc)
	}

	// Tax
	r.pdf.SetX(ItemColTaxOffset)
	if i.Tax == nil {
		r.pdf.CellFormat(ItemColDiscountOffset-ItemColTaxOffset, colHeight, r.encode("--"), "0", 0, "", false, 0, "")
	} else {
		taxType, taxAmount := i.Tax.Value()
		dCost := i.TotalWithoutTaxAndWithDiscount()

		var title, desc string
		if taxType == invoice.TaxTypePercent {
			title = fmt.Sprintf("%s %s", taxAmount, "%")
			desc = r.money(dCost.Mul(taxAmount.Div(hundred)))
		} else {
			title = fmt.Sprintf("%s %s", r.ac.Symbol, taxAmount)
			if dCost.IsZero() {
				desc = "0.00 %"
			} else {
				desc = fmt.Sprintf("%s %%", taxAmount.Mul(hundred).Div(dCost).StringFixed(2))
			}
		}
		r.twoLineCell(ItemColTaxOffset, ItemColDiscountOffset-ItemColTaxOffset, baseY, colHeight, title, desc)
	}

	// Total with tax
	r.pdf.SetX(ItemColTotalTTCOffset)
	r.pdf.CellFormat(190-ItemColTotalTTCOffset, colHeight, r.encode(r.money(i.TotalWithTaxAndDiscount())), "0", 0, "", false, 0, "")

	r.pdf.SetY(baseY + colHeight)
}

// twoLineCell draws title on the upper half and desc, in small grey text, on
// the lower half of a cell, then restores font and Y.
func (r *renderer) twoLineCell(x, w, baseY, height float64, title, desc string) {
	r.pdf.SetXY(x, baseY)
	r.pdf.CellFormat(w, height/2, r.encode(title), "0", 0, "LB", false, 0, "")

	r.pdf.SetXY(x, baseY+(height/2))
	r.pdf.SetFont(r.opts.Font, "", SmallTextFontSize)
	r.setTextColor(r.opts.GreyTextColor)
	r.pdf.CellFormat(w, height/2, r.encode(desc), "0", 0, "LT", false, 0, "")

	r.pdf.SetFont(r.opts.Font, "", BaseTextFontSize)
	r.setTextColor(r.opts.BaseTextColor)
	r.pdf.SetY(baseY)
}
