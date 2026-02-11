package schemaapi

import (
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

func TestNewHandler(t *testing.T) {
	schemaMap := map[string]schema.ContentType{
		"posts": {Name: "posts", DisplayName: "Blog Posts"},
	}

	h := NewHandler(nil, "./schema", schemaMap, nil, nil)

	if h.engine != nil {
		t.Error("expected nil engine")
	}
	if h.schemaDir != "./schema" {
		t.Errorf("expected schemaDir ./schema, got %s", h.schemaDir)
	}
	if h.audit != nil {
		t.Error("expected nil audit service")
	}
	if h.onRefresh != nil {
		t.Error("expected nil onRefresh")
	}

	got := h.SchemaMap()
	if len(got) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(got))
	}
	if got["posts"].DisplayName != "Blog Posts" {
		t.Errorf("unexpected display name: %s", got["posts"].DisplayName)
	}
}

func TestSchemaMapConcurrentAccess(t *testing.T) {
	schemaMap := map[string]schema.ContentType{
		"posts": {Name: "posts"},
	}

	h := NewHandler(nil, "./schema", schemaMap, nil, nil)

	// Simulate concurrent reads and a write.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			_ = h.SchemaMap()
		}
	}()

	// Simulate a schema map update.
	newMap := map[string]schema.ContentType{
		"posts":    {Name: "posts"},
		"articles": {Name: "articles"},
	}
	h.mu.Lock()
	h.schemaMap = newMap
	h.mu.Unlock()

	<-done

	got := h.SchemaMap()
	if len(got) != 2 {
		t.Errorf("expected 2 schemas after update, got %d", len(got))
	}
}
