package errs

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Issues []ValidationIssue `json:"field"`
}

type ValidationIssue struct {
	Field string `json:"field"`
	Msg   string `json:"msg"`
}

func MapValidationError(err error) ValidationError {
	errors := err.(validator.ValidationErrors)
	result := ValidationError{
		Issues: []ValidationIssue{},
	}

	for _, fieldError := range errors {

		result.Issues = append(result.Issues, ValidationIssue{
			Field: fieldError.Field(),
			Msg:   parseValidateErrorMessage(fieldError),
		})
	}

	return result
}

func parseValidateErrorMessage(fieldError validator.FieldError) string {
	switch fieldError.ActualTag() {
	case "required":
		return fmt.Sprintf(
			"%s is required.",
			fieldError.Field(),
		)
	default:
		return fmt.Sprintf(
			"%s is '%s' but it should be an '%s'",
			fieldError.Field(),
			fieldError.Value(),
			fieldError.Tag(),
		)
	}
}
