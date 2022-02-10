package generator

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"
)

// Build pdf document from data provided
func (doc *Document) Build() (*gofpdf.Fpdf, error) {
	// Validate document data
	err := doc.Validate()
	if err != nil {
		return nil, err
	}

	// Build base doc
	doc.pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
	doc.pdf.SetXY(10, 10)
	doc.pdf.SetTextColor(
		doc.Options.BaseTextColor[0],
		doc.Options.BaseTextColor[1],
		doc.Options.BaseTextColor[2],
	)

	// Set header
	if doc.Header != nil {
		err = doc.Header.applyHeader(doc)

		if err != nil {
			return nil, err
		}
	}

	// Set footer
	if doc.Footer != nil {
		err = doc.Footer.applyFooter(doc)

		if err != nil {
			return nil, err
		}
	}

	// Add first page
	doc.pdf.AddPage()

	// Load font
	doc.pdf.SetFont("Helvetica", "", 12)

	// Appenf document title
	doc.appendTitle()

	// Appenf document metas (ref & version)
	doc.appendMetas()

	// Append company contact to doc
	companyBottom := doc.Company.appendCompanyContactToDoc(doc)

	// Append customer contact to doc
	customerBottom := doc.Customer.appendCustomerContactToDoc(doc)

	if customerBottom > companyBottom {
		doc.pdf.SetXY(10, customerBottom)
	} else {
		doc.pdf.SetXY(10, companyBottom)
	}

	// Append description
	doc.appendDescription()

	// Append items
	doc.appendItems()

	// Check page height (total bloc height = 30, 45 when doc discount)
	offset := doc.pdf.GetY() + 30
	if doc.Discount != nil {
		offset += 15
	}
	if offset > MaxPageHeight {
		doc.pdf.AddPage()
	}

	// Append notes
	doc.appendNotes()

	// Append total
	doc.appendTotal()

	// Append payment term
	doc.appendPaymentTerm()

	// Append js to autoprint if AutoPrint == true
	if doc.Options.AutoPrint {
		doc.pdf.SetJavascript("print(true);")
	}

	return doc.pdf, nil
}

func (doc *Document) appendTitle() {
	title := doc.typeAsString()

	// Set x y
	doc.pdf.SetXY(120, BaseMarginTop)

	// Draw rect
	doc.pdf.SetFillColor(doc.Options.DarkBgColor[0], doc.Options.DarkBgColor[1], doc.Options.DarkBgColor[2])
	doc.pdf.Rect(120, BaseMarginTop, 80, 10, "F")

	// Draw text
	doc.pdf.SetFont("Helvetica", "", 14)
	doc.pdf.CellFormat(80, 10, encodeString(title), "0", 0, "C", false, 0, "")
}

func (doc *Document) appendMetas() {
	// Append ref
	refString := fmt.Sprintf("%s: %s", doc.Options.TextRefTitle, doc.Ref)

	doc.pdf.SetXY(120, BaseMarginTop+11)
	doc.pdf.SetFont("Helvetica", "", 8)
	doc.pdf.CellFormat(80, 4, encodeString(refString), "0", 0, "R", false, 0, "")

	// Append version
	if len(doc.Version) > 0 {
		versionString := fmt.Sprintf("%s: %s", doc.Options.TextVersionTitle, doc.Version)
		doc.pdf.SetXY(120, BaseMarginTop+15)
		doc.pdf.SetFont("Helvetica", "", 8)
		doc.pdf.CellFormat(80, 4, encodeString(versionString), "0", 0, "R", false, 0, "")
	}

	// Append date
	date := time.Now().Format("02/01/2006")
	if len(doc.Date) > 0 {
		date = doc.Date
	}
	dateString := fmt.Sprintf("%s: %s", doc.Options.TextDateTitle, date)
	doc.pdf.SetXY(120, BaseMarginTop+19)
	doc.pdf.SetFont("Helvetica", "", 8)
	doc.pdf.CellFormat(80, 4, encodeString(dateString), "0", 0, "R", false, 0, "")
}

func (doc *Document) appendDescription() {
	if len(doc.Description) > 0 {
		doc.pdf.SetY(doc.pdf.GetY() + 10)
		doc.pdf.SetFont("Helvetica", "", 10)
		doc.pdf.MultiCell(190, 5, encodeString(doc.Description), "B", "L", false)
	}
}

func (doc *Document) drawsTableTitles() {
	// Draw table titles
	doc.pdf.SetX(10)
	doc.pdf.SetY(doc.pdf.GetY() + 5)
	doc.pdf.SetFont("Helvetica", "B", 8)

	// Draw rec
	doc.pdf.SetFillColor(doc.Options.GreyBgColor[0], doc.Options.GreyBgColor[1], doc.Options.GreyBgColor[2])
	doc.pdf.Rect(10, doc.pdf.GetY(), 190, 6, "F")

	// Name
	doc.pdf.SetX(ItemColNameOffset)
	doc.pdf.CellFormat(
		ItemColUnitPriceOffset-ItemColNameOffset,
		6,
		encodeString(doc.Options.TextItemsNameTitle),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Unit price
	doc.pdf.SetX(ItemColUnitPriceOffset)
	doc.pdf.CellFormat(
		ItemColQuantityOffset-ItemColUnitPriceOffset,
		6,
		encodeString(doc.Options.TextItemsUnitCostTitle),
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
		6,
		encodeString(doc.Options.TextItemsQuantityTitle),
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
		6,
		encodeString(doc.Options.TextItemsTotalHTTitle),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Tax
	doc.pdf.SetX(ItemColTaxOffset)
	doc.pdf.CellFormat(
		ItemColDiscountOffset-ItemColTaxOffset,
		6,
		encodeString(doc.Options.TextItemsTaxTitle),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Discount
	doc.pdf.SetX(ItemColDiscountOffset)
	doc.pdf.CellFormat(
		ItemColTotalTTCOffset-ItemColDiscountOffset,
		6,
		encodeString(doc.Options.TextItemsDiscountTitle),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// TOTAL TTC
	doc.pdf.SetX(ItemColTotalTTCOffset)
	doc.pdf.CellFormat(
		190-ItemColTotalTTCOffset,
		6,
		encodeString(doc.Options.TextItemsTotalTTCTitle),
		"0",
		0,
		"",
		false,
		0,
		"",
	)
}

func (doc *Document) appendItems() {
	doc.drawsTableTitles()

	doc.pdf.SetX(10)
	doc.pdf.SetY(doc.pdf.GetY() + 8)
	doc.pdf.SetFont("Helvetica", "", 8)

	for i := 0; i < len(doc.Items); i++ {
		item := doc.Items[i]

		// Check item tax
		if item.Tax == nil {
			item.Tax = doc.DefaultTax
		}

		// Append to pdf
		item.appendColTo(doc.Options, doc)

		if doc.pdf.GetY() > MaxPageHeight {
			// Add page
			doc.pdf.AddPage()
			doc.drawsTableTitles()
			doc.pdf.SetFont("Helvetica", "", 8)
		}

		doc.pdf.SetX(10)
		doc.pdf.SetY(doc.pdf.GetY() + 6)
	}
}

func (doc *Document) appendNotes() {
	if len(doc.Notes) == 0 {
		return
	}

	currentY := doc.pdf.GetY()

	doc.pdf.SetFont("Helvetica", "", 9)
	doc.pdf.SetX(BaseMargin)
	doc.pdf.SetRightMargin(100)
	doc.pdf.SetY(currentY + 10)

	_, lineHt := doc.pdf.GetFontSize()
	html := doc.pdf.HTMLBasicNew()
	html.Write(lineHt, encodeString(doc.Notes))

	doc.pdf.SetRightMargin(BaseMargin)
	doc.pdf.SetY(currentY)
}

func (doc *Document) appendTotal() {
	ac := accounting.Accounting{
		Symbol:    encodeString(doc.Options.CurrencySymbol),
		Precision: doc.Options.CurrencyPrecision,
		Thousand:  doc.Options.CurrencyThousand,
		Decimal:   doc.Options.CurrencyDecimal,
	}

	// Get total (without tax)
	total, _ := decimal.NewFromString("0")

	for _, item := range doc.Items {
		total = total.Add(item.totalWithoutTaxAndWithDiscount())
	}

	// Apply document discount
	totalWithDiscount := decimal.NewFromFloat(0)
	if doc.Discount != nil {
		discountType, discountNumber := doc.Discount.getDiscount()

		if discountType == "amount" {
			totalWithDiscount = total.Sub(discountNumber)
		} else {
			// Percent
			toSub := total.Mul(discountNumber.Div(decimal.NewFromFloat(100)))
			totalWithDiscount = total.Sub(toSub)
		}
	}

	// Tax
	totalTax := decimal.NewFromFloat(0)
	if doc.Discount == nil {
		for _, item := range doc.Items {
			totalTax = totalTax.Add(item.taxWithDiscount())
		}
	} else {
		discountType, discountAmount := doc.Discount.getDiscount()
		discountPercent := discountAmount
		if discountType == "amount" {
			// Get percent from total discounted
			discountPercent = discountAmount.Mul(decimal.NewFromFloat(100)).Div(totalWithDiscount)
		}

		for _, item := range doc.Items {
			if item.Tax != nil {
				taxType, taxAmount := item.Tax.getTax()
				if taxType == "amount" {
					// If tax type is amount, juste add amount to tax
					totalTax = totalTax.Add(taxAmount)
				} else {
					// Else, remove doc discount % from item total without tax and item discount
					itemTotal := item.totalWithoutTaxAndWithDiscount()
					toSub := discountPercent.Mul(itemTotal).Div(decimal.NewFromFloat(100))
					itemTotalDiscounted := itemTotal.Sub(toSub)

					// Then recompute tax on itemTotalDiscounted
					itemTaxDiscounted := taxAmount.Mul(itemTotalDiscounted).Div(decimal.NewFromFloat(100))

					totalTax = totalTax.Add(itemTaxDiscounted)
				}
			}
		}
	}

	// finalTotal
	totalWithTax := total.Add(totalTax)
	if doc.Discount != nil {
		totalWithTax = totalWithDiscount.Add(totalTax)
	}

	doc.pdf.SetY(doc.pdf.GetY() + 10)
	doc.pdf.SetFont("Helvetica", "", LargeTextFontSize)
	doc.pdf.SetTextColor(
		doc.Options.BaseTextColor[0],
		doc.Options.BaseTextColor[1],
		doc.Options.BaseTextColor[2],
	)

	// Draw TOTAL HT title
	doc.pdf.SetX(120)
	doc.pdf.SetFillColor(doc.Options.DarkBgColor[0], doc.Options.DarkBgColor[1], doc.Options.DarkBgColor[2])
	doc.pdf.Rect(120, doc.pdf.GetY(), 40, 10, "F")
	doc.pdf.CellFormat(38, 10, encodeString(doc.Options.TextTotalTotal), "0", 0, "R", false, 0, "")

	// Draw TOTAL HT amount
	doc.pdf.SetX(162)
	doc.pdf.SetFillColor(doc.Options.GreyBgColor[0], doc.Options.GreyBgColor[1], doc.Options.GreyBgColor[2])
	doc.pdf.Rect(160, doc.pdf.GetY(), 40, 10, "F")
	doc.pdf.CellFormat(40, 10, ac.FormatMoneyDecimal(total), "0", 0, "L", false, 0, "")

	if doc.Discount != nil {
		baseY := doc.pdf.GetY() + 10

		// Draw DISCOUNTED title
		doc.pdf.SetXY(120, baseY)
		doc.pdf.SetFillColor(doc.Options.DarkBgColor[0], doc.Options.DarkBgColor[1], doc.Options.DarkBgColor[2])
		doc.pdf.Rect(120, doc.pdf.GetY(), 40, 15, "F")

		// title
		doc.pdf.CellFormat(38, 7.5, encodeString(doc.Options.TextTotalDiscounted), "0", 0, "BR", false, 0, "")

		// description
		doc.pdf.SetXY(120, baseY+7.5)
		doc.pdf.SetFont("Helvetica", "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		var descString bytes.Buffer
		discountType, discountAmount := doc.Discount.getDiscount()
		if discountType == "percent" {
			descString.WriteString("-")
			descString.WriteString(discountAmount.String())
			descString.WriteString(" % / -")
			descString.WriteString(ac.FormatMoneyDecimal(total.Sub(totalWithDiscount)))
		} else {
			descString.WriteString("-")
			descString.WriteString(ac.FormatMoneyDecimal(discountAmount))
			descString.WriteString(" / -")
			descString.WriteString(
				discountAmount.Mul(decimal.NewFromFloat(100)).Div(total).StringFixed(2),
			)
			descString.WriteString(" %")
		}

		doc.pdf.CellFormat(38, 7.5, descString.String(), "0", 0, "TR", false, 0, "")

		doc.pdf.SetFont("Helvetica", "", LargeTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)

		// Draw DISCOUNT amount
		doc.pdf.SetY(baseY)
		doc.pdf.SetX(162)
		doc.pdf.SetFillColor(doc.Options.GreyBgColor[0], doc.Options.GreyBgColor[1], doc.Options.GreyBgColor[2])
		doc.pdf.Rect(160, doc.pdf.GetY(), 40, 15, "F")
		doc.pdf.CellFormat(40, 15, ac.FormatMoneyDecimal(totalWithDiscount), "0", 0, "L", false, 0, "")
		doc.pdf.SetY(doc.pdf.GetY() + 15)
	} else {
		doc.pdf.SetY(doc.pdf.GetY() + 10)
	}

	// Draw TAX title
	doc.pdf.SetX(120)
	doc.pdf.SetFillColor(doc.Options.DarkBgColor[0], doc.Options.DarkBgColor[1], doc.Options.DarkBgColor[2])
	doc.pdf.Rect(120, doc.pdf.GetY(), 40, 10, "F")
	doc.pdf.CellFormat(38, 10, encodeString(doc.Options.TextTotalTax), "0", 0, "R", false, 0, "")

	// Draw TAX amount
	doc.pdf.SetX(162)
	doc.pdf.SetFillColor(doc.Options.GreyBgColor[0], doc.Options.GreyBgColor[1], doc.Options.GreyBgColor[2])
	doc.pdf.Rect(160, doc.pdf.GetY(), 40, 10, "F")
	doc.pdf.CellFormat(40, 10, ac.FormatMoneyDecimal(totalTax), "0", 0, "L", false, 0, "")

	// Draw TOTAL TTC title
	doc.pdf.SetY(doc.pdf.GetY() + 10)
	doc.pdf.SetX(120)
	doc.pdf.SetFillColor(doc.Options.DarkBgColor[0], doc.Options.DarkBgColor[1], doc.Options.DarkBgColor[2])
	doc.pdf.Rect(120, doc.pdf.GetY(), 40, 10, "F")
	doc.pdf.CellFormat(38, 10, encodeString(doc.Options.TextTotalWithTax), "0", 0, "R", false, 0, "")

	// Draw TOTAL TTC amount
	doc.pdf.SetX(162)
	doc.pdf.SetFillColor(doc.Options.GreyBgColor[0], doc.Options.GreyBgColor[1], doc.Options.GreyBgColor[2])
	doc.pdf.Rect(160, doc.pdf.GetY(), 40, 10, "F")
	doc.pdf.CellFormat(40, 10, ac.FormatMoneyDecimal(totalWithTax), "0", 0, "L", false, 0, "")
}

func (doc *Document) appendPaymentTerm() {
	if len(doc.PaymentTerm) > 0 {
		paymentTermString := fmt.Sprintf(
			"%s: %s",
			encodeString(doc.Options.TextPaymentTermTitle),
			encodeString(doc.PaymentTerm),
		)
		doc.pdf.SetY(doc.pdf.GetY() + 15)

		doc.pdf.SetX(120)
		doc.pdf.SetFont("Helvetica", "B", 10)
		doc.pdf.CellFormat(80, 4, paymentTermString, "0", 0, "R", false, 0, "")
	}
}
