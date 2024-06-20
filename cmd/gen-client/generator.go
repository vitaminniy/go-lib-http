package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"go/format"
	"slices"
	"strings"
	"text/template"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

const contentType = "application/json"

var (
	//go:embed templates/client.tmpl
	rawClientTemplate string
	clientTemplate    = mustparse("client", rawClientTemplate)

	//go:embed templates/post_request.tmpl
	rawPostRequestTemplate string
	postRequestTemplate    = mustparse("post_request", rawPostRequestTemplate)
)

func mustparse(name, tmpl string) *template.Template {
	return must(template.New(name).Parse(tmpl))
}

type Request struct {
	Name string
	Body *RequestBody
}

type RequestBody struct {
	Name       string
	Required   bool
	Properties []Property
}

type Response struct {
	Name string
}

type Property struct {
	Name string
	Type string
	Tag  string
}

type Generator struct {
	buf bytes.Buffer
}

func (g *Generator) GenerateClient(name string) error {
	return clientTemplate.Execute(&g.buf, map[string]string{
		"ClientName": name,
	})
}

func (g *Generator) GenerateMethod(
	ctx context.Context,
	client, path, method string,
	op *v3.Operation,
) error {
	if op == nil {
		return nil
	}

	request := Request{
		Name: method + canonizePath(path) + "Request",
	}

	if op.RequestBody != nil {
		reqbody := &RequestBody{}
		reqbody.Name = method + canonizePath(path) + "RequestBody"
		reqbody.Required = resolveptr(op.RequestBody.Required)

		media := op.RequestBody.Content.GetOrZero(contentType)
		reqbody.Properties = collectProperties(ctx, media, reqbody.Name)

		request.Body = reqbody
	}

	parameters := make(map[string]any)
	parameters["Client"] = client
	parameters["Path"] = path
	parameters["Method"] = method + canonizePath(path)
	parameters["Response"] = Response{
		Name: method + canonizePath(path) + "Response",
	}

	parameters["Request"] = request

	return postRequestTemplate.Execute(&g.buf, parameters)
}

func (g *Generator) Source() ([]byte, error) {
	source, err := format.Source(g.buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not format source: %w", err)
	}

	return source, nil
}

func resolveptr[T any](ptr *T) T {
	var val T

	if ptr == nil {
		return val
	}

	return *ptr
}

func collectProperties(
	ctx context.Context,
	media *v3.MediaType,
	parentName string,
) []Property {
	if media == nil || media.Schema == nil {
		return nil
	}

	schema := media.Schema.Schema()

	result := make([]Property, 0)
	properties := orderedmap.Iterate(ctx, schema.Properties)

	for property := range properties {
		key := property.Key()
		value := property.Value().Schema()
		required := slices.Contains(schema.Required, key)

		typ := "any"

		switch ttyp := value.Type[0]; ttyp {
		case "integer":
			typ = "int64"
		case "string":
			typ = "string"
		case "boolean":
			typ = "bool"
		case "object":
			// NOTE(max): it's a hack and might panic; please do something
			// about it.
			// TODO(max): generate struct for inline property.
			reference := value.ParentProxy.GetReference()
			if reference == "" {
				reference = parentName + canonizePath(key)
			}

			splits := strings.Split(reference, "/")
			typ = splits[len(splits)-1]
		case "array":
			// NOTE(max): it's a hack and might panic; please do something
			// about it.
			// TODO(max): generate struct for inline property.
			itemsSchema := value.Items.A.Schema()
			reference := itemsSchema.ParentProxy.GetReference()
			if reference == "" {
				reference = parentName + canonizePath(key)
			}

			splits := strings.Split(reference, "/")
			typ = "[]" + splits[len(splits)-1]
		}

		tag := key
		if !required {
			tag = tag + ",omitempty"
		}

		result = append(result, Property{
			Name: canonizePath(key),
			Type: typ,
			Tag:  tag,
		})
	}

	return result
}
