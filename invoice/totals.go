package invoice

import (
	"sort"

	"github.com/shopspring/decimal"
)

// The totals below are views over Breakdown, so every figure printed on the
// PDF shares one rounding model with e-invoice exports.

// TotalWithoutTaxAndWithoutDocumentDiscount returns the sum of the rounded item
// net amounts, before the document discount.
func (doc *Document) TotalWithoutTaxAndWithoutDocumentDiscount() decimal.Decimal {
	return doc.Breakdown().LineTotal
}

// DocumentDiscount returns the rounded document-level discount amount.
func (doc *Document) DocumentDiscount() decimal.Decimal {
	return doc.Breakdown().Allowance
}

// TotalWithoutTax returns the taxable total: item net amounts minus the
// document discount.
func (doc *Document) TotalWithoutTax() decimal.Decimal {
	return doc.Breakdown().TaxBasis
}

// TotalWithTax returns the amount due: TotalWithoutTax plus Tax.
func (doc *Document) TotalWithTax() decimal.Decimal {
	return doc.Breakdown().GrandTotal
}

// Tax returns the total tax, summed over the rounded per-rate group taxes.
func (doc *Document) Tax() decimal.Decimal {
	return doc.Breakdown().Tax
}

// TaxLine holds the aggregated tax amount for one named group.
type TaxLine struct {
	Name   string
	Amount decimal.Decimal
}

// TaxLines returns per-name tax lines when at least one item tax has a Name.
// Returns nil when no item tax has a name (caller should use Tax() instead).
// Named taxes are sorted alphabetically; unnamed taxes come last. The amounts
// always add up to Tax().
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

	prec := doc.precision()
	amounts := map[string]decimal.Decimal{}
	for _, g := range doc.Breakdown().Groups {
		for name, amt := range g.taxByName(prec) {
			amounts[name] = amounts[name].Add(amt)
		}
	}

	var names []string
	hasUnnamed := false
	for name := range amounts {
		if name == "" {
			hasUnnamed = true
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if hasUnnamed {
		names = append(names, "")
	}

	lines := make([]TaxLine, 0, len(names))
	for _, name := range names {
		lines = append(lines, TaxLine{Name: name, Amount: amounts[name]})
	}
	return lines
}
