package generator

import (
	"sort"

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
			if totalWithoutTaxAndWithoutDocDiscount.IsZero() {
				return decimal.NewFromFloat(0)
			}
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

// TaxLine holds the aggregated tax amount for one named group.
type TaxLine struct {
	Name   string
	Amount decimal.Decimal
}

// TaxLines returns per-name tax lines when at least one item tax has a Name.
// Returns nil when no item tax has a name (caller should use Tax() instead).
// Named taxes are sorted alphabetically; unnamed taxes come last.
func (doc *Document) TaxLines() []TaxLine {
	hasName := false
	for _, item := range doc.Items {
		if item.Tax != nil && item.Tax.Name != "" {
			hasName = true
			break
		}
	}
	if !hasName {
		return nil
	}

	seen := map[string]bool{}
	var names []string
	hasUnnamed := false
	for _, item := range doc.Items {
		if item.Tax == nil {
			continue
		}
		n := item.Tax.Name
		if n == "" {
			hasUnnamed = true
			continue
		}
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	sort.Strings(names)
	if hasUnnamed {
		names = append(names, "")
	}

	totalNoDocDiscount := doc.TotalWithoutTaxAndWithoutDocumentDiscount()
	var lines []TaxLine
	for _, name := range names {
		lines = append(lines, TaxLine{Name: name, Amount: doc.taxForGroup(name, totalNoDocDiscount)})
	}
	return lines
}

// taxForGroup computes the aggregate tax for items whose Tax.Name == name,
// applying any document-level discount identically to Tax().
func (doc *Document) taxForGroup(name string, totalWithoutDocDiscount decimal.Decimal) decimal.Decimal {
	result := decimal.NewFromFloat(0)

	if doc.Discount == nil {
		for _, item := range doc.Items {
			if item.Tax != nil && item.Tax.Name == name {
				result = result.Add(item.TaxWithTotalDiscounted())
			}
		}
		return result
	}

	discountType, discountAmount := doc.Discount.getDiscount()
	discountPercent := discountAmount
	if discountType == DiscountTypeAmount {
		if totalWithoutDocDiscount.IsZero() {
			return decimal.NewFromFloat(0)
		}
		discountPercent = discountAmount.Mul(decimal.NewFromFloat(100)).Div(totalWithoutDocDiscount)
	}

	for _, item := range doc.Items {
		if item.Tax == nil || item.Tax.Name != name {
			continue
		}
		taxType, taxAmt := item.Tax.getTax()
		if taxType == TaxTypeAmount {
			result = result.Add(taxAmt)
		} else {
			itemTotal := item.TotalWithoutTaxAndWithDiscount()
			toSub := discountPercent.Mul(itemTotal).Div(decimal.NewFromFloat(100))
			result = result.Add(taxAmt.Mul(itemTotal.Sub(toSub)).Div(decimal.NewFromFloat(100)))
		}
	}
	return result
}
