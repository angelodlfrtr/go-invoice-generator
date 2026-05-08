package generator

import (
	"github.com/shopspring/decimal"
)

// Item represents a product or service line on a document
type Item struct {
	Name        string    `json:"name,omitempty" validate:"required"`
	Description string    `json:"description,omitempty"`
	UnitCost    string    `json:"unit_cost,omitempty"`
	Quantity    string    `json:"quantity,omitempty"`
	Tax         *Tax      `json:"tax,omitempty"`
	Discount    *Discount `json:"discount,omitempty"`

	_unitCost decimal.Decimal
	_quantity decimal.Decimal
}

// Prepare parses UnitCost and Quantity strings into decimal values
func (i *Item) Prepare() error {
	unitCost, err := decimal.NewFromString(i.UnitCost)
	if err != nil {
		return err
	}
	i._unitCost = unitCost

	quantity, err := decimal.NewFromString(i.Quantity)
	if err != nil {
		return err
	}
	i._quantity = quantity

	if i.Tax != nil {
		if err := i.Tax.Prepare(); err != nil {
			return err
		}
	}

	if i.Discount != nil {
		if err := i.Discount.Prepare(); err != nil {
			return err
		}
	}

	return nil
}

// TotalWithoutTaxAndWithoutDiscount returns unit cost × quantity
func (i *Item) TotalWithoutTaxAndWithoutDiscount() decimal.Decimal {
	return i._unitCost.Mul(i._quantity)
}

// TotalWithoutTaxAndWithDiscount returns the subtotal after applying the item discount
func (i *Item) TotalWithoutTaxAndWithDiscount() decimal.Decimal {
	total := i.TotalWithoutTaxAndWithoutDiscount()

	if i.Discount != nil {
		dType, dNum := i.Discount.getDiscount()
		if dType == DiscountTypeAmount {
			total = total.Sub(dNum)
		} else {
			total = total.Sub(total.Mul(dNum.Div(decimal.NewFromFloat(100))))
		}
	}

	return total
}

// TaxWithTotalDiscounted returns the tax amount computed on the discounted subtotal
func (i *Item) TaxWithTotalDiscounted() decimal.Decimal {
	if i.Tax == nil {
		return decimal.NewFromFloat(0)
	}

	totalHT := i.TotalWithoutTaxAndWithDiscount()
	taxType, taxAmount := i.Tax.getTax()

	if taxType == TaxTypeAmount {
		return taxAmount
	}
	return totalHT.Mul(taxAmount.Div(decimal.NewFromFloat(100)))
}

// TotalWithTaxAndDiscount returns the final line total including tax and discount
func (i *Item) TotalWithTaxAndDiscount() decimal.Decimal {
	return i.TotalWithoutTaxAndWithDiscount().Add(i.TaxWithTotalDiscounted())
}
