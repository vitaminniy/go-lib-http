package main

import (
	"strings"
	"unicode"
)

func canonizePath(path string) string {
	sb := &strings.Builder{}
	sb.Grow(len(path))

	nextUpper := false

	for _, r := range path {
		if nextUpper {
			r = unicode.ToUpper(r)
			nextUpper = false
		}

		if r == '-' || r == '/' {
			nextUpper = true
			continue
		}

		_, _ = sb.WriteRune(r)
	}

	return sb.String()
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}
