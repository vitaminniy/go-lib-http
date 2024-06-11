package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"unicode"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/orderedmap"
)

var (
	src = flag.String("src", "", "source specification; must be set")
	dst = flag.String("dst", "", "output destination; empty will print to stdout")
)

var (
	//go:embed request.tmpl
	rawRequestTemplate string
	requestTemplate    = must(template.New("request").Parse(rawRequestTemplate))
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	log.SetPrefix("gen-client: ")

	flag.Parse()

	if *src == "" {
		log.Println("'src' flag is missing")
		flag.Usage()
		os.Exit(1)
	}

	raw, err := os.ReadFile(*src)
	if err != nil {
		log.Fatalf("could not read %q: %v\n", *src, err)
	}

	document, err := libopenapi.NewDocument(raw)
	if err != nil {
		log.Fatalf("could not open openapi spec: %v\n", err)
	}

	model, errs := document.BuildV3Model()
	if len(errs) != 0 {
		log.Fatalf("could not build openapi v3 model: %v\n", errors.Join(errs...))
	}

	kvs := orderedmap.Iterate(context.Background(), model.Model.Paths.PathItems)
	for pair := range kvs {
		name := canonizeName(pair.Key())

		pitem := pair.Value()
		if pitem.Post != nil {
			requestTemplate.Execute(os.Stdout, map[string]string{
				"ClientName": "VendorsClient",
				"Name":       fmt.Sprintf("Post%s", name),
				"Request":    "Post" + name + "Request",
				"Response":   "Post" + name + "Response",
			})
		}
	}
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

func canonizeName(name string) string {
	var sb strings.Builder
	sb.Grow(len(name))

	nextcap := false
	for _, r := range name {
		if r == '/' {
			nextcap = true
			continue
		}

		if nextcap {
			r = unicode.ToUpper(r)
			nextcap = false
		}

		sb.WriteRune(r)
	}

	return sb.String()
}
