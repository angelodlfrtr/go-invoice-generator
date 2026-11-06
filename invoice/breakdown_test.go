package invoice

import (
	"testing"

	"github.com/shopspring/decimal"
)

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}

func assertDec(t *testing.T, what string, got decimal.Decimal, want string) {
	t.Helper()
	if !got.Equal(mustDec(t, want)) {
		t.Errorf("%s: got %s, want %s", what, got, want)
	}
}

func breakdownDoc(t *testing.T) *Document {
	t.Helper()
	doc, err := New(Invoice)
	if err != nil {
		t.Fatal(err)
	}
	doc.SetRef("BD-1").SetDate("01/01/2024")
	doc.SetCompany(&Contact{Name: "A", Address: &Address{Address: "x", Country: "FR"}})
	doc.SetCustomer(&Contact{Name: "B", Address: &Address{Address: "y", Country: "FR"}})
	return doc
}

// TestBreakdownRounding uses lines whose unrounded taxes sum differently from
// the sum of the rounded per-rate taxes, and a discount that does not split
// evenly, to pin the EN 16931 rounding model.
func TestBreakdownRounding(t *testing.T) {
	doc := breakdownDoc(t)
	doc.SetDiscount(&Discount{Percent: "5"})
	// 3 x 12.33 = 36.99 at 5.5%, 1 x 10.01 at 20%, 7 x 0.333 = 2.331 -> 2.33 at 20%
	doc.AppendItem(&Item{Name: "a", UnitCost: "12.33", Quantity: "3", Tax: &Tax{Percent: "5.5", Name: "VAT 5.5%"}})
	doc.AppendItem(&Item{Name: "b", UnitCost: "10.01", Quantity: "1", Tax: &Tax{Percent: "20", Name: "VAT 20%"}})
	doc.AppendItem(&Item{Name: "c", UnitCost: "0.333", Quantity: "7", Tax: &Tax{Percent: "20", Name: "VAT 20%"}})
	doc.AppendItem(&Item{Name: "d", UnitCost: "5", Quantity: "1"}) // untaxed
	doc.AppendItem(&Item{Name: "e", UnitCost: "9.99", Quantity: "1", Tax: &Tax{Amount: "1.234", Name: "Eco"}})
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}

	bd := doc.Breakdown()

	// Lines are rounded individually: 36.99 + 10.01 + 2.33 + 5.00 + 9.99 = 64.32
	assertDec(t, "line total", bd.LineTotal, "64.32")
	assertDec(t, "line c rounded", bd.Lines[2], "2.33")

	// Discount 5% of 64.32 = 3.216 -> 3.22, split per group with residual.
	assertDec(t, "allowance", bd.Allowance, "3.22")
	var allowances decimal.Decimal
	for _, g := range bd.Groups {
		allowances = allowances.Add(g.Allowance)
	}
	assertDec(t, "sum of group allowances", allowances, "3.22")

	// Groups sorted: rate groups (5.5, 20), then untaxed, then fixed.
	if len(bd.Groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(bd.Groups))
	}
	if !bd.Groups[0].Percent.Equal(mustDec(t, "5.5")) || !bd.Groups[1].Percent.Equal(mustDec(t, "20")) ||
		!bd.Groups[2].Untaxed || !bd.Groups[3].Fixed {
		t.Fatalf("unexpected group order: %+v", bd.Groups)
	}

	// Each group's tax is computed on its rounded basis and rounded.
	for _, g := range bd.Groups {
		if g.Fixed || g.Untaxed {
			continue
		}
		want := g.Basis().Mul(g.Percent).Div(hundred).Round(2)
		if !g.Tax.Equal(want) {
			t.Errorf("group %s tax %s, want %s", g.Percent, g.Tax, want)
		}
	}
	assertDec(t, "fixed tax rounded", bd.Groups[3].Tax, "1.23")

	// Identities.
	var bases, taxes decimal.Decimal
	for _, g := range bd.Groups {
		bases = bases.Add(g.Basis())
		taxes = taxes.Add(g.Tax)
	}
	if !bd.TaxBasis.Equal(bd.LineTotal.Sub(bd.Allowance)) || !bd.TaxBasis.Equal(bases) {
		t.Errorf("tax basis %s != line total - allowance (%s) or sum of bases (%s)", bd.TaxBasis, bd.LineTotal.Sub(bd.Allowance), bases)
	}
	if !bd.Tax.Equal(taxes) {
		t.Errorf("tax %s != sum of group taxes %s", bd.Tax, taxes)
	}
	if !bd.GrandTotal.Equal(bd.TaxBasis.Add(bd.Tax)) {
		t.Errorf("grand total %s != basis + tax", bd.GrandTotal)
	}

	// The document-level accessors are views over the breakdown.
	if !doc.TotalWithoutTaxAndWithoutDocumentDiscount().Equal(bd.LineTotal) ||
		!doc.DocumentDiscount().Equal(bd.Allowance) ||
		!doc.TotalWithoutTax().Equal(bd.TaxBasis) ||
		!doc.Tax().Equal(bd.Tax) ||
		!doc.TotalWithTax().Equal(bd.GrandTotal) {
		t.Error("document totals differ from the breakdown")
	}
}

func TestTaxLinesAddUpToTax(t *testing.T) {
	doc := breakdownDoc(t)
	doc.SetDiscount(&Discount{Amount: "7.77"})
	// Two names inside the same 20% group force a proportional split.
	doc.AppendItem(&Item{Name: "a", UnitCost: "33.33", Quantity: "1", Tax: &Tax{Percent: "20", Name: "VAT 20%"}})
	doc.AppendItem(&Item{Name: "b", UnitCost: "66.67", Quantity: "1", Tax: &Tax{Percent: "20", Name: "VAT 20% (services)"}})
	doc.AppendItem(&Item{Name: "c", UnitCost: "10", Quantity: "1", Tax: &Tax{Percent: "20"}}) // unnamed
	doc.AppendItem(&Item{Name: "d", UnitCost: "10", Quantity: "1", Tax: &Tax{Percent: "10", Name: "VAT 10%"}})
	doc.AppendItem(&Item{Name: "e", UnitCost: "10", Quantity: "1", Tax: &Tax{Amount: "2.5", Name: "Eco"}})
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}

	lines := doc.TaxLines()
	if len(lines) != 5 {
		t.Fatalf("expected 5 tax lines, got %d: %+v", len(lines), lines)
	}
	if lines[len(lines)-1].Name != "" {
		t.Errorf("unnamed line should come last: %+v", lines)
	}
	for i := 1; i < len(lines)-1; i++ {
		if lines[i-1].Name > lines[i].Name {
			t.Errorf("named lines not sorted: %+v", lines)
		}
	}

	var sum decimal.Decimal
	for _, l := range lines {
		sum = sum.Add(l.Amount)
	}
	if !sum.Equal(doc.Tax()) {
		t.Errorf("tax lines sum %s != Tax() %s", sum, doc.Tax())
	}
	for _, l := range lines {
		if l.Name == "Eco" && !l.Amount.Equal(mustDec(t, "2.50")) {
			t.Errorf("fixed tax line: %s", l.Amount)
		}
	}
}

func TestTaxLinesNilWithoutNames(t *testing.T) {
	doc := breakdownDoc(t)
	doc.AppendItem(&Item{Name: "a", UnitCost: "10", Quantity: "1", Tax: &Tax{Percent: "20"}})
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if doc.TaxLines() != nil {
		t.Error("TaxLines should be nil when no tax has a name")
	}
}

// TestBreakdownPrecision rounds everything to one decimal.
func TestBreakdownPrecision(t *testing.T) {
	doc, err := New(Invoice)
	if err != nil {
		t.Fatal(err)
	}
	doc.SetRef("BD-2").SetDate("01/01/2024").SetCurrency("EUR", 1)
	doc.SetCompany(&Contact{Name: "A", Address: &Address{Address: "x"}})
	doc.SetCustomer(&Contact{Name: "B", Address: &Address{Address: "y"}})
	doc.AppendItem(&Item{Name: "a", UnitCost: "1234.56", Quantity: "1", Tax: &Tax{Percent: "10"}})
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	bd := doc.Breakdown()
	assertDec(t, "line total at precision 1", bd.LineTotal, "1234.6")
	assertDec(t, "tax at precision 1", bd.Tax, "123.5")
	assertDec(t, "grand total at precision 1", bd.GrandTotal, "1358.1")
}
