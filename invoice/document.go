// Package invoice holds the data model of a commercial document (invoice,
// quotation, delivery note or credit note) together with its validation and
// monetary arithmetic. It has no rendering dependency: the generator package
// draws a PDF from it and the facturx package derives an e-invoice from it.
package invoice

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Document types.
const (
	// Invoice is a commercial invoice (UN/CEFACT 380).
	Invoice string = "INVOICE"
	// Quotation is a quote; it has no e-invoice representation.
	Quotation string = "QUOTATION"
	// DeliveryNote is a delivery note; it has no e-invoice representation.
	DeliveryNote string = "DELIVERY_NOTE"
	// CreditNote corrects a previous invoice (UN/CEFACT 381).
	CreditNote string = "CREDIT_NOTE"
)

var (
	// ErrInvalidDocumentType is returned for a Type outside the constants above.
	ErrInvalidDocumentType = errors.New("invalid document type")
	// ErrInvalidCurrency is returned when Currency.Code is not a three-letter
	// ISO 4217 code or Precision is outside 0..8.
	ErrInvalidCurrency = errors.New("invalid currency")
	// ErrMissingParty is returned when Company or Customer has no name.
	ErrMissingParty = errors.New("company and customer need a name")
)

var currencyCodeRe = regexp.MustCompile(`^[A-Z]{3}$`)

// Currency of the document. Precision is the number of decimals every amount
// is rounded to; it drives the Breakdown and the PDF formatting.
type Currency struct {
	Code      string `json:"code,omitempty"` // ISO 4217, e.g. "EUR"
	Precision int    `json:"precision"`      // 0..8, usually 2
}

// EUR is the default currency with two decimals.
var EUR = Currency{Code: "EUR", Precision: 2}

// Delivery describes when and where goods or services were delivered.
type Delivery struct {
	// Date is the actual delivery date (BT-72).
	Date string `json:"date,omitempty"`
	// PeriodStart and PeriodEnd define the invoicing period (BG-14).
	PeriodStart string `json:"period_start,omitempty"`
	PeriodEnd   string `json:"period_end,omitempty"`
	// ShipTo is the deliver-to party (BG-13) when it differs from the
	// customer. Only Name and Address are used.
	ShipTo *Contact `json:"ship_to,omitempty"`
}

// Payment holds the payment instructions (BG-16).
type Payment struct {
	IBAN string `json:"iban,omitempty"`
	BIC  string `json:"bic,omitempty"`
	// MeansCode is the UN/ECE 4461 payment means code, e.g. "30" credit
	// transfer, "58" SEPA credit transfer. Exporters default it when empty.
	MeansCode string `json:"means_code,omitempty"`
	// Reference is the remittance information the buyer should quote (BT-83).
	Reference string `json:"reference,omitempty"`
}

// DocumentRef points to another document, e.g. the invoice a credit note
// corrects (BG-3).
type DocumentRef struct {
	Ref  string `json:"ref,omitempty"`
	Date string `json:"date,omitempty"`
}

// Note is a free-text note (BG-1). SubjectCode is an optional UNTDID 4451
// code; French invoices need "PMD" (late payment penalties), "PMT"
// (recovery indemnity) and "AAB" (discount terms).
type Note struct {
	Content     string `json:"content,omitempty"`
	SubjectCode string `json:"subject_code,omitempty"`
}

// Document is the data of a commercial document. Dates are strings in
// "DD/MM/YYYY", "YYYY-MM-DD" or "YYYYMMDD" form; see ParseDate.
type Document struct {
	Type string `json:"type,omitempty" validate:"required,oneof=INVOICE DELIVERY_NOTE QUOTATION CREDIT_NOTE"`
	// Ref is the document number (BT-1).
	Ref     string `json:"ref,omitempty" validate:"required,min=1,max=35"`
	Version string `json:"version,omitempty" validate:"max=32"`
	// ClientRef is the buyer's own reference, such as a cost centre or
	// routing identifier (BT-10).
	ClientRef string `json:"client_ref,omitempty" validate:"max=64"`
	// PurchaseOrderRef is the buyer's purchase order number (BT-13).
	PurchaseOrderRef string `json:"purchase_order_ref,omitempty" validate:"max=64"`
	Description      string `json:"description,omitempty" validate:"max=1024"`
	Notes            []Note `json:"notes,omitempty"`

	Company  *Contact `json:"company,omitempty" validate:"required"`
	Customer *Contact `json:"customer,omitempty" validate:"required"`
	Items    []*Item  `json:"items,omitempty"`

	// Date is the issue date (BT-2).
	Date         string `json:"date,omitempty"`
	ValidityDate string `json:"validity_date,omitempty"`
	// DueDate is the payment due date (BT-9).
	DueDate string `json:"due_date,omitempty"`
	// PaymentTerm is the free-text payment terms (BT-20), e.g. "30 days".
	// When DueDate is empty and PaymentTerm is a date, it is used as due date.
	PaymentTerm string `json:"payment_term,omitempty"`

	Payment          *Payment     `json:"payment,omitempty"`
	Delivery         *Delivery    `json:"delivery,omitempty"`
	PrecedingInvoice *DocumentRef `json:"preceding_invoice,omitempty"`

	// DefaultTax applies to items that have no Tax of their own.
	DefaultTax *Tax      `json:"default_tax,omitempty"`
	Discount   *Discount `json:"discount,omitempty"`
	Currency   Currency  `json:"currency"`
}

// New returns a document of the given type in EUR with two decimals.
func New(docType string) (*Document, error) {
	switch docType {
	case Invoice, Quotation, DeliveryNote, CreditNote:
	default:
		return nil, ErrInvalidDocumentType
	}
	return &Document{Type: docType, Currency: EUR}, nil
}

// Validate checks the document and prepares all monetary values. It must run
// before Breakdown or any exporter; generator.Generator.Build calls it.
func (d *Document) Validate() error {
	if err := validator.New().Struct(d); err != nil {
		return err
	}

	if !currencyCodeRe.MatchString(d.Currency.Code) || d.Currency.Precision < 0 || d.Currency.Precision > 8 {
		return fmt.Errorf("%w: code %q precision %d", ErrInvalidCurrency, d.Currency.Code, d.Currency.Precision)
	}

	if strings.TrimSpace(d.Company.Name) == "" || strings.TrimSpace(d.Customer.Name) == "" {
		return ErrMissingParty
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

// SetType sets the document type.
func (d *Document) SetType(docType string) *Document {
	d.Type = docType
	return d
}

// SetRef sets the document number.
func (d *Document) SetRef(ref string) *Document {
	d.Ref = ref
	return d
}

// SetVersion sets the document version.
func (d *Document) SetVersion(version string) *Document {
	d.Version = version
	return d
}

// SetClientRef sets the buyer's reference (BT-10).
func (d *Document) SetClientRef(ref string) *Document {
	d.ClientRef = ref
	return d
}

// SetPurchaseOrderRef sets the purchase order number (BT-13).
func (d *Document) SetPurchaseOrderRef(ref string) *Document {
	d.PurchaseOrderRef = ref
	return d
}

// SetDescription sets the document description.
func (d *Document) SetDescription(desc string) *Document {
	d.Description = desc
	return d
}

// SetNotes replaces the notes with a single note without subject code.
// Basic HTML (bold, italic, line breaks) is rendered on the PDF.
func (d *Document) SetNotes(notes string) *Document {
	d.Notes = []Note{{Content: notes}}
	return d
}

// AddNote appends a note with an optional UNTDID 4451 subject code.
func (d *Document) AddNote(content, subjectCode string) *Document {
	d.Notes = append(d.Notes, Note{Content: content, SubjectCode: subjectCode})
	return d
}

// SetCompany sets the seller.
func (d *Document) SetCompany(company *Contact) *Document {
	d.Company = company
	return d
}

// SetCustomer sets the buyer.
func (d *Document) SetCustomer(customer *Contact) *Document {
	d.Customer = customer
	return d
}

// AppendItem appends a line.
func (d *Document) AppendItem(item *Item) *Document {
	d.Items = append(d.Items, item)
	return d
}

// SetDate sets the issue date.
func (d *Document) SetDate(date string) *Document {
	d.Date = date
	return d
}

// SetValidityDate sets the validity date of a quotation.
func (d *Document) SetValidityDate(date string) *Document {
	d.ValidityDate = date
	return d
}

// SetDueDate sets the payment due date.
func (d *Document) SetDueDate(date string) *Document {
	d.DueDate = date
	return d
}

// SetPaymentTerm sets the free-text payment terms.
func (d *Document) SetPaymentTerm(term string) *Document {
	d.PaymentTerm = term
	return d
}

// SetPayment sets the payment instructions.
func (d *Document) SetPayment(payment *Payment) *Document {
	d.Payment = payment
	return d
}

// SetDelivery sets the delivery information.
func (d *Document) SetDelivery(delivery *Delivery) *Document {
	d.Delivery = delivery
	return d
}

// SetPrecedingInvoice references the invoice this document corrects.
func (d *Document) SetPrecedingInvoice(ref, date string) *Document {
	d.PrecedingInvoice = &DocumentRef{Ref: ref, Date: date}
	return d
}

// SetDefaultTax sets the tax applied to items without an explicit tax.
func (d *Document) SetDefaultTax(tax *Tax) *Document {
	d.DefaultTax = tax
	return d
}

// SetDiscount sets the document-level discount.
func (d *Document) SetDiscount(discount *Discount) *Document {
	d.Discount = discount
	return d
}

// SetCurrency sets the currency code and precision.
func (d *Document) SetCurrency(code string, precision int) *Document {
	d.Currency = Currency{Code: strings.ToUpper(code), Precision: precision}
	return d
}

// EffectiveDueDate returns DueDate, or PaymentTerm when it parses as a date.
// The boolean reports whether a due date was found.
func (d *Document) EffectiveDueDate() (string, bool) {
	if d.DueDate != "" {
		return d.DueDate, true
	}
	if _, err := ParseDate(d.PaymentTerm); err == nil && d.PaymentTerm != "" {
		return d.PaymentTerm, true
	}
	return "", false
}

// PaymentTermsText returns PaymentTerm when it is free text rather than the
// date that EffectiveDueDate already reports.
func (d *Document) PaymentTermsText() string {
	if d.PaymentTerm == "" {
		return ""
	}
	if d.DueDate == "" {
		if _, err := ParseDate(d.PaymentTerm); err == nil {
			return ""
		}
	}
	return d.PaymentTerm
}
