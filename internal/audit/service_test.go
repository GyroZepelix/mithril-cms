package audit

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestLog_NonBlocking(t *testing.T) {
	// Create a service with no repo (won't start background goroutine).
	// Fill the channel completely, then verify Log does not block.
	s := &Service{
		eventCh: make(chan Event, 2),
		done:    make(chan struct{}),
	}

	// Fill the channel.
	s.eventCh <- Event{Action: "test.one"}
	s.eventCh <- Event{Action: "test.two"}

	// This should not block — the event should be dropped.
	done := make(chan struct{})
	go func() {
		s.Log(context.Background(), Event{Action: "test.dropped"})
		close(done)
	}()

	select {
	case <-done:
		// Good, Log returned without blocking.
	case <-time.After(1 * time.Second):
		t.Fatal("Log blocked when channel was full")
	}

	// Verify channel still has exactly 2 events.
	if len(s.eventCh) != 2 {
		t.Fatalf("expected 2 events in channel, got %d", len(s.eventCh))
	}

	// Verify dropped count incremented.
	if s.DroppedCount() != 1 {
		t.Fatalf("expected dropped count 1, got %d", s.DroppedCount())
	}
}

func TestShutdown_AlwaysWaitsForDone(t *testing.T) {
	// Create a service that simulates a slow drain.
	s := &Service{
		eventCh: make(chan Event),
		done:    make(chan struct{}),
	}

	// Start a goroutine that closes done after a delay.
	go func() {
		time.Sleep(200 * time.Millisecond)
		close(s.done)
	}()

	// Create a context that expires immediately.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Shutdown should still wait for done even though context expires.
	start := time.Now()
	s.Shutdown(ctx)
	elapsed := time.Since(start)

	// Verify we waited at least 200ms (not just the 1ms context timeout).
	if elapsed < 200*time.Millisecond {
		t.Errorf("Shutdown returned too quickly (%v), did not wait for done channel", elapsed)
	}
}

func TestNullIfEmpty(t *testing.T) {
	tests := []struct {
		input string
		isNil bool
	}{
		{"", true},
		{"abc", false},
		{"00000000-0000-0000-0000-000000000000", false},
	}

	for _, tt := range tests {
		result := nullIfEmpty(tt.input)
		if tt.isNil && result != nil {
			t.Errorf("nullIfEmpty(%q) = %v, want nil", tt.input, result)
		}
		if !tt.isNil {
			if result == nil {
				t.Errorf("nullIfEmpty(%q) = nil, want non-nil", tt.input)
			} else if *result != tt.input {
				t.Errorf("nullIfEmpty(%q) = %q, want %q", tt.input, *result, tt.input)
			}
		}
	}
}

func TestNullableJSON(t *testing.T) {
	if nullableJSON(nil) != nil {
		t.Error("nullableJSON(nil) should return nil")
	}
	if nullableJSON([]byte{}) != nil {
		t.Error("nullableJSON(empty) should return nil")
	}
	data := []byte(`{"key":"value"}`)
	result := nullableJSON(data)
	if result == nil {
		t.Error("nullableJSON(non-empty) should not return nil")
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		wantPage    int
		wantPerPage int
	}{
		{"defaults", "", 1, 20},
		{"custom page", "page=3", 3, 20},
		{"custom per_page", "per_page=50", 1, 50},
		{"both", "page=2&per_page=10", 2, 10},
		{"invalid page", "page=-1", 1, 20},
		{"invalid per_page", "per_page=abc", 1, 20},
		{"per_page capped", "per_page=200", 1, 100},
		{"zero page", "page=0", 1, 20},
		{"zero per_page", "per_page=0", 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/audit-log?"+tt.queryString, nil)
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
