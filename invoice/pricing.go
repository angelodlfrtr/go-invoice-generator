package invoice

import (
	"errors"
	"fmt"

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

	// Category is the VAT category code (UNTDID 5305) used by e-invoicing
	// exports such as Factur-X: "S" standard, "Z" zero-rated, "E" exempt,
	// "AE" reverse charge, "K" intra-community, "G" export, "O" not subject.
	// It has no effect on the rendered PDF. Empty means the exporter default.
	Category string `json:"category,omitempty"`

	// ExemptionReason is the free-text reason required by EN 16931 when
	// Category is anything other than "S" or "Z" (e.g. "TVA non applicable,
	// art. 293 B du CGI"). ExemptionReasonCode is the matching VATEX code
	// (e.g. "VATEX-EU-G"). At least one of the two must be set in that case.
	ExemptionReason     string `json:"exemption_reason,omitempty"`
	ExemptionReasonCode string `json:"exemption_reason_code,omitempty"`

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

// Value returns the discount kind (DiscountTypeAmount or DiscountTypePercent)
// and its parsed value; Prepare must have run.
func (d *Discount) Value() (string, decimal.Decimal) { return d.getDiscount() }

// Value returns the tax kind (TaxTypeAmount or TaxTypePercent) and its parsed
// value; Prepare must have run.
func (t *Tax) Value() (string, decimal.Decimal) { return t.getTax() }

// -----------------------------------------------------------------------

// ErrInvalidCharge is returned when a Charge has no amount or a fixed-amount tax.
var ErrInvalidCharge = errors.New("invalid charge")

// Charge is a document-level charge (BG-21) such as freight or packaging,
// added to the taxable amount on top of the items. Amount excludes tax.
type Charge struct {
	Name   string `json:"name,omitempty" validate:"required"`
	Amount string `json:"amount,omitempty"` // e.g. "15.00", excluding tax
	// ReasonCode is the UNTDID 7161 charge reason, e.g. "FC" freight,
	// "PC" packing, "AEM" eco-contribution. Optional.
	ReasonCode string `json:"reason_code,omitempty"`
	// Tax is the percentage tax applied to the charge; nil means untaxed.
	Tax *Tax `json:"tax,omitempty"`

	_amount decimal.Decimal
}

// Prepare parses and validates the charge.
func (c *Charge) Prepare() error {
	amount, err := decimal.NewFromString(c.Amount)
	if err != nil {
		return fmt.Errorf("%w: amount %q: %v", ErrInvalidCharge, c.Amount, err)
	}
	c._amount = amount
	if c.Tax != nil {
		if err := c.Tax.Prepare(); err != nil {
			return err
		}
		if len(c.Tax.Amount) > 0 {
			return fmt.Errorf("%w: %q: a charge needs a percentage tax", ErrInvalidCharge, c.Name)
		}
	}
	return nil
}

// AmountDecimal returns the parsed amount; Prepare must have run.
func (c *Charge) AmountDecimal() decimal.Decimal { return c._amount }
