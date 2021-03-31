package main

import (
	"github.com/go-playground/validator/v10"
	"testing"
)

func TestValidation(t *testing.T) {
	s := Request{}
	validate := validator.New()
	err := validate.Struct(s)

	if err == nil {
		t.Errorf("Expected struct to be invalid but its valid")
	}
}
