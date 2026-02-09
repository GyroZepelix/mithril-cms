package schema

import (
	"strings"
	"testing"
)

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", `"simple"`},
		{"with space", `"with space"`},
		{`has"quote`, `"has""quote"`},
		{"ct_posts", `"ct_posts"`},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := quoteIdent(tc.input)
			if got != tc.want {
				t.Errorf("quoteIdent(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFieldSQLBaseType_AllTypes(t *testing.T) {
	maxLen := 255
	tests := []struct {
		name  string
		field Field
		want  string
	}{
		{"string_no_max", Field{Name: "f", Type: FieldTypeString}, "TEXT"},
		{"string_with_max", Field{Name: "f", Type: FieldTypeString, MaxLength: &maxLen}, "VARCHAR(255)"},
		{"text", Field{Name: "f", Type: FieldTypeText}, "TEXT"},
		{"richtext", Field{Name: "f", Type: FieldTypeRichText}, "TEXT"},
		{"int", Field{Name: "f", Type: FieldTypeInt}, "INTEGER"},
		{"float", Field{Name: "f", Type: FieldTypeFloat}, "DOUBLE PRECISION"},
		{"boolean", Field{Name: "f", Type: FieldTypeBoolean}, "BOOLEAN"},
		{"date", Field{Name: "f", Type: FieldTypeDate}, "DATE"},
		{"time", Field{Name: "f", Type: FieldTypeTime}, "TIME"},
		{"enum", Field{Name: "f", Type: FieldTypeEnum, Values: []string{"a", "b"}}, "TEXT"},
		{"json", Field{Name: "f", Type: FieldTypeJSON}, "JSONB"},
		{"media", Field{Name: "f", Type: FieldTypeMedia}, "UUID"},
		{"relation_one", Field{Name: "f", Type: FieldTypeRelation, RelatesTo: "target", RelationType: RelationOne}, "UUID"},
		{"relation_many", Field{Name: "f", Type: FieldTypeRelation, RelatesTo: "target", RelationType: RelationMany}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fieldSQLBaseType(tc.field)
			if got != tc.want {
				t.Errorf("fieldSQLBaseType(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestFieldSQLDefault(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  string
	}{
		{"boolean", Field{Type: FieldTypeBoolean}, "DEFAULT false"},
		{"string", Field{Type: FieldTypeString}, ""},
		{"int", Field{Type: FieldTypeInt}, ""},
		{"text", Field{Type: FieldTypeText}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fieldSQLDefault(tc.field)
			if got != tc.want {
				t.Errorf("fieldSQLDefault(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestFieldSQLConstraints(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  string
	}{
		{"media", Field{Type: FieldTypeMedia}, `REFERENCES "media"("id") ON DELETE SET NULL`},
		{"relation_one", Field{Type: FieldTypeRelation, RelatesTo: "authors", RelationType: RelationOne},
			`REFERENCES "ct_authors"("id") ON DELETE SET NULL`},
		{"string", Field{Type: FieldTypeString}, ""},
		{"relation_many", Field{Type: FieldTypeRelation, RelationType: RelationMany}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fieldSQLConstraints(tc.field, "ct_posts")
			if got != tc.want {
				t.Errorf("fieldSQLConstraints(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestEnumCheckConstraint(t *testing.T) {
	f := Field{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design"}}
	got := enumCheckConstraint("ct_posts", f)
	want := `CONSTRAINT "chk_ct_posts_category" CHECK("category" IN ('tech','design'))`
	if got != want {
		t.Errorf("enumCheckConstraint() = %q, want %q", got, want)
	}
}

func TestEnumCheckConstraint_WithSingleQuote(t *testing.T) {
	f := Field{Name: "tag", Type: FieldTypeEnum, Values: []string{"it's", "fine"}}
	got := enumCheckConstraint("ct_posts", f)
	assertContains(t, got, `'it''s'`)
	assertContains(t, got, `'fine'`)
}

func TestEnumCheckConstraint_NonEnum(t *testing.T) {
	f := Field{Name: "title", Type: FieldTypeString}
	got := enumCheckConstraint("ct_posts", f)
	if got != "" {
		t.Errorf("expected empty for non-enum, got %q", got)
	}
}

func TestGenerateCreateTable_BasicFields(t *testing.T) {
	maxLen := 200
	ct := ContentType{
		Name:        "articles",
		DisplayName: "Articles",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true, MaxLength: &maxLen},
			{Name: "body", Type: FieldTypeText},
			{Name: "count", Type: FieldTypeInt},
			{Name: "rating", Type: FieldTypeFloat},
			{Name: "active", Type: FieldTypeBoolean},
			{Name: "pub_date", Type: FieldTypeDate},
			{Name: "pub_time", Type: FieldTypeTime},
			{Name: "metadata", Type: FieldTypeJSON},
		},
	}

	sql := GenerateCreateTable(ct)

	// Table structure.
	assertContains(t, sql, `CREATE TABLE "ct_articles" (`)
	assertContains(t, sql, `"id" UUID PRIMARY KEY DEFAULT gen_random_uuid()`)
	assertContains(t, sql, `"status" TEXT NOT NULL DEFAULT 'draft' CHECK("status" IN ('draft','published'))`)

	// Field types (quoted identifiers, separated concerns).
	assertContains(t, sql, `"title" VARCHAR(200) NOT NULL`)
	assertContains(t, sql, `"body" TEXT`)
	assertContains(t, sql, `"count" INTEGER`)
	assertContains(t, sql, `"rating" DOUBLE PRECISION`)
	assertContains(t, sql, `"active" BOOLEAN DEFAULT false`)
	assertContains(t, sql, `"pub_date" DATE`)
	assertContains(t, sql, `"pub_time" TIME`)
	assertContains(t, sql, `"metadata" JSONB`)

	// System columns.
	assertContains(t, sql, `"search_vector" TSVECTOR`)
	assertContains(t, sql, `"created_by" UUID REFERENCES "admins"("id")`)
	assertContains(t, sql, `"updated_by" UUID REFERENCES "admins"("id")`)
	assertContains(t, sql, `"created_at" TIMESTAMPTZ NOT NULL DEFAULT now()`)
	assertContains(t, sql, `"updated_at" TIMESTAMPTZ NOT NULL DEFAULT now()`)
	assertContains(t, sql, `"published_at" TIMESTAMPTZ`)

	// Standard indexes.
	assertContains(t, sql, `CREATE INDEX "idx_ct_articles_status" ON "ct_articles"("status")`)
	assertContains(t, sql, `CREATE INDEX "idx_ct_articles_created_at" ON "ct_articles"("created_at")`)

	// updated_at trigger.
	assertContains(t, sql, `CREATE OR REPLACE FUNCTION "update_updated_at"() RETURNS trigger`)
	assertContains(t, sql, "NEW.updated_at := now()")
	assertContains(t, sql, `CREATE TRIGGER "trg_ct_articles_updated_at"`)
	assertContains(t, sql, `BEFORE UPDATE ON "ct_articles"`)
	assertContains(t, sql, `FOR EACH ROW EXECUTE FUNCTION "update_updated_at"()`)
}

func TestGenerateCreateTable_StringWithoutMaxLength(t *testing.T) {
	ct := ContentType{
		Name:        "items",
		DisplayName: "Items",
		Fields: []Field{
			{Name: "description", Type: FieldTypeString},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `"description" TEXT`)
	// Should NOT contain VARCHAR.
	if strings.Contains(sql, "VARCHAR") {
		t.Errorf("string without max_length should use TEXT, not VARCHAR, got:\n%s", sql)
	}
}

func TestGenerateCreateTable_UniqueField(t *testing.T) {
	ct := ContentType{
		Name:        "pages",
		DisplayName: "Pages",
		Fields: []Field{
			{Name: "slug", Type: FieldTypeString, Required: true, Unique: true},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `"slug" TEXT NOT NULL UNIQUE`)
}

func TestGenerateCreateTable_EnumCheckConstraint(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "business"}},
		},
	}

	sql := GenerateCreateTable(ct)
	// Enum column should be plain TEXT.
	assertContains(t, sql, `"category" TEXT`)
	// Named CHECK constraint should be a table-level constraint.
	assertContains(t, sql, `CONSTRAINT "chk_ct_posts_category" CHECK("category" IN ('tech','design','business'))`)
}

func TestGenerateCreateTable_EnumWithSingleQuoteInValue(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "tag", Type: FieldTypeEnum, Values: []string{"it's", "fine"}},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `"tag" TEXT`)
	assertContains(t, sql, `CHECK("tag" IN ('it''s','fine'))`)
}

func TestGenerateCreateTable_MediaFK(t *testing.T) {
	ct := ContentType{
		Name:        "authors",
		DisplayName: "Authors",
		Fields: []Field{
			{Name: "avatar", Type: FieldTypeMedia},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `"avatar" UUID REFERENCES "media"("id") ON DELETE SET NULL`)
	assertContains(t, sql, `CREATE INDEX "idx_ct_authors_avatar" ON "ct_authors"("avatar")`)
}

func TestGenerateCreateTable_RelationOneFK(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "author", Type: FieldTypeRelation, RelatesTo: "authors", RelationType: RelationOne},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `"author" UUID REFERENCES "ct_authors"("id") ON DELETE SET NULL`)
	assertContains(t, sql, `CREATE INDEX "idx_ct_posts_author" ON "ct_posts"("author")`)
}

func TestGenerateCreateTable_RelationManyJunctionTable(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "tags", Type: FieldTypeRelation, RelatesTo: "tags", RelationType: RelationMany},
		},
	}

	sql := GenerateCreateTable(ct)

	// Many-to-many should NOT produce a column in the main table.
	if strings.Contains(sql, `"tags" UUID`) {
		t.Errorf("many-to-many field should not produce a column, got:\n%s", sql)
	}

	// Should produce a junction table.
	assertContains(t, sql, `CREATE TABLE "ct_posts_tags_rel" (`)
	assertContains(t, sql, `"source_id" UUID NOT NULL REFERENCES "ct_posts"("id") ON DELETE CASCADE`)
	assertContains(t, sql, `"target_id" UUID NOT NULL REFERENCES "ct_tags"("id") ON DELETE CASCADE`)
	assertContains(t, sql, `PRIMARY KEY ("source_id", "target_id")`)
}

func TestGenerateCreateTable_SearchTrigger_Present(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Searchable: true},
			{Name: "body", Type: FieldTypeRichText, Searchable: true},
			{Name: "slug", Type: FieldTypeString},
		},
	}

	sql := GenerateCreateTable(ct)

	// GIN index on search_vector.
	assertContains(t, sql, `CREATE INDEX "idx_ct_posts_search" ON "ct_posts" USING GIN("search_vector")`)

	// Trigger function.
	assertContains(t, sql, `CREATE OR REPLACE FUNCTION "ct_posts_search_update"() RETURNS trigger`)
	assertContains(t, sql, `to_tsvector('english', coalesce(NEW."title",'') || ' ' || coalesce(NEW."body",''))`)
	assertContains(t, sql, "RETURN NEW")

	// Trigger.
	assertContains(t, sql, `CREATE TRIGGER "trg_ct_posts_search"`)
	assertContains(t, sql, `BEFORE INSERT OR UPDATE ON "ct_posts"`)
	assertContains(t, sql, `FOR EACH ROW EXECUTE FUNCTION "ct_posts_search_update"()`)
}

func TestGenerateCreateTable_SearchTrigger_Absent(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
			{Name: "count", Type: FieldTypeInt},
		},
	}

	sql := GenerateCreateTable(ct)

	// No search-related artifacts.
	if strings.Contains(sql, `GIN("search_vector")`) {
		t.Errorf("no searchable fields: should not have GIN index, got:\n%s", sql)
	}
	if strings.Contains(sql, "search_update") {
		t.Errorf("no searchable fields: should not have search trigger, got:\n%s", sql)
	}
}

func TestGenerateCreateTable_SingleSearchableField(t *testing.T) {
	ct := ContentType{
		Name:        "items",
		DisplayName: "Items",
		Fields: []Field{
			{Name: "name", Type: FieldTypeString, Searchable: true},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `to_tsvector('english', coalesce(NEW."name",''))`)
	// Should NOT contain || ' ' || since there's only one field.
	triggerBody := extractBetween(sql, "to_tsvector(", ");")
	if strings.Contains(triggerBody, "||") {
		t.Errorf("single searchable field should not have || concatenation, got: %s", triggerBody)
	}
}

func TestGenerateCreateTable_RequiredNotNull(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true},
			{Name: "body", Type: FieldTypeText, Required: false},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `"title" TEXT NOT NULL`)

	// body should NOT have NOT NULL.
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		if strings.Contains(line, `"body" TEXT`) && strings.Contains(line, "NOT NULL") {
			t.Errorf("non-required field 'body' should not have NOT NULL, got: %s", line)
		}
	}
}

func TestGenerateCreateTable_RequiredBoolean(t *testing.T) {
	ct := ContentType{
		Name:        "items",
		DisplayName: "Items",
		Fields: []Field{
			{Name: "active", Type: FieldTypeBoolean, Required: true},
		},
	}

	sql := GenerateCreateTable(ct)
	assertContains(t, sql, `"active" BOOLEAN DEFAULT false NOT NULL`)
}

func TestGenerateCreateTable_FullBlogPosts(t *testing.T) {
	// Exercise a realistic schema matching the blog_posts.yaml example.
	maxLen := 200
	ct := ContentType{
		Name:        "blog_posts",
		DisplayName: "Blog Posts",
		PublicRead:  true,
		Fields: []Field{
			{Name: "title", Type: FieldTypeString, Required: true, Searchable: true, MaxLength: &maxLen},
			{Name: "slug", Type: FieldTypeString, Required: true, Unique: true, Regex: `^[a-z0-9]+(?:-[a-z0-9]+)*$`},
			{Name: "body", Type: FieldTypeRichText, Required: true, Searchable: true},
			{Name: "category", Type: FieldTypeEnum, Values: []string{"tech", "design", "business"}},
			{Name: "featured", Type: FieldTypeBoolean},
			{Name: "author", Type: FieldTypeRelation, RelatesTo: "authors", RelationType: RelationOne},
		},
	}

	sql := GenerateCreateTable(ct)

	assertContains(t, sql, `CREATE TABLE "ct_blog_posts" (`)
	assertContains(t, sql, `"title" VARCHAR(200) NOT NULL`)
	assertContains(t, sql, `"slug" TEXT NOT NULL UNIQUE`)
	assertContains(t, sql, `"body" TEXT NOT NULL`)
	assertContains(t, sql, `"category" TEXT`)
	assertContains(t, sql, `CONSTRAINT "chk_ct_blog_posts_category" CHECK("category" IN ('tech','design','business'))`)
	assertContains(t, sql, `"featured" BOOLEAN DEFAULT false`)
	assertContains(t, sql, `"author" UUID REFERENCES "ct_authors"("id") ON DELETE SET NULL`)
	assertContains(t, sql, `CREATE INDEX "idx_ct_blog_posts_author" ON "ct_blog_posts"("author")`)
	assertContains(t, sql, `to_tsvector('english', coalesce(NEW."title",'') || ' ' || coalesce(NEW."body",''))`)

	// updated_at trigger.
	assertContains(t, sql, `CREATE TRIGGER "trg_ct_blog_posts_updated_at"`)
	assertContains(t, sql, `BEFORE UPDATE ON "ct_blog_posts"`)
}

func TestGenerateCreateTable_UpdatedAtTrigger(t *testing.T) {
	ct := ContentType{
		Name:        "simple",
		DisplayName: "Simple",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	sql := GenerateCreateTable(ct)

	// Shared trigger function.
	assertContains(t, sql, `CREATE OR REPLACE FUNCTION "update_updated_at"() RETURNS trigger AS $$`)
	assertContains(t, sql, "NEW.updated_at := now()")
	assertContains(t, sql, "RETURN NEW")

	// Per-table trigger.
	assertContains(t, sql, `CREATE TRIGGER "trg_ct_simple_updated_at"`)
	assertContains(t, sql, `BEFORE UPDATE ON "ct_simple"`)
	assertContains(t, sql, `FOR EACH ROW EXECUTE FUNCTION "update_updated_at"()`)
}

func TestGenerateDropTable_Basic(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "title", Type: FieldTypeString},
		},
	}

	sql := GenerateDropTable(ct)
	assertContains(t, sql, `DROP FUNCTION IF EXISTS "ct_posts_search_update"() CASCADE`)
	assertContains(t, sql, `DROP TABLE IF EXISTS "ct_posts" CASCADE`)
}

func TestGenerateDropTable_WithJunctionTable(t *testing.T) {
	ct := ContentType{
		Name:        "posts",
		DisplayName: "Posts",
		Fields: []Field{
			{Name: "tags", Type: FieldTypeRelation, RelatesTo: "tags", RelationType: RelationMany},
		},
	}

	sql := GenerateDropTable(ct)
	assertContains(t, sql, `DROP TABLE IF EXISTS "ct_posts_tags_rel"`)
	assertContains(t, sql, `DROP TABLE IF EXISTS "ct_posts" CASCADE`)

	// Junction table should be dropped before main table.
	junctionIdx := strings.Index(sql, "ct_posts_tags_rel")
	mainIdx := strings.Index(sql, `DROP TABLE IF EXISTS "ct_posts" CASCADE`)
	if junctionIdx > mainIdx {
		t.Error("junction table should be dropped before main table")
	}
}

func TestGenerateAddColumn_Basic(t *testing.T) {
	sql := GenerateAddColumn("ct_posts", Field{Name: "subtitle", Type: FieldTypeString})
	assertContains(t, sql, `ALTER TABLE "ct_posts" ADD COLUMN "subtitle" TEXT;`)
}

func TestGenerateAddColumn_RequiredField(t *testing.T) {
	sql := GenerateAddColumn("ct_posts", Field{Name: "title", Type: FieldTypeString, Required: true})
	assertContains(t, sql, `ALTER TABLE "ct_posts" ADD COLUMN "title" TEXT NOT NULL;`)
}

func TestGenerateAddColumn_NoUnique(t *testing.T) {
	// Fix 5: UNIQUE should NOT be in GenerateAddColumn; handled by diff engine.
	sql := GenerateAddColumn("ct_posts", Field{Name: "slug", Type: FieldTypeString, Unique: true})
	if strings.Contains(sql, "UNIQUE") {
		t.Errorf("GenerateAddColumn should not include UNIQUE, got: %s", sql)
	}
}

func TestGenerateAddColumn_EnumWithCheckConstraint(t *testing.T) {
	f := Field{Name: "category", Type: FieldTypeEnum, Values: []string{"a", "b", "c"}}
	sql := GenerateAddColumn("ct_posts", f)
	assertContains(t, sql, `ALTER TABLE "ct_posts" ADD COLUMN "category" TEXT;`)
	assertContains(t, sql, `ALTER TABLE "ct_posts" ADD CONSTRAINT "chk_ct_posts_category" CHECK("category" IN ('a','b','c'));`)
}

func TestGenerateAddColumn_MediaField(t *testing.T) {
	f := Field{Name: "cover", Type: FieldTypeMedia}
	sql := GenerateAddColumn("ct_posts", f)
	assertContains(t, sql, `ALTER TABLE "ct_posts" ADD COLUMN "cover" UUID REFERENCES "media"("id") ON DELETE SET NULL;`)
}

func TestGenerateAddColumn_RelationOneField(t *testing.T) {
	f := Field{Name: "author", Type: FieldTypeRelation, RelatesTo: "authors", RelationType: RelationOne}
	sql := GenerateAddColumn("ct_posts", f)
	assertContains(t, sql, `ALTER TABLE "ct_posts" ADD COLUMN "author" UUID REFERENCES "ct_authors"("id") ON DELETE SET NULL;`)
}

func TestGenerateAddColumn_ManyRelation(t *testing.T) {
	sql := GenerateAddColumn("ct_posts", Field{
		Name:         "tags",
		Type:         FieldTypeRelation,
		RelatesTo:    "tags",
		RelationType: RelationMany,
	})
	// Many-to-many does not produce an ALTER TABLE ADD COLUMN.
	if sql != "" {
		t.Errorf("expected empty SQL for many-relation ADD COLUMN, got: %s", sql)
	}
}

func TestGenerateAddColumn_BooleanWithDefault(t *testing.T) {
	f := Field{Name: "active", Type: FieldTypeBoolean}
	sql := GenerateAddColumn("ct_posts", f)
	assertContains(t, sql, `ALTER TABLE "ct_posts" ADD COLUMN "active" BOOLEAN DEFAULT false;`)
}

func TestGenerateDropColumn_Basic(t *testing.T) {
	sql := GenerateDropColumn("ct_posts", "subtitle")
	want := `ALTER TABLE "ct_posts" DROP COLUMN "subtitle";`
	if sql != want {
		t.Errorf("unexpected DROP COLUMN SQL: %s, want: %s", sql, want)
	}
}

// ----- Helpers -----

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected SQL to contain %q, but it does not.\nFull SQL:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected SQL NOT to contain %q, but it does.\nFull SQL:\n%s", needle, haystack)
	}
}

// extractBetween returns the substring between the first occurrence of start and end.
func extractBetween(s, start, end string) string {
	si := strings.Index(s, start)
	if si == -1 {
		return ""
	}
	s = s[si+len(start):]
	ei := strings.Index(s, end)
	if ei == -1 {
		return s
	}
	return s[:ei]
}
