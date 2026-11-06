// Package generator renders an invoice.Document as a PDF with fpdf.
//
// Two entry points exist. Generator renders a document you built with the
// invoice package:
//
//	doc, _ := invoice.New(invoice.Invoice)
//	doc.SetRef("INV-1") // ...
//	pdf, err := generator.NewGenerator(&generator.Options{}).Build(doc)
//
// Document couples a document with its generator so the fluent setters and
// Build live on one value, as in earlier versions:
//
//	doc, _ := generator.New(generator.Invoice, &generator.Options{})
//	doc.SetRef("INV-1") // ...
//	pdf, err := doc.Build()
package generator

import (
	"codeberg.org/go-pdf/fpdf"
	"github.com/creasty/defaults"
	"github.com/leekchan/accounting"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

// Generator renders one document. Create a new Generator per document: the
// underlying PDF accumulates pages.
type Generator struct {
	pdf       *fpdf.Fpdf
	translate UnicodeTranslateFunc

	// Options holds labels, colours, fonts and currency formatting.
	Options *Options
	// Header and Footer are optional.
	Header *HeaderFooter
	Footer *HeaderFooter
}

// NewGenerator returns a generator with the given options; nil means defaults.
func NewGenerator(options *Options) *Generator {
	if options == nil {
		options = &Options{}
	}
	_ = defaults.Set(options)

	g := &Generator{pdf: newPDF(), Options: options}
	// UTF-8 fonts (registered by newPDF) pass strings straight through, so
	// no translation is needed unless the caller installs another font.
	g.translate = options.UnicodeTranslateFunc
	if g.translate == nil {
		g.translate = func(s string) string { return s }
	}
	options.UnicodeTranslateFunc = g.translate
	return g
}

func newPDF() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	registerDefaultFonts(pdf)
	return pdf
}

// Pdf returns the underlying fpdf instance, e.g. to register fonts.
func (g *Generator) Pdf() *fpdf.Fpdf { return g.pdf }

// SetHeader sets the page header.
func (g *Generator) SetHeader(header *HeaderFooter) *Generator {
	g.Header = header
	return g
}

// SetFooter sets the page footer.
func (g *Generator) SetFooter(footer *HeaderFooter) *Generator {
	g.Footer = footer
	return g
}

// SetUnicodeTranslator sets a custom unicode translation function.
// See https://pkg.go.dev/codeberg.org/go-pdf/fpdf#UnicodeTranslator
func (g *Generator) SetUnicodeTranslator(fn UnicodeTranslateFunc) *Generator {
	g.translate = fn
	g.Options.UnicodeTranslateFunc = fn
	return g
}

// Build validates doc (which prepares all monetary values) and renders it.
func (g *Generator) Build(doc *invoice.Document) (*fpdf.Fpdf, error) {
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	r := &renderer{
		pdf:  g.pdf,
		opts: g.Options,
		doc:  doc,
		gen:  g,
		ac: accounting.Accounting{
			Symbol:    g.Options.CurrencySymbol,
			Precision: doc.Currency.Precision,
			Thousand:  g.Options.CurrencyThousand,
			Decimal:   g.Options.CurrencyDecimal,
		},
	}
	return r.build()
}

// Document couples an invoice.Document with its Generator. The document's
// fluent setters are promoted, so existing code keeps working.
type Document struct {
	*invoice.Document

	// Options are the generator options.
	Options *Options
	gen     *Generator
}

// New returns a document of the given type (Invoice, Quotation, DeliveryNote
// or CreditNote) in EUR with two decimals, bound to a generator using options.
func New(docType string, options *Options) (*Document, error) {
	inv, err := invoice.New(docType)
	if err != nil {
		return nil, err
	}
	g := NewGenerator(options)
	return &Document{Document: inv, Options: g.Options, gen: g}, nil
}

// Generator returns the bound generator.
func (d *Document) Generator() *Generator { return d.gen }

// Pdf returns the underlying fpdf instance.
func (d *Document) Pdf() *fpdf.Fpdf { return d.gen.pdf }

// SetHeader sets the page header.
func (d *Document) SetHeader(header *HeaderFooter) *Document {
	d.gen.SetHeader(header)
	return d
}

// SetFooter sets the page footer.
func (d *Document) SetFooter(footer *HeaderFooter) *Document {
	d.gen.SetFooter(footer)
	return d
}

// SetUnicodeTranslator sets a custom unicode translation function.
func (d *Document) SetUnicodeTranslator(fn UnicodeTranslateFunc) {
	d.gen.SetUnicodeTranslator(fn)
}

// Build renders the document.
func (d *Document) Build() (*fpdf.Fpdf, error) {
	return d.gen.Build(d.Document)
}
