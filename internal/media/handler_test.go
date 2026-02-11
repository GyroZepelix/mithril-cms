package media

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantPage    int
		wantPerPage int
	}{
		{"defaults", "", 1, 20},
		{"custom page", "page=3", 3, 20},
		{"custom per_page", "per_page=50", 1, 50},
		{"both", "page=2&per_page=10", 2, 10},
		{"invalid page", "page=abc", 1, 20},
		{"negative page", "page=-1", 1, 20},
		{"zero page", "page=0", 1, 20},
		{"per_page over max", "per_page=200", 1, 100},
		{"per_page zero", "per_page=0", 1, 20},
		{"per_page negative", "per_page=-5", 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)
			page, perPage := parsePagination(r)
			if page != tt.wantPage {
				t.Errorf("page = %d, want %d", page, tt.wantPage)
			}
			if perPage != tt.wantPerPage {
				t.Errorf("perPage = %d, want %d", perPage, tt.wantPerPage)
			}
		})
	}
}
