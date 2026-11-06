package generator

// UnicodeTranslateFunc converts a string before it is written to the PDF.
type UnicodeTranslateFunc func(string) string

// Options controls the rendering: currency formatting, labels, colours and
// fonts. The currency precision comes from invoice.Document.Currency.
type Options struct {
	CurrencySymbol   string `default:"€ " json:"currency_symbol,omitempty"`
	CurrencyDecimal  string `default:"." json:"currency_decimal,omitempty"`
	CurrencyThousand string `default:" " json:"currency_thousand,omitempty"`

	TextTypeInvoice      string `default:"INVOICE" json:"text_type_invoice,omitempty"`
	TextTypeQuotation    string `default:"QUOTATION" json:"text_type_quotation,omitempty"`
	TextTypeDeliveryNote string `default:"DELIVERY NOTE" json:"text_type_delivery_note,omitempty"`
	TextTypeCreditNote   string `default:"CREDIT NOTE" json:"text_type_credit_note,omitempty"`

	TextRefTitle              string `default:"Ref." json:"text_ref_title,omitempty"`
	TextVersionTitle          string `default:"Version" json:"text_version_title,omitempty"`
	TextDateTitle             string `default:"Date" json:"text_date_title,omitempty"`
	TextDueDateTitle          string `default:"Due date" json:"text_due_date_title,omitempty"`
	TextPaymentTermTitle      string `default:"Payment term" json:"text_payment_term_title,omitempty"`
	TextPaymentReferenceTitle string `default:"Payment ref." json:"text_payment_reference_title,omitempty"`
	TextPrecedingInvoiceTitle string `default:"Refers to invoice" json:"text_preceding_invoice_title,omitempty"`
	TextDeliveryDateTitle     string `default:"Delivery date" json:"text_delivery_date_title,omitempty"`
	TextDeliveryPeriodTitle   string `default:"Period" json:"text_delivery_period_title,omitempty"`
	TextDeliveredToTitle      string `default:"Delivered to" json:"text_delivered_to_title,omitempty"`

	// Labels of the identifiers printed under each party's address.
	TextContactVATID   string `default:"VAT" json:"text_contact_vat_id,omitempty"`
	TextContactLegalID string `default:"Reg. no." json:"text_contact_legal_id,omitempty"`

	TextItemsNameTitle     string `default:"Name" json:"text_items_name_title,omitempty"`
	TextItemsUnitCostTitle string `default:"Unit price" json:"text_items_unit_cost_title,omitempty"`
	TextItemsQuantityTitle string `default:"Qty" json:"text_items_quantity_title,omitempty"`
	TextItemsTotalHTTitle  string `default:"Total no tax" json:"text_items_total_ht_title,omitempty"`
	TextItemsTaxTitle      string `default:"Tax" json:"text_items_tax_title,omitempty"`
	TextItemsDiscountTitle string `default:"Discount" json:"text_items_discount_title,omitempty"`
	TextItemsTotalTTCTitle string `default:"Total" json:"text_items_total_ttc_title,omitempty"`

	TextTotalTotal      string `default:"Total" json:"text_total_total,omitempty"`
	TextTotalDiscounted string `default:"Total discounted" json:"text_total_discounted,omitempty"`
	TextTotalTax        string `default:"Tax" json:"text_total_tax,omitempty"`
	TextTotalTaxOther   string `default:"Other" json:"text_total_tax_other,omitempty"`
	TextTotalWithTax    string `default:"Total with tax" json:"text_total_with_tax,omitempty"`

	BaseTextColor []int `default:"[35,35,35]" json:"base_text_color,omitempty"`
	GreyTextColor []int `default:"[82,82,82]" json:"grey_text_color,omitempty"`
	GreyBgColor   []int `default:"[232,232,232]" json:"grey_bg_color,omitempty"`
	DarkBgColor   []int `default:"[212,212,212]" json:"dark_bg_color,omitempty"`

	Font     string `default:"Roboto"`
	BoldFont string `default:"Roboto"`

	UnicodeTranslateFunc UnicodeTranslateFunc
}
