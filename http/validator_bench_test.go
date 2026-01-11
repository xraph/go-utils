package http

import (
	"reflect"
	"testing"

	"github.com/xraph/go-utils/val"
)

// Benchmark struct with various validation rules.
type BenchmarkValidationStruct struct {
	// Using go-playground/validator tags
	Name  string `json:"name"  validate:"required,min=3,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"required,gte=0,lte=120"`

	// Using our custom tags (still supported)
	Code        string `json:"code"        optional:"true" pattern:"^[A-Z]{3}$"`
	Description string `json:"description" maxLength:"500" minLength:"10"`
}

// Legacy validation struct (before go-playground/validator).
type LegacyValidationStruct struct {
	Name        string `json:"name"        maxLength:"50"  minLength:"3"        required:"true"`
	Email       string `format:"email"     json:"email"    required:"true"`
	Age         int    `json:"age"         maximum:"120"   minimum:"0"          required:"true"`
	Code        string `json:"code"        optional:"true" pattern:"^[A-Z]{3}$"`
	Description string `json:"description" maxLength:"500" minLength:"10"`
}

func BenchmarkValidation_WithPlaygroundValidator(b *testing.B) {
	ctx := &Ctx{}
	testData := BenchmarkValidationStruct{
		Name:        "John Doe",
		Email:       "john@example.com",
		Age:         30,
		Code:        "ABC",
		Description: "This is a test description with enough characters",
	}

	errors := &val.ValidationError{}

	for b.Loop() {
		errors.Errors = nil // Reset errors
		_ = ctx.validateStruct(&testData, reflect.TypeFor[BenchmarkValidationStruct](), errors)
	}
}

func BenchmarkValidation_CustomTagsOnly(b *testing.B) {
	ctx := &Ctx{}
	testData := LegacyValidationStruct{
		Name:        "John Doe",
		Email:       "john@example.com",
		Age:         30,
		Code:        "ABC",
		Description: "This is a test description with enough characters",
	}

	errors := &val.ValidationError{}

	for b.Loop() {
		errors.Errors = nil // Reset errors
		_ = ctx.validateStruct(&testData, reflect.TypeFor[LegacyValidationStruct](), errors)
	}
}

func BenchmarkValidation_ComplexStruct(b *testing.B) {
	ctx := &Ctx{}

	type ComplexStruct struct {
		Field1  string `json:"field1"  validate:"required,min=1,max=100"`
		Field2  string `json:"field2"  validate:"required,email"`
		Field3  int    `json:"field3"  validate:"required,gte=0,lte=1000"`
		Field4  string `json:"field4"  validate:"required,url"`
		Field5  string `json:"field5"  validate:"required,uuid"`
		Field6  int    `json:"field6"  validate:"required,oneof=1 2 3 4 5"`
		Field7  string `json:"field7"  maxLength:"50"                      minLength:"5"`
		Field8  int    `json:"field8"  maximum:"100"                       minimum:"10"`
		Field9  string `json:"field9"  optional:"true"                     pattern:"^[A-Z]+$"`
		Field10 string `format:"email" json:"field10"`
	}

	testData := ComplexStruct{
		Field1:  "test value",
		Field2:  "test@example.com",
		Field3:  500,
		Field4:  "https://example.com",
		Field5:  "550e8400-e29b-41d4-a716-446655440000",
		Field6:  3,
		Field7:  "test value",
		Field8:  50,
		Field9:  "TESTCODE",
		Field10: "another@example.com",
	}

	errors := &val.ValidationError{}

	for b.Loop() {
		errors.Errors = nil // Reset errors
		_ = ctx.validateStruct(&testData, reflect.TypeFor[ComplexStruct](), errors)
	}
}
