package generator

import (
	"gopkg.in/go-playground/validator.v9"
)

// Validate document fields
func (d *Document) Validate() error {
	validate := validator.New()
	return validate.Struct(d)
}
