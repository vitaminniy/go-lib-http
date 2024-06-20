package config

import (
	"context"
	"sync"
	"time"

	"github.com/vitaminniy/go-lib-http/retry"
)

func NewSnapshot(cfg ServiceConfig) *Snapshot {
	return &Snapshot{
		svccfg: cfg,
	}
}

type Snapshot struct {
	mu     sync.RWMutex
	svccfg ServiceConfig
}

func (s *Snapshot) Update(cfg ServiceConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.svccfg = cfg
}

// QOS returns QOS configuration for the given endpoint.
func (s *Snapshot) QOS(name string) QOS {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if qos, ok := s.svccfg.Endpoints[name]; ok {
		return qos
	}

	return s.svccfg.Default
}

// ServiceConfig controls service configuration.
type ServiceConfig struct {
	Default   QOS
	Endpoints map[string]QOS
}

type QOS struct {
	Timeout time.Duration
	Retry   retry.Config
}

func (q *QOS) Context(ctx context.Context) (context.Context, context.CancelFunc) {
	if q.Timeout == 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, q.Timeout)
}
