package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/orderedmap"
)

var (
	//go:embed templates/client.tmpl
	rawClientTemplate string
	clientTemplate    = must(template.New("client").Parse(rawClientTemplate))

	//go:embed templates/post_request.tmpl
	rawPostRequestTemplate string
	postRequestTemplate    = must(template.New("post_request").Parse(rawPostRequestTemplate))
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: go-http-gen [options] <input-file>\n")
	fmt.Fprintf(os.Stderr, "\tgo-http-gen testdata/vendor-search.yaml\n")

	flag.PrintDefaults()
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)
	log.SetPrefix("go-http-gen: ")

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	raw, err := os.ReadFile(args[0])
	if err != nil {
		log.Fatalf("could not read file %q: %v", args[0], err)
	}

	cfg := datamodel.NewDocumentConfiguration()

	doc, err := libopenapi.NewDocumentWithConfiguration(raw, cfg)
	if err != nil {
		log.Fatalf("could not parse open api document: %v", err)
	}

	model, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		log.Fatalf("could prepare open api model: %v", errors.Join(errs...))
	}

	if model.Model.Paths == nil {
		log.Fatalf("document %q doesn't contain paths", args[0])
	}

	pairs := orderedmap.Iterate(context.Background(), model.Model.Paths.PathItems)
	for pair := range pairs {
		path := canonizePath(pair.Key())
		pitem := pair.Value()

		if pitem == nil {
			log.Printf("%q doesn't have items", path)
			continue
		}

		var buf bytes.Buffer
		clientTemplate.Execute(&buf, map[string]string{
			"ClientName": "VendorsClient",
		})

		if pitem.Post != nil {
			postRequestTemplate.Execute(&buf, map[string]string{
				"ClientName": "VendorsClient",
				"Name":       fmt.Sprintf("Post%s", path),
				"Path":       pair.Key(),
				"Request":    "Post" + path + "Request",
				"Response":   "Post" + path + "Response",
			})

		}

		fmt.Println(buf.String())
	}
}
