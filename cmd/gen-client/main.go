package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/orderedmap"
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

	fname, err := filepath.Abs(args[0])
	if err != nil {
		log.Fatalf("could not resolve file name: %v", err)
	}

	raw, err := os.ReadFile(fname)
	if err != nil {
		log.Fatalf("could not read file %q: %v", args[0], err)
	}

	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath = path.Dir(fname)
	cfg.AllowFileReferences = true
	cfg.AllowRemoteReferences = true
	cfg.ExtractRefsSequentially = true
	cfg.BundleInlineRefs = true

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

	ctx := context.Background()
	g := Generator{}
	client := "VendorsClient"

	if err = g.GenerateClient(client); err != nil {
		log.Fatalf("couldn't generate client: %v", err)
	}

	pairs := orderedmap.Iterate(ctx, model.Model.Paths.PathItems)
	for pair := range pairs {
		path := pair.Key()
		pitem := pair.Value()

		g.GenerateMethod(ctx, client, path, http.MethodPost, pitem.Post)
	}

	source, err := g.Source()
	if err != nil {
		log.Fatalf("could not generate client: %v", err)
	}

	fmt.Println(string(source))
}
