package pslanes

import (
	"context"
	"sync"
)

// PSLanes provides per-Permission System serialization for access operations
type PSLanes struct {
	// createLanes provides per-PS serialization for create operations
	createLanes map[string]chan struct{}
	// deleteLanes provides per-PS serialization for delete operations
	deleteLanes map[string]chan struct{}
	mutex       sync.RWMutex
}

// NewPSLanes creates a new PSLanes instance
func NewPSLanes() *PSLanes {
	return &PSLanes{
		createLanes: make(map[string]chan struct{}),
		deleteLanes: make(map[string]chan struct{}),
	}
}

// WithCreateLane executes the given function with create lane serialization for the specified Permission System
func (p *PSLanes) WithCreateLane(ctx context.Context, psID string, fn func() error) error {
	lane := p.getCreateLane(psID)

	select {
	case lane <- struct{}{}:
		defer func() { <-lane }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WithDeleteLane executes the given function with delete lane serialization for the specified Permission System
func (p *PSLanes) WithDeleteLane(ctx context.Context, psID string, fn func() error) error {
	lane := p.getDeleteLane(psID)

	select {
	case lane <- struct{}{}:
		defer func() { <-lane }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getCreateLane returns the create lane for the specified Permission System (capacity=1 for serialization)
func (p *PSLanes) getCreateLane(psID string) chan struct{} {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if lane, exists := p.createLanes[psID]; exists {
		return lane
	}

	lane := make(chan struct{}, 1) // capacity=1 for serialization
	p.createLanes[psID] = lane
	return lane
}

// getDeleteLane returns the delete lane for the specified Permission System (capacity=1 for serialization)
func (p *PSLanes) getDeleteLane(psID string) chan struct{} {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if lane, exists := p.deleteLanes[psID]; exists {
		return lane
	}

	lane := make(chan struct{}, 1) // capacity=1 for serialization
	p.deleteLanes[psID] = lane
	return lane
}
