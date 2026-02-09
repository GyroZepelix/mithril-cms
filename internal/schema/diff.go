package schema

import (
	"fmt"
	"slices"
	"strings"
)

// ChangeType describes the kind of schema change detected between a loaded
// content type and its existing state in the database.
type ChangeType string

// Supported change types.
const (
	ChangeCreateTable    ChangeType = "create_table"
	ChangeAddColumn      ChangeType = "add_column"
	ChangeDropColumn     ChangeType = "drop_column"
	ChangeAlterColumn    ChangeType = "alter_column"
	ChangeAddIndex       ChangeType = "add_index"
	ChangeDropIndex      ChangeType = "drop_index"
	ChangeAddConstraint  ChangeType = "add_constraint"
	ChangeDropConstraint ChangeType = "drop_constraint"
)

// Change represents a single schema change with its SQL and safety classification.
type Change struct {
	// Type is the kind of change (create_table, add_column, etc.).
	Type ChangeType

	// Table is the target table name (e.g., "ct_blog_posts").
	Table string

	// Column is the affected column name, if applicable.
	Column string

	// SQL is the DDL statement to execute this change.
	SQL string

	// Safe indicates whether this change can be auto-applied without data loss.
	// Safe changes: add nullable column, add index, create new table, drop NOT NULL.
	// Breaking changes: drop column, change type, add NOT NULL column, set NOT NULL.
	Safe bool

	// Detail is a human-readable description of the change.
	Detail string
}

// DiffSchema compares a loaded content type against its existing state in the
// database. If existing is nil, the schema is new and a ChangeCreateTable is
// returned. Otherwise, fields are compared to detect additions, removals,
// type changes, nullability changes, and enum value changes.
func DiffSchema(loaded ContentType, existing *ContentType) []Change {
	tableName := "ct_" + loaded.Name

	// New content type: generate full CREATE TABLE.
	if existing == nil {
		return []Change{{
			Type:   ChangeCreateTable,
			Table:  tableName,
			SQL:    GenerateCreateTable(loaded),
			Safe:   true,
			Detail: fmt.Sprintf("create new table %s", tableName),
		}}
	}

	var changes []Change

	// Build field maps for comparison.
	existingFields := make(map[string]Field, len(existing.Fields))
	for _, f := range existing.Fields {
		existingFields[f.Name] = f
	}

	loadedFields := make(map[string]Field, len(loaded.Fields))
	for _, f := range loaded.Fields {
		loadedFields[f.Name] = f
	}

	// Detect new fields (in loaded but not in existing).
	for _, f := range loaded.Fields {
		if _, exists := existingFields[f.Name]; exists {
			continue
		}

		if f.Type == FieldTypeRelation && f.RelationType == RelationMany {
			// Many-to-many: create junction table instead of column.
			junctionSQL := generateJunctionTable(loaded.Name, f)
			changes = append(changes, Change{
				Type:   ChangeAddColumn,
				Table:  tableName,
				Column: f.Name,
				SQL:    junctionSQL,
				Safe:   true,
				Detail: fmt.Sprintf("add many-to-many junction table for field %q", f.Name),
			})
			continue
		}

		addSQL := GenerateAddColumn(tableName, f)
		if addSQL == "" {
			continue
		}

		// Adding a nullable column is safe. Adding a NOT NULL column is breaking
		// because existing rows would violate the constraint.
		safe := !f.Required
		detail := fmt.Sprintf("add column %s.%s (%s)", tableName, f.Name, fieldSQLBaseType(f))
		if !safe {
			detail += " [BREAKING: NOT NULL on existing table]"
		}

		changes = append(changes, Change{
			Type:   ChangeAddColumn,
			Table:  tableName,
			Column: f.Name,
			SQL:    addSQL,
			Safe:   safe,
			Detail: detail,
		})

		// If the new field has a unique constraint, add the index change.
		if f.Unique {
			idxName := fmt.Sprintf("idx_%s_%s_unique", tableName, f.Name)
			idxSQL := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s(%s);",
				quoteIdent(idxName), quoteIdent(tableName), quoteIdent(f.Name))
			changes = append(changes, Change{
				Type:   ChangeAddIndex,
				Table:  tableName,
				Column: f.Name,
				SQL:    idxSQL,
				Safe:   true,
				Detail: fmt.Sprintf("add unique index on %s.%s", tableName, f.Name),
			})
		}

		// If the new field is a FK (relation one or media), add an FK index.
		if f.Type == FieldTypeMedia || (f.Type == FieldTypeRelation && f.RelationType == RelationOne) {
			fkIdxName := fmt.Sprintf("idx_%s_%s", tableName, f.Name)
			fkIdxSQL := fmt.Sprintf("CREATE INDEX %s ON %s(%s);",
				quoteIdent(fkIdxName), quoteIdent(tableName), quoteIdent(f.Name))
			changes = append(changes, Change{
				Type:   ChangeAddIndex,
				Table:  tableName,
				Column: f.Name,
				SQL:    fkIdxSQL,
				Safe:   true,
				Detail: fmt.Sprintf("add FK index on %s.%s", tableName, f.Name),
			})
		}
	}

	// Detect removed fields (in existing but not in loaded).
	for _, f := range existing.Fields {
		if _, exists := loadedFields[f.Name]; exists {
			continue
		}

		if f.Type == FieldTypeRelation && f.RelationType == RelationMany {
			// Drop junction table.
			junctionTable := fmt.Sprintf("ct_%s_%s_rel", loaded.Name, f.Name)
			changes = append(changes, Change{
				Type:   ChangeDropColumn,
				Table:  tableName,
				Column: f.Name,
				SQL:    fmt.Sprintf("DROP TABLE IF EXISTS %s;", quoteIdent(junctionTable)),
				Safe:   false,
				Detail: fmt.Sprintf("drop junction table %s for removed field %q [BREAKING: data loss]", junctionTable, f.Name),
			})
			continue
		}

		// If the dropped field was an enum, also drop its named CHECK constraint.
		dropSQL := GenerateDropColumn(tableName, f.Name)
		if f.Type == FieldTypeEnum && len(f.Values) > 0 {
			constraintName := fmt.Sprintf("chk_%s_%s", tableName, f.Name)
			dropSQL = fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n",
				quoteIdent(tableName), quoteIdent(constraintName)) + dropSQL
		}

		changes = append(changes, Change{
			Type:   ChangeDropColumn,
			Table:  tableName,
			Column: f.Name,
			SQL:    dropSQL,
			Safe:   false,
			Detail: fmt.Sprintf("drop column %s.%s [BREAKING: data loss]", tableName, f.Name),
		})
	}

	// Detect type changes, nullability changes, and enum value changes for
	// fields that exist in both loaded and existing.
	for _, lf := range loaded.Fields {
		ef, exists := existingFields[lf.Name]
		if !exists {
			continue
		}

		// Check if the base SQL type changed (using separated base type comparison).
		loadedBase := fieldSQLBaseType(lf)
		existingBase := fieldSQLBaseType(ef)
		if loadedBase != existingBase {
			// For many-to-many relations, the SQL type is empty; compare relation types.
			if lf.Type == FieldTypeRelation && lf.RelationType == RelationMany &&
				ef.Type == FieldTypeRelation && ef.RelationType == RelationMany {
				// Both are many-to-many with same column (none) -- check if target changed.
				if lf.RelatesTo != ef.RelatesTo {
					changes = append(changes, Change{
						Type:   ChangeAlterColumn,
						Table:  tableName,
						Column: lf.Name,
						SQL:    "", // Complex migration needed; not auto-generated.
						Safe:   false,
						Detail: fmt.Sprintf("change relation target for %s.%s from %q to %q [BREAKING]", tableName, lf.Name, ef.RelatesTo, lf.RelatesTo),
					})
				}
				continue
			}

			changes = append(changes, Change{
				Type:   ChangeAlterColumn,
				Table:  tableName,
				Column: lf.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
					quoteIdent(tableName), quoteIdent(lf.Name), loadedBase),
				Safe: false,
				Detail: fmt.Sprintf("change type of %s.%s from %s to %s [BREAKING]",
					tableName, lf.Name, existingBase, loadedBase),
			})
		}

		// Check enum value changes (type stays enum but values differ).
		if lf.Type == FieldTypeEnum && ef.Type == FieldTypeEnum {
			if !slices.Equal(lf.Values, ef.Values) {
				changes = append(changes, diffEnumValues(tableName, ef, lf)...)
			}
		}

		// Check required (NOT NULL) changes.
		if lf.Required != ef.Required {
			if lf.Required && !ef.Required {
				// Adding NOT NULL: breaking change.
				changes = append(changes, Change{
					Type:  ChangeAlterColumn,
					Table: tableName, Column: lf.Name,
					SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
						quoteIdent(tableName), quoteIdent(lf.Name)),
					Safe:   false,
					Detail: fmt.Sprintf("set NOT NULL on %s.%s [BREAKING: existing NULLs will fail]", tableName, lf.Name),
				})
			} else {
				// Removing NOT NULL: safe change.
				changes = append(changes, Change{
					Type:  ChangeAlterColumn,
					Table: tableName, Column: lf.Name,
					SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
						quoteIdent(tableName), quoteIdent(lf.Name)),
					Safe:   true,
					Detail: fmt.Sprintf("drop NOT NULL on %s.%s", tableName, lf.Name),
				})
			}
		}

		// Check unique constraint changes.
		if lf.Unique && !ef.Unique {
			idxName := fmt.Sprintf("idx_%s_%s_unique", tableName, lf.Name)
			changes = append(changes, Change{
				Type:   ChangeAddIndex,
				Table:  tableName,
				Column: lf.Name,
				SQL: fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s(%s);",
					quoteIdent(idxName), quoteIdent(tableName), quoteIdent(lf.Name)),
				Safe:   true,
				Detail: fmt.Sprintf("add unique index on %s.%s", tableName, lf.Name),
			})
		}
		if !lf.Unique && ef.Unique {
			idxName := fmt.Sprintf("idx_%s_%s_unique", tableName, lf.Name)
			changes = append(changes, Change{
				Type:   ChangeDropIndex,
				Table:  tableName,
				Column: lf.Name,
				SQL:    fmt.Sprintf("DROP INDEX IF EXISTS %s;", quoteIdent(idxName)),
				Safe:   true,
				Detail: fmt.Sprintf("drop unique index on %s.%s", tableName, lf.Name),
			})
		}
	}

	// Check if searchable fields changed -- may need trigger update.
	oldSearchable := collectSearchableFields(*existing)
	newSearchable := collectSearchableFields(loaded)
	if !slices.Equal(oldSearchable, newSearchable) {
		if len(newSearchable) > 0 {
			triggerSQL := generateSearchTrigger(tableName, newSearchable)
			// Also ensure GIN index exists.
			ginSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN(%s);\n",
				quoteIdent("idx_"+tableName+"_search"), quoteIdent(tableName), quoteIdent("search_vector"))
			changes = append(changes, Change{
				Type:   ChangeAlterColumn,
				Table:  tableName,
				Column: "search_vector",
				SQL:    ginSQL + triggerSQL,
				Safe:   true,
				Detail: "update search_vector trigger for changed searchable fields",
			})
		} else if len(oldSearchable) > 0 && len(newSearchable) == 0 {
			// Remove trigger and index.
			dropSQL := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;\n",
				quoteIdent("trg_"+tableName+"_search"), quoteIdent(tableName))
			dropSQL += fmt.Sprintf("DROP FUNCTION IF EXISTS %s();\n", quoteIdent(tableName+"_search_update"))
			dropSQL += fmt.Sprintf("DROP INDEX IF EXISTS %s;\n", quoteIdent("idx_"+tableName+"_search"))
			changes = append(changes, Change{
				Type:   ChangeDropIndex,
				Table:  tableName,
				Column: "search_vector",
				SQL:    dropSQL,
				Safe:   true,
				Detail: "remove search_vector trigger and index (no searchable fields)",
			})
		}
	}

	return changes
}

// diffEnumValues generates changes to update enum CHECK constraints when the
// allowed values change but the field type remains enum. Instead of ALTER
// COLUMN TYPE (which cannot carry an inline CHECK), we drop the old named
// CHECK constraint and add a new one.
//
// The ADD CONSTRAINT is marked as breaking (Safe: false) when any old enum
// values are removed, because existing rows may contain those values.
// Purely additive changes (all old values preserved) are safe.
func diffEnumValues(tableName string, existing, loaded Field) []Change {
	constraintName := fmt.Sprintf("chk_%s_%s", tableName, loaded.Name)

	quoted := make([]string, len(loaded.Values))
	for i, v := range loaded.Values {
		quoted[i] = "'" + escapeSQLString(v) + "'"
	}

	dropSQL := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;",
		quoteIdent(tableName), quoteIdent(constraintName))
	addSQL := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK(%s IN (%s));",
		quoteIdent(tableName),
		quoteIdent(constraintName),
		quoteIdent(loaded.Name),
		strings.Join(quoted, ","))

	// Determine if any old values were removed. If so, the new constraint
	// is breaking because existing rows may contain the removed values.
	newSet := make(map[string]bool, len(loaded.Values))
	for _, v := range loaded.Values {
		newSet[v] = true
	}
	addSafe := true
	for _, v := range existing.Values {
		if !newSet[v] {
			addSafe = false
			break
		}
	}

	addDetail := fmt.Sprintf("add new CHECK constraint on %s.%s with updated enum values", tableName, loaded.Name)
	if !addSafe {
		addDetail += " [BREAKING: enum values removed]"
	}

	return []Change{
		{
			Type:   ChangeDropConstraint,
			Table:  tableName,
			Column: loaded.Name,
			SQL:    dropSQL,
			Safe:   true,
			Detail: fmt.Sprintf("drop old CHECK constraint on %s.%s for enum value change", tableName, loaded.Name),
		},
		{
			Type:   ChangeAddConstraint,
			Table:  tableName,
			Column: loaded.Name,
			SQL:    addSQL,
			Safe:   addSafe,
			Detail: addDetail,
		},
	}
}

