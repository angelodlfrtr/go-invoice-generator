package generator

import "github.com/angelodlfrtr/go-invoice-generator/invoice"

// The model lives in the invoice package. These aliases keep code written
// against earlier versions compiling; prefer the invoice package in new code.

type (
	// Contact is an alias of invoice.Contact.
	Contact = invoice.Contact
	// Address is an alias of invoice.Address.
	Address = invoice.Address
	// Item is an alias of invoice.Item.
	Item = invoice.Item
	// Tax is an alias of invoice.Tax.
	Tax = invoice.Tax
	// Discount is an alias of invoice.Discount.
	Discount = invoice.Discount
	// Note is an alias of invoice.Note.
	Note = invoice.Note
	// Payment is an alias of invoice.Payment.
	Payment = invoice.Payment
	// Delivery is an alias of invoice.Delivery.
	Delivery = invoice.Delivery
	// Currency is an alias of invoice.Currency.
	Currency = invoice.Currency
	// TaxLine is an alias of invoice.TaxLine.
	TaxLine = invoice.TaxLine
	// TaxGroup is an alias of invoice.TaxGroup.
	TaxGroup = invoice.TaxGroup
	// Breakdown is an alias of invoice.Breakdown.
	Breakdown = invoice.Breakdown
)

// Document types, see the invoice package.
const (
	Invoice      = invoice.Invoice
	Quotation    = invoice.Quotation
	DeliveryNote = invoice.DeliveryNote
	CreditNote   = invoice.CreditNote
)

// Tax and discount kinds, see the invoice package.
const (
	DiscountTypeAmount  = invoice.DiscountTypeAmount
	DiscountTypePercent = invoice.DiscountTypePercent
	TaxTypeAmount       = invoice.TaxTypeAmount
	TaxTypePercent      = invoice.TaxTypePercent
)

// Errors, see the invoice package.
var (
	ErrInvalidDocumentType = invoice.ErrInvalidDocumentType
	ErrInvalidTax          = invoice.ErrInvalidTax
	ErrInvalidDiscount     = invoice.ErrInvalidDiscount
)
