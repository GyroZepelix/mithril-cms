package schema

import (
	"fmt"
	"regexp"
	"strings"
)

// namePattern matches valid content type and field names: lowercase letter
// followed by lowercase letters, digits, or underscores.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// sqlReservedWords is a set of SQL keywords that must not be used as content
// type names because they would collide with SQL syntax in generated DDL.
var sqlReservedWords = map[string]bool{
	"select":   true,
	"insert":   true,
	"update":   true,
	"delete":   true,
	"drop":     true,
	"table":    true,
	"create":   true,
	"alter":    true,
	"index":    true,
	"where":    true,
	"from":     true,
	"join":     true,
	"order":    true,
	"group":    true,
	"having":   true,
	"limit":    true,
	"offset":   true,
	"union":    true,
	"distinct": true,
	"and":      true,
	"or":       true,
	"not":      true,
	"null":     true,
	"true":     true,
	"false":    true,
	"in":       true,
	"between":  true,
	"like":     true,
	"is":       true,
	"exists":   true,
	"case":     true,
	"when":     true,
	"then":     true,
	"else":     true,
	"end":      true,
	"as":       true,
	"on":       true,
	"into":     true,
	"values":   true,
	"set":      true,
	"primary":  true,
	"foreign":  true,
	"key":      true,
	"check":    true,
	"default":  true,
	"grant":    true,
	"revoke":   true,
	"cascade":  true,
	"trigger":  true,
	"begin":    true,
	"commit":   true,
	"rollback": true,
}

// reservedColumnNames is the set of column names automatically added to every
// content table. User-defined fields must not use these names.
var reservedColumnNames = map[string]bool{
	"id":            true,
	"status":        true,
	"search_vector": true,
	"created_by":    true,
	"updated_by":    true,
	"created_at":    true,
	"updated_at":    true,
	"published_at":  true,
}

// textFieldTypes are the field types that support searchable, min_length, and max_length.
var textFieldTypes = map[FieldType]bool{
	FieldTypeString:   true,
	FieldTypeText:     true,
	FieldTypeRichText: true,
}

// numericFieldTypes are the field types that support min and max.
var numericFieldTypes = map[FieldType]bool{
	FieldTypeInt:   true,
	FieldTypeFloat: true,
}

// ValidateSchemas validates all schemas together, including cross-references
// between content types (e.g., relation targets). It returns a multi-error
// listing ALL validation problems found, or nil if all schemas are valid.
func ValidateSchemas(schemas []ContentType) error {
	// Build a set of known content type names for relation target validation.
	knownTypes := make(map[string]bool, len(schemas))
	for _, ct := range schemas {
		if ct.Name != "" {
			knownTypes[ct.Name] = true
		}
	}

	var allErrors []string

	// Check for duplicate content type names across files.
	nameCount := make(map[string]int, len(schemas))
	for _, ct := range schemas {
		nameCount[ct.Name]++
	}
	for name, count := range nameCount {
		if count > 1 && name != "" {
			allErrors = append(allErrors, fmt.Sprintf("content type name %q is defined %d times", name, count))
		}
	}

	for _, ct := range schemas {
		problems := validateContentType(ct, knownTypes)
		for _, msg := range problems {
			allErrors = append(allErrors, fmt.Sprintf("content type %q: %s", ct.Name, msg))
		}
	}

	if len(allErrors) == 0 {
		return nil
	}

	return &ValidationError{Problems: allErrors}
}

// ValidationError holds a list of all validation problems found across schemas.
type ValidationError struct {
	Problems []string
}

// Error returns a human-readable summary of all validation problems.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("schema validation failed with %d problem(s):\n- %s",
		len(e.Problems), strings.Join(e.Problems, "\n- "))
}

// maxContentTypeNameLength is the maximum length for a content type name.
// PostgreSQL identifiers are limited to 63 bytes, and content tables are
// named "ct_{name}", so the name itself is limited to 59 characters.
const maxContentTypeNameLength = 59

// maxFieldNameLength is the maximum length for a field name. PostgreSQL
// identifiers are limited to 63 bytes.
const maxFieldNameLength = 63

// validateContentType validates a single content type and returns a list of
// validation error messages. It receives the set of all known content type
// names for relation target validation.
func validateContentType(ct ContentType, knownTypes map[string]bool) []string {
	var problems []string

	// Validate content type name.
	if ct.Name == "" {
		problems = append(problems, "name is required")
	} else {
		if !namePattern.MatchString(ct.Name) {
			problems = append(problems, "name must match ^[a-z][a-z0-9_]*$")
		}
		if len(ct.Name) > maxContentTypeNameLength {
			problems = append(problems, fmt.Sprintf("name must be at most %d characters (got %d); PostgreSQL identifier limit is 63 and tables use \"ct_\" prefix", maxContentTypeNameLength, len(ct.Name)))
		}
		if strings.HasPrefix(ct.Name, "ct_") {
			problems = append(problems, "name must not start with \"ct_\" (reserved for table prefix)")
		}
		if sqlReservedWords[strings.ToLower(ct.Name)] {
			problems = append(problems, fmt.Sprintf("name %q is a reserved SQL keyword", ct.Name))
		}
	}

	// Validate display name.
	if ct.DisplayName == "" {
		problems = append(problems, "display_name is required")
	}

	// Validate fields.
	if len(ct.Fields) == 0 {
		problems = append(problems, "at least one field is required")
		return problems
	}

	fieldNames := make(map[string]bool, len(ct.Fields))

	for i, f := range ct.Fields {
		prefix := fmt.Sprintf("field[%d] (%s)", i, f.Name)

		// Validate field name.
		if f.Name == "" {
			problems = append(problems, fmt.Sprintf("field[%d]: name is required", i))
		} else {
			if !namePattern.MatchString(f.Name) {
				problems = append(problems, fmt.Sprintf("%s: name must match ^[a-z][a-z0-9_]*$", prefix))
			}
			if len(f.Name) > maxFieldNameLength {
				problems = append(problems, fmt.Sprintf("%s: name must be at most %d characters (got %d)", prefix, maxFieldNameLength, len(f.Name)))
			}
			if reservedColumnNames[f.Name] {
				problems = append(problems, fmt.Sprintf("%s: name %q is a reserved column name", prefix, f.Name))
			}
			if fieldNames[f.Name] {
				problems = append(problems, fmt.Sprintf("%s: duplicate field name", prefix))
			}
			fieldNames[f.Name] = true
		}

		// Validate field type.
		if !validFieldTypes[f.Type] {
			problems = append(problems, fmt.Sprintf("%s: invalid field type %q", prefix, f.Type))
			continue // Skip further checks that depend on type.
		}

		// Validate searchable: only on text-like types.
		if f.Searchable && !textFieldTypes[f.Type] {
			problems = append(problems, fmt.Sprintf("%s: searchable is only valid on string, text, richtext types", prefix))
		}

		// Validate min_length/max_length: only on text-like types.
		if f.MinLength != nil && !textFieldTypes[f.Type] {
			problems = append(problems, fmt.Sprintf("%s: min_length is only valid on string, text, richtext types", prefix))
		}
		if f.MaxLength != nil && !textFieldTypes[f.Type] {
			problems = append(problems, fmt.Sprintf("%s: max_length is only valid on string, text, richtext types", prefix))
		}

		// Validate min_length/max_length are non-negative.
		if f.MinLength != nil && *f.MinLength < 0 {
			problems = append(problems, fmt.Sprintf("%s: min_length must be >= 0 (got %d)", prefix, *f.MinLength))
		}
		if f.MaxLength != nil && *f.MaxLength <= 0 {
			problems = append(problems, fmt.Sprintf("%s: max_length must be > 0 (got %d)", prefix, *f.MaxLength))
		}
		if f.MinLength != nil && f.MaxLength != nil && *f.MinLength > *f.MaxLength {
			problems = append(problems, fmt.Sprintf("%s: min_length (%d) must be <= max_length (%d)", prefix, *f.MinLength, *f.MaxLength))
		}

		// Validate min/max: only on numeric types.
		if f.Min != nil && !numericFieldTypes[f.Type] {
			problems = append(problems, fmt.Sprintf("%s: min is only valid on int, float types", prefix))
		}
		if f.Max != nil && !numericFieldTypes[f.Type] {
			problems = append(problems, fmt.Sprintf("%s: max is only valid on int, float types", prefix))
		}
		if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
			problems = append(problems, fmt.Sprintf("%s: min (%g) must be <= max (%g)", prefix, *f.Min, *f.Max))
		}

		// Validate regex: only on string type.
		if f.Regex != "" {
			if f.Type != FieldTypeString {
				problems = append(problems, fmt.Sprintf("%s: regex is only valid on string type", prefix))
			} else {
				if _, err := regexp.Compile(f.Regex); err != nil {
					problems = append(problems, fmt.Sprintf("%s: invalid regex %q: %v", prefix, f.Regex, err))
				}
			}
		}

		// Validate values: only valid on enum type.
		if len(f.Values) > 0 && f.Type != FieldTypeEnum {
			problems = append(problems, fmt.Sprintf("%s: values is only valid on enum type", prefix))
		}

		// Validate relates_to and relation_type: only valid on relation type.
		if f.RelatesTo != "" && f.Type != FieldTypeRelation {
			problems = append(problems, fmt.Sprintf("%s: relates_to is only valid on relation type", prefix))
		}
		if f.RelationType != "" && f.Type != FieldTypeRelation {
			problems = append(problems, fmt.Sprintf("%s: relation_type is only valid on relation type", prefix))
		}

		// Validate enum fields: must have non-empty values list.
		if f.Type == FieldTypeEnum {
			if len(f.Values) == 0 {
				problems = append(problems, fmt.Sprintf("%s: enum field must have a non-empty values list", prefix))
			} else {
				seen := make(map[string]bool, len(f.Values))
				for j, v := range f.Values {
					if v == "" {
						problems = append(problems, fmt.Sprintf("%s: values[%d] must not be empty", prefix, j))
					} else if seen[v] {
						problems = append(problems, fmt.Sprintf("%s: duplicate enum value %q", prefix, v))
					}
					seen[v] = true
				}
			}
		}

		// Validate relation fields: must have relates_to and valid relation_type.
		if f.Type == FieldTypeRelation {
			if f.RelatesTo == "" {
				problems = append(problems, fmt.Sprintf("%s: relation field must have relates_to", prefix))
			} else if !knownTypes[f.RelatesTo] {
				problems = append(problems, fmt.Sprintf("%s: relates_to references unknown content type %q", prefix, f.RelatesTo))
			}
			if f.RelationType != RelationOne && f.RelationType != RelationMany {
				problems = append(problems, fmt.Sprintf("%s: relation field must have relation_type of \"one\" or \"many\", got %q", prefix, f.RelationType))
			}
		}
	}

	return problems
}
