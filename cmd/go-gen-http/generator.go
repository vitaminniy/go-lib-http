package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"go/format"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

const contentType = "application/json"

var (
	//go:embed templates/client.tmpl
	rawClientTemplate string
	clientTemplate    = mustparse("client", rawClientTemplate)

	//go:embed templates/components.tmpl
	rawComponentsTemplate string
	componentsTemplate    = mustparse("components", rawComponentsTemplate)

	//go:embed templates/config.tmpl
	rawConfigTemplate string
	configTemplate    = mustparse("config", rawConfigTemplate)

	//go:embed templates/request.tmpl
	rawRequestTemplate string
	requestTemplate    = mustparse("request", rawRequestTemplate)
)

func mustparse(name, tmpl string) *template.Template {
	return must(template.New(name).Parse(tmpl))
}

type Header struct {
	Name string
	Key  string
}

type QueryParam struct {
	Name     string
	Key      string
	Required bool
}

type Value struct {
	Name     string
	Key      string
	Required bool
}

type QueryParams struct {
	Values []Value
}

type Request struct {
	Name        string
	Headers     []Header
	QueryParams *QueryParams
	Body        *RequestBody
}

type RequestBody struct {
	Name     string
	Required bool
}

type Response struct {
	Name  string
	Codes []ResponseCode
}

type ResponseCode struct {
	Code int
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

func (g *Generator) Generate(
	ctx context.Context,
	doc v3.Document,
	client string,
	args []string,
) error {
	client = canonize(client)

	if err := g.generateClient(client, args); err != nil {
		return fmt.Errorf("could not generate client: %w", err)
	}

	if err := g.generateComponents(ctx, doc.Components); err != nil {
		return fmt.Errorf("could not generate components: %w", err)
	}

	if err := g.generateMethods(ctx, client, doc.Paths); err != nil {
		return fmt.Errorf("could not generate methods: %w", err)
	}

	return nil
}

func (g *Generator) generateClient(name string, args []string) error {
	return clientTemplate.Execute(&g.buf, map[string]string{
		"ClientName": name,
		"Package":    strings.ToLower(name),
		"CodeGen":    strings.Join(args, " "),
	})
}

func (g *Generator) generateComponents(
	ctx context.Context,
	components *v3.Components,
) error {
	schemas := orderedmap.Iterate(ctx, components.Schemas)
	for proxy := range schemas {
		schema := proxy.Value().Schema()

		if len(schema.Type) == 0 {
			log.Printf("invalid schema %q", proxy.Key())
			continue
		}

		typ := schema.Type[0]
		if typ != "object" {
			continue
		}

		properties := collectProperties(ctx, schema, "")

		err := componentsTemplate.Execute(&g.buf, map[string]any{
			"Name":       proxy.Key(),
			"Properties": properties,
		})
		if err != nil {
			return fmt.Errorf("could not render %q: %w", proxy.Key(), err)
		}
	}

	return nil
}

func (g *Generator) generateMethods(
	ctx context.Context,
	client string,
	paths *v3.Paths,
) error {
	if paths == nil {
		return nil
	}

	pathitems := collectPathItems(ctx, paths)
	if len(pathitems) == 0 {
		return nil
	}

	if err := g.generateConfigs(pathitems); err != nil {
		return fmt.Errorf("could not generate configs: %w", err)
	}

	for _, item := range pathitems {
		if err := g.generateMethod(ctx, client, item.Path, item.Method, item.Op); err != nil {
			return fmt.Errorf("could not generate method %q: %w", item.Path, err)
		}
	}

	return nil
}

type pathitem struct {
	Path   string
	Method string
	Op     *v3.Operation
}

func collectPathItems(ctx context.Context, paths *v3.Paths) []pathitem {
	result := make([]pathitem, 0)

	for pair := range orderedmap.Iterate(ctx, paths.PathItems) {
		path := pair.Key()
		pitem := pair.Value()

		result = appendPathItem(result, path, http.MethodHead, pitem.Head)
		result = appendPathItem(result, path, http.MethodGet, pitem.Get)
		result = appendPathItem(result, path, http.MethodPut, pitem.Put)
		result = appendPathItem(result, path, http.MethodPost, pitem.Post)
		result = appendPathItem(result, path, http.MethodDelete, pitem.Delete)
	}

	return result
}

func appendPathItem(dst []pathitem, path, method string, op *v3.Operation) []pathitem {
	if op == nil {
		return dst
	}

	return append(dst, pathitem{
		Path:   path,
		Method: method,
		Op:     op,
	})
}

func (g *Generator) generateConfigs(pathItems []pathitem) error {
	if len(pathItems) == 0 {
		return nil
	}

	methods := make([]string, 0, len(pathItems))
	for _, item := range pathItems {
		methods = append(methods, item.Method+canonize(item.Path))
	}

	return configTemplate.Execute(&g.buf, map[string]any{"Methods": methods})
}

func (g *Generator) generateMethod(
	ctx context.Context,
	client, path, method string,
	op *v3.Operation,
) error {
	if op == nil {
		return nil
	}

	request := Request{
		Name: method + canonize(path) + "Request",
	}

	queryparams := make([]Value, 0)

	for _, parameter := range op.Parameters {
		switch parameter.In {
		case "header":
			request.Headers = append(request.Headers, Header{
				Name: canonize(parameter.Name),
				Key:  parameter.Name,
			})
		case "query":
			queryparams = append(queryparams, Value{
				Name:     canonize(parameter.Name),
				Key:      parameter.Name,
				Required: resolveptr(parameter.Required),
			})
		}
	}

	if len(queryparams) > 0 {
		request.QueryParams = &QueryParams{
			Values: queryparams,
		}
	}

	if op.RequestBody != nil {
		reqbody := &RequestBody{}
		reqbody.Name = method + canonize(path) + "RequestBody"
		reqbody.Required = resolveptr(op.RequestBody.Required)

		media := op.RequestBody.Content.GetOrZero(contentType)
		if media != nil && media.Schema != nil {
			reference := media.Schema.GetReference()
			splits := strings.Split(reference, "/")

			reqbody.Name = splits[len(splits)-1]
		}

		request.Body = reqbody
	}

	response := Response{
		Name: method + canonize(path) + "Response",
	}

	if op.Responses != nil {
		codes := orderedmap.Iterate(ctx, op.Responses.Codes)
		for code := range codes {
			httpcode, err := strconv.ParseInt(code.Key(), 10, 64)
			if err != nil {
				return fmt.Errorf("could not parse code %q: %q: %w", path, code.Key(), err)
			}

			if httpcode >= 300 {
				continue
			}

			schema := code.Value()
			media := schema.Content.GetOrZero(contentType)
			if media == nil || media.Schema == nil {
				return fmt.Errorf("invalid response schema %q %q: %w", path, code.Key(), err)
			}

			reference := media.Schema.GetReference()
			splits := strings.Split(reference, "/")

			response.Codes = append(response.Codes, ResponseCode{
				Code: int(httpcode),
				Name: splits[len(splits)-1],
			})
		}
	}

	parameters := make(map[string]any)
	parameters["Client"] = client
	parameters["Path"] = path
	parameters["Name"] = method + canonize(path)
	parameters["Method"] = method
	parameters["Response"] = response
	parameters["Request"] = request

	return requestTemplate.Execute(&g.buf, parameters)
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
	schema *base.Schema,
	parentName string,
) []Property {
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
		case "number":
			typ = "float64"
		case "object":
			// NOTE(max): it's a hack and might panic; please do something
			// about it.
			// TODO(max): generate struct for inline property.
			reference := value.ParentProxy.GetReference()
			if reference == "" {
				reference = parentName + canonize(key)
			}

			splits := strings.Split(reference, "/")
			typ = splits[len(splits)-1]
		case "array":
			// NOTE(max): it's a hack and might panic; please do something
			// about it.
			itemsSchema := value.Items.A.Schema()

			atyps := itemsSchema.Type
			if len(atyps) == 0 {
				log.Fatalf("invalid schema typ: %q", canonize(key))
			}

			atyp := atyps[0]

			typ = "[]" + atyp

			if atyp == "object" {
				reference := itemsSchema.ParentProxy.GetReference()
				if reference == "" {
					reference = parentName + canonize(key)
				}

				splits := strings.Split(reference, "/")
				typ = "[]" + splits[len(splits)-1]
			}
		}

		tag := key
		if !required {
			tag = tag + ",omitempty"
		}

		result = append(result, Property{
			Name: canonize(key),
			Type: typ,
			Tag:  tag,
		})
	}

	return result
}