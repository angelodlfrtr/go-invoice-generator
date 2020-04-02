package generator

import (
	"github.com/jung-kurt/gofpdf"
)

func encodeString(str string) string {
	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	return tr(str)
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
