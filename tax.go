package generator

import (
	"errors"

	"github.com/shopspring/decimal"
)

// ErrInvalidTax when percent and amount are empty
var ErrInvalidTax = errors.New("invalid tax")

// Tax types
const (
	TaxTypeAmount  string = "amount"
	TaxTypePercent string = "percent"
)

// Tax define tax as percent or fixed amount
type Tax struct {
	Percent string `json:"percent,omitempty"` // Tax in percent ex 17
	Amount  string `json:"amount,omitempty"`  // Tax in amount ex 123.40

	_percent decimal.Decimal
	_amount  decimal.Decimal
}

// Prepare convert strings to decimal
func (t *Tax) Prepare() error {
	if len(t.Percent) == 0 && len(t.Amount) == 0 {
		return ErrInvalidTax
	}

	// Percent
	if len(t.Percent) > 0 {
		percent, err := decimal.NewFromString(t.Percent)
		if err != nil {
			return err
		}
		t._percent = percent
	}

	// Amount
	if len(t.Amount) > 0 {
		amount, err := decimal.NewFromString(t.Amount)
		if err != nil {
			return err
		}
		t._amount = amount
	}

	return nil
}

// getTax return the tax type and value
func (t *Tax) getTax() (string, decimal.Decimal) {
	tax := "0"
	taxType := TaxTypePercent

	if len(t.Percent) > 0 {
		tax = t.Percent
	}

	if len(t.Amount) > 0 {
		tax = t.Amount
		taxType = TaxTypeAmount
	}

	decVal, _ := decimal.NewFromString(tax)

	return taxType, decVal
}
