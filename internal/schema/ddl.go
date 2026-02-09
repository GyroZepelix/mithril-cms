package schema

import (
	"fmt"
	"strings"
)

// quoteIdent quotes a SQL identifier using double quotes, escaping any embedded
// double quotes by doubling them. This provides defense-in-depth even though the
// validator already restricts names to safe characters.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// fieldSQLBaseType returns ONLY the PostgreSQL base type for a given field,
// without any defaults, constraints, or references.
func fieldSQLBaseType(f Field) string {
	switch f.Type {
	case FieldTypeString:
		if f.MaxLength != nil {
			return fmt.Sprintf("VARCHAR(%d)", *f.MaxLength)
		}
		return "TEXT"
	case FieldTypeText, FieldTypeRichText:
		return "TEXT"
	case FieldTypeInt:
		return "INTEGER"
	case FieldTypeFloat:
		return "DOUBLE PRECISION"
	case FieldTypeBoolean:
		return "BOOLEAN"
	case FieldTypeDate:
		return "DATE"
	case FieldTypeTime:
		return "TIME"
	case FieldTypeEnum:
		return "TEXT"
	case FieldTypeJSON:
		return "JSONB"
	case FieldTypeMedia:
		return "UUID"
	case FieldTypeRelation:
		if f.RelationType == RelationOne {
			return "UUID"
		}
		// Many-to-many relations do not produce a column; handled by junction table.
		return ""
	default:
		return "TEXT"
	}
}

// fieldSQLDefault returns the DEFAULT clause for a field, or an empty string
// if the field has no default value.
func fieldSQLDefault(f Field) string {
	if f.Type == FieldTypeBoolean {
		return "DEFAULT false"
	}
	return ""
}

// fieldSQLConstraints returns the CHECK and/or REFERENCES clauses for a field,
// or an empty string if there are none. The tableName parameter is used to
// construct named CHECK constraints for enum fields.
func fieldSQLConstraints(f Field, tableName string) string {
	switch f.Type {
	case FieldTypeMedia:
		return fmt.Sprintf("REFERENCES %s(%s) ON DELETE SET NULL", quoteIdent("media"), quoteIdent("id"))
	case FieldTypeRelation:
		if f.RelationType == RelationOne {
			return fmt.Sprintf("REFERENCES %s(%s) ON DELETE SET NULL", quoteIdent("ct_"+f.RelatesTo), quoteIdent("id"))
		}
		return ""
	default:
		return ""
	}
}

// enumCheckConstraint returns a named CHECK constraint clause for an enum field.
// It is emitted as a separate table-level constraint, not inline with the column.
func enumCheckConstraint(tableName string, f Field) string {
	if f.Type != FieldTypeEnum || len(f.Values) == 0 {
		return ""
	}
	quoted := make([]string, len(f.Values))
	for i, v := range f.Values {
		quoted[i] = "'" + escapeSQLString(v) + "'"
	}
	constraintName := fmt.Sprintf("chk_%s_%s", tableName, f.Name)
	return fmt.Sprintf("CONSTRAINT %s CHECK(%s IN (%s))",
		quoteIdent(constraintName),
		quoteIdent(f.Name),
		strings.Join(quoted, ","))
}

// escapeSQLString escapes single quotes in a SQL string literal by doubling them.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// buildColumnDef builds the full column definition for use in CREATE TABLE,
// composing base type, default, NOT NULL, UNIQUE, and REFERENCES.
func buildColumnDef(f Field, tableName string) string {
	baseType := fieldSQLBaseType(f)
	if baseType == "" {
		return ""
	}

	parts := []string{quoteIdent(f.Name), baseType}

	if def := fieldSQLDefault(f); def != "" {
		parts = append(parts, def)
	}
	if constraint := fieldSQLConstraints(f, tableName); constraint != "" {
		parts = append(parts, constraint)
	}
	if f.Required {
		parts = append(parts, "NOT NULL")
	}
	if f.Unique {
		parts = append(parts, "UNIQUE")
	}

	return strings.Join(parts, " ")
}

// GenerateCreateTable generates the full CREATE TABLE statement, indexes,
// triggers, and junction tables for a content type. The returned SQL is ready
// to execute as a single batch (multiple statements separated by newlines).
func GenerateCreateTable(ct ContentType) string {
	var b strings.Builder
	tableName := "ct_" + ct.Name
	qTable := quoteIdent(tableName)

	// -- CREATE TABLE --
	b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", qTable))
	b.WriteString(fmt.Sprintf("    %s UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n", quoteIdent("id")))
	b.WriteString(fmt.Sprintf("    %s TEXT NOT NULL DEFAULT 'draft' CHECK(%s IN ('draft','published')),\n",
		quoteIdent("status"), quoteIdent("status")))

	// User-defined columns (skip many-relations, they get junction tables).
	var enumConstraints []string
	for _, f := range ct.Fields {
		if f.Type == FieldTypeRelation && f.RelationType == RelationMany {
			continue
		}
		colDef := buildColumnDef(f, tableName)
		if colDef == "" {
			continue
		}
		b.WriteString("    " + colDef + ",\n")

		// Collect enum CHECK constraints as named table constraints.
		if chk := enumCheckConstraint(tableName, f); chk != "" {
			enumConstraints = append(enumConstraints, chk)
		}
	}

	// System columns.
	b.WriteString(fmt.Sprintf("    %s TSVECTOR,\n", quoteIdent("search_vector")))
	b.WriteString(fmt.Sprintf("    %s UUID REFERENCES %s(%s),\n", quoteIdent("created_by"), quoteIdent("admins"), quoteIdent("id")))
	b.WriteString(fmt.Sprintf("    %s UUID REFERENCES %s(%s),\n", quoteIdent("updated_by"), quoteIdent("admins"), quoteIdent("id")))
	b.WriteString(fmt.Sprintf("    %s TIMESTAMPTZ NOT NULL DEFAULT now(),\n", quoteIdent("created_at")))
	b.WriteString(fmt.Sprintf("    %s TIMESTAMPTZ NOT NULL DEFAULT now(),\n", quoteIdent("updated_at")))
	b.WriteString(fmt.Sprintf("    %s TIMESTAMPTZ", quoteIdent("published_at")))

	// Emit named enum CHECK constraints as table-level constraints.
	for _, chk := range enumConstraints {
		b.WriteString(",\n    " + chk)
	}
	b.WriteString("\n);\n")

	// -- Standard indexes --
	b.WriteString(fmt.Sprintf("\nCREATE INDEX %s ON %s(%s);\n",
		quoteIdent("idx_"+tableName+"_status"), qTable, quoteIdent("status")))
	b.WriteString(fmt.Sprintf("CREATE INDEX %s ON %s(%s);\n",
		quoteIdent("idx_"+tableName+"_created_at"), qTable, quoteIdent("created_at")))

	// -- Search index (only if there are searchable fields) --
	searchableFields := collectSearchableFields(ct)
	if len(searchableFields) > 0 {
		b.WriteString(fmt.Sprintf("CREATE INDEX %s ON %s USING GIN(%s);\n",
			quoteIdent("idx_"+tableName+"_search"), qTable, quoteIdent("search_vector")))
	}

	// -- FK indexes for relation (one) and media fields --
	for _, f := range ct.Fields {
		if f.Type == FieldTypeMedia ||
			(f.Type == FieldTypeRelation && f.RelationType == RelationOne) {
			b.WriteString(fmt.Sprintf("CREATE INDEX %s ON %s(%s);\n",
				quoteIdent("idx_"+tableName+"_"+f.Name), qTable, quoteIdent(f.Name)))
		}
	}

	// -- updated_at trigger --
	b.WriteString(generateUpdatedAtTrigger(tableName))

	// -- Search trigger (only if there are searchable fields) --
	if len(searchableFields) > 0 {
		b.WriteString(generateSearchTrigger(tableName, searchableFields))
	}

	// -- Junction tables for many-to-many relations --
	for _, f := range ct.Fields {
		if f.Type == FieldTypeRelation && f.RelationType == RelationMany {
			b.WriteString(generateJunctionTable(ct.Name, f))
		}
	}

	return b.String()
}

// collectSearchableFields returns the names of all fields marked searchable.
func collectSearchableFields(ct ContentType) []string {
	var fields []string
	for _, f := range ct.Fields {
		if f.Searchable {
			fields = append(fields, f.Name)
		}
	}
	return fields
}

// generateUpdatedAtTrigger generates the shared trigger function (CREATE OR
// REPLACE is idempotent) and the per-table trigger that auto-updates
// updated_at on row updates.
func generateUpdatedAtTrigger(tableName string) string {
	qTable := quoteIdent(tableName)
	trigName := quoteIdent("trg_" + tableName + "_updated_at")
	funcName := quoteIdent("update_updated_at")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\nCREATE OR REPLACE FUNCTION %s() RETURNS trigger AS $$\n", funcName))
	b.WriteString("BEGIN\n")
	b.WriteString("    NEW.updated_at := now();\n")
	b.WriteString("    RETURN NEW;\n")
	b.WriteString("END $$ LANGUAGE plpgsql;\n")
	b.WriteString(fmt.Sprintf("\nCREATE TRIGGER %s\n", trigName))
	b.WriteString(fmt.Sprintf("    BEFORE UPDATE ON %s\n", qTable))
	b.WriteString(fmt.Sprintf("    FOR EACH ROW EXECUTE FUNCTION %s();\n", funcName))
	return b.String()
}

// generateSearchTrigger generates the PL/pgSQL function and trigger that
// automatically updates the search_vector column on INSERT or UPDATE.
func generateSearchTrigger(tableName string, searchableFields []string) string {
	parts := make([]string, len(searchableFields))
	for i, name := range searchableFields {
		parts[i] = fmt.Sprintf("coalesce(NEW.%s,'')", quoteIdent(name))
	}
	expr := strings.Join(parts, " || ' ' || ")

	funcName := quoteIdent(tableName + "_search_update")
	qTable := quoteIdent(tableName)
	trigName := quoteIdent("trg_" + tableName + "_search")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\nCREATE OR REPLACE FUNCTION %s() RETURNS trigger AS $$\n", funcName))
	b.WriteString("BEGIN\n")
	b.WriteString(fmt.Sprintf("    NEW.%s := to_tsvector('english', %s);\n", quoteIdent("search_vector"), expr))
	b.WriteString("    RETURN NEW;\n")
	b.WriteString("END $$ LANGUAGE plpgsql;\n")
	b.WriteString(fmt.Sprintf("\nCREATE TRIGGER %s\n", trigName))
	b.WriteString(fmt.Sprintf("    BEFORE INSERT OR UPDATE ON %s\n", qTable))
	b.WriteString(fmt.Sprintf("    FOR EACH ROW EXECUTE FUNCTION %s();\n", funcName))

	return b.String()
}

// generateJunctionTable generates a junction table for a many-to-many relation.
func generateJunctionTable(sourceName string, f Field) string {
	junctionTable := fmt.Sprintf("ct_%s_%s_rel", sourceName, f.Name)
	sourceTable := "ct_" + sourceName
	targetTable := "ct_" + f.RelatesTo

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\nCREATE TABLE %s (\n", quoteIdent(junctionTable)))
	b.WriteString(fmt.Sprintf("    %s UUID NOT NULL REFERENCES %s(%s) ON DELETE CASCADE,\n",
		quoteIdent("source_id"), quoteIdent(sourceTable), quoteIdent("id")))
	b.WriteString(fmt.Sprintf("    %s UUID NOT NULL REFERENCES %s(%s) ON DELETE CASCADE,\n",
		quoteIdent("target_id"), quoteIdent(targetTable), quoteIdent("id")))
	b.WriteString(fmt.Sprintf("    PRIMARY KEY (%s, %s)\n", quoteIdent("source_id"), quoteIdent("target_id")))
	b.WriteString(");\n")

	return b.String()
}

// GenerateDropTable generates the DROP TABLE statement for a content type,
// including any junction tables for many-to-many relations.
func GenerateDropTable(ct ContentType) string {
	var b strings.Builder
	tableName := "ct_" + ct.Name

	// Drop junction tables first (they reference the main table).
	for _, f := range ct.Fields {
		if f.Type == FieldTypeRelation && f.RelationType == RelationMany {
			junctionTable := fmt.Sprintf("ct_%s_%s_rel", ct.Name, f.Name)
			b.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", quoteIdent(junctionTable)))
		}
	}

	// Drop the trigger function (IF EXISTS to be safe).
	b.WriteString(fmt.Sprintf("DROP FUNCTION IF EXISTS %s() CASCADE;\n", quoteIdent(tableName+"_search_update")))

	// Drop the main table.
	b.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;\n", quoteIdent(tableName)))

	return b.String()
}

// GenerateAddColumn generates an ALTER TABLE ADD COLUMN statement for a field.
// The UNIQUE constraint is NOT included here; it is handled by the diff engine
// as a separate CREATE UNIQUE INDEX to avoid duplicate index creation.
func GenerateAddColumn(tableName string, f Field) string {
	if f.Type == FieldTypeRelation && f.RelationType == RelationMany {
		// Many-relations don't add columns; they add junction tables.
		return ""
	}
	baseType := fieldSQLBaseType(f)
	if baseType == "" {
		return ""
	}

	parts := []string{
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			quoteIdent(tableName), quoteIdent(f.Name), baseType),
	}

	if def := fieldSQLDefault(f); def != "" {
		parts = append(parts, def)
	}
	if constraint := fieldSQLConstraints(f, tableName); constraint != "" {
		parts = append(parts, constraint)
	}
	if f.Required {
		parts = append(parts, "NOT NULL")
	}

	// Note: UNIQUE is intentionally omitted here. The diff engine adds a
	// separate CREATE UNIQUE INDEX statement to avoid duplicating indexes.

	result := strings.Join(parts, " ") + ";"

	// For enum fields, also add the named CHECK constraint.
	if f.Type == FieldTypeEnum && len(f.Values) > 0 {
		quoted := make([]string, len(f.Values))
		for i, v := range f.Values {
			quoted[i] = "'" + escapeSQLString(v) + "'"
		}
		constraintName := fmt.Sprintf("chk_%s_%s", tableName, f.Name)
		result += fmt.Sprintf("\nALTER TABLE %s ADD CONSTRAINT %s CHECK(%s IN (%s));",
			quoteIdent(tableName),
			quoteIdent(constraintName),
			quoteIdent(f.Name),
			strings.Join(quoted, ","))
	}

	return result
}

// GenerateDropColumn generates an ALTER TABLE DROP COLUMN statement.
func GenerateDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", quoteIdent(tableName), quoteIdent(columnName))
}
