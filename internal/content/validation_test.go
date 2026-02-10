package content

import (
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

func intPtr(v int) *int       { return &v }
func floatPtr(v float64) *float64 { return &v }

func TestValidateEntry_RequiredFields(t *testing.T) {
	ct := schema.ContentType{
		Name: "posts",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldTypeString, Required: true},
			{Name: "body", Type: schema.FieldTypeText, Required: false},
		},
	}

	t.Run("create missing required field", func(t *testing.T) {
		errs := ValidateEntry(ct, map[string]any{"body": "hello"}, false)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "title" || errs[0].Message != "is required" {
			t.Errorf("unexpected error: %+v", errs[0])
		}
	})

	t.Run("create with nil required field", func(t *testing.T) {
		errs := ValidateEntry(ct, map[string]any{"title": nil}, false)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
	})

	t.Run("update skips missing required field", func(t *testing.T) {
		errs := ValidateEntry(ct, map[string]any{"body": "hello"}, true)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors on update, got %d: %v", len(errs), errs)
		}
	})

	t.Run("create with all required fields", func(t *testing.T) {
		errs := ValidateEntry(ct, map[string]any{"title": "Hello"}, false)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestValidateEntry_UnknownFields(t *testing.T) {
	ct := schema.ContentType{
		Name: "posts",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldTypeString},
		},
	}

	errs := ValidateEntry(ct, map[string]any{"title": "Hi", "evil_field": "drop table"}, false)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for unknown field, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "evil_field" {
		t.Errorf("expected error on 'evil_field', got %q", errs[0].Field)
	}
}

func TestValidateEntry_TypeChecking(t *testing.T) {
	tests := []struct {
		name    string
		field   schema.Field
		value   any
		wantErr bool
	}{
		{"string valid", schema.Field{Name: "f", Type: schema.FieldTypeString}, "hello", false},
		{"string invalid", schema.Field{Name: "f", Type: schema.FieldTypeString}, 42.0, true},
		{"int valid", schema.Field{Name: "f", Type: schema.FieldTypeInt}, 42.0, false},
		{"int float value", schema.Field{Name: "f", Type: schema.FieldTypeInt}, 42.5, true},
		{"int not number", schema.Field{Name: "f", Type: schema.FieldTypeInt}, "abc", true},
		{"float valid", schema.Field{Name: "f", Type: schema.FieldTypeFloat}, 3.14, false},
		{"float not number", schema.Field{Name: "f", Type: schema.FieldTypeFloat}, true, true},
		{"boolean valid", schema.Field{Name: "f", Type: schema.FieldTypeBoolean}, true, false},
		{"boolean invalid", schema.Field{Name: "f", Type: schema.FieldTypeBoolean}, "true", true},
		{"date valid", schema.Field{Name: "f", Type: schema.FieldTypeDate}, "2024-01-15", false},
		{"date invalid format", schema.Field{Name: "f", Type: schema.FieldTypeDate}, "01-15-2024", true},
		{"date not string", schema.Field{Name: "f", Type: schema.FieldTypeDate}, 123.0, true},
		{"time valid HH:MM", schema.Field{Name: "f", Type: schema.FieldTypeTime}, "14:30", false},
		{"time valid HH:MM:SS", schema.Field{Name: "f", Type: schema.FieldTypeTime}, "14:30:00", false},
		{"time invalid", schema.Field{Name: "f", Type: schema.FieldTypeTime}, "2:30 PM", true},
		{"enum valid", schema.Field{Name: "f", Type: schema.FieldTypeEnum, Values: []string{"a", "b"}}, "a", false},
		{"enum invalid value", schema.Field{Name: "f", Type: schema.FieldTypeEnum, Values: []string{"a", "b"}}, "c", true},
		{"enum not string", schema.Field{Name: "f", Type: schema.FieldTypeEnum, Values: []string{"a"}}, 1.0, true},
		{"json any value", schema.Field{Name: "f", Type: schema.FieldTypeJSON}, map[string]any{"key": "val"}, false},
		{"media valid uuid", schema.Field{Name: "f", Type: schema.FieldTypeMedia}, "550e8400-e29b-41d4-a716-446655440000", false},
		{"media invalid uuid", schema.Field{Name: "f", Type: schema.FieldTypeMedia}, "not-a-uuid", true},
		{"media not string", schema.Field{Name: "f", Type: schema.FieldTypeMedia}, 123.0, true},
		{"relation one valid", schema.Field{Name: "f", Type: schema.FieldTypeRelation, RelationType: schema.RelationOne}, "550e8400-e29b-41d4-a716-446655440000", false},
		{"relation one invalid", schema.Field{Name: "f", Type: schema.FieldTypeRelation, RelationType: schema.RelationOne}, "bad", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := schema.ContentType{
				Name:   "test",
				Fields: []schema.Field{tt.field},
			}
			errs := ValidateEntry(ct, map[string]any{tt.field.Name: tt.value}, false)
			if tt.wantErr && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidateEntry_StringConstraints(t *testing.T) {
	ct := schema.ContentType{
		Name: "test",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldTypeString, MinLength: intPtr(3), MaxLength: intPtr(10), Regex: "^[a-z]+$"},
		},
	}

	tests := []struct {
		name    string
		value   string
		wantErr int
	}{
		{"valid", "hello", 0},
		{"too short", "ab", 1},
		{"too long", "abcdefghijk", 1},
		{"regex fail", "Hello", 1},
		{"too short and regex fail", "AB", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateEntry(ct, map[string]any{"title": tt.value}, false)
			if len(errs) != tt.wantErr {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErr, len(errs), errs)
			}
		})
	}
}

func TestValidateEntry_NumericConstraints(t *testing.T) {
	ct := schema.ContentType{
		Name: "test",
		Fields: []schema.Field{
			{Name: "age", Type: schema.FieldTypeInt, Min: floatPtr(0), Max: floatPtr(150)},
		},
	}

	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{"valid", 25, false},
		{"at min", 0, false},
		{"at max", 150, false},
		{"below min", -1, true},
		{"above max", 151, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateEntry(ct, map[string]any{"age": tt.value}, false)
			if tt.wantErr && len(errs) == 0 {
				t.Error("expected error, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidateEntry_ManyRelationSkipped(t *testing.T) {
	ct := schema.ContentType{
		Name: "test",
		Fields: []schema.Field{
			{Name: "tags", Type: schema.FieldTypeRelation, RelationType: schema.RelationMany, Required: true},
		},
	}

	// Many-to-many relations should be skipped entirely during validation.
	errs := ValidateEntry(ct, map[string]any{}, false)
	if len(errs) != 0 {
		t.Errorf("expected no errors for many-relation, got %v", errs)
	}
}

func TestValidateEntry_DateSemanticValidation(t *testing.T) {
	ct := schema.ContentType{
		Name: "test",
		Fields: []schema.Field{
			{Name: "d", Type: schema.FieldTypeDate},
		},
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid date", "2024-01-15", false},
		{"valid leap day", "2024-02-29", false},
		{"invalid Feb 31", "2024-02-31", true},
		{"invalid month 13", "2024-13-01", true},
		{"invalid day 00", "2024-01-00", true},
		{"wrong format", "01-15-2024", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateEntry(ct, map[string]any{"d": tt.value}, false)
			if tt.wantErr && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidateEntry_TimeSemanticValidation(t *testing.T) {
	ct := schema.ContentType{
		Name: "test",
		Fields: []schema.Field{
			{Name: "t", Type: schema.FieldTypeTime},
		},
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid HH:MM", "14:30", false},
		{"valid HH:MM:SS", "14:30:00", false},
		{"valid midnight", "00:00", false},
		{"valid 23:59:59", "23:59:59", false},
		{"invalid hour 99", "99:99", true},
		{"invalid hour 25", "25:00", true},
		{"invalid minute 60", "12:60", true},
		{"invalid second 60", "12:30:60", true},
		{"AM/PM format", "2:30 PM", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateEntry(ct, map[string]any{"t": tt.value}, false)
			if tt.wantErr && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidateEntry_MultipleErrors(t *testing.T) {
	ct := schema.ContentType{
		Name: "test",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldTypeString, Required: true},
			{Name: "count", Type: schema.FieldTypeInt, Required: true},
		},
	}

	errs := ValidateEntry(ct, map[string]any{}, false)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}
