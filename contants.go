package generator

// Invoice define the "invoice" document type
const Invoice string = "INVOICE"

// Quotation define the "quotation" document type
const Quotation string = "QUOTATION"

// DeliveryNote define the "delievry note" document type
const DeliveryNote string = "DELIVERY_NOTE"

// BaseMargin define base margin used in documents
const BaseMargin float64 = 10

// BaseMarginTop define base margin top used in documents
const BaseMarginTop float64 = 20

// HeaderMarginTop define base header margin top used in documents
const HeaderMarginTop float64 = 5

// BaseTextColor define the base color used for text in document
var BaseTextColor = []int{35, 35, 35}

// GreyBgColor define the grey background color used for text in document
var GreyBgColor = []int{232, 232, 232}

// DarkBgColor define the grey background color used for text in document
var DarkBgColor = []int{212, 212, 212}

// MaxPageHeight define the maximum height for a single page
const MaxPageHeight float64 = 260
