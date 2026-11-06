package generator

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"codeberg.org/go-pdf/fpdf"
	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

// renderer draws one invoice.Document on one fpdf instance.
type renderer struct {
	pdf  *fpdf.Fpdf
	ac   accounting.Accounting
	opts *Options
	doc  *invoice.Document
	gen  *Generator
}

func (r *renderer) encode(s string) string { return r.gen.translate(s) }

func (r *renderer) money(d decimal.Decimal) string { return r.ac.FormatMoneyDecimal(d) }

func (r *renderer) setTextColor(c []int) { r.pdf.SetTextColor(c[0], c[1], c[2]) }

func (r *renderer) setFillColor(c []int) { r.pdf.SetFillColor(c[0], c[1], c[2]) }

func (r *renderer) typeAsString() string {
	switch r.doc.Type {
	case invoice.Invoice:
		return r.opts.TextTypeInvoice
	case invoice.Quotation:
		return r.opts.TextTypeQuotation
	case invoice.CreditNote:
		return r.opts.TextTypeCreditNote
	default:
		return r.opts.TextTypeDeliveryNote
	}
}

// probe returns a renderer on a throwaway PDF positioned like r, used to
// detect page breaks before drawing for real.
func (r *renderer) probe() *renderer {
	p := &renderer{pdf: newPDF(), ac: r.ac, opts: r.opts, doc: r.doc, gen: r.gen}
	for i := 0; i < r.pdf.PageCount(); i++ {
		p.pdf.AddPage()
	}
	p.pdf.SetPage(r.pdf.PageNo())
	p.pdf.SetXY(r.pdf.GetXY())
	return p
}

// pageTxn dry-runs cb on a probe to detect page breaks, then executes for
// real. On a break, AddPage is called and onBreak (if any) runs first, e.g. to
// redraw table headers.
func (r *renderer) pageTxn(cb func(*renderer), onBreak ...func(*renderer)) {
	p := r.probe()
	currentPage := p.pdf.PageNo()
	cb(p)

	if p.pdf.PageNo() > currentPage || p.pdf.GetY() > MaxPageHeight {
		r.pdf.AddPage()
		if len(onBreak) > 0 && onBreak[0] != nil {
			onBreak[0](r)
		}
	}

	cb(r)
}

func (r *renderer) build() (*fpdf.Fpdf, error) {
	r.pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
	r.pdf.SetXY(10, 10)
	r.setTextColor(r.opts.BaseTextColor)

	if r.gen.Header != nil {
		if err := r.gen.Header.applyHeader(r); err != nil {
			return nil, err
		}
	}
	if r.gen.Footer != nil {
		if err := r.gen.Footer.applyFooter(r); err != nil {
			return nil, err
		}
	}

	r.pdf.AddPage()
	r.pdf.SetFont(r.opts.Font, "", 12)

	r.appendTitle()
	r.appendMetas()

	x, y, _, _ := r.pdf.GetMargins()
	companyBottom := r.appendContact(r.doc.Company, x, y)
	customerBottom := r.appendContact(r.doc.Customer, 130, BaseMarginTop+25)
	if customerBottom > companyBottom {
		r.pdf.SetXY(10, customerBottom)
	} else {
		r.pdf.SetXY(10, companyBottom)
	}

	r.appendDescription()
	r.appendDelivery()
	r.appendItems()

	// Total and payment term share the right column and must stay together.
	r.pageTxn(func(p *renderer) {
		// Notes resets Y after rendering (left column, side-by-side with total).
		p.appendNotes()
		p.appendTotal()
		p.appendPaymentTerm()
	})

	return r.pdf, nil
}

func (r *renderer) appendTitle() {
	r.pdf.SetXY(120, BaseMarginTop)
	r.setFillColor(r.opts.DarkBgColor)
	r.pdf.Rect(120, BaseMarginTop, 80, 10, "F")
	r.pdf.SetFont(r.opts.Font, "", 14)
	r.pdf.CellFormat(80, 10, r.encode(r.typeAsString()), "0", 0, "C", false, 0, "")
}

func (r *renderer) appendMetas() {
	r.pdf.SetFont(r.opts.Font, "", 8)

	r.pdf.SetXY(120, BaseMarginTop+11)
	r.pdf.CellFormat(80, 4, r.encode(fmt.Sprintf("%s: %s", r.opts.TextRefTitle, r.doc.Ref)), "0", 0, "R", false, 0, "")

	if len(r.doc.Version) > 0 {
		r.pdf.SetXY(120, BaseMarginTop+15)
		r.pdf.CellFormat(80, 4, r.encode(fmt.Sprintf("%s: %s", r.opts.TextVersionTitle, r.doc.Version)), "0", 0, "R", false, 0, "")
	}

	date := time.Now().Format("02/01/2006")
	if len(r.doc.Date) > 0 {
		date = r.doc.Date
	}
	r.pdf.SetXY(120, BaseMarginTop+19)
	r.pdf.CellFormat(80, 4, r.encode(fmt.Sprintf("%s: %s", r.opts.TextDateTitle, date)), "0", 0, "R", false, 0, "")
}

func (r *renderer) appendDescription() {
	if ref := r.doc.PrecedingInvoice; ref != nil && ref.Ref != "" {
		text := fmt.Sprintf("%s: %s", r.opts.TextPrecedingInvoiceTitle, ref.Ref)
		if ref.Date != "" {
			text += fmt.Sprintf(" (%s)", ref.Date)
		}
		r.pdf.SetY(r.pdf.GetY() + 10)
		r.pdf.SetFont(r.opts.Font, "", 10)
		r.pdf.MultiCell(190, 5, r.encode(text), "", "L", false)
		if len(r.doc.Description) > 0 {
			r.pdf.SetY(r.pdf.GetY() - 10)
		}
	}
	if len(r.doc.Description) > 0 {
		r.pdf.SetY(r.pdf.GetY() + 10)
		r.pdf.SetFont(r.opts.Font, "", 10)
		r.pdf.MultiCell(190, 5, r.encode(r.doc.Description), "B", "L", false)
	}
}

// appendDelivery prints the delivery date, period and deliver-to address.
func (r *renderer) appendDelivery() {
	d := r.doc.Delivery
	if d == nil {
		return
	}
	var lines []string
	if d.Date != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", r.opts.TextDeliveryDateTitle, d.Date))
	}
	if d.PeriodStart != "" || d.PeriodEnd != "" {
		lines = append(lines, fmt.Sprintf("%s: %s - %s", r.opts.TextDeliveryPeriodTitle, d.PeriodStart, d.PeriodEnd))
	}
	if d.ShipTo != nil {
		parts := []string{}
		if d.ShipTo.Name != "" {
			parts = append(parts, d.ShipTo.Name)
		}
		if d.ShipTo.Address != nil {
			parts = append(parts, strings.ReplaceAll(d.ShipTo.Address.ToString(), "\n", ", "))
		}
		lines = append(lines, fmt.Sprintf("%s: %s", r.opts.TextDeliveredToTitle, strings.Join(parts, ", ")))
	}
	if len(lines) == 0 {
		return
	}
	r.pdf.SetY(r.pdf.GetY() + 5)
	r.pdf.SetFont(r.opts.Font, "", BaseTextFontSize)
	r.setTextColor(r.opts.GreyTextColor)
	for _, line := range lines {
		r.pdf.SetX(BaseMargin)
		r.pdf.MultiCell(190, 4, r.encode(line), "", "L", false)
	}
	r.setTextColor(r.opts.BaseTextColor)
}

func (r *renderer) drawTableTitles() {
	r.pdf.SetX(10)
	r.pdf.SetY(r.pdf.GetY() + 5)
	r.pdf.SetFont(r.opts.BoldFont, "B", 8)

	r.setFillColor(r.opts.GreyBgColor)
	r.pdf.Rect(10, r.pdf.GetY(), 190, 6, "F")

	cols := []struct {
		x, w  float64
		title string
	}{
		{ItemColNameOffset, ItemColUnitPriceOffset - ItemColNameOffset, r.opts.TextItemsNameTitle},
		{ItemColUnitPriceOffset, ItemColQuantityOffset - ItemColUnitPriceOffset, r.opts.TextItemsUnitCostTitle},
		{ItemColQuantityOffset, ItemColTaxOffset - ItemColQuantityOffset, r.opts.TextItemsQuantityTitle},
		{ItemColTotalHTOffset, ItemColTaxOffset - ItemColTotalHTOffset, r.opts.TextItemsTotalHTTitle},
		{ItemColTaxOffset, ItemColDiscountOffset - ItemColTaxOffset, r.opts.TextItemsTaxTitle},
		{ItemColDiscountOffset, ItemColTotalTTCOffset - ItemColDiscountOffset, r.opts.TextItemsDiscountTitle},
		{ItemColTotalTTCOffset, 190 - ItemColTotalTTCOffset, r.opts.TextItemsTotalTTCTitle},
	}
	for _, c := range cols {
		r.pdf.SetX(c.x)
		r.pdf.CellFormat(c.w, 6, r.encode(c.title), "0", 0, "", false, 0, "")
	}
}

func (r *renderer) appendItems() {
	r.drawTableTitles()

	r.pdf.SetX(10)
	r.pdf.SetY(r.pdf.GetY() + 8)
	r.pdf.SetFont(r.opts.Font, "", 8)

	for _, item := range r.doc.Items {
		r.pageTxn(func(p *renderer) {
			p.appendItem(item)
		}, func(p *renderer) {
			p.drawTableTitles()
			p.pdf.SetFont(p.opts.Font, "", 8)
			p.pdf.SetX(10)
			p.pdf.SetY(p.pdf.GetY() + 8)
		})

		// Gray separator line at the bottom of the item row
		r.pdf.SetY(r.pdf.GetY() + 3)
		lineY := r.pdf.GetY()
		r.pdf.SetDrawColor(r.opts.GreyBgColor[0], r.opts.GreyBgColor[1], r.opts.GreyBgColor[2])
		r.pdf.Line(10, lineY, 200, lineY)
		r.pdf.SetDrawColor(0, 0, 0)

		r.pdf.SetX(10)
		r.pdf.SetY(r.pdf.GetY() + 3)
	}
}

func (r *renderer) appendNotes() {
	if len(r.doc.Notes) == 0 {
		return
	}
	contents := make([]string, 0, len(r.doc.Notes))
	for _, n := range r.doc.Notes {
		if n.Content != "" {
			contents = append(contents, n.Content)
		}
	}
	if len(contents) == 0 {
		return
	}

	currentY := r.pdf.GetY()

	r.pdf.SetFont(r.opts.Font, "", 9)
	r.pdf.SetX(BaseMargin)
	r.pdf.SetRightMargin(100)
	r.pdf.SetY(currentY + 10)

	_, lineHt := r.pdf.GetFontSize()
	html := r.pdf.HTMLBasicNew()
	html.Write(lineHt, r.encode(strings.Join(contents, "<br>")))

	r.pdf.SetRightMargin(BaseMargin)
	r.pdf.SetY(currentY)
}

func (r *renderer) appendTotal() {
	r.pdf.SetY(r.pdf.GetY() + 10)
	r.pdf.SetFont(r.opts.Font, "", LargeTextFontSize)
	r.setTextColor(r.opts.BaseTextColor)

	// Total before document discount
	r.totalRow(r.opts.TextTotalTotal, r.money(r.doc.TotalWithoutTaxAndWithoutDocumentDiscount()))

	if r.doc.Discount != nil {
		baseY := r.pdf.GetY()

		r.pdf.SetXY(120, baseY)
		r.setFillColor(r.opts.DarkBgColor)
		r.pdf.Rect(120, baseY, 40, 15, "F")
		r.pdf.CellFormat(38, 7.5, r.encode(r.opts.TextTotalDiscounted), "0", 0, "BR", false, 0, "")

		r.pdf.SetXY(120, baseY+7.5)
		r.pdf.SetFont(r.opts.Font, "", BaseTextFontSize)
		r.setTextColor(r.opts.GreyTextColor)

		var desc bytes.Buffer
		discountType, discountAmount := r.doc.Discount.Value()
		if discountType == invoice.DiscountTypePercent {
			desc.WriteString("-")
			desc.WriteString(discountAmount.String())
			desc.WriteString(" % / -")
			desc.WriteString(r.money(r.doc.DocumentDiscount()))
		} else {
			desc.WriteString("-")
			desc.WriteString(r.money(discountAmount))
			desc.WriteString(" / -")
			if base := r.doc.TotalWithoutTaxAndWithoutDocumentDiscount(); !base.IsZero() {
				desc.WriteString(discountAmount.Mul(decimal.NewFromInt(100)).Div(base).StringFixed(2))
			} else {
				desc.WriteString("0.00")
			}
			desc.WriteString(" %")
		}
		r.pdf.CellFormat(38, 7.5, r.encode(desc.String()), "0", 0, "TR", false, 0, "")

		r.pdf.SetFont(r.opts.Font, "", LargeTextFontSize)
		r.setTextColor(r.opts.BaseTextColor)

		r.pdf.SetXY(162, baseY)
		r.setFillColor(r.opts.GreyBgColor)
		r.pdf.Rect(160, baseY, 40, 15, "F")
		r.pdf.CellFormat(40, 15, r.encode(r.money(r.doc.TotalWithoutTax())), "0", 0, "L", false, 0, "")
		r.pdf.SetY(baseY + 15)
	}

	// Tax
	r.totalRow(r.opts.TextTotalTax, r.money(r.doc.Tax()))

	// Per-name breakdown in smaller font when named taxes exist.
	if taxLines := r.doc.TaxLines(); taxLines != nil {
		r.pdf.SetFont(r.opts.Font, "", SmallTextFontSize)
		r.setTextColor(r.opts.GreyTextColor)
		for _, tl := range taxLines {
			label := tl.Name
			if label == "" {
				label = r.opts.TextTotalTaxOther
				if label == "" {
					label = "Other"
				}
			}
			r.pdf.SetX(120)
			r.setFillColor(r.opts.GreyBgColor)
			r.pdf.Rect(120, r.pdf.GetY(), 80, 6, "F")
			r.pdf.CellFormat(38, 6, r.encode(label), "0", 0, "R", false, 0, "")
			r.pdf.SetX(162)
			r.pdf.CellFormat(40, 6, r.encode(r.money(tl.Amount)), "0", 0, "L", false, 0, "")
			r.pdf.SetY(r.pdf.GetY() + 6)
		}
		r.pdf.SetFont(r.opts.Font, "", LargeTextFontSize)
		r.setTextColor(r.opts.BaseTextColor)
	}

	// Total with tax; the row leaves Y at its bottom edge.
	r.totalRow(r.opts.TextTotalWithTax, r.money(r.doc.TotalWithTax()))
	r.pdf.SetY(r.pdf.GetY() - 10)
}

// totalRow draws a 10mm label/value row at the current Y and moves Y below it.
func (r *renderer) totalRow(label, value string) {
	y := r.pdf.GetY()
	r.pdf.SetX(120)
	r.setFillColor(r.opts.DarkBgColor)
	r.pdf.Rect(120, y, 40, 10, "F")
	r.pdf.CellFormat(38, 10, r.encode(label), "0", 0, "R", false, 0, "")

	r.pdf.SetX(162)
	r.setFillColor(r.opts.GreyBgColor)
	r.pdf.Rect(160, y, 40, 10, "F")
	r.pdf.CellFormat(40, 10, r.encode(value), "0", 0, "L", false, 0, "")
	r.pdf.SetY(y + 10)
}

// appendPaymentTerm prints the due date, payment terms and bank details.
func (r *renderer) appendPaymentTerm() {
	var lines []string
	if r.doc.DueDate != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", r.opts.TextDueDateTitle, r.doc.DueDate))
	}
	if r.doc.PaymentTerm != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", r.opts.TextPaymentTermTitle, r.doc.PaymentTerm))
	}
	var bank []string
	if p := r.doc.Payment; p != nil {
		if p.IBAN != "" {
			bank = append(bank, "IBAN: "+p.IBAN)
		}
		if p.BIC != "" {
			bank = append(bank, "BIC: "+p.BIC)
		}
		if p.Reference != "" {
			bank = append(bank, r.opts.TextPaymentReferenceTitle+": "+p.Reference)
		}
	}
	if len(lines) == 0 && len(bank) == 0 {
		return
	}

	r.pdf.SetY(r.pdf.GetY() + 15)
	r.pdf.SetFont(r.opts.BoldFont, "B", 10)
	for _, line := range lines {
		r.pdf.SetX(120)
		r.pdf.CellFormat(80, 4, r.encode(line), "0", 0, "R", false, 0, "")
		r.pdf.SetY(r.pdf.GetY() + 5)
	}
	if len(bank) > 0 {
		r.pdf.SetFont(r.opts.Font, "", BaseTextFontSize)
		r.setTextColor(r.opts.GreyTextColor)
		for _, line := range bank {
			r.pdf.SetX(120)
			r.pdf.CellFormat(80, 4, r.encode(line), "0", 0, "R", false, 0, "")
			r.pdf.SetY(r.pdf.GetY() + 4)
		}
		r.setTextColor(r.opts.BaseTextColor)
	}
}
