package generator

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"
)

// Document define base document
type Document struct {
	Options      *Options
	Header       *HeaderFooter
	Footer       *HeaderFooter
	Type         string `validate:"required,oneof=INVOICE DELIVERY_NOTE QUOTATION"`
	Ref          string `validate:"required,min=1,max=32"`
	Version      string `validate:"max=32"`
	ClientRef    string `validate:"max=64"`
	Description  string `validate:"max=1024"`
	Notes        string
	Company      *Contact `validate:"required"`
	Customer     *Contact `validate:"required"`
	Items        []*Item
	Date         string
	ValidityDate string
	PaymentTerm  string
}

// Build pdf document from data provided
func (d *Document) Build() (*gofpdf.Fpdf, error) {
	// Validate document data
	err := d.Validate()
	if err != nil {
		return nil, err
	}

	// Build base doc
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
	pdf.SetXY(10, 10)
	pdf.SetTextColor(BaseTextColor[0], BaseTextColor[1], BaseTextColor[2])

	// Set header
	if d.Header != nil {
		err = d.Header.applyHeader(d, pdf)

		if err != nil {
			return nil, err
		}
	}

	// Set footer
	if d.Footer != nil {
		err = d.Footer.applyFooter(d, pdf)

		if err != nil {
			return nil, err
		}
	}

	// Add first page
	pdf.AddPage()

	// Load font
	pdf.SetFont("Helvetica", "", 12)

	// Appenf document title
	d.appendTitle(pdf)

	// Appenf document metas (ref & version)
	d.appendMetas(pdf)

	// Append company contact to doc
	companyBottom := d.Company.appendCompanyContactToDoc(pdf)

	// Append customer contact to doc
	customerBottom := d.Customer.appendCustomerContactToDoc(pdf)

	if customerBottom > companyBottom {
		pdf.SetXY(10, customerBottom)
	} else {
		pdf.SetXY(10, companyBottom)
	}

	// Append description
	d.appendDescription(pdf)

	// Append items
	d.appendItems(pdf)

	// Append notes
	d.appendNotes(pdf)

	// Append total
	d.appendTotal(pdf)

	// Append payment term
	d.appendPaymentTerm(pdf)

	// Append js to autoprint if AutoPrint == true
	if d.Options.AutoPrint {
		pdf.SetJavascript("print(true);")
	}

	return pdf, nil
}

// SetType set type of document
func (d *Document) SetType(docType string) *Document {
	d.Type = docType
	return d
}

// SetHeader set header of document
func (d *Document) SetHeader(header *HeaderFooter) *Document {
	d.Header = header
	return d
}

// SetFooter set footer of document
func (d *Document) SetFooter(footer *HeaderFooter) *Document {
	d.Footer = footer
	return d
}

// SetRef of document
func (d *Document) SetRef(ref string) *Document {
	d.Ref = ref
	return d
}

// SetVersion of document
func (d *Document) SetVersion(version string) *Document {
	d.Version = version
	return d
}

// SetDescription of document
func (d *Document) SetDescription(desc string) *Document {
	d.Description = desc
	return d
}

// SetNotes of document
func (d *Document) SetNotes(notes string) *Document {
	d.Notes = notes
	return d
}

// SetCompany of document
func (d *Document) SetCompany(company *Contact) *Document {
	d.Company = company
	return d
}

// SetCustomer of document
func (d *Document) SetCustomer(customer *Contact) *Document {
	d.Customer = customer
	return d
}

// AppendItem to document items
func (d *Document) AppendItem(item *Item) *Document {
	d.Items = append(d.Items, item)
	return d
}

// SetDate of document
func (d *Document) SetDate(date string) *Document {
	d.Date = date
	return d
}

// SetPaymentTerm of document
func (d *Document) SetPaymentTerm(term string) *Document {
	d.PaymentTerm = term
	return d
}

// ===========================
// PRIVATE ===================
// ===========================

func (d *Document) appendTitle(pdf *gofpdf.Fpdf) {
	title := d.typeAsString()

	// Set x y
	pdf.SetXY(120, BaseMarginTop)

	// Draw rect
	pdf.SetFillColor(DarkBgColor[0], DarkBgColor[1], DarkBgColor[2])
	pdf.Rect(120, BaseMarginTop, 80, 10, "F")

	// Draw text
	pdf.SetFont("Helvetica", "", 14)
	pdf.CellFormat(80, 10, encodeString(title), "0", 0, "C", false, 0, "")
}

func (d *Document) appendMetas(pdf *gofpdf.Fpdf) {
	// Append ref
	refString := fmt.Sprintf("%s: %s", encodeString(d.Options.TextRefTitle), d.Ref)

	pdf.SetXY(120, BaseMarginTop+11)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(80, 4, encodeString(refString), "0", 0, "R", false, 0, "")

	// Append version
	if len(d.Version) > 0 {
		versionString := fmt.Sprintf("%s: %s", encodeString(d.Options.TextVersionTitle), d.Version)
		pdf.SetXY(120, BaseMarginTop+15)
		pdf.SetFont("Helvetica", "", 8)
		pdf.CellFormat(80, 4, encodeString(versionString), "0", 0, "R", false, 0, "")
	}

	// Append date
	date := time.Now().Format("02/01/2006")
	if len(d.Date) > 0 {
		date = d.Date
	}
	dateString := fmt.Sprintf("%s: %s", encodeString(d.Options.TextDateTitle), date)
	pdf.SetXY(120, BaseMarginTop+19)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(80, 4, encodeString(dateString), "0", 0, "R", false, 0, "")
}

func (d *Document) appendDescription(pdf *gofpdf.Fpdf) {
	if len(d.Description) > 0 {
		pdf.SetY(pdf.GetY() + 10)
		pdf.SetFont("Helvetica", "", 10)
		pdf.MultiCell(190, 5, encodeString(d.Description), "B", "L", false)
	}
}

func (d *Document) drawsTableTitles(pdf *gofpdf.Fpdf) {
	// Draw table titles
	pdf.SetX(10)
	pdf.SetY(pdf.GetY() + 5)
	pdf.SetFont("Helvetica", "B", 8)

	// Draw rec
	pdf.SetFillColor(GreyBgColor[0], GreyBgColor[1], GreyBgColor[2])
	pdf.Rect(10, pdf.GetY(), 190, 6, "F")

	// Description
	pdf.CellFormat(80, 6, encodeString(d.Options.TextItemsDescriptionTitle), "0", 0, "", false, 0, "")

	// Unit price
	pdf.SetX(90)
	pdf.CellFormat(30, 6, encodeString(d.Options.TextItemsUnitCostTitle), "0", 0, "", false, 0, "")

	// Quantity
	pdf.SetX(120)
	pdf.CellFormat(15, 6, encodeString(d.Options.TextItemsQuantityTitle), "0", 0, "", false, 0, "")

	// Total HT
	pdf.SetX(135)
	pdf.CellFormat(20, 6, encodeString(d.Options.TextItemsTotalHTTitle), "0", 0, "", false, 0, "")

	// Tax
	pdf.SetX(155)
	pdf.CellFormat(20, 6, encodeString(d.Options.TextItemsTaxTitle), "0", 0, "", false, 0, "")

	// TOTAL TTC
	pdf.SetX(175)
	pdf.CellFormat(25, 6, encodeString(d.Options.TextItemsTotalTTCTitle), "0", 0, "", false, 0, "")
}

func (d *Document) appendItems(pdf *gofpdf.Fpdf) {
	d.drawsTableTitles(pdf)

	pdf.SetX(10)
	pdf.SetY(pdf.GetY() + 6)
	pdf.SetFont("Helvetica", "", 8)

	for i := 0; i < len(d.Items); i++ {
		item := d.Items[i]
		item.appendColTo(d.Options, pdf)

		if pdf.GetY() > MaxPageHeight {
			// Add page
			pdf.AddPage()
			d.drawsTableTitles(pdf)
			pdf.SetFont("Helvetica", "", 8)
		}

		pdf.SetX(10)
		pdf.SetY(pdf.GetY() + 6)
	}
}

func (d *Document) appendNotes(pdf *gofpdf.Fpdf) {
	if len(d.Notes) == 0 {
		return
	}

	currentY := pdf.GetY()

	if currentY+30 > MaxPageHeight {
		pdf.AddPage()
		currentY = pdf.GetY()
	}

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetX(BaseMargin)
	pdf.SetRightMargin(100)
	pdf.SetY(currentY + 10)

	_, lineHt := pdf.GetFontSize()
	html := pdf.HTMLBasicNew()
	html.Write(lineHt, d.Notes)

	pdf.SetRightMargin(BaseMargin)
	pdf.SetY(currentY)
}

func (d *Document) appendTotal(pdf *gofpdf.Fpdf) {
	ac := accounting.Accounting{
		Symbol:    encodeString(d.Options.CurrencySymbol),
		Precision: d.Options.CurrencyPrecision,
		Thousand:  d.Options.CurrencyThousand,
		Decimal:   d.Options.CurrencyDecimal,
	}

	// Get total HT & totalTTC
	totalHT, _ := decimal.NewFromString("0")
	totalTTC, _ := decimal.NewFromString("0")

	for i := 0; i < len(d.Items); i++ {
		item := d.Items[i]
		totalHT = totalHT.Add(item.totalHT())
		totalTTC = totalTTC.Add(item.totalTTC())
	}

	totalTax := totalTTC.Sub(totalHT)

	// Check page height (total bloc height = 30)
	if pdf.GetY()+30 > MaxPageHeight {
		pdf.AddPage()
	}

	pdf.SetY(pdf.GetY() + 10)
	pdf.SetFont("Helvetica", "", 10)

	// Draw TOTAL HT title
	pdf.SetX(120)
	pdf.SetFillColor(DarkBgColor[0], DarkBgColor[1], DarkBgColor[2])
	pdf.Rect(120, pdf.GetY(), 40, 10, "F")
	pdf.CellFormat(38, 10, encodeString(d.Options.TextTotalTotal), "0", 0, "R", false, 0, "")

	// Draw TOTAL HT amount
	pdf.SetX(162)
	pdf.SetFillColor(GreyBgColor[0], GreyBgColor[1], GreyBgColor[2])
	pdf.Rect(160, pdf.GetY(), 40, 10, "F")
	pdf.CellFormat(40, 10, ac.FormatMoneyDecimal(totalHT), "0", 0, "L", false, 0, "")

	// Draw TAX title
	pdf.SetY(pdf.GetY() + 10)
	pdf.SetX(120)
	pdf.SetFillColor(DarkBgColor[0], DarkBgColor[1], DarkBgColor[2])
	pdf.Rect(120, pdf.GetY(), 40, 10, "F")
	pdf.CellFormat(38, 10, encodeString(d.Options.TextTotalTax), "0", 0, "R", false, 0, "")

	// Draw TAX amount
	pdf.SetX(162)
	pdf.SetFillColor(GreyBgColor[0], GreyBgColor[1], GreyBgColor[2])
	pdf.Rect(160, pdf.GetY(), 40, 10, "F")
	pdf.CellFormat(40, 10, ac.FormatMoneyDecimal(totalTax), "0", 0, "L", false, 0, "")

	// Draw TOTAL TTC title
	pdf.SetY(pdf.GetY() + 10)
	pdf.SetX(120)
	pdf.SetFillColor(DarkBgColor[0], DarkBgColor[1], DarkBgColor[2])
	pdf.Rect(120, pdf.GetY(), 40, 10, "F")
	pdf.CellFormat(38, 10, encodeString(d.Options.TextTotalWithTax), "0", 0, "R", false, 0, "")

	// Draw TOTAL TTC amount
	pdf.SetX(162)
	pdf.SetFillColor(GreyBgColor[0], GreyBgColor[1], GreyBgColor[2])
	pdf.Rect(160, pdf.GetY(), 40, 10, "F")
	pdf.CellFormat(40, 10, ac.FormatMoneyDecimal(totalTTC), "0", 0, "L", false, 0, "")
}

func (d *Document) appendPaymentTerm(pdf *gofpdf.Fpdf) {
	if len(d.PaymentTerm) > 0 {
		paymentTermString := fmt.Sprintf("%s: %s", encodeString(d.Options.TextPaymentTermTitle), encodeString(d.PaymentTerm))
		pdf.SetY(pdf.GetY() + 15)

		pdf.SetX(120)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(80, 4, paymentTermString, "0", 0, "R", false, 0, "")
	}
}
