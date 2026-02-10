// Package content implements the Content CRUD API for the Mithril CMS,
// including HTTP handlers, business logic, dynamic SQL generation, and
// content validation against YAML-defined schemas.
package content

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

// QueryParams holds parsed and validated query parameters for list endpoints.
type QueryParams struct {
	Page    int
	PerPage int
	Sort    string
	Order   string            // "asc" or "desc"
	Filters map[string]string // field name -> value
	Search  string            // full-text search query (Task 8)
}

// systemSortColumns are columns that exist on every content table and are
// valid sort targets in addition to user-defined fields.
var systemSortColumns = map[string]bool{
	"id":           true,
	"status":       true,
	"created_at":   true,
	"updated_at":   true,
	"published_at": true,
	"created_by":   true,
	"updated_by":   true,
}

// ParseQueryParams extracts and validates query parameters from the request
// URL against the given content type schema.
func ParseQueryParams(r *http.Request, ct schema.ContentType) (QueryParams, error) {
	q := QueryParams{
		Page:    1,
		PerPage: 20,
		Sort:    "created_at",
		Order:   "desc",
		Filters: make(map[string]string),
	}

	query := r.URL.Query()

	// Parse page.
	if v := query.Get("page"); v != "" {
		page, err := strconv.Atoi(v)
		if err != nil || page < 1 {
			return q, fmt.Errorf("page must be a positive integer")
		}
		q.Page = page
	}

	// Parse per_page.
	if v := query.Get("per_page"); v != "" {
		perPage, err := strconv.Atoi(v)
		if err != nil || perPage < 1 {
			return q, fmt.Errorf("per_page must be a positive integer")
		}
		if perPage > 100 {
			perPage = 100
		}
		q.PerPage = perPage
	}

	// Build field name lookup for validation.
	fieldNames := make(map[string]bool, len(ct.Fields))
	for _, f := range ct.Fields {
		fieldNames[f.Name] = true
	}

	// Parse sort.
	if v := query.Get("sort"); v != "" {
		if !fieldNames[v] && !systemSortColumns[v] {
			return q, fmt.Errorf("invalid sort field: %s", v)
		}
		q.Sort = v
	}

	// Parse order.
	if v := query.Get("order"); v != "" {
		lower := strings.ToLower(v)
		if lower != "asc" && lower != "desc" {
			return q, fmt.Errorf("order must be 'asc' or 'desc'")
		}
		q.Order = lower
	}

	// Parse filters: filter[field_name]=value.
	for key, values := range query {
		if !strings.HasPrefix(key, "filter[") || !strings.HasSuffix(key, "]") {
			continue
		}
		fieldName := key[len("filter[") : len(key)-1]
		if !fieldNames[fieldName] && !systemSortColumns[fieldName] {
			return q, fmt.Errorf("invalid filter field: %s", fieldName)
		}
		if len(values) > 0 {
			q.Filters[fieldName] = values[0]
		}
	}

	// Parse search query (captured here, implemented in Task 8).
	q.Search = query.Get("q")

	return q, nil
}
