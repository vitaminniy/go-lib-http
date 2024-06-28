package generator

import (
	"strings"
	"unicode"
)

func canonize(path string) string {
	sb := &strings.Builder{}
	sb.Grow(len(path))

	nextUpper := true

	for _, r := range path {
		if nextUpper {
			r = unicode.ToUpper(r)
			nextUpper = false
		}

		if r == '-' || r == '/' || r == '_' {
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
