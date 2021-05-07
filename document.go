package generator

import "github.com/jung-kurt/gofpdf"

// Document define base document
type Document struct {
	pdf *gofpdf.Fpdf

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
