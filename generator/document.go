package generator

import (
	"errors"

	"github.com/creasty/defaults"
	"codeberg.org/go-pdf/fpdf"
	"github.com/go-playground/validator/v10"
	"github.com/leekchan/accounting"
)

var ErrInvalidDocumentType = errors.New("invalid document type")

// Document define base document
type Document struct {
	pdf *fpdf.Fpdf
	ac  accounting.Accounting

	Options      *Options      `json:"options,omitempty"`
	Header       *HeaderFooter `json:"header,omitempty"`
	Footer       *HeaderFooter `json:"footer,omitempty"`
	Type         string        `json:"type,omitempty" validate:"required,oneof=INVOICE DELIVERY_NOTE QUOTATION"`
	Ref          string        `json:"ref,omitempty" validate:"required,min=1,max=32"`
	Version      string        `json:"version,omitempty" validate:"max=32"`
	ClientRef    string        `json:"client_ref,omitempty" validate:"max=64"`
	Description  string        `json:"description,omitempty" validate:"max=1024"`
	Notes        string        `json:"notes,omitempty"`
	Company      *Contact      `json:"company,omitempty" validate:"required"`
	Customer     *Contact      `json:"customer,omitempty" validate:"required"`
	Items        []*Item       `json:"items,omitempty"`
	Date         string        `json:"date,omitempty"`
	ValidityDate string        `json:"validity_date,omitempty"`
	PaymentTerm  string        `json:"payment_term,omitempty"`
	DefaultTax   *Tax          `json:"default_tax,omitempty"`
	Discount     *Discount     `json:"discount,omitempty"`
}

// New return a new document with provided type and defaults
func New(docType string, options *Options) (*Document, error) {
	_ = defaults.Set(options)

	if docType != Invoice && docType != Quotation && docType != DeliveryNote {
		return nil, ErrInvalidDocumentType
	}

	doc := &Document{
		Options: options,
		Type:    docType,
	}

	doc.pdf = fpdf.New("P", "mm", "A4", "")
	registerDefaultFonts(doc.pdf)
	// UTF-8 fonts (registered above) pass strings straight through; no cp1252
	// translation is needed. Callers using a different font can override this.
	doc.Options.UnicodeTranslateFunc = func(s string) string { return s }

	doc.ac = accounting.Accounting{
		Symbol:    doc.Options.CurrencySymbol,
		Precision: doc.Options.CurrencyPrecision,
		Thousand:  doc.Options.CurrencyThousand,
		Decimal:   doc.Options.CurrencyDecimal,
	}

	return doc, nil
}

// Validate document fields and prepare all monetary values
func (d *Document) Validate() error {
	validate := validator.New()
	if err := validate.Struct(d); err != nil {
		return err
	}

	for _, item := range d.Items {
		if item.Tax == nil {
			item.Tax = d.DefaultTax
		}
		if err := item.Prepare(); err != nil {
			return err
		}
	}

	if d.Discount != nil {
		if err := d.Discount.Prepare(); err != nil {
			return err
		}
	}

	return nil
}

// Pdf returns the underlying *fpdf.Fpdf used to build the document
func (doc *Document) Pdf() *fpdf.Fpdf {
	return doc.pdf
}

// SetUnicodeTranslator sets a custom unicode translation function.
// See https://pkg.go.dev/codeberg.org/go-pdf/fpdf#UnicodeTranslator
func (doc *Document) SetUnicodeTranslator(fn UnicodeTranslateFunc) {
	doc.Options.UnicodeTranslateFunc = fn
}

// SetType sets the document type
func (d *Document) SetType(docType string) *Document {
	d.Type = docType
	return d
}

// SetHeader sets the document header
func (d *Document) SetHeader(header *HeaderFooter) *Document {
	d.Header = header
	return d
}

// SetFooter sets the document footer
func (d *Document) SetFooter(footer *HeaderFooter) *Document {
	d.Footer = footer
	return d
}

// SetRef sets the document reference
func (d *Document) SetRef(ref string) *Document {
	d.Ref = ref
	return d
}

// SetVersion sets the document version
func (d *Document) SetVersion(version string) *Document {
	d.Version = version
	return d
}

// SetDescription sets the document description
func (d *Document) SetDescription(desc string) *Document {
	d.Description = desc
	return d
}

// SetNotes sets the document notes
func (d *Document) SetNotes(notes string) *Document {
	d.Notes = notes
	return d
}

// SetCompany sets the company contact
func (d *Document) SetCompany(company *Contact) *Document {
	d.Company = company
	return d
}

// SetCustomer sets the customer contact
func (d *Document) SetCustomer(customer *Contact) *Document {
	d.Customer = customer
	return d
}

// AppendItem appends an item to the document
func (d *Document) AppendItem(item *Item) *Document {
	d.Items = append(d.Items, item)
	return d
}

// SetDate sets the document date
func (d *Document) SetDate(date string) *Document {
	d.Date = date
	return d
}

// SetPaymentTerm sets the payment term
func (d *Document) SetPaymentTerm(term string) *Document {
	d.PaymentTerm = term
	return d
}

// SetDefaultTax sets the default tax applied to items without an explicit tax
func (d *Document) SetDefaultTax(tax *Tax) *Document {
	d.DefaultTax = tax
	return d
}

// SetDiscount sets the document-level discount
func (d *Document) SetDiscount(discount *Discount) *Document {
	d.Discount = discount
	return d
}

func (doc *Document) encodeString(str string) string {
	return doc.Options.UnicodeTranslateFunc(str)
}

func (d *Document) typeAsString() string {
	if d.Type == Invoice {
		return d.Options.TextTypeInvoice
	}
	if d.Type == Quotation {
		return d.Options.TextTypeQuotation
	}
	return d.Options.TextTypeDeliveryNote
}

func (d *Document) fakePdfDoc() *Document {
	optsCopy := *d.Options
	fakeDoc, err := New(d.Type, &optsCopy)
	if err != nil {
		// Should never panic, since we already called New() with the same args.
		panic(err)
	}

	// Copy data fields so conditional rendering logic (e.g. discount block in
	// appendTotal, PaymentTerm guard in appendPaymentTerm) behaves identically
	// in the probe and in the real render.
	fakeDoc.Header = d.Header
	fakeDoc.Footer = d.Footer
	fakeDoc.Ref = d.Ref
	fakeDoc.Version = d.Version
	fakeDoc.ClientRef = d.ClientRef
	fakeDoc.Description = d.Description
	fakeDoc.Notes = d.Notes
	fakeDoc.Company = d.Company
	fakeDoc.Customer = d.Customer
	fakeDoc.Items = d.Items
	fakeDoc.Date = d.Date
	fakeDoc.ValidityDate = d.ValidityDate
	fakeDoc.PaymentTerm = d.PaymentTerm
	fakeDoc.DefaultTax = d.DefaultTax
	fakeDoc.Discount = d.Discount

	pageCount := d.pdf.PageCount()
	for i := 0; i < pageCount; i++ {
		fakeDoc.pdf.AddPage()
	}
	fakeDoc.pdf.SetPage(d.pdf.PageNo())
	fakeDoc.pdf.SetXY(d.pdf.GetXY())

	return fakeDoc
}

type pageTxnFn func(*Document)

// pageTxn dry-runs cb on a probe document to detect page breaks, then executes
// for real. If a page break is detected, AddPage is called on the real document
// and onBreak (if provided) runs before the real render — use it to redraw
// table headers or reset font/position state.
func (d *Document) pageTxn(cb pageTxnFn, onBreak ...pageTxnFn) {
	fdoc := d.fakePdfDoc()

	currentPage := fdoc.pdf.PageNo()
	cb(fdoc)

	if fdoc.pdf.PageNo() > currentPage || fdoc.pdf.GetY() > MaxPageHeight {
		d.pdf.AddPage()
		if len(onBreak) > 0 && onBreak[0] != nil {
			onBreak[0](d)
		}
	}

	cb(d)
}
