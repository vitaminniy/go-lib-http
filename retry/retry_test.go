package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

var ErrOperationFailed = errors.New("operation failed")

func compareErrors(t *testing.T, want, got error) {
	t.Helper()

	if want == nil && got == nil {
		return
	}

	if want == nil && got != nil {
		t.Fatalf("unexpected error: %v", got)
	}

	if want != nil && got == nil {
		t.Fatalf("expected %v but got nil", want)
	}

	if !errors.Is(got, want) {
		t.Fatalf("error mismatch: want %v; got %v", want, got)
	}
}

type operation struct {
	calls        atomic.Int32
	successAfter int32
}

func (op *operation) do(context.Context) error {
	call := op.calls.Add(1)
	if call > op.successAfter {
		return nil
	}

	return ErrOperationFailed
}

type onErrorTestCase struct {
	name         string
	config       Config
	successAfter int32
	calls        int32
	err          error
}

//nolint:gochecknoglobals // List of test cases; Having in function triggers funlen.
var onErrorTestCases = []onErrorTestCase{
	{
		name:         "default-constructed config does not retry on fail",
		config:       Config{},
		successAfter: 1_000_000,
		calls:        1,
		err:          ErrOperationFailed,
	},
	{
		name:         "default-constructed config does not retry after success",
		config:       Config{},
		successAfter: 0,
		calls:        1,
		err:          nil,
	},
	{
		name: "do not retry more than configured",
		config: Config{
			Attempts: 2,
		},
		successAfter: 2,
		calls:        2,
		err:          ErrOperationFailed,
	},
	{
		name: "eventually return on success",
		config: Config{
			Attempts: 2,
		},
		successAfter: 1,
		calls:        2,
		err:          nil,
	},
}

func TestOnError(t *testing.T) {
	t.Parallel()

	for _, testCase := range onErrorTestCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			op := operation{} //nolint:varnamelen
			op.successAfter = testCase.successAfter

			err := OnError(context.Background(), testCase.config, op.do)
			compareErrors(t, testCase.err, err)

			calls := op.calls.Load()
			if testCase.calls != calls {
				t.Fatalf("calls mismatch: want %d; got %d", testCase.calls, calls)
			}
		})
	}
}

func TestOnErrorContextError(t *testing.T) {
	t.Parallel()

	alwaysfail := operation{successAfter: 1_000_000}
	config := Config{
		Attempts: 1_000_000,
		Backoff:  time.Second * 1,
		Jitter:   time.Millisecond * 10,
	}

	t.Run("context canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			<-time.After(time.Second * 1)
			cancel()
		}()

		err := OnError(ctx, config, alwaysfail.do)
		compareErrors(t, context.Canceled, err)
	})

	t.Run("context timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		err := OnError(ctx, config, alwaysfail.do)
		compareErrors(t, context.DeadlineExceeded, err)
	})
}

func TestOnErrorBackoffSuceeded(t *testing.T) {
	t.Parallel()

	var (
		attempts  = uint(5)
		backoff   = time.Second * 2
		now       = time.Now()
		operation = func(context.Context) error {
			elapsed := time.Since(now)
			if elapsed > backoff {
				return nil
			}

			return errors.New("not ready yet")
		}
	)

	defer func() {
		elapsed := time.Since(now)
		if elapsed < backoff {
			t.Fatalf("spent less than expected: min %q; got %q", backoff, elapsed)
		}

		max := backoff * time.Duration(attempts)
		if elapsed > max {
			t.Fatalf("spent more than expected: max %q; got %q", max, elapsed)
		}
	}()

	config := Config{
		Attempts: attempts,
		Backoff:  backoff,
		Jitter:   time.Millisecond * 10,
	}

	err := OnError(context.Background(), config, operation)
	if err != nil {
		t.Fatalf("could not retry op: %v", err)
	}
}
