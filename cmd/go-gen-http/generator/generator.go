package generator

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"go/format"
	"log"
	"slices"
	"strings"
	"text/template"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

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

type RequestBody struct {
	Name     string
	Required bool
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
	doc v3high.Document,
	client string,
	args []string,
) error {
	client = canonize(client)

	paths, err := CollectPaths(ctx, doc.Paths)
	if err != nil {
		return fmt.Errorf("could not collect paths: %w", err)
	}

	if len(paths) == 0 {
		return nil
	}

	if err := g.generateClient(client, args); err != nil {
		return fmt.Errorf("could not generate client: %w", err)
	}

	if err := g.generateConfig(paths); err != nil {
		return fmt.Errorf("could not generate config: %w", err)
	}

	if err := g.generateComponents(ctx, doc.Components); err != nil {
		return fmt.Errorf("could not generate components: %w", err)
	}

	if err := g.generateMethods(client, paths); err != nil {
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

func (g *Generator) generateConfig(paths []Path) error {
	return configTemplate.Execute(&g.buf, map[string]any{
		"Paths": paths,
	})
}

func (g *Generator) generateComponents(
	ctx context.Context,
	components *v3high.Components,
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

func (g *Generator) generateMethods(client string, paths []Path) error {
	for _, path := range paths {
		if err := g.generateMethod(client, path); err != nil {
			return fmt.Errorf("could not generate method %q: %w", path.URL, err)
		}
	}

	return nil
}

func (g *Generator) generateMethod(client string, path Path) error {
	parameters := make(map[string]any)
	parameters["Client"] = client
	parameters["Path"] = path

	return requestTemplate.Execute(&g.buf, parameters)
}

func (g *Generator) Source() ([]byte, error) {
	source, err := format.Source(g.buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not format source: %w", err)
	}

	return source, nil
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
