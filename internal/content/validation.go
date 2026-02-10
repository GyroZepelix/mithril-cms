package content

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// uuidRegex matches a standard UUID format.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidUUID reports whether s is a valid UUID string.
func isValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// ValidateEntry validates content data against the content type schema.
// On create (isUpdate=false), required fields must be present. On update,
// missing fields are skipped. Returns all validation errors, not just the first.
func ValidateEntry(ct schema.ContentType, data map[string]any, isUpdate bool) []server.FieldError {
	var errs []server.FieldError

	// Build field lookup for unknown-field detection.
	fieldMap := make(map[string]schema.Field, len(ct.Fields))
	for _, f := range ct.Fields {
		fieldMap[f.Name] = f
	}

	// Reject unknown fields to prevent SQL injection via field names.
	for key := range data {
		if _, ok := fieldMap[key]; !ok {
			errs = append(errs, server.FieldError{
				Field:   key,
				Message: "unknown field",
			})
		}
	}

	// Validate each schema field.
	for _, f := range ct.Fields {
		// Skip many-to-many relations (handled separately, not direct columns).
		if f.Type == schema.FieldTypeRelation && f.RelationType == schema.RelationMany {
			continue
		}

		val, present := data[f.Name]

		// Required check (create only).
		if !isUpdate && f.Required && (!present || val == nil) {
			errs = append(errs, server.FieldError{
				Field:   f.Name,
				Message: "is required",
			})
			continue
		}

		// If the field is not present or nil, skip further validation.
		if !present || val == nil {
			continue
		}

		errs = append(errs, validateFieldValue(f, val)...)
	}

	return errs
}

// validateFieldValue validates a single field value against its schema definition.
func validateFieldValue(f schema.Field, val any) []server.FieldError {
	var errs []server.FieldError

	switch f.Type {
	case schema.FieldTypeString, schema.FieldTypeText, schema.FieldTypeRichText:
		s, ok := val.(string)
		if !ok {
			return []server.FieldError{{Field: f.Name, Message: "must be a string"}}
		}
		errs = append(errs, validateStringConstraints(f, s)...)

	case schema.FieldTypeInt:
		n, ok := toFloat64(val)
		if !ok {
			return []server.FieldError{{Field: f.Name, Message: "must be a number"}}
		}
		// Check it's a whole number.
		if n != math.Trunc(n) {
			return []server.FieldError{{Field: f.Name, Message: "must be an integer"}}
		}
		errs = append(errs, validateNumericConstraints(f, n)...)

	case schema.FieldTypeFloat:
		n, ok := toFloat64(val)
		if !ok {
			return []server.FieldError{{Field: f.Name, Message: "must be a number"}}
		}
		errs = append(errs, validateNumericConstraints(f, n)...)

	case schema.FieldTypeBoolean:
		if _, ok := val.(bool); !ok {
			errs = append(errs, server.FieldError{Field: f.Name, Message: "must be a boolean"})
		}

	case schema.FieldTypeDate:
		s, ok := val.(string)
		if !ok {
			return []server.FieldError{{Field: f.Name, Message: "must be a string"}}
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			errs = append(errs, server.FieldError{Field: f.Name, Message: "must be a valid date (YYYY-MM-DD)"})
		}

	case schema.FieldTypeTime:
		s, ok := val.(string)
		if !ok {
			return []server.FieldError{{Field: f.Name, Message: "must be a string"}}
		}
		if _, err := time.Parse("15:04:05", s); err != nil {
			if _, err2 := time.Parse("15:04", s); err2 != nil {
				errs = append(errs, server.FieldError{Field: f.Name, Message: "must be a valid time (HH:MM or HH:MM:SS)"})
			}
		}

	case schema.FieldTypeEnum:
		s, ok := val.(string)
		if !ok {
			return []server.FieldError{{Field: f.Name, Message: "must be a string"}}
		}
		valid := false
		for _, v := range f.Values {
			if s == v {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, server.FieldError{
				Field:   f.Name,
				Message: fmt.Sprintf("must be one of: %s", joinValues(f.Values)),
			})
		}

	case schema.FieldTypeJSON:
		// Any valid JSON value is acceptable; it already parsed from JSON input.

	case schema.FieldTypeMedia:
		s, ok := val.(string)
		if !ok {
			return []server.FieldError{{Field: f.Name, Message: "must be a string (UUID)"}}
		}
		if !uuidRegex.MatchString(s) {
			errs = append(errs, server.FieldError{Field: f.Name, Message: "must be a valid UUID"})
		}

	case schema.FieldTypeRelation:
		if f.RelationType == schema.RelationOne {
			s, ok := val.(string)
			if !ok {
				return []server.FieldError{{Field: f.Name, Message: "must be a string (UUID)"}}
			}
			if !uuidRegex.MatchString(s) {
				errs = append(errs, server.FieldError{Field: f.Name, Message: "must be a valid UUID"})
			}
		}
	}

	return errs
}

// validateStringConstraints checks min_length, max_length, and regex on a string value.
func validateStringConstraints(f schema.Field, s string) []server.FieldError {
	var errs []server.FieldError
	runeCount := utf8.RuneCountInString(s)

	if f.MinLength != nil && runeCount < *f.MinLength {
		errs = append(errs, server.FieldError{
			Field:   f.Name,
			Message: fmt.Sprintf("must be at least %d characters", *f.MinLength),
		})
	}
	if f.MaxLength != nil && runeCount > *f.MaxLength {
		errs = append(errs, server.FieldError{
			Field:   f.Name,
			Message: fmt.Sprintf("must be at most %d characters", *f.MaxLength),
		})
	}
	if f.Regex != "" {
		re, err := regexp.Compile(f.Regex)
		if err == nil && !re.MatchString(s) {
			errs = append(errs, server.FieldError{
				Field:   f.Name,
				Message: fmt.Sprintf("must match pattern %s", f.Regex),
			})
		}
	}
	return errs
}

// validateNumericConstraints checks min and max on a numeric value.
func validateNumericConstraints(f schema.Field, n float64) []server.FieldError {
	var errs []server.FieldError
	if f.Min != nil && n < *f.Min {
		errs = append(errs, server.FieldError{
			Field:   f.Name,
			Message: fmt.Sprintf("must be at least %g", *f.Min),
		})
	}
	if f.Max != nil && n > *f.Max {
		errs = append(errs, server.FieldError{
			Field:   f.Name,
			Message: fmt.Sprintf("must be at most %g", *f.Max),
		})
	}
	return errs
}

// toFloat64 converts a value to float64, handling JSON number types.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// joinValues joins string values with ", " for error messages.
func joinValues(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = v
	}
	return strings.Join(quoted, ", ")
}
