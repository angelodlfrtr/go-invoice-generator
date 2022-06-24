package generator

import (
	"errors"

	"github.com/shopspring/decimal"
)

// ErrInvalidDiscount when percent and amount are empty
var ErrInvalidDiscount = errors.New("invalid discount")

// Discount types
const (
	DiscountTypeAmount  string = "amount"
	DiscountTypePercent string = "percent"
)

// Discount define discount as percent or fixed amount
type Discount struct {
	Percent string `json:"percent,omitempty"` // Discount in percent ex 17
	Amount  string `json:"amount,omitempty"`  // Discount in amount ex 123.40

	_percent decimal.Decimal
	_amount  decimal.Decimal
}

// Prepare convert strings to decimal
func (d *Discount) Prepare() error {
	if len(d.Percent) == 0 && len(d.Amount) == 0 {
		return ErrInvalidDiscount
	}

	// Percent
	if len(d.Percent) > 0 {
		percent, err := decimal.NewFromString(d.Percent)
		if err != nil {
			return err
		}
		d._percent = percent
	}

	// Amount
	if len(d.Amount) > 0 {
		amount, err := decimal.NewFromString(d.Amount)
		if err != nil {
			return err
		}
		d._amount = amount
	}

	return nil
}

// getDiscount as return the discount type and value
func (t *Discount) getDiscount() (string, decimal.Decimal) {
	tax := "0"
	taxType := DiscountTypePercent

	if len(t.Percent) > 0 {
		tax = t.Percent
	}

	if len(t.Amount) > 0 {
		tax = t.Amount
		taxType = DiscountTypeAmount
	}

	decVal, _ := decimal.NewFromString(tax)

	return taxType, decVal
}
