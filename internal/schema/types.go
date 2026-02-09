// Package schema handles loading, parsing, and validating YAML content type
// definitions for the Mithril CMS.
package schema

// FieldType represents the type of a content field.
type FieldType string

// Supported field types for content type schemas.
const (
	FieldTypeString   FieldType = "string"
	FieldTypeText     FieldType = "text"
	FieldTypeRichText FieldType = "richtext"
	FieldTypeInt      FieldType = "int"
	FieldTypeFloat    FieldType = "float"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeDate     FieldType = "date"
	FieldTypeTime     FieldType = "time"
	FieldTypeEnum     FieldType = "enum"
	FieldTypeJSON     FieldType = "json"
	FieldTypeMedia    FieldType = "media"
	FieldTypeRelation FieldType = "relation"
)

// validFieldTypes is the set of all supported field types, used for validation.
var validFieldTypes = map[FieldType]bool{
	FieldTypeString:   true,
	FieldTypeText:     true,
	FieldTypeRichText: true,
	FieldTypeInt:      true,
	FieldTypeFloat:    true,
	FieldTypeBoolean:  true,
	FieldTypeDate:     true,
	FieldTypeTime:     true,
	FieldTypeEnum:     true,
	FieldTypeJSON:     true,
	FieldTypeMedia:    true,
	FieldTypeRelation: true,
}

// RelationType represents the cardinality of a relation field.
type RelationType string

// Supported relation types.
const (
	RelationOne  RelationType = "one"
	RelationMany RelationType = "many"
)

// ContentType represents a parsed YAML content type schema definition.
type ContentType struct {
	// Name is the internal identifier (snake_case), used in table names and API routes.
	Name string `yaml:"name"`

	// DisplayName is the human-readable label shown in the admin UI.
	DisplayName string `yaml:"display_name"`

	// PublicRead indicates whether entries are readable via the public API.
	PublicRead bool `yaml:"public_read"`

	// Fields defines the list of fields for this content type.
	Fields []Field `yaml:"fields"`

	// SchemaHash is the SHA256 hex digest of the raw YAML file bytes.
	// It is computed after loading and is not deserialized from YAML.
	SchemaHash string `yaml:"-"`
}

// Field represents a single field within a content type definition.
type Field struct {
	// Name is the field identifier (snake_case), used as the database column name.
	Name string `yaml:"name"`

	// Type is the field type, which determines SQL type and validation rules.
	Type FieldType `yaml:"type"`

	// Required indicates the field must be provided on create (NOT NULL in SQL).
	Required bool `yaml:"required"`

	// Unique indicates the field value must be unique across entries (UNIQUE constraint).
	Unique bool `yaml:"unique"`

	// Searchable indicates the field is included in the full-text search vector.
	// Only valid on string, text, and richtext types.
	Searchable bool `yaml:"searchable"`

	// MinLength is the minimum character length. Only valid on string, text, richtext.
	MinLength *int `yaml:"min_length,omitempty"`

	// MaxLength is the maximum character length. Only valid on string, text, richtext.
	// For string fields, this also sets the VARCHAR limit.
	MaxLength *int `yaml:"max_length,omitempty"`

	// Min is the minimum numeric value. Only valid on int, float.
	Min *float64 `yaml:"min,omitempty"`

	// Max is the maximum numeric value. Only valid on int, float.
	Max *float64 `yaml:"max,omitempty"`

	// Regex is a Go regular expression pattern for validation. Only valid on string type.
	Regex string `yaml:"regex,omitempty"`

	// Values is the list of allowed values for enum fields.
	Values []string `yaml:"values,omitempty"`

	// RelatesTo is the target content type name for relation fields.
	RelatesTo string `yaml:"relates_to,omitempty"`

	// RelationType is the cardinality of the relation (one or many).
	RelationType RelationType `yaml:"relation_type,omitempty"`
}
