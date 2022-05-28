// Package generator allows you to easily generate invoices, delivery notes and quotations in GoLang.
package generator

import (
	"github.com/creasty/defaults"
	"github.com/jung-kurt/gofpdf"
)

// New return a new documents with provided types and defaults
func New(docType string, options *Options) (*Document, error) {
	_ = defaults.Set(options)

	doc := &Document{
		Options: options,
		Type:    docType,
	}

	doc.pdf = gofpdf.New("P", "mm", "A4", "")
	doc.Options.UnicodeTranslateFunc = doc.pdf.UnicodeTranslatorFromDescriptor("")

	return doc, nil
}
