package schema

import (
	"testing"
)

func TestRefreshResultTypes(t *testing.T) {
	// Verify RefreshResult struct can be instantiated and fields are accessible.
	result := RefreshResult{
		Applied: []Change{
			{Type: ChangeAddColumn, Table: "ct_posts", Column: "summary", Safe: true},
		},
		Breaking: []Change{
			{Type: ChangeDropColumn, Table: "ct_posts", Column: "old", Safe: false},
		},
		NewTypes:     []string{"articles"},
		UpdatedTypes: []string{"posts"},
	}

	if len(result.Applied) != 1 {
		t.Errorf("expected 1 applied change, got %d", len(result.Applied))
	}
	if len(result.Breaking) != 1 {
		t.Errorf("expected 1 breaking change, got %d", len(result.Breaking))
	}
	if len(result.NewTypes) != 1 || result.NewTypes[0] != "articles" {
		t.Errorf("unexpected NewTypes: %v", result.NewTypes)
	}
	if len(result.UpdatedTypes) != 1 || result.UpdatedTypes[0] != "posts" {
		t.Errorf("unexpected UpdatedTypes: %v", result.UpdatedTypes)
	}
}
