package generator

import (
	"github.com/shopspring/decimal"
)

// Discount define discount as percent or fixed amount
type Discount struct {
	Percent string // Discount in percent ex 17
	Amount  string // Discount in amount ex 123.40
}

func (t *Tax) getDiscount() (string, decimal.Decimal) {
	var tax string
	var taxType string = "percent"

	if len(t.Percent) > 0 {
		tax = t.Percent
	}

	if len(t.Amount) > 0 {
		tax = t.Amount
		taxType = "amount"
	}

	decVal, _ := decimal.NewFromString(tax)

	return taxType, decVal
}
