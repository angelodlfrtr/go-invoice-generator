package generator

import (
	"gopkg.in/go-playground/validator.v9"
)

var validate *validator.Validate

func (d *Document) Validate() error {
	validate = validator.New()
	err := validate.Struct(d)

	return err
}
