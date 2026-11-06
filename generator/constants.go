package generator

// Layout constants, in millimetres on an A4 portrait page.
const (
	// BaseMargin is the left and right margin.
	BaseMargin float64 = 10

	// BaseMarginTop is the top margin.
	BaseMarginTop float64 = 20

	// HeaderMarginTop is the top margin of the page header.
	HeaderMarginTop float64 = 5

	// MaxPageHeight is the Y position past which content moves to a new page.
	MaxPageHeight float64 = 260
)

// Item table column offsets.
const (
	ItemColNameOffset      float64 = 10
	ItemColUnitPriceOffset float64 = 80
	ItemColQuantityOffset  float64 = 103
	ItemColTotalHTOffset   float64 = 113
	ItemColDiscountOffset  float64 = 140
	ItemColTaxOffset       float64 = 157
	ItemColTotalTTCOffset  float64 = 175
)

// Font sizes.
var (
	BaseTextFontSize       float64 = 8
	SmallTextFontSize      float64 = 7
	ExtraSmallTextFontSize float64 = 6
	LargeTextFontSize      float64 = 10
)
