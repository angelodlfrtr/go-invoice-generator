package generator

import (
	"github.com/shopspring/decimal"
)

// Tax define tax as percent or fixed amount
type Tax struct {
	Percent string `json:"percent,omitempty"` // Tax in percent ex 17
	Amount  string `json:"amount,omitempty"`  // Tax in amount ex 123.40
}

// getTax return the tax type and value
func (t *Tax) getTax() (string, decimal.Decimal) {
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
