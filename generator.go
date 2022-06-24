// Package generator allows you to easily generate invoices, delivery notes and quotations in GoLang.
package generator

import (
	"github.com/creasty/defaults"
	"github.com/go-pdf/fpdf"
	"github.com/leekchan/accounting"
)

// New return a new documents with provided types and defaults
func New(docType string, options *Options) (*Document, error) {
	_ = defaults.Set(options)

	doc := &Document{
		Options: options,
		Type:    docType,
	}

	// Prepare pdf
	doc.pdf = fpdf.New("P", "mm", "A4", "")
	doc.Options.UnicodeTranslateFunc = doc.pdf.UnicodeTranslatorFromDescriptor("")

	// Prepare accounting
	doc.ac = accounting.Accounting{
		Symbol:    doc.Options.CurrencySymbol,
		Precision: doc.Options.CurrencyPrecision,
		Thousand:  doc.Options.CurrencyThousand,
		Decimal:   doc.Options.CurrencyDecimal,
	}

	return doc, nil
}
