// Package search provides PostgreSQL full-text search query building for
// content entries in the Mithril CMS.
package search

import (
	"fmt"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

// BuildSearchClause generates PostgreSQL full-text search SQL fragments.
// It takes the search query string, list of searchable fields,
// and the starting parameter index (for $N placeholders).
//
// Returns:
//   - whereClause: e.g., "search_vector" @@ plainto_tsquery('english', $3)
//   - orderClause: e.g., ts_rank("search_vector", plainto_tsquery('english', $3)) DESC
//   - headlineExpr: e.g., ts_headline('english', "title", plainto_tsquery('english', $3)) AS "_search_headline"
//   - args: the query string value(s) to bind
//
// If no searchable fields exist, all return values are zero/nil (search not available).
func BuildSearchClause(query string, searchableFields []schema.Field, paramIdx int) (whereClause, orderClause, headlineExpr string, args []any) {
	if len(searchableFields) == 0 {
		return "", "", "", nil
	}

	tsquery := fmt.Sprintf("plainto_tsquery('english', $%d)", paramIdx)
	searchVec := schema.QuoteIdent("search_vector")

	whereClause = fmt.Sprintf("%s @@ %s", searchVec, tsquery)
	orderClause = fmt.Sprintf("ts_rank(%s, %s) DESC", searchVec, tsquery)

	// Use the first searchable field for the headline snippet.
	firstField := schema.QuoteIdent(searchableFields[0].Name)
	headlineExpr = fmt.Sprintf("ts_headline('english', %s, %s) AS %s",
		firstField, tsquery, schema.QuoteIdent("_search_headline"))

	args = []any{query}
	return whereClause, orderClause, headlineExpr, args
}
