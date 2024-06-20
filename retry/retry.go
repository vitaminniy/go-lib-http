// Package retry provides facitlities to retry failed operations.
package retry

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// Config controls retry behaviour.
type Config struct {
	// Attempts is a number of times operation should attempt to execute
	// successfully before returning.
	Attempts uint
	// Backoff is a backoff interval between attempts.
	Backoff time.Duration
	// Jitter is a random interval added to backoff: rand(0, jitter) + backoff.
	// Jitter is not applied when backoff set to 0.
	Jitter time.Duration
}

// attempts returns number of attempts.
func (cfg *Config) attempts() uint {
	if cfg.Attempts == 0 {
		return 1
	}

	return cfg.Attempts
}

// backoff returns exponential backoff interval with applied jitter.
func (cfg *Config) backoff(attempt uint) time.Duration {
	if cfg.Backoff == 0 {
		return 0
	}

	backoff := cfg.Backoff.Nanoseconds()

	if cfg.Jitter != 0 {
		// NOTE(max): seeding rand every time to avoid races; doing it here is
		// fine since it's not costly for stdrand.
		//
		//nolint:gosec // We're not doing any security-related stuff here.
		rand := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
		jitter := int64(rand.Float64() * float64(cfg.Jitter))
		backoff += jitter
	}

	// NOTE(max): attempt always start from zero but we want one for
	// multiplication.
	attempt++

	return time.Duration(backoff * int64(attempt))
}

// OnError retries operation on any occurred error.
//
//nolint:varnamelen // op is a common name for passed functions.
func OnError(
	ctx context.Context,
	cfg Config,
	op func(context.Context) error,
) (err error) {
	var (
		attempt  uint
		attempts = cfg.attempts()
	)

	for ; attempt < attempts; attempt++ {
		if err = op(ctx); err == nil {
			return nil
		}

		backoff := cfg.backoff(attempt)
		if backoff == 0 {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err() //nolint:wrapcheck // We want to have actual context error.
		case <-time.After(backoff):
		}
	}

	return fmt.Errorf("retry: all attempts failed: %w", err)
}
