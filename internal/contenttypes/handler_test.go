package contenttypes

import (
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

func TestBuildResponse(t *testing.T) {
	minLen := 1
	maxLen := 255
	minVal := 0.0
	maxVal := 100.0

	ct := schema.ContentType{
		Name:        "articles",
		DisplayName: "Articles",
		PublicRead:  true,
		Fields: []schema.Field{
			{
				Name:       "title",
				Type:       schema.FieldTypeString,
				Required:   true,
				Unique:     true,
				Searchable: true,
				MinLength:  &minLen,
				MaxLength:  &maxLen,
			},
			{
				Name: "body",
				Type: schema.FieldTypeRichText,
			},
			{
				Name:   "rating",
				Type:   schema.FieldTypeFloat,
				Min:    &minVal,
				Max:    &maxVal,
			},
			{
				Name:   "status_field",
				Type:   schema.FieldTypeEnum,
				Values: []string{"active", "inactive"},
			},
			{
				Name:         "category",
				Type:         schema.FieldTypeRelation,
				RelatesTo:    "categories",
				RelationType: schema.RelationOne,
			},
		},
	}

	resp := buildResponse(ct, 42)

	if resp.Name != "articles" {
		t.Errorf("Name = %q, want %q", resp.Name, "articles")
	}
	if resp.DisplayName != "Articles" {
		t.Errorf("DisplayName = %q, want %q", resp.DisplayName, "Articles")
	}
	if !resp.PublicRead {
		t.Error("PublicRead = false, want true")
	}
	if resp.EntryCount != 42 {
		t.Errorf("EntryCount = %d, want %d", resp.EntryCount, 42)
	}
	if len(resp.Fields) != 5 {
		t.Fatalf("len(Fields) = %d, want %d", len(resp.Fields), 5)
	}

	// Verify first field (title) preserves all attributes.
	title := resp.Fields[0]
	if title.Name != "title" {
		t.Errorf("Fields[0].Name = %q, want %q", title.Name, "title")
	}
	if title.Type != schema.FieldTypeString {
		t.Errorf("Fields[0].Type = %q, want %q", title.Type, schema.FieldTypeString)
	}
	if !title.Required {
		t.Error("Fields[0].Required = false, want true")
	}
	if !title.Unique {
		t.Error("Fields[0].Unique = false, want true")
	}
	if !title.Searchable {
		t.Error("Fields[0].Searchable = false, want true")
	}
	if title.MinLength == nil || *title.MinLength != 1 {
		t.Errorf("Fields[0].MinLength = %v, want 1", title.MinLength)
	}
	if title.MaxLength == nil || *title.MaxLength != 255 {
		t.Errorf("Fields[0].MaxLength = %v, want 255", title.MaxLength)
	}

	// Verify numeric field constraints.
	rating := resp.Fields[2]
	if rating.Min == nil || *rating.Min != 0.0 {
		t.Errorf("Fields[2].Min = %v, want 0.0", rating.Min)
	}
	if rating.Max == nil || *rating.Max != 100.0 {
		t.Errorf("Fields[2].Max = %v, want 100.0", rating.Max)
	}

	// Verify enum field.
	enumField := resp.Fields[3]
	if len(enumField.Values) != 2 || enumField.Values[0] != "active" {
		t.Errorf("Fields[3].Values = %v, want [active inactive]", enumField.Values)
	}

	// Verify relation field.
	rel := resp.Fields[4]
	if rel.RelatesTo != "categories" {
		t.Errorf("Fields[4].RelatesTo = %q, want %q", rel.RelatesTo, "categories")
	}
	if rel.RelationType != schema.RelationOne {
		t.Errorf("Fields[4].RelationType = %q, want %q", rel.RelationType, schema.RelationOne)
	}
}

func TestBuildResponseEmptyFields(t *testing.T) {
	ct := schema.ContentType{
		Name:        "empty",
		DisplayName: "Empty Type",
		Fields:      []schema.Field{},
	}

	resp := buildResponse(ct, 0)

	if len(resp.Fields) != 0 {
		t.Errorf("len(Fields) = %d, want 0", len(resp.Fields))
	}
	if resp.EntryCount != 0 {
		t.Errorf("EntryCount = %d, want 0", resp.EntryCount)
	}
}

func TestGetSchemasSorted(t *testing.T) {
	schemas := map[string]schema.ContentType{
		"zebras":   {Name: "zebras", DisplayName: "Zebras"},
		"articles": {Name: "articles", DisplayName: "Articles"},
		"media":    {Name: "media", DisplayName: "Media"},
	}

	h := NewHandler(nil, schemas)
	result := h.getSchemas()

	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0].Name != "articles" {
		t.Errorf("[0].Name = %q, want %q", result[0].Name, "articles")
	}
	if result[1].Name != "media" {
		t.Errorf("[1].Name = %q, want %q", result[1].Name, "media")
	}
	if result[2].Name != "zebras" {
		t.Errorf("[2].Name = %q, want %q", result[2].Name, "zebras")
	}
}

func TestGetSchemaLookup(t *testing.T) {
	schemas := map[string]schema.ContentType{
		"articles": {Name: "articles", DisplayName: "Articles"},
	}

	h := NewHandler(nil, schemas)

	ct, ok := h.getSchema("articles")
	if !ok {
		t.Fatal("getSchema returned false for existing key")
	}
	if ct.Name != "articles" {
		t.Errorf("Name = %q, want %q", ct.Name, "articles")
	}

	_, ok = h.getSchema("nonexistent")
	if ok {
		t.Error("getSchema returned true for nonexistent key")
	}
}

func TestUpdateSchemas(t *testing.T) {
	h := NewHandler(nil, map[string]schema.ContentType{
		"old": {Name: "old"},
	})

	newSchemas := map[string]schema.ContentType{
		"new": {Name: "new"},
	}
	h.UpdateSchemas(newSchemas)

	_, ok := h.getSchema("old")
	if ok {
		t.Error("old schema should not exist after update")
	}

	ct, ok := h.getSchema("new")
	if !ok {
		t.Fatal("new schema should exist after update")
	}
	if ct.Name != "new" {
		t.Errorf("Name = %q, want %q", ct.Name, "new")
	}
}
