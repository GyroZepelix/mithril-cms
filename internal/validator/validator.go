package validator

import (
	"fmt"
	"reflect"
)

const validatorTag string = "validate"

type ErrValidateError struct {
	ValidationIssues []validateIssue `json:"validation_issues"`
}

func (e ErrValidateError) Error() string {
	return fmt.Sprintf("Validation result: %+v", e.ValidationIssues)
}

type validateIssue struct {
	FieldName string `json:"field_name"`
	Msg       string `json:"msg"`
}

func Validate(v any) error {
	reflectionType := reflect.TypeOf(v)
	reflectionValue := reflect.ValueOf(v)
	issues := []validateIssue{}
	examiner(reflectionType, reflectionValue, 0, &issues)

	if len(issues) != 0 {
		return ErrValidateError{
			ValidationIssues: issues,
		}
	}

	return nil
}

func examiner(t reflect.Type, v reflect.Value, depth int, issues *[]validateIssue) {
	switch t.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
		if t.Kind() == reflect.Ptr && !v.IsNil() {
			t = t.Elem()
			v = v.Elem()
		}
		examiner(t.Elem(), v, depth+1, issues)
	case reflect.Struct:
		examinerStruct(t, v, issues)
	}
}

func examinerStruct(t reflect.Type, v reflect.Value, issues *[]validateIssue) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if tag := field.Tag.Get(validatorTag); tag != "" {
			var fieldName string
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				fieldName = jsonTag
			} else {
				fieldName = field.Name
			}

			if tag == "required" && isEmpty(fieldValue) {
				*issues = append(*issues, validateIssue{
					FieldName: fieldName,
					Msg:       fmt.Sprintf("Field '%s' is required but empty", fieldName),
				})
			}

		}
	}
}

func isEmpty(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return value.IsNil()
	default:
		return false
	}
}
