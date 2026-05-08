package generator

import (
	"errors"

	"github.com/shopspring/decimal"
)

// ErrInvalidDiscount is returned when a Discount has neither or both fields set
var ErrInvalidDiscount = errors.New("invalid discount")

// Discount types
const (
	DiscountTypeAmount  string = "amount"
	DiscountTypePercent string = "percent"
)

// Discount defines a discount as either a percentage or a fixed amount (mutually exclusive)
type Discount struct {
	Percent string `json:"percent,omitempty"` // e.g. "17"
	Amount  string `json:"amount,omitempty"`  // e.g. "123.40"

	_percent decimal.Decimal
	_amount  decimal.Decimal
}

// Prepare parses and validates the discount fields
func (d *Discount) Prepare() error {
	if len(d.Percent) == 0 && len(d.Amount) == 0 {
		return ErrInvalidDiscount
	}
	if len(d.Percent) > 0 && len(d.Amount) > 0 {
		return ErrInvalidDiscount
	}

	if len(d.Percent) > 0 {
		percent, err := decimal.NewFromString(d.Percent)
		if err != nil {
			return err
		}
		d._percent = percent
	}

	if len(d.Amount) > 0 {
		amount, err := decimal.NewFromString(d.Amount)
		if err != nil {
			return err
		}
		d._amount = amount
	}

	return nil
}

func (d *Discount) getDiscount() (string, decimal.Decimal) {
	if len(d.Amount) > 0 {
		return DiscountTypeAmount, d._amount
	}
	return DiscountTypePercent, d._percent
}

// -----------------------------------------------------------------------

// ErrInvalidTax is returned when a Tax has neither or both fields set
var ErrInvalidTax = errors.New("invalid tax")

// Tax types
const (
	TaxTypeAmount  string = "amount"
	TaxTypePercent string = "percent"
)

// Tax defines a tax as either a percentage or a fixed amount (mutually exclusive)
type Tax struct {
	Name    string `json:"name,omitempty"`    // e.g. "TVA"
	Percent string `json:"percent,omitempty"` // e.g. "20"
	Amount  string `json:"amount,omitempty"`  // e.g. "89"

	_percent decimal.Decimal
	_amount  decimal.Decimal
}

// Prepare parses and validates the tax fields
func (t *Tax) Prepare() error {
	if len(t.Percent) == 0 && len(t.Amount) == 0 {
		return ErrInvalidTax
	}
	if len(t.Percent) > 0 && len(t.Amount) > 0 {
		return ErrInvalidTax
	}

	if len(t.Percent) > 0 {
		percent, err := decimal.NewFromString(t.Percent)
		if err != nil {
			return err
		}
		t._percent = percent
	}

	if len(t.Amount) > 0 {
		amount, err := decimal.NewFromString(t.Amount)
		if err != nil {
			return err
		}
		t._amount = amount
	}

	return nil
}

func (t *Tax) getTax() (string, decimal.Decimal) {
	if len(t.Amount) > 0 {
		return TaxTypeAmount, t._amount
	}
	return TaxTypePercent, t._percent
}
