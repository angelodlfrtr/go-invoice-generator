package generator

const (
	// Invoice define the "invoice" document type
	Invoice string = "INVOICE"

	// Quotation define the "quotation" document type
	Quotation string = "QUOTATION"

	// DeliveryNote define the "delievry note" document type
	DeliveryNote string = "DELIVERY_NOTE"

	// BaseMargin define base margin used in documents
	BaseMargin float64 = 10

	// BaseMarginTop define base margin top used in documents
	BaseMarginTop float64 = 20

	// HeaderMarginTop define base header margin top used in documents
	HeaderMarginTop float64 = 5

	// MaxPageHeight define the maximum height for a single page
	MaxPageHeight float64 = 260
)

var (
	// BaseTextColor define the base color used for text in document
	BaseTextColor = []int{35, 35, 35}

	// GreyBgColor define the grey background color used for text in document
	GreyBgColor = []int{232, 232, 232}

	// DarkBgColor define the grey background color used for text in document
	DarkBgColor = []int{212, 212, 212}
)
