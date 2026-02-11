package audit

import (
	"context"
	"log/slog"
	"sync/atomic"
)

const (
	// eventChannelSize is the buffer size for the async event channel.
	// If the channel is full, events are dropped with a warning log.
	eventChannelSize = 256
)

// Event represents an audit event to be logged.
type Event struct {
	Action     string         // e.g., "entry.create", "admin.login.success"
	ActorID    string         // admin UUID (can be empty for login failures)
	Resource   string         // e.g., "blog_posts", "media"
	ResourceID string         // UUID of affected resource
	Payload    map[string]any // additional context data
}

// Service provides asynchronous audit logging. Events are sent to a buffered
// channel and written to the database by a background goroutine, ensuring
// that audit logging never blocks or fails API requests.
type Service struct {
	repo         *Repository
	eventCh      chan Event
	done         chan struct{}
	droppedCount atomic.Uint64 // count of events dropped due to full channel
}

// NewService creates a new audit Service with the given repository.
// Call Start() to begin processing events, and Shutdown() to drain and stop.
func NewService(repo *Repository) *Service {
	return &Service{
		repo:    repo,
		eventCh: make(chan Event, eventChannelSize),
		done:    make(chan struct{}),
	}
}

// Log sends an audit event for asynchronous persistence. It never blocks the
// caller. If the internal channel is full, the event is dropped and a warning
// is logged.
func (s *Service) Log(ctx context.Context, event Event) {
	select {
	case s.eventCh <- event:
		// Event queued successfully.
	default:
		dropped := s.droppedCount.Add(1)
		slog.Warn("audit event channel full, dropping event",
			"action", event.Action,
			"actor_id", event.ActorID,
			"resource", event.Resource,
			"resource_id", event.ResourceID,
			"total_dropped", dropped,
		)
	}
}

// Start begins the background goroutine that reads events from the channel
// and writes them to the database. Must be called once after NewService.
func (s *Service) Start() {
	go s.processEvents()
}

// Shutdown signals the background goroutine to stop, drains any remaining
// events in the channel, and waits for completion. The provided context
// controls the maximum time to wait; if the context times out, a warning is
// logged, but Shutdown always waits for the background goroutine to finish
// to prevent race conditions with database writes.
func (s *Service) Shutdown(ctx context.Context) {
	close(s.eventCh)

	select {
	case <-s.done:
		slog.Info("audit service shutdown complete")
	case <-ctx.Done():
		slog.Warn("audit service shutdown timeout, still waiting for drain")
		<-s.done // Always wait for completion even if context times out
	}
}

// processEvents is the background goroutine that reads events from the channel
// and inserts them into the database. It runs until the channel is closed
// (via Shutdown), then drains any remaining buffered events before signalling
// completion on the done channel.
func (s *Service) processEvents() {
	defer close(s.done)

	for event := range s.eventCh {
		s.writeEvent(event)
	}
}

// writeEvent inserts a single event into the database. Errors are logged but
// never propagated, so a database failure cannot break the caller.
func (s *Service) writeEvent(event Event) {
	// Use a background context because the original request context may
	// already be cancelled by the time we process the event.
	ctx := context.Background()

	if err := s.repo.Insert(ctx, event); err != nil {
		slog.Error("failed to write audit event",
			"action", event.Action,
			"actor_id", event.ActorID,
			"resource", event.Resource,
			"resource_id", event.ResourceID,
			"error", err,
		)
	}
}

// DroppedCount returns the total number of events dropped since service start.
// This metric can be used for monitoring and alerting.
func (s *Service) DroppedCount() uint64 {
	return s.droppedCount.Load()
}

// List retrieves a paginated, filtered list of audit entries ordered by
// created_at DESC. It returns the entries, total count, and any error.
func (s *Service) List(ctx context.Context, filters AuditFilters, page, perPage int) ([]*AuditEntry, int, error) {
	return s.repo.List(ctx, filters, page, perPage)
}
