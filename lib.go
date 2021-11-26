package generator

import "github.com/jung-kurt/gofpdf"

var trFunc func(string) string

func encodeString(str string) string {
	if trFunc == nil {
		pdf := gofpdf.New("P", "mm", "A4", "")
		trFunc = pdf.UnicodeTranslatorFromDescriptor("")
	}

	return trFunc(str)
}

func (d *Document) typeAsString() string {
	if d.Type == Invoice {
		return d.Options.TextTypeInvoice
	}

	if d.Type == InvoiceMonthly {
		return d.Options.TextTypeInvoiceMonthly
	}

	if d.Type == Quotation {
		return d.Options.TextTypeQuotation
	}

	return d.Options.TextTypeDeliveryNote
}
