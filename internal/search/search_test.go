package search

import (
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

func TestBuildSearchClause_WithSearchableFields(t *testing.T) {
	fields := []schema.Field{
		{Name: "title", Type: schema.FieldTypeString, Searchable: true},
		{Name: "body", Type: schema.FieldTypeText, Searchable: true},
	}

	where, order, headline, args := BuildSearchClause("hello world", fields, 3)

	if where != `"search_vector" @@ plainto_tsquery('english', $3)` {
		t.Errorf("unexpected whereClause: %s", where)
	}

	if order != `ts_rank("search_vector", plainto_tsquery('english', $3)) DESC` {
		t.Errorf("unexpected orderClause: %s", order)
	}

	// Headline should use the first searchable field ("title").
	expectedHeadline := `ts_headline('english', "title", plainto_tsquery('english', $3)) AS "_search_headline"`
	if headline != expectedHeadline {
		t.Errorf("unexpected headlineExpr:\n  got:  %s\n  want: %s", headline, expectedHeadline)
	}

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != "hello world" {
		t.Errorf("expected arg 'hello world', got %v", args[0])
	}
}

func TestBuildSearchClause_NoSearchableFields(t *testing.T) {
	// An empty slice means the caller determined no fields are searchable.
	var fields []schema.Field

	where, order, headline, args := BuildSearchClause("test", fields, 1)

	if where != "" {
		t.Errorf("expected empty whereClause, got: %s", where)
	}
	if order != "" {
		t.Errorf("expected empty orderClause, got: %s", order)
	}
	if headline != "" {
		t.Errorf("expected empty headlineExpr, got: %s", headline)
	}
	if args != nil {
		t.Errorf("expected nil args, got: %v", args)
	}
}

func TestBuildSearchClause_EmptyFieldSlice(t *testing.T) {
	where, order, headline, args := BuildSearchClause("test", nil, 1)

	if where != "" || order != "" || headline != "" || args != nil {
		t.Error("expected all zero values for nil fields")
	}
}

func TestBuildSearchClause_ParameterIndex(t *testing.T) {
	fields := []schema.Field{
		{Name: "name", Type: schema.FieldTypeString, Searchable: true},
	}

	tests := []struct {
		paramIdx    int
		wantContain string
	}{
		{1, "$1"},
		{5, "$5"},
		{12, "$12"},
	}

	for _, tt := range tests {
		t.Run(tt.wantContain, func(t *testing.T) {
			where, order, headline, _ := BuildSearchClause("query", fields, tt.paramIdx)

			if where == "" {
				t.Fatal("expected non-empty whereClause")
			}

			// All three should reference the same $N.
			for _, clause := range []struct {
				name, val string
			}{
				{"where", where},
				{"order", order},
				{"headline", headline},
			} {
				if !containsStr(clause.val, tt.wantContain) {
					t.Errorf("%s clause does not contain %s: %s", clause.name, tt.wantContain, clause.val)
				}
			}
		})
	}
}

func TestBuildSearchClause_HeadlineUsesFirstSearchableField(t *testing.T) {
	fields := []schema.Field{
		{Name: "summary", Type: schema.FieldTypeText, Searchable: true},
		{Name: "content", Type: schema.FieldTypeRichText, Searchable: true},
		{Name: "title", Type: schema.FieldTypeString, Searchable: true},
	}

	_, _, headline, _ := BuildSearchClause("test", fields, 1)

	// Should use "summary" (the first in the slice), not "title" or "content".
	expected := `ts_headline('english', "summary", plainto_tsquery('english', $1)) AS "_search_headline"`
	if headline != expected {
		t.Errorf("headline should use first field 'summary':\n  got:  %s\n  want: %s", headline, expected)
	}
}

func TestBuildSearchClause_QuoteIdentApplied(t *testing.T) {
	// Field names should be double-quoted in the output.
	fields := []schema.Field{
		{Name: "my_field", Type: schema.FieldTypeString, Searchable: true},
	}

	where, _, headline, _ := BuildSearchClause("test", fields, 1)

	// search_vector should be quoted.
	if !containsStr(where, `"search_vector"`) {
		t.Errorf("whereClause should contain quoted search_vector: %s", where)
	}

	// Headline field name should be quoted.
	if !containsStr(headline, `"my_field"`) {
		t.Errorf("headlineExpr should contain quoted field name: %s", headline)
	}

	// Headline alias should be quoted.
	if !containsStr(headline, `"_search_headline"`) {
		t.Errorf("headlineExpr should contain quoted alias: %s", headline)
	}
}

// containsStr is a small helper to check substring presence.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
