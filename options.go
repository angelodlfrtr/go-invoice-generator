package generator

// Options for Document
type Options struct {
	AutoPrint bool

	CurrencySymbol    string `default:"€ "`
	CurrencyPrecision int    `default:"2"`
	CurrencyDecimal   string `default:","`
	CurrencyThousand  string `default:"."`

	TextTypeInvoice      string `default:"INVOICE"`
	TextTypeQuotation    string `default:"QUOTATION"`
	TextTypeDeliveryNote string `default:"DELIVERY NOTE"`

	TextRefTitle         string `default:"Ref."`
	TextVersionTitle     string `default:"Version"`
	TextDateTitle        string `default:"Date"`
	TextPaymentTermTitle string `default:"Payment term"`

	TextItemsDescriptionTitle string `default:"Description"`
	TextItemsUnitCostTitle    string `default:"Unit price"`
	TextItemsQuantityTitle    string `default:"Quantity"`
	TextItemsTotalHTTitle     string `default:"Total"`
	TextItemsTaxTitle         string `default:"Tax"`
	TextItemsTotalTTCTitle    string `default:"Total with tax"`

	TextTotalTotal   string `default:"TOTAL"`
	TextTotalTax     string `default:"TAX"`
	TextTotalWithTax string `default:"TOTAL WITH TAX"`
}
