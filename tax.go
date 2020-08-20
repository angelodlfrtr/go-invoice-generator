package generator

import (
	"github.com/shopspring/decimal"
)

// Tax define tax as percent or fixed amount
type Tax struct {
	Percent string // Tax in percent ex 17
	Amount  string // Tax in amount ex 123.40
}

func (t *Tax) getTax() (string, decimal.Decimal) {
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
