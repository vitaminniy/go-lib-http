package main

import (
	"errors"
	"testing"
)

func TestCanonize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "capitalize",
			path: "foo",
			want: "Foo",
		},
		{
			name: "capitalize with dash",
			path: "foo-bar",
			want: "FooBar",
		},
		{
			name: "slash replace",
			path: "/api/v1/path",
			want: "ApiV1Path",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := canonize(c.path)
			if c.want != got {
				t.Fatalf("mismatch: want %q; got %q", c.want, got)
			}
		})
	}
}

func TestMust(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if err := recover(); err != nil {
				t.Fatalf("unhandled panic: %v", err)
			}
		}()

		const want = 1

		got := must(func() (int, error) {
			return want, nil
		}())

		if got != want {
			t.Fatalf("mismatch: want %d; got %d", want, got)
		}
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		want := errors.New("fail on purpose")

		defer func() {
			rerr := recover()
			if rerr == nil {
				t.Fatal("expected panic but got nil")
			}

			err, ok := rerr.(error)
			if !ok {
				t.Fatalf("expected error but got %T", rerr)
			}

			if !errors.Is(err, want) {
				t.Fatalf("mismatch: want %v; got %v", want, err)
			}
		}()

		_ = must(func() (int, error) {
			return 0, want
		}())
	})
}
