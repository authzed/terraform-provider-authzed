package deletelanes

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// DeleteLanes manages per-Permission System delete serialization (capacity=1)
type DeleteLanes struct {
	lanes map[string]*semaphore.Weighted
	mu    sync.RWMutex
}

// NewDeleteLanes creates a new delete lanes manager
func NewDeleteLanes() *DeleteLanes {
	return &DeleteLanes{
		lanes: make(map[string]*semaphore.Weighted),
	}
}

// WithDelete executes fn with delete serialization for the given Permission System
func (l *DeleteLanes) WithDelete(ctx context.Context, psID string, fn func(context.Context) error) error {
	// Get or create semaphore for this Permission System
	l.mu.RLock()
	lane, exists := l.lanes[psID]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		// Double-check after acquiring write lock
		if lane, exists = l.lanes[psID]; !exists {
			lane = semaphore.NewWeighted(1) // Capacity 1 for serialization
			l.lanes[psID] = lane
		}
		l.mu.Unlock()
	}

	// Acquire delete lane for this Permission System
	if err := lane.Acquire(ctx, 1); err != nil {
		return err
	}
	defer lane.Release(1)

	return fn(ctx)
}

// WithCreate executes fn with create serialization for the given Permission System
// This prevents FGAM conflicts when creating multiple resources simultaneously
func (l *DeleteLanes) WithCreate(ctx context.Context, psID string, fn func(context.Context) error) error {
	// Reuse the same serialization mechanism as deletes
	return l.WithDelete(ctx, psID, fn)
}
