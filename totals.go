package generator

import (
	"github.com/shopspring/decimal"
)

// TotalWithoutTaxAndWithoutDocumentDiscount return total without tax and without document discount
func (doc *Document) TotalWithoutTaxAndWithoutDocumentDiscount() decimal.Decimal {
	total := decimal.NewFromInt(0)

	for _, item := range doc.Items {
		total = total.Add(item.TotalWithoutTaxAndWithDiscount())
	}

	return total
}

// TotalWithoutTax return total without tax and with document discount
func (doc *Document) TotalWithoutTax() decimal.Decimal {
	total := doc.TotalWithoutTaxAndWithoutDocumentDiscount()

	// Apply document discount
	if doc.Discount != nil {
		discountType, discountNumber := doc.Discount.getDiscount()

		if discountType == DiscountTypeAmount {
			total = total.Sub(discountNumber)
		} else {
			// Percent
			toSub := total.Mul(discountNumber.Div(decimal.NewFromFloat(100)))
			total = total.Sub(toSub)
		}
	}

	return total
}

// TotalWithTax return total with tax and with document discount
func (doc *Document) TotalWithTax() decimal.Decimal {
	totalWithoutTax := doc.TotalWithoutTax()
	tax := doc.Tax()

	return totalWithoutTax.Add(tax)
}

// Tax return the total tax with document discount
func (doc *Document) Tax() decimal.Decimal {
	totalWithoutTaxAndWithoutDocDiscount := doc.TotalWithoutTaxAndWithoutDocumentDiscount()
	totalTax := decimal.NewFromFloat(0)

	if doc.Discount == nil {
		for _, item := range doc.Items {
			totalTax = totalTax.Add(item.TaxWithTotalDiscounted())
		}
	} else {
		discountType, discountAmount := doc.Discount.getDiscount()
		discountPercent := discountAmount
		if discountType == DiscountTypeAmount {
			// Get percent from total discounted
			discountPercent = discountAmount.Mul(decimal.NewFromFloat(100)).Div(totalWithoutTaxAndWithoutDocDiscount)
		}

		for _, item := range doc.Items {
			if item.Tax != nil {
				taxType, taxAmount := item.Tax.getTax()
				if taxType == TaxTypeAmount {
					// If tax type is amount, just add amount to tax
					totalTax = totalTax.Add(taxAmount)
				} else {
					// Else, remove doc discount % from item total without tax and item discount
					itemTotal := item.TotalWithoutTaxAndWithDiscount()
					toSub := discountPercent.Mul(itemTotal).Div(decimal.NewFromFloat(100))
					itemTotalDiscounted := itemTotal.Sub(toSub)

					// Then recompute tax on itemTotalDiscounted
					itemTaxDiscounted := taxAmount.Mul(itemTotalDiscounted).Div(decimal.NewFromFloat(100))

					totalTax = totalTax.Add(itemTaxDiscounted)
				}
			}
		}
	}

	return totalTax
}
