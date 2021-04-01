package main

import (
	"fmt"
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

func TestSubstitution(t *testing.T) {
	z := "<img src=\"%s\" />"
	z = fmt.Sprintf(z, "asd")
	if z != "<img src=\"asd\" />" {
		t.Errorf("Got: %s", z)
	}
}
