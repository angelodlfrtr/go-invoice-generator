// Package generator allows you to easily generate invoices, delivery notes and quotations in GoLang.
package generator

import (
	"github.com/creasty/defaults"
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

	return doc, nil
}
