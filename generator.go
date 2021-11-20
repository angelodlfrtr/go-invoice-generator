// Package generator allows you to easily generate invoices, delivery notes and quotations in GoLang.
package generator

import (
	"github.com/creasty/defaults"
	"github.com/jung-kurt/gofpdf"
)

// New return a new documents with provided types and defaults
func New(docType string, options *Options) (*Document, error) {
	if err := defaults.Set(options); err != nil {
		return nil, err
	}

	doc := &Document{
		Options: options,
		Type:    docType,
	}

	doc.pdf = gofpdf.New("P", "mm", "A4", "")

	// UTF-8 fonts has to be added manually and .ttf file must exist
	doc.pdf.AddUTF8Font("dejavu", "", "fonts/DejaVuSansCondensed.ttf")

	return doc, nil
}
