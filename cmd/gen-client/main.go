package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
)

var (
	clientName = flag.String("client-name", "", "name of the generated client; name will be canonized; must be set")
	output     = flag.String("output", "", "output file name; if not set, stdout will be used")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: go-http-gen [options] <input-file>\n")
	fmt.Fprintf(os.Stderr, "\tgo-http-gen -client-name ExampleService spec.yaml\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")

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

	if *clientName == "" {
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
	client := canonize(*clientName)

	if err = g.GenerateClient(client, os.Args); err != nil {
		log.Fatalf("couldn't generate client: %v", err)
	}

	if err = g.GenerateComponents(ctx, model.Model.Components); err != nil {
		log.Fatalf("couldn't generate components: %v", err)
	}

	if err = g.GenerateMethods(ctx, client, model.Model.Paths); err != nil {
		log.Fatalf("couldn't generate methods: %v", err)
	}

	source, err := g.Source()
	if err != nil {
		log.Fatalf("could not generate client: %v", err)
	}

	if *output == "" {
		fmt.Println(string(source))
		return
	}

	if err = os.WriteFile(*output, source, 0644); err != nil {
		log.Fatalf("could not wirte file %v", err)
	}
}
