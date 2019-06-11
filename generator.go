package generator

import (
	"github.com/creasty/defaults"
)

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
