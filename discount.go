package generator

import (
	"github.com/shopspring/decimal"
)

// Discount define discount as percent or fixed amount
type Discount struct {
	Percent string `json:"percent,omitempty"` // Discount in percent ex 17
	Amount  string `json:"amount,omitempty"`  // Discount in amount ex 123.40
}

// getDiscount as return the discount type and value
func (t *Discount) getDiscount() (string, decimal.Decimal) {
	tax := "0"
	taxType := "percent"

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
