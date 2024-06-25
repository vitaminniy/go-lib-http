package config

import (
	"context"
	"sync"
	"time"
)

// NewSnapshot creates new config snapshot.
func NewSnapshot[T any](cfg T) *Snapshot[T] {
	return &Snapshot[T]{
		cfg: cfg,
	}
}

// Snapshot provides thread-safe access to service configuration.
type Snapshot[T any] struct {
	mu  sync.RWMutex
	cfg T
}

func (s *Snapshot[T]) Update(cfg T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg = cfg
}

func (s *Snapshot[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg
}

type QOS struct {
	Timeout time.Duration
}

func (q *QOS) Context(ctx context.Context) (context.Context, context.CancelFunc) {
	if q.Timeout == 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, q.Timeout)
}
