package schema

import (
	"strings"
	"testing"
)

func TestDiffSchema_NewSchema_NilExisting(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
		},
	}

	changes := DiffSchema(ct, nil)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	c := changes[0]
	if c.Type != ChangeCreateTable {
		t.Errorf("expected ChangeCreateTable, got %s", c.Type)
	}
	if !c.Safe {
		t.Error("create table should be safe")
	}
	if c.Table != "ct_posts" {
		t.Errorf("table = %q, want %q", c.Table, "ct_posts")
	}
	assertContains(t, c.SQL, `CREATE TABLE "ct_posts" (`)
	assertContains(t, c.Detail, "create new table")
}

func TestDiffSchema_IdenticalSchemas_NoChanges(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
			{Name: "body", Type: FieldTypeText},
		},
	}

	existing := ct // identical copy

	changes := DiffSchema(ct, &existing)

	if len(changes) != 0 {
		t.Errorf("expected 0 changes for identical schemas, got %d:", len(changes))
		for _, c := range changes {
			t.Logf("  - %s: %s", c.Type, c.Detail)
		}
	}
}

func TestDiffSchema_AddNullableField_Safe(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
			{Name: "subtitle", Type: FieldTypeString}, // new, nullable
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addChanges := filterByType(changes, ChangeAddColumn)
	if len(addChanges) != 1 {
		t.Fatalf("expected 1 AddColumn change, got %d", len(addChanges))
	}

	c := addChanges[0]
	if !c.Safe {
		t.Error("adding nullable column should be safe")
	}
	if c.Column != "subtitle" {
		t.Errorf("column = %q, want %q", c.Column, "subtitle")
	}
	assertContains(t, c.SQL, `ALTER TABLE "ct_posts" ADD COLUMN "subtitle" TEXT`)
}

func TestDiffSchema_AddRequiredField_Breaking(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
			{Name: "slug", Type: FieldTypeString, Required: true}, // new, NOT NULL
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addChanges := filterByType(changes, ChangeAddColumn)
	if len(addChanges) != 1 {
		t.Fatalf("expected 1 AddColumn change, got %d", len(addChanges))
	}

	c := addChanges[0]
	if c.Safe {
		t.Error("adding NOT NULL column should be breaking")
	}
	assertContains(t, c.SQL, "NOT NULL")
	assertContains(t, c.Detail, "BREAKING")
}

func TestDiffSchema_RemoveField_Breaking(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
			{Name: "body", Type: FieldTypeText},
		},
	}

	changes := DiffSchema(loaded, &existing)

	dropChanges := filterByType(changes, ChangeDropColumn)
	if len(dropChanges) != 1 {
		t.Fatalf("expected 1 DropColumn change, got %d", len(dropChanges))
	}

	c := dropChanges[0]
	if c.Safe {
		t.Error("dropping column should be breaking")
	}
	if c.Column != "body" {
		t.Errorf("column = %q, want %q", c.Column, "body")
	}
	assertContains(t, c.SQL, `DROP COLUMN "body"`)
	assertContains(t, c.Detail, "BREAKING")
}

func TestDiffSchema_ChangeFieldType_Breaking(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "count", Type: FieldTypeFloat}, // was int, now float
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "count", Type: FieldTypeInt},
		},
	}

	changes := DiffSchema(loaded, &existing)

	alterChanges := filterByType(changes, ChangeAlterColumn)
	if len(alterChanges) != 1 {
		t.Fatalf("expected 1 AlterColumn change, got %d", len(alterChanges))
	}

	c := alterChanges[0]
	if c.Safe {
		t.Error("changing column type should be breaking")
	}
	if c.Column != "count" {
		t.Errorf("column = %q, want %q", c.Column, "count")
	}
	assertContains(t, c.SQL, `ALTER TABLE "ct_posts" ALTER COLUMN "count" TYPE DOUBLE PRECISION`)
	assertContains(t, c.Detail, "BREAKING")
}

func TestDiffSchema_AddUniqueConstraint_Safe(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "slug", Type: FieldTypeString, Unique: true},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "slug", Type: FieldTypeString, Unique: false},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addIdxChanges := filterByType(changes, ChangeAddIndex)
	if len(addIdxChanges) != 1 {
		t.Fatalf("expected 1 AddIndex change, got %d", len(addIdxChanges))
	}

	c := addIdxChanges[0]
	if !c.Safe {
		t.Error("adding unique index should be safe")
	}
	assertContains(t, c.SQL, "CREATE UNIQUE INDEX")
	assertContains(t, c.SQL, `"ct_posts"("slug")`)
}

func TestDiffSchema_RemoveUniqueConstraint_Safe(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "slug", Type: FieldTypeString, Unique: false},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "slug", Type: FieldTypeString, Unique: true},
		},
	}

	changes := DiffSchema(loaded, &existing)

	dropIdxChanges := filterByType(changes, ChangeDropIndex)
	if len(dropIdxChanges) != 1 {
		t.Fatalf("expected 1 DropIndex change, got %d", len(dropIdxChanges))
	}

	c := dropIdxChanges[0]
	if !c.Safe {
		t.Error("removing unique index should be safe")
	}
	assertContains(t, c.SQL, "DROP INDEX IF EXISTS")
}

func TestDiffSchema_AddSearchableField_UpdatesTrigger(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Searchable: true},
			{Name: "body", Type: FieldTypeText, Searchable: true}, // newly searchable
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Searchable: true},
			{Name: "body", Type: FieldTypeText, Searchable: false},
		},
	}

	changes := DiffSchema(loaded, &existing)

	// Should have a trigger update change.
	found := false
	for _, c := range changes {
		if c.Column == "search_vector" && strings.Contains(c.Detail, "search_vector trigger") {
			found = true
			assertContains(t, c.SQL, "search_update")
			assertContains(t, c.SQL, `coalesce(NEW."title",'')`)
			assertContains(t, c.SQL, `coalesce(NEW."body",'')`)
		}
	}
	if !found {
		t.Error("expected a search_vector trigger update change")
	}
}

func TestDiffSchema_RemoveAllSearchableFields_DropsSearch(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Searchable: false},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Searchable: true},
		},
	}

	changes := DiffSchema(loaded, &existing)

	found := false
	for _, c := range changes {
		if c.Column == "search_vector" && strings.Contains(c.Detail, "no searchable fields") {
			found = true
			assertContains(t, c.SQL, "DROP TRIGGER")
			assertContains(t, c.SQL, "DROP FUNCTION")
			assertContains(t, c.SQL, "DROP INDEX")
		}
	}
	if !found {
		t.Error("expected a change to remove search trigger and index")
	}
}

func TestDiffSchema_AddManyRelation_Safe(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "tags", Type: FieldTypeRelation, RelatesTo: "tags", RelationType: RelationMany},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addChanges := filterByType(changes, ChangeAddColumn)
	if len(addChanges) != 1 {
		t.Fatalf("expected 1 AddColumn change for junction table, got %d", len(addChanges))
	}

	c := addChanges[0]
	if !c.Safe {
		t.Error("adding many-to-many junction table should be safe")
	}
	assertContains(t, c.SQL, "ct_posts_tags_rel")
	assertContains(t, c.Detail, "junction table")
}

func TestDiffSchema_RemoveManyRelation_Breaking(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "tags", Type: FieldTypeRelation, RelatesTo: "tags", RelationType: RelationMany},
		},
	}

	changes := DiffSchema(loaded, &existing)

	dropChanges := filterByType(changes, ChangeDropColumn)
	if len(dropChanges) != 1 {
		t.Fatalf("expected 1 DropColumn change for junction table, got %d", len(dropChanges))
	}

	c := dropChanges[0]
	if c.Safe {
		t.Error("dropping junction table should be breaking")
	}
	assertContains(t, c.SQL, `DROP TABLE IF EXISTS "ct_posts_tags_rel"`)
}

func TestDiffSchema_AddMediaField_WithIndex(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "cover", Type: FieldTypeMedia},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addCols := filterByType(changes, ChangeAddColumn)
	if len(addCols) != 1 {
		t.Fatalf("expected 1 AddColumn, got %d", len(addCols))
	}
	assertContains(t, addCols[0].SQL, `UUID REFERENCES "media"("id")`)

	addIdx := filterByType(changes, ChangeAddIndex)
	if len(addIdx) != 1 {
		t.Fatalf("expected 1 AddIndex for FK, got %d", len(addIdx))
	}
	assertContains(t, addIdx[0].SQL, `"idx_ct_posts_cover"`)
}

func TestDiffSchema_AddRelationOneField_WithIndex(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "author", Type: FieldTypeRelation, RelatesTo: "authors", RelationType: RelationOne},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addCols := filterByType(changes, ChangeAddColumn)
	if len(addCols) != 1 {
		t.Fatalf("expected 1 AddColumn, got %d", len(addCols))
	}
	assertContains(t, addCols[0].SQL, `UUID REFERENCES "ct_authors"("id")`)

	addIdx := filterByType(changes, ChangeAddIndex)
	if len(addIdx) != 1 {
		t.Fatalf("expected 1 AddIndex for FK, got %d", len(addIdx))
	}
	assertContains(t, addIdx[0].SQL, `"idx_ct_posts_author"`)
}

func TestDiffSchema_MultipleChanges(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
			{Name: "subtitle", Type: FieldTypeString}, // new
			// body removed
			{Name: "count", Type: FieldTypeFloat}, // was int
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
			{Name: "body", Type: FieldTypeText}, // removed
			{Name: "count", Type: FieldTypeInt},  // type changed
		},
	}

	changes := DiffSchema(loaded, &existing)

	addCols := filterByType(changes, ChangeAddColumn)
	dropCols := filterByType(changes, ChangeDropColumn)
	alterCols := filterByType(changes, ChangeAlterColumn)

	if len(addCols) != 1 {
		t.Errorf("expected 1 AddColumn, got %d", len(addCols))
	}
	if len(dropCols) != 1 {
		t.Errorf("expected 1 DropColumn, got %d", len(dropCols))
	}
	if len(alterCols) != 1 {
		t.Errorf("expected 1 AlterColumn, got %d", len(alterCols))
	}
}

func TestDiffSchema_AddUniqueNewField_NoDuplicateIndex(t *testing.T) {
	// Fix 5: Adding a unique field should produce one AddColumn (without UNIQUE)
	// and one AddIndex (CREATE UNIQUE INDEX). No duplicate.
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "slug", Type: FieldTypeString, Unique: true},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	changes := DiffSchema(loaded, &existing)

	// Should have both an add column and an add index.
	addCols := filterByType(changes, ChangeAddColumn)
	addIdxs := filterByType(changes, ChangeAddIndex)

	if len(addCols) != 1 {
		t.Fatalf("expected 1 AddColumn, got %d", len(addCols))
	}
	if len(addIdxs) != 1 {
		t.Fatalf("expected 1 AddIndex for unique, got %d", len(addIdxs))
	}
	assertContains(t, addIdxs[0].SQL, "UNIQUE INDEX")

	// The AddColumn SQL must NOT contain UNIQUE (Fix 5).
	assertNotContains(t, addCols[0].SQL, "UNIQUE")
}

// ----- Fix 3: NOT NULL change tests -----

func TestDiffSchema_SetNotNull_Breaking(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true}, // was false
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: false},
		},
	}

	changes := DiffSchema(loaded, &existing)

	alterChanges := filterByType(changes, ChangeAlterColumn)
	if len(alterChanges) != 1 {
		t.Fatalf("expected 1 AlterColumn change for SET NOT NULL, got %d", len(alterChanges))
	}

	c := alterChanges[0]
	if c.Safe {
		t.Error("SET NOT NULL should be breaking")
	}
	if c.Column != "title" {
		t.Errorf("column = %q, want %q", c.Column, "title")
	}
	assertContains(t, c.SQL, `ALTER TABLE "ct_posts" ALTER COLUMN "title" SET NOT NULL`)
	assertContains(t, c.Detail, "BREAKING")
	assertContains(t, c.Detail, "set NOT NULL")
}

func TestDiffSchema_DropNotNull_Safe(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: false}, // was true
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
		},
	}

	changes := DiffSchema(loaded, &existing)

	alterChanges := filterByType(changes, ChangeAlterColumn)
	if len(alterChanges) != 1 {
		t.Fatalf("expected 1 AlterColumn change for DROP NOT NULL, got %d", len(alterChanges))
	}

	c := alterChanges[0]
	if !c.Safe {
		t.Error("DROP NOT NULL should be safe")
	}
	if c.Column != "title" {
		t.Errorf("column = %q, want %q", c.Column, "title")
	}
	assertContains(t, c.SQL, `ALTER TABLE "ct_posts" ALTER COLUMN "title" DROP NOT NULL`)
}

func TestDiffSchema_NotNull_Unchanged_NoChange(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
		},
	}

	existing := ct // identical

	changes := DiffSchema(ct, &existing)

	if len(changes) != 0 {
		t.Errorf("expected 0 changes for identical required, got %d:", len(changes))
		for _, c := range changes {
			t.Logf("  - %s: %s", c.Type, c.Detail)
		}
	}
}

// ----- Fix 8: Enum value change tests -----

func TestDiffSchema_EnumValueChange_DropAndAddConstraint(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "science"}},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "business"}},
		},
	}

	changes := DiffSchema(loaded, &existing)

	dropConstraints := filterByType(changes, ChangeDropConstraint)
	addConstraints := filterByType(changes, ChangeAddConstraint)

	if len(dropConstraints) != 1 {
		t.Fatalf("expected 1 DropConstraint, got %d", len(dropConstraints))
	}
	if len(addConstraints) != 1 {
		t.Fatalf("expected 1 AddConstraint, got %d", len(addConstraints))
	}

	// Drop old constraint.
	dc := dropConstraints[0]
	if !dc.Safe {
		t.Error("dropping CHECK constraint should be safe")
	}
	assertContains(t, dc.SQL, `DROP CONSTRAINT IF EXISTS "chk_ct_posts_category"`)
	assertContains(t, dc.Detail, "enum value change")

	// Add new constraint -- should be breaking because "business" was removed.
	ac := addConstraints[0]
	if ac.Safe {
		t.Error("adding CHECK constraint with removed values should be breaking")
	}
	assertContains(t, ac.SQL, `ADD CONSTRAINT "chk_ct_posts_category"`)
	assertContains(t, ac.SQL, `CHECK("category" IN ('tech','design','science'))`)
	assertContains(t, ac.Detail, "BREAKING")

	// There should be NO AlterColumn for type change (type stays TEXT).
	alterCols := filterByType(changes, ChangeAlterColumn)
	if len(alterCols) != 0 {
		t.Errorf("expected 0 AlterColumn for enum value change, got %d", len(alterCols))
		for _, c := range alterCols {
			t.Logf("  - %s", c.Detail)
		}
	}
}

func TestDiffSchema_EnumValuesIdentical_NoChange(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design"}},
		},
	}

	existing := ct

	changes := DiffSchema(ct, &existing)

	if len(changes) != 0 {
		t.Errorf("expected 0 changes for identical enum values, got %d:", len(changes))
		for _, c := range changes {
			t.Logf("  - %s: %s", c.Type, c.Detail)
		}
	}
}

func TestDiffSchema_EnumAddValue(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "new"}},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design"}},
		},
	}

	changes := DiffSchema(loaded, &existing)

	dropConstraints := filterByType(changes, ChangeDropConstraint)
	addConstraints := filterByType(changes, ChangeAddConstraint)

	if len(dropConstraints) != 1 {
		t.Fatalf("expected 1 DropConstraint for added enum value, got %d", len(dropConstraints))
	}
	if len(addConstraints) != 1 {
		t.Fatalf("expected 1 AddConstraint for added enum value, got %d", len(addConstraints))
	}
	assertContains(t, addConstraints[0].SQL, "'new'")

	// Purely additive change: all old values preserved, so ADD CONSTRAINT is safe.
	if !addConstraints[0].Safe {
		t.Error("adding enum values (widening) should be safe")
	}
}

func TestDiffSchema_EnumNarrowValues_Breaking(t *testing.T) {
	// Removing an enum value is breaking because existing rows may have it.
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech"}},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "business"}},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addConstraints := filterByType(changes, ChangeAddConstraint)
	if len(addConstraints) != 1 {
		t.Fatalf("expected 1 AddConstraint, got %d", len(addConstraints))
	}

	ac := addConstraints[0]
	if ac.Safe {
		t.Error("narrowing enum values should be breaking")
	}
	assertContains(t, ac.Detail, "BREAKING")
	assertContains(t, ac.SQL, `CHECK("category" IN ('tech'))`)

	// Drop constraint should always be safe.
	dropConstraints := filterByType(changes, ChangeDropConstraint)
	if len(dropConstraints) != 1 {
		t.Fatalf("expected 1 DropConstraint, got %d", len(dropConstraints))
	}
	if !dropConstraints[0].Safe {
		t.Error("dropping CHECK constraint should always be safe")
	}
}

func TestDiffSchema_EnumWidenValues_Safe(t *testing.T) {
	// Adding new enum values without removing any old ones is safe.
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "business", "science"}},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "business"}},
		},
	}

	changes := DiffSchema(loaded, &existing)

	addConstraints := filterByType(changes, ChangeAddConstraint)
	if len(addConstraints) != 1 {
		t.Fatalf("expected 1 AddConstraint, got %d", len(addConstraints))
	}

	ac := addConstraints[0]
	if !ac.Safe {
		t.Error("widening enum values (all old values preserved) should be safe")
	}
	assertContains(t, ac.SQL, `'science'`)
}

func TestDiffSchema_RemoveEnumField_DropsConstraint(t *testing.T) {
	loaded := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	existing := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "category", Type: FieldTypeEnum, Values: []string{"a", "b"}},
		},
	}

	changes := DiffSchema(loaded, &existing)

	dropCols := filterByType(changes, ChangeDropColumn)
	if len(dropCols) != 1 {
		t.Fatalf("expected 1 DropColumn, got %d", len(dropCols))
	}

	// The drop SQL should include dropping the named constraint first.
	assertContains(t, dropCols[0].SQL, `DROP CONSTRAINT IF EXISTS "chk_ct_posts_category"`)
	assertContains(t, dropCols[0].SQL, `DROP COLUMN "category"`)
}

// ----- Helpers -----

func filterByType(changes []Change, ct ChangeType) []Change {
	var result []Change
	for _, c := range changes {
		if c.Type == ct {
			result = append(result, c)
		}
	}
	return result
}
