package invoice

import (
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// TaxGroup aggregates the items that share a tax category and rate. It is the
// unit on which taxes are rounded, following the EN 16931 model: line amounts
// are rounded first, the document discount is split across groups, and the tax
// of each group is computed on its rounded basis.
type TaxGroup struct {
	// Category is the upper-cased Tax.Category shared by the group's items.
	// Empty when the items carry no category or no tax at all.
	Category string
	// Percent is the tax rate. Zero for Fixed and Untaxed groups.
	Percent decimal.Decimal
	// Fixed marks a group of fixed-amount taxes; Tax is then the sum of the
	// amounts rather than a rate applied to the basis.
	Fixed bool
	// Untaxed marks the items that have no Tax.
	Untaxed bool
	// ExemptionReason and ExemptionReasonCode are taken from the first item
	// of the group that sets them.
	ExemptionReason     string
	ExemptionReasonCode string

	// LineTotal is the sum of the rounded net amounts of the group's items.
	LineTotal decimal.Decimal
	// Allowance is the group's rounded share of the document discount.
	Allowance decimal.Decimal
	// Tax is the group's rounded tax amount.
	Tax decimal.Decimal

	names     []string
	nameBasis map[string]decimal.Decimal // rate groups: basis per Tax.Name
	nameTax   map[string]decimal.Decimal // fixed groups: tax per Tax.Name
}

// Basis returns the taxable amount of the group: LineTotal minus Allowance.
func (g *TaxGroup) Basis() decimal.Decimal {
	return g.LineTotal.Sub(g.Allowance)
}

// Breakdown holds every monetary total of a document, computed once with
// consistent rounding so that:
//
//	LineTotal  = sum of Lines
//	Allowance  = sum of group allowances
//	TaxBasis   = LineTotal - Allowance = sum of group bases
//	Tax        = sum of group taxes
//	GrandTotal = TaxBasis + Tax
//
// These are the identities required by EN 16931 (BR-CO-10, 11, 13, 14, 15),
// so an e-invoice derived from a Breakdown always matches the PDF.
type Breakdown struct {
	// Lines holds the rounded net amount of each item, in document order.
	Lines []decimal.Decimal
	// Groups are sorted by category then rate; fixed-amount taxes come last.
	Groups []*TaxGroup

	LineTotal  decimal.Decimal
	Allowance  decimal.Decimal
	TaxBasis   decimal.Decimal
	Tax        decimal.Decimal
	GrandTotal decimal.Decimal
}

var hundred = decimal.NewFromInt(100)

// precision returns the currency precision used for rounding, clamped to
// [0, 8] so the int32 conversion is safe.
func (doc *Document) precision() int32 {
	p := doc.Currency.Precision
	if p < 0 {
		p = 0
	}
	if p > 8 {
		p = 8
	}
	return int32(p)
}

// Breakdown computes the document totals. Items must have been prepared by
// Validate (called by Build) first.
//
// Amounts are rounded to Currency.Precision at three points: each
// item's net amount, each group's share of the document discount and each
// group's tax. Totals are sums of those rounded parts, never rounded sums.
func (doc *Document) Breakdown() *Breakdown {
	prec := doc.precision()
	bd := &Breakdown{Lines: make([]decimal.Decimal, len(doc.Items))}
	byKey := map[string]*TaxGroup{}

	for i, item := range doc.Items {
		line := item.TotalWithoutTaxAndWithDiscount().Round(prec)
		bd.Lines[i] = line
		bd.LineTotal = bd.LineTotal.Add(line)

		var (
			key  string
			name string
			g    *TaxGroup
		)
		switch {
		case item.Tax == nil:
			key = "untaxed"
		case len(item.Tax.Amount) > 0:
			key = "fixed"
			name = item.Tax.Name
		default:
			category := strings.ToUpper(strings.TrimSpace(item.Tax.Category))
			key = "rate|" + category + "|" + item.Tax._percent.String()
			name = item.Tax.Name
		}

		g, ok := byKey[key]
		if !ok {
			g = &TaxGroup{
				nameBasis: map[string]decimal.Decimal{},
				nameTax:   map[string]decimal.Decimal{},
			}
			switch {
			case item.Tax == nil:
				g.Untaxed = true
			case len(item.Tax.Amount) > 0:
				g.Fixed = true
			default:
				g.Category = strings.ToUpper(strings.TrimSpace(item.Tax.Category))
				g.Percent = item.Tax._percent
			}
			byKey[key] = g
			bd.Groups = append(bd.Groups, g)
		}

		g.LineTotal = g.LineTotal.Add(line)
		if item.Tax != nil {
			if g.ExemptionReason == "" {
				g.ExemptionReason = item.Tax.ExemptionReason
			}
			if g.ExemptionReasonCode == "" {
				g.ExemptionReasonCode = item.Tax.ExemptionReasonCode
			}
			if _, seen := g.nameBasis[name]; !seen {
				g.names = append(g.names, name)
			}
			g.nameBasis[name] = g.nameBasis[name].Add(line)
			if g.Fixed {
				g.nameTax[name] = g.nameTax[name].Add(item.Tax._amount.Round(prec))
			}
		}
	}

	sort.SliceStable(bd.Groups, func(i, j int) bool {
		a, b := bd.Groups[i], bd.Groups[j]
		if a.Fixed != b.Fixed {
			return !a.Fixed
		}
		if a.Untaxed != b.Untaxed {
			return b.Untaxed
		}
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		return a.Percent.Cmp(b.Percent) < 0
	})

	// Document discount, rounded, then split across groups in proportion to
	// their line totals with the rounding residual on the largest group.
	if doc.Discount != nil && len(bd.Groups) > 0 && bd.LineTotal.Sign() != 0 {
		dType, dValue := doc.Discount.getDiscount()
		var discount decimal.Decimal
		if dType == DiscountTypeAmount {
			discount = dValue.Round(prec)
		} else {
			discount = bd.LineTotal.Mul(dValue).Div(hundred).Round(prec)
		}
		if discount.Sign() != 0 {
			var allocated decimal.Decimal
			largest := bd.Groups[0]
			for _, g := range bd.Groups {
				g.Allowance = g.LineTotal.Mul(discount).Div(bd.LineTotal).Round(prec)
				allocated = allocated.Add(g.Allowance)
				if g.LineTotal.Cmp(largest.LineTotal) > 0 {
					largest = g
				}
			}
			largest.Allowance = largest.Allowance.Add(discount.Sub(allocated))
			bd.Allowance = discount
		}
	}

	for _, g := range bd.Groups {
		switch {
		case g.Fixed:
			for _, amt := range g.nameTax {
				g.Tax = g.Tax.Add(amt)
			}
		case g.Untaxed:
			// no tax
		default:
			g.Tax = g.Basis().Mul(g.Percent).Div(hundred).Round(prec)
		}
		bd.TaxBasis = bd.TaxBasis.Add(g.Basis())
		bd.Tax = bd.Tax.Add(g.Tax)
	}
	bd.GrandTotal = bd.TaxBasis.Add(bd.Tax)

	return bd
}

// taxByName splits the group tax per Tax.Name. Rate groups are split in
// proportion to each name's basis with the residual on the largest share, so
// the parts always add up to the group's Tax. Fixed groups report the amounts
// as given.
func (g *TaxGroup) taxByName(prec int32) map[string]decimal.Decimal {
	out := map[string]decimal.Decimal{}
	if g.Fixed {
		for n, amt := range g.nameTax {
			out[n] = amt
		}
		return out
	}
	if g.Untaxed || len(g.names) == 0 || g.LineTotal.Sign() == 0 {
		return out
	}
	if len(g.names) == 1 {
		out[g.names[0]] = g.Tax
		return out
	}

	var allocated decimal.Decimal
	largest := g.names[0]
	for _, n := range g.names {
		share := g.Tax.Mul(g.nameBasis[n]).Div(g.LineTotal).Round(prec)
		out[n] = share
		allocated = allocated.Add(share)
		if g.nameBasis[n].Cmp(g.nameBasis[largest]) > 0 {
			largest = n
		}
	}
	out[largest] = out[largest].Add(g.Tax.Sub(allocated))
	return out
}
