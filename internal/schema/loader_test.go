package schema

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeYAML is a test helper that writes a YAML file into the given directory.
func writeYAML(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("writing test YAML file %s: %v", filename, err)
	}
}

// ----- LoadSchemas tests -----

func TestLoadSchemas_ValidSchemas(t *testing.T) {
	// Use the real schema directory from the project root.
	schemas, err := LoadSchemas("../../schema")
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}

	if len(schemas) < 2 {
		t.Fatalf("expected at least 2 schemas, got %d", len(schemas))
	}

	// Schemas should be sorted by name.
	for i := 1; i < len(schemas); i++ {
		if schemas[i].Name < schemas[i-1].Name {
			t.Errorf("schemas not sorted: %q comes after %q", schemas[i].Name, schemas[i-1].Name)
		}
	}

	// Find the authors and blog_posts schemas.
	var authors, blogPosts *ContentType
	for i := range schemas {
		switch schemas[i].Name {
		case "authors":
			authors = &schemas[i]
		case "blog_posts":
			blogPosts = &schemas[i]
		}
	}

	if authors == nil {
		t.Fatal("expected to find authors schema")
	}
	if blogPosts == nil {
		t.Fatal("expected to find blog_posts schema")
	}

	// Check authors.
	if authors.DisplayName != "Authors" {
		t.Errorf("authors.DisplayName = %q, want %q", authors.DisplayName, "Authors")
	}
	if !authors.PublicRead {
		t.Error("authors.PublicRead should be true")
	}
	if len(authors.Fields) != 3 {
		t.Errorf("authors.Fields has %d fields, want 3", len(authors.Fields))
	}

	// Check blog_posts.
	if blogPosts.DisplayName != "Blog Posts" {
		t.Errorf("blogPosts.DisplayName = %q, want %q", blogPosts.DisplayName, "Blog Posts")
	}
	if len(blogPosts.Fields) != 6 {
		t.Errorf("blogPosts.Fields has %d fields, want 6", len(blogPosts.Fields))
	}

	// Check that the author relation is correctly parsed.
	var authorField *Field
	for i := range blogPosts.Fields {
		if blogPosts.Fields[i].Name == "author" {
			authorField = &blogPosts.Fields[i]
			break
		}
	}
	if authorField == nil {
		t.Fatal("blog_posts should have an author field")
	}
	if authorField.Type != FieldTypeRelation {
		t.Errorf("author field type = %q, want %q", authorField.Type, FieldTypeRelation)
	}
	if authorField.RelatesTo != "authors" {
		t.Errorf("author.RelatesTo = %q, want %q", authorField.RelatesTo, "authors")
	}
	if authorField.RelationType != RelationOne {
		t.Errorf("author.RelationType = %q, want %q", authorField.RelationType, RelationOne)
	}
}

func TestLoadSchemas_HashIsComputedAndNonEmpty(t *testing.T) {
	schemas, err := LoadSchemas("../../schema")
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}

	for _, ct := range schemas {
		if ct.SchemaHash == "" {
			t.Errorf("content type %q has empty SchemaHash", ct.Name)
		}
		// SHA256 hex is 64 characters.
		if len(ct.SchemaHash) != 64 {
			t.Errorf("content type %q SchemaHash has length %d, want 64", ct.Name, len(ct.SchemaHash))
		}
	}
}

func TestLoadSchemas_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	schemas, err := LoadSchemas(dir)
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("expected empty slice, got %d schemas", len(schemas))
	}
}

func TestLoadSchemas_MissingDirectory(t *testing.T) {
	_, err := LoadSchemas("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLoadSchemas_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "bad.yaml", "{{{{invalid yaml content")

	_, err := LoadSchemas(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing YAML") {
		t.Errorf("error should mention YAML parsing, got: %v", err)
	}
}

func TestLoadSchemas_SkipsNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "valid.yaml", `
name: test
display_name: Test
fields:
  - name: title
    type: string
`)
	writeYAML(t, dir, "readme.txt", "not a yaml schema")
	writeYAML(t, dir, "notes.md", "# Notes")

	schemas, err := LoadSchemas(dir)
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}
	if len(schemas) != 1 {
		t.Errorf("expected 1 schema (skipping non-.yaml files), got %d", len(schemas))
	}
}

func TestLoadSchemas_SortedByName(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "zebras.yaml", `
name: zebras
display_name: Zebras
fields:
  - name: stripe_count
    type: int
`)
	writeYAML(t, dir, "apples.yaml", `
name: apples
display_name: Apples
fields:
  - name: color
    type: string
`)
	writeYAML(t, dir, "middle.yaml", `
name: middle
display_name: Middle
fields:
  - name: value
    type: string
`)

	schemas, err := LoadSchemas(dir)
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}
	if len(schemas) != 3 {
		t.Fatalf("expected 3 schemas, got %d", len(schemas))
	}
	if schemas[0].Name != "apples" || schemas[1].Name != "middle" || schemas[2].Name != "zebras" {
		t.Errorf("schemas not sorted by name: got [%s, %s, %s]",
			schemas[0].Name, schemas[1].Name, schemas[2].Name)
	}
}

// ----- ValidateSchemas tests -----

func TestValidateSchemas_ValidSchemas(t *testing.T) {
	schemas, err := LoadSchemas("../../schema")
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}
	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("ValidateSchemas() error: %v", err)
	}
}

func TestValidateSchemas_MissingName(t *testing.T) {
	schemas := []ContentType{{
		DisplayName: "Test",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "name is required")
}

func TestValidateSchemas_InvalidName(t *testing.T) {
	tests := []struct {
		name    string
		ctName  string
		wantMsg string
	}{
		{
			name:    "starts with uppercase",
			ctName:  "BlogPosts",
			wantMsg: "name must match",
		},
		{
			name:    "starts with digit",
			ctName:  "1posts",
			wantMsg: "name must match",
		},
		{
			name:    "contains hyphen",
			ctName:  "blog-posts",
			wantMsg: "name must match",
		},
		{
			name:    "contains space",
			ctName:  "blog posts",
			wantMsg: "name must match",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schemas := []ContentType{{
				Name:        tc.ctName,
				DisplayName: "Test",
				Fields: []Field{
					{Name: "title", Type: FieldTypeString},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, tc.wantMsg)
		})
	}
}

func TestValidateSchemas_ReservedSQLName(t *testing.T) {
	reserved := []string{"select", "insert", "update", "delete", "drop", "table", "create", "alter", "index", "where", "from", "join"}

	for _, word := range reserved {
		t.Run(word, func(t *testing.T) {
			schemas := []ContentType{{
				Name:        word,
				DisplayName: "Test",
				Fields: []Field{
					{Name: "title", Type: FieldTypeString},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, "reserved SQL keyword")
		})
	}
}

func TestValidateSchemas_CTPrefix(t *testing.T) {
	schemas := []ContentType{{
		Name:        "ct_something",
		DisplayName: "Test",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "must not start with \"ct_\"")
}

func TestValidateSchemas_MissingDisplayName(t *testing.T) {
	schemas := []ContentType{{
		Name: "posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "display_name is required")
}

func TestValidateSchemas_NoFields(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields:      []Field{},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "at least one field is required")
}

func TestValidateSchemas_InvalidFieldName(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		wantMsg   string
	}{
		{
			name:      "starts with uppercase",
			fieldName: "Title",
			wantMsg:   "name must match",
		},
		{
			name:      "starts with digit",
			fieldName: "1field",
			wantMsg:   "name must match",
		},
		{
			name:      "contains hyphen",
			fieldName: "field-name",
			wantMsg:   "name must match",
		},
		{
			name:      "empty field name",
			fieldName: "",
			wantMsg:   "name is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: tc.fieldName, Type: FieldTypeString},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, tc.wantMsg)
		})
	}
}

func TestValidateSchemas_ReservedFieldName(t *testing.T) {
	reserved := []string{"id", "status", "search_vector", "created_by", "updated_by", "created_at", "updated_at", "published_at"}

	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: name, Type: FieldTypeString},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, "reserved column name")
		})
	}
}

func TestValidateSchemas_DuplicateFieldName(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "title", Type: FieldTypeText},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "duplicate field name")
}

func TestValidateSchemas_InvalidFieldType(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldType("unknown")},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "invalid field type")
}

func TestValidateSchemas_EnumWithoutValues(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "enum field must have a non-empty values list")
}

func TestValidateSchemas_EnumWithEmptyValue(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "", "design"}},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "values[1] must not be empty")
}

func TestValidateSchemas_RelationWithoutRelatesTo(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "author", Type: FieldTypeRelation, RelationType: RelationOne},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "must have relates_to")
}

func TestValidateSchemas_RelationWithoutRelationType(t *testing.T) {
	schemas := []ContentType{
		{
			Name:        "authors",
			DisplayName: "Authors",
			Fields: []Field{
				{Name: "name", Type: FieldTypeString},
			},
		},
		{
			Name:        "posts",
			DisplayName: "Posts",
			Fields: []Field{
				{Name: "author", Type: FieldTypeRelation, RelatesTo: "authors"},
			},
		},
	}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "relation_type")
}

func TestValidateSchemas_RelationWithInvalidTarget(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "author", Type: FieldTypeRelation, RelatesTo: "nonexistent", RelationType: RelationOne},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "references unknown content type")
}

func TestValidateSchemas_RelationWithValidTarget(t *testing.T) {
	schemas := []ContentType{
		{
			Name:        "authors",
			DisplayName: "Authors",
			Fields: []Field{
				{Name: "name", Type: FieldTypeString},
			},
		},
		{
			Name:        "posts",
			DisplayName: "Posts",
			Fields: []Field{
				{Name: "author", Type: FieldTypeRelation, RelatesTo: "authors", RelationType: RelationOne},
			},
		},
	}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid schemas, got: %v", err)
	}
}

func TestValidateSchemas_SearchableOnNonTextField(t *testing.T) {
	nonTextTypes := []FieldType{
		FieldTypeInt, FieldTypeFloat, FieldTypeBoolean,
		FieldTypeDate, FieldTypeTime, FieldTypeJSON, FieldTypeMedia,
	}

	for _, ft := range nonTextTypes {
		t.Run(string(ft), func(t *testing.T) {
			fields := []Field{{Name: "val", Type: ft, Searchable: true}}
			// Enum needs values, relation needs extra config, so skip those.
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields:      fields,
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, "searchable is only valid on string, text, richtext")
		})
	}
}

func TestValidateSchemas_SearchableOnTextFieldIsValid(t *testing.T) {
	for _, ft := range []FieldType{FieldTypeString, FieldTypeText, FieldTypeRichText} {
		t.Run(string(ft), func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: "val", Type: ft, Searchable: true},
				},
			}}
			if err := ValidateSchemas(schemas); err != nil {
				t.Fatalf("expected valid, got: %v", err)
			}
		})
	}
}

func TestValidateSchemas_MinMaxOnNonNumericField(t *testing.T) {
	min := 1.0
	max := 10.0

	nonNumericTypes := []FieldType{
		FieldTypeString, FieldTypeText, FieldTypeRichText,
		FieldTypeBoolean, FieldTypeDate, FieldTypeTime,
		FieldTypeJSON, FieldTypeMedia,
	}

	for _, ft := range nonNumericTypes {
		t.Run(string(ft)+"_min", func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: "val", Type: ft, Min: &min},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, "min is only valid on int, float")
		})
		t.Run(string(ft)+"_max", func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: "val", Type: ft, Max: &max},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, "max is only valid on int, float")
		})
	}
}

func TestValidateSchemas_MinMaxOnNumericFieldIsValid(t *testing.T) {
	min := 0.0
	max := 100.0

	for _, ft := range []FieldType{FieldTypeInt, FieldTypeFloat} {
		t.Run(string(ft), func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: "val", Type: ft, Min: &min, Max: &max},
				},
			}}
			if err := ValidateSchemas(schemas); err != nil {
				t.Fatalf("expected valid, got: %v", err)
			}
		})
	}
}

func TestValidateSchemas_MinGreaterThanMax(t *testing.T) {
	min := 100.0
	max := 1.0

	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "val", Type: FieldTypeInt, Min: &min, Max: &max},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "min (100) must be <= max (1)")
}

func TestValidateSchemas_MinLengthMaxLengthOnNonTextField(t *testing.T) {
	minLen := 1
	maxLen := 10

	nonTextTypes := []FieldType{
		FieldTypeInt, FieldTypeFloat, FieldTypeBoolean,
		FieldTypeDate, FieldTypeTime, FieldTypeJSON, FieldTypeMedia,
	}

	for _, ft := range nonTextTypes {
		t.Run(string(ft)+"_min_length", func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: "val", Type: ft, MinLength: &minLen},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, "min_length is only valid on string, text, richtext")
		})
		t.Run(string(ft)+"_max_length", func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: "val", Type: ft, MaxLength: &maxLen},
				},
			}}
			err := ValidateSchemas(schemas)
			requireValidationError(t, err, "max_length is only valid on string, text, richtext")
		})
	}
}

func TestValidateSchemas_MinLengthMaxLengthOnTextFieldIsValid(t *testing.T) {
	minLen := 1
	maxLen := 100

	for _, ft := range []FieldType{FieldTypeString, FieldTypeText, FieldTypeRichText} {
		t.Run(string(ft), func(t *testing.T) {
			schemas := []ContentType{{
				Name:        "posts",
				DisplayName: "Posts",
				Fields: []Field{
					{Name: "val", Type: ft, MinLength: &minLen, MaxLength: &maxLen},
				},
			}}
			if err := ValidateSchemas(schemas); err != nil {
				t.Fatalf("expected valid, got: %v", err)
			}
		})
	}
}

func TestValidateSchemas_MinLengthGreaterThanMaxLength(t *testing.T) {
	minLen := 100
	maxLen := 10

	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "val", Type: FieldTypeString, MinLength: &minLen, MaxLength: &maxLen},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "min_length (100) must be <= max_length (10)")
}

func TestValidateSchemas_InvalidRegex(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "slug", Type: FieldTypeString, Regex: "[invalid(regex"},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "invalid regex")
}

func TestValidateSchemas_ValidRegex(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "slug", Type: FieldTypeString, Regex: `^[a-z0-9]+(?:-[a-z0-9]+)*$`},
		},
	}}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateSchemas_RegexOnNonStringField(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "body", Type: FieldTypeText, Regex: `^[a-z]+$`},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "regex is only valid on string type")
}

func TestValidateSchemas_MultipleProblemsReportedTogether(t *testing.T) {
	schemas := []ContentType{{
		Name: "",
		// Missing display_name too.
		Fields: []Field{},
	}}

	err := ValidateSchemas(schemas)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	// Should have at least: name required, display_name required, no fields.
	if len(ve.Problems) < 3 {
		t.Errorf("expected at least 3 problems, got %d: %v", len(ve.Problems), ve.Problems)
	}
}

func TestValidateSchemas_DuplicateContentTypeName(t *testing.T) {
	schemas := []ContentType{
		{
			Name:        "posts",
			DisplayName: "Posts",
			Fields:      []Field{{Name: "title", Type: FieldTypeString}},
		},
		{
			Name:        "posts",
			DisplayName: "Posts Again",
			Fields:      []Field{{Name: "body", Type: FieldTypeText}},
		},
	}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "defined 2 times")
}

func TestValidateSchemas_EnumFieldValid(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "business"}},
		},
	}}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateSchemas_AllFieldTypesAccepted(t *testing.T) {
	min := 0.0
	max := 100.0

	schemas := []ContentType{
		{
			Name:        "target",
			DisplayName: "Target",
			Fields: []Field{
				{Name: "name", Type: FieldTypeString},
			},
		},
		{
			Name:        "all_types",
			DisplayName: "All Types",
			Fields: []Field{
				{Name: "f_string", Type: FieldTypeString},
				{Name: "f_text", Type: FieldTypeText},
				{Name: "f_richtext", Type: FieldTypeRichText},
				{Name: "f_int", Type: FieldTypeInt, Min: &min, Max: &max},
				{Name: "f_float", Type: FieldTypeFloat},
				{Name: "f_boolean", Type: FieldTypeBoolean},
				{Name: "f_date", Type: FieldTypeDate},
				{Name: "f_time", Type: FieldTypeTime},
				{Name: "f_enum", Type: FieldTypeEnum, Values: []string{"a", "b"}},
				{Name: "f_json", Type: FieldTypeJSON},
				{Name: "f_media", Type: FieldTypeMedia},
				{Name: "f_relation", Type: FieldTypeRelation, RelatesTo: "target", RelationType: RelationOne},
			},
		},
	}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

// ----- Integration: load and validate from temp dir -----

func TestLoadAndValidate_FromTempDir(t *testing.T) {
	dir := t.TempDir()

	writeYAML(t, dir, "authors.yaml", `
name: authors
display_name: Authors
public_read: true
fields:
  - name: name
    type: string
    required: true
    searchable: true
    max_length: 100
  - name: bio
    type: text
    searchable: true
`)
	writeYAML(t, dir, "posts.yaml", `
name: posts
display_name: Posts
public_read: true
fields:
  - name: title
    type: string
    required: true
    searchable: true
  - name: author
    type: relation
    relates_to: authors
    relation_type: one
`)

	schemas, err := LoadSchemas(dir)
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("ValidateSchemas() error: %v", err)
	}
}

func TestLoadAndValidate_RelationTargetNotFound(t *testing.T) {
	dir := t.TempDir()

	writeYAML(t, dir, "posts.yaml", `
name: posts
display_name: Posts
fields:
  - name: author
    type: relation
    relates_to: authors_that_dont_exist
    relation_type: one
`)

	schemas, err := LoadSchemas(dir)
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}

	err = ValidateSchemas(schemas)
	requireValidationError(t, err, "references unknown content type")
}

func TestValidateSchemas_RelationManyType(t *testing.T) {
	schemas := []ContentType{
		{
			Name:        "tags",
			DisplayName: "Tags",
			Fields: []Field{
				{Name: "label", Type: FieldTypeString},
			},
		},
		{
			Name:        "posts",
			DisplayName: "Posts",
			Fields: []Field{
				{Name: "tags", Type: FieldTypeRelation, RelatesTo: "tags", RelationType: RelationMany},
			},
		},
	}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

// ----- Fix 1: Accept .yml files -----

func TestLoadSchemas_AcceptsYMLExtension(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "items.yml", `
name: items
display_name: Items
fields:
  - name: title
    type: string
`)

	schemas, err := LoadSchemas(dir)
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	if schemas[0].Name != "items" {
		t.Errorf("schema name = %q, want %q", schemas[0].Name, "items")
	}
}

func TestLoadSchemas_MixedYAMLandYML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "apples.yaml", `
name: apples
display_name: Apples
fields:
  - name: color
    type: string
`)
	writeYAML(t, dir, "bananas.yml", `
name: bananas
display_name: Bananas
fields:
  - name: length
    type: int
`)

	schemas, err := LoadSchemas(dir)
	if err != nil {
		t.Fatalf("LoadSchemas() error: %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}
	if schemas[0].Name != "apples" || schemas[1].Name != "bananas" {
		t.Errorf("expected [apples, bananas], got [%s, %s]", schemas[0].Name, schemas[1].Name)
	}
}

// ----- Fix 2: Non-negative min_length/max_length -----

func TestValidateSchemas_NegativeMinLength(t *testing.T) {
	neg := -1
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, MinLength: &neg},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "min_length must be >= 0")
}

func TestValidateSchemas_ZeroMaxLength(t *testing.T) {
	zero := 0
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, MaxLength: &zero},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "max_length must be > 0")
}

func TestValidateSchemas_NegativeMaxLength(t *testing.T) {
	neg := -5
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, MaxLength: &neg},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "max_length must be > 0")
}

func TestValidateSchemas_ZeroMinLengthIsValid(t *testing.T) {
	zero := 0
	maxLen := 100
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, MinLength: &zero, MaxLength: &maxLen},
		},
	}}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid (min_length=0 is ok), got: %v", err)
	}
}

// ----- Fix 3: Type-inapplicable properties -----

func TestValidateSchemas_ValuesOnNonEnumField(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Values: []string{"a", "b"}},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "values is only valid on enum type")
}

func TestValidateSchemas_RelatesToOnNonRelationField(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, RelatesTo: "authors"},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "relates_to is only valid on relation type")
}

func TestValidateSchemas_RelationTypeOnNonRelationField(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, RelationType: RelationOne},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "relation_type is only valid on relation type")
}

// ----- Fix 4: Duplicate enum values -----

func TestValidateSchemas_DuplicateEnumValues(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "tech"}},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "duplicate enum value \"tech\"")
}

func TestValidateSchemas_MultipleDuplicateEnumValues(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"a", "b", "a", "b", "c"}},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "duplicate enum value \"a\"")
	requireValidationError(t, err, "duplicate enum value \"b\"")
}

// ----- Fix 5: Name length limits -----

func TestValidateSchemas_ContentTypeNameTooLong(t *testing.T) {
	// 60 characters exceeds the 59-character limit.
	longName := strings.Repeat("a", 60)
	schemas := []ContentType{{
		Name:        longName,
		DisplayName: "Long",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "name must be at most 59 characters")
}

func TestValidateSchemas_ContentTypeNameAtLimit(t *testing.T) {
	// Exactly 59 characters should be valid.
	name59 := strings.Repeat("a", 59)
	schemas := []ContentType{{
		Name:        name59,
		DisplayName: "Limit",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid (59 chars is at limit), got: %v", err)
	}
}

func TestValidateSchemas_FieldNameTooLong(t *testing.T) {
	// 64 characters exceeds the 63-character limit.
	longField := strings.Repeat("f", 64)
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: longField, Type: FieldTypeString},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "name must be at most 63 characters")
}

func TestValidateSchemas_FieldNameAtLimit(t *testing.T) {
	// Exactly 63 characters should be valid.
	field63 := strings.Repeat("f", 63)
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: field63, Type: FieldTypeString},
		},
	}}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid (63 chars is at limit), got: %v", err)
	}
}

// ----- Fix 6: KnownFields rejects unknown YAML keys -----

func TestLoadSchemas_UnknownYAMLFieldRejected(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "bad.yaml", `
name: posts
display_name: Posts
unknown_property: true
fields:
  - name: title
    type: string
`)

	_, err := LoadSchemas(dir)
	if err == nil {
		t.Fatal("expected error for unknown YAML field, got nil")
	}
	if !strings.Contains(err.Error(), "parsing YAML") {
		t.Errorf("error should mention YAML parsing, got: %v", err)
	}
}

func TestLoadSchemas_MisspelledFieldPropertyRejected(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "bad.yaml", `
name: posts
display_name: Posts
fields:
  - name: title
    type: string
    requred: true
`)

	_, err := LoadSchemas(dir)
	if err == nil {
		t.Fatal("expected error for misspelled YAML field 'requred', got nil")
	}
	if !strings.Contains(err.Error(), "parsing YAML") {
		t.Errorf("error should mention YAML parsing, got: %v", err)
	}
}

// ----- Fix: required media/relation-one fields rejected -----

func TestValidateSchemas_RequiredMediaField_Rejected(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "cover", Type: FieldTypeMedia, Required: true},
		},
	}}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "required is not supported on media/relation fields")
}

func TestValidateSchemas_RequiredRelationOneField_Rejected(t *testing.T) {
	schemas := []ContentType{
		{
			Name:        "authors",
			DisplayName: "Authors",
			Fields: []Field{
				{Name: "name", Type: FieldTypeString},
			},
		},
		{
			Name:        "posts",
			DisplayName: "Posts",
			Fields: []Field{
				{Name: "author", Type: FieldTypeRelation, RelatesTo: "authors", RelationType: RelationOne, Required: true},
			},
		},
	}

	err := ValidateSchemas(schemas)
	requireValidationError(t, err, "required is not supported on media/relation fields")
}

func TestValidateSchemas_OptionalMediaField_Valid(t *testing.T) {
	schemas := []ContentType{{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "cover", Type: FieldTypeMedia, Required: false},
		},
	}}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid (optional media field), got: %v", err)
	}
}

func TestValidateSchemas_RequiredRelationManyField_NotRejected(t *testing.T) {
	// relation-many fields do not produce a FK column, so required + ON DELETE
	// SET NULL is not a concern.
	schemas := []ContentType{
		{
			Name:        "tags",
			DisplayName: "Tags",
			Fields: []Field{
				{Name: "label", Type: FieldTypeString},
			},
		},
		{
			Name:        "posts",
			DisplayName: "Posts",
			Fields: []Field{
				{Name: "tags", Type: FieldTypeRelation, RelatesTo: "tags", RelationType: RelationMany, Required: true},
			},
		},
	}

	if err := ValidateSchemas(schemas); err != nil {
		t.Fatalf("expected valid (relation-many required is fine), got: %v", err)
	}
}

// ----- Helpers -----

// requireValidationError asserts that err is a *ValidationError containing
// at least one problem that includes the given substring.
func requireValidationError(t *testing.T, err error, wantSubstring string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected validation error containing %q, got nil", wantSubstring)
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}

	for _, problem := range ve.Problems {
		if strings.Contains(problem, wantSubstring) {
			return
		}
	}

	t.Errorf("expected a problem containing %q, got problems:\n- %s",
		wantSubstring, strings.Join(ve.Problems, "\n- "))
}
