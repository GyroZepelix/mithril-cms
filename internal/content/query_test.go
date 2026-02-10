package content

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

func newRequest(query string) *http.Request {
	u, _ := url.Parse("http://example.com/api/posts?" + query)
	return &http.Request{URL: u}
}

var testCT = schema.ContentType{
	Name: "posts",
	Fields: []schema.Field{
		{Name: "title", Type: schema.FieldTypeString},
		{Name: "body", Type: schema.FieldTypeText},
		{Name: "category", Type: schema.FieldTypeEnum, Values: []string{"tech", "design"}},
	},
}

func TestParseQueryParams_Defaults(t *testing.T) {
	q, err := ParseQueryParams(newRequest(""), testCT)
	if err != nil {
		t.Fatal(err)
	}
	if q.Page != 1 {
		t.Errorf("page: got %d, want 1", q.Page)
	}
	if q.PerPage != 20 {
		t.Errorf("per_page: got %d, want 20", q.PerPage)
	}
	if q.Sort != "created_at" {
		t.Errorf("sort: got %q, want 'created_at'", q.Sort)
	}
	if q.Order != "desc" {
		t.Errorf("order: got %q, want 'desc'", q.Order)
	}
}

func TestParseQueryParams_CustomValues(t *testing.T) {
	q, err := ParseQueryParams(newRequest("page=3&per_page=50&sort=title&order=asc"), testCT)
	if err != nil {
		t.Fatal(err)
	}
	if q.Page != 3 {
		t.Errorf("page: got %d, want 3", q.Page)
	}
	if q.PerPage != 50 {
		t.Errorf("per_page: got %d, want 50", q.PerPage)
	}
	if q.Sort != "title" {
		t.Errorf("sort: got %q, want 'title'", q.Sort)
	}
	if q.Order != "asc" {
		t.Errorf("order: got %q, want 'asc'", q.Order)
	}
}

func TestParseQueryParams_PerPageClamped(t *testing.T) {
	q, err := ParseQueryParams(newRequest("per_page=500"), testCT)
	if err != nil {
		t.Fatal(err)
	}
	if q.PerPage != 100 {
		t.Errorf("per_page should be clamped to 100, got %d", q.PerPage)
	}
}

func TestParseQueryParams_InvalidPage(t *testing.T) {
	_, err := ParseQueryParams(newRequest("page=0"), testCT)
	if err == nil {
		t.Error("expected error for page=0")
	}

	_, err = ParseQueryParams(newRequest("page=abc"), testCT)
	if err == nil {
		t.Error("expected error for page=abc")
	}
}

func TestParseQueryParams_InvalidSort(t *testing.T) {
	_, err := ParseQueryParams(newRequest("sort=nonexistent"), testCT)
	if err == nil {
		t.Error("expected error for invalid sort field")
	}
}

func TestParseQueryParams_SystemSortColumns(t *testing.T) {
	for _, col := range []string{"created_at", "updated_at", "published_at", "status", "id"} {
		q, err := ParseQueryParams(newRequest("sort="+col), testCT)
		if err != nil {
			t.Errorf("system column %q should be valid sort field: %v", col, err)
		}
		if q.Sort != col {
			t.Errorf("sort: got %q, want %q", q.Sort, col)
		}
	}
}

func TestParseQueryParams_InvalidOrder(t *testing.T) {
	_, err := ParseQueryParams(newRequest("order=random"), testCT)
	if err == nil {
		t.Error("expected error for invalid order")
	}
}

func TestParseQueryParams_Filters(t *testing.T) {
	q, err := ParseQueryParams(newRequest("filter[category]=tech&filter[title]=hello"), testCT)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(q.Filters))
	}
	if q.Filters["category"] != "tech" {
		t.Errorf("category filter: got %q, want 'tech'", q.Filters["category"])
	}
	if q.Filters["title"] != "hello" {
		t.Errorf("title filter: got %q, want 'hello'", q.Filters["title"])
	}
}

func TestParseQueryParams_InvalidFilter(t *testing.T) {
	_, err := ParseQueryParams(newRequest("filter[evil]=value"), testCT)
	if err == nil {
		t.Error("expected error for invalid filter field")
	}
}

func TestParseQueryParams_Search(t *testing.T) {
	q, err := ParseQueryParams(newRequest("q=hello+world"), testCT)
	if err != nil {
		t.Fatal(err)
	}
	if q.Search != "hello world" {
		t.Errorf("search: got %q, want 'hello world'", q.Search)
	}
}
